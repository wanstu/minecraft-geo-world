package transportgeo

import (
	"fmt"
	"math"
	"sort"
)

const (
	CorridorRiverValley   = "river_valley"
	CorridorHillPass      = "hill_pass"
	CorridorMountainPass  = "mountain_pass"
	CorridorPlateauAccess = "plateau_access"
	CorridorBasinGateway  = "basin_gateway"
	CorridorLowDivide     = "low_divide"
)

type RoutePoint struct {
	GridX       int     `json:"gridX"`
	GridZ       int     `json:"gridZ"`
	X           float64 `json:"x"`
	Z           float64 `json:"z"`
	Suitability float64 `json:"suitability"`
}

type Route struct {
	ID                 int          `json:"id"`
	Key                string       `json:"key"`
	Class              string       `json:"class"`
	ComponentCellCount int          `json:"componentCellCount"`
	PathCellCount      int          `json:"pathCellCount"`
	LengthBlocks       float64      `json:"lengthBlocks"`
	SpanBlocks         float64      `json:"spanBlocks"`
	MeanSuitability    float64      `json:"meanSuitability"`
	MinSuitability     float64      `json:"minSuitability"`
	PeakSuitability    float64      `json:"peakSuitability"`
	StartX             float64      `json:"startX"`
	StartZ             float64      `json:"startZ"`
	EndX               float64      `json:"endX"`
	EndZ               float64      `json:"endZ"`
	CoreX              float64      `json:"coreX"`
	CoreZ              float64      `json:"coreZ"`
	CoreOnInterior     bool         `json:"coreOnInterior"`
	RiverContext       float64      `json:"riverContextFraction"`
	HillContext        float64      `json:"hillContextFraction"`
	MountainContext    float64      `json:"mountainContextFraction"`
	CrossesPlateauEdge bool         `json:"crossesPlateauBoundary"`
	CrossesBasinEdge   bool         `json:"crossesBasinBoundary"`
	SemanticComplete   bool         `json:"semanticComplete"`
	Path               []RoutePoint `json:"path"`
}

type RouteSummary struct {
	TotalRoutes           int            `json:"totalRoutes"`
	CompleteRoutes        int            `json:"completeRoutes"`
	RoutesByClass         map[string]int `json:"routesByClass"`
	CompleteRoutesByClass map[string]int `json:"completeRoutesByClass"`
	LongestRouteBlocks    float64        `json:"longestRouteBlocks"`
	LongestCompleteBlocks float64        `json:"longestCompleteBlocks"`
}

func buildRoutes(
	cells []Cell,
	scores []Score,
	width, height int,
	cfg Config,
) ([]Route, RouteSummary) {
	summary := RouteSummary{
		RoutesByClass:         make(map[string]int, 6),
		CompleteRoutesByClass: make(map[string]int, 6),
	}
	if width <= 0 || height <= 0 ||
		len(cells) != width*height || len(scores) != len(cells) {
		return nil, summary
	}

	classes := [...]string{
		CorridorRiverValley,
		CorridorHillPass,
		CorridorMountainPass,
		CorridorPlateauAccess,
		CorridorBasinGateway,
		CorridorLowDivide,
	}
	spacing := routeGridSpacing(cells, width, height)
	all := make([]Route, 0, len(classes)*cfg.MaxRoutesPerClass)
	for _, class := range classes {
		classRoutes := buildClassRoutes(
			cells,
			scores,
			width,
			height,
			class,
			cfg,
			spacing,
		)
		sort.SliceStable(classRoutes, func(i, j int) bool {
			a, b := classRoutes[i], classRoutes[j]
			if a.SemanticComplete != b.SemanticComplete {
				return a.SemanticComplete
			}
			if a.LengthBlocks != b.LengthBlocks {
				return a.LengthBlocks > b.LengthBlocks
			}
			if a.MeanSuitability != b.MeanSuitability {
				return a.MeanSuitability > b.MeanSuitability
			}
			return a.Key < b.Key
		})
		if len(classRoutes) > cfg.MaxRoutesPerClass {
			classRoutes = classRoutes[:cfg.MaxRoutesPerClass]
		}
		for i := range classRoutes {
			classRoutes[i].ID = len(all) + 1
			all = append(all, classRoutes[i])
		}
	}

	for _, route := range all {
		summary.TotalRoutes++
		summary.RoutesByClass[route.Class]++
		if route.LengthBlocks > summary.LongestRouteBlocks {
			summary.LongestRouteBlocks = route.LengthBlocks
		}
		if route.SemanticComplete {
			summary.CompleteRoutes++
			summary.CompleteRoutesByClass[route.Class]++
			if route.LengthBlocks > summary.LongestCompleteBlocks {
				summary.LongestCompleteBlocks = route.LengthBlocks
			}
		}
	}
	return all, summary
}

func buildClassRoutes(
	cells []Cell,
	scores []Score,
	width, height int,
	class string,
	cfg Config,
	spacing float64,
) []Route {
	visited := make([]bool, len(cells))
	out := make([]Route, 0, 16)
	approachMinimum := routeApproachMinimum(class, cfg.MinRouteSuitability)
	for start := range cells {
		if visited[start] ||
			!routeEligible(cells[start], scores[start], class, approachMinimum) {
			continue
		}
		component := routeComponent(
			cells,
			scores,
			width,
			height,
			start,
			class,
			approachMinimum,
			visited,
		)
		if len(component) < cfg.MinRouteCells {
			continue
		}
		route := summarizeRoute(
			cells,
			scores,
			width,
			height,
			class,
			component,
			cfg,
			spacing,
		)
		if route.PathCellCount >= cfg.MinRouteCells {
			out = append(out, route)
		}
	}
	return out
}

func routeComponent(
	cells []Cell,
	scores []Score,
	width, height, start int,
	class string,
	minimum float64,
	visited []bool,
) []int {
	queue := []int{start}
	visited[start] = true
	component := make([]int, 0, 16)
	for head := 0; head < len(queue); head++ {
		index := queue[head]
		component = append(component, index)
		gx := index % width
		gz := index / width
		for _, d := range routeNeighborOffsets {
			nx, nz := gx+d[0], gz+d[1]
			if nx < 0 || nx >= width || nz < 0 || nz >= height {
				continue
			}
			neighbor := nz*width + nx
			if visited[neighbor] ||
				!routeEligible(cells[neighbor], scores[neighbor], class, minimum) {
				continue
			}
			visited[neighbor] = true
			queue = append(queue, neighbor)
		}
	}
	return component
}

var routeNeighborOffsets = [8][2]int{
	{-1, 0}, {1, 0}, {0, -1}, {0, 1},
	{-1, -1}, {1, -1}, {-1, 1}, {1, 1},
}

func summarizeRoute(
	cells []Cell,
	scores []Score,
	width, height int,
	class string,
	component []int,
	cfg Config,
	spacing float64,
) Route {
	members := make(map[int]struct{}, len(component))
	anchor := component[0]
	for _, index := range component {
		members[index] = struct{}{}
		if index < anchor {
			anchor = index
		}
	}
	a, _, _ := routeFarthest(component[0], members, width, height)
	b, _, parents := routeFarthest(a, members, width, height)
	pathIndices := routeReconstruct(a, b, parents)
	if class == CorridorPlateauAccess {
		if crossing := routePlateauCrossingPath(
			component,
			members,
			cells,
			width,
			height,
		); len(crossing) > 0 {
			pathIndices = crossing
		}
	}
	if len(pathIndices) == 0 {
		pathIndices = []int{component[0]}
	}

	peakIndex := pathIndices[0]
	peak := routeScore(scores[peakIndex], class)
	for _, index := range pathIndices[1:] {
		value := routeScore(scores[index], class)
		if value > peak || (value == peak && index < peakIndex) {
			peak = value
			peakIndex = index
		}
	}
	if class == CorridorPlateauAccess {
		if semanticPath := routeAcrossPlateauBoundary(
			peakIndex,
			cells,
			members,
			width,
			height,
		); len(semanticPath) >= cfg.MinRouteCells {
			pathIndices = semanticPath
		}
	}

	path := make([]RoutePoint, 0, len(pathIndices))
	minimum := math.Inf(1)
	sum := 0.0
	length := 0.0
	corePathIndex := -1
	riverCells, hillCells, mountainCells := 0, 0, 0
	hasPlateau, hasNonPlateau := false, false
	basins := make(map[int]struct{}, 3)
	for i, index := range pathIndices {
		cell := cells[index]
		value := routePathScore(scores[index], class)
		path = append(path, RoutePoint{
			GridX:       cell.GridX,
			GridZ:       cell.GridZ,
			X:           cell.X,
			Z:           cell.Z,
			Suitability: value,
		})
		if value < minimum {
			minimum = value
		}
		sum += value
		if i > 0 {
			previous := cells[pathIndices[i-1]]
			length += math.Hypot(cell.X-previous.X, cell.Z-previous.Z)
		}
		if index == peakIndex {
			corePathIndex = i
		}
		if riverContext(cell) {
			riverCells++
		}
		if hillContext(cell) {
			hillCells++
		}
		if mountainContext(cell) {
			mountainCells++
		}
		if normalize(cell.Terrain) == "plateau" {
			hasPlateau = true
		} else {
			hasNonPlateau = true
		}
		if cell.BasinID > 0 {
			basins[cell.BasinID] = struct{}{}
		}
	}
	if math.IsNaN(minimum) || math.IsInf(minimum, 0) {
		minimum = 0
	}
	count := float64(len(path))
	start := cells[pathIndices[0]]
	end := cells[pathIndices[len(pathIndices)-1]]
	span := math.Hypot(end.X-start.X, end.Z-start.Z)
	core := cells[peakIndex]
	anchorGX := anchor % width
	anchorGZ := anchor / width

	route := Route{
		Key: fmt.Sprintf(
			"%s-g%04d-%04d",
			class,
			anchorGX,
			anchorGZ,
		),
		Class:              class,
		ComponentCellCount: len(component),
		PathCellCount:      len(path),
		LengthBlocks:       length,
		SpanBlocks:         span,
		MeanSuitability:    sum / count,
		MinSuitability:     minimum,
		PeakSuitability:    peak,
		StartX:             start.X,
		StartZ:             start.Z,
		EndX:               end.X,
		EndZ:               end.Z,
		CoreX:              core.X,
		CoreZ:              core.Z,
		CoreOnInterior:     corePathIndex > 0 && corePathIndex+1 < len(path),
		RiverContext:       float64(riverCells) / count,
		HillContext:        float64(hillCells) / count,
		MountainContext:    float64(mountainCells) / count,
		CrossesPlateauEdge: hasPlateau && hasNonPlateau,
		CrossesBasinEdge:   len(basins) >= 2,
		Path:               path,
	}
	route.SemanticComplete = routeSemanticallyComplete(route, cfg, spacing)
	return route
}

func routeSemanticallyComplete(route Route, cfg Config, spacing float64) bool {
	approachMinimum := routeApproachMinimum(route.Class, cfg.MinRouteSuitability)
	if route.PathCellCount < cfg.MinRouteCells ||
		route.MinSuitability < approachMinimum ||
		route.PeakSuitability < cfg.MinRouteSuitability {
		return false
	}
	if spacing > 0 && route.LengthBlocks < 2*spacing {
		return false
	}
	switch route.Class {
	case CorridorRiverValley:
		return route.RiverContext >= 0.60
	case CorridorHillPass:
		return route.HillContext >= 0.75
	case CorridorMountainPass:
		return route.PeakSuitability >= cfg.MinRouteSuitability &&
			route.MountainContext >= 0.75 && route.CoreOnInterior
	case CorridorPlateauAccess:
		return route.PeakSuitability >= cfg.MinRouteSuitability &&
			route.CrossesPlateauEdge
	case CorridorBasinGateway, CorridorLowDivide:
		return route.CrossesBasinEdge
	default:
		return false
	}
}

func routeApproachMinimum(class string, minimum float64) float64 {
	if class == CorridorMountainPass {
		return math.Max(0.30, minimum-0.05)
	}
	return minimum
}

func routeEligible(cell Cell, score Score, class string, minimum float64) bool {
	if !cell.Land || routePathScore(score, class) < minimum {
		return false
	}
	switch class {
	case CorridorRiverValley:
		return riverContext(cell)
	case CorridorHillPass:
		return hillContext(cell)
	case CorridorMountainPass:
		return mountainContext(cell)
	case CorridorPlateauAccess:
		t := normalize(cell.Terrain)
		return t == "plateau" || t == "foothills" || t == "hills"
	case CorridorBasinGateway, CorridorLowDivide:
		return cell.BasinID > 0
	default:
		return false
	}
}

func routePathScore(score Score, class string) float64 {
	value := routeScore(score, class)
	switch class {
	case CorridorMountainPass, CorridorPlateauAccess:
		return math.Max(value, score.CorridorSuitability)
	default:
		return value
	}
}

func routeScore(score Score, class string) float64 {
	switch class {
	case CorridorRiverValley:
		return score.RiverValleyCorridor
	case CorridorHillPass:
		return score.HillPassCorridor
	case CorridorMountainPass:
		return score.MountainPassCorridor
	case CorridorPlateauAccess:
		return score.PlateauAccessCorridor
	case CorridorBasinGateway:
		return score.BasinGatewayCorridor
	case CorridorLowDivide:
		return score.LowDivideCorridor
	default:
		return 0
	}
}

func riverContext(cell Cell) bool {
	return cell.RiverOrder > 0 ||
		cell.FlowAccumulation > 0 ||
		cell.FreshWaterAccess >= 0.70
}

func hillContext(cell Cell) bool {
	t := normalize(cell.Terrain)
	return t == "hills" || t == "foothills"
}

func mountainContext(cell Cell) bool {
	t := normalize(cell.Terrain)
	return t == "mountain" || t == "canyon" || t == "foothills"
}

func routeAcrossPlateauBoundary(
	start int,
	cells []Cell,
	members map[int]struct{},
	width, height int,
) []int {
	if start < 0 || start >= len(cells) {
		return nil
	}
	startPlateau := normalize(cells[start].Terrain) == "plateau"
	queue := []int{start}
	distance := map[int]int{start: 0}
	parents := make(map[int]int, len(members))
	parents[start] = -1
	best, bestDistance := -1, -1
	for head := 0; head < len(queue); head++ {
		index := queue[head]
		d := distance[index]
		isPlateau := normalize(cells[index].Terrain) == "plateau"
		if isPlateau != startPlateau &&
			(d > bestDistance || (d == bestDistance && (best < 0 || index < best))) {
			best, bestDistance = index, d
		}
		gx := index % width
		gz := index / width
		for _, delta := range routeNeighborOffsets {
			nx, nz := gx+delta[0], gz+delta[1]
			if nx < 0 || nx >= width || nz < 0 || nz >= height {
				continue
			}
			neighbor := nz*width + nx
			if _, ok := members[neighbor]; !ok {
				continue
			}
			if _, seen := distance[neighbor]; seen {
				continue
			}
			distance[neighbor] = d + 1
			parents[neighbor] = index
			queue = append(queue, neighbor)
		}
	}
	if best < 0 {
		return nil
	}
	return routeReconstruct(start, best, parents)
}

func routeFarthest(
	start int,
	members map[int]struct{},
	width, height int,
) (int, int, map[int]int) {
	queue := []int{start}
	distance := map[int]int{start: 0}
	parents := make(map[int]int, len(members))
	parents[start] = -1
	best, bestDistance := start, 0
	for head := 0; head < len(queue); head++ {
		index := queue[head]
		d := distance[index]
		if d > bestDistance || (d == bestDistance && index < best) {
			best, bestDistance = index, d
		}
		gx := index % width
		gz := index / width
		for _, delta := range routeNeighborOffsets {
			nx, nz := gx+delta[0], gz+delta[1]
			if nx < 0 || nx >= width || nz < 0 || nz >= height {
				continue
			}
			neighbor := nz*width + nx
			if _, ok := members[neighbor]; !ok {
				continue
			}
			if _, seen := distance[neighbor]; seen {
				continue
			}
			distance[neighbor] = d + 1
			parents[neighbor] = index
			queue = append(queue, neighbor)
		}
	}
	return best, bestDistance, parents
}

func routePlateauCrossingPath(
	component []int,
	members map[int]struct{},
	cells []Cell,
	width, height int,
) []int {
	start := -1
	for _, index := range component {
		if normalize(cells[index].Terrain) == "plateau" &&
			(start < 0 || index < start) {
			start = index
		}
	}
	if start < 0 {
		return nil
	}
	nonPlateau, distance, parents := routeFarthestMatching(
		start,
		members,
		width,
		height,
		func(index int) bool {
			return normalize(cells[index].Terrain) != "plateau"
		},
	)
	if nonPlateau < 0 || distance < 1 {
		return nil
	}
	plateau, distance, reverseParents := routeFarthestMatching(
		nonPlateau,
		members,
		width,
		height,
		func(index int) bool {
			return normalize(cells[index].Terrain) == "plateau"
		},
	)
	if plateau >= 0 && distance >= 1 {
		if path := routeReconstruct(nonPlateau, plateau, reverseParents); len(path) > 0 {
			return path
		}
	}
	return routeReconstruct(start, nonPlateau, parents)
}

func routeFarthestMatching(
	start int,
	members map[int]struct{},
	width, height int,
	match func(int) bool,
) (int, int, map[int]int) {
	queue := []int{start}
	distance := map[int]int{start: 0}
	parents := make(map[int]int, len(members))
	parents[start] = -1
	best, bestDistance := -1, -1
	for head := 0; head < len(queue); head++ {
		index := queue[head]
		d := distance[index]
		if match(index) &&
			(d > bestDistance || (d == bestDistance && (best < 0 || index < best))) {
			best, bestDistance = index, d
		}
		gx := index % width
		gz := index / width
		for _, delta := range routeNeighborOffsets {
			nx, nz := gx+delta[0], gz+delta[1]
			if nx < 0 || nx >= width || nz < 0 || nz >= height {
				continue
			}
			neighbor := nz*width + nx
			if _, ok := members[neighbor]; !ok {
				continue
			}
			if _, seen := distance[neighbor]; seen {
				continue
			}
			distance[neighbor] = d + 1
			parents[neighbor] = index
			queue = append(queue, neighbor)
		}
	}
	return best, bestDistance, parents
}

func routeReconstruct(start, end int, parents map[int]int) []int {
	path := make([]int, 0, 16)
	for current := end; current >= 0; current = parents[current] {
		path = append(path, current)
		if current == start {
			break
		}
		parent, ok := parents[current]
		if !ok || parent < 0 {
			return nil
		}
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func routeGridSpacing(cells []Cell, width, height int) float64 {
	best := math.Inf(1)
	if width > 1 {
		for z := 0; z < height; z++ {
			for x := 1; x < width; x++ {
				a := cells[z*width+x-1]
				b := cells[z*width+x]
				d := math.Hypot(b.X-a.X, b.Z-a.Z)
				if d > 0 && d < best {
					best = d
				}
			}
		}
	}
	if height > 1 {
		for z := 1; z < height; z++ {
			for x := 0; x < width; x++ {
				a := cells[(z-1)*width+x]
				b := cells[z*width+x]
				d := math.Hypot(b.X-a.X, b.Z-a.Z)
				if d > 0 && d < best {
					best = d
				}
			}
		}
	}
	if math.IsInf(best, 1) {
		return 0
	}
	return best
}
