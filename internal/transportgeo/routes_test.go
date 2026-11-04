package transportgeo

import "testing"

func TestRouteExtractionRejectsSingleHighPoint(t *testing.T) {
	const width, height = 5, 5
	cells := baseGrid(width, height)
	scores := make([]Score, len(cells))
	center := 2*width + 2
	cells[center].Terrain = "Mountain"
	scores[center].MountainPassCorridor = .95

	routes, summary := buildRoutes(cells, scores, width, height, Config{
		MinRouteSuitability: .50,
		MinRouteCells:       3,
		MaxRoutesPerClass:   8,
	})
	if len(routes) != 0 || summary.TotalRoutes != 0 {
		t.Fatalf("single high point must not become a route: %+v", routes)
	}
}

func TestContinuousRiverValleyBecomesCompleteRoute(t *testing.T) {
	const width, height = 5, 3
	cells := baseGrid(width, height)
	scores := make([]Score, len(cells))
	for x := 0; x < width; x++ {
		i := width + x
		cells[i].RiverOrder = 2
		cells[i].FreshWaterAccess = .92
		scores[i].RiverValleyCorridor = .76
	}
	routes, summary := buildRoutes(cells, scores, width, height, Config{
		MinRouteSuitability: .50,
		MinRouteCells:       3,
		MaxRoutesPerClass:   8,
	})
	route, ok := completeRoute(routes, CorridorRiverValley)
	if !ok {
		t.Fatalf("continuous river valley missing: routes=%+v summary=%+v", routes, summary)
	}
	if route.PathCellCount != 5 ||
		route.LengthBlocks != 4*128 ||
		route.RiverContext != 1 {
		t.Fatalf("unexpected river route: %+v", route)
	}
}

func TestMountainPassNeedsContinuousApproachesOnBothSides(t *testing.T) {
	const width, height = 5, 3
	cells := baseGrid(width, height)
	scores := make([]Score, len(cells))
	values := []float64{.58, .72, .96, .70, .57}
	for x, value := range values {
		i := width + x
		cells[i].Terrain = "Mountain"
		scores[i].MountainPassCorridor = value
	}
	routes, _ := buildRoutes(cells, scores, width, height, Config{
		MinRouteSuitability: .50,
		MinRouteCells:       3,
		MaxRoutesPerClass:   8,
	})
	route, ok := completeRoute(routes, CorridorMountainPass)
	if !ok {
		t.Fatalf("complete mountain route missing: %+v", routes)
	}
	if !route.CoreOnInterior || route.PathCellCount != 5 ||
		route.MountainContext != 1 {
		t.Fatalf("mountain approaches are not represented: %+v", route)
	}
}

func TestPlateauAccessRequiresCrossingPlateauBoundary(t *testing.T) {
	const width, height = 5, 3
	cells := baseGrid(width, height)
	scores := make([]Score, len(cells))
	terrain := []string{"Hills", "Foothills", "Plateau", "Plateau", "Plateau"}
	for x, value := range terrain {
		i := width + x
		cells[i].Terrain = value
		scores[i].PlateauAccessCorridor = .74
	}
	routes, _ := buildRoutes(cells, scores, width, height, Config{
		MinRouteSuitability: .50,
		MinRouteCells:       3,
		MaxRoutesPerClass:   8,
	})
	route, ok := completeRoute(routes, CorridorPlateauAccess)
	if !ok || !route.CrossesPlateauEdge {
		t.Fatalf("plateau boundary route missing: %+v", routes)
	}
}

func TestBasinGatewayAndLowDivideRequireCrossBasinPath(t *testing.T) {
	const width, height = 5, 3
	cells := baseGrid(width, height)
	scores := make([]Score, len(cells))
	for x := 0; x < width; x++ {
		i := width + x
		if x >= 3 {
			cells[i].BasinID = 2
		}
		scores[i].BasinGatewayCorridor = .78
		scores[i].LowDivideCorridor = .72
	}
	routes, summary := buildRoutes(cells, scores, width, height, Config{
		MinRouteSuitability: .50,
		MinRouteCells:       3,
		MaxRoutesPerClass:   8,
	})
	for _, class := range []string{CorridorBasinGateway, CorridorLowDivide} {
		route, ok := completeRoute(routes, class)
		if !ok || !route.CrossesBasinEdge {
			t.Fatalf("%s cross-basin route missing: routes=%+v summary=%+v", class, routes, summary)
		}
	}
}

func TestHillRouteIsContinuousRatherThanSingleLowPoint(t *testing.T) {
	const width, height = 5, 3
	cells := baseGrid(width, height)
	scores := make([]Score, len(cells))
	for x := 0; x < width; x++ {
		i := width + x
		cells[i].Terrain = "Hills"
		scores[i].HillPassCorridor = .71
	}
	routes, _ := buildRoutes(cells, scores, width, height, Config{
		MinRouteSuitability: .50,
		MinRouteCells:       3,
		MaxRoutesPerClass:   8,
	})
	route, ok := completeRoute(routes, CorridorHillPass)
	if !ok || route.HillContext != 1 || route.PathCellCount != 5 {
		t.Fatalf("continuous hill route missing: %+v", routes)
	}
}

func TestMountainPassAllowsLowerScoringApproachesAroundStrongCore(t *testing.T) {
	const width, height = 5, 3
	cells := baseGrid(width, height)
	scores := make([]Score, len(cells))
	values := []float64{.35, .39, .72, .38, .35}
	for x, value := range values {
		i := width + x
		cells[i].Terrain = "Mountain"
		scores[i].MountainPassCorridor = value
	}
	routes, _ := buildRoutes(cells, scores, width, height, Config{
		MinRouteSuitability: .40,
		MinRouteCells:       3,
		MaxRoutesPerClass:   8,
	})
	route, ok := completeRoute(routes, CorridorMountainPass)
	if !ok {
		t.Fatalf("mountain route should preserve lower-scoring approaches around a strong core: %+v", routes)
	}
	if !route.CoreOnInterior || route.PeakSuitability < .40 ||
		route.MinSuitability < .35 {
		t.Fatalf("mountain approach/core constraints not preserved: %+v", route)
	}
}

func completeRoute(routes []Route, class string) (Route, bool) {
	for _, route := range routes {
		if route.Class == class && route.SemanticComplete {
			return route, true
		}
	}
	return Route{}, false
}
