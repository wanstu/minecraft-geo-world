package transportgeo

import (
	"math"
	"sort"
	"strings"
)

const (
	NodeConfluence   = "confluence"
	NodeLakeOutlet   = "lake_outlet"
	NodeEstuary      = "estuary"
	NodeMountainPass = "mountain_pass"
	NodeBasinGateway = "basin_gateway"
	NodeLowDivide    = "low_divide"
	NodeStrait       = "strait"
	NodeIsthmus      = "isthmus"
)

type Cell struct {
	X, Z         float64
	GridX, GridZ int

	Land      bool
	WaterKind string
	Terrain   string

	Elevation float64
	Slope     float64

	Buildability             float64
	FreshWaterAccess         float64
	FloodRisk                float64
	TransportAccess          float64
	WaterwayNavigability     float64
	NavigationBottleneckRisk float64

	RiverOrder       int
	FlowAccumulation float64
	BasinID          int
	FlowTo           int
}

type Feature struct {
	Kind     string
	X, Z     float64
	Strength float64
}

type Score struct {
	CorridorSuitability    float64 `json:"corridorSuitability"`
	RiverValleyCorridor    float64 `json:"riverValleyCorridor"`
	HillPassCorridor       float64 `json:"hillPassCorridor"`
	MountainPassCorridor   float64 `json:"mountainPassCorridor"`
	PlateauAccessCorridor  float64 `json:"plateauAccessCorridor"`
	BasinGatewayCorridor   float64 `json:"basinGatewayCorridor"`
	LowDivideCorridor      float64 `json:"lowDivideCorridor"`
	StraitIsthmusPotential float64 `json:"straitIsthmusPotential"`
	ChokepointStrength     float64 `json:"chokepointStrength"`
	CorridorClass          string  `json:"corridorClass"`
}

type Node struct {
	ID          int     `json:"id"`
	X           float64 `json:"x"`
	Z           float64 `json:"z"`
	GridX       int     `json:"gridX"`
	GridZ       int     `json:"gridZ"`
	Class       string  `json:"class"`
	Suitability float64 `json:"suitability"`
	Reason      string  `json:"reason"`
}

type Model struct {
	Width        int          `json:"width"`
	Height       int          `json:"height"`
	Scores       []Score      `json:"-"`
	Nodes        []Node       `json:"nodes"`
	Routes       []Route      `json:"routes"`
	RouteSummary RouteSummary `json:"routeSummary"`
}

type Config struct {
	MinNodeSuitability float64
	MinNodeSeparation  float64
	MaxNodes           int
	MinNodesPerClass   int
	MaxNodesPerClass   int

	MinRouteSuitability float64
	MinRouteCells       int
	MaxRoutesPerClass   int
}

func Analyze(
	cells []Cell,
	width, height int,
	features []Feature,
	cfg Config,
) Model {
	if width <= 0 || height <= 0 || len(cells) != width*height {
		return Model{}
	}
	if cfg.MinNodeSuitability <= 0 {
		cfg.MinNodeSuitability = 0.58
	}
	if cfg.MinNodeSeparation <= 0 {
		cfg.MinNodeSeparation = 512
	}
	if cfg.MaxNodes <= 0 {
		cfg.MaxNodes = 96
	}
	if cfg.MinNodesPerClass <= 0 {
		cfg.MinNodesPerClass = 2
	}
	if cfg.MaxNodesPerClass <= 0 {
		cfg.MaxNodesPerClass = 18
	}
	if cfg.MinRouteSuitability <= 0 {
		cfg.MinRouteSuitability = 0.50
	}
	if cfg.MinRouteCells <= 0 {
		cfg.MinRouteCells = 3
	}
	if cfg.MaxRoutesPerClass <= 0 {
		cfg.MaxRoutesPerClass = 8
	}

	maxAccum := 0.0
	for _, cell := range cells {
		if cell.FlowAccumulation > maxAccum {
			maxAccum = cell.FlowAccumulation
		}
	}
	if maxAccum <= 0 {
		maxAccum = 1
	}

	scores := make([]Score, len(cells))
	candidates := make([]Node, 0, 128)
	for i, cell := range cells {
		score := evaluateCell(cells, width, height, i, maxAccum)
		scores[i] = score
		if !cell.Land {
			if score.StraitIsthmusPotential >= cfg.MinNodeSuitability {
				candidates = append(candidates, Node{
					X: cell.X, Z: cell.Z,
					GridX: cell.GridX, GridZ: cell.GridZ,
					Class:       NodeStrait,
					Suitability: score.StraitIsthmusPotential,
					Reason:      "opposed land masses frame a narrow natural water crossing",
				})
			}
			continue
		}

		confluence := confluenceScore(cells, width, height, i, maxAccum)
		appendCandidate := func(class string, suitability float64, reason string) {
			if suitability < cfg.MinNodeSuitability {
				return
			}
			candidates = append(candidates, Node{
				X: cell.X, Z: cell.Z,
				GridX: cell.GridX, GridZ: cell.GridZ,
				Class:       class,
				Suitability: clamp01(suitability),
				Reason:      reason,
			})
		}
		appendCandidate(
			NodeConfluence,
			confluence,
			"multiple upstream drainage branches converge into one natural corridor",
		)
		appendCandidate(
			NodeMountainPass,
			score.MountainPassCorridor,
			"terrain forms a low saddle through otherwise costly mountain ground",
		)
		appendCandidate(
			NodeBasinGateway,
			score.BasinGatewayCorridor,
			"basin boundary and local relief concentrate movement through a limited outlet",
		)
		appendCandidate(
			NodeLowDivide,
			score.LowDivideCorridor,
			"low-relief basin boundary offers a practical cross-watershed connection",
		)
		if score.StraitIsthmusPotential >= cfg.MinNodeSuitability {
			appendCandidate(
				NodeIsthmus,
				score.StraitIsthmusPotential,
				"opposed water bodies narrow the land connection into an isthmus",
			)
		}
	}

	for _, feature := range features {
		class := strings.TrimSpace(feature.Kind)
		if class != NodeLakeOutlet && class != NodeEstuary {
			continue
		}
		index := nearestCell(cells, feature.X, feature.Z)
		if index < 0 {
			continue
		}
		cell := cells[index]
		if !cell.Land {
			if next := nearestDryCell(cells, width, height, index); next >= 0 {
				index = next
				cell = cells[index]
			}
		}
		strength := clamp01(feature.Strength)
		suitability := 0.55*strength +
			0.20*clamp01(cell.TransportAccess) +
			0.15*clamp01(cell.Buildability)
		reason := "large lake outlet concentrates a natural lake-to-river transition"
		if class == NodeEstuary {
			suitability += 0.10 * clamp01(cell.WaterwayNavigability)
			reason = "river reaches the ocean at a natural transport and transfer node"
		} else {
			suitability += 0.10 * clamp01(cell.FreshWaterAccess)
		}
		featureMinimum := math.Min(cfg.MinNodeSuitability, 0.35)
		if suitability >= featureMinimum {
			candidates = append(candidates, Node{
				X: cell.X, Z: cell.Z,
				GridX: cell.GridX, GridZ: cell.GridZ,
				Class:       class,
				Suitability: clamp01(suitability),
				Reason:      reason,
			})
		}
	}

	nodes := selectSparseNodes(candidates, cfg)
	routes, routeSummary := buildRoutes(cells, scores, width, height, cfg)
	return Model{
		Width:        width,
		Height:       height,
		Scores:       scores,
		Nodes:        nodes,
		Routes:       routes,
		RouteSummary: routeSummary,
	}
}

func evaluateCell(
	cells []Cell,
	width, height, index int,
	maxAccum float64,
) Score {
	cell := cells[index]
	straitIsthmus := straitIsthmusScore(cells, width, height, index)
	if !cell.Land {
		return Score{
			StraitIsthmusPotential: straitIsthmus,
			ChokepointStrength:     straitIsthmus,
			CorridorClass:          "none",
		}
	}

	lowSlope := 1 - smoothstep(0.0015, 0.014, math.Max(0, cell.Slope))
	build := clamp01(cell.Buildability)
	transport := clamp01(cell.TransportAccess)
	fresh := clamp01(cell.FreshWaterAccess)
	floodSafety := 1 - clamp01(cell.FloodRisk)
	saddle := saddleScore(cells, width, height, index)
	basinBoundary := basinBoundaryScore(cells, width, height, index)
	plateauEdge := plateauEdgeScore(cells, width, height, index)
	riverStrength := riverStrength(cell, maxAccum)

	riverValley := clamp01(
		0.28*transport +
			0.22*fresh +
			0.18*lowSlope +
			0.14*build +
			0.12*riverStrength +
			0.06*floodSafety,
	)

	hillContext := terrainWeight(cell.Terrain, "hills", "foothills")
	hillPass := clamp01(
		hillContext *
			(0.34*transport +
				0.24*build +
				0.22*saddle +
				0.20*lowSlope),
	)

	mountainContext := terrainWeight(cell.Terrain, "mountain", "canyon", "foothills")
	mountainPass := clamp01(
		mountainContext *
			(0.44*saddle +
				0.24*transport +
				0.18*lowSlope +
				0.14*build),
	)

	plateauContext := terrainWeight(cell.Terrain, "plateau", "foothills", "hills")
	plateauAccess := clamp01(
		plateauContext *
			(0.36*plateauEdge +
				0.28*transport +
				0.20*build +
				0.16*lowSlope),
	)

	basinGateway := clamp01(
		basinBoundary *
			(0.32*transport +
				0.22*build +
				0.20*lowSlope +
				0.14*fresh +
				0.12*(1-saddle)),
	)

	lowDivide := clamp01(
		basinBoundary *
			(0.34*lowSlope +
				0.28*transport +
				0.18*build +
				0.12*(1-saddle) +
				0.08*floodSafety),
	)

	values := []float64{
		riverValley,
		hillPass,
		mountainPass,
		plateauAccess,
		basinGateway,
		lowDivide,
	}
	classes := []string{
		"river_valley",
		"hill_pass",
		"mountain_pass",
		"plateau_access",
		"basin_gateway",
		"low_divide",
	}
	best := 0
	for i := 1; i < len(values); i++ {
		if values[i] > values[best] {
			best = i
		}
	}
	corridorClass := classes[best]
	if values[best] < 0.34 {
		corridorClass = "low_potential"
	}
	chokepoint := math.Max(
		math.Max(mountainPass, hillPass),
		math.Max(
			math.Max(basinGateway, lowDivide),
			straitIsthmus,
		),
	)
	return Score{
		CorridorSuitability:    clamp01(values[best]),
		RiverValleyCorridor:    riverValley,
		HillPassCorridor:       hillPass,
		MountainPassCorridor:   mountainPass,
		PlateauAccessCorridor:  plateauAccess,
		BasinGatewayCorridor:   basinGateway,
		LowDivideCorridor:      lowDivide,
		StraitIsthmusPotential: straitIsthmus,
		ChokepointStrength:     clamp01(chokepoint),
		CorridorClass:          corridorClass,
	}
}

func confluenceScore(
	cells []Cell,
	width, height, index int,
	maxAccum float64,
) float64 {
	cell := cells[index]
	if !cell.Land || cell.RiverOrder <= 0 {
		return 0
	}
	incoming := 0
	strongestIncoming := 0.0
	x, z := cell.GridX, cell.GridZ
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dz == 0 {
				continue
			}
			nx, nz := x+dx, z+dz
			if nx < 0 || nx >= width || nz < 0 || nz >= height {
				continue
			}
			ni := nz*width + nx
			n := cells[ni]
			if n.FlowTo != index || n.RiverOrder <= 0 {
				continue
			}
			incoming++
			if strength := riverStrength(n, maxAccum); strength > strongestIncoming {
				strongestIncoming = strength
			}
		}
	}
	if incoming < 2 {
		return 0
	}
	incomingScore := clamp01(float64(incoming-1) / 3)
	return clamp01(
		0.32*incomingScore +
			0.28*riverStrength(cell, maxAccum) +
			0.18*strongestIncoming +
			0.12*clamp01(cell.TransportAccess) +
			0.10*clamp01(cell.Buildability),
	)
}

func riverStrength(cell Cell, maxAccum float64) float64 {
	order := clamp01(float64(cell.RiverOrder) / 5)
	accum := 0.0
	if maxAccum > 0 && cell.FlowAccumulation > 0 {
		accum = math.Sqrt(clamp01(cell.FlowAccumulation / maxAccum))
	}
	return math.Max(order, accum)
}

func saddleScore(cells []Cell, width, height, index int) float64 {
	cell := cells[index]
	x, z := cell.GridX, cell.GridZ
	if x <= 0 || x+1 >= width || z <= 0 || z+1 >= height {
		return 0
	}
	n := cells[(z-1)*width+x].Elevation
	s := cells[(z+1)*width+x].Elevation
	w := cells[z*width+x-1].Elevation
	e := cells[z*width+x+1].Elevation
	c := cell.Elevation

	ns := math.Min(math.Max(0, n-c), math.Max(0, s-c))
	ew := math.Min(math.Max(0, e-c), math.Max(0, w-c))
	barrier := math.Max(ns, ew)
	open := math.Min(ns, ew)
	barrierScore := smoothstep(0.006, 0.055, barrier)
	anisotropy := clamp01((barrier - open) / 0.035)
	return clamp01(barrierScore * (0.45 + 0.55*anisotropy))
}

func basinBoundaryScore(cells []Cell, width, height, index int) float64 {
	cell := cells[index]
	if cell.BasinID <= 0 {
		return 0
	}
	seen := map[int]struct{}{cell.BasinID: {}}
	x, z := cell.GridX, cell.GridZ
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, nz := x+d[0], z+d[1]
		if nx < 0 || nx >= width || nz < 0 || nz >= height {
			continue
		}
		b := cells[nz*width+nx].BasinID
		if b > 0 {
			seen[b] = struct{}{}
		}
	}
	if len(seen) < 2 {
		return 0
	}
	return clamp01(0.55 + 0.15*float64(len(seen)-2))
}

func plateauEdgeScore(cells []Cell, width, height, index int) float64 {
	cell := cells[index]
	isPlateau := normalize(cell.Terrain) == "plateau"
	x, z := cell.GridX, cell.GridZ
	transitions := 0
	neighbors := 0
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, nz := x+d[0], z+d[1]
		if nx < 0 || nx >= width || nz < 0 || nz >= height {
			continue
		}
		neighbors++
		other := normalize(cells[nz*width+nx].Terrain) == "plateau"
		if other != isPlateau {
			transitions++
		}
	}
	if neighbors == 0 || transitions == 0 {
		return 0
	}
	return clamp01(float64(transitions) / float64(neighbors))
}

func straitIsthmusScore(cells []Cell, width, height, index int) float64 {
	cell := cells[index]
	x, z := cell.GridX, cell.GridZ
	if x <= 0 || x+1 >= width || z <= 0 || z+1 >= height {
		return 0
	}

	const maxRadius = 4
	best := 0.0
	if cell.Land {
		if cells[z*width+x-1].Land && cells[z*width+x+1].Land {
			if a, okA := firstStateDistance(cells, width, height, x, z, 0, -1, false, maxRadius); okA {
				if b, okB := firstStateDistance(cells, width, height, x, z, 0, 1, false, maxRadius); okB {
					best = math.Max(best, crossingNarrowness(a, b, maxRadius))
				}
			}
		}
		if cells[(z-1)*width+x].Land && cells[(z+1)*width+x].Land {
			if a, okA := firstStateDistance(cells, width, height, x, z, -1, 0, false, maxRadius); okA {
				if b, okB := firstStateDistance(cells, width, height, x, z, 1, 0, false, maxRadius); okB {
					best = math.Max(best, crossingNarrowness(a, b, maxRadius))
				}
			}
		}
		return best
	}

	if !cells[z*width+x-1].Land && !cells[z*width+x+1].Land {
		if a, okA := firstStateDistance(cells, width, height, x, z, 0, -1, true, maxRadius); okA {
			if b, okB := firstStateDistance(cells, width, height, x, z, 0, 1, true, maxRadius); okB {
				best = math.Max(best, crossingNarrowness(a, b, maxRadius))
			}
		}
	}
	if !cells[(z-1)*width+x].Land && !cells[(z+1)*width+x].Land {
		if a, okA := firstStateDistance(cells, width, height, x, z, -1, 0, true, maxRadius); okA {
			if b, okB := firstStateDistance(cells, width, height, x, z, 1, 0, true, maxRadius); okB {
				best = math.Max(best, crossingNarrowness(a, b, maxRadius))
			}
		}
	}
	return best
}

func firstStateDistance(
	cells []Cell,
	width, height, x, z, dx, dz int,
	wantLand bool,
	maxRadius int,
) (int, bool) {
	for distance := 1; distance <= maxRadius; distance++ {
		nx, nz := x+dx*distance, z+dz*distance
		if nx < 0 || nx >= width || nz < 0 || nz >= height {
			return 0, false
		}
		if cells[nz*width+nx].Land == wantLand {
			return distance, true
		}
	}
	return 0, false
}

func crossingNarrowness(a, b, maxRadius int) float64 {
	if a <= 0 || b <= 0 || maxRadius <= 1 {
		return 0
	}
	excess := float64(a+b-2) / float64(2*(maxRadius-1))
	return clamp01(1 - 0.30*excess)
}

func terrainWeight(terrain string, values ...string) float64 {
	t := normalize(terrain)
	for i, value := range values {
		if t == value {
			switch i {
			case 0:
				return 1
			case 1:
				return 0.78
			default:
				return 0.58
			}
		}
	}
	return 0
}

func nearestCell(cells []Cell, x, z float64) int {
	best := -1
	bestDistance := math.Inf(1)
	for i, cell := range cells {
		d := math.Hypot(cell.X-x, cell.Z-z)
		if d < bestDistance {
			best = i
			bestDistance = d
		}
	}
	return best
}

func nearestDryCell(cells []Cell, width, height, index int) int {
	if index < 0 || index >= len(cells) {
		return -1
	}
	origin := cells[index]
	best := -1
	bestDistance := math.Inf(1)
	for dz := -2; dz <= 2; dz++ {
		for dx := -2; dx <= 2; dx++ {
			x, z := origin.GridX+dx, origin.GridZ+dz
			if x < 0 || x >= width || z < 0 || z >= height {
				continue
			}
			i := z*width + x
			if !cells[i].Land {
				continue
			}
			d := math.Hypot(cells[i].X-origin.X, cells[i].Z-origin.Z)
			if d < bestDistance {
				best = i
				bestDistance = d
			}
		}
	}
	return best
}

func selectSparseNodes(candidates []Node, cfg Config) []Node {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Suitability == candidates[j].Suitability {
			if candidates[i].Class == candidates[j].Class {
				if candidates[i].Z == candidates[j].Z {
					return candidates[i].X < candidates[j].X
				}
				return candidates[i].Z < candidates[j].Z
			}
			return candidates[i].Class < candidates[j].Class
		}
		return candidates[i].Suitability > candidates[j].Suitability
	})
	out := make([]Node, 0, minInt(cfg.MaxNodes, len(candidates)))
	classCounts := make(map[string]int)
	tryAdd := func(candidate Node) bool {
		if len(out) >= cfg.MaxNodes ||
			classCounts[candidate.Class] >= cfg.MaxNodesPerClass {
			return false
		}
		for _, node := range out {
			if node.Class == candidate.Class &&
				math.Hypot(node.X-candidate.X, node.Z-candidate.Z) <
					cfg.MinNodeSeparation {
				return false
			}
		}
		candidate.ID = len(out) + 1
		out = append(out, candidate)
		classCounts[candidate.Class]++
		return true
	}

	// Preserve a few scarce but semantically explicit node classes before the
	// global score fill. This prevents real lake outlets/confluences from being
	// crowded out by many high-scoring basin/divide candidates.
	classOrder := [...]string{
		NodeLakeOutlet,
		NodeEstuary,
		NodeConfluence,
		NodeMountainPass,
		NodeBasinGateway,
		NodeLowDivide,
		NodeStrait,
		NodeIsthmus,
	}
	for _, class := range classOrder {
		for _, candidate := range candidates {
			if classCounts[class] >= cfg.MinNodesPerClass ||
				len(out) >= cfg.MaxNodes {
				break
			}
			if candidate.Class == class {
				tryAdd(candidate)
			}
		}
	}

	for _, candidate := range candidates {
		if len(out) >= cfg.MaxNodes {
			break
		}
		tryAdd(candidate)
	}
	return out
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge1 <= edge0 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v <= 0 {
		return 0
	}
	if v >= 1 {
		return 1
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
