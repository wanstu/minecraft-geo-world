package main

import (
	"image"
	"image/color"
	"math"

	"geoworld/internal/advantage"
	"geoworld/internal/reconcile"
)

type regionalAdvantagePreviewCell struct {
	Score    advantage.Score
	RegionID int
}

func buildRegionalAdvantagePreviewGrid(
	active *activeWorld,
) ([]regionalAdvantagePreviewCell, advantage.Summary, advantage.RegionModel) {
	if active == nil ||
		len(active.settlementGrid) == 0 ||
		len(active.navigationGrid) == 0 ||
		len(active.miningGrid) == 0 {
		return nil, advantage.Summary{}, advantage.RegionModel{}
	}
	g := active.geometry
	n := g.Width * g.Height
	if len(active.settlementGrid) != n ||
		len(active.navigationGrid) != n ||
		len(active.miningGrid) != n {
		return nil, advantage.Summary{}, advantage.RegionModel{}
	}

	grid := make([]regionalAdvantagePreviewCell, n)
	scores := make([]advantage.Score, n)
	for gz := 0; gz < g.Height; gz++ {
		for gx := 0; gx < g.Width; gx++ {
			i := gz*g.Width + gx
			x, z := g.NodePosition(gx, gz)
			sample := active.sample(x, z)
			settlement := active.settlementGrid[i].Score
			navigation := active.navigationGrid[i].Score
			mining := active.miningGrid[i].Score

			score := advantage.Evaluate(advantage.Inputs{
				Land: sample.SourceLand &&
					sample.World.WaterKind == reconcile.WaterNone,
				SettlementSuitability:    settlement.SettlementSuitability,
				Buildability:             settlement.Buildability,
				FreshWaterAccess:         settlement.FreshWaterAccess,
				FloodRisk:                settlement.FloodRisk,
				AgricultureSuitability:   settlement.AgricultureSuitability,
				TransportAccess:          settlement.TransportAccess,
				ResourceAccess:           settlement.ResourceAccess,
				WaterwayNavigability:     navigation.WaterwayNavigability,
				PortSuitability:          navigation.PortSuitability,
				NavigationBottleneckRisk: navigation.NavigationBottleneckRisk,
				MiningSuitability:        mining.MiningSuitability,
				IndustrialValue:          mining.IndustrialValue,
				ResourceDiversity:        mining.ResourceDiversity,
			})
			grid[i].Score = score
			scores[i] = score
		}
	}
	regions := advantage.BuildRegions(scores, advantage.RegionConfig{
		Width:        g.Width,
		Height:       g.Height,
		OriginX:      g.OriginX,
		OriginZ:      g.OriginZ,
		SpacingX:     g.SpacingX(),
		SpacingZ:     g.SpacingZ(),
		MinPotential: advantage.DefaultRegionPotentialThreshold,
	})
	for i := range grid {
		if i < len(regions.CellRegionIDs) {
			grid[i].RegionID = regions.CellRegionIDs[i]
		}
	}
	return grid, advantage.Summarize(scores), regions
}

func regionalAdvantageCellIndex(
	active *activeWorld,
	worldX, worldZ float64,
) (index, gx, gz int, ok bool) {
	if active == nil || len(active.regionalAdvantageGrid) == 0 {
		return 0, 0, 0, false
	}
	g := active.geometry
	gx = int(math.Round((worldX - g.OriginX) / g.SpacingX()))
	gz = int(math.Round((worldZ - g.OriginZ) / g.SpacingZ()))
	gx = clampInt(gx, 0, g.Width-1)
	gz = clampInt(gz, 0, g.Height-1)
	index = gz*g.Width + gx
	return index, gx, gz, index >= 0 && index < len(active.regionalAdvantageGrid)
}

func regionalAdvantageScoreAt(
	active *activeWorld,
	worldX, worldZ float64,
) (advantage.Score, bool) {
	index, _, _, ok := regionalAdvantageCellIndex(active, worldX, worldZ)
	if !ok {
		return advantage.Score{}, false
	}
	return active.regionalAdvantageGrid[index].Score, true
}

func regionalAdvantageRegionAt(
	active *activeWorld,
	worldX, worldZ float64,
) (advantage.Region, bool) {
	index, _, _, ok := regionalAdvantageCellIndex(active, worldX, worldZ)
	if !ok || index >= len(active.regionalAdvantageRegions.CellRegionIDs) {
		return advantage.Region{}, false
	}
	return active.regionalAdvantageRegions.Region(
		active.regionalAdvantageRegions.CellRegionIDs[index],
	)
}

func regionalAdvantageBoundaryAt(
	active *activeWorld,
	gx, gz int,
	worldX, worldZ float64,
	bounds image.Rectangle,
) bool {
	if active == nil || len(active.regionalAdvantageGrid) == 0 {
		return false
	}
	g := active.geometry
	index := gz*g.Width + gx
	if index < 0 || index >= len(active.regionalAdvantageGrid) {
		return false
	}
	regionID := active.regionalAdvantageGrid[index].RegionID
	gxFloat := (worldX - g.OriginX) / g.SpacingX()
	gzFloat := (worldZ - g.OriginZ) / g.SpacingZ()
	pixelHalfX := math.Min(0.45, 0.55*float64(g.Width-1)/
		float64(maxInt(bounds.Dx()-1, 1)))
	pixelHalfZ := math.Min(0.45, 0.55*float64(g.Height-1)/
		float64(maxInt(bounds.Dy()-1, 1)))
	regionDiffers := func(nx, nz int) bool {
		return active.regionalAdvantageGrid[nz*g.Width+nx].RegionID != regionID
	}
	if gx > 0 && regionDiffers(gx-1, gz) &&
		math.Abs(gxFloat-(float64(gx)-0.5)) <= pixelHalfX {
		return true
	}
	if gx+1 < g.Width && regionDiffers(gx+1, gz) &&
		math.Abs(gxFloat-(float64(gx)+0.5)) <= pixelHalfX {
		return true
	}
	if gz > 0 && regionDiffers(gx, gz-1) &&
		math.Abs(gzFloat-(float64(gz)-0.5)) <= pixelHalfZ {
		return true
	}
	if gz+1 < g.Height && regionDiffers(gx, gz+1) &&
		math.Abs(gzFloat-(float64(gz)+0.5)) <= pixelHalfZ {
		return true
	}
	return false
}

func regionalAdvantageScoreExact(
	active *activeWorld,
	worldX, worldZ float64,
) (advantage.Score, bool) {
	if active == nil || active.sample == nil {
		return advantage.Score{}, false
	}
	sample := active.sample(worldX, worldZ)
	settlement, ok := settlementScoreExact(active, worldX, worldZ, 16)
	if !ok {
		return advantage.Score{}, false
	}
	navigation, ok := navigationScoreExact(active, worldX, worldZ)
	if !ok {
		return advantage.Score{}, false
	}
	mining, ok := miningScoreExact(active, worldX, worldZ)
	if !ok {
		return advantage.Score{}, false
	}
	return advantage.Evaluate(advantage.Inputs{
		Land: sample.SourceLand &&
			sample.World.WaterKind == reconcile.WaterNone,
		SettlementSuitability:    settlement.SettlementSuitability,
		Buildability:             settlement.Buildability,
		FreshWaterAccess:         settlement.FreshWaterAccess,
		FloodRisk:                settlement.FloodRisk,
		AgricultureSuitability:   settlement.AgricultureSuitability,
		TransportAccess:          settlement.TransportAccess,
		ResourceAccess:           settlement.ResourceAccess,
		WaterwayNavigability:     navigation.WaterwayNavigability,
		PortSuitability:          navigation.PortSuitability,
		NavigationBottleneckRisk: navigation.NavigationBottleneckRisk,
		MiningSuitability:        mining.MiningSuitability,
		IndustrialValue:          mining.IndustrialValue,
		ResourceDiversity:        mining.ResourceDiversity,
	}), true
}

func renderRegionalAdvantagePreviewLayer(
	img *image.RGBA,
	active *activeWorld,
) {
	bounds := img.Bounds()
	g := active.geometry
	for py := 0; py < bounds.Dy(); py++ {
		z := g.OriginZ +
			float64(py)/float64(maxInt(bounds.Dy()-1, 1))*g.SpanZBlocks
		for px := 0; px < bounds.Dx(); px++ {
			x := g.OriginX +
				float64(px)/float64(maxInt(bounds.Dx()-1, 1))*g.SpanXBlocks
			sample := previewSampleAt(active, x, z)
			base := layerColor("final", sample)
			if !sample.SourceLand ||
				sample.World.WaterKind != reconcile.WaterNone {
				img.SetRGBA(px, py, base)
				continue
			}
			index, gx, gz, ok := regionalAdvantageCellIndex(active, x, z)
			if !ok {
				img.SetRGBA(px, py, base)
				continue
			}
			score := active.regionalAdvantageGrid[index].Score
			best := math.Max(
				math.Max(score.BuildingAdvantage, score.AgricultureAdvantage),
				math.Max(
					score.PortAdvantage,
					math.Max(score.MiningAdvantage, score.TransportAdvantage),
				),
			)
			if best < 0.42 {
				img.SetRGBA(px, py, base)
			} else {
				img.SetRGBA(px, py, lerp(
					base,
					regionalAdvantageColor(score),
					0.22+0.42*clamp((best-0.42)/0.58, 0, 1),
				))
			}
			if best >= 0.42 &&
				regionalAdvantageBoundaryAt(active, gx, gz, x, z, bounds) {
				img.SetRGBA(px, py, lerp(base, color.RGBA{27, 29, 32, 255}, 0.72))
			}
		}
	}
}

func regionalAdvantageColor(s advantage.Score) color.RGBA {
	var target color.RGBA
	switch s.RegionalClass {
	case "building_region":
		target = color.RGBA{125, 151, 171, 255}
	case "agriculture_region":
		target = color.RGBA{92, 169, 91, 255}
	case "port_region":
		target = color.RGBA{61, 143, 190, 255}
	case "mining_region":
		target = color.RGBA{177, 116, 66, 255}
	case "transport_region":
		target = color.RGBA{199, 164, 74, 255}
	case "multi_advantage_region":
		target = color.RGBA{155, 104, 182, 255}
	default:
		target = color.RGBA{101, 98, 89, 255}
	}
	best := math.Max(
		math.Max(s.BuildingAdvantage, s.AgricultureAdvantage),
		math.Max(
			s.PortAdvantage,
			math.Max(s.MiningAdvantage, s.TransportAdvantage),
		),
	)
	intensity := clamp(best, 0, 1)
	return lerp(
		color.RGBA{70, 69, 64, 255},
		target,
		0.18+0.82*intensity,
	)
}
