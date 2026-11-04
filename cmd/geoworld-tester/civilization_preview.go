package main

import (
	"image"
	"image/color"
	"math"

	"geoworld/internal/civilization"
	"geoworld/internal/reconcile"
)

type civilizationPreviewCell struct {
	Score civilization.Score
}

func buildCivilizationPreview(
	active *activeWorld,
) ([]civilizationPreviewCell, civilization.Network) {
	if active == nil ||
		len(active.settlementGrid) == 0 ||
		len(active.navigationGrid) == 0 ||
		len(active.miningGrid) == 0 {
		return nil, civilization.Network{}
	}
	g := active.geometry
	n := g.Width * g.Height
	if len(active.settlementGrid) != n ||
		len(active.navigationGrid) != n ||
		len(active.miningGrid) != n {
		return nil, civilization.Network{}
	}

	grid := make([]civilizationPreviewCell, n)
	cells := make([]civilization.Cell, n)
	for gz := 0; gz < g.Height; gz++ {
		for gx := 0; gx < g.Width; gx++ {
			i := gz*g.Width + gx
			x, z := g.NodePosition(gx, gz)
			sample := active.sample(x, z)
			settlement := active.settlementGrid[i].Score
			navigation := active.navigationGrid[i].Score
			mining := active.miningGrid[i].Score
			land := sample.SourceLand &&
				sample.World.WaterKind == reconcile.WaterNone

			score := civilization.Evaluate(civilization.Inputs{
				Land:                     land,
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
			cells[i] = civilization.Cell{
				X: x, Z: z,
				GridX: gx, GridZ: gz,
				Land:                       land,
				Score:                      score,
				Buildability:               settlement.Buildability,
				TransportAccess:            settlement.TransportAccess,
				NaturalCorridorSuitability: active.transportGeoModel.Scores[i].CorridorSuitability,
				WaterwayNavigability:       navigation.WaterwayNavigability,
				NavigationBottleneckRisk:   navigation.NavigationBottleneckRisk,
			}
		}
	}

	network := civilization.BuildNetwork(
		cells,
		civilization.NetworkConfig{
			Width:               g.Width,
			Height:              g.Height,
			MinNodeSuitability:  0.60,
			LocalMaximumRadius:  2,
			MinSeparationBlocks: 768,
			MaxNodes:            72,
			MaxRelationsPerNode: 3,
			MaxRelationDistance: 6400,
		},
	)
	return grid, network
}

func civilizationScoreAt(
	active *activeWorld,
	worldX, worldZ float64,
) (civilization.Score, bool) {
	if active == nil || len(active.civilizationGrid) == 0 {
		return civilization.Score{}, false
	}
	g := active.geometry
	gx := int(math.Round((worldX - g.OriginX) / g.SpacingX()))
	gz := int(math.Round((worldZ - g.OriginZ) / g.SpacingZ()))
	gx = clampInt(gx, 0, g.Width-1)
	gz = clampInt(gz, 0, g.Height-1)
	return active.civilizationGrid[gz*g.Width+gx].Score, true
}

func civilizationScoreExact(
	active *activeWorld,
	worldX, worldZ float64,
) (civilization.Score, bool) {
	if active == nil || active.sample == nil {
		return civilization.Score{}, false
	}
	sample := active.sample(worldX, worldZ)
	settlement, ok := settlementScoreExact(active, worldX, worldZ, 16)
	if !ok {
		return civilization.Score{}, false
	}
	navigation, ok := navigationScoreExact(active, worldX, worldZ)
	if !ok {
		return civilization.Score{}, false
	}
	mining, ok := miningScoreExact(active, worldX, worldZ)
	if !ok {
		return civilization.Score{}, false
	}

	return civilization.Evaluate(civilization.Inputs{
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

func renderCivilizationPreviewLayer(
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
			score, ok := civilizationScoreAt(active, x, z)
			if !ok {
				img.SetRGBA(px, py, base)
				continue
			}
			v := clamp(score.NodeSuitability, 0, 1)
			if v < 0.48 {
				img.SetRGBA(px, py, base)
				continue
			}
			img.SetRGBA(px, py, lerp(
				base,
				civilizationSuitabilityColor(score),
				0.16+0.34*clamp((v-0.48)/0.52, 0, 1),
			))
		}
	}
	renderCivilizationRelations(img, active)
	renderCivilizationNodes(img, active)
}

func civilizationSuitabilityColor(s civilization.Score) color.RGBA {
	v := clamp(s.NodeSuitability, 0, 1)
	target := civilizationClassColor(s.NodeClass)
	return lerp(
		color.RGBA{71, 69, 63, 255},
		target,
		0.18+0.82*v,
	)
}

func civilizationClassColor(class string) color.RGBA {
	switch class {
	case "agricultural_settlement":
		return color.RGBA{91, 169, 92, 255}
	case "port_city":
		return color.RGBA{62, 143, 188, 255}
	case "mining_town":
		return color.RGBA{174, 116, 68, 255}
	case "transport_node":
		return color.RGBA{193, 161, 77, 255}
	case "multi_role_hub":
		return color.RGBA{156, 102, 183, 255}
	default:
		return color.RGBA{105, 100, 88, 255}
	}
}

func renderCivilizationRelations(img *image.RGBA, active *activeWorld) {
	if active == nil || len(active.civilizationNetwork.Nodes) == 0 {
		return
	}
	nodes := make(map[int]civilization.Node, len(active.civilizationNetwork.Nodes))
	for _, node := range active.civilizationNetwork.Nodes {
		nodes[node.ID] = node
	}
	for _, relation := range active.civilizationNetwork.Relations {
		from, okFrom := nodes[relation.From]
		to, okTo := nodes[relation.To]
		if !okFrom || !okTo {
			continue
		}
		x0, y0 := civilizationPixel(active, img.Bounds(), from.X, from.Z)
		x1, y1 := civilizationPixel(active, img.Bounds(), to.X, to.Z)
		c := color.RGBA{203, 190, 133, 180}
		switch relation.Mode {
		case "water":
			c = color.RGBA{103, 186, 220, 190}
		case "mixed":
			c = color.RGBA{171, 143, 189, 180}
		}
		drawCivilizationLine(
			img,
			x0, y0, x1, y1,
			c,
			1+int(math.Round(2*relation.CorridorSuitability)),
		)
	}
}

func renderCivilizationNodes(img *image.RGBA, active *activeWorld) {
	for _, node := range active.civilizationNetwork.Nodes {
		x, y := civilizationPixel(active, img.Bounds(), node.X, node.Z)
		radius := 3 + int(math.Round(3*node.Score.NodeSuitability))
		c := civilizationClassColor(node.Class)
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				if dx*dx+dy*dy > radius*radius {
					continue
				}
				px, py := x+dx, y+dy
				if image.Pt(px, py).In(img.Bounds()) {
					img.SetRGBA(px, py, c)
				}
			}
		}
	}
}

func civilizationPixel(
	active *activeWorld,
	bounds image.Rectangle,
	worldX, worldZ float64,
) (int, int) {
	g := active.geometry
	x := int(math.Round(
		(worldX - g.OriginX) / g.SpanXBlocks *
			float64(maxInt(bounds.Dx()-1, 1)),
	))
	y := int(math.Round(
		(worldZ - g.OriginZ) / g.SpanZBlocks *
			float64(maxInt(bounds.Dy()-1, 1)),
	))
	return x, y
}

func drawCivilizationLine(
	img *image.RGBA,
	x0, y0, x1, y1 int,
	c color.RGBA,
	width int,
) {
	dx := absInt(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -absInt(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		for oy := -width / 2; oy <= width/2; oy++ {
			for ox := -width / 2; ox <= width/2; ox++ {
				px, py := x0+ox, y0+oy
				if image.Pt(px, py).In(img.Bounds()) {
					img.SetRGBA(px, py, c)
				}
			}
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}
