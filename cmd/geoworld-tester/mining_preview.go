package main

import (
	"image"
	"image/color"
	"math"

	"geoworld/internal/mining"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
)

type miningPreviewCell struct {
	Score mining.Score
}

func buildMiningPreviewGrid(active *activeWorld) []miningPreviewCell {
	if active == nil || active.sample == nil {
		return nil
	}
	g := active.geometry
	grid := make([]miningPreviewCell, g.Width*g.Height)
	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			i := z*g.Width + x
			wx, wz := g.NodePosition(x, z)
			s := active.sample(wx, wz)
			grid[i].Score = miningScore(active, wx, wz, s)
		}
	}
	return grid
}

func miningScoreAt(
	active *activeWorld,
	worldX, worldZ float64,
) (mining.Score, bool) {
	if active == nil || len(active.miningGrid) == 0 {
		return mining.Score{}, false
	}
	g := active.geometry
	gx := int(math.Round((worldX - g.OriginX) / g.SpacingX()))
	gz := int(math.Round((worldZ - g.OriginZ) / g.SpacingZ()))
	gx = clampInt(gx, 0, g.Width-1)
	gz = clampInt(gz, 0, g.Height-1)
	return active.miningGrid[gz*g.Width+gx].Score, true
}

func miningScoreExact(
	active *activeWorld,
	worldX, worldZ float64,
) (mining.Score, bool) {
	if active == nil || active.sample == nil {
		return mining.Score{}, false
	}
	return miningScore(
		active,
		worldX,
		worldZ,
		active.sample(worldX, worldZ),
	), true
}

func miningScore(
	active *activeWorld,
	worldX, worldZ float64,
	s runtimeprofile.Sample,
) mining.Score {
	return mining.Evaluate(mining.Inputs{
		Seed:                   active.seed,
		MineralRegionalization: active.parameterSnapshot.MineralRegionalization,
		X:                      worldX,
		Z:                      worldZ,
		Land:                   s.SourceLand,
		WaterKind:              waterKindName(s.World.WaterKind),
		Terrain:                s.TerrainVariant.String(),
		Biome:                  s.BiomeFamily.String(),
	})
}

func renderMiningPreviewLayer(img *image.RGBA, active *activeWorld) {
	bounds := img.Bounds()
	g := active.geometry
	for py := 0; py < bounds.Dy(); py++ {
		z := g.OriginZ + float64(py)/float64(maxInt(bounds.Dy()-1, 1))*g.SpanZBlocks
		for px := 0; px < bounds.Dx(); px++ {
			x := g.OriginX + float64(px)/float64(maxInt(bounds.Dx()-1, 1))*g.SpanXBlocks
			s := previewSampleAt(active, x, z)
			base := layerColor("final", s)
			if !s.SourceLand || s.World.WaterKind != reconcile.WaterNone {
				img.SetRGBA(px, py, base)
				continue
			}
			score, ok := miningScoreAt(active, x, z)
			if !ok {
				img.SetRGBA(px, py, base)
				continue
			}
			v := clamp(score.MiningSuitability, 0, 1)
			if v < 0.42 {
				img.SetRGBA(px, py, base)
				continue
			}
			img.SetRGBA(px, py, lerp(
				base,
				miningSuitabilityColor(score),
				0.22+0.48*clamp((v-0.42)/0.58, 0, 1),
			))
		}
	}
}

func miningSuitabilityColor(s mining.Score) color.RGBA {
	v := clamp(s.MiningSuitability, 0, 1)
	base := color.RGBA{78, 74, 68, 255}
	switch s.MiningClass {
	case "surface_district":
		base = color.RGBA{165, 114, 64, 255}
	case "shallow_district":
		base = color.RGBA{145, 120, 72, 255}
	case "deep_district":
		base = color.RGBA{105, 92, 126, 255}
	case "mixed_district":
		base = color.RGBA{151, 135, 78, 255}
	}
	return lerp(
		color.RGBA{68, 68, 66, 255},
		base,
		0.25+0.75*v,
	)
}
