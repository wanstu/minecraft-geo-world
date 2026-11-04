package transportgeo

import (
	"math"
	"testing"
)

func baseGrid(width, height int) []Cell {
	cells := make([]Cell, width*height)
	for z := 0; z < height; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			cells[i] = Cell{
				X:                float64(x * 128),
				Z:                float64(z * 128),
				GridX:            x,
				GridZ:            z,
				Land:             true,
				WaterKind:        "None",
				Terrain:          "Plain",
				Elevation:        0.50,
				Slope:            0.002,
				Buildability:     0.90,
				FreshWaterAccess: 0.55,
				FloodRisk:        0.08,
				TransportAccess:  0.85,
				BasinID:          1,
				FlowTo:           -1,
			}
		}
	}
	return cells
}

func TestMountainSaddleBecomesNaturalPass(t *testing.T) {
	const width, height = 5, 5
	cells := baseGrid(width, height)
	center := 2*width + 2
	cells[center].Terrain = "Mountain"
	cells[center].Buildability = 0.58
	cells[center].TransportAccess = 0.68
	cells[center].Elevation = 0.55
	cells[(2-1)*width+2].Elevation = 0.64
	cells[(2+1)*width+2].Elevation = 0.64
	cells[2*width+1].Elevation = 0.555
	cells[2*width+3].Elevation = 0.555

	model := Analyze(cells, width, height, nil, Config{
		MinNodeSuitability: 0.50,
		MinNodeSeparation:  1,
	})
	score := model.Scores[center]
	if score.MountainPassCorridor < 0.50 {
		t.Fatalf("mountain pass too weak: %+v", score)
	}
	found := false
	for _, node := range model.Nodes {
		if node.Class == NodeMountainPass &&
			node.GridX == 2 && node.GridZ == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("center saddle missing mountain-pass node: %+v", model.Nodes)
	}
}

func TestFlatBasinBoundaryCreatesLowDivide(t *testing.T) {
	const width, height = 5, 5
	cells := baseGrid(width, height)
	center := 2*width + 2
	cells[center].BasinID = 1
	cells[center].Terrain = "Rolling Plain"
	cells[center].TransportAccess = 0.92
	cells[center].Buildability = 0.93
	cells[center].Slope = 0.001
	cells[2*width+3].BasinID = 2
	cells[2*width+1].BasinID = 2

	model := Analyze(cells, width, height, nil, Config{
		MinNodeSuitability: 0.45,
		MinNodeSeparation:  1,
	})
	score := model.Scores[center]
	if score.LowDivideCorridor < 0.45 {
		t.Fatalf("low divide too weak: %+v", score)
	}
	found := false
	for _, node := range model.Nodes {
		if node.Class == NodeLowDivide &&
			node.GridX == 2 && node.GridZ == 2 {
			found = true
		}
	}
	if !found {
		t.Fatalf("low divide node missing: %+v", model.Nodes)
	}
}

func TestDrainageConfluenceCreatesNaturalNode(t *testing.T) {
	const width, height = 5, 5
	cells := baseGrid(width, height)
	center := 2*width + 2
	cells[center].RiverOrder = 3
	cells[center].FlowAccumulation = 1000
	cells[center].FreshWaterAccess = 1
	cells[center].TransportAccess = .9

	left := 2*width + 1
	up := 1*width + 2
	cells[left].RiverOrder = 2
	cells[left].FlowAccumulation = 600
	cells[left].FlowTo = center
	cells[up].RiverOrder = 2
	cells[up].FlowAccumulation = 550
	cells[up].FlowTo = center

	model := Analyze(cells, width, height, nil, Config{
		MinNodeSuitability: 0.48,
		MinNodeSeparation:  1,
	})
	found := false
	for _, node := range model.Nodes {
		if node.Class == NodeConfluence &&
			node.GridX == 2 && node.GridZ == 2 {
			if node.Suitability < 0.48 {
				t.Fatalf("confluence suitability too low: %+v", node)
			}
			found = true
		}
	}
	if !found {
		t.Fatalf("confluence node missing: %+v", model.Nodes)
	}
}

func TestStraitAndIsthmusMorphology(t *testing.T) {
	const width, height = 5, 5

	strait := baseGrid(width, height)
	center := 2*width + 2
	strait[center].Land = false
	strait[center].WaterKind = "Ocean"
	strait[(2-1)*width+2].Land = true
	strait[(2+1)*width+2].Land = true
	strait[2*width+1].Land = false
	strait[2*width+1].WaterKind = "Ocean"
	strait[2*width+3].Land = false
	strait[2*width+3].WaterKind = "Ocean"

	model := Analyze(strait, width, height, nil, Config{
		MinNodeSuitability: 0.9,
		MinNodeSeparation:  1,
	})
	if model.Scores[center].StraitIsthmusPotential != 1 {
		t.Fatalf("strait potential=%v", model.Scores[center].StraitIsthmusPotential)
	}
	if !hasNode(model.Nodes, NodeStrait, 2, 2) {
		t.Fatalf("strait node missing: %+v", model.Nodes)
	}

	isthmus := baseGrid(width, height)
	isthmus[(2-1)*width+2].Land = false
	isthmus[(2-1)*width+2].WaterKind = "Ocean"
	isthmus[(2+1)*width+2].Land = false
	isthmus[(2+1)*width+2].WaterKind = "Ocean"
	isthmus[2*width+1].Land = true
	isthmus[2*width+3].Land = true

	model = Analyze(isthmus, width, height, nil, Config{
		MinNodeSuitability: 0.9,
		MinNodeSeparation:  1,
	})
	if model.Scores[center].StraitIsthmusPotential != 1 {
		t.Fatalf("isthmus potential=%v", model.Scores[center].StraitIsthmusPotential)
	}
	if !hasNode(model.Nodes, NodeIsthmus, 2, 2) {
		t.Fatalf("isthmus node missing: %+v", model.Nodes)
	}
}

func TestLakeOutletAndEstuaryFeaturesBecomeNodes(t *testing.T) {
	const width, height = 5, 5
	cells := baseGrid(width, height)
	features := []Feature{
		{Kind: NodeLakeOutlet, X: 128, Z: 128, Strength: .95},
		{Kind: NodeEstuary, X: 384, Z: 384, Strength: .92},
	}
	for i := range cells {
		cells[i].FreshWaterAccess = .9
		cells[i].WaterwayNavigability = .9
	}
	model := Analyze(cells, width, height, features, Config{
		MinNodeSuitability: .55,
		MinNodeSeparation:  1,
	})
	if !hasClass(model.Nodes, NodeLakeOutlet) {
		t.Fatalf("lake outlet node missing: %+v", model.Nodes)
	}
	if !hasClass(model.Nodes, NodeEstuary) {
		t.Fatalf("estuary node missing: %+v", model.Nodes)
	}
}

func TestWaterHasNoLandCorridor(t *testing.T) {
	const width, height = 3, 3
	cells := baseGrid(width, height)
	center := width + 1
	cells[center].Land = false
	cells[center].WaterKind = "Ocean"
	cells[center].Buildability = 1
	cells[center].TransportAccess = 1
	cells[center].FreshWaterAccess = 1

	model := Analyze(cells, width, height, nil, Config{})
	score := model.Scores[center]
	if score.CorridorSuitability != 0 ||
		score.RiverValleyCorridor != 0 ||
		score.MountainPassCorridor != 0 {
		t.Fatalf("water gained land corridor: %+v", score)
	}
	if math.IsNaN(score.ChokepointStrength) {
		t.Fatal("water chokepoint is NaN")
	}
}

func hasNode(nodes []Node, class string, x, z int) bool {
	for _, node := range nodes {
		if node.Class == class && node.GridX == x && node.GridZ == z {
			return true
		}
	}
	return false
}

func hasClass(nodes []Node, class string) bool {
	for _, node := range nodes {
		if node.Class == class {
			return true
		}
	}
	return false
}
