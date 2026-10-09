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
	want := 0.60 - m.Config.MinRiverBedDepthNormalized*
		(1+1.5*riverWidthFraction(m.Config, m.Rivers[0].HalfWidth))
	if got.FinalHeight > want+1e-12 ||
		got.FinalHeight < want-navigationSurfaceStepNormalized-1e-12 ||
		got.RiverCarve <= 0 {
		t.Fatalf(
			"river centerline outside allowed navigable carve: got=%+v baseTarget=%.6f step=%.6f",
			got,
			want,
			navigationSurfaceStepNormalized,
		)
	}
	if got.FinalHeight > 0.60+1e-12 {
		t.Fatalf("river overlay raised broad terrain: %+v", got)
	}
}

func TestNavigableRiverProfileSnapsDownWithoutRaising(t *testing.T) {
	cfg, err := normalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	in := []float64{0.612, 0.610, 0.608, 0.607}
	out := snapNavigableRiverProfile(
		in,
		cfg,
		12,
		navigationSurfaceStepNormalized,
	)
	if len(out) != len(in) {
		t.Fatalf("snapped profile length=%d want=%d", len(out), len(in))
	}
	for i := range out {
		if out[i] > in[i]+1e-12 {
			t.Fatalf("navigation snap raised bed[%d]: %.12f -> %.12f", i, in[i], out[i])
		}
		if in[i]-out[i] > navigationSurfaceStepNormalized+1e-12 {
			t.Fatalf(
				"navigation snap carved too deeply at %d: %.12f -> %.12f step=%.6f",
				i,
				in[i],
				out[i],
				navigationSurfaceStepNormalized,
			)
		}
		if i > 0 && out[i] > out[i-1]+1e-12 {
			t.Fatalf("snapped river profile rose at %d: %.12f -> %.12f", i, out[i-1], out[i])
		}
	}
	want := out[len(out)-1]
	for i := range out {
		if math.Abs(out[i]-want) > 1e-12 {
			t.Fatalf("gentle navigable profile is not flat after snap: %v", out)
		}
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
	eps := 1e-4

	innerZ := seg.Z0 + seg.HalfWidth
	inner := m.SampleNormalized(x, innerZ)
	insideSlope :=
		(inner - m.SampleNormalized(x, innerZ-eps)) / eps
	outsideSlope :=
		(m.SampleNormalized(x, innerZ+eps) - inner) / eps
	if inner >= 0.70 || inner <= sampleRiverProfile(seg, 0.5) {
		t.Fatalf(
			"inner bank should sit between river bed and broad terrain: %.12f",
			inner,
		)
	}
	if math.Abs(insideSlope-outsideSlope) > 2e-4 {
		t.Fatalf(
			"inner bank slope kink: inside=%.12g outside=%.12g",
			insideSlope,
			outsideSlope,
		)
	}

	outerZ := innerZ + riverBankBlendBlocks(seg.HalfWidth)
	outer := m.SampleNormalized(x, outerZ)
	outerInsideSlope :=
		(outer - m.SampleNormalized(x, outerZ-eps)) / eps
	outerOutsideSlope :=
		(m.SampleNormalized(x, outerZ+eps) - outer) / eps
	if math.Abs(outer-0.70) > 1e-10 {
		t.Fatalf("outer bank height=%.12f want broad terrain 0.70", outer)
	}
	if math.Abs(outerInsideSlope-outerOutsideSlope) > 2e-4 {
		t.Fatalf(
			"outer bank slope kink: inside=%.12g outside=%.12g",
			outerInsideSlope,
			outerOutsideSlope,
		)
	}
}

func TestLakeBasinCarvesToCeilingAndExposesWaterSurface(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.60)
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
	if got.WaterKind != reconcile.WaterLake || !got.WaterSurfaceDefined {
		t.Fatalf("lake water metadata wrong: %+v", got)
	}
	if math.Abs(got.WaterSurfaceElevation-m.Lakes[0].WaterSurface) > 1e-12 {
		t.Fatalf(
			"lake sample surface=%.12f model datum=%.12f",
			got.WaterSurfaceElevation, m.Lakes[0].WaterSurface,
		)
	}

	far := m.Sample(x+100, z)
	if math.Abs(far.FinalHeight-0.60) > 1e-12 || far.LakeCarve != 0 {
		t.Fatalf("lake carve leaked outside local footprint: %+v", far)
	}
}

func TestLakeDoesNotExcavateHighDetailedTerrain(t *testing.T) {
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
		WaterKind:             make([]reconcile.WaterKind, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
	}
	rec.LakeBedCeilingDefined[center] = true
	rec.LakeBedCeiling[center] = 0.55
	rec.WaterKind[center] = reconcile.WaterLake
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
	if math.Abs(got.FinalHeight-0.70) > 1e-12 || got.LakeCarve != 0 {
		t.Fatalf("high detailed terrain was excavated into lake: %+v", got)
	}
	if got.WaterKind == reconcile.WaterLake || got.WaterSurfaceDefined {
		t.Fatalf("lake flooded terrain above its spill datum: %+v", got)
	}
}

func TestLakeSurfaceCapsAtDetailedSpillRim(t *testing.T) {
	g := testGeometry()
	// The macro lake claims a 0.70 datum but detailed terrain around its
	// rendered footprint is only 0.55. A physically valid lake must spill/lower
	// instead of standing as a raised wall above the surrounding land.
	base := testTerrain(t, g, 0.55)
	n := g.Width * g.Height
	center := 4
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.20},
		RiverBaseElevation:    filled(n, 0.55),
		RiverBedTarget:        filled(n, 0.55),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterKind:             make([]reconcile.WaterKind, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
	}
	rec.LakeBedCeilingDefined[center] = true
	rec.LakeBedCeiling[center] = 0.65
	rec.WaterKind[center] = reconcile.WaterLake
	rec.WaterSurfaceDefined[center] = true
	rec.WaterSurfaceElevation[center] = 0.70

	h := emptyHydrology(g)
	h.LakeID[center] = 1
	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Lakes) != 1 {
		t.Fatalf("lake nodes=%d want=1", len(m.Lakes))
	}
	lake := m.Lakes[0]
	if lake.WaterSurface >= 0.55 {
		t.Fatalf(
			"invalid raised lake survived detailed spill rim: surface=%.6f rim=0.55",
			lake.WaterSurface,
		)
	}
	if lake.BedCeiling >= lake.WaterSurface {
		t.Fatalf(
			"lake bed not kept below capped surface: bed=%.6f surface=%.6f",
			lake.BedCeiling, lake.WaterSurface,
		)
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
	if downstreamSegment.Bed0 > 0.55+1e-12 ||
		downstreamSegment.Bed0 < 0.55-navigationSurfaceStepNormalized-1e-12 {
		t.Fatalf(
			"downstream bed start=%.12f outside propagated/navigation range [%.12f, %.12f]",
			downstreamSegment.Bed0,
			0.55-navigationSurfaceStepNormalized,
			0.55,
		)
	}

	d := m.Audit(32, 1e-9)
	if d.RiverUpslopeSteps != 0 {
		t.Fatalf("confluence introduced %d upslope steps, max rise %.12g", d.RiverUpslopeSteps, d.MaxRiverRise)
	}
}

func TestFinalConfluencePassKeepsOverlappingTributarySurfacesEqual(t *testing.T) {
	g := world.GridGeometry{
		Width: 4, Height: 3,
		SpanXBlocks: 384, SpanZBlocks: 256,
	}
	base := testTerrain(t, g, 0.70)
	n := g.Width * g.Height
	fromWest, fromNorth, junction, downstream := 4, 1, 5, 6

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
			{From: fromWest, To: junction, RiverStrength: 0.55},
			{From: fromNorth, To: junction, RiverStrength: 1.00},
			{From: junction, To: downstream, RiverStrength: 1.00},
		},
	}
	rec.RiverBedTarget[fromWest] = 0.58
	rec.RiverBedTarget[fromNorth] = 0.54
	rec.RiverBedTarget[junction] = 0.54
	rec.RiverBedTarget[downstream] = 0.50

	h := emptyHydrology(g)
	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}

	var incoming []*RiverSegment
	for i := range m.Rivers {
		if m.Rivers[i].To == junction {
			incoming = append(incoming, &m.Rivers[i])
		}
	}
	if len(incoming) != 2 {
		t.Fatalf("incoming segments=%d want=2", len(incoming))
	}

	// By the final 5% of both tributaries their channel footprints overlap
	// around the shared junction. Any later cascade/receiver adjustment must
	// therefore leave them on one free-water datum; otherwise Minecraft ceil
	// quantization can turn a sub-block seam into an isolated 1-block source
	// peak.
	const sampleT = 0.95
	surfaces := make([]float64, 0, len(incoming))
	for _, seg := range incoming {
		bed := sampleRiverProfile(*seg, sampleT)
		surface := bed + riverCenterDepth(m.Config, seg.HalfWidth)
		if seg.ReceiverSurfaceDefined && seg.ReceiverSurface > surface {
			surface = seg.ReceiverSurface
		}
		surfaces = append(surfaces, surface)
	}
	if math.Abs(surfaces[0]-surfaces[1]) > 1e-9 {
		t.Fatalf(
			"overlapping tributary surfaces diverged after final passes: %.12f vs %.12f",
			surfaces[0],
			surfaces[1],
		)
	}
}

func TestTerminalRiverSegmentInheritsOceanSurfaceAtItsEndpoint(t *testing.T) {
	g := testGeometry()
	base := testTerrain(t, g, 0.55)
	n := g.Width * g.Height
	from, ocean := 3, 4
	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.55),
		RiverBedTarget:        filled(n, 0.55),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterKind:             make([]reconcile.WaterKind, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: from, To: ocean, RiverStrength: 1,
		}},
	}
	rec.RiverBedTarget[from] = 0.53
	rec.RiverBedTarget[ocean] = 0.47
	rec.WaterKind[ocean] = reconcile.WaterOcean
	rec.WaterSurfaceDefined[ocean] = true
	rec.WaterSurfaceElevation[ocean] = 0.50

	h := emptyHydrology(g)
	h.LandMask[from] = true
	h.FlowTo[from] = ocean

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rivers) != 1 {
		t.Fatalf("river segments=%d want=1", len(m.Rivers))
	}
	seg := m.Rivers[0]
	if !seg.ReceiverSurfaceDefined || math.Abs(seg.ReceiverSurface-0.50) > 1e-12 {
		t.Fatalf("terminal ocean surface not inherited: %+v", seg)
	}
	depth := riverCenterDepth(m.Config, seg.HalfWidth)
	for i, bed := range seg.Profile {
		if surface := bed + depth; surface > 0.50+1e-12 {
			t.Fatalf(
				"terminal receiving reach[%d] natural surface=%.12f above ocean 0.50",
				i,
				surface,
			)
		}
	}
}

func TestFinalCascadeSlopeIsBackedByRiverBed(t *testing.T) {
	cfg, err := normalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	rivers := []RiverSegment{{
		From:      0,
		To:        1,
		X0:        0,
		Z0:        0,
		X1:        128,
		Z1:        0,
		HalfWidth: 8,
		Profile: []float64{
			0.70,
			0.69,
			0.66,
			0.61,
			0.57,
			0.52,
			0.48,
			0.44,
			0.40,
		},
	}}
	enforceSupportedRiverCascades(rivers, cfg)
	seg := rivers[0]
	sampleBlocks :=
		math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0) /
			float64(len(seg.Profile)-1)
	maxDrop := 0.70 * sampleBlocks / 514.0
	for i := 1; i < len(seg.Profile); i++ {
		drop := seg.Profile[i-1] - seg.Profile[i]
		if drop > maxDrop+1e-12 {
			t.Fatalf(
				"unsupported cascade step %d drop=%.12f max=%.12f profile=%v",
				i,
				drop,
				maxDrop,
				seg.Profile,
			)
		}
		if drop < -1e-12 {
			t.Fatalf("cascade rose downstream at %d: %v", i, seg.Profile)
		}
	}
}

func TestDiagonalLakeFootprintBackwatersCrossingRiver(t *testing.T) {
	g := testGeometry()
	// Keep detailed terrain close to the synthetic lake datum so the lake owns
	// the crossing without requiring excavation of high dry terrain.
	base := testTerrain(t, g, 0.60)
	n := g.Width * g.Height

	rec := &reconcile.Model{
		Geometry:              g,
		Config:                reconcile.Config{RiverStrengthThreshold: 0.30},
		RiverBaseElevation:    filled(n, 0.55),
		RiverBedTarget:        filled(n, 0.55),
		LakeBedCeilingDefined: make([]bool, n),
		LakeBedCeiling:        make([]float64, n),
		WaterKind:             make([]reconcile.WaterKind, n),
		WaterSurfaceDefined:   make([]bool, n),
		WaterSurfaceElevation: make([]float64, n),
		RiverEdges: []reconcile.RiverEdgeConstraint{{
			From: 0, To: 4, RiverStrength: 1,
		}},
	}
	rec.RiverBedTarget[0] = 0.54
	rec.RiverBedTarget[4] = 0.52

	// Node 1 is at (128,0). The diagonal segment 0 -> 4 passes through the
	// rounded-square lake footprint near (64,64). Final rendered connectivity
	// is authoritative: once river and lake actually touch, the lake datum must
	// become a backwater floor so Minecraft never renders an upward water wall.
	lake := 1
	rec.LakeBedCeilingDefined[lake] = true
	rec.LakeBedCeiling[lake] = 0.50
	rec.WaterKind[lake] = reconcile.WaterLake
	rec.WaterSurfaceDefined[lake] = true
	rec.WaterSurfaceElevation[lake] = 0.60

	h := emptyHydrology(g)
	h.FlowTo[0] = 4
	h.LakeID[lake] = 1

	m, err := Build(base, rec, h, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rivers) != 1 {
		t.Fatalf("river segments=%d want=1", len(m.Rivers))
	}
	seg := m.Rivers[0]
	if !seg.ReceiverSurfaceDefined {
		t.Fatalf("spatial lake crossing did not create backwater: %+v", seg)
	}
	if math.Abs(seg.ReceiverSurface-m.Lakes[0].WaterSurface) > 1e-12 {
		t.Fatalf(
			"crossing river receiver=%.12f lake=%.12f",
			seg.ReceiverSurface,
			m.Lakes[0].WaterSurface,
		)
	}
	mid := m.Sample(64, 64)
	if mid.WaterKind != reconcile.WaterLake || !mid.WaterSurfaceDefined {
		t.Fatalf("diagonal lake footprint is not continuous at crossing: %+v", mid)
	}
	if math.Abs(mid.WaterSurfaceElevation-m.Lakes[0].WaterSurface) > 1e-12 {
		t.Fatalf(
			"lake surface=%.12f model datum=%.12f",
			mid.WaterSurfaceElevation,
			m.Lakes[0].WaterSurface,
		)
	}
}
