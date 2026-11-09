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
		{"total erosion", input.TotalErosion},
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
		Config:                      cfg,
		Geometry:                    geometry,
		Width:                       input.Width,
		Height:                      input.Height,
		LandMask:                    append([]bool(nil), input.LandMask...),
		UpstreamSlope:               make([]float64, n),
		UpstreamSteepMemory:         make([]float64, n),
		UpstreamMountainMemory:      make([]float64, n),
		SlopeDrop:                   make([]float64, n),
		SedimentSupply:              make([]float64, n),
		SedimentProduction:          make([]float64, n),
		SedimentFlux:                make([]float64, n),
		TransportCapacity:           make([]float64, n),
		SedimentDeposition:          make([]float64, n),
		SedimentExport:              make([]float64, n),
		FloodplainWeight:            make([]float64, n),
		AlluvialPlainWeight:         make([]float64, n),
		AlluvialFanCore:             make([]float64, n),
		AlluvialFanWeight:           make([]float64, n),
		CanyonWeight:                make([]float64, n),
		ValleyFloorWidth:            make([]float64, n),
		DepositionPotential:         make([]float64, n),
		LongitudinalGradeCorrection: make([]float64, n),
	}

	m.computeUpstreamContext(input)
	m.computeWeights(input)
	m.spreadAlluvialFans(input)
	m.computeSedimentBudget(input)
	m.computeDeposition()
	if m.Config.LongitudinalGradeVersion >= 1 {
		m.computeLongitudinalGrade(input)
	}
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

func (m *Model) computeSedimentBudget(input Input) {
	n := len(input.FlowTo)
	if n == 0 {
		return
	}

	indegree := make([]int, n)
	for i, to := range input.FlowTo {
		if !input.LandMask[i] ||
			to < 0 || to >= n ||
			!input.LandMask[to] ||
			to == i {
			continue
		}
		indegree[to]++
	}

	queue := make([]int, 0, n)
	landCount := 0
	for i, land := range input.LandMask {
		if !land {
			continue
		}
		landCount++
		if indegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	cellAreaScale :=
		(m.Geometry.SpacingX() * m.Geometry.SpacingZ()) /
			(world.DefaultMacroSpacingBlocks * world.DefaultMacroSpacingBlocks)
	for i, land := range input.LandMask {
		if !land {
			continue
		}
		// TotalErosion is the actual normalized height removed by Erosion,
		// after all sea-level and resistance limits. Multiplying by cell-area
		// scale turns it into a resolution-independent volume proxy.
		m.SedimentProduction[i] =
			math.Max(0, input.TotalErosion[i]) * cellAreaScale
	}

	pending := make([]float64, n)
	processed := make([]bool, n)
	head := 0
	for head < len(queue) {
		i := queue[head]
		head++
		if processed[i] {
			continue
		}
		processed[i] = true

		available := pending[i] + m.SedimentProduction[i]
		capacity := m.sedimentTransportCapacity(input, i)
		accommodation := m.sedimentAccommodation(input, i)

		// Capacity is a through-flow fraction, not a second sediment source.
		// Low energy alone does not force full deposition unless the local
		// landform also provides accommodation space.
		depositionFraction :=
			(1 - capacity) * (0.18 + 0.82*accommodation)
		if input.LakeID[i] > 0 {
			// Lakes are strong sediment traps. Keep some through-flow so this
			// remains a routing proxy rather than an absolute reservoir model.
			depositionFraction = math.Max(depositionFraction, 0.68)
		}
		depositionFraction = clamp(depositionFraction, 0, 0.95)

		deposition := available * depositionFraction
		carry := math.Max(0, available-deposition)

		m.TransportCapacity[i] = capacity
		m.SedimentDeposition[i] = deposition
		m.SedimentFlux[i] = carry

		to := input.FlowTo[i]
		if to >= 0 && to < n &&
			input.LandMask[to] &&
			to != i {
			pending[to] += carry
			indegree[to]--
			if indegree[to] == 0 {
				queue = append(queue, to)
			}
		} else {
			m.SedimentExport[i] = carry
		}
	}

	if len(queue) == landCount {
		return
	}

	// Hydrology normally guarantees an acyclic FlowTo network. If an invalid
	// caller supplies a cycle, do not silently create or destroy proxy mass:
	// expose all unprocessed material as terminal export.
	for i, land := range input.LandMask {
		if !land || processed[i] {
			continue
		}
		available := pending[i] + m.SedimentProduction[i]
		m.SedimentFlux[i] = available
		m.SedimentExport[i] = available
	}
}

func (m *Model) sedimentTransportCapacity(input Input, i int) float64 {
	if i < 0 || i >= len(input.LandMask) || !input.LandMask[i] {
		return 0
	}

	accum := normalizedAccumulation(
		input.FlowAccumulation[i],
		input.RiverThreshold,
		m.Config.AccumulationScale,
	)
	slopeNorm := clamp(
		input.Slope[i]/math.Max(m.Config.SlopeScale, 1e-9),
		0,
		1,
	)
	river := clamp(input.RiverStrength[i], 0, 1)
	fluvial := clamp(input.FluvialPower[i], 0, 1)

	capacity := 0.08 +
		0.30*math.Sqrt(accum) +
		0.32*fluvial +
		0.18*slopeNorm +
		0.12*river
	if input.LakeID[i] > 0 {
		capacity *= 0.25
	}
	return clamp(capacity, 0.02, 0.98)
}

func (m *Model) sedimentAccommodation(input Input, i int) float64 {
	if i < 0 || i >= len(input.LandMask) || !input.LandMask[i] {
		return 0
	}
	if input.LakeID[i] > 0 {
		return 1
	}

	slopeNorm := clamp(
		input.Slope[i]/math.Max(m.Config.SlopeScale, 1e-9),
		0,
		1,
	)
	relief := clamp(input.ReliefIndex[i], 0, 1)
	lowSlope := 1 - smoothstep(0.16, 0.60, slopeNorm)
	lowRelief := 1 - smoothstep(0.30, 0.70, relief)

	landformSpace := clamp(
		0.42*m.FloodplainWeight[i]+
			0.34*m.AlluvialPlainWeight[i]+
			0.24*m.AlluvialFanWeight[i],
		0,
		1,
	)
	backgroundSpace := 0.35 * lowSlope * lowRelief
	return clamp(
		math.Max(landformSpace, backgroundSpace),
		0,
		1,
	)
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

func (m *Model) computeLongitudinalGrade(input Input) {
	n := len(input.FlowTo)
	if n == 0 {
		return
	}

	// A river's long profile is a network property. Select one main-stem
	// predecessor per node from accumulated discharge, then grade contiguous
	// mature lowland reaches as whole reaches. This avoids local relaxation
	// stopping at one weak alluvial cell and leaving the broad downstream
	// knickzone untouched.
	mainUpstream := make([]int, n)
	for i := range mainUpstream {
		mainUpstream[i] = -1
	}
	for i, to := range input.FlowTo {
		if to < 0 || to >= n || !input.LandMask[i] || !input.LandMask[to] {
			continue
		}
		current := mainUpstream[to]
		if current < 0 ||
			input.FlowAccumulation[i] > input.FlowAccumulation[current] ||
			(input.FlowAccumulation[i] == input.FlowAccumulation[current] && i < current) {
			mainUpstream[to] = i
		}
	}

	influence := make([]float64, n)
	for i := 0; i < n; i++ {
		if !input.LandMask[i] || input.LakeID[i] > 0 {
			continue
		}
		to := input.FlowTo[i]
		if to < 0 || to >= n || !input.LandMask[to] || input.LakeID[to] > 0 {
			continue
		}

		accum := normalizedAccumulation(
			input.FlowAccumulation[i],
			input.RiverThreshold,
			m.Config.AccumulationScale,
		)
		riverSignal := smoothstep(
			0.06,
			0.48,
			clamp(input.RiverStrength[i], 0, 1)*(0.30+0.70*accum),
		)

		alluvial := clamp(
			0.45*m.AlluvialPlainWeight[i]+
				0.30*m.FloodplainWeight[i]+
				0.25*m.DepositionPotential[i],
			0,
			1,
		)
		lowlandForm := clamp(
			0.45*input.RollingPlainWeight[i]+
				0.30*input.ValleyWeight[i]+
				0.25*alluvial,
			0,
			1,
		)
		lowlandGate := smoothstep(0.08, 0.45, lowlandForm)

		// Preserve genuine mountain/canyon rivers. Lowland grading is allowed
		// to pass through rolling plains and broad valleys, but not through a
		// tectonically supported steep reach.
		highReliefForm := clamp(
			0.65*input.MountainWeight[i]+
				0.35*input.FoothillWeight[i]+
				0.75*m.CanyonWeight[i],
			0,
			1,
		)
		reliefGate := 1 - smoothstep(
			0.48,
			0.82,
			clamp(input.ReliefIndex[i], 0, 1),
		)
		maturity := 1 - 0.50*clamp(input.TectonicSupport[i], 0, 1)

		influence[i] = clamp(
			riverSignal*
				(0.55+0.45*lowlandGate)*
				(1-highReliefForm)*
				reliefGate*
				maturity,
			0,
			1,
		)
	}

	eligible := make([]bool, n)
	for i, w := range influence {
		eligible[i] = w >= 0.08
	}

	const (
		minReachNodes        = 5
		maxCorrection        = 0.024
		minimumLandFreeboard = 0.001
	)

	for start := 0; start < n; start++ {
		if !eligible[start] {
			continue
		}
		up := mainUpstream[start]
		if up >= 0 && up < n && eligible[up] && input.FlowTo[up] == start {
			// This node belongs to an upstream continuation of the same main
			// stem and will be handled by that reach.
			continue
		}

		reach := make([]int, 0, 16)
		current := start
		for current >= 0 && current < n && eligible[current] {
			reach = append(reach, current)
			to := input.FlowTo[current]
			if to < 0 || to >= n || !eligible[to] || mainUpstream[to] != current {
				break
			}
			current = to
		}
		if len(reach) < minReachNodes {
			continue
		}

		distances := make([]float64, len(reach))
		for k := 1; k < len(reach); k++ {
			distances[k] = distances[k-1] +
				gridNodeDistanceBlocks(m.Geometry, reach[k-1], reach[k])
		}
		length := distances[len(distances)-1]
		if length < 512 {
			continue
		}
		upperProcess, lowerProcess := m.reachAlluvialProcessEvidence(
			reach,
			distances,
		)
		if upperProcess < 0.14 ||
			lowerProcess < 0.22 ||
			lowerProcess-upperProcess < 0.04 {
			// Grade only reaches that are actually behaving as a mature
			// alluvial system. Plain terrain by itself is not sufficient, and
			// downstream alluvial/depositional evidence must strengthen as river
			// energy falls; otherwise a plain stream is not a deposition-driven
			// graded reach.
			continue
		}

		startElevation := input.Elevation[reach[0]]
		endElevation := input.Elevation[reach[len(reach)-1]]
		if startElevation <= endElevation+1e-9 {
			continue
		}

		upperSlope, lowerSlope, ok := reachThirdSlopes(
			reach,
			distances,
			input.Elevation,
		)
		if !ok ||
			lowerSlope <= upperSlope*1.35 ||
			lowerSlope-upperSlope < 0.008 {
			// Already-graded or naturally concave lowland reaches are not
			// touched. The correction is reserved for broad convex profiles
			// whose lower third is materially steeper than the upper third.
			continue
		}

		for k := 1; k+1 < len(reach); k++ {
			i := reach[k]
			fraction := distances[k] / length
			// Mature graded rivers are generally concave-up in long profile:
			// steeper upstream, progressively gentler downstream. Keep the
			// reach anchors fixed and use a mild concave equilibrium target.
			const profileExponent = 1.22
			target := endElevation +
				(startElevation-endElevation)*
					math.Pow(1-fraction, profileExponent)
			delta := target - input.Elevation[i]
			if math.Abs(delta) <= 1e-12 {
				continue
			}

			var processStrength float64
			if delta > 0 {
				// Filling a low spot is deposition-controlled. River-network
				// eligibility says this is a valid lowland reach, but available
				// sediment still limits how much accommodation space can fill.
				depositionSignal := clamp(
					0.55*m.DepositionPotential[i]+
						0.30*m.SedimentSupply[i]+
						0.15*m.AlluvialPlainWeight[i],
					0,
					1,
				)
				processStrength = 0.45 + 0.55*depositionSignal
			} else {
				// Once a mature lowland reach is eligible, fluvial incision
				// should actually remove a convex longitudinal high point.
				// Do not multiply by the eligibility weight a second time.
				processStrength = 0.75 +
					0.25*clamp(input.FluvialPower[i], 0, 1)
			}

			// Corrections fade gently toward reach anchors and remain bounded.
			// Endpoint elevations still come from the existing hydrology /
			// erosion solution; this only redistributes interior relief.
			edgeBlend := 0.70 + 0.30*math.Sin(math.Pi*fraction)
			correction := delta * processStrength * edgeBlend
			correction = clamp(correction, -maxCorrection, maxCorrection)

			lo := m.Config.SeaLevelNormalized +
				minimumLandFreeboard - input.Elevation[i]
			hi := 0.999 - input.Elevation[i]
			m.LongitudinalGradeCorrection[i] = clamp(correction, lo, hi)
		}
	}
}

func (m *Model) reachAlluvialProcessEvidence(
	reach []int,
	distances []float64,
) (float64, float64) {
	if len(reach) == 0 || len(distances) != len(reach) {
		return 0, 0
	}
	length := distances[len(distances)-1]
	if length <= 1e-9 {
		return 0, 0
	}
	upperSum, lowerSum := 0.0, 0.0
	upperCount, lowerCount := 0.0, 0.0
	for k, i := range reach {
		if i < 0 ||
			i >= len(m.AlluvialPlainWeight) ||
			i >= len(m.FloodplainWeight) ||
			i >= len(m.DepositionPotential) {
			continue
		}
		evidence := clamp(
			0.50*m.AlluvialPlainWeight[i]+
				0.30*m.FloodplainWeight[i]+
				0.20*m.DepositionPotential[i],
			0,
			1,
		)
		q := distances[k] / length
		if q <= 1.0/3.0 {
			upperSum += evidence
			upperCount++
		}
		if q >= 2.0/3.0 {
			lowerSum += evidence
			lowerCount++
		}
	}
	if upperCount == 0 || lowerCount == 0 {
		return 0, 0
	}
	return upperSum / upperCount, lowerSum / lowerCount
}

func reachThirdSlopes(
	reach []int,
	distances []float64,
	elevation []float64,
) (float64, float64, bool) {
	if len(reach) < 3 || len(distances) != len(reach) {
		return 0, 0, false
	}
	length := distances[len(distances)-1]
	if length <= 1e-9 {
		return 0, 0, false
	}
	at := func(distance float64) (float64, bool) {
		if distance <= 0 {
			i := reach[0]
			return elevation[i], i >= 0 && i < len(elevation)
		}
		for k := 1; k < len(reach); k++ {
			if distances[k] < distance {
				continue
			}
			a, b := reach[k-1], reach[k]
			if a < 0 || b < 0 || a >= len(elevation) || b >= len(elevation) {
				return 0, false
			}
			span := distances[k] - distances[k-1]
			if span <= 1e-9 {
				return elevation[b], true
			}
			t := (distance - distances[k-1]) / span
			return elevation[a] + (elevation[b]-elevation[a])*t, true
		}
		i := reach[len(reach)-1]
		if i < 0 || i >= len(elevation) {
			return 0, false
		}
		return elevation[i], true
	}

	start, okStart := at(0)
	q1, okQ1 := at(length / 3)
	q2, okQ2 := at(2 * length / 3)
	end, okEnd := at(length)
	if !okStart || !okQ1 || !okQ2 || !okEnd {
		return 0, 0, false
	}
	thirdKm := length / 3 / 1000
	if thirdKm <= 1e-9 {
		return 0, 0, false
	}
	return (start - q1) / thirdKm, (q2 - end) / thirdKm, true
}

func gridNodeDistanceBlocks(g world.GridGeometry, a, b int) float64 {
	if a < 0 || b < 0 || a >= g.Width*g.Height || b >= g.Width*g.Height {
		return 0
	}
	ax, az := a%g.Width, a/g.Width
	bx, bz := b%g.Width, b/g.Width
	dx := float64(ax-bx) * g.SpacingX()
	dz := float64(az-bz) * g.SpacingZ()
	return math.Hypot(dx, dz)
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
