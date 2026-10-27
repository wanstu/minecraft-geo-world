package main

import (
	"image"
	"image/color"
	"math"
	"strings"

	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
)

func buildPreviewGrid(active *activeWorld) []runtimeprofile.Sample {
	if active == nil || active.sample == nil {
		return nil
	}
	g := active.geometry
	grid := make([]runtimeprofile.Sample, g.Width*g.Height)
	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			wx, wz := g.NodePosition(x, z)
			grid[z*g.Width+x] = active.sample(wx, wz)
		}
	}
	return grid
}

func previewSampleAt(active *activeWorld, x, z float64) runtimeprofile.Sample {
	if active == nil {
		return runtimeprofile.Sample{}
	}
	if len(active.previewGrid) == 0 {
		return active.sample(x, z)
	}
	g := active.geometry
	gx := int(math.Round((x - g.OriginX) / g.SpacingX()))
	gz := int(math.Round((z - g.OriginZ) / g.SpacingZ()))
	gx = clampInt(gx, 0, g.Width-1)
	gz = clampInt(gz, 0, g.Height-1)
	return active.previewGrid[gz*g.Width+gx]
}

func renderTectonicPreviewLayer(img *image.RGBA, active *activeWorld) {
	bounds := img.Bounds()
	g := active.geometry

	var features []phase1.TectonicFeature
	var intensity []float64
	if active.model != nil && active.model.Phase1 != nil {
		features = active.model.Phase1.Feature
		intensity = active.model.Phase1.TectonicIntensity
	} else if active.reopened != nil && active.reopened.TectonicQuery != nil {
		features = active.reopened.TectonicQuery.Feature
		intensity = active.reopened.TectonicQuery.Intensity
	}

	for py := 0; py < bounds.Dy(); py++ {
		z := g.OriginZ + float64(py)/float64(maxInt(bounds.Dy()-1, 1))*g.SpanZBlocks
		gz := int(math.Round((z - g.OriginZ) / g.SpacingZ()))
		gz = clampInt(gz, 0, g.Height-1)
		for px := 0; px < bounds.Dx(); px++ {
			x := g.OriginX + float64(px)/float64(maxInt(bounds.Dx()-1, 1))*g.SpanXBlocks
			gx := int(math.Round((x - g.OriginX) / g.SpacingX()))
			gx = clampInt(gx, 0, g.Width-1)
			index := gz*g.Width + gx

			base := previewSampleAt(active, x, z)
			c := tectonicBaseColor(base)
			if index < len(features) && features[index] != phase1.NoFeature {
				strength := 0.55
				if index < len(intensity) {
					strength = clamp(intensity[index], 0, 1)
				}
				c = blendColor(c, tectonicFeatureColor(features[index]), 0.45+0.45*strength)
			}
			img.SetRGBA(px, py, c)
		}
	}
}

func tectonicBaseColor(s runtimeprofile.Sample) color.RGBA {
	if s.World.WaterKind == reconcile.WaterOcean {
		return color.RGBA{26, 57, 79, 255}
	}
	h := clamp(s.World.GroundHeight, 0, 1)
	return blendColor(
		color.RGBA{58, 69, 62, 255},
		color.RGBA{141, 139, 125, 255},
		h,
	)
}

func tectonicFeatureColor(feature phase1.TectonicFeature) color.RGBA {
	switch feature {
	case phase1.CollisionOrogen:
		return color.RGBA{220, 90, 72, 255}
	case phase1.VolcanicArc:
		return color.RGBA{239, 120, 55, 255}
	case phase1.Trench:
		return color.RGBA{70, 69, 122, 255}
	case phase1.IslandArc:
		return color.RGBA{204, 143, 66, 255}
	case phase1.MidOceanRidge:
		return color.RGBA{188, 83, 126, 255}
	case phase1.ContinentalRift:
		return color.RGBA{205, 176, 64, 255}
	case phase1.TransformFault:
		return color.RGBA{150, 101, 170, 255}
	default:
		return color.RGBA{112, 112, 112, 255}
	}
}

func mineralProvinceColor(s runtimeprofile.Sample) color.RGBA {
	if s.World.WaterKind == reconcile.WaterOcean {
		return color.RGBA{31, 74, 105, 255}
	}
	terrain := s.TerrainVariant.String()
	biome := s.BiomeFamily.String()

	type mineral struct {
		name   string
		weight float64
		color  color.RGBA
	}
	minerals := []mineral{
		{"coal", 7.5, color.RGBA{72, 68, 62, 255}},
		{"iron", 9.0, color.RGBA{166, 139, 122, 255}},
		{"copper", 6.0, color.RGBA{191, 108, 66, 255}},
		{"gold", 3.2, color.RGBA{219, 176, 55, 255}},
		{"lapis", 2.7, color.RGBA{55, 91, 177, 255}},
		{"redstone", 4.2, color.RGBA{181, 62, 57, 255}},
		{"diamond", 1.0, color.RGBA{73, 189, 187, 255}},
		{"emerald", 1.2, color.RGBA{65, 164, 91, 255}},
	}

	best := minerals[0]
	bestScore := -1.0
	for _, m := range minerals {
		score := mineralProvinceAffinity(m.name, m.weight, terrain, biome)
		if score > bestScore {
			bestScore = score
			best = m
		}
	}
	intensity := clamp(bestScore/20.0, 0.25, 1)
	return blendColor(color.RGBA{92, 98, 87, 255}, best.color, 0.45+0.45*intensity)
}

func mineralProvinceAffinity(name string, weight float64, terrain, biome string) float64 {
	mountains := terrain == mcprofile.TerrainMountain.String()
	foothills := terrain == mcprofile.TerrainFoothills.String() || terrain == mcprofile.TerrainHills.String()
	sedimentary := terrain == mcprofile.TerrainPlain.String() ||
		terrain == mcprofile.TerrainRollingPlain.String() ||
		terrain == mcprofile.TerrainFloodplain.String() ||
		terrain == mcprofile.TerrainAlluvialPlain.String() ||
		terrain == mcprofile.TerrainAlluvialFan.String()
	arid := strings.Contains(biome, "Desert") || biome == "Savanna"
	highlands := mountains || foothills || terrain == mcprofile.TerrainPlateau.String()

	multiplier := 1.0
	switch name {
	case "coal":
		if sedimentary {
			multiplier = 2.1
		} else {
			multiplier = 0.8
		}
	case "iron":
		if mountains {
			multiplier = 2.2
		} else if foothills {
			multiplier = 1.6
		}
	case "copper":
		if highlands {
			multiplier = 2.0
		} else {
			multiplier = 0.9
		}
	case "gold":
		if terrain == mcprofile.TerrainPlateau.String() ||
			terrain == mcprofile.TerrainCanyon.String() || arid {
			multiplier = 2.1
		} else {
			multiplier = 0.75
		}
	case "lapis":
		if sedimentary {
			multiplier = 1.5
		}
	case "redstone":
		if highlands {
			multiplier = 1.25
		}
	case "emerald":
		if mountains {
			multiplier = 2.4
		} else if foothills {
			multiplier = 1.4
		} else {
			multiplier = 0.08
		}
	}
	return weight * multiplier
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func blendColor(a, b color.RGBA, t float64) color.RGBA {
	t = clamp(t, 0, 1)
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}
