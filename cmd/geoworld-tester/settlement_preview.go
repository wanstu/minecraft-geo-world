package main

import (
	"image"
	"image/color"
	"math"

	"geoworld/internal/reconcile"
	"geoworld/internal/settlement"
	"geoworld/internal/waterbed"
)

type settlementPreviewCell struct {
	Score settlement.Score
}

func buildSettlementPreviewGrid(active *activeWorld) []settlementPreviewCell {
	if active == nil || active.sample == nil || active.runtimeWorld == nil {
		return nil
	}
	g := active.geometry
	grid := make([]settlementPreviewCell, g.Width*g.Height)
	water := activeWaterbed(active)
	stepX := math.Max(1, g.SpacingX())
	stepZ := math.Max(1, g.SpacingZ())

	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			wx, wz := g.NodePosition(x, z)
			s := active.sample(wx, wz)
			climate := active.runtimeWorld.SampleClimate(wx, wz)
			riverDistance, hasRiver, lakeDistance, hasLake :=
				settlementFreshWaterDistances(water, wx, wz)
			slope := settlementSlope(active, wx, wz, stepX, stepZ)
			score := settlement.Evaluate(settlement.Inputs{
				Land:                  s.SourceLand,
				WaterKind:             waterKindName(s.World.WaterKind),
				Slope:                 slope,
				Terrain:               s.TerrainVariant.String(),
				Surface:               s.SurfaceProfile.String(),
				Biome:                 s.BiomeFamily.String(),
				AnnualMeanTempC:       climate.AnnualMeanTempC,
				AnnualPrecipMM:        climate.AnnualPrecipMM,
				DistanceToRiverBlocks: riverDistance,
				HasRiverDistance:      hasRiver,
				DistanceToLakeBlocks:  lakeDistance,
				HasLakeDistance:       hasLake,
			})
			grid[z*g.Width+x] = settlementPreviewCell{Score: score}
		}
	}
	return grid
}

func settlementScoreExact(
	active *activeWorld,
	worldX, worldZ, slopeStep float64,
) (settlement.Score, bool) {
	if active == nil || active.sample == nil || active.runtimeWorld == nil {
		return settlement.Score{}, false
	}
	slopeStep = math.Max(1, slopeStep)
	s := active.sample(worldX, worldZ)
	climate := active.runtimeWorld.SampleClimate(worldX, worldZ)
	riverDistance, hasRiver, lakeDistance, hasLake :=
		settlementFreshWaterDistances(activeWaterbed(active), worldX, worldZ)
	return settlement.Evaluate(settlement.Inputs{
		Land:                  s.SourceLand,
		WaterKind:             waterKindName(s.World.WaterKind),
		Slope:                 settlementSlope(active, worldX, worldZ, slopeStep, slopeStep),
		Terrain:               s.TerrainVariant.String(),
		Surface:               s.SurfaceProfile.String(),
		Biome:                 s.BiomeFamily.String(),
		AnnualMeanTempC:       climate.AnnualMeanTempC,
		AnnualPrecipMM:        climate.AnnualPrecipMM,
		DistanceToRiverBlocks: riverDistance,
		HasRiverDistance:      hasRiver,
		DistanceToLakeBlocks:  lakeDistance,
		HasLakeDistance:       hasLake,
	}), true
}

func settlementScoreAt(active *activeWorld, worldX, worldZ float64) (settlement.Score, bool) {
	if active == nil || len(active.settlementGrid) == 0 {
		return settlement.Score{}, false
	}
	g := active.geometry
	gx := int(math.Round((worldX - g.OriginX) / g.SpacingX()))
	gz := int(math.Round((worldZ - g.OriginZ) / g.SpacingZ()))
	gx = clampInt(gx, 0, g.Width-1)
	gz = clampInt(gz, 0, g.Height-1)
	return active.settlementGrid[gz*g.Width+gx].Score, true
}

func renderSettlementPreviewLayer(img *image.RGBA, active *activeWorld) {
	bounds := img.Bounds()
	g := active.geometry
	for py := 0; py < bounds.Dy(); py++ {
		z := g.OriginZ + float64(py)/float64(maxInt(bounds.Dy()-1, 1))*g.SpanZBlocks
		for px := 0; px < bounds.Dx(); px++ {
			x := g.OriginX + float64(px)/float64(maxInt(bounds.Dx()-1, 1))*g.SpanXBlocks
			sample := previewSampleAt(active, x, z)
			base := layerColor("final", sample)
			if sample.World.WaterKind != reconcile.WaterNone || !sample.SourceLand {
				img.SetRGBA(px, py, base)
				continue
			}
			score, ok := settlementScoreAt(active, x, z)
			if !ok {
				img.SetRGBA(px, py, base)
				continue
			}
			v := clamp(score.SettlementSuitability, 0, 1)
			if v < 0.45 {
				img.SetRGBA(px, py, base)
				continue
			}
			alpha := 0.22 + 0.48*clamp((v-0.45)/0.55, 0, 1)
			img.SetRGBA(px, py, lerp(base, settlementSuitabilityColor(v), alpha))
		}
	}
}

func settlementSuitabilityColor(v float64) color.RGBA {
	v = clamp(v, 0, 1)
	switch {
	case v < 0.30:
		return lerp(color.RGBA{92, 70, 62, 255}, color.RGBA{151, 112, 73, 255}, v/0.30)
	case v < 0.55:
		return lerp(color.RGBA{151, 112, 73, 255}, color.RGBA{171, 162, 91, 255}, (v-0.30)/0.25)
	case v < 0.75:
		return lerp(color.RGBA{171, 162, 91, 255}, color.RGBA{91, 151, 89, 255}, (v-0.55)/0.20)
	default:
		return lerp(color.RGBA{91, 151, 89, 255}, color.RGBA{68, 188, 125, 255}, (v-0.75)/0.25)
	}
}

func settlementSlope(active *activeWorld, x, z, stepX, stepZ float64) float64 {
	left := active.sample(x-stepX, z).World.GroundHeight
	right := active.sample(x+stepX, z).World.GroundHeight
	down := active.sample(x, z-stepZ).World.GroundHeight
	up := active.sample(x, z+stepZ).World.GroundHeight
	dx := (right - left) / (2 * stepX)
	dz := (up - down) / (2 * stepZ)
	return math.Hypot(dx, dz)
}

func settlementFreshWaterDistances(
	water *waterbed.Model,
	worldX, worldZ float64,
) (float64, bool, float64, bool) {
	if water == nil {
		return 0, false, 0, false
	}
	riverDistance, hasRiver := water.DistanceToNearestRiverBank(worldX, worldZ)
	lakeDistance, hasLake := water.DistanceToNearestLakeShore(worldX, worldZ)
	return riverDistance, hasRiver, lakeDistance, hasLake
}
