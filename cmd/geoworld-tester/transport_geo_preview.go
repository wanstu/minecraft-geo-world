package main

import (
	"image"
	"image/color"
	"math"

	"geoworld/internal/transportgeo"
)

type transportHydrologyFacts struct {
	BasinID          []int
	FlowTo           []int
	FlowAccumulation []float64
	RiverOrder       []uint8
}

func activeTransportHydrology(active *activeWorld) transportHydrologyFacts {
	if active == nil {
		return transportHydrologyFacts{}
	}
	if active.model != nil && active.model.Hydrology != nil {
		return transportHydrologyFacts{
			BasinID:          active.model.Hydrology.BasinID,
			FlowTo:           active.model.Hydrology.FlowTo,
			FlowAccumulation: active.model.Hydrology.FlowAccumulation,
			RiverOrder:       active.model.Hydrology.RiverOrder,
		}
	}
	if active.reopened != nil && active.reopened.HydrologyQuery != nil {
		return transportHydrologyFacts{
			BasinID:          active.reopened.HydrologyQuery.BasinID,
			FlowTo:           active.reopened.HydrologyQuery.FlowTo,
			FlowAccumulation: active.reopened.HydrologyQuery.FlowAccumulation,
			RiverOrder:       active.reopened.HydrologyQuery.RiverOrder,
		}
	}
	return transportHydrologyFacts{}
}

func buildTransportGeography(active *activeWorld) transportgeo.Model {
	if active == nil ||
		active.sample == nil ||
		len(active.settlementGrid) == 0 ||
		len(active.navigationGrid) == 0 {
		return transportgeo.Model{}
	}
	g := active.geometry
	n := g.Width * g.Height
	if len(active.settlementGrid) != n ||
		len(active.navigationGrid) != n {
		return transportgeo.Model{}
	}

	surface := activeSurface(active)
	hydrology := activeTransportHydrology(active)
	cells := make([]transportgeo.Cell, n)
	stepX := math.Max(1, g.SpacingX())
	stepZ := math.Max(1, g.SpacingZ())
	for gz := 0; gz < g.Height; gz++ {
		for gx := 0; gx < g.Width; gx++ {
			i := gz*g.Width + gx
			x, z := g.NodePosition(gx, gz)
			sample := active.sample(x, z)
			settlement := active.settlementGrid[i].Score
			navigation := active.navigationGrid[i].Score

			macroLand := sample.SourceLand
			if surface != nil && i < len(surface.LandMask) {
				macroLand = surface.LandMask[i]
			}
			cell := transportgeo.Cell{
				X:                        x,
				Z:                        z,
				GridX:                    gx,
				GridZ:                    gz,
				Land:                     macroLand,
				WaterKind:                waterKindName(sample.World.WaterKind),
				Terrain:                  sample.TerrainVariant.String(),
				Elevation:                sample.World.GroundHeight,
				Slope:                    settlementSlope(active, x, z, stepX, stepZ),
				Buildability:             settlement.Buildability,
				FreshWaterAccess:         settlement.FreshWaterAccess,
				FloodRisk:                settlement.FloodRisk,
				TransportAccess:          settlement.TransportAccess,
				WaterwayNavigability:     navigation.WaterwayNavigability,
				NavigationBottleneckRisk: navigation.NavigationBottleneckRisk,
				FlowTo:                   -1,
			}
			if i < len(hydrology.BasinID) {
				cell.BasinID = hydrology.BasinID[i]
			}
			if i < len(hydrology.FlowTo) {
				cell.FlowTo = hydrology.FlowTo[i]
			}
			if i < len(hydrology.FlowAccumulation) {
				cell.FlowAccumulation = hydrology.FlowAccumulation[i]
			}
			if i < len(hydrology.RiverOrder) {
				cell.RiverOrder = int(hydrology.RiverOrder[i])
			}
			cells[i] = cell
		}
	}

	features := make([]transportgeo.Feature, 0, 32)
	if water := activeWaterbed(active); water != nil {
		for _, segment := range water.Rivers {
			strength := clamp(0.40+0.60*(segment.HalfWidth/48), 0, 1)
			if segment.LakeOutlet {
				features = append(features, transportgeo.Feature{
					Kind:     transportgeo.NodeLakeOutlet,
					X:        segment.X0,
					Z:        segment.Z0,
					Strength: strength,
				})
			}
			if segment.OceanMouth {
				features = append(features, transportgeo.Feature{
					Kind:     transportgeo.NodeEstuary,
					X:        segment.X1,
					Z:        segment.Z1,
					Strength: strength,
				})
			}
		}
	}

	return transportgeo.Analyze(
		cells,
		g.Width,
		g.Height,
		features,
		transportgeo.Config{
			MinNodeSuitability:  0.58,
			MinNodeSeparation:   640,
			MaxNodes:            96,
			MaxNodesPerClass:    18,
			MinRouteSuitability: 0.40,
			MinRouteCells:       3,
			MaxRoutesPerClass:   8,
		},
	)
}

func transportGeoScoreAt(
	active *activeWorld,
	worldX, worldZ float64,
) (transportgeo.Score, bool) {
	if active == nil ||
		len(active.transportGeoModel.Scores) !=
			active.geometry.Width*active.geometry.Height {
		return transportgeo.Score{}, false
	}
	g := active.geometry
	gx := int(math.Round((worldX - g.OriginX) / g.SpacingX()))
	gz := int(math.Round((worldZ - g.OriginZ) / g.SpacingZ()))
	gx = clampInt(gx, 0, g.Width-1)
	gz = clampInt(gz, 0, g.Height-1)
	return active.transportGeoModel.Scores[gz*g.Width+gx], true
}

func renderTransportGeographyLayer(
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
			score, ok := transportGeoScoreAt(active, x, z)
			if !ok {
				img.SetRGBA(px, py, base)
				continue
			}
			intensity := math.Max(
				score.CorridorSuitability,
				score.StraitIsthmusPotential,
			)
			if intensity < 0.42 {
				img.SetRGBA(px, py, base)
				continue
			}
			img.SetRGBA(px, py, lerp(
				base,
				transportGeographyColor(score),
				0.24+0.46*clamp((intensity-0.42)/0.58, 0, 1),
			))
		}
	}
	renderTransportGeographyNodes(img, active)
}

func transportGeographyColor(score transportgeo.Score) color.RGBA {
	target := color.RGBA{96, 91, 78, 255}
	switch score.CorridorClass {
	case "river_valley":
		target = color.RGBA{80, 153, 134, 255}
	case "hill_pass":
		target = color.RGBA{154, 153, 91, 255}
	case "mountain_pass":
		target = color.RGBA{177, 137, 89, 255}
	case "plateau_access":
		target = color.RGBA{151, 112, 79, 255}
	case "basin_gateway":
		target = color.RGBA{161, 126, 163, 255}
	case "low_divide":
		target = color.RGBA{191, 169, 89, 255}
	}
	if score.StraitIsthmusPotential > score.CorridorSuitability {
		target = color.RGBA{78, 160, 188, 255}
	}
	intensity := math.Max(
		score.CorridorSuitability,
		score.StraitIsthmusPotential,
	)
	return lerp(
		color.RGBA{69, 68, 64, 255},
		target,
		0.20+0.80*clamp(intensity, 0, 1),
	)
}

func renderTransportGeographyNodes(
	img *image.RGBA,
	active *activeWorld,
) {
	if active == nil {
		return
	}
	for _, node := range active.transportGeoModel.Nodes {
		x, y := civilizationPixel(
			active,
			img.Bounds(),
			node.X,
			node.Z,
		)
		radius := 2 + int(math.Round(2*node.Suitability))
		c := transportNodeColor(node.Class)
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

func transportNodeColor(class string) color.RGBA {
	switch class {
	case transportgeo.NodeConfluence:
		return color.RGBA{73, 185, 183, 255}
	case transportgeo.NodeLakeOutlet:
		return color.RGBA{67, 147, 220, 255}
	case transportgeo.NodeEstuary:
		return color.RGBA{60, 185, 221, 255}
	case transportgeo.NodeMountainPass:
		return color.RGBA{231, 159, 78, 255}
	case transportgeo.NodeBasinGateway:
		return color.RGBA{193, 115, 185, 255}
	case transportgeo.NodeLowDivide:
		return color.RGBA{228, 202, 84, 255}
	case transportgeo.NodeStrait:
		return color.RGBA{82, 201, 221, 255}
	case transportgeo.NodeIsthmus:
		return color.RGBA{224, 184, 95, 255}
	default:
		return color.RGBA{230, 230, 220, 255}
	}
}
