package worldsample

import (
	"math"
	"testing"

	"geoworld/internal/hydrology"
	"geoworld/internal/reconcile"
	"geoworld/internal/terrain"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

func TestOceanUsesSharedSeaLevel(t *testing.T) {
	g := testGeometry()
	wb := emptyWaterbed(t, g, 0.40)
	surface := testSurface(g, false)

	m, err := Build(surface, wb, Config{})
	if err != nil {
		t.Fatal(err)
	}
	x, z := g.NodePosition(1, 1)
	got := m.Sample(x, z)
	if got.WaterKind != reconcile.WaterOcean || !got.WaterSurfaceDefined {
		t.Fatalf("expected ocean water, got %+v", got)
	}
	if math.Abs(got.WaterSurfaceElevation-0.5) > 1e-12 ||
		math.Abs(got.WaterDepth-0.1) > 1e-12 {
		t.Fatalf("ocean datum/depth wrong: %+v", got)
	}
}

func TestEnclosedBelowSeaLandStaysDryWithoutLake(t *testing.T) {
	g := testGeometry()
	wb := emptyWaterbed(t, g, 0.40)
	surface := testSurface(g, true)
	center := g.Width + 1
	surface.EnclosedBelowSea[center] = true

	m, err := Build(surface, wb, Config{})
	if err != nil {
		t.Fatal(err)
	}
	x, z := g.NodePosition(1, 1)
	got := m.Sample(x, z)
	if got.WaterKind != reconcile.WaterNone || got.WaterSurfaceDefined {
		t.Fatalf("enclosed lowland became water without hydrology: %+v", got)
	}
	if got.OceanEligibility != 0 {
		t.Fatalf("enclosed lowland ocean eligibility=%.12f want=0", got.OceanEligibility)
	}
}

func TestExplicitLakeUsesHydrologyWaterDatum(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.70)
	rec := baseReconciliation(g, 0.70)
	center := g.Width + 1
	rec.LakeBedCeilingDefined[center] = true
	rec.LakeBedCeiling[center] = 0.55
	rec.WaterSurfaceDefined[center] = true
	rec.WaterSurfaceElevation[center] = 0.60
	h := emptyHydrology(g)
	h.LakeID[center] = 1
	wb, err := waterbed.Build(base, rec, h, waterbed.Config{})
	if err != nil {
		t.Fatal(err)
	}

	m, err := Build(testSurface(g, true), wb, Config{})
	if err != nil {
		t.Fatal(err)
	}
	x, z := g.NodePosition(1, 1)
	got := m.Sample(x, z)
	if got.WaterKind != reconcile.WaterLake || !got.WaterSurfaceDefined {
		t.Fatalf("expected explicit lake, got %+v", got)
	}
	if math.Abs(got.GroundHeight-0.55) > 1e-12 ||
		math.Abs(got.WaterSurfaceElevation-0.60) > 1e-12 ||
		math.Abs(got.WaterDepth-0.05) > 1e-12 {
		t.Fatalf("lake ground/surface/depth wrong: %+v", got)
	}
}

func TestRiverDepthTapersSmoothlyToBank(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.70)
	rec := baseReconciliation(g, 0.70)
	from, to := g.Width, g.Width+1
	rec.RiverBedTarget[from] = 0.65
	rec.RiverBedTarget[to] = 0.60
	rec.RiverEdges = []reconcile.RiverEdgeConstraint{{
		From: from, To: to, RiverStrength: 1,
	}}
	h := emptyHydrology(g)
	wb, err := waterbed.Build(base, rec, h, waterbed.Config{})
	if err != nil {
		t.Fatal(err)
	}

	m, err := Build(testSurface(g, true), wb, Config{RiverCenterDepthNormalized: 0.01})
	if err != nil {
		t.Fatal(err)
	}

	x0, z0 := g.NodePosition(0, 1)
	x1, _ := g.NodePosition(1, 1)
	midX := (x0 + x1) * 0.5

	center := m.Sample(midX, z0)
	if center.WaterKind != reconcile.WaterRiver || !center.WaterSurfaceDefined {
		t.Fatalf("expected river center water, got %+v", center)
	}
	if math.Abs(center.RiverWeight-1) > 1e-12 ||
		math.Abs(center.WaterDepth-0.01) > 1e-12 {
		t.Fatalf("river center depth wrong: %+v", center)
	}

	nearBank := m.Sample(midX, z0+17)
	if nearBank.WaterKind != reconcile.WaterRiver || !nearBank.WaterSurfaceDefined {
		t.Fatalf("expected tapered river water near bank, got %+v", nearBank)
	}
	if !(nearBank.WaterDepth > 0 && nearBank.WaterDepth < center.WaterDepth) {
		t.Fatalf("river depth did not taper: center=%.6f bank=%.6f", center.WaterDepth, nearBank.WaterDepth)
	}

	outside := m.Sample(midX, z0+20)
	if outside.WaterKind != reconcile.WaterNone || outside.WaterSurfaceDefined {
		t.Fatalf("river water leaked outside corridor: %+v", outside)
	}
}

func testGeometry() world.GridGeometry {
	return world.GridGeometry{
		Width: 3, Height: 3,
		SpanXBlocks: 256, SpanZBlocks: 256,
	}
}

func testSurface(g world.GridGeometry, land bool) *world.Surface {
	n := g.Width * g.Height
	s := &world.Surface{
		Width: g.Width, Height: g.Height,
		SeaLevelNormalized: 0.5,
		LandMask:           make([]bool, n),
		EnclosedBelowSea:   make([]bool, n),
	}
	for i := range s.LandMask {
		s.LandMask[i] = land
	}
	return s
}

func testTerrain(t *testing.T, g world.GridGeometry, height float64) *terrain.Model {
	t.Helper()
	n := g.Width * g.Height
	zeros := make([]float64, n)
	base := make([]float64, n)
	for i := range base {
		base[i] = height
	}
	in := terrain.Input{
		Width: g.Width, Height: g.Height, Geometry: g,
		BaseElevation:       base,
		RollingPlainWeight:  zeros,
		HillWeight:          zeros,
		FoothillWeight:      zeros,
		MountainWeight:      zeros,
		PlateauWeight:       zeros,
		ValleyWeight:        zeros,
		FloodplainWeight:    zeros,
		AlluvialPlainWeight: zeros,
		AlluvialFanWeight:   zeros,
		CanyonWeight:        zeros,
		ValleyFloorWidth:    zeros,
		DepositionPotential: zeros,
	}
	m, err := terrain.Generate(in, terrain.Config{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func baseReconciliation(g world.GridGeometry, height float64) *reconcile.Model {
	n := g.Width * g.Height
	return &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, height),
		RiverBedTarget:        filled(n, height),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
	}
}

func emptyHydrology(g world.GridGeometry) *hydrology.Model {
	n := g.Width * g.Height
	flow := make([]int, n)
	for i := range flow {
		flow[i] = -1
	}
	return &hydrology.Model{
		Geometry: g, Width: g.Width, Height: g.Height,
		LandMask:      make([]bool, n),
		FlowTo:        flow,
		LakeID:        make([]int, n),
		RiverStrength: make([]float64, n),
	}
}

func emptyWaterbed(t *testing.T, g world.GridGeometry, height float64) *waterbed.Model {
	t.Helper()
	base := testTerrain(t, g, height)
	rec := baseReconciliation(g, height)
	wb, err := waterbed.Build(base, rec, emptyHydrology(g), waterbed.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return wb
}

func filled(n int, value float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = value
	}
	return out
}
