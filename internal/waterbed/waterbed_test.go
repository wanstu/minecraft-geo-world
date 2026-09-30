package waterbed

import (
	"math"
	"testing"

	"geoworld/internal/hydrology"
	"geoworld/internal/reconcile"
	"geoworld/internal/terrain"
	"geoworld/internal/world"
)

func TestRiverCorridorHitsBedTargetAndStaysLocal(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	from, to := 3, 4
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.70),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: from, To: to, RiverStrength: 1,
		}},
	}
	rec.RiverBedTarget[from] = 0.65
	rec.RiverBedTarget[to] = 0.60
	h := emptyHydrology(g)

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rivers) != 1 {
		t.Fatalf("river segments=%d want=1", len(m.Rivers))
	}

	x0, z0 := g.NodePosition(0, 1)
	x1, _ := g.NodePosition(1, 1)
	midX := (x0 + x1) * 0.5
	got := m.Sample(midX, z0)
	if math.Abs(got.FinalHeight-0.625) > 1e-12 {
		t.Fatalf("centerline height=%.12f want=0.625", got.FinalHeight)
	}
	if got.RiverCarve <= 0 || got.WaterKind != reconcile.WaterRiver {
		t.Fatalf("missing river carve metadata: %+v", got)
	}

	far := m.Sample(midX, z0+40)
	if math.Abs(far.FinalHeight-0.70) > 1e-12 || far.RiverCarve != 0 {
		t.Fatalf("river carve leaked outside corridor: %+v", far)
	}
}

func TestRiverCorridorNeverRaisesBroadTerrain(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.60)
	n := g.Width * g.Height
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.60),
		RiverBedTarget:        filled(n, 0.80),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: 3, To: 4, RiverStrength: 1,
		}},
	}
	h := emptyHydrology(g)

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	x0, z0 := g.NodePosition(0, 1)
	x1, _ := g.NodePosition(1, 1)
	got := m.Sample((x0+x1)*0.5, z0)
	if math.Abs(got.FinalHeight-0.60) > 1e-12 || got.RiverCarve != 0 {
		t.Fatalf("river overlay raised or carved invalid bed: %+v", got)
	}
}

func TestRiverBankBlendIsContinuous(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.60),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: 3, To: 4, RiverStrength: 1,
		}},
	}
	m, err := Build(base, rec, emptyHydrology(g), Config{})
	if err != nil {
		t.Fatal(err)
	}
	seg := m.Rivers[0]
	x := (seg.X0 + seg.X1) * 0.5
	z := seg.Z0 + seg.HalfWidth
	eps := 1e-4
	at := m.SampleNormalized(x, z)
	insideSlope := (at - m.SampleNormalized(x, z-eps)) / eps
	outsideSlope := (m.SampleNormalized(x, z+eps) - at) / eps
	if math.Abs(at-0.70) > 1e-10 {
		t.Fatalf("bank edge height=%.12f want broad terrain 0.70", at)
	}
	if math.Abs(insideSlope-outsideSlope) > 1e-5 {
		t.Fatalf("bank slope kink: inside=%.12g outside=%.12g", insideSlope, outsideSlope)
	}
}

func TestLakeBasinCarvesToCeilingAndExposesWaterSurface(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	center := 4
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.70),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
	}
	rec.LakeBedCeilingDefined[center] = true
	rec.LakeBedCeiling[center] = 0.55
	rec.WaterSurfaceDefined[center] = true
	rec.WaterSurfaceElevation[center] = 0.60
	h := emptyHydrology(g)
	h.LakeID[center] = 1

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	x, z := g.NodePosition(1, 1)
	got := m.Sample(x, z)
	if math.Abs(got.FinalHeight-0.55) > 1e-12 {
		t.Fatalf("lake center bed=%.12f want=0.55", got.FinalHeight)
	}
	if got.WaterKind != reconcile.WaterLake || !got.WaterSurfaceDefined ||
		math.Abs(got.WaterSurfaceElevation-0.60) > 1e-12 {
		t.Fatalf("lake water metadata wrong: %+v", got)
	}

	far := m.Sample(x+100, z)
	if math.Abs(far.FinalHeight-0.70) > 1e-12 || far.LakeCarve != 0 {
		t.Fatalf("lake carve leaked outside local footprint: %+v", far)
	}
}

func testGeometry() world.GridGeometry {
	return world.GridGeometry{
		Width: 3, Height: 3,
		SpanXBlocks: 256, SpanZBlocks: 256,
	}
}

func testTerrain(t *testing.T, g world.GridGeometry, height float64) *terrain.Model {
	t.Helper()
	n := g.Width * g.Height
	zeros := make([]float64, n)
	in := terrain.Input{
		Width: g.Width, Height: g.Height, Geometry: g,
		BaseElevation:       filled(n, height),
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

func emptyHydrology(g world.GridGeometry) *hydrology.Model {
	n := g.Width * g.Height
	flow := make([]int, n)
	for i := range flow {
		flow[i] = -1
	}
	return &hydrology.Model{
		Geometry:      g,
		Width:         g.Width,
		Height:        g.Height,
		LandMask:      make([]bool, n),
		FlowTo:        flow,
		LakeID:        make([]int, n),
		RiverStrength: make([]float64, n),
	}
}

func filled(n int, value float64) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = value
	}
	return out
}

func TestConfluencePropagatesActualUpstreamOutlet(t *testing.T) {
	g := world.GridGeometry{
		Width: 4, Height: 3,
		SpanXBlocks: 384, SpanZBlocks: 256,
	}
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	from, junction, downstream := 4, 5, 6

	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.70),
		RiverBedTarget:        filled(n, 0.70),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{
			{From: from, To: junction, RiverStrength: 1},
			{From: junction, To: downstream, RiverStrength: 1},
		},
	}
	// The incoming segment's actual outlet will clamp to 0.55 even though the
	// junction macro target is 0.60. The downstream segment must inherit 0.55,
	// otherwise the overlapping incoming corridor creates a dip then a rise.
	rec.RiverBedTarget[from] = 0.55
	rec.RiverBedTarget[junction] = 0.60
	rec.RiverBedTarget[downstream] = 0.50

	h := emptyHydrology(g)
	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rivers) != 2 {
		t.Fatalf("river segments=%d want=2", len(m.Rivers))
	}

	var downstreamSegment *RiverSegment
	for i := range m.Rivers {
		if m.Rivers[i].From == junction && m.Rivers[i].To == downstream {
			downstreamSegment = &m.Rivers[i]
			break
		}
	}
	if downstreamSegment == nil {
		t.Fatal("downstream segment not found")
	}
	if math.Abs(downstreamSegment.Bed0-0.55) > 1e-12 {
		t.Fatalf("downstream bed start=%.12f want propagated 0.55", downstreamSegment.Bed0)
	}

	d := m.Audit(32, 1e-9)
	if d.RiverUpslopeSteps != 0 {
		t.Fatalf("confluence introduced %d upslope steps, max rise %.12g", d.RiverUpslopeSteps, d.MaxRiverRise)
	}
}
