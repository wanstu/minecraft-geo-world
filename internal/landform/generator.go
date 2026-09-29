package landform

import (
	"container/heap"
	"errors"
	"math"
)

type distanceItem struct {
	index    int
	distance float64
}
type distanceHeap []distanceItem

func (h distanceHeap) Len() int { return len(h) }
func (h distanceHeap) Less(i, j int) bool {
	if h[i].distance == h[j].distance {
		return h[i].index < h[j].index
	}
	return h[i].distance < h[j].distance
}
func (h distanceHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *distanceHeap) Push(x any)   { *h = append(*h, x.(distanceItem)) }
func (h *distanceHeap) Pop() any {
	old := *h
	n := len(old)
	v := old[n-1]
	*h = old[:n-1]
	return v
}

func Generate(input Input, cfg Config) (*Model, error) {
	if input.Width < 2 || input.Height < 2 {
		return nil, errors.New("landform: grid must be at least 2x2")
	}
	n := input.Width * input.Height
	for _, field := range []struct {
		name string
		data []float64
	}{
		{"elevation", input.Elevation},
		{"relief", input.ReliefIndex},
		{"slope", input.Slope},
		{"terrain maturity", input.TerrainMaturity},
		{"tectonic support", input.TectonicSupport},
		{"crust thickness", input.CrustThickness},
		{"valley incision", input.ValleyIncision},
		{"fluvial power", input.FluvialPower},
	} {
		if len(field.data) != n {
			return nil, errors.New("landform: " + field.name + " size does not match dimensions")
		}
	}
	if len(input.LandMask) != n {
		return nil, errors.New("landform: land mask size does not match dimensions")
	}

	cfg = normalizeConfig(cfg, input.Width, input.Height)
	m := &Model{
		Config:             cfg,
		Width:              input.Width,
		Height:             input.Height,
		LandMask:           append([]bool(nil), input.LandMask...),
		MountainCore:       make([]float64, n),
		MountainDistance:   make([]float64, n),
		MountainInfluence:  make([]float64, n),
		RollingPlainWeight: make([]float64, n),
		HillWeight:         make([]float64, n),
		FoothillWeight:     make([]float64, n),
		MountainWeight:     make([]float64, n),
		PlateauWeight:      make([]float64, n),
		ValleyCore:         make([]float64, n),
		ValleyWeight:       make([]float64, n),
	}

	m.computeMountainContext(input)
	m.computeWeights(input)
	m.spreadValleys()
	return m, nil
}

func normalizeConfig(cfg Config, width, height int) Config {
	if cfg.SeaLevelNormalized <= 0 || cfg.SeaLevelNormalized >= 1 {
		cfg.SeaLevelNormalized = 0.5
	}
	minDim := width
	if height < minDim {
		minDim = height
	}
	if cfg.MountainInfluenceCells <= 0 {
		cfg.MountainInfluenceCells = math.Max(2, float64(minDim)*0.025)
	}
	if cfg.ValleyRadiusCells <= 0 {
		cfg.ValleyRadiusCells = int(math.Round(float64(minDim) * 0.01))
		if cfg.ValleyRadiusCells < 1 {
			cfg.ValleyRadiusCells = 1
		}
	}
	if cfg.ValleyDecayCells <= 0 {
		cfg.ValleyDecayCells = math.Max(0.75, float64(minDim)*0.0055)
	}
	return cfg
}

func (m *Model) computeMountainContext(input Input) {
	n := m.Width * m.Height
	const inf = 1e30

	h := &distanceHeap{}
	heap.Init(h)

	for i := 0; i < n; i++ {
		m.MountainDistance[i] = inf
		if !m.LandMask[i] {
			continue
		}

		elevation := normalizedLandElevation(input.Elevation[i], m.Config.SeaLevelNormalized)
		relief := clamp(input.ReliefIndex[i], 0, 1)
		support := clamp(input.TectonicSupport[i], 0, 1)

		core := smoothstep(0.48, 0.82, relief) *
			(0.55 + 0.45*smoothstep(0.16, 0.55, elevation)) *
			(0.72 + 0.28*support)
		m.MountainCore[i] = clamp(core, 0, 1)

		if m.MountainCore[i] >= 0.52 {
			m.MountainDistance[i] = 0
			heap.Push(h, distanceItem{index: i, distance: 0})
		}
	}

	dirs := []struct {
		dx, dz int
		cost   float64
	}{
		{1, 0, 1}, {-1, 0, 1}, {0, 1, 1}, {0, -1, 1},
		{1, 1, math.Sqrt2}, {1, -1, math.Sqrt2}, {-1, 1, math.Sqrt2}, {-1, -1, math.Sqrt2},
	}

	for h.Len() > 0 {
		item := heap.Pop(h).(distanceItem)
		if item.distance > m.MountainDistance[item.index]+1e-12 {
			continue
		}
		x, z := item.index%m.Width, item.index/m.Width

		for _, d := range dirs {
			nx, nz := x+d.dx, z+d.dz
			if nx < 0 || nx >= m.Width || nz < 0 || nz >= m.Height {
				continue
			}
			ni := nz*m.Width + nx
			if !m.LandMask[ni] {
				continue
			}
			next := item.distance + d.cost
			if next < m.MountainDistance[ni] {
				m.MountainDistance[ni] = next
				heap.Push(h, distanceItem{index: ni, distance: next})
			}
		}
	}

	for i := range m.MountainInfluence {
		if !m.LandMask[i] || m.MountainDistance[i] >= inf/2 {
			continue
		}
		m.MountainInfluence[i] = math.Exp(-m.MountainDistance[i] / m.Config.MountainInfluenceCells)
	}
}

func (m *Model) computeWeights(input Input) {
	for i := range m.LandMask {
		if !m.LandMask[i] {
			continue
		}

		elevation := normalizedLandElevation(input.Elevation[i], m.Config.SeaLevelNormalized)
		relief := clamp(input.ReliefIndex[i], 0, 1)
		slope := clamp(input.Slope[i]/0.06, 0, 1)
		maturity := clamp(input.TerrainMaturity[i], 0, 1)
		support := clamp(input.TectonicSupport[i], 0, 1)
		crust := clamp(input.CrustThickness[i], 0, 1)
		mountainInfluence := clamp(m.MountainInfluence[i], 0, 1)

		lowRelief := 1 - smoothstep(0.18, 0.43, relief)
		midRelief := smoothBand(relief, 0.16, 0.30, 0.58, 0.76)
		highRelief := smoothstep(0.48, 0.78, relief)

		thickCrust := smoothstep(0.62, 0.80, crust)
		plateauSignal :=
			smoothstep(0.22, 0.43, elevation) *
				(1 - smoothstep(0.30, 0.55, relief)) *
				(0.18 + 0.82*thickCrust)

		plain := lowRelief *
			(0.62 + 0.38*maturity) *
			(1 - 0.82*smoothstep(0.20, 0.50, elevation)) *
			(1 - 0.65*plateauSignal)

		plateau := 2.35 *
			plateauSignal *
			(0.80 + 0.20*maturity)

		mountain := highRelief *
			(0.45 + 0.55*smoothstep(0.18, 0.55, elevation)) *
			(0.72 + 0.28*support)

		foothill := 2.10 *
			math.Pow(mountainInfluence, 0.72) *
			(1 - 0.78*mountain) *
			smoothBand(relief, 0.14, 0.28, 0.62, 0.80) *
			(0.48 + 0.52*slope)

		hill := 1.05 *
			midRelief *
			(0.52 + 0.48*maturity) *
			(1 - 0.68*mountainInfluence) *
			(1 - 0.55*plateau)

		sum := plain + hill + foothill + mountain + plateau
		if sum <= 1e-12 {
			plain = 1
			sum = 1
		}

		m.RollingPlainWeight[i] = plain / sum
		m.HillWeight[i] = hill / sum
		m.FoothillWeight[i] = foothill / sum
		m.MountainWeight[i] = mountain / sum
		m.PlateauWeight[i] = plateau / sum

		incision := smoothstep(0.00025, 0.010, input.ValleyIncision[i])
		fluvial := smoothstep(0.025, 0.55, input.FluvialPower[i])
		m.ValleyCore[i] = clamp(0.65*incision+0.35*fluvial, 0, 1)
	}
}

func (m *Model) spreadValleys() {
	radius := m.Config.ValleyRadiusCells
	decay := m.Config.ValleyDecayCells

	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			if !m.LandMask[i] {
				continue
			}

			best := m.ValleyCore[i]
			for dz := -radius; dz <= radius; dz++ {
				nz := z + dz
				if nz < 0 || nz >= m.Height {
					continue
				}
				for dx := -radius; dx <= radius; dx++ {
					nx := x + dx
					if nx < 0 || nx >= m.Width {
						continue
					}
					ni := nz*m.Width + nx
					core := m.ValleyCore[ni]
					if core <= 0 {
						continue
					}
					distance := math.Hypot(float64(dx), float64(dz))
					if distance > float64(radius) {
						continue
					}
					influence := core * math.Exp(-distance/decay)
					if influence > best {
						best = influence
					}
				}
			}
			m.ValleyWeight[i] = clamp(best, 0, 1)
		}
	}
}

func normalizedLandElevation(elevation, sea float64) float64 {
	if elevation <= sea {
		return 0
	}
	return clamp((elevation-sea)/(1-sea), 0, 1)
}

func smoothBand(x, rise0, rise1, fall0, fall1 float64) float64 {
	return smoothstep(rise0, rise1, x) * (1 - smoothstep(fall0, fall1, x))
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge1 <= edge0 {
		return 0
	}
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
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
