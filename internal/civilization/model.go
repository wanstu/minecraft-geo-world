package civilization

import (
	"math"
	"sort"
)

type Inputs struct {
	Land bool

	SettlementSuitability  float64
	Buildability           float64
	FreshWaterAccess       float64
	FloodRisk              float64
	AgricultureSuitability float64
	TransportAccess        float64
	ResourceAccess         float64

	WaterwayNavigability     float64
	PortSuitability          float64
	NavigationBottleneckRisk float64

	MiningSuitability float64
	IndustrialValue   float64
	ResourceDiversity float64
}

type Score struct {
	AgriculturalSettlementSuitability float64 `json:"agriculturalSettlementSuitability"`
	PortCitySuitability               float64 `json:"portCitySuitability"`
	MiningTownSuitability             float64 `json:"miningTownSuitability"`
	TransportNodeSuitability          float64 `json:"transportNodeSuitability"`
	NodeSuitability                   float64 `json:"nodeSuitability"`
	SpecializationStrength            float64 `json:"specializationStrength"`
	NodeClass                         string  `json:"nodeClass"`
}

func Evaluate(in Inputs) Score {
	if !in.Land {
		return Score{NodeClass: "none"}
	}

	settlement := clamp01(in.SettlementSuitability)
	build := clamp01(in.Buildability)
	fresh := clamp01(in.FreshWaterAccess)
	floodSafety := 1 - clamp01(in.FloodRisk)
	agriculture := clamp01(in.AgricultureSuitability)
	transport := clamp01(in.TransportAccess)
	resource := clamp01(in.ResourceAccess)
	waterway := clamp01(in.WaterwayNavigability)
	port := clamp01(in.PortSuitability)
	navSafety := 1 - clamp01(in.NavigationBottleneckRisk)
	mining := clamp01(in.MiningSuitability)
	industrial := clamp01(in.IndustrialValue)
	diversity := clamp01(in.ResourceDiversity)

	agri := clamp01(
		0.28*settlement +
			0.24*agriculture +
			0.15*fresh +
			0.13*build +
			0.10*transport +
			0.10*floodSafety,
	)

	portCity := clamp01(
		0.32*port +
			0.18*settlement +
			0.14*transport +
			0.10*build +
			0.09*fresh +
			0.09*industrial +
			0.08*navSafety,
	)
	if port < 0.20 {
		portCity *= 0.55 + 1.75*port
	}

	miningTown := clamp01(
		0.31*mining +
			0.17*industrial +
			0.13*resource +
			0.13*settlement +
			0.11*transport +
			0.08*build +
			0.07*diversity,
	)

	transportNode := clamp01(
		0.28*transport +
			0.18*settlement +
			0.15*build +
			0.13*port +
			0.11*waterway +
			0.08*mining +
			0.07*navSafety,
	)

	values := []float64{agri, portCity, miningTown, transportNode}
	labels := []string{
		"agricultural_settlement",
		"port_city",
		"mining_town",
		"transport_node",
	}
	order := []int{0, 1, 2, 3}
	sort.SliceStable(order, func(i, j int) bool {
		return values[order[i]] > values[order[j]]
	})
	best := values[order[0]]
	second := values[order[1]]

	overall := clamp01(
		0.68*best +
			0.20*second +
			0.12*settlement,
	)
	specialization := clamp01((best - second) / 0.24)

	class := labels[order[0]]
	switch {
	case overall < 0.45:
		class = "low_potential"
	case best >= 0.68 && second >= 0.64 && best-second <= 0.07:
		class = "multi_role_hub"
	}

	return Score{
		AgriculturalSettlementSuitability: agri,
		PortCitySuitability:               portCity,
		MiningTownSuitability:             miningTown,
		TransportNodeSuitability:          transportNode,
		NodeSuitability:                   overall,
		SpecializationStrength:            specialization,
		NodeClass:                         class,
	}
}

type Cell struct {
	X, Z  float64
	GridX int
	GridZ int
	Land  bool
	Score Score

	Buildability               float64
	TransportAccess            float64
	NaturalCorridorSuitability float64
	WaterwayNavigability       float64
	NavigationBottleneckRisk   float64
}

type Node struct {
	ID    int     `json:"id"`
	X     float64 `json:"x"`
	Z     float64 `json:"z"`
	GridX int     `json:"gridX"`
	GridZ int     `json:"gridZ"`
	Class string  `json:"class"`
	Score Score   `json:"score"`
}

type Relation struct {
	From                int     `json:"from"`
	To                  int     `json:"to"`
	DistanceBlocks      float64 `json:"distanceBlocks"`
	CorridorSuitability float64 `json:"corridorSuitability"`
	Mode                string  `json:"mode"`
}

type Network struct {
	Nodes     []Node     `json:"nodes"`
	Relations []Relation `json:"relations"`
}

type NetworkConfig struct {
	Width, Height       int
	MinNodeSuitability  float64
	LocalMaximumRadius  int
	MinSeparationBlocks float64
	MaxNodes            int
	MinNodesPerClass    int
	MaxNodesPerClass    int
	MaxRelationsPerNode int
	MaxRelationDistance float64
}

func BuildNetwork(cells []Cell, cfg NetworkConfig) Network {
	if cfg.Width <= 0 || cfg.Height <= 0 ||
		len(cells) != cfg.Width*cfg.Height {
		return Network{}
	}
	if cfg.MinNodeSuitability <= 0 {
		cfg.MinNodeSuitability = 0.60
	}
	if cfg.LocalMaximumRadius <= 0 {
		cfg.LocalMaximumRadius = 2
	}
	if cfg.MinSeparationBlocks <= 0 {
		cfg.MinSeparationBlocks = 640
	}
	if cfg.MaxNodes <= 0 {
		cfg.MaxNodes = 80
	}
	if cfg.MinNodesPerClass <= 0 {
		cfg.MinNodesPerClass = 4
	}
	if cfg.MaxNodesPerClass <= 0 {
		cfg.MaxNodesPerClass = maxInt(8, cfg.MaxNodes/3)
	}
	if cfg.MaxRelationsPerNode <= 0 {
		cfg.MaxRelationsPerNode = 3
	}
	if cfg.MaxRelationDistance <= 0 {
		cfg.MaxRelationDistance = 6400
	}

	candidates := make([]Cell, 0, 128)
	for i, cell := range cells {
		if !cell.Land || cell.Score.NodeSuitability < cfg.MinNodeSuitability {
			continue
		}
		if localMaximum(cells, cfg, i) {
			candidates = append(candidates, cell)
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score.NodeSuitability == candidates[j].Score.NodeSuitability {
			if candidates[i].Z == candidates[j].Z {
				return candidates[i].X < candidates[j].X
			}
			return candidates[i].Z < candidates[j].Z
		}
		return candidates[i].Score.NodeSuitability > candidates[j].Score.NodeSuitability
	})

	nodes := make([]Node, 0, minInt(cfg.MaxNodes, len(candidates)))
	classCounts := make(map[string]int)
	addCandidate := func(candidate Cell, enforceClassCap bool) bool {
		if len(nodes) >= cfg.MaxNodes {
			return false
		}
		class := candidate.Score.NodeClass
		if enforceClassCap && classCounts[class] >= cfg.MaxNodesPerClass {
			return false
		}
		for _, node := range nodes {
			if math.Hypot(candidate.X-node.X, candidate.Z-node.Z) <
				cfg.MinSeparationBlocks {
				return false
			}
		}
		nodes = append(nodes, Node{
			ID: len(nodes) + 1,
			X:  candidate.X, Z: candidate.Z,
			GridX: candidate.GridX, GridZ: candidate.GridZ,
			Class: class,
			Score: candidate.Score,
		})
		classCounts[class]++
		return true
	}

	roleOrder := [...]string{
		"port_city",
		"mining_town",
		"transport_node",
		"agricultural_settlement",
		"multi_role_hub",
	}
	for _, class := range roleOrder {
		for _, candidate := range candidates {
			if classCounts[class] >= cfg.MinNodesPerClass ||
				len(nodes) >= cfg.MaxNodes {
				break
			}
			if candidate.Score.NodeClass != class {
				continue
			}
			addCandidate(candidate, false)
		}
	}

	for _, candidate := range candidates {
		if len(nodes) >= cfg.MaxNodes {
			break
		}
		addCandidate(candidate, true)
	}

	relations := buildRelations(nodes, cells, cfg)
	return Network{Nodes: nodes, Relations: relations}
}

func localMaximum(cells []Cell, cfg NetworkConfig, index int) bool {
	cell := cells[index]
	class := cell.Score.NodeClass
	score := roleScore(cell.Score, class)
	r := cfg.LocalMaximumRadius
	for dz := -r; dz <= r; dz++ {
		z := cell.GridZ + dz
		if z < 0 || z >= cfg.Height {
			continue
		}
		for dx := -r; dx <= r; dx++ {
			x := cell.GridX + dx
			if x < 0 || x >= cfg.Width || (dx == 0 && dz == 0) {
				continue
			}
			other := cells[z*cfg.Width+x]
			if other.Score.NodeClass != class {
				continue
			}
			otherScore := roleScore(other.Score, class)
			if otherScore > score+1e-12 {
				return false
			}
			if math.Abs(otherScore-score) <= 1e-12 &&
				(z*cfg.Width+x) < index {
				return false
			}
		}
	}
	return true
}

func roleScore(score Score, class string) float64 {
	switch class {
	case "agricultural_settlement":
		return score.AgriculturalSettlementSuitability
	case "port_city":
		return score.PortCitySuitability
	case "mining_town":
		return score.MiningTownSuitability
	case "transport_node":
		return score.TransportNodeSuitability
	default:
		return score.NodeSuitability
	}
}

func buildRelations(nodes []Node, cells []Cell, cfg NetworkConfig) []Relation {
	type edge struct {
		a, b int
		d    float64
	}
	all := make([]edge, 0)
	for i := 0; i < len(nodes); i++ {
		for j := i + 1; j < len(nodes); j++ {
			d := math.Hypot(nodes[i].X-nodes[j].X, nodes[i].Z-nodes[j].Z)
			if d <= cfg.MaxRelationDistance {
				all = append(all, edge{i, j, d})
			}
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].d == all[j].d {
			if all[i].a == all[j].a {
				return all[i].b < all[j].b
			}
			return all[i].a < all[j].a
		}
		return all[i].d < all[j].d
	})

	degree := make([]int, len(nodes))
	out := make([]Relation, 0, len(nodes)*2)
	for _, e := range all {
		if degree[e.a] >= cfg.MaxRelationsPerNode ||
			degree[e.b] >= cfg.MaxRelationsPerNode {
			continue
		}
		suitability, mode := corridorScore(nodes[e.a], nodes[e.b], cells, cfg)
		if suitability < 0.30 {
			continue
		}
		degree[e.a]++
		degree[e.b]++
		out = append(out, Relation{
			From:                nodes[e.a].ID,
			To:                  nodes[e.b].ID,
			DistanceBlocks:      e.d,
			CorridorSuitability: suitability,
			Mode:                mode,
		})
	}
	return out
}

func corridorScore(a, b Node, cells []Cell, cfg NetworkConfig) (float64, string) {
	const samples = 16
	landSum := 0.0
	waterSum := 0.0
	waterSamples := 0
	for i := 0; i <= samples; i++ {
		t := float64(i) / samples
		gx := int(math.Round(float64(a.GridX) + float64(b.GridX-a.GridX)*t))
		gz := int(math.Round(float64(a.GridZ) + float64(b.GridZ-a.GridZ)*t))
		gx = clampInt(gx, 0, cfg.Width-1)
		gz = clampInt(gz, 0, cfg.Height-1)
		cell := cells[gz*cfg.Width+gx]
		if cell.Land {
			landSum += 0.42*clamp01(cell.TransportAccess) +
				0.28*clamp01(cell.Buildability) +
				0.30*clamp01(cell.NaturalCorridorSuitability)
		} else {
			waterSamples++
			waterSum += clamp01(cell.WaterwayNavigability) *
				(1 - 0.55*clamp01(cell.NavigationBottleneckRisk))
		}
	}
	n := float64(samples + 1)
	landFraction := float64(samples+1-waterSamples) / n
	waterFraction := float64(waterSamples) / n
	landQuality := landSum / n
	waterQuality := waterSum / n

	endpoint := 0.5 * (a.Score.NodeSuitability + b.Score.NodeSuitability)
	mode := "mixed"
	pathQuality := 0.65*landQuality + 0.35*waterQuality
	if landFraction >= 0.78 {
		mode = "land"
		pathQuality = landQuality / math.Max(landFraction, 1e-9)
	} else if waterFraction >= 0.45 && waterQuality > 0.25 {
		mode = "water"
		pathQuality = waterQuality / math.Max(waterFraction, 1e-9)
	}
	distanceFactor := 1 - 0.28*clamp01(
		math.Hypot(a.X-b.X, a.Z-b.Z)/cfg.MaxRelationDistance,
	)
	return clamp01(
		(0.68*pathQuality + 0.32*endpoint) * distanceFactor,
	), mode
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
