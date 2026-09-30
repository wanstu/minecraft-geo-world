package geomorph

import (
	"errors"
	"math"
	"sort"

	"geoworld/internal/world"
)

func Generate(input Input, cfg Config) (*Model, error) {
	if input.Width < 2 || input.Height < 2 {
		return nil, errors.New("geomorph: grid must be at least 2x2")
	}
	n := input.Width * input.Height

	floatFields := []struct {
		name string
		data []float64
	}{
		{"elevation", input.Elevation},
		{"slope", input.Slope},
		{"relief", input.ReliefIndex},
		{"valley incision", input.ValleyIncision},
		{"fluvial power", input.FluvialPower},
		{"tectonic support", input.TectonicSupport},
		{"flow accumulation", input.FlowAccumulation},
		{"river strength", input.RiverStrength},
		{"perennial river", input.PerennialRiverPotential},
		{"seasonal river", input.SeasonalRiverPotential},
		{"wetland", input.WetlandPotential},
		{"rolling plain", input.RollingPlainWeight},
		{"hill", input.HillWeight},
		{"foothill", input.FoothillWeight},
		{"mountain", input.MountainWeight},
		{"plateau", input.PlateauWeight},
		{"valley", input.ValleyWeight},
	}
	for _, field := range floatFields {
		if len(field.data) != n {
			return nil, errors.New("geomorph: " + field.name + " size does not match dimensions")
		}
	}
	if len(input.LandMask) != n || len(input.FlowTo) != n || len(input.LakeID) != n {
		return nil, errors.New("geomorph: discrete field size does not match dimensions")
	}

	geometry, err := resolveInputGeometry(input)
	if err != nil {
		return nil, err
	}
	cfg = normalizeConfig(cfg)
	m := &Model{
		Config:                 cfg,
		Geometry:               geometry,
		Width:                  input.Width,
		Height:                 input.Height,
		LandMask:               append([]bool(nil), input.LandMask...),
		UpstreamSlope:          make([]float64, n),
		UpstreamSteepMemory:    make([]float64, n),
		UpstreamMountainMemory: make([]float64, n),
		SlopeDrop:              make([]float64, n),
		SedimentSupply:         make([]float64, n),
		FloodplainWeight:       make([]float64, n),
		AlluvialPlainWeight:    make([]float64, n),
		AlluvialFanCore:        make([]float64, n),
		AlluvialFanWeight:      make([]float64, n),
		CanyonWeight:           make([]float64, n),
		ValleyFloorWidth:       make([]float64, n),
		DepositionPotential:    make([]float64, n),
	}

	m.computeUpstreamContext(input)
	m.computeWeights(input)
	m.spreadAlluvialFans(input)
	m.computeDeposition()
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
		return world.GridGeometry{}, errors.New("geomorph: geometry dimensions do not match input")
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
	if cfg.AccumulationScale <= 0 {
		cfg.AccumulationScale = 8
	}
	if cfg.SlopeScale <= 0 {
		cfg.SlopeScale = 0.05
	}
	if cfg.ReliefScale <= 0 {
		cfg.ReliefScale = 1
	}
	return cfg
}

func (m *Model) computeUpstreamContext(input Input) {
	n := len(input.FlowTo)
	sumSlope := make([]float64, n)
	sumSupply := make([]float64, n)
	count := make([]int, n)

	for i, to := range input.FlowTo {
		if !input.LandMask[i] || to < 0 || to >= n || !input.LandMask[to] {
			continue
		}

		supply := clamp(
			0.55*clamp(input.ReliefIndex[i], 0, 1)+
				0.30*clamp(input.FluvialPower[i], 0, 1)+
				0.15*clamp(input.TectonicSupport[i], 0, 1),
			0,
			1,
		)

		sumSlope[to] += input.Slope[i]
		sumSupply[to] += supply
		count[to]++
	}

	for i := range m.UpstreamSlope {
		if count[i] == 0 {
			m.UpstreamSlope[i] = input.Slope[i]
			m.SedimentSupply[i] = clamp(
				0.65*input.ReliefIndex[i]+0.35*input.FluvialPower[i],
				0,
				1,
			)
		} else {
			m.UpstreamSlope[i] = sumSlope[i] / float64(count[i])
			upSupply := sumSupply[i] / float64(count[i])
			localSupply := clamp(
				0.55*input.ReliefIndex[i]+
					0.30*input.FluvialPower[i]+
					0.15*input.TectonicSupport[i],
				0,
				1,
			)
			m.SedimentSupply[i] = clamp(0.72*upSupply+0.28*localSupply, 0, 1)
		}
	}

	// Accumulation is monotonic downstream for normal drainage networks, so
	// sorting by accumulation gives a stable upstream-to-downstream traversal.
	order := make([]int, 0, n)
	for i := range input.LandMask {
		if input.LandMask[i] {
			order = append(order, i)
		}
	}
	sort.Slice(order, func(a, b int) bool {
		ia, ib := order[a], order[b]
		if input.FlowAccumulation[ia] == input.FlowAccumulation[ib] {
			return ia < ib
		}
		return input.FlowAccumulation[ia] < input.FlowAccumulation[ib]
	})

	stepScale := m.Geometry.SpacingX() / world.DefaultMacroSpacingBlocks
	steepDecayPerHop := math.Pow(0.78, stepScale)
	mountainDecayPerHop := math.Pow(0.82, stepScale)

	for _, i := range order {
		localSteep := clamp(input.Slope[i]/m.Config.SlopeScale, 0, 1)
		localMountain := clamp(
			0.70*input.MountainWeight[i]+0.30*input.FoothillWeight[i],
			0,
			1,
		)

		if localSteep > m.UpstreamSteepMemory[i] {
			m.UpstreamSteepMemory[i] = localSteep
		}
		if localMountain > m.UpstreamMountainMemory[i] {
			m.UpstreamMountainMemory[i] = localMountain
		}

		to := input.FlowTo[i]
		if to < 0 || to >= n || !input.LandMask[to] {
			continue
		}

		steepMemory := m.UpstreamSteepMemory[i] * steepDecayPerHop
		if steepMemory > m.UpstreamSteepMemory[to] {
			m.UpstreamSteepMemory[to] = steepMemory
		}
		mountainMemory := m.UpstreamMountainMemory[i] * mountainDecayPerHop
		if mountainMemory > m.UpstreamMountainMemory[to] {
			m.UpstreamMountainMemory[to] = mountainMemory
		}
	}

	for i := range m.SlopeDrop {
		currentSlope := clamp(input.Slope[i]/m.Config.SlopeScale, 0, 1)
		m.SlopeDrop[i] = clamp(m.UpstreamSteepMemory[i]-currentSlope, 0, 1)
	}
}

func (m *Model) computeWeights(input Input) {
	for i, land := range input.LandMask {
		if !land {
			continue
		}

		slope := clamp(input.Slope[i]/m.Config.SlopeScale, 0, 1)
		relief := clamp(input.ReliefIndex[i]/m.Config.ReliefScale, 0, 1)
		river := clamp(input.RiverStrength[i], 0, 1)
		valley := clamp(input.ValleyWeight[i], 0, 1)
		plain := clamp(input.RollingPlainWeight[i], 0, 1)
		wetland := clamp(input.WetlandPotential[i], 0, 1)
		perennial := clamp(input.PerennialRiverPotential[i], 0, 1)

		accum := normalizedAccumulation(input.FlowAccumulation[i], input.RiverThreshold, m.Config.AccumulationScale)
		lowSlope := 1 - smoothstep(0.16, 0.52, slope)
		veryLowSlope := 1 - smoothstep(0.08, 0.34, slope)
		lowRelief := 1 - smoothstep(0.25, 0.58, relief)
		highRelief := smoothstep(0.42, 0.78, relief)
		highSlope := smoothstep(0.20, 0.66, slope)

		floodplain := valley *
			(0.30 + 0.70*accum) *
			lowSlope *
			(0.58 + 0.42*lowRelief) *
			(0.70 + 0.30*math.Max(perennial, wetland))
		if input.LakeID[i] > 0 {
			floodplain *= 0.18
		}
		m.FloodplainWeight[i] = clamp(floodplain, 0, 1)

		alluvialPlain := (0.45*plain + 0.35*m.FloodplainWeight[i] + 0.20*wetland) *
			(0.35 + 0.65*accum) *
			veryLowSlope *
			lowRelief
		if input.LakeID[i] > 0 {
			alluvialPlain *= 0.15
		}
		m.AlluvialPlainWeight[i] = clamp(alluvialPlain, 0, 1)

		slopeBreak := smoothstep(0.02, 0.16, m.SlopeDrop[i])
		mountainSource := smoothstep(0.08, 0.42, m.UpstreamMountainMemory[i])
		fanCore := river *
			(0.35 + 0.65*accum) *
			slopeBreak *
			(0.45 + 0.55*m.SedimentSupply[i]) *
			(0.25 + 0.75*mountainSource) *
			lowSlope *
			(1 - 0.45*highRelief)
		if input.LakeID[i] > 0 {
			fanCore *= 0.08
		}
		m.AlluvialFanCore[i] = clamp(fanCore, 0, 1)
		m.AlluvialFanWeight[i] = m.AlluvialFanCore[i]

		canyon := river *
			(0.30 + 0.70*accum) *
			highRelief *
			highSlope *
			(0.45 + 0.55*clamp(input.ValleyIncision[i]/0.015, 0, 1))
		if input.LakeID[i] > 0 {
			canyon = 0
		}
		m.CanyonWeight[i] = clamp(canyon, 0, 1)

		wideFlow := clamp(0.55*accum+0.45*m.FloodplainWeight[i], 0, 1)
		width := valley *
			(0.18 + 0.82*wideFlow) *
			(0.42 + 0.58*lowSlope) *
			(1 - 0.82*m.CanyonWeight[i])
		m.ValleyFloorWidth[i] = clamp(width, 0, 1)

	}
}

func (m *Model) spreadAlluvialFans(input Input) {
	for i, core := range m.AlluvialFanCore {
		if core <= 0.02 || !m.LandMask[i] {
			continue
		}

		to := input.FlowTo[i]
		if to < 0 || to >= len(m.LandMask) || !m.LandMask[to] {
			continue
		}

		x0, z0 := i%m.Width, i/m.Width
		x1, z1 := to%m.Width, to/m.Width
		dx := float64(x1 - x0)
		dz := float64(z1 - z0)
		length := math.Hypot(dx, dz)
		if length <= 0 {
			continue
		}
		dx /= length
		dz /= length

		accum := normalizedAccumulation(input.FlowAccumulation[i], input.RiverThreshold, m.Config.AccumulationScale)
		radiusRefCells := 2 + int(math.Round(4*accum))
		if radiusRefCells < 2 {
			radiusRefCells = 2
		}
		if radiusRefCells > 7 {
			radiusRefCells = 7
		}
		radiusRef := float64(radiusRefCells)
		cellSizeBlocks := m.Geometry.SpacingX()
		radiusBlocks := radiusRef * world.DefaultMacroSpacingBlocks
		radiusCells := int(math.Ceil(radiusBlocks / cellSizeBlocks))

		for oz := -radiusCells; oz <= radiusCells; oz++ {
			z := z0 + oz
			if z < 0 || z >= m.Height {
				continue
			}
			for ox := -radiusCells; ox <= radiusCells; ox++ {
				x := x0 + ox
				if x < 0 || x >= m.Width {
					continue
				}
				j := z*m.Width + x
				if !m.LandMask[j] || input.LakeID[j] > 0 {
					continue
				}

				rxBlocks := float64(ox) * cellSizeBlocks
				rzBlocks := float64(oz) * cellSizeBlocks
				forwardRef := (rxBlocks*dx + rzBlocks*dz) / world.DefaultMacroSpacingBlocks
				if forwardRef < -0.35 || forwardRef > radiusRef {
					continue
				}
				lateralRef := math.Abs(rxBlocks*dz-rzBlocks*dx) / world.DefaultMacroSpacingBlocks
				halfWidthRef := 0.75 + 0.70*math.Max(forwardRef, 0)
				if lateralRef > halfWidthRef {
					continue
				}

				distanceDecay := math.Exp(-math.Max(forwardRef, 0) / math.Max(1.5, radiusRef*0.75))
				lateralDecay := math.Exp(-math.Pow(lateralRef/math.Max(halfWidthRef*0.72, 0.5), 2))

				slopeNorm := clamp(input.Slope[j]/m.Config.SlopeScale, 0, 1)
				relief := clamp(input.ReliefIndex[j], 0, 1)
				lowlandSuitability :=
					(1 - smoothstep(0.18, 0.62, slopeNorm)) *
						(1 - smoothstep(0.45, 0.78, relief)) *
						(0.35 + 0.65*clamp(input.RollingPlainWeight[j]+input.FoothillWeight[j], 0, 1))

				w := core * distanceDecay * lateralDecay * lowlandSuitability
				if w > m.AlluvialFanWeight[j] {
					m.AlluvialFanWeight[j] = clamp(w, 0, 1)
				}
			}
		}
	}
}

func (m *Model) computeDeposition() {
	for i, land := range m.LandMask {
		if !land {
			continue
		}
		deposition := (0.42*m.FloodplainWeight[i] +
			0.32*m.AlluvialPlainWeight[i] +
			0.26*m.AlluvialFanWeight[i]) *
			(0.42 + 0.58*m.SedimentSupply[i])
		m.DepositionPotential[i] = clamp(deposition, 0, 1)
	}
}

func normalizedAccumulation(accumulation, threshold, scale float64) float64 {
	if accumulation <= 0 {
		return 0
	}
	if threshold <= 0 {
		threshold = 1
	}
	if scale <= 1 {
		scale = 8
	}
	return clamp(math.Log1p(accumulation/threshold)/math.Log1p(scale), 0, 1)
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
