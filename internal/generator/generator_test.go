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
	g, err := geomorph.Generate(
		geomorph.InputFromModels(h, e, l),
		got.Config.Geomorph,
	)
	if err != nil {
		t.Fatal(err)
	}
	eco, err := ecology.Generate(ecology.InputFromModels(c, h, e, l, g), got.Config.Ecology)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := mcprofile.Generate(mcprofile.InputFromModels(eco, l, g))
	if err != nil {
		t.Fatal(err)
	}
	tr, err := terrain.Generate(
		terrain.InputFromModels(e, l, g),
		got.Config.Terrain,
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

func TestV120RoutedDepositionFeedbackIsVersionedSmallAndNonNegative(t *testing.T) {
	geometry := world.GridGeometry{
		Width:       96,
		Height:      60,
		OriginX:     -95 * 64,
		OriginZ:     -59 * 64,
		SpanXBlocks: 95 * world.DefaultMacroSpacingBlocks,
		SpanZBlocks: 59 * world.DefaultMacroSpacingBlocks,
	}
	baseConfig := Config{
		Seed:                   123456,
		Geometry:               geometry,
		PlateCount:             18,
		OceanFraction:          0.68,
		ContinentFragmentation: 0.35,
		ContinentSeparation:    0.65,
		WorldProfile:           StandardWorldProfile(),
	}
	baseConfig.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.19"
	base, err := GenerateWorld(context.Background(), baseConfig, nil)
	if err != nil {
		t.Fatal(err)
	}

	currentConfig := baseConfig
	currentConfig.WorldProfile = StandardWorldProfile()
	current, err := GenerateWorld(context.Background(), currentConfig, nil)
	if err != nil {
		t.Fatal(err)
	}

	if base.Config.Terrain.RoutedDepositionVersion != 0 {
		t.Fatalf(
			"locked v1.1.19 routed deposition version=%d want=0",
			base.Config.Terrain.RoutedDepositionVersion,
		)
	}
	if current.Config.Terrain.RoutedDepositionVersion != 1 {
		t.Fatalf(
			"current routed deposition version=%d want=1",
			current.Config.Terrain.RoutedDepositionVersion,
		)
	}
	if !reflect.DeepEqual(
		base.Geomorph.RoutedDepositionThickness,
		current.Geomorph.RoutedDepositionThickness,
	) {
		t.Fatal("v1.1.20 changed routed sediment facts instead of only consuming them in Terrain")
	}

	raised := 0
	maxLift := 0.0
	for z := 0; z < geometry.Height; z++ {
		for x := 0; x < geometry.Width; x++ {
			wx, wz := geometry.NodePosition(x, z)
			before := base.Terrain.SampleNormalized(wx, wz)
			after := current.Terrain.SampleNormalized(wx, wz)
			delta := after - before
			if delta < -1e-12 {
				t.Fatalf(
					"routed deposition lowered Terrain at node %d,%d: before=%g after=%g delta=%g",
					x,
					z,
					before,
					after,
					delta,
				)
			}
			if delta > 1e-12 {
				raised++
				maxLift = math.Max(maxLift, delta)
			}
		}
	}
	if raised == 0 {
		t.Fatal("v1.1.20 routed deposition did not affect any Terrain node")
	}
	if maxLift > current.Config.Terrain.RoutedDepositionMaxLift+1e-9 {
		t.Fatalf(
			"routed deposition lift exceeded configured bound: max=%g bound=%g",
			maxLift,
			current.Config.Terrain.RoutedDepositionMaxLift,
		)
	}
	t.Logf(
		"routed deposition terrain feedback: raisedNodes=%d/%d maxNormalizedLift=%.6f",
		raised,
		geometry.Width*geometry.Height,
		maxLift,
	)
}

func TestV122NaturalHeadwatersPreserveLegacyWorldGraph(t *testing.T) {
	const seed int64 = 123456
	cfg := testConfig(seed)
	cfg.WorldProfile = StandardWorldProfile()
	cfg.WorldProfile.Parameters.MainRiverScale = 2
	old := cfg
	old.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.21"
	legacy, err := GenerateWorld(context.Background(), old, nil)
	if err != nil {
		t.Fatal(err)
	}
	modern, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Config.Waterbed.RiverGeometryVersion != 6 || modern.Config.Waterbed.RiverGeometryVersion != 7 {
		t.Fatalf("unexpected locked geometry versions %d and %d", legacy.Config.Waterbed.RiverGeometryVersion, modern.Config.Waterbed.RiverGeometryVersion)
	}
	if len(legacy.Waterbed.Rivers) != len(modern.Waterbed.Rivers) {
		t.Fatalf("river topology changed %d -> %d", len(legacy.Waterbed.Rivers), len(modern.Waterbed.Rivers))
	}
	incoming := make(map[int]int)
	for _, seg := range legacy.Waterbed.Rivers {
		incoming[seg.To]++
	}
	n := 0
	for i, seg := range modern.Waterbed.Rivers {
		previous := legacy.Waterbed.Rivers[i]
		if previous.HeadwaterTaper {
			t.Fatalf("legacy river %d unexpectedly tapered", i)
		}
		if seg.HeadwaterTaper {
			n++
			if incoming[seg.From] != 0 || modern.Hydrology.LakeID[seg.From] != 0 || seg.HalfWidth < 10 || seg.OceanMouth || seg.LakeMouth || seg.LakeOutlet || math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0) < 96 {
				t.Fatalf("river %d incorrectly identified as a source: %+v", i, seg)
			}
		}
		seg.HeadwaterTaper = false
		if !reflect.DeepEqual(previous, seg) {
			t.Fatalf("v22 changed legacy river segment %d beyond the source-width flag", i)
		}
	}
	if !reflect.DeepEqual(legacy.Geomorph.RoutedDepositionThickness, modern.Geomorph.RoutedDepositionThickness) {
		t.Fatal("headwater-only rule unexpectedly changed macro deposition")
	}
	t.Logf("R7.2 seed %d: %d eligible tapered headwaters of %d stable graph reaches", seed, n, len(modern.Waterbed.Rivers))
}

func TestV121MeanderGeometryIsVersionLocked(t *testing.T) {
	const seed int64 = 123456
	legacyConfig := testConfig(seed)
	legacyConfig.WorldProfile = StandardWorldProfile()
	legacyConfig.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.20"
	legacy, err := GenerateWorld(context.Background(), legacyConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	candidateConfig := legacyConfig
	candidateConfig.WorldProfile = StandardWorldProfile()
	candidateConfig.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.21"
	if candidateConfig.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.21" {
		t.Fatalf("unexpected default rule %s", candidateConfig.WorldProfile.GenerationRulesVersion)
	}
	candidate, err := GenerateWorld(context.Background(), candidateConfig, nil)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Config.Waterbed.RiverGeometryVersion != 5 || candidate.Config.Waterbed.RiverGeometryVersion != 6 {
		t.Fatalf("geometry version changed unexpectedly: legacy=%d candidate=%d", legacy.Config.Waterbed.RiverGeometryVersion, candidate.Config.Waterbed.RiverGeometryVersion)
	}
	if legacy.Config.Terrain.RoutedDepositionVersion != candidate.Config.Terrain.RoutedDepositionVersion ||
		legacy.Config.Geomorph.LongitudinalGradeVersion != candidate.Config.Geomorph.LongitudinalGradeVersion {
		t.Fatal("new river version changed sediment or longitudinal-grade defaults")
	}
	if !reflect.DeepEqual(legacy.Geomorph.RoutedDepositionThickness, candidate.Geomorph.RoutedDepositionThickness) {
		t.Fatal("R7 river geometry changed macro routed sediment facts")
	}
	if len(legacy.Waterbed.Rivers) != len(candidate.Waterbed.Rivers) {
		t.Fatalf("river topology changed: %d vs %d", len(legacy.Waterbed.Rivers), len(candidate.Waterbed.Rivers))
	}
	changed := 0
	for i, a := range legacy.Waterbed.Rivers {
		b := candidate.Waterbed.Rivers[i]
		if a.From != b.From || a.To != b.To || a.X0 != b.X0 || a.Z0 != b.Z0 || a.X1 != b.X1 || a.Z1 != b.Z1 || a.HalfWidth != b.HalfWidth {
			t.Fatalf("river reach %d moved endpoint or width instead of centerline", i)
		}
		if a.MeanderOffset != b.MeanderOffset {
			changed++
			if a.MeanderOffset != 0 || a.HalfWidth < 10 || a.OceanMouth || a.LakeMouth || a.LakeOutlet {
				t.Fatalf("v21 modified noncandidate/receiver reach %d: old=%f new=%f mouth=%v/%v/%v from=%d to=%d width=%.2f endpoints=(%.0f,%.0f)->(%.0f,%.0f)", i, a.MeanderOffset, b.MeanderOffset, a.OceanMouth, a.LakeMouth, a.LakeOutlet, a.From, a.To, a.HalfWidth, a.X0, a.Z0, a.X1, a.Z1)
			}
			length := math.Hypot(a.X1-a.X0, a.Z1-a.Z0)
			if length < 96 || (a.X1 != a.X0 && a.Z1 != a.Z0) || math.Abs(b.MeanderOffset) > 0.75*a.HalfWidth+1e-9 {
				t.Fatalf("v21 reach %d has an unbounded or inappropriate bend: %+v", i, b)
			}
		}
		for j := 1; j < len(b.Profile); j++ {
			if b.Profile[j] > b.Profile[j-1]+1e-10 {
				t.Fatalf("v21 river %d uphill at profile sample %d", i, j)
			}
		}
	}
	if changed == 0 {
		t.Fatal("R7 candidate did not improve any wide straight river")
	}
	t.Logf("R7 seed %d: changed %d of %d ordinary reaches, locked v1.1.20 unchanged", seed, changed, len(candidate.Waterbed.Rivers))
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
	if got := current.Waterbed.RiverGeometryVersion; got != 7 {
		t.Fatalf("current river geometry version=%d want=7", got)
	}
	if got := current.Geomorph.LongitudinalGradeVersion; got != 1 {
		t.Fatalf("current longitudinal grade version=%d want=1", got)
	}
	if got := current.Terrain.RoutedDepositionVersion; got != 1 {
		t.Fatalf("current routed deposition version=%d want=1", got)
	}

	lockedV1119RulesCfg := testConfig(2026)
	lockedV1119RulesCfg.WorldProfile = StandardWorldProfile()
	lockedV1119RulesCfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.19"
	lockedV1119Rules, err := resolveTopLevelConfig(lockedV1119RulesCfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockedV1119Rules.Geomorph.LongitudinalGradeVersion; got != 1 {
		t.Fatalf("locked v1.1.19 longitudinal grade version=%d want=1", got)
	}
	if got := lockedV1119Rules.Terrain.RoutedDepositionVersion; got != 0 {
		t.Fatalf("locked v1.1.19 routed deposition version=%d want=0", got)
	}
	if got := lockedV1119Rules.Waterbed.RiverGeometryVersion; got != 5 {
		t.Fatalf("locked v1.1.19 river geometry version=%d want=5", got)
	}

	lockedV1118RulesCfg := testConfig(2026)
	lockedV1118RulesCfg.WorldProfile = StandardWorldProfile()
	lockedV1118RulesCfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.18"
	lockedV1118Rules, err := resolveTopLevelConfig(lockedV1118RulesCfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockedV1118Rules.Geomorph.LongitudinalGradeVersion; got != 0 {
		t.Fatalf("locked v1.1.18 longitudinal grade version=%d want=0", got)
	}
	if got := lockedV1118Rules.Waterbed.RiverGeometryVersion; got != 5 {
		t.Fatalf("locked v1.1.18 river geometry version=%d want=5", got)
	}

	lockedV1116Cfg := testConfig(2026)
	lockedV1116Cfg.WorldProfile = StandardWorldProfile()
	lockedV1116Cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.16"
	lockedV1116, err := resolveTopLevelConfig(lockedV1116Cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockedV1116.Waterbed.RiverGeometryVersion; got != 5 {
		t.Fatalf("locked v1.1.16 river geometry version=%d want=5", got)
	}

	lockedV1112Cfg := testConfig(2026)
	lockedV1112Cfg.WorldProfile = StandardWorldProfile()
	lockedV1112Cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.12"
	lockedV1112, err := resolveTopLevelConfig(lockedV1112Cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockedV1112.Waterbed.RiverGeometryVersion; got != 4 {
		t.Fatalf("locked v1.1.12 river geometry version=%d want=4", got)
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
	if !current.Ecology.EnsureWetlandRegion {
		t.Fatal("current rules must guarantee a natural wetland region")
	}

	lockedV1111Cfg := testConfig(2026)
	lockedV1111Cfg.WorldProfile = StandardWorldProfile()
	lockedV1111Cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.11"
	lockedV1111, err := resolveTopLevelConfig(lockedV1111Cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockedV1111.Waterbed.RiverGeometryVersion; got != 3 {
		t.Fatalf("locked v1.1.11 river geometry version=%d want=3", got)
	}

	lockedV118Cfg := testConfig(2026)
	lockedV118Cfg.WorldProfile = StandardWorldProfile()
	lockedV118Cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.8"
	lockedV118, err := resolveTopLevelConfig(lockedV118Cfg)
	if err != nil {
		t.Fatal(err)
	}
	if lockedV118.Ecology.EnsureWetlandRegion {
		t.Fatal("locked v1.1.8 must preserve pre-guarantee wetland behavior")
	}
	if got := lockedV118.Waterbed.RiverGeometryVersion; got != 3 {
		t.Fatalf("locked v1.1.8 river geometry version=%d want=3", got)
	}

	lockedV117Cfg := testConfig(2026)
	lockedV117Cfg.WorldProfile = StandardWorldProfile()
	lockedV117Cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.7"
	lockedV117, err := resolveTopLevelConfig(lockedV117Cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockedV117.Waterbed.MinRiverBedDepthNormalized; math.Abs(got-0.006) > 1e-12 {
		t.Fatalf("locked v1.1.7 river bed depth=%v want=0.006", got)
	}
	if got := lockedV117.WorldSample.RiverCenterDepthNormalized; math.Abs(got-0.004) > 1e-12 {
		t.Fatalf("locked v1.1.7 river center depth=%v want=0.004", got)
	}
	if got := lockedV117.Waterbed.RiverGeometryVersion; got != 2 {
		t.Fatalf("locked v1.1.7 river geometry version=%d want=2", got)
	}
	if got := lockedV117.Waterbed.LakeGeometryVersion; got != 1 {
		t.Fatalf("locked v1.1.7 lake geometry version=%d want=1", got)
	}
	if got := lockedV117.Terrain.RareMacroStrength; math.Abs(got-1) > 1e-12 {
		t.Fatalf("locked v1.1.7 rare macro strength=%v want=1", got)
	}
	if got := lockedV117.Terrain.MountainStrengthScale; math.Abs(got-1) > 1e-12 {
		t.Fatalf("locked v1.1.7 mountain strength scale=%v want=1", got)
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
				offset := float64(lateral) / 10 * waterbed.RiverHalfWidthAt(seg, tAlong)
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

func TestV1113RealGenerationEnablesDeltaAndV1112StaysLocked(t *testing.T) {
	const width, height = 160, 100
	cell := float64(world.DefaultMacroSpacingBlocks)
	base := Config{
		Seed: 123456,
		Geometry: world.GridGeometry{
			Width: width, Height: height,
			OriginX:     -float64(width-1) * cell / 2,
			OriginZ:     -float64(height-1) * cell / 2,
			SpanXBlocks: float64(width-1) * cell,
			SpanZBlocks: float64(height-1) * cell,
		},
		PlateCount:    18,
		OceanFraction: 0.68,
		WorldProfile:  StandardWorldProfile(),
	}

	current, err := GenerateWorld(context.Background(), base, nil)
	if err != nil {
		t.Fatal(err)
	}
	oceanMouths := 0
	versionFiveMouths := 0
	for _, seg := range current.Waterbed.Rivers {
		if !seg.OceanMouth {
			continue
		}
		oceanMouths++
		if seg.MouthExpansionVersion >= 5 {
			versionFiveMouths++
		}
	}
	deltaCandidates := 0
	for _, candidate := range current.Waterbed.AcceptanceFluvialCandidates() {
		if candidate.Kind == "delta" {
			deltaCandidates++
		}
	}
	if oceanMouths == 0 || versionFiveMouths != oceanMouths {
		t.Fatalf(
			"v1.1.13 real ocean-mouth activation failed: ocean=%d v5=%d",
			oceanMouths,
			versionFiveMouths,
		)
	}
	if deltaCandidates == 0 {
		t.Fatal("v1.1.13 real generation produced no deterministic delta candidate")
	}
	deltaRoute := current.Waterbed.AcceptanceDeltaRoute()
	if len(deltaRoute) < 4 {
		t.Fatalf("v1.1.13 real delta acceptance route points=%d want>=4", len(deltaRoute))
	}

	locked := base
	locked.WorldProfile = StandardWorldProfile()
	locked.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.12"
	legacy, err := GenerateWorld(context.Background(), locked, nil)
	if err != nil {
		t.Fatal(err)
	}
	lockedV5Mouths := 0
	for _, seg := range legacy.Waterbed.Rivers {
		if seg.OceanMouth && seg.MouthExpansionVersion >= 5 {
			lockedV5Mouths++
		}
	}
	lockedDeltaCandidates := 0
	for _, candidate := range legacy.Waterbed.AcceptanceFluvialCandidates() {
		if candidate.Kind == "delta" {
			lockedDeltaCandidates++
		}
	}
	if lockedV5Mouths != 0 || lockedDeltaCandidates != 0 {
		t.Fatalf(
			"locked v1.1.12 unexpectedly enabled delta geometry: oceanV5=%d delta=%d",
			lockedV5Mouths,
			lockedDeltaCandidates,
		)
	}
	if route := legacy.Waterbed.AcceptanceDeltaRoute(); len(route) != 0 {
		t.Fatalf("locked v1.1.12 unexpectedly produced delta route points=%d", len(route))
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

func TestR81cCoastalBackwaterProfileIsolatedFromLockedV122(t *testing.T) {
	cfg := testConfig(2026)
	cfg.WorldProfile = StandardWorldProfile()
	modern, err := resolveTopLevelConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	legacyCfg := cfg
	legacyCfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.22"
	locked, err := resolveTopLevelConfig(legacyCfg)
	if err != nil {
		t.Fatal(err)
	}
	if modern.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.25" ||
		modern.WorldSample.CoastalBackwaterVersion != 1 {
		t.Fatalf("modern rules not isolated: version=%s coastal=%d", modern.WorldProfile.GenerationRulesVersion, modern.WorldSample.CoastalBackwaterVersion)
	}
	if locked.WorldSample.CoastalBackwaterVersion != 0 {
		t.Fatalf("locked v1.1.22 changed coastal rule: %d", locked.WorldSample.CoastalBackwaterVersion)
	}
	if modern.Waterbed.RiverGeometryVersion != locked.Waterbed.RiverGeometryVersion ||
		modern.WorldSample.RiverCenterDepthNormalized != locked.WorldSample.RiverCenterDepthNormalized {
		t.Fatal("coastal backwater profile unexpectedly modified existing river geometry")
	}
}
