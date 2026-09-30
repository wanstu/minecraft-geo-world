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

	if err := m.buildRiverConstraints(macro, h); err != nil {
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
