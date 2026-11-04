package main

import (
	"image"
	"image/color"
	"math"

	"geoworld/internal/navigation"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

type navigationPreviewCell struct {
	Score         navigation.Score
	CoastDistance float64
}

func buildNavigationPreviewGrid(active *activeWorld) []navigationPreviewCell {
	if active == nil || active.sample == nil || active.runtimeWorld == nil {
		return nil
	}
	g := active.geometry
	grid := make([]navigationPreviewCell, g.Width*g.Height)
	coast := buildCoastDistanceGrid(activeSurface(active), g)
	water := activeWaterbed(active)

	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			i := z*g.Width + x
			wx, wz := g.NodePosition(x, z)
			s := active.sample(wx, wz)
			settlementScore, _ := settlementScoreAt(active, wx, wz)
			hasCoast := i < len(coast) && !math.IsInf(coast[i], 1)
			coastDistance := 0.0
			if hasCoast {
				coastDistance = coast[i]
			}
			score := navigationScore(
				s,
				water,
				wx,
				wz,
				coastDistance,
				hasCoast,
				settlementScore.Buildability,
				settlementScore.FloodRisk,
				settlementScore.SettlementSuitability,
			)
			grid[i] = navigationPreviewCell{
				Score:         score,
				CoastDistance: coastDistance,
			}
		}
	}
	return grid
}

func navigationScoreAt(
	active *activeWorld,
	worldX, worldZ float64,
) (navigation.Score, bool) {
	if active == nil || len(active.navigationGrid) == 0 {
		return navigation.Score{}, false
	}
	g := active.geometry
	gx := int(math.Round((worldX - g.OriginX) / g.SpacingX()))
	gz := int(math.Round((worldZ - g.OriginZ) / g.SpacingZ()))
	gx = clampInt(gx, 0, g.Width-1)
	gz = clampInt(gz, 0, g.Height-1)
	return active.navigationGrid[gz*g.Width+gx].Score, true
}

func navigationScoreExact(
	active *activeWorld,
	worldX, worldZ float64,
) (navigation.Score, bool) {
	if active == nil || active.sample == nil {
		return navigation.Score{}, false
	}
	s := active.sample(worldX, worldZ)
	settlementScore, ok := settlementScoreExact(active, worldX, worldZ, 16)
	if !ok {
		return navigation.Score{}, false
	}
	coastDistance, hasCoast := coastDistanceAt(active, worldX, worldZ)
	return navigationScore(
		s,
		activeWaterbed(active),
		worldX,
		worldZ,
		coastDistance,
		hasCoast,
		settlementScore.Buildability,
		settlementScore.FloodRisk,
		settlementScore.SettlementSuitability,
	), true
}

func navigationScore(
	s runtimeprofile.Sample,
	water *waterbed.Model,
	worldX, worldZ float64,
	coastDistance float64,
	hasCoastDistance bool,
	buildability float64,
	floodRisk float64,
	settlementSuitability float64,
) navigation.Score {
	riverSample, hasRiver := waterbed.RiverNavigationSample{}, false
	lakeDistance, hasLake := 0.0, false
	if water != nil {
		riverSample, hasRiver = water.NavigationSampleAt(worldX, worldZ)
		lakeDistance, hasLake = water.DistanceToNearestLakeShore(worldX, worldZ)
	}

	return navigation.Evaluate(navigation.Inputs{
		Land:                 s.SourceLand && s.World.WaterKind == reconcile.WaterNone,
		WaterKind:            waterKindName(s.World.WaterKind),
		WaterDepthNormalized: s.World.WaterDepth,

		RiverAvailable:  hasRiver,
		RiverHalfWidth:  riverSample.HalfWidth,
		RiverGradient:   riverSample.Gradient,
		RiverOceanMouth: riverSample.OceanMouth,
		RiverLakeMouth:  riverSample.LakeMouth,
		RiverLakeOutlet: riverSample.LakeOutlet,

		DistanceToRiverBlocks: riverSample.BankDistance,
		HasRiverDistance:      hasRiver,
		DistanceToLakeBlocks:  lakeDistance,
		HasLakeDistance:       hasLake,
		DistanceToCoastBlocks: coastDistance,
		HasCoastDistance:      hasCoastDistance,

		Buildability:          buildability,
		FloodRisk:             floodRisk,
		SettlementSuitability: settlementSuitability,
	})
}

func buildCoastDistanceGrid(
	surface *world.Surface,
	g world.GridGeometry,
) []float64 {
	n := g.Width * g.Height
	out := make([]float64, n)
	for i := range out {
		out[i] = math.Inf(1)
	}
	if surface == nil || len(surface.LandMask) != n || g.Width <= 0 || g.Height <= 0 {
		return out
	}

	// Ocean macro nodes are distance zero. Two chamfer passes then approximate
	// Euclidean distance to the nearest ocean/coast while remaining O(N).
	for i, land := range surface.LandMask {
		if !land {
			out[i] = 0
		}
	}

	sx := math.Max(1, g.SpacingX())
	sz := math.Max(1, g.SpacingZ())
	diag := math.Hypot(sx, sz)
	update := func(i, from int, cost float64) {
		if from < 0 || from >= n {
			return
		}
		if candidate := out[from] + cost; candidate < out[i] {
			out[i] = candidate
		}
	}

	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			i := z*g.Width + x
			if x > 0 {
				update(i, i-1, sx)
			}
			if z > 0 {
				update(i, i-g.Width, sz)
				if x > 0 {
					update(i, i-g.Width-1, diag)
				}
				if x+1 < g.Width {
					update(i, i-g.Width+1, diag)
				}
			}
		}
	}
	for z := g.Height - 1; z >= 0; z-- {
		for x := g.Width - 1; x >= 0; x-- {
			i := z*g.Width + x
			if x+1 < g.Width {
				update(i, i+1, sx)
			}
			if z+1 < g.Height {
				update(i, i+g.Width, sz)
				if x > 0 {
					update(i, i+g.Width-1, diag)
				}
				if x+1 < g.Width {
					update(i, i+g.Width+1, diag)
				}
			}
		}
	}
	return out
}

func coastDistanceAt(
	active *activeWorld,
	worldX, worldZ float64,
) (float64, bool) {
	if active == nil || len(active.navigationGrid) == 0 {
		return 0, false
	}
	g := active.geometry
	gx := int(math.Round((worldX - g.OriginX) / g.SpacingX()))
	gz := int(math.Round((worldZ - g.OriginZ) / g.SpacingZ()))
	gx = clampInt(gx, 0, g.Width-1)
	gz = clampInt(gz, 0, g.Height-1)
	d := active.navigationGrid[gz*g.Width+gx].CoastDistance
	if math.IsInf(d, 1) || math.IsNaN(d) {
		return 0, false
	}
	return d, true
}

func renderNavigationPreviewLayer(img *image.RGBA, active *activeWorld) {
	bounds := img.Bounds()
	g := active.geometry
	for py := 0; py < bounds.Dy(); py++ {
		z := g.OriginZ + float64(py)/float64(maxInt(bounds.Dy()-1, 1))*g.SpanZBlocks
		for px := 0; px < bounds.Dx(); px++ {
			x := g.OriginX + float64(px)/float64(maxInt(bounds.Dx()-1, 1))*g.SpanXBlocks
			s := previewSampleAt(active, x, z)
			base := layerColor("final", s)
			score, ok := navigationScoreAt(active, x, z)
			if !ok {
				img.SetRGBA(px, py, base)
				continue
			}
			if s.World.WaterKind != reconcile.WaterNone {
				n := clamp(score.WaterwayNavigability, 0, 1)
				b := clamp(score.NavigationBottleneckRisk, 0, 1)
				if n < 0.35 && b < 0.45 {
					img.SetRGBA(px, py, base)
					continue
				}
				alpha := 0.28 + 0.48*math.Max(n, b)
				img.SetRGBA(px, py, lerp(
					base,
					navigationWaterColor(n, b),
					clamp(alpha, 0, 0.78),
				))
				continue
			}
			v := clamp(score.PortSuitability, 0, 1)
			if v < 0.45 {
				img.SetRGBA(px, py, base)
				continue
			}
			img.SetRGBA(px, py, lerp(
				base,
				portSuitabilityColor(v),
				0.22+0.48*clamp((v-0.45)/0.55, 0, 1),
			))
		}
	}
}

func navigationWaterColor(navigability, bottleneck float64) color.RGBA {
	navigability = clamp(navigability, 0, 1)
	bottleneck = clamp(bottleneck, 0, 1)
	base := lerp(
		color.RGBA{42, 63, 76, 255},
		color.RGBA{42, 151, 198, 255},
		navigability,
	)
	return lerp(base, color.RGBA{177, 77, 62, 255}, 0.50*bottleneck)
}

func portSuitabilityColor(v float64) color.RGBA {
	v = clamp(v, 0, 1)
	switch {
	case v < 0.30:
		return lerp(
			color.RGBA{78, 70, 62, 255},
			color.RGBA{135, 103, 68, 255},
			v/0.30,
		)
	case v < 0.65:
		return lerp(
			color.RGBA{135, 103, 68, 255},
			color.RGBA{188, 164, 83, 255},
			(v-0.30)/0.35,
		)
	default:
		return lerp(
			color.RGBA{188, 164, 83, 255},
			color.RGBA{76, 176, 126, 255},
			(v-0.65)/0.35,
		)
	}
}
