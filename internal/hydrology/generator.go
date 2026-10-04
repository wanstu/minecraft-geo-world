package hydrology

import (
	"container/heap"
	"errors"
	"math"
	"sort"

	"geoworld/internal/world"
)

type floodItem struct {
	index int
	level float64
}

type minHeap []floodItem

func (h minHeap) Len() int { return len(h) }
func (h minHeap) Less(i, j int) bool {
	if h[i].level == h[j].level {
		return h[i].index < h[j].index
	}
	return h[i].level < h[j].level
}
func (h minHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)   { *h = append(*h, x.(floodItem)) }
func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	v := old[n-1]
	*h = old[:n-1]
	return v
}

func Generate(input Input, cfg Config) (*Model, error) {
	if input.Width < 2 || input.Height < 2 {
		return nil, errors.New("hydrology: grid must be at least 2x2")
	}
	n := input.Width * input.Height
	if len(input.Elevation) != n ||
		len(input.LandMask) != n ||
		len(input.AnnualPrecipMM) != n ||
		len(input.RainfallSeasonality) != n {
		return nil, errors.New("hydrology: input field sizes do not match dimensions")
	}

	warmPrecip := input.WarmSeasonPrecipMM
	coldPrecip := input.ColdSeasonPrecipMM
	if len(warmPrecip) != n || len(coldPrecip) != n {
		warmPrecip = make([]float64, n)
		coldPrecip = make([]float64, n)
		for i, annual := range input.AnnualPrecipMM {
			warmPrecip[i] = annual * 0.5
			coldPrecip[i] = annual * 0.5
		}
	}

	geometry, err := resolveInputGeometry(input)
	if err != nil {
		return nil, err
	}
	cfg = normalizeConfig(cfg)
	m := &Model{
		Config:                  cfg,
		Geometry:                geometry,
		Width:                   input.Width,
		Height:                  input.Height,
		LandMask:                append([]bool(nil), input.LandMask...),
		OriginalElevation:       append([]float64(nil), input.Elevation...),
		FilledElevation:         make([]float64, n),
		FillDepth:               make([]float64, n),
		FlowTo:                  make([]int, n),
		LocalRunoffMM:           make([]float64, n),
		FlowAccumulation:        make([]float64, n),
		WarmFlowAccumulation:    make([]float64, n),
		ColdFlowAccumulation:    make([]float64, n),
		BasinID:                 make([]int, n),
		RiverStrength:           make([]float64, n),
		RiverOrder:              make([]uint8, n),
		PerennialRiverPotential: make([]float64, n),
		SeasonalRiverPotential:  make([]float64, n),
		LakePotential:           make([]float64, n),
		WetlandPotential:        make([]float64, n),
		LakeID:                  make([]int, n),
	}
	for i := range m.FlowTo {
		m.FlowTo[i] = -1
	}

	parent := m.fillDepressions()
	m.computeFlowDirection(parent)
	m.computeRunoff(input.AnnualPrecipMM, input.RainfallSeasonality)
	m.computeAccumulation()
	m.assignBasins()
	m.classifyWaterFeatures(input.RainfallSeasonality)
	m.countMajorBasins()
	m.computeSeasonalFlow(warmPrecip, coldPrecip, input.RainfallSeasonality)
	m.computeRiverOrder()
	m.identifyLakes()

	return m, nil
}

func resolveInputGeometry(input Input) (world.GridGeometry, error) {
	geometry := input.Geometry
	if geometry.Width == 0 && geometry.Height == 0 &&
		geometry.SpanXBlocks == 0 && geometry.SpanZBlocks == 0 &&
		geometry.OriginX == 0 && geometry.OriginZ == 0 {
		geometry = world.GridGeometry{
			Width: input.Width, Height: input.Height,
			SpanXBlocks: float64(input.Width-1) * world.DefaultMacroSpacingBlocks,
			SpanZBlocks: float64(input.Height-1) * world.DefaultMacroSpacingBlocks,
		}
	}
	if geometry.Width != input.Width || geometry.Height != input.Height {
		return world.GridGeometry{}, errors.New("hydrology: geometry dimensions do not match input")
	}
	if _, err := geometry.SquareCellSize(); err != nil {
		return world.GridGeometry{}, err
	}
	return geometry, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.SeaLevelNormalized <= 0 || cfg.SeaLevelNormalized >= 1 {
		cfg.SeaLevelNormalized = world.DefaultSeaLevelNormalized
	}
	if cfg.FillEpsilon <= 0 || cfg.FillEpsilon > 0.01 {
		cfg.FillEpsilon = 1e-5
	}
	if cfg.RunoffLossMM <= 0 {
		cfg.RunoffLossMM = 180
	}
	if cfg.BaseRunoffCoefficient <= 0 || cfg.BaseRunoffCoefficient > 1 {
		cfg.BaseRunoffCoefficient = 0.48
	}
	if cfg.SeasonalityRunoffBoost < 0 || cfg.SeasonalityRunoffBoost > 1 {
		cfg.SeasonalityRunoffBoost = 0.22
	}
	if cfg.RiverQuantile <= 0.5 || cfg.RiverQuantile >= 1 {
		// Keep the large drainage skeleton selective, but admit enough
		// headwater cells to form visible tributary trees instead of isolated
		// trunk fragments.
		cfg.RiverQuantile = 0.95
	}
	if cfg.MinRiverAccumulation <= 0 {
		cfg.MinRiverAccumulation = 5000
	}
	if cfg.LakeDepthScale <= 0 {
		cfg.LakeDepthScale = 0.025
	}
	if cfg.WetlandAccumulationScale <= 0 {
		cfg.WetlandAccumulationScale = 12000
	}
	if cfg.LakePotentialThreshold <= 0 || cfg.LakePotentialThreshold >= 1 {
		cfg.LakePotentialThreshold = 0.35
	}
	if cfg.MinLakeAreaBlocks2 <= 0 {
		cfg.MinLakeAreaBlocks2 = 2 * world.DefaultMacroSpacingBlocks * world.DefaultMacroSpacingBlocks
	}
	return cfg
}

func (m *Model) fillDepressions() []int {
	n := m.Width * m.Height
	visited := make([]bool, n)
	parent := make([]int, n)
	for i := range parent {
		parent[i] = -1
	}

	h := &minHeap{}
	heap.Init(h)

	// The ocean is a fixed hydrological outlet at sea level. We seed every
	// ocean cell so inland depressions are filled only as high as necessary to
	// reach some coastline.
	for i := 0; i < n; i++ {
		if m.LandMask[i] {
			continue
		}
		visited[i] = true
		m.FilledElevation[i] = m.Config.SeaLevelNormalized
		heap.Push(h, floodItem{index: i, level: m.Config.SeaLevelNormalized})
	}

	// Fallback for synthetic/all-land tests.
	if h.Len() == 0 {
		for x := 0; x < m.Width; x++ {
			m.seedFloodCell(x, 0, visited, h)
			m.seedFloodCell(x, m.Height-1, visited, h)
		}
		for z := 1; z+1 < m.Height; z++ {
			m.seedFloodCell(0, z, visited, h)
			m.seedFloodCell(m.Width-1, z, visited, h)
		}
	}

	for h.Len() > 0 {
		item := heap.Pop(h).(floodItem)
		x := item.index % m.Width
		z := item.index / m.Width

		for _, d := range d8Directions {
			nx, nz := x+d.dx, z+d.dz
			if nx < 0 || nx >= m.Width || nz < 0 || nz >= m.Height {
				continue
			}
			ni := nz*m.Width + nx
			if visited[ni] {
				continue
			}
			visited[ni] = true
			parent[ni] = item.index

			original := m.OriginalElevation[ni]
			filled := original
			stepScale := m.Geometry.SpacingX() / world.DefaultMacroSpacingBlocks
			minDrainLevel := item.level + m.Config.FillEpsilon*stepScale
			if filled < minDrainLevel {
				filled = minDrainLevel
			}
			m.FilledElevation[ni] = filled
			if m.LandMask[ni] && filled > original {
				m.FillDepth[ni] = filled - original
			}
			heap.Push(h, floodItem{index: ni, level: filled})
		}
	}

	return parent
}

func (m *Model) seedFloodCell(x, z int, visited []bool, h *minHeap) {
	i := z*m.Width + x
	if visited[i] {
		return
	}
	visited[i] = true
	level := m.OriginalElevation[i]
	m.FilledElevation[i] = level
	heap.Push(h, floodItem{index: i, level: level})
}

type dir8 struct {
	dx   int
	dz   int
	dist float64
}

var d8Directions = []dir8{
	{1, 0, 1}, {-1, 0, 1}, {0, 1, 1}, {0, -1, 1},
	{1, 1, math.Sqrt2}, {1, -1, math.Sqrt2}, {-1, 1, math.Sqrt2}, {-1, -1, math.Sqrt2},
}

func (m *Model) computeFlowDirection(parent []int) {
	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			if !m.LandMask[i] {
				m.FlowTo[i] = -1
				continue
			}

			current := m.FilledElevation[i]
			best := -1
			bestSlope := 0.0

			for _, d := range d8Directions {
				nx, nz := x+d.dx, z+d.dz
				if nx < 0 || nx >= m.Width || nz < 0 || nz >= m.Height {
					continue
				}
				ni := nz*m.Width + nx
				level := m.FilledElevation[ni]
				if !m.LandMask[ni] {
					level = m.Config.SeaLevelNormalized
				}
				drop := current - level
				if drop <= 0 {
					continue
				}
				referenceDistance := d.dist * m.Geometry.SpacingX() / world.DefaultMacroSpacingBlocks
				slope := drop / referenceDistance
				if slope > bestSlope+1e-15 || (math.Abs(slope-bestSlope) <= 1e-15 && (best < 0 || ni < best)) {
					bestSlope = slope
					best = ni
				}
			}

			if best < 0 && i < len(parent) {
				best = parent[i]
			}
			m.FlowTo[i] = best
		}
	}
}

func (m *Model) computeRunoff(precip, seasonality []float64) {
	local := m.runoffField(precip, seasonality, m.Config.RunoffLossMM)
	copy(m.LocalRunoffMM, local)
}

func (m *Model) computeAccumulation() {
	m.FlowAccumulation = m.accumulateField(m.LocalRunoffMM)
}

func (m *Model) runoffField(precip, seasonality []float64, lossMM float64) []float64 {
	local := make([]float64, len(m.FlowTo))
	for i := range local {
		if !m.LandMask[i] {
			continue
		}
		effective := math.Max(0, precip[i]-lossMM)
		coefficient := m.Config.BaseRunoffCoefficient +
			m.Config.SeasonalityRunoffBoost*clamp(seasonality[i], 0, 1)
		coefficient = clamp(coefficient, 0.05, 0.95)
		cellScale := m.Geometry.SpacingX() / world.DefaultMacroSpacingBlocks
		local[i] = effective * coefficient * cellScale * cellScale
	}
	return local
}

func (m *Model) sortedLandOrder() []int {
	order := make([]int, 0, len(m.FlowTo))
	for i := range m.FlowTo {
		if m.LandMask[i] {
			order = append(order, i)
		}
	}
	sort.Slice(order, func(a, b int) bool {
		ia, ib := order[a], order[b]
		if m.FilledElevation[ia] == m.FilledElevation[ib] {
			return ia > ib
		}
		return m.FilledElevation[ia] > m.FilledElevation[ib]
	})
	return order
}

func (m *Model) accumulateField(local []float64) []float64 {
	acc := append([]float64(nil), local...)
	for _, i := range m.sortedLandOrder() {
		to := m.FlowTo[i]
		if to >= 0 && to < len(acc) && m.LandMask[to] {
			acc[to] += acc[i]
		}
	}
	return acc
}

func (m *Model) computeSeasonalFlow(warmPrecip, coldPrecip, seasonality []float64) {
	warmLocal := m.runoffField(warmPrecip, seasonality, m.Config.RunoffLossMM*0.5)
	coldLocal := m.runoffField(coldPrecip, seasonality, m.Config.RunoffLossMM*0.5)
	m.WarmFlowAccumulation = m.accumulateField(warmLocal)
	m.ColdFlowAccumulation = m.accumulateField(coldLocal)

	for i := range m.RiverStrength {
		if !m.LandMask[i] || m.RiverStrength[i] <= 0 {
			continue
		}
		warm := m.WarmFlowAccumulation[i]
		cold := m.ColdFlowAccumulation[i]
		high := math.Max(warm, cold)
		low := math.Min(warm, cold)
		balance := 0.0
		if high > 0 {
			balance = clamp(low/(high+1), 0, 1)
		}

		m.PerennialRiverPotential[i] = m.RiverStrength[i] * math.Sqrt(balance)
		m.SeasonalRiverPotential[i] = m.RiverStrength[i] *
			(1 - balance) *
			clamp(seasonality[i], 0, 1)
	}
}

func (m *Model) assignBasins() {
	n := len(m.FlowTo)
	outletToBasin := map[int]int{}
	nextBasin := 1

	for start := 0; start < n; start++ {
		if !m.LandMask[start] || m.BasinID[start] != 0 {
			continue
		}

		path := make([]int, 0, 32)
		cur := start
		seen := map[int]struct{}{}
		basin := 0
		outlet := -1

		for {
			if cur < 0 || cur >= n || !m.LandMask[cur] {
				if len(path) > 0 {
					outlet = path[len(path)-1]
				}
				break
			}
			if m.BasinID[cur] != 0 {
				basin = m.BasinID[cur]
				break
			}
			if _, ok := seen[cur]; ok {
				// Should never happen after priority-flood + epsilon, but keep
				// basin assignment deterministic if future algorithms regress.
				outlet = cur
				break
			}
			seen[cur] = struct{}{}
			path = append(path, cur)
			next := m.FlowTo[cur]
			if next < 0 || next >= n || !m.LandMask[next] {
				outlet = cur
				break
			}
			cur = next
		}

		if basin == 0 {
			if existing, ok := outletToBasin[outlet]; ok {
				basin = existing
			} else {
				basin = nextBasin
				nextBasin++
				outletToBasin[outlet] = basin
			}
		}
		for _, i := range path {
			m.BasinID[i] = basin
		}
	}

	m.BasinCount = nextBasin - 1
}

func (m *Model) classifyWaterFeatures(seasonality []float64) {
	values := make([]float64, 0, len(m.FlowAccumulation))
	for i, v := range m.FlowAccumulation {
		if m.LandMask[i] && v > 0 {
			values = append(values, v)
		}
	}
	sort.Float64s(values)

	threshold := m.Config.MinRiverAccumulation
	if len(values) > 0 {
		q := quantileSorted(values, m.Config.RiverQuantile)
		if q > threshold {
			threshold = q
		}
	}
	m.RiverThreshold = threshold

	for i := range m.FlowAccumulation {
		if !m.LandMask[i] {
			continue
		}

		acc := m.FlowAccumulation[i]
		if acc >= threshold && threshold > 0 {
			m.RiverStrength[i] = clamp(
				math.Log1p(acc/threshold)/math.Log(21),
				0,
				1,
			)
		}

		depthFactor := clamp(m.FillDepth[i]/m.Config.LakeDepthScale, 0, 1)
		supplyFactor := clamp(math.Log1p(acc/1000)/math.Log(101), 0, 1)
		m.LakePotential[i] = depthFactor * (0.35 + 0.65*supplyFactor)

		slope := m.downstreamSlope(i)
		flatness := math.Exp(-slope / 0.008)
		waterSupply := clamp(
			math.Log1p(acc/m.Config.WetlandAccumulationScale+1)/math.Log(8),
			0,
			1,
		)
		stability := 1 - 0.45*clamp(seasonality[i], 0, 1)
		m.WetlandPotential[i] = clamp(waterSupply*flatness*stability, 0, 1)
	}
}

func (m *Model) downstreamSlope(i int) float64 {
	to := m.FlowTo[i]
	if to < 0 || to >= len(m.FlowTo) {
		return 0
	}
	dx := math.Abs(float64((to % m.Width) - (i % m.Width)))
	dz := math.Abs(float64((to / m.Width) - (i / m.Width)))
	dist := 1.0
	if dx > 0 && dz > 0 {
		dist = math.Sqrt2
	}
	drop := m.FilledElevation[i] - m.FilledElevation[to]
	if !m.LandMask[to] {
		drop = m.FilledElevation[i] - m.Config.SeaLevelNormalized
	}
	referenceDistance := dist * m.Geometry.SpacingX() / world.DefaultMacroSpacingBlocks
	return math.Max(0, drop/referenceDistance)
}

func quantileSorted(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	q = clamp(q, 0, 1)
	pos := q * float64(len(values)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return values[lo]
	}
	t := pos - float64(lo)
	return values[lo] + (values[hi]-values[lo])*t
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

func (m *Model) countMajorBasins() {
	if m.BasinCount <= 0 || m.RiverThreshold <= 0 {
		m.MajorBasinCount = 0
		return
	}
	maxAccum := make([]float64, m.BasinCount+1)
	for i, basin := range m.BasinID {
		if basin <= 0 || basin >= len(maxAccum) {
			continue
		}
		if m.FlowAccumulation[i] > maxAccum[basin] {
			maxAccum[basin] = m.FlowAccumulation[i]
		}
	}
	count := 0
	for basin := 1; basin < len(maxAccum); basin++ {
		if maxAccum[basin] >= m.RiverThreshold {
			count++
		}
	}
	m.MajorBasinCount = count
}

func (m *Model) computeRiverOrder() {
	order := m.sortedLandOrder()
	maxUpstream := make([]uint8, len(m.FlowTo))
	maxCount := make([]uint8, len(m.FlowTo))

	for i := range m.RiverOrder {
		if m.RiverStrength[i] > 0 {
			m.RiverOrder[i] = 1
		}
	}

	for _, i := range order {
		if m.RiverStrength[i] <= 0 {
			continue
		}
		ord := m.RiverOrder[i]
		if ord == 0 {
			ord = 1
			m.RiverOrder[i] = ord
		}

		to := m.FlowTo[i]
		if to < 0 || to >= len(m.FlowTo) || m.RiverStrength[to] <= 0 {
			continue
		}

		switch {
		case ord > maxUpstream[to]:
			maxUpstream[to] = ord
			maxCount[to] = 1
		case ord == maxUpstream[to]:
			if maxCount[to] < 255 {
				maxCount[to]++
			}
		}

		nextOrder := maxUpstream[to]
		if maxCount[to] >= 2 && nextOrder < 255 {
			nextOrder++
		}
		if nextOrder > m.RiverOrder[to] {
			m.RiverOrder[to] = nextOrder
		}
	}
}

func (m *Model) identifyLakes() {
	n := len(m.LakePotential)
	candidate := make([]bool, n)
	visited := make([]bool, n)
	for i := range candidate {
		candidate[i] = m.LandMask[i] &&
			m.FillDepth[i] > 0 &&
			m.LakePotential[i] >= m.Config.LakePotentialThreshold
	}

	nextID := 1
	for start := 0; start < n; start++ {
		if !candidate[start] || visited[start] {
			continue
		}

		queue := []int{start}
		visited[start] = true
		component := make([]int, 0, 8)

		for head := 0; head < len(queue); head++ {
			i := queue[head]
			component = append(component, i)
			x, z := i%m.Width, i/m.Width
			for _, d := range d8Directions {
				nx, nz := x+d.dx, z+d.dz
				if nx < 0 || nx >= m.Width || nz < 0 || nz >= m.Height {
					continue
				}
				ni := nz*m.Width + nx
				if candidate[ni] && !visited[ni] {
					visited[ni] = true
					queue = append(queue, ni)
				}
			}
		}

		componentAreaBlocks2 := float64(len(component)) * m.Geometry.SpacingX() * m.Geometry.SpacingZ()
		if componentAreaBlocks2+1e-9 < m.Config.MinLakeAreaBlocks2 {
			continue
		}

		id := nextID
		nextID++
		surface := math.Inf(1)
		maxDepth := 0.0
		maxAccum := 0.0
		basin := 0
		for _, i := range component {
			m.LakeID[i] = id
			if m.FilledElevation[i] < surface {
				surface = m.FilledElevation[i]
			}
			if m.FillDepth[i] > maxDepth {
				maxDepth = m.FillDepth[i]
			}
			if m.FlowAccumulation[i] > maxAccum {
				maxAccum = m.FlowAccumulation[i]
				basin = m.BasinID[i]
			}
		}

		outlet := -1
		outletLevel := math.Inf(1)
		for _, i := range component {
			to := m.FlowTo[i]
			if to >= 0 && to < n && m.LakeID[to] == id {
				continue
			}
			level := m.FilledElevation[i]
			if level < outletLevel {
				outletLevel = level
				outlet = i
			}
		}

		m.Lakes = append(m.Lakes, Lake{
			ID:                 id,
			CellCount:          len(component),
			AreaBlocks2:        componentAreaBlocks2,
			SurfaceElevation:   surface,
			MaxDepth:           maxDepth,
			SpillOutlet:        outlet,
			BasinID:            basin,
			InflowAccumulation: maxAccum,
		})
	}
}
