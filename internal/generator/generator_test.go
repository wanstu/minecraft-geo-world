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
	"geoworld/internal/terrain"
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
	tr, err := terrain.Generate(terrain.InputFromModels(e, l, g), terrain.Config{Seed: cfg.Seed})
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
	if m.Reconciliation == nil {
		t.Fatal("world model is missing reconciliation stage")
	}
	if m.Reconciliation.SurfaceMismatchAfter != 0 {
		t.Fatalf("reconciliation surface mismatches=%d want=0", m.Reconciliation.SurfaceMismatchAfter)
	}
	if m.Reconciliation.TargetUpslopeRiverEdges != 0 {
		t.Fatalf("reconciliation target river upslope edges=%d want=0", m.Reconciliation.TargetUpslopeRiverEdges)
	}
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
