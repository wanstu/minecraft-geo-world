package reconcile

import (
	"errors"
	"fmt"
	"math"

	"geoworld/internal/hydrology"
	"geoworld/internal/sampling"
	"geoworld/internal/terrain"
	"geoworld/internal/world"
)

type Input struct {
	Geometry           world.GridGeometry
	SeaLevelNormalized float64

	MacroSurface   *world.Surface
	MacroElevation []float64
	Hydrology      *hydrology.Model
	Terrain        *terrain.Model
}

func Build(input Input, cfg Config) (*Model, error) {
	if _, err := input.Geometry.SquareCellSize(); err != nil {
		return nil, fmt.Errorf("reconcile: geometry: %w", err)
	}
	n, err := world.NodeCount(input.Geometry.Width, input.Geometry.Height)
	if err != nil {
		return nil, fmt.Errorf("reconcile: geometry: %w", err)
	}
	if !finite(input.SeaLevelNormalized) || input.SeaLevelNormalized <= 0 || input.SeaLevelNormalized >= 1 {
		return nil, errors.New("reconcile: sea level must be finite and between 0 and 1")
	}
	if input.MacroSurface == nil {
		return nil, errors.New("reconcile: macro surface is required")
	}
	if input.MacroSurface.Width != input.Geometry.Width || input.MacroSurface.Height != input.Geometry.Height ||
		len(input.MacroSurface.LandMask) != n || len(input.MacroSurface.EnclosedBelowSea) != n {
		return nil, errors.New("reconcile: macro surface dimensions do not match geometry")
	}
	if math.Abs(input.MacroSurface.SeaLevelNormalized-input.SeaLevelNormalized) > 1e-12 {
		return nil, errors.New("reconcile: macro surface sea level conflicts with input")
	}
	if len(input.MacroElevation) != n {
		return nil, errors.New("reconcile: macro elevation size does not match geometry")
	}
	if input.Hydrology == nil {
		return nil, errors.New("reconcile: hydrology model is required")
	}
	if input.Hydrology.Width != input.Geometry.Width || input.Hydrology.Height != input.Geometry.Height ||
		!sameGeometry(input.Hydrology.Geometry, input.Geometry) {
		return nil, errors.New("reconcile: hydrology geometry does not match")
	}
	if len(input.Hydrology.FlowTo) != n || len(input.Hydrology.LandMask) != n ||
		len(input.Hydrology.LakeID) != n || len(input.Hydrology.RiverStrength) != n {
		return nil, errors.New("reconcile: hydrology field sizes do not match geometry")
	}
	if input.Terrain == nil || !sameGeometry(input.Terrain.Geometry, input.Geometry) {
		return nil, errors.New("reconcile: terrain geometry does not match")
	}

	cfg, err = normalizeConfig(cfg, input.SeaLevelNormalized)
	if err != nil {
		return nil, err
	}

	m := &Model{
		Config:                cfg,
		Geometry:              input.Geometry,
		SeaLevelNormalized:    input.SeaLevelNormalized,
		RawTerrainElevation:   make([]float64, n),
		CoastTargetElevation:  make([]float64, n),
		CoastCorrection:       make([]float64, n),
		WaterKind:             make([]WaterKind, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		RiverBedActive:        make([]bool, n),
		RiverBedTarget:        make([]float64, n),
	}

	for z := 0; z < input.Geometry.Height; z++ {
		for x := 0; x < input.Geometry.Width; x++ {
			i := z*input.Geometry.Width + x
			wx, wz := input.Geometry.NodePosition(x, z)
			h := input.Terrain.SampleNormalized(wx, wz)
			if !finite(h) || h < 0 || h > 1 {
				return nil, errors.New("reconcile: terrain sample must be finite and normalized")
			}
			m.RawTerrainElevation[i] = h
			m.CoastTargetElevation[i] = h
			m.RiverBedTarget[i] = h
		}
	}

	m.RawTerrainSurface, err = world.ClassifySurface(
		input.Geometry.Width,
		input.Geometry.Height,
		m.RawTerrainElevation,
		input.SeaLevelNormalized,
	)
	if err != nil {
		return nil, fmt.Errorf("reconcile: classify raw terrain surface: %w", err)
	}
	m.SurfaceMismatchBefore = maskMismatchCount(input.MacroSurface.LandMask, m.RawTerrainSurface.LandMask)

	coastLow := input.SeaLevelNormalized - cfg.CoastMarginNormalized
	coastHigh := input.SeaLevelNormalized + cfg.CoastMarginNormalized
	for i, macroLand := range input.MacroSurface.LandMask {
		old := m.CoastTargetElevation[i]
		target := old

		if !macroLand {
			if target >= input.SeaLevelNormalized {
				target = coastLow
			}
			m.WaterKind[i] = WaterOcean
			m.WaterSurfaceDefined[i] = true
			m.WaterSurfaceElevation[i] = input.SeaLevelNormalized
		} else if !input.MacroSurface.EnclosedBelowSea[i] && target < input.SeaLevelNormalized {
			target = coastHigh
		}

		m.CoastCorrection[i] = target - old
		m.CoastTargetElevation[i] = target
	}

	if err := m.refineCoastCorrection(input.Terrain, input.MacroSurface); err != nil {
		return nil, err
	}
	m.finalizeCoastTargets()

	m.ConstrainedSurface, err = world.ClassifySurface(
		input.Geometry.Width,
		input.Geometry.Height,
		m.CoastTargetElevation,
		input.SeaLevelNormalized,
	)
	if err != nil {
		return nil, fmt.Errorf("reconcile: classify constrained surface: %w", err)
	}
	m.SurfaceMismatchAfter = maskMismatchCount(input.MacroSurface.LandMask, m.ConstrainedSurface.LandMask)

	if err := m.buildLakeConstraints(input.Hydrology); err != nil {
		return nil, err
	}
	if err := m.buildRiverConstraints(input.MacroElevation, input.Hydrology, m.RawTerrainElevation); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Model) refineCoastCorrection(raw *terrain.Model, surface *world.Surface) error {
	if raw == nil || surface == nil {
		return errors.New("reconcile: coast refinement requires terrain and surface")
	}
	offsets := [][2]float64{
		{0.25, 0.25},
		{0.75, 0.25},
		{0.50, 0.50},
		{0.25, 0.75},
		{0.75, 0.75},
	}
	coastLow := m.SeaLevelNormalized - m.Config.CoastMarginNormalized
	coastHigh := m.SeaLevelNormalized + m.Config.CoastMarginNormalized
	spacing := m.Geometry.SpacingX()

	for iteration := 0; iteration < 4; iteration++ {
		grid, err := sampling.NewScalarGridWithGeometry(m.Geometry, m.CoastCorrection)
		if err != nil {
			return fmt.Errorf("reconcile: build coast correction grid: %w", err)
		}
		adjustment := make([]float64, len(m.CoastCorrection))
		changed := false

		for z := 0; z < m.Geometry.Height-1; z++ {
			for x := 0; x < m.Geometry.Width-1; x++ {
				indices := [4]int{
					z*m.Geometry.Width + x,
					z*m.Geometry.Width + x + 1,
					(z+1)*m.Geometry.Width + x,
					(z+1)*m.Geometry.Width + x + 1,
				}

				allOcean := true
				allNormalLand := true
				for _, i := range indices {
					if surface.LandMask[i] {
						allOcean = false
					} else {
						allNormalLand = false
					}
					if !surface.LandMask[i] || surface.EnclosedBelowSea[i] {
						allNormalLand = false
					}
				}
				if !allOcean && !allNormalLand {
					continue
				}

				required := 0.0
				for _, off := range offsets {
					wx := m.Geometry.OriginX + (float64(x)+off[0])*spacing
					wz := m.Geometry.OriginZ + (float64(z)+off[1])*spacing
					h := clamp(
						raw.SampleNormalized(wx, wz)+grid.SampleSmoothstepBilinear(wx, wz),
						0,
						1,
					)
					if allOcean {
						required = math.Max(required, h-coastLow)
					} else {
						required = math.Max(required, coastHigh-h)
					}
				}
				if required <= 1e-12 {
					continue
				}
				changed = true
				for _, i := range indices {
					if allOcean {
						adjustment[i] = math.Min(adjustment[i], -required)
					} else {
						adjustment[i] = math.Max(adjustment[i], required)
					}
				}
			}
		}

		if !changed {
			return nil
		}
		for i, delta := range adjustment {
			m.CoastCorrection[i] += delta
			if !finite(m.CoastCorrection[i]) {
				return errors.New("reconcile: coast refinement produced non-finite correction")
			}
		}
	}

	grid, err := sampling.NewScalarGridWithGeometry(m.Geometry, m.CoastCorrection)
	if err != nil {
		return err
	}
	for z := 0; z < m.Geometry.Height-1; z++ {
		for x := 0; x < m.Geometry.Width-1; x++ {
			indices := [4]int{
				z*m.Geometry.Width + x,
				z*m.Geometry.Width + x + 1,
				(z+1)*m.Geometry.Width + x,
				(z+1)*m.Geometry.Width + x + 1,
			}
			allOcean := true
			allNormalLand := true
			for _, i := range indices {
				if surface.LandMask[i] {
					allOcean = false
				} else {
					allNormalLand = false
				}
				if !surface.LandMask[i] || surface.EnclosedBelowSea[i] {
					allNormalLand = false
				}
			}
			if !allOcean && !allNormalLand {
				continue
			}
			for _, off := range offsets {
				wx := m.Geometry.OriginX + (float64(x)+off[0])*spacing
				wz := m.Geometry.OriginZ + (float64(z)+off[1])*spacing
				h := clamp(raw.SampleNormalized(wx, wz)+grid.SampleSmoothstepBilinear(wx, wz), 0, 1)
				if allOcean && h > coastLow+1e-10 {
					return errors.New("reconcile: ocean interior coast refinement did not converge")
				}
				if allNormalLand && h < coastHigh-1e-10 {
					return errors.New("reconcile: land interior coast refinement did not converge")
				}
			}
		}
	}
	return nil
}

func (m *Model) finalizeCoastTargets() {
	m.CoastAdjustedNodes = 0
	m.CoastRaisedNodes = 0
	m.CoastCarvedNodes = 0
	m.MaxCoastRaise = 0
	m.MaxCoastCarve = 0

	for i, raw := range m.RawTerrainElevation {
		target := clamp(raw+m.CoastCorrection[i], 0, 1)
		m.CoastTargetElevation[i] = target
		delta := target - raw
		if math.Abs(delta) <= 1e-15 {
			continue
		}
		m.CoastAdjustedNodes++
		if delta > 0 {
			m.CoastRaisedNodes++
			m.MaxCoastRaise = math.Max(m.MaxCoastRaise, delta)
		} else {
			m.CoastCarvedNodes++
			m.MaxCoastCarve = math.Max(m.MaxCoastCarve, -delta)
		}
	}
}

func (m *Model) buildLakeConstraints(h *hydrology.Model) error {
	lakeSurface := make(map[int]float64, len(h.Lakes))
	for _, lake := range h.Lakes {
		if lake.ID <= 0 || !finite(lake.SurfaceElevation) || lake.SurfaceElevation < 0 || lake.SurfaceElevation > 1 {
			return errors.New("reconcile: invalid lake metadata")
		}
		if _, exists := lakeSurface[lake.ID]; exists {
			return errors.New("reconcile: duplicate lake id")
		}
		lakeSurface[lake.ID] = lake.SurfaceElevation
	}

	for i, id := range h.LakeID {
		if id <= 0 {
			continue
		}
		surface, ok := lakeSurface[id]
		if !ok {
			return fmt.Errorf("reconcile: lake cell references missing lake id %d", id)
		}
		m.WaterKind[i] = WaterLake
		m.WaterSurfaceDefined[i] = true
		m.WaterSurfaceElevation[i] = surface

		ceiling := math.Max(0, surface-m.Config.LakeBedClearanceNormalized)
		m.LakeBedCeilingDefined[i] = true
		m.LakeBedCeiling[i] = ceiling
		if m.RawTerrainElevation[i] > ceiling+1e-12 {
			m.RawLakeBedViolations++
			m.LakeBedViolations++
		}
	}
	return nil
}

func (m *Model) buildRiverConstraints(macroElevation []float64, h *hydrology.Model, bedBase []float64) error {
	n := len(h.FlowTo)
	if len(macroElevation) != n || len(bedBase) != n {
		return errors.New("reconcile: river constraint inputs do not match hydrology dimensions")
	}

	for i := range m.WaterKind {
		if m.WaterKind[i] == WaterRiver {
			m.WaterKind[i] = WaterNone
		}
	}
	m.RiverBaseElevation = append(m.RiverBaseElevation[:0], bedBase...)
	m.RiverBedTarget = append(m.RiverBedTarget[:0], bedBase...)
	m.RiverBedActive = make([]bool, n)
	m.RiverEdges = nil
	m.QualifyingRiverEdges = 0
	m.OriginallyDownhillEdges = 0
	m.RawUpslopeRiverEdges = 0
	m.TargetUpslopeRiverEdges = 0
	m.RiverBedCarvedNodes = 0
	m.MaxRiverBedCarve = 0

	edgeTo := make([]int, n)
	active := make([]bool, n)
	indegree := make([]int, n)
	edgeStrengthFloor := make([]float64, n)
	parents := make([][]int, n)
	for i := range edgeTo {
		edgeTo[i] = -1
	}
	for from, to := range h.FlowTo {
		if to >= 0 && to < n {
			parents[to] = append(parents[to], from)
		}
	}

	traceRiver := func(seed int, strengthFloor float64) {
		cur := seed
		for steps := 0; steps < n; steps++ {
			if cur < 0 || cur >= n || !h.LandMask[cur] {
				break
			}
			next := h.FlowTo[cur]
			if next < 0 || next >= n {
				break
			}
			if active[cur] {
				if strengthFloor > edgeStrengthFloor[cur] {
					edgeStrengthFloor[cur] = strengthFloor
				}
				// This source has joined a downstream chain already traced by
				// another source. The rest of the route is already guaranteed.
				break
			}

			active[cur] = true
			edgeTo[cur] = next
			indegree[next]++
			if strengthFloor > edgeStrengthFloor[cur] {
				edgeStrengthFloor[cur] = strengthFloor
			}
			m.RiverBedActive[cur] = true
			m.RiverBedActive[next] = true

			// The coarse LakeID boundary is only the handoff point. Stop the
			// normal FlowTo trace as soon as it enters a lake candidate; Waterbed
			// later knows the final detailed lake footprint and connects this
			// inlet to the actual rendered shoreline. A spill-out trace starts
			// inside the lake but has a non-lake next node, so it continues.
			if !h.LandMask[next] || h.LakeID[next] != 0 {
				break
			}
			cur = next
		}
	}

	// RiverStrength decides where visible tributaries begin, but once a stream
	// has begun its downstream FlowTo chain is topology, not an optional local
	// classification. Trace every qualifying source all the way into a lake,
	// ocean, or model outlet.
	for seed := range h.FlowTo {
		if !h.LandMask[seed] ||
			h.LakeID[seed] != 0 ||
			h.RiverStrength[seed] < m.Config.RiverStrengthThreshold {
			continue
		}
		to := h.FlowTo[seed]
		if to < 0 || to >= n {
			continue
		}
		m.QualifyingRiverEdges++
		traceRiver(seed, 0)
	}

	// Large or small, a hydrologic lake should visibly belong to its drainage
	// network. Guarantee at least one feeder by tracing the strongest direct
	// inflow a few macro cells upstream even when that feeder falls just below
	// the global visible-river threshold. Also continue every exorheic lake from
	// its spill outlet so Runtime does not render a visually dead reservoir.
	bestInflow := make(map[int]int)
	bestInflowAccum := make(map[int]float64)
	bestInflowStrength := make(map[int]float64)
	for from, to := range h.FlowTo {
		if to < 0 || to >= n || !h.LandMask[from] {
			continue
		}
		lakeID := h.LakeID[to]
		if lakeID <= 0 || h.LakeID[from] == lakeID {
			continue
		}
		acc := h.FlowAccumulation[from]
		if old, ok := bestInflowAccum[lakeID]; !ok || acc > old {
			bestInflowAccum[lakeID] = acc
			bestInflow[lakeID] = from
		}
		if h.RiverStrength[from] > bestInflowStrength[lakeID] {
			bestInflowStrength[lakeID] = h.RiverStrength[from]
		}
	}
	for _, lake := range h.Lakes {
		if direct, ok := bestInflow[lake.ID]; ok {
			seed := direct
			// Expose enough upstream length to read as an actual tributary,
			// rather than a one-cell blue stub touching a large lake.
			for hops := 0; hops < 3; hops++ {
				bestParent := -1
				bestAccum := -1.0
				for _, parent := range parents[seed] {
					if parent < 0 || parent >= n ||
						!h.LandMask[parent] ||
						h.LakeID[parent] != 0 {
						continue
					}
					if h.FlowAccumulation[parent] > bestAccum {
						bestAccum = h.FlowAccumulation[parent]
						bestParent = parent
					}
				}
				if bestParent < 0 {
					break
				}
				seed = bestParent
			}
			traceRiver(
				seed,
				math.Max(
					m.Config.RiverStrengthThreshold,
					bestInflowStrength[lake.ID],
				),
			)
		}

		outlet := lake.SpillOutlet
		if outlet < 0 || outlet >= n {
			continue
		}
		to := h.FlowTo[outlet]
		if to < 0 || to >= n || h.LakeID[to] == lake.ID {
			continue
		}
		traceRiver(
			outlet,
			math.Max(
				m.Config.RiverStrengthThreshold,
				bestInflowStrength[lake.ID],
			),
		)
	}

	for i, isActive := range active {
		if !isActive {
			continue
		}
		to := edgeTo[i]
		if m.WaterKind[i] == WaterNone {
			m.WaterKind[i] = WaterRiver
		}
		if m.WaterKind[to] == WaterNone {
			m.WaterKind[to] = WaterRiver
		}

		originalDrop := macroElevation[i] - macroElevation[to]
		if originalDrop > m.Config.OriginalDownhillEpsilon {
			m.OriginallyDownhillEdges++
		}
		rawRise := bedBase[to] - bedBase[i]
		if rawRise > m.Config.FinalUpslopeEpsilon {
			m.RawUpslopeRiverEdges++
		}

		strength := math.Max(
			math.Max(
				h.RiverStrength[i],
				m.Config.RiverStrengthThreshold,
			),
			edgeStrengthFloor[i],
		)
		flowAccumulation := 0.0
		if i < len(h.FlowAccumulation) {
			flowAccumulation = h.FlowAccumulation[i]
		}
		riverOrder := uint8(0)
		if i < len(h.RiverOrder) {
			riverOrder = h.RiverOrder[i]
		}
		m.RiverEdges = append(m.RiverEdges, RiverEdgeConstraint{
			From: i, To: to,
			RiverStrength:    strength,
			FlowAccumulation: flowAccumulation,
			RiverOrder:       riverOrder,
			OriginalDrop:     originalDrop,
			RawTerrainRise:   rawRise,
		})
	}

	queue := make([]int, 0, n)
	for i := 0; i < n; i++ {
		if indegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	processedEdges := 0
	for head := 0; head < len(queue); head++ {
		i := queue[head]
		if !active[i] {
			continue
		}
		to := edgeTo[i]
		limit := m.RiverBedTarget[i] - m.Config.MinRiverBedDropNormalized
		if limit < 0 {
			limit = 0
		}
		if m.RiverBedTarget[to] > limit {
			m.RiverBedTarget[to] = limit
		}
		processedEdges++
		indegree[to]--
		if indegree[to] == 0 {
			queue = append(queue, to)
		}
	}

	if processedEdges != len(m.RiverEdges) {
		return errors.New("reconcile: river constraint graph contains a cycle")
	}

	for i := range m.RiverEdges {
		edge := &m.RiverEdges[i]
		edge.TargetBedDrop = m.RiverBedTarget[edge.From] - m.RiverBedTarget[edge.To]
		if m.RiverBedTarget[edge.To]-m.RiverBedTarget[edge.From] > m.Config.FinalUpslopeEpsilon {
			m.TargetUpslopeRiverEdges++
		}
	}

	for i := range m.RiverBedTarget {
		if m.RiverBedTarget[i] < bedBase[i]-1e-12 {
			m.RiverBedCarvedNodes++
			m.MaxRiverBedCarve = math.Max(
				m.MaxRiverBedCarve,
				bedBase[i]-m.RiverBedTarget[i],
			)
		}
	}
	return nil
}

// RebuildRiverConstraintsFromTerrain recomputes the local channel-bed profile
// after a broad terrain correction has been applied. River bed constraints must
// follow the actual broad terrain that the channel will cut into, not a stale
// pre-coast heightfield.
func (m *Model) RebuildRiverConstraintsFromTerrain(applied *terrain.Model, macroElevation []float64, h *hydrology.Model) error {
	if applied == nil || !sameGeometry(applied.Geometry, m.Geometry) {
		return errors.New("reconcile: river base terrain geometry does not match")
	}
	n := m.Geometry.Width * m.Geometry.Height
	bedBase := make([]float64, n)
	for z := 0; z < m.Geometry.Height; z++ {
		for x := 0; x < m.Geometry.Width; x++ {
			i := z*m.Geometry.Width + x
			wx, wz := m.Geometry.NodePosition(x, z)
			bedBase[i] = applied.SampleNormalized(wx, wz)
		}
	}
	return m.buildRiverConstraints(macroElevation, h, bedBase)
}

func (m *Model) EvaluateCoastApplication(applied *terrain.Model, macroSurface *world.Surface) error {
	if applied == nil {
		return errors.New("reconcile: applied coast terrain is required")
	}
	if !sameGeometry(applied.Geometry, m.Geometry) {
		return errors.New("reconcile: applied coast terrain geometry does not match")
	}
	if macroSurface == nil ||
		macroSurface.Width != m.Geometry.Width ||
		macroSurface.Height != m.Geometry.Height ||
		len(macroSurface.LandMask) != len(m.CoastTargetElevation) {
		return errors.New("reconcile: applied coast macro surface does not match")
	}

	n := len(m.CoastTargetElevation)
	m.AppliedCoastElevation = make([]float64, n)
	m.AppliedCoastMaxNodeError = 0
	for z := 0; z < m.Geometry.Height; z++ {
		for x := 0; x < m.Geometry.Width; x++ {
			i := z*m.Geometry.Width + x
			wx, wz := m.Geometry.NodePosition(x, z)
			h := applied.SampleNormalized(wx, wz)
			if !finite(h) || h < 0 || h > 1 {
				return errors.New("reconcile: applied coast terrain sample must be finite and normalized")
			}
			m.AppliedCoastElevation[i] = h
			m.AppliedCoastMaxNodeError = math.Max(
				m.AppliedCoastMaxNodeError,
				math.Abs(h-m.CoastTargetElevation[i]),
			)
		}
	}

	surface, err := world.ClassifySurface(
		m.Geometry.Width,
		m.Geometry.Height,
		m.AppliedCoastElevation,
		m.SeaLevelNormalized,
	)
	if err != nil {
		return fmt.Errorf("reconcile: classify applied coast terrain: %w", err)
	}
	m.AppliedCoastSurface = surface
	m.AppliedCoastSurfaceMismatch = maskMismatchCount(macroSurface.LandMask, surface.LandMask)

	// Lake bed constraints must also be evaluated against the broad terrain that
	// will actually receive the future basin/channel overlays.
	m.LakeBedViolations = 0
	for i, defined := range m.LakeBedCeilingDefined {
		if defined && m.AppliedCoastElevation[i] > m.LakeBedCeiling[i]+1e-12 {
			m.LakeBedViolations++
		}
	}
	return nil
}

func normalizeConfig(cfg Config, seaLevel float64) (Config, error) {
	if cfg.CoastMarginNormalized == 0 {
		cfg.CoastMarginNormalized = 1e-4
	}
	if !finite(cfg.CoastMarginNormalized) || cfg.CoastMarginNormalized <= 0 ||
		cfg.CoastMarginNormalized >= math.Min(seaLevel, 1-seaLevel) {
		return Config{}, errors.New("reconcile: coast margin must be finite, positive and fit inside normalized height range")
	}
	if cfg.RiverStrengthThreshold == 0 {
		// Hydrology's first valid river cells are ~0.227 strength at the
		// accumulation threshold. Keep reconciliation below that so small
		// tributaries are not silently discarded before topology closure.
		cfg.RiverStrengthThreshold = 0.20
	}
	if !finite(cfg.RiverStrengthThreshold) || cfg.RiverStrengthThreshold <= 0 || cfg.RiverStrengthThreshold > 1 {
		return Config{}, errors.New("reconcile: river strength threshold must be finite and inside (0,1]")
	}
	if cfg.OriginalDownhillEpsilon == 0 {
		cfg.OriginalDownhillEpsilon = 1e-6
	}
	if !finite(cfg.OriginalDownhillEpsilon) || cfg.OriginalDownhillEpsilon < 0 {
		return Config{}, errors.New("reconcile: original downhill epsilon must be finite and non-negative")
	}
	if cfg.FinalUpslopeEpsilon == 0 {
		cfg.FinalUpslopeEpsilon = 1e-5
	}
	if !finite(cfg.FinalUpslopeEpsilon) || cfg.FinalUpslopeEpsilon < 0 {
		return Config{}, errors.New("reconcile: final upslope epsilon must be finite and non-negative")
	}
	if cfg.MinRiverBedDropNormalized == 0 {
		cfg.MinRiverBedDropNormalized = 1e-5
	}
	if !finite(cfg.MinRiverBedDropNormalized) || cfg.MinRiverBedDropNormalized <= 0 {
		return Config{}, errors.New("reconcile: minimum river bed drop must be finite and positive")
	}
	if cfg.LakeBedClearanceNormalized == 0 {
		cfg.LakeBedClearanceNormalized = 1e-4
	}
	if !finite(cfg.LakeBedClearanceNormalized) || cfg.LakeBedClearanceNormalized <= 0 {
		return Config{}, errors.New("reconcile: lake bed clearance must be finite and positive")
	}
	return cfg, nil
}

func maskMismatchCount(a, b []bool) int {
	n := 0
	for i := range a {
		if a[i] != b[i] {
			n++
		}
	}
	return n
}

func sameGeometry(a, b world.GridGeometry) bool {
	return a.Width == b.Width &&
		a.Height == b.Height &&
		math.Abs(a.OriginX-b.OriginX) <= 1e-12 &&
		math.Abs(a.OriginZ-b.OriginZ) <= 1e-12 &&
		math.Abs(a.SpanXBlocks-b.SpanXBlocks) <= 1e-9 &&
		math.Abs(a.SpanZBlocks-b.SpanZBlocks) <= 1e-9
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

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
