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
	base := testTerrain(t, g, 0.60)
	rec := baseReconciliation(g, 0.60)
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
	wantSurface := wb.Lakes[0].WaterSurface
	if math.Abs(got.GroundHeight-0.55) > 1e-12 ||
		math.Abs(got.WaterSurfaceElevation-wantSurface) > 1e-12 ||
		math.Abs(got.WaterDepth-(wantSurface-0.55)) > 1e-12 {
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
	wb, err := waterbed.Build(
		base,
		rec,
		h,
		waterbed.Config{RiverCenterDepthNormalized: 0.01},
	)
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
	widthT := (wb.Rivers[0].HalfWidth - wb.Config.MinRiverHalfWidthBlocks) /
		(wb.Config.MaxRiverHalfWidthBlocks - wb.Config.MinRiverHalfWidthBlocks)
	expectedCenterDepth := 0.01 * (1 + 2*math.Pow(widthT, 1.15))
	if math.Abs(center.RiverWeight-1) > 1e-12 ||
		math.Abs(center.WaterDepth-expectedCenterDepth) > 1e-12 {
		t.Fatalf(
			"river center depth wrong: got=%.6f want=%.6f sample=%+v",
			center.WaterDepth, expectedCenterDepth, center,
		)
	}

	nearBank := m.Sample(midX, z0+6)
	if nearBank.WaterKind != reconcile.WaterRiver || !nearBank.WaterSurfaceDefined {
		t.Fatalf("expected river water inside bank, got %+v", nearBank)
	}
	if !(nearBank.WaterDepth > 0 && nearBank.WaterDepth < center.WaterDepth) {
		t.Fatalf("river depth did not taper: center=%.6f bank=%.6f", center.WaterDepth, nearBank.WaterDepth)
	}
	if math.Abs(nearBank.WaterSurfaceElevation-center.WaterSurfaceElevation) > 1e-12 {
		t.Fatalf(
			"river cross-section water plane is not flat: center=%.6f bank=%.6f",
			center.WaterSurfaceElevation, nearBank.WaterSurfaceElevation,
		)
	}

	shallowFringe := m.Sample(midX, z0+17)
	if shallowFringe.WaterKind != reconcile.WaterNone || shallowFringe.WaterSurfaceDefined {
		t.Fatalf("sub-block river fringe should be dry bank, got %+v", shallowFringe)
	}

	outside := m.Sample(midX, z0+20)
	if outside.WaterKind != reconcile.WaterNone || outside.WaterSurfaceDefined {
		t.Fatalf("river water leaked outside corridor: %+v", outside)
	}
}

func TestConfluenceWaterSurfaceDoesNotRiseWhenMainStemWidens(t *testing.T) {
	g := world.GridGeometry{
		Width: 4, Height: 3,
		SpanXBlocks: 384, SpanZBlocks: 256,
	}
	base := testTerrain(t, g, 0.72)
	rec := baseReconciliation(g, 0.72)
	from, junction, downstream := g.Width, g.Width+1, g.Width+2
	rec.RiverBedTarget[from] = 0.63
	rec.RiverBedTarget[junction] = 0.62
	rec.RiverBedTarget[downstream] = 0.60
	rec.RiverEdges = []reconcile.RiverEdgeConstraint{
		{From: from, To: junction, RiverStrength: 0.31},
		{From: junction, To: downstream, RiverStrength: 1.0},
	}

	h := emptyHydrology(g)
	const centerDepth = 0.01
	wb, err := waterbed.Build(
		base,
		rec,
		h,
		waterbed.Config{RiverCenterDepthNormalized: centerDepth},
	)
	if err != nil {
		t.Fatal(err)
	}
	m, err := Build(
		testSurface(g, true),
		wb,
		Config{RiverCenterDepthNormalized: centerDepth},
	)
	if err != nil {
		t.Fatal(err)
	}

	x0, z := g.NodePosition(0, 1)
	x2, _ := g.NodePosition(2, 1)
	step := 2.0
	havePrev := false
	prev := 0.0
	for x := x0; x <= x2; x += step {
		s := m.Sample(x, z)
		if s.WaterKind != reconcile.WaterRiver || !s.WaterSurfaceDefined {
			continue
		}
		if havePrev && s.WaterSurfaceElevation > prev+1e-9 {
			t.Fatalf(
				"river surface rose through widening confluence at x=%.1f: %.9f -> %.9f",
				x, prev, s.WaterSurfaceElevation,
			)
		}
		prev = s.WaterSurfaceElevation
		havePrev = true
	}
	if !havePrev {
		t.Fatal("did not sample river water")
	}
}

func TestReceiverBackwaterFormsFlatFloorWithoutDownstreamRise(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.55)
	rec := baseReconciliation(g, 0.55)
	rec.WaterKind = make([]reconcile.WaterKind, g.Width*g.Height)
	from, to := g.Width, g.Width+1
	rec.RiverBedTarget[from] = 0.44
	rec.RiverBedTarget[to] = 0.43
	rec.RiverEdges = []reconcile.RiverEdgeConstraint{{
		From: from, To: to, RiverStrength: 1,
	}}
	rec.WaterKind[to] = reconcile.WaterOcean
	rec.WaterSurfaceDefined[to] = true
	rec.WaterSurfaceElevation[to] = 0.50

	h := emptyHydrology(g)
	h.FlowTo[from] = to
	wb, err := waterbed.Build(
		base,
		rec,
		h,
		waterbed.Config{RiverCenterDepthNormalized: 0.01},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !wb.Rivers[0].ReceiverSurfaceDefined {
		t.Fatal("terminal segment did not receive direct ocean datum")
	}

	m, err := Build(
		testSurface(g, true),
		wb,
		Config{RiverCenterDepthNormalized: 0.01},
	)
	if err != nil {
		t.Fatal(err)
	}

	x0, z0 := g.NodePosition(0, 1)
	x1, _ := g.NodePosition(1, 1)
	prev := math.Inf(1)
	for _, frac := range []float64{0.10, 0.25, 0.50, 0.75, 0.95} {
		s := m.Sample(x0+(x1-x0)*frac, z0)
		if s.WaterKind != reconcile.WaterRiver || !s.WaterSurfaceDefined {
			t.Fatalf("expected river backwater at t=%.2f, got %+v", frac, s)
		}
		if math.Abs(s.WaterSurfaceElevation-0.50) > 1e-12 {
			t.Fatalf(
				"receiver backwater is not flat at t=%.2f: got=%.12f want=0.50",
				frac,
				s.WaterSurfaceElevation,
			)
		}
		if s.WaterSurfaceElevation > prev+1e-12 {
			t.Fatalf(
				"receiver backwater rose downstream at t=%.2f: %.12f -> %.12f",
				frac,
				prev,
				s.WaterSurfaceElevation,
			)
		}
		prev = s.WaterSurfaceElevation
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
