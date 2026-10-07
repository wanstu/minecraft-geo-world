package reconcile

import (
	"math"
	"testing"

	"geoworld/internal/hydrology"
	"geoworld/internal/sampling"
	"geoworld/internal/terrain"
	"geoworld/internal/world"
)

func TestBuildCoastTargetsPreserveMacroOcean(t *testing.T) {
	g := world.GridGeometry{
		Width: 5, Height: 5,
		SpanXBlocks: 512, SpanZBlocks: 512,
	}
	n := g.Width * g.Height
	macroElevation := make([]float64, n)
	for i := range macroElevation {
		macroElevation[i] = 0.40
	}
	surface, err := world.ClassifySurface(g.Width, g.Height, macroElevation, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	tr := constantTerrain(t, g, 0.80)
	h := emptyHydrology(g, surface.LandMask)

	m, err := Build(Input{
		Geometry: g, SeaLevelNormalized: 0.5,
		MacroSurface: surface, MacroElevation: macroElevation,
		Hydrology: h, Terrain: tr,
	}, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.SurfaceMismatchBefore != n {
		t.Fatalf("raw mismatch=%d want=%d", m.SurfaceMismatchBefore, n)
	}
	if m.SurfaceMismatchAfter != 0 {
		t.Fatalf("constrained mismatch=%d want=0", m.SurfaceMismatchAfter)
	}
	if m.CoastCarvedNodes != n || m.CoastRaisedNodes != 0 {
		t.Fatalf("coast adjustments carved=%d raised=%d want=%d/0",
			m.CoastCarvedNodes, m.CoastRaisedNodes, n)
	}
	for i := range m.CoastTargetElevation {
		if m.CoastTargetElevation[i] >= 0.5 {
			t.Fatalf("ocean target[%d]=%.6f must stay below sea", i, m.CoastTargetElevation[i])
		}
		if m.WaterKind[i] != WaterOcean || !m.WaterSurfaceDefined[i] || m.WaterSurfaceElevation[i] != 0.5 {
			t.Fatalf("ocean water metadata missing at %d", i)
		}
	}
}

func TestBuildCoastTargetsPreserveAboveSeaLand(t *testing.T) {
	g := world.GridGeometry{
		Width: 5, Height: 5,
		SpanXBlocks: 512, SpanZBlocks: 512,
	}
	n := g.Width * g.Height
	macroElevation := make([]float64, n)
	for i := range macroElevation {
		macroElevation[i] = 0.60
	}
	surface, err := world.ClassifySurface(g.Width, g.Height, macroElevation, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	tr := constantTerrain(t, g, 0.40)
	h := emptyHydrology(g, surface.LandMask)

	m, err := Build(Input{
		Geometry: g, SeaLevelNormalized: 0.5,
		MacroSurface: surface, MacroElevation: macroElevation,
		Hydrology: h, Terrain: tr,
	}, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.SurfaceMismatchAfter != 0 {
		t.Fatalf("constrained mismatch=%d want=0", m.SurfaceMismatchAfter)
	}
	if m.CoastRaisedNodes != n || m.CoastCarvedNodes != 0 {
		t.Fatalf("coast adjustments raised=%d carved=%d want=%d/0",
			m.CoastRaisedNodes, m.CoastCarvedNodes, n)
	}
	for i := range m.CoastTargetElevation {
		if m.CoastTargetElevation[i] <= 0.5 {
			t.Fatalf("land target[%d]=%.6f must stay above sea", i, m.CoastTargetElevation[i])
		}
	}
}

func TestRiverBedTargetsBecomeMonotonicWithoutRaisingBed(t *testing.T) {
	raw := []float64{0.60, 0.62, 0.61, 0.64}
	m := &Model{
		Config: Config{
			RiverStrengthThreshold:    0.30,
			OriginalDownhillEpsilon:   1e-6,
			FinalUpslopeEpsilon:       1e-5,
			MinRiverBedDropNormalized: 1e-5,
		},
		RawTerrainElevation: append([]float64(nil), raw...),
		RiverBedTarget:      append([]float64(nil), raw...),
		WaterKind:           make([]WaterKind, len(raw)),
		RiverBedActive:      make([]bool, len(raw)),
	}
	h := &hydrology.Model{
		Width:         4,
		Height:        1,
		LandMask:      []bool{true, true, true, true},
		FlowTo:        []int{1, 2, 3, -1},
		LakeID:        []int{0, 0, 0, 0},
		RiverStrength: []float64{0.8, 0.8, 0.8, 0},
	}
	macro := []float64{0.80, 0.70, 0.60, 0.50}

	if err := m.buildRiverConstraints(macro, h, raw); err != nil {
		t.Fatal(err)
	}
	if m.QualifyingRiverEdges != 3 || m.OriginallyDownhillEdges != 3 {
		t.Fatalf("edge counts qualifying=%d downhill=%d want=3/3",
			m.QualifyingRiverEdges, m.OriginallyDownhillEdges)
	}
	if m.RawUpslopeRiverEdges != 2 {
		t.Fatalf("raw upslope=%d want=2", m.RawUpslopeRiverEdges)
	}
	if m.TargetUpslopeRiverEdges != 0 {
		t.Fatalf("target upslope=%d want=0", m.TargetUpslopeRiverEdges)
	}
	for i := 0; i < len(raw)-1; i++ {
		if m.RiverBedTarget[i+1] > m.RiverBedTarget[i]-m.Config.MinRiverBedDropNormalized+1e-12 {
			t.Fatalf("river bed is not downhill at edge %d->%d: %.8f -> %.8f",
				i, i+1, m.RiverBedTarget[i], m.RiverBedTarget[i+1])
		}
	}
	for i := range raw {
		if m.RiverBedTarget[i] > raw[i]+1e-12 {
			t.Fatalf("river constraint raised bed at %d: raw=%.8f target=%.8f",
				i, raw[i], m.RiverBedTarget[i])
		}
	}
}

func TestRiverConstraintKeepsTerminalEdgeIntoOcean(t *testing.T) {
	raw := []float64{0.58, 0.54, 0.46}
	m := &Model{
		Config: Config{
			RiverStrengthThreshold:    0.30,
			OriginalDownhillEpsilon:   1e-6,
			FinalUpslopeEpsilon:       1e-5,
			MinRiverBedDropNormalized: 1e-5,
		},
		RawTerrainElevation: append([]float64(nil), raw...),
		RiverBedTarget:      append([]float64(nil), raw...),
		WaterKind:           []WaterKind{WaterNone, WaterNone, WaterOcean},
		RiverBedActive:      make([]bool, len(raw)),
	}
	h := &hydrology.Model{
		Width:         3,
		Height:        1,
		LandMask:      []bool{true, true, false},
		FlowTo:        []int{1, 2, -1},
		LakeID:        []int{0, 0, 0},
		RiverStrength: []float64{0.9, 0.9, 0},
	}
	macro := []float64{0.70, 0.58, 0.40}

	if err := m.buildRiverConstraints(macro, h, raw); err != nil {
		t.Fatal(err)
	}
	if len(m.RiverEdges) != 2 {
		t.Fatalf("river edges=%d want=2 including terminal ocean edge", len(m.RiverEdges))
	}
	last := m.RiverEdges[1]
	if last.From != 1 || last.To != 2 {
		t.Fatalf("terminal edge=%d->%d want=1->2", last.From, last.To)
	}
	if m.WaterKind[2] != WaterOcean {
		t.Fatalf("ocean receiver reclassified as %v", m.WaterKind[2])
	}
}

func TestRiverTopologyClosesWeakAndFlatDownstreamToOcean(t *testing.T) {
	raw := []float64{0.64, 0.64, 0.63, 0.49}
	m := &Model{
		Config: Config{
			RiverStrengthThreshold:    0.20,
			OriginalDownhillEpsilon:   1e-6,
			FinalUpslopeEpsilon:       1e-5,
			MinRiverBedDropNormalized: 1e-5,
		},
		RawTerrainElevation:   append([]float64(nil), raw...),
		RiverBedTarget:        append([]float64(nil), raw...),
		WaterKind:             []WaterKind{WaterNone, WaterNone, WaterNone, WaterOcean},
		WaterSurfaceDefined:   []bool{false, false, false, true},
		WaterSurfaceElevation: []float64{0, 0, 0, 0.50},
		RiverBedActive:        make([]bool, len(raw)),
	}
	h := &hydrology.Model{
		Width:    4,
		Height:   1,
		LandMask: []bool{true, true, true, false},
		FlowTo:   []int{1, 2, 3, -1},
		LakeID:   []int{0, 0, 0, 0},
		// Only the headwater qualifies. The downstream cells deliberately fall
		// below threshold; topology must still keep 0->1->2->ocean continuous.
		RiverStrength: []float64{0.35, 0.10, 0.0, 0},
	}
	macro := []float64{0.70, 0.70, 0.62, 0.40}

	if err := m.buildRiverConstraints(macro, h, raw); err != nil {
		t.Fatal(err)
	}
	if len(m.RiverEdges) != 3 {
		t.Fatalf("closed river edges=%d want=3", len(m.RiverEdges))
	}
	want := [][2]int{{0, 1}, {1, 2}, {2, 3}}
	for i, edge := range m.RiverEdges {
		if edge.From != want[i][0] || edge.To != want[i][1] {
			t.Fatalf(
				"edge[%d]=%d->%d want=%d->%d",
				i, edge.From, edge.To, want[i][0], want[i][1],
			)
		}
	}
	if m.WaterKind[1] != WaterRiver || m.WaterKind[2] != WaterRiver {
		t.Fatalf("weak downstream cells were not retained as river: %v", m.WaterKind)
	}
	if m.TargetUpslopeRiverEdges != 0 {
		t.Fatalf("closed river still has %d target upslope edges", m.TargetUpslopeRiverEdges)
	}
}

func TestLakeKeepsVisibleFeederAndSpillOutflow(t *testing.T) {
	raw := []float64{0.66, 0.64, 0.60, 0.57, 0.46}
	m := &Model{
		Config: Config{
			RiverStrengthThreshold:    0.30,
			OriginalDownhillEpsilon:   1e-6,
			FinalUpslopeEpsilon:       1e-5,
			MinRiverBedDropNormalized: 1e-5,
		},
		RawTerrainElevation:   append([]float64(nil), raw...),
		RiverBedTarget:        append([]float64(nil), raw...),
		WaterKind:             []WaterKind{WaterNone, WaterNone, WaterLake, WaterNone, WaterOcean},
		WaterSurfaceDefined:   []bool{false, false, true, false, true},
		WaterSurfaceElevation: []float64{0, 0, 0.61, 0, 0.50},
		RiverBedActive:        make([]bool, len(raw)),
	}
	h := &hydrology.Model{
		Width:            5,
		Height:           1,
		LandMask:         []bool{true, true, true, true, false},
		FlowTo:           []int{1, 2, 3, 4, -1},
		LakeID:           []int{0, 0, 1, 0, 0},
		RiverStrength:    []float64{0.05, 0.08, 0, 0, 0},
		FlowAccumulation: []float64{20, 40, 80, 90, 100},
		Lakes: []hydrology.Lake{{
			ID:          1,
			SpillOutlet: 2,
		}},
	}
	macro := []float64{0.72, 0.68, 0.61, 0.56, 0.44}

	if err := m.buildRiverConstraints(macro, h, raw); err != nil {
		t.Fatal(err)
	}

	want := map[[2]int]bool{
		{0, 1}: true,
		{1, 2}: true,
		{2, 3}: true,
		{3, 4}: true,
	}
	if len(m.RiverEdges) != len(want) {
		t.Fatalf(
			"lake drainage edges=%d want=%d: %+v",
			len(m.RiverEdges),
			len(want),
			m.RiverEdges,
		)
	}
	for _, edge := range m.RiverEdges {
		key := [2]int{edge.From, edge.To}
		if !want[key] {
			t.Fatalf("unexpected lake drainage edge %d->%d", edge.From, edge.To)
		}
		if edge.RiverStrength < m.Config.RiverStrengthThreshold {
			t.Fatalf(
				"forced lake drainage edge %d->%d strength=%.3f below visible threshold %.3f",
				edge.From,
				edge.To,
				edge.RiverStrength,
				m.Config.RiverStrengthThreshold,
			)
		}
	}
	if m.WaterKind[2] != WaterLake {
		t.Fatalf("lake outlet node was reclassified: %v", m.WaterKind[2])
	}
	if m.WaterKind[4] != WaterOcean {
		t.Fatalf("ocean receiver was reclassified: %v", m.WaterKind[4])
	}
}

func TestLakeConstraintExposesWaterSurfaceAndBedCeiling(t *testing.T) {
	m := &Model{
		Config:                Config{LakeBedClearanceNormalized: 0.01},
		RawTerrainElevation:   []float64{0.70, 0.40},
		WaterKind:             []WaterKind{WaterNone, WaterNone},
		WaterSurfaceDefined:   make([]bool, 2),
		WaterSurfaceElevation: make([]float64, 2),
		LakeBedCeilingDefined: make([]bool, 2),
		LakeBedCeiling:        make([]float64, 2),
	}
	h := &hydrology.Model{
		LakeID: []int{1, 0},
		Lakes:  []hydrology.Lake{{ID: 1, SurfaceElevation: 0.60}},
	}
	if err := m.buildLakeConstraints(h); err != nil {
		t.Fatal(err)
	}
	if m.WaterKind[0] != WaterLake || !m.WaterSurfaceDefined[0] || m.WaterSurfaceElevation[0] != 0.60 {
		t.Fatalf("lake water metadata not set: kind=%v defined=%v level=%.4f",
			m.WaterKind[0], m.WaterSurfaceDefined[0], m.WaterSurfaceElevation[0])
	}
	if !m.LakeBedCeilingDefined[0] || math.Abs(m.LakeBedCeiling[0]-0.59) > 1e-12 {
		t.Fatalf("lake bed ceiling=%.12f want=0.59", m.LakeBedCeiling[0])
	}
	if m.LakeBedViolations != 1 {
		t.Fatalf("lake bed violations=%d want=1", m.LakeBedViolations)
	}
}

func constantTerrain(t *testing.T, g world.GridGeometry, height float64) *terrain.Model {
	t.Helper()
	n := g.Width * g.Height
	baseValues := make([]float64, n)
	zeroValues := make([]float64, n)
	for i := range baseValues {
		baseValues[i] = height
	}
	base, err := sampling.NewScalarGridWithGeometry(g, baseValues)
	if err != nil {
		t.Fatal(err)
	}
	zero, err := sampling.NewScalarGridWithGeometry(g, zeroValues)
	if err != nil {
		t.Fatal(err)
	}
	return &terrain.Model{
		Geometry:            g,
		BaseElevation:       base,
		RollingPlainWeight:  zero,
		HillWeight:          zero,
		FoothillWeight:      zero,
		MountainWeight:      zero,
		PlateauWeight:       zero,
		ValleyWeight:        zero,
		FloodplainWeight:    zero,
		AlluvialPlainWeight: zero,
		AlluvialFanWeight:   zero,
		CanyonWeight:        zero,
		ValleyFloorWidth:    zero,
		DepositionPotential: zero,
	}
}

func emptyHydrology(g world.GridGeometry, landMask []bool) *hydrology.Model {
	n := g.Width * g.Height
	flowTo := make([]int, n)
	for i := range flowTo {
		flowTo[i] = -1
	}
	return &hydrology.Model{
		Geometry: g, Width: g.Width, Height: g.Height,
		LandMask:      append([]bool(nil), landMask...),
		FlowTo:        flowTo,
		LakeID:        make([]int, n),
		RiverStrength: make([]float64, n),
	}
}
