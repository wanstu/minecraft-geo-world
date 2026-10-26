package generator

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"

	"geoworld/internal/climate"
	"geoworld/internal/ecology"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
	"geoworld/internal/reconcile"
	"geoworld/internal/terrain"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

func testConfig(seed int64) Config {
	return Config{
		Seed: seed,
		Geometry: world.GridGeometry{
			Width: 48, Height: 30,
			OriginX: -2048, OriginZ: 1024,
			SpanXBlocks: 47 * world.DefaultMacroSpacingBlocks,
			SpanZBlocks: 29 * world.DefaultMacroSpacingBlocks,
		},
		PlateCount:    12,
		OceanFraction: 0.68,
	}
}

func TestGenerateWorldMatchesManualPipeline(t *testing.T) {
	cfg := testConfig(731894205)
	got, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

	p1 := phase1.Generate(phase1.Config{
		Seed: cfg.Seed, Width: cfg.Geometry.Width, Height: cfg.Geometry.Height,
		OriginX: cfg.Geometry.OriginX, OriginZ: cfg.Geometry.OriginZ,
		SpanXBlocks: cfg.Geometry.SpanXBlocks, SpanZBlocks: cfg.Geometry.SpanZBlocks,
		SeaLevelNormalized: world.DefaultSeaLevelNormalized,
		PlateCount:         cfg.PlateCount, OceanFraction: cfg.OceanFraction,
		EdgeOceanBufferFraction: 0.04,
		RareMacroStrength:       1,
	})
	c, err := climate.Generate(climate.InputFromPhase1(p1), climate.Config{SeaLevelNormalized: world.DefaultSeaLevelNormalized})
	if err != nil {
		t.Fatal(err)
	}
	h, err := hydrology.Generate(hydrology.InputFromWorld(p1, c), hydrology.Config{SeaLevelNormalized: world.DefaultSeaLevelNormalized})
	if err != nil {
		t.Fatal(err)
	}
	e, err := erosion.Generate(erosion.InputFromWorld(p1, c, h), erosion.Config{SeaLevelNormalized: world.DefaultSeaLevelNormalized})
	if err != nil {
		t.Fatal(err)
	}
	l, err := landform.Generate(landform.InputFromWorld(p1, e), landform.Config{SeaLevelNormalized: world.DefaultSeaLevelNormalized})
	if err != nil {
		t.Fatal(err)
	}
	g, err := geomorph.Generate(geomorph.InputFromModels(h, e, l), geomorph.Config{SeaLevelNormalized: world.DefaultSeaLevelNormalized})
	if err != nil {
		t.Fatal(err)
	}
	eco, err := ecology.Generate(ecology.InputFromModels(c, h, e, l, g), ecology.Config{})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := mcprofile.Generate(mcprofile.InputFromModels(eco, l, g))
	if err != nil {
		t.Fatal(err)
	}
	tr, err := terrain.Generate(
		terrain.InputFromModels(e, l, g),
		terrain.Config{
			Seed:              cfg.Seed,
			RareMacroStrength: 1,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got.Phase1.Elevation, p1.Elevation) {
		t.Fatal("unified phase1 elevation differs from manual pipeline")
	}
	if !reflect.DeepEqual(got.Hydrology.FlowTo, h.FlowTo) {
		t.Fatal("unified hydrology topology differs from manual pipeline")
	}
	if !reflect.DeepEqual(got.Profile.BiomeFamily, profile.BiomeFamily) {
		t.Fatal("unified Minecraft profile differs from manual pipeline")
	}

	for _, node := range [][2]int{{0, 0}, {7, 4}, {23, 12}, {47, 29}} {
		x, z := cfg.Geometry.NodePosition(node[0], node[1])
		a := got.Terrain.SampleNormalized(x, z)
		b := tr.SampleNormalized(x, z)
		if math.Abs(a-b) > 1e-12 {
			t.Fatalf("terrain differs at node %v: unified=%.12f manual=%.12f", node, a, b)
		}
	}
}

func TestGenerateWorldUsesOneSeaLevel(t *testing.T) {
	cfg := testConfig(10400)
	cfg.SeaLevelNormalized = 0.47

	m, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Surface.SeaLevelNormalized != cfg.SeaLevelNormalized {
		t.Fatalf("surface sea level %.12f want %.12f", m.Surface.SeaLevelNormalized, cfg.SeaLevelNormalized)
	}
	for name, got := range map[string]float64{
		"phase1":    m.Config.Phase1.SeaLevelNormalized,
		"climate":   m.Config.Climate.SeaLevelNormalized,
		"hydrology": m.Config.Hydrology.SeaLevelNormalized,
		"erosion":   m.Config.Erosion.SeaLevelNormalized,
		"landform":  m.Config.Landform.SeaLevelNormalized,
		"geomorph":  m.Config.Geomorph.SeaLevelNormalized,
	} {
		if got != cfg.SeaLevelNormalized {
			t.Fatalf("%s sea level %.12f want %.12f", name, got, cfg.SeaLevelNormalized)
		}
	}
}

func TestGenerateWorldStoresResolvedConfig(t *testing.T) {
	cfg := testConfig(2026)
	m, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.Config.SeaLevelNormalized != world.DefaultSeaLevelNormalized {
		t.Fatalf("resolved sea level %.12f want default %.12f", m.Config.SeaLevelNormalized, world.DefaultSeaLevelNormalized)
	}
	if m.Config.Phase1.EdgeOceanBufferFraction != 0.04 {
		t.Fatalf("resolved phase1 edge buffer %.12f want 0.04", m.Config.Phase1.EdgeOceanBufferFraction)
	}
	if m.Config.Climate.CoastalTempDecayBlocks <= 0 {
		t.Fatal("resolved climate defaults were not retained")
	}
	if m.Config.Hydrology.MinLakeAreaBlocks2 <= 0 {
		t.Fatal("resolved hydrology defaults were not retained")
	}
	if m.Config.Erosion.ReliefRadiusBlocks <= 0 {
		t.Fatal("resolved erosion defaults were not retained")
	}
	if m.Config.Landform.ValleyRadiusBlocks <= 0 {
		t.Fatal("resolved landform defaults were not retained")
	}
	if m.Config.Terrain.CellSizeBlocks != cfg.Geometry.SpacingX() {
		t.Fatalf("resolved terrain spacing %.12f want %.12f", m.Config.Terrain.CellSizeBlocks, cfg.Geometry.SpacingX())
	}
	if m.Config.Reconcile.CoastMarginNormalized <= 0 || m.Config.Reconcile.MinRiverBedDropNormalized <= 0 {
		t.Fatal("resolved reconciliation defaults were not retained")
	}
	if m.Config.Waterbed.MinRiverHalfWidthBlocks <= 0 ||
		m.Config.Waterbed.MaxRiverHalfWidthBlocks <= 0 ||
		m.Config.Waterbed.RiverProfileStepBlocks <= 0 ||
		m.Config.Waterbed.MinRiverBedDepthNormalized <= 0 {
		t.Fatal("resolved waterbed defaults were not retained")
	}
	if m.Config.WorldSample.RiverCenterDepthNormalized <= 0 || m.Config.WorldSample.OceanEligibilityThreshold <= 0 {
		t.Fatal("resolved worldsample defaults were not retained")
	}
	if m.Reconciliation == nil {
		t.Fatal("world model is missing reconciliation stage")
	}
	if m.Reconciliation.SurfaceMismatchAfter != 0 {
		t.Fatalf("reconciliation surface mismatches=%d want=0", m.Reconciliation.SurfaceMismatchAfter)
	}
	if m.Reconciliation.TargetUpslopeRiverEdges != 0 {
		t.Fatalf("reconciliation target river upslope edges=%d want=0", m.Reconciliation.TargetUpslopeRiverEdges)
	}
	if m.CoastTerrain == nil {
		t.Fatal("world model is missing applied coast terrain")
	}
	if m.Waterbed == nil {
		t.Fatal("world model is missing local waterbed stage")
	}
	if m.Runtime == nil {
		t.Fatal("world model is missing runtime terrain/water sampler")
	}
	wd := m.Waterbed.Audit(16, m.Config.Reconcile.FinalUpslopeEpsilon)
	if wd.RiverUpslopeSteps != 0 || wd.LakeBedViolations != 0 {
		t.Fatalf("waterbed audit failed: %+v", wd)
	}
	if m.Reconciliation.AppliedCoastSurfaceMismatch != 0 {
		t.Fatalf("applied coast surface mismatches=%d want=0", m.Reconciliation.AppliedCoastSurfaceMismatch)
	}
	if m.Reconciliation.AppliedCoastMaxNodeError > 1e-10 {
		t.Fatalf("applied coast node error=%.12g want<=1e-10", m.Reconciliation.AppliedCoastMaxNodeError)
	}
	if len(m.Reconciliation.RiverBaseElevation) != cfg.Geometry.Width*cfg.Geometry.Height {
		t.Fatalf("river base length=%d want=%d", len(m.Reconciliation.RiverBaseElevation), cfg.Geometry.Width*cfg.Geometry.Height)
	}
	for _, node := range [][2]int{{0, 0}, {7, 4}, {23, 12}, {47, 29}} {
		i := node[1]*cfg.Geometry.Width + node[0]
		x, z := cfg.Geometry.NodePosition(node[0], node[1])
		got := m.CoastTerrain.SampleNormalized(x, z)
		if math.Abs(got-m.Reconciliation.AppliedCoastElevation[i]) > 1e-12 {
			t.Fatalf("coast terrain sample differs from applied coast elevation at %v", node)
		}
		if math.Abs(got-m.Reconciliation.RiverBaseElevation[i]) > 1e-12 {
			t.Fatalf("river constraints were not rebuilt from coast terrain at %v", node)
		}
	}
}

func TestRiverDepthDefaultsAreGenerationRuleLocked(t *testing.T) {
	currentCfg := testConfig(2026)
	currentCfg.WorldProfile = StandardWorldProfile()
	current, err := resolveTopLevelConfig(currentCfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := current.Waterbed.MinRiverBedDepthNormalized; math.Abs(got-0.006) > 1e-12 {
		t.Fatalf("current river bed depth=%v want=0.006", got)
	}
	if got := current.WorldSample.RiverCenterDepthNormalized; math.Abs(got-0.004) > 1e-12 {
		t.Fatalf("current river center depth=%v want=0.004", got)
	}
	if got := current.Waterbed.RiverGeometryVersion; got != 2 {
		t.Fatalf("current river geometry version=%d want=2", got)
	}
	if got := current.Waterbed.LakeGeometryVersion; got != 1 {
		t.Fatalf("current lake geometry version=%d want=1", got)
	}
	if got := current.Terrain.RareMacroStrength; math.Abs(got-1) > 1e-12 {
		t.Fatalf("current rare macro strength=%v want=1", got)
	}
	if got := current.Terrain.MountainStrengthScale; math.Abs(got-1) > 1e-12 {
		t.Fatalf("current mountain strength scale=%v want=1", got)
	}

	lockedV116Cfg := testConfig(2026)
	lockedV116Cfg.WorldProfile = StandardWorldProfile()
	lockedV116Cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.6"
	lockedV116, err := resolveTopLevelConfig(lockedV116Cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockedV116.Waterbed.MinRiverBedDepthNormalized; math.Abs(got-0.006) > 1e-12 {
		t.Fatalf("locked v1.1.6 river bed depth=%v want=0.006", got)
	}
	if got := lockedV116.WorldSample.RiverCenterDepthNormalized; math.Abs(got-0.004) > 1e-12 {
		t.Fatalf("locked v1.1.6 river center depth=%v want=0.004", got)
	}
	if got := lockedV116.Waterbed.RiverGeometryVersion; got != 1 {
		t.Fatalf("locked v1.1.6 river geometry version=%d want=1", got)
	}
	if got := lockedV116.Waterbed.LakeGeometryVersion; got != 0 {
		t.Fatalf("locked v1.1.6 lake geometry version=%d want=0", got)
	}
	if got := lockedV116.Terrain.RareMacroStrength; got != 0 {
		t.Fatalf("locked v1.1.6 rare macro strength=%v want=0", got)
	}
	if got := lockedV116.Terrain.MountainStrengthScale; got != 0 {
		t.Fatalf("locked v1.1.6 mountain strength scale=%v want=0", got)
	}

	lockedV114Cfg := testConfig(2026)
	lockedV114Cfg.WorldProfile = StandardWorldProfile()
	lockedV114Cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.4"
	lockedV114, err := resolveTopLevelConfig(lockedV114Cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockedV114.Waterbed.MinRiverBedDepthNormalized; math.Abs(got-0.006) > 1e-12 {
		t.Fatalf("locked v1.1.4 river bed depth=%v want=0.006", got)
	}
	if got := lockedV114.WorldSample.RiverCenterDepthNormalized; math.Abs(got-0.004) > 1e-12 {
		t.Fatalf("locked v1.1.4 river center depth=%v want=0.004", got)
	}
	if got := lockedV114.Waterbed.RiverGeometryVersion; got != 0 {
		t.Fatalf("locked v1.1.4 river geometry version=%d want=0", got)
	}

	lockedCfg := testConfig(2026)
	lockedCfg.WorldProfile = StandardWorldProfile()
	lockedCfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.3"
	locked, err := resolveTopLevelConfig(lockedCfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := locked.Waterbed.MinRiverBedDepthNormalized; math.Abs(got-0.004) > 1e-12 {
		t.Fatalf("locked v1.1.3 river bed depth=%v want=0.004", got)
	}
	if got := locked.WorldSample.RiverCenterDepthNormalized; math.Abs(got-0.0025) > 1e-12 {
		t.Fatalf("locked v1.1.3 river center depth=%v want=0.0025", got)
	}
}

func TestV115OrdinaryRiverInnerWetCoreHasNoDryHoles(t *testing.T) {
	const (
		seed      = int64(123456)
		cell      = 128.0
		width     = 320
		height    = 200
		coreRatio = 0.80
	)
	cfg := Config{
		Seed: seed,
		Geometry: world.GridGeometry{
			Width: width, Height: height,
			OriginX:     -float64(width-1) * cell / 2,
			OriginZ:     -float64(height-1) * cell / 2,
			SpanXBlocks: float64(width-1) * cell,
			SpanZBlocks: float64(height-1) * cell,
		},
		PlateCount:             18,
		OceanFraction:          0.68,
		ContinentFragmentation: 0.35,
		ContinentSeparation:    0.65,
		WorldProfile:           StandardWorldProfile(),
	}
	m, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

	samples := 0
	dry := 0
	for _, seg := range m.Waterbed.Rivers {
		if seg.MouthExpansionVersion < 4 ||
			seg.HalfWidth < 8 ||
			seg.OceanMouth ||
			seg.LakeMouth ||
			seg.LakeOutlet ||
			math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0) < 80 {
			continue
		}
		for step := 2; step <= 8; step++ {
			tAlong := float64(step) / 10
			cx, cz, nx, nz := riverFrameForWetCoreTest(seg, tAlong)
			for lateral := -8; lateral <= 8; lateral++ {
				offset := float64(lateral) / 10 * seg.HalfWidth
				q := m.Runtime.Sample(cx+nx*offset, cz+nz*offset)
				samples++
				if !q.WaterSurfaceDefined ||
					q.WaterKind == reconcile.WaterNone {
					dry++
				}
			}
		}
	}
	if samples < 10000 {
		t.Fatalf("wet-core regression sampled only %d points", samples)
	}
	if dry != 0 {
		t.Fatalf(
			"v1.1.5 ordinary river inner %.0f%% wet core has %d dry holes across %d samples",
			coreRatio*100,
			dry,
			samples,
		)
	}
}

func riverFrameForWetCoreTest(
	seg waterbed.RiverSegment,
	t float64,
) (float64, float64, float64, float64) {
	return waterbed.RiverCenterFrame(seg, t)
}

func TestGenerateWorldReportsOrderedProgress(t *testing.T) {
	cfg := testConfig(42)
	var got []Progress
	_, err := GenerateWorld(context.Background(), cfg, func(p Progress) {
		got = append(got, p)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(orderedStages) {
		t.Fatalf("progress events=%d want=%d", len(got), len(orderedStages))
	}
	for i, p := range got {
		if p.Stage != orderedStages[i] || p.Completed != i+1 || p.Total != len(orderedStages) {
			t.Fatalf("progress[%d]=%+v want stage=%s completed=%d total=%d",
				i, p, orderedStages[i], i+1, len(orderedStages))
		}
	}
}

func TestGenerateWorldHonorsCancellationBeforeWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := GenerateWorld(ctx, testConfig(1), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v want context canceled", err)
	}
}

func TestGenerateWorldRejectsSharedContractConflicts(t *testing.T) {
	t.Run("sea level", func(t *testing.T) {
		cfg := testConfig(1)
		cfg.SeaLevelNormalized = 0.47
		cfg.Hydrology.SeaLevelNormalized = 0.5
		if _, err := GenerateWorld(context.Background(), cfg, nil); err == nil {
			t.Fatal("expected stage sea-level conflict")
		}
	})

	t.Run("terrain geometry", func(t *testing.T) {
		cfg := testConfig(1)
		cfg.Terrain.CellSizeBlocks = 64
		if _, err := GenerateWorld(context.Background(), cfg, nil); err == nil {
			t.Fatal("expected terrain legacy geometry conflict")
		}
	})

	t.Run("terrain seed", func(t *testing.T) {
		cfg := testConfig(1)
		cfg.Terrain.Seed = 2
		if _, err := GenerateWorld(context.Background(), cfg, nil); err == nil {
			t.Fatal("expected terrain seed conflict")
		}
	})
}

func TestGenerateWorldRejectsInvalidGeometryWithoutPanic(t *testing.T) {
	cfg := testConfig(1)
	cfg.Geometry.SpanZBlocks = cfg.Geometry.SpanXBlocks
	if _, err := GenerateWorld(context.Background(), cfg, nil); err == nil {
		t.Fatal("expected anisotropic spacing to return an error")
	}
}

func TestPhase1GenerateCheckedReturnsGeometryError(t *testing.T) {
	_, err := phase1.GenerateChecked(phase1.Config{
		Width: 4, Height: 4,
		SpanXBlocks: 384,
		SpanZBlocks: 192,
	})
	if err == nil {
		t.Fatal("expected GenerateChecked to return invalid geometry error")
	}
}

func TestGenerateWorldRuntimeProfileMatchesFinalSemantics(t *testing.T) {
	cfg := testConfig(731894205)
	m, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m.RuntimeProfile == nil {
		t.Fatal("world model is missing runtime profile sampler")
	}

	check := func(label string, x, z float64) {
		s := m.RuntimeProfile.Sample(x, z)
		if s.SemanticFallback {
			t.Fatalf("%s used semantic fallback at %.2f,%.2f", label, x, z)
		}
		if s.SemanticMismatch {
			t.Fatalf("%s profile contradicts runtime water=%v sourceLand=%v biome=%v terrain=%v",
				label, s.World.WaterKind, s.SourceLand, s.BiomeFamily, s.TerrainVariant)
		}
		if s.EcoRegion == ecology.RegionUnknown ||
			s.BiomeFamily == mcprofile.BiomeUnknown ||
			s.TerrainVariant == mcprofile.TerrainUnknown ||
			s.SurfaceProfile == mcprofile.SurfaceUnknown ||
			s.VegetationProfile == mcprofile.VegetationUnknown {
			t.Fatalf("%s returned unknown runtime profile: %+v", label, s)
		}
	}

	for z := 0; z < cfg.Geometry.Height; z++ {
		for x := 0; x < cfg.Geometry.Width; x++ {
			wx, wz := cfg.Geometry.NodePosition(x, z)
			check("macro-node", wx, wz)
		}
	}

	offsets := [][2]float64{
		{0.25, 0.25},
		{0.75, 0.25},
		{0.50, 0.50},
		{0.25, 0.75},
		{0.75, 0.75},
	}
	mixedCells := 0
	spacing := cfg.Geometry.SpacingX()
	for z := 0; z < cfg.Geometry.Height-1; z++ {
		for x := 0; x < cfg.Geometry.Width-1; x++ {
			ids := [4]int{
				z*cfg.Geometry.Width + x,
				z*cfg.Geometry.Width + x + 1,
				(z+1)*cfg.Geometry.Width + x,
				(z+1)*cfg.Geometry.Width + x + 1,
			}
			land := 0
			for _, i := range ids {
				if m.Surface.LandMask[i] {
					land++
				}
			}
			if land == 0 || land == 4 {
				continue
			}
			mixedCells++
			for _, off := range offsets {
				wx := cfg.Geometry.OriginX + (float64(x)+off[0])*spacing
				wz := cfg.Geometry.OriginZ + (float64(z)+off[1])*spacing
				check("mixed-coast", wx, wz)
			}
		}
	}
	if mixedCells == 0 {
		t.Fatal("test world did not contain a mixed coast cell")
	}
}
