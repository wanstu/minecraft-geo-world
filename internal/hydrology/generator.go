package hydrology

import (
	"container/heap"
	"errors"
	"math"
	"sort"
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

	cfg = normalizeConfig(cfg)
	m := &Model{
		Config:            cfg,
		Width:             input.Width,
		Height:            input.Height,
		LandMask:          append([]bool(nil), input.LandMask...),
		OriginalElevation: append([]float64(nil), input.Elevation...),
		FilledElevation:   make([]float64, n),
		FillDepth:         make([]float64, n),
		FlowTo:            make([]int, n),
		LocalRunoffMM:     make([]float64, n),
		FlowAccumulation:  make([]float64, n),
		BasinID:           make([]int, n),
		RiverStrength:     make([]float64, n),
		LakePotential:     make([]float64, n),
		WetlandPotential:  make([]float64, n),
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

	return m, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.SeaLevelNormalized <= 0 || cfg.SeaLevelNormalized >= 1 {
		cfg.SeaLevelNormalized = 0.5
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
		cfg.RiverQuantile = 0.965
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
			minDrainLevel := item.level + m.Config.FillEpsilon
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
				slope := drop / d.dist
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
	for i := range m.LocalRunoffMM {
		if !m.LandMask[i] {
			continue
		}
		effective := math.Max(0, precip[i]-m.Config.RunoffLossMM)
		coefficient := m.Config.BaseRunoffCoefficient +
			m.Config.SeasonalityRunoffBoost*clamp(seasonality[i], 0, 1)
		coefficient = clamp(coefficient, 0.05, 0.95)
		m.LocalRunoffMM[i] = effective * coefficient
		m.FlowAccumulation[i] = m.LocalRunoffMM[i]
	}
}

func (m *Model) computeAccumulation() {
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

	for _, i := range order {
		to := m.FlowTo[i]
		if to >= 0 && to < len(m.FlowAccumulation) && m.LandMask[to] {
			m.FlowAccumulation[to] += m.FlowAccumulation[i]
		}
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
	return math.Max(0, drop/dist)
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
