// geoworld-river-navigation-audit measures Minecraft integer-water-level
// stair frequency on low-gradient, boat-sized rivers. Steep reaches are
// reported separately as rapids rather than being forced into flat canals.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"

	"geoworld/internal/generator"
	"geoworld/internal/reconcile"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
	"geoworld/internal/worldsample"
)

type bucket struct {
	Name        string  `json:"name"`
	Segments    int     `json:"segments"`
	Length      float64 `json:"lengthBlocks"`
	Changes     int     `json:"waterYChanges"`
	ChangesPerK float64 `json:"changesPerKm"`
	Rises       int     `json:"downstreamRises"`
	DrySamples  int     `json:"dryCenterlineSamples"`
	MaxDrop     int     `json:"maxSingleDropBlocks"`
}

type report struct {
	Seed                        int64    `json:"seed"`
	SampleStep                  float64  `json:"sampleStepBlocks"`
	LowGradient                 []bucket `json:"lowGradient"`
	Rapids                      []bucket `json:"rapids"`
	TotalRises                  int      `json:"totalDownstreamRises"`
	TotalDry                    int      `json:"totalDryCenterlineSamples"`
	MaxAdjacentWaterStep        int      `json:"maxAdjacentWaterStepBlocks"`
	AdjacentStepViolations      int      `json:"adjacentWaterStepViolations"`
	MinCenterlineWaterDepth     int      `json:"minCenterlineWaterDepthBlocks"`
	OneBlockDepthSamples        int      `json:"oneBlockCenterlineSamples"`
	LakeFeeders                 int      `json:"lakeFeeders"`
	DisconnectedLakeFeeders     int      `json:"disconnectedLakeFeeders"`
	LakeMouths                  int      `json:"lakeMouths"`
	LakeOutlets                 int      `json:"lakeOutlets"`
	NarrowLakeConnections       int      `json:"narrowLakeConnections"`
	MinLakeConnectionWidthRatio float64  `json:"minLakeConnectionWidthRatio"`
	StrictPassed                bool     `json:"strictPassed"`
}

func main() {
	seed := flag.Int64("seed", 123456, "world seed")
	width := flag.Int("width", 320, "macro grid width")
	height := flag.Int("height", 200, "macro grid height")
	cell := flag.Float64("cell-size", 128, "macro cell size in blocks")
	step := flag.Float64("sample-step", 4, "centerline sample spacing in blocks")
	strict := flag.Bool("strict", false, "exit non-zero when navigable-river limits fail")
	flag.Parse()

	g := world.GridGeometry{
		Width:       *width,
		Height:      *height,
		OriginX:     -float64(*width-1) * *cell / 2,
		OriginZ:     -float64(*height-1) * *cell / 2,
		SpanXBlocks: float64(*width-1) * *cell,
		SpanZBlocks: float64(*height-1) * *cell,
	}
	m, err := generator.GenerateWorld(context.Background(), generator.Config{
		Seed:                   *seed,
		Geometry:               g,
		PlateCount:             18,
		OceanFraction:          0.68,
		ContinentFragmentation: 0.35,
		ContinentSeparation:    0.65,
	}, nil)
	if err != nil {
		panic(err)
	}

	low := map[string]*bucket{
		"small":  {Name: "small 8-12"},
		"medium": {Name: "medium 12-16"},
		"large":  {Name: "large >=16"},
	}
	rapid := map[string]*bucket{
		"small":  {Name: "small 8-12"},
		"medium": {Name: "medium 12-16"},
		"large":  {Name: "large >=16"},
	}

	totalRises, totalDry := 0, 0
	minCenterlineDepth, oneBlockDepthSamples := 0, 0
	for _, seg := range m.Waterbed.Rivers {
		if seg.HalfWidth < 8 {
			continue
		}
		key := widthBucket(seg.HalfWidth)
		depth := riverCenterDepth(m.Waterbed.Config, seg.HalfWidth)
		startY := minecraftWaterY(seg.Profile[0] + depth)
		endY := minecraftWaterY(seg.Profile[len(seg.Profile)-1] + depth)
		isLow := startY-endY <= 3
		target := rapid[key]
		if isLow {
			target = low[key]
		}
		target.Segments++

		length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
		target.Length += length
		steps := maxInt(1, int(math.Ceil(length/math.Max(1, *step))))
		prevSet := false
		prevY := 0
		for j := 0; j <= steps; j++ {
			t := float64(j) / float64(steps)
			x, z := riverCenter(seg, t)
			s := m.Runtime.Sample(x, z)
			if !s.WaterSurfaceDefined {
				target.DrySamples++
				totalDry++
				prevSet = false
				continue
			}
			// Lake/ocean at the last samples is valid navigable receiving water.
			if s.WaterKind != reconcile.WaterRiver &&
				s.WaterKind != reconcile.WaterLake &&
				s.WaterKind != reconcile.WaterOcean {
				target.DrySamples++
				totalDry++
				prevSet = false
				continue
			}
			y := minecraftWaterY(s.WaterSurfaceElevation)
			if s.WaterKind == reconcile.WaterRiver {
				groundY := minecraftGroundY(s.GroundHeight)
				depthBlocks := y - groundY
				// Mirror the Minecraft shoreline fallback: a planned river column
				// is never shallower than one block even when integer rounding
				// would otherwise collapse water and ground to the same boundary.
				if depthBlocks < 1 {
					depthBlocks = 1
				}
				if minCenterlineDepth == 0 || depthBlocks < minCenterlineDepth {
					minCenterlineDepth = depthBlocks
				}
				if depthBlocks == 1 {
					oneBlockDepthSamples++
				}
			}
			if prevSet && y != prevY {
				target.Changes++
				d := prevY - y
				if d < 0 {
					target.Rises++
					totalRises++
				} else if d > target.MaxDrop {
					target.MaxDrop = d
				}
			}
			prevY = y
			prevSet = true
		}
	}

	maxAdjacentStep, adjacentStepViolations := auditAdjacentWaterSteps(m)
	lakeFeeders, disconnectedLakeFeeders := auditLakeFeeders(m)
	lakeMouths, lakeOutlets, narrowLakeConnections, minLakeConnectionWidthRatio :=
		auditLakeConnectionBreadth(m)

	lowOrder := []bucket{*low["small"], *low["medium"], *low["large"]}
	rapidOrder := []bucket{*rapid["small"], *rapid["medium"], *rapid["large"]}
	for i := range lowOrder {
		if lowOrder[i].Length > 0 {
			lowOrder[i].ChangesPerK = 1000 * float64(lowOrder[i].Changes) / lowOrder[i].Length
		}
	}
	for i := range rapidOrder {
		if rapidOrder[i].Length > 0 {
			rapidOrder[i].ChangesPerK = 1000 * float64(rapidOrder[i].Changes) / rapidOrder[i].Length
		}
	}

	pass := totalRises == 0 &&
		adjacentStepViolations == 0 &&
		oneBlockDepthSamples == 0 &&
		disconnectedLakeFeeders == 0 &&
		narrowLakeConnections == 0 &&
		lowOrder[0].ChangesPerK <= 5.0 &&
		lowOrder[1].ChangesPerK <= 3.5 &&
		lowOrder[2].ChangesPerK <= 2.0
	r := report{
		Seed:                        *seed,
		SampleStep:                  *step,
		LowGradient:                 lowOrder,
		Rapids:                      rapidOrder,
		TotalRises:                  totalRises,
		TotalDry:                    totalDry,
		MaxAdjacentWaterStep:        maxAdjacentStep,
		AdjacentStepViolations:      adjacentStepViolations,
		MinCenterlineWaterDepth:     minCenterlineDepth,
		OneBlockDepthSamples:        oneBlockDepthSamples,
		LakeFeeders:                 lakeFeeders,
		DisconnectedLakeFeeders:     disconnectedLakeFeeders,
		LakeMouths:                  lakeMouths,
		LakeOutlets:                 lakeOutlets,
		NarrowLakeConnections:       narrowLakeConnections,
		MinLakeConnectionWidthRatio: minLakeConnectionWidthRatio,
		StrictPassed:                pass,
	}
	data, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(data))
	if *strict && !pass {
		os.Exit(1)
	}
}

func auditAdjacentWaterSteps(m *generator.WorldModel) (int, int) {
	// Minecraft only materializes integer X/Z columns. Build a dense set of
	// integer centerline cells, then compare only face-adjacent block columns.
	// Continuous half-block samples can cross two quantization thresholds at
	// once and report a fake two-block "wall" that no MC column pair contains.
	cells := make(map[[2]int]struct{})
	for _, seg := range m.Waterbed.Rivers {
		chord := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
		pathUpperBound := chord + math.Pi*math.Abs(seg.MeanderOffset)
		steps := maxInt(1, int(math.Ceil(pathUpperBound/0.5)))
		for j := 0; j <= steps; j++ {
			t := float64(j) / float64(steps)
			x, z := riverCenter(seg, t)
			ix, iz := int(math.Round(x)), int(math.Round(z))
			cells[[2]int{ix, iz}] = struct{}{}
			// Include a one-block collar so diagonal rasterization and junction
			// ownership are checked on the same face-adjacent columns MC uses.
			cells[[2]int{ix + 1, iz}] = struct{}{}
			cells[[2]int{ix - 1, iz}] = struct{}{}
			cells[[2]int{ix, iz + 1}] = struct{}{}
			cells[[2]int{ix, iz - 1}] = struct{}{}
		}
	}

	maxStep := 0
	violations := 0
	for cell := range cells {
		current := m.Runtime.Sample(float64(cell[0]), float64(cell[1]))
		if !runtimeWater(current) {
			continue
		}
		currentY := minecraftWaterY(current.WaterSurfaceElevation)
		for _, direction := range [][2]int{{1, 0}, {0, 1}} {
			nx := cell[0] + direction[0]
			nz := cell[1] + direction[1]
			neighbor := m.Runtime.Sample(float64(nx), float64(nz))
			if !runtimeWater(neighbor) {
				continue
			}
			neighborY := minecraftWaterY(neighbor.WaterSurfaceElevation)
			d := currentY - neighborY
			if d < 0 {
				d = -d
			}
			if d > maxStep {
				maxStep = d
			}
			if d > 1 {
				violations++
			}
		}
	}
	return maxStep, violations
}

func runtimeWater(s worldsample.Sample) bool {
	return s.WaterSurfaceDefined &&
		(s.WaterKind == reconcile.WaterRiver ||
			s.WaterKind == reconcile.WaterLake ||
			s.WaterKind == reconcile.WaterOcean)
}

func auditLakeFeeders(m *generator.WorldModel) (int, int) {
	if m == nil || m.Hydrology == nil ||
		m.Reconciliation == nil || m.Waterbed == nil {
		return 0, 0
	}
	pair := make(map[[2]int]int, len(m.Waterbed.Rivers))
	outgoing := make(map[int][]int)
	for id, seg := range m.Waterbed.Rivers {
		pair[[2]int{seg.From, seg.To}] = id
		outgoing[seg.From] = append(outgoing[seg.From], id)
	}

	feeders := 0
	disconnected := 0
	for _, edge := range m.Reconciliation.RiverEdges {
		if edge.From < 0 || edge.From >= len(m.Hydrology.LakeID) ||
			edge.To < 0 || edge.To >= len(m.Hydrology.LakeID) {
			continue
		}
		lakeID := m.Hydrology.LakeID[edge.To]
		if lakeID <= 0 || m.Hydrology.LakeID[edge.From] == lakeID {
			continue
		}
		feeders++
		start, ok := pair[[2]int{edge.From, edge.To}]
		if !ok || !riverGraphTouchesLake(m, start, lakeID, outgoing) {
			disconnected++
		}
	}
	return feeders, disconnected
}

func auditLakeConnectionBreadth(
	m *generator.WorldModel,
) (mouths, outlets, narrow int, minRatio float64) {
	if m == nil || m.Waterbed == nil || m.Runtime == nil {
		return 0, 0, 0, 0
	}

	const minimumTransitionRatio = 1.70
	first := true
	for _, seg := range m.Waterbed.Rivers {
		transitionT := -1.0
		if seg.LakeMouth {
			mouths++
			transitionT = 0.85
		}
		if seg.LakeOutlet {
			outlets++
			transitionT = 0.15
		}
		if transitionT < 0 || seg.HalfWidth <= 0 {
			continue
		}

		visibleWidth := visibleWaterCrossSectionWidth(
			m,
			seg,
			transitionT,
		)
		ratio := float64(visibleWidth) / (2 * seg.HalfWidth)
		if first || ratio < minRatio {
			minRatio = ratio
			first = false
		}
		if ratio < minimumTransitionRatio {
			narrow++
		}
	}
	return mouths, outlets, narrow, minRatio
}

func visibleWaterCrossSectionWidth(
	m *generator.WorldModel,
	seg waterbed.RiverSegment,
	t float64,
) int {
	centerX, centerZ, normalX, normalZ :=
		waterbed.RiverCenterFrame(seg, t)
	if !runtimeWater(m.Runtime.Sample(centerX, centerZ)) {
		return 0
	}

	// v1.1.7 river half-width tops out far below this scan radius. A bounded
	// scan keeps the audit cheap while still allowing a receiving lake to make
	// the measured connection much wider than its incoming river.
	const scanRadius = 192
	left := 0
	right := 0
	for distance := 1; distance <= scanRadius; distance++ {
		d := float64(distance)
		if !runtimeWater(
			m.Runtime.Sample(
				centerX-normalX*d,
				centerZ-normalZ*d,
			),
		) {
			break
		}
		left = distance
	}
	for distance := 1; distance <= scanRadius; distance++ {
		d := float64(distance)
		if !runtimeWater(
			m.Runtime.Sample(
				centerX+normalX*d,
				centerZ+normalZ*d,
			),
		) {
			break
		}
		right = distance
	}
	return left + right + 1
}

func riverGraphTouchesLake(
	m *generator.WorldModel,
	start int,
	lakeID int,
	outgoing map[int][]int,
) bool {
	queue := []int{start}
	visited := make(map[int]bool)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if id < 0 || id >= len(m.Waterbed.Rivers) || visited[id] {
			continue
		}
		visited[id] = true
		seg := m.Waterbed.Rivers[id]
		length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
		steps := maxInt(1, int(math.Ceil(length/4)))
		for j := 0; j <= steps; j++ {
			t := float64(j) / float64(steps)
			x, z := riverCenter(seg, t)
			s := m.Runtime.Sample(x, z)
			if s.WaterSurfaceDefined &&
				s.WaterKind != reconcile.WaterNone &&
				sampleBelongsToLake(m, x, z, lakeID) {
				// A river can remain the categorical owner while crossing a
				// hydrologic lake cell (especially at a narrow inlet/outlet).
				// Connectivity only asks whether continuous water entered the
				// target LakeID; visible lake breadth is audited separately.
				return true
			}
		}
		nextIDs := outgoing[seg.To]
		if len(nextIDs) == 0 &&
			seg.To >= 0 &&
			seg.To < len(m.Hydrology.LakeID) &&
			m.Hydrology.LakeID[seg.To] == lakeID {
			// Connectivity is about continuous navigable water, not the final
			// categorical owner of an overlap. A tiny/shallow hydrologic lake
			// can be absorbed by a wide river, while a coastal lake can hand
			// off to Ocean at the final connector node. Both are connected if
			// the formal feeder reaches the target basin without a dry gap.
			end := m.Runtime.Sample(seg.X1, seg.Z1)
			if end.WaterSurfaceDefined &&
				end.WaterKind != reconcile.WaterNone {
				return true
			}
		}
		for _, next := range nextIDs {
			if !visited[next] {
				queue = append(queue, next)
			}
		}
	}
	return false
}

func sampleBelongsToLake(
	m *generator.WorldModel,
	x,
	z float64,
	lakeID int,
) bool {
	g := m.Waterbed.Geometry
	spacingX := g.SpacingX()
	spacingZ := g.SpacingZ()
	if spacingX <= 0 || spacingZ <= 0 {
		return false
	}
	gx := int(math.Round((x - g.OriginX) / spacingX))
	gz := int(math.Round((z - g.OriginZ) / spacingZ))
	if gx < 0 || gx >= g.Width || gz < 0 || gz >= g.Height {
		return false
	}
	index := gz*g.Width + gx
	return index >= 0 &&
		index < len(m.Hydrology.LakeID) &&
		m.Hydrology.LakeID[index] == lakeID
}

func widthBucket(halfWidth float64) string {
	if halfWidth >= 16 {
		return "large"
	}
	if halfWidth >= 12 {
		return "medium"
	}
	return "small"
}

func riverCenter(seg waterbed.RiverSegment, t float64) (float64, float64) {
	return waterbed.RiverCenterPoint(seg, t)
}

func riverCenterDepth(cfg waterbed.Config, halfWidth float64) float64 {
	span := math.Max(1e-9, cfg.MaxRiverHalfWidthBlocks-cfg.MinRiverHalfWidthBlocks)
	t := clamp((halfWidth-cfg.MinRiverHalfWidthBlocks)/span, 0, 1)
	return cfg.RiverCenterDepthNormalized * (1 + 2*math.Pow(t, 1.15))
}

func minecraftWaterY(h float64) int {
	h = clamp(h, 0, 1)
	var mapped float64
	if h <= 0.5 {
		mapped = -64 + 127*(h/0.5)
	} else {
		mapped = 63 + 257*((h-0.5)/0.5)
	}
	return int(math.Ceil(mapped - 1e-9))
}

func minecraftGroundY(h float64) int {
	h = clamp(h, 0, 1)
	var mapped float64
	if h <= 0.5 {
		mapped = -64 + 127*(h/0.5)
	} else {
		mapped = 63 + 257*((h-0.5)/0.5)
	}
	return int(math.Round(mapped))
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
