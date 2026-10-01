package worldmodelio

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"geoworld/internal/generator"
	"geoworld/internal/world"
)

func testGeneratedWorld(t *testing.T) *generator.WorldModel {
	t.Helper()
	g := world.GridGeometry{
		Width: 48, Height: 30,
		OriginX: -2048, OriginZ: 1024,
		SpanXBlocks: 47 * world.DefaultMacroSpacingBlocks,
		SpanZBlocks: 29 * world.DefaultMacroSpacingBlocks,
	}
	m, err := generator.GenerateWorld(context.Background(), generator.Config{
		Seed:          731894205,
		Geometry:      g,
		PlateCount:    12,
		OceanFraction: 0.68,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestSaveLoadRoundTripPreservesRuntimeSamplesExactly(t *testing.T) {
	original := testGeneratedWorld(t)
	path := filepath.Join(t.TempDir(), "world.geoworld")
	if err := Save(path, original); err != nil {
		t.Fatal(err)
	}

	reopened, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Manifest.SchemaVersion != CurrentSchemaVersion ||
		reopened.Manifest.GeneratorVersion != CurrentGeneratorVersion {
		t.Fatalf("unexpected manifest: %+v", reopened.Manifest)
	}
	if !reflect.DeepEqual(reopened.Config, original.Config) {
		t.Fatal("resolved config changed after reopen")
	}
	if reopened.Geometry != original.Geometry {
		t.Fatalf("geometry changed after reopen: got=%+v want=%+v", reopened.Geometry, original.Geometry)
	}

	points := [][2]float64{
		{original.Geometry.OriginX, original.Geometry.OriginZ},
		{original.Geometry.OriginX + original.Geometry.SpanXBlocks, original.Geometry.OriginZ + original.Geometry.SpanZBlocks},
		{-1024, 2048}, // negative X and a Minecraft chunk boundary
	}
	x, z := original.Geometry.NodePosition(17, 11)
	points = append(points, [2]float64{x, z})
	points = append(points, [2]float64{x + 37.25, z + 79.5})

	if len(original.Waterbed.Rivers) > 0 {
		seg := original.Waterbed.Rivers[0]
		points = append(points, [2]float64{
			seg.X0 + (seg.X1-seg.X0)*0.37,
			seg.Z0 + (seg.Z1-seg.Z0)*0.37,
		})
	}
	if len(original.Waterbed.Lakes) > 0 {
		lake := original.Waterbed.Lakes[0]
		points = append(points, [2]float64{lake.X, lake.Z})
	}
	if p, ok := firstMixedCoastPoint(original); ok {
		points = append(points, p)
	}

	for _, p := range points {
		before := original.RuntimeProfile.Sample(p[0], p[1])
		after := reopened.RuntimeProfile.Sample(p[0], p[1])
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("runtime sample changed after reopen at %.6f,%.6f\nbefore=%+v\nafter=%+v",
				p[0], p[1], before, after)
		}
	}

	beforeDiag := original.Waterbed.Audit(16, original.Config.Reconcile.FinalUpslopeEpsilon)
	afterDiag := reopened.Waterbed.Audit(16, reopened.Config.Reconcile.FinalUpslopeEpsilon)
	if !reflect.DeepEqual(beforeDiag, afterDiag) {
		t.Fatalf("waterbed diagnostics changed after reopen\nbefore=%+v\nafter=%+v", beforeDiag, afterDiag)
	}
}

func TestSaveRefusesExistingTargetWithoutChangingIt(t *testing.T) {
	model := testGeneratedWorld(t)
	path := filepath.Join(t.TempDir(), "existing.geoworld")
	want := []byte("existing-complete-model")
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}

	err := Save(path, model)
	if !errors.Is(err, ErrTargetExists) {
		t.Fatalf("error=%v want ErrTargetExists", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("existing target changed: got=%q want=%q", got, want)
	}
}

func TestLoadRejectsPayloadChecksumMismatch(t *testing.T) {
	model := testGeneratedWorld(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.geoworld")
	if err := Save(path, model); err != nil {
		t.Fatal(err)
	}

	env, err := readEnvelope(path)
	if err != nil {
		t.Fatal(err)
	}
	env.Payload.Surface.LandMask[0] = !env.Payload.Surface.LandMask[0]

	corrupt := filepath.Join(dir, "checksum-mismatch.geoworld")
	writeEnvelopeForTest(t, corrupt, env)
	if _, err := Load(corrupt); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("error=%v want ErrChecksumMismatch", err)
	}
}

func TestLoadRejectsUnsupportedVersions(t *testing.T) {
	model := testGeneratedWorld(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.geoworld")
	if err := Save(path, model); err != nil {
		t.Fatal(err)
	}
	env, err := readEnvelope(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("schema", func(t *testing.T) {
		bad := env
		bad.Manifest.SchemaVersion++
		p := filepath.Join(dir, "bad-schema.geoworld")
		writeEnvelopeForTest(t, p, bad)
		if _, err := Load(p); !errors.Is(err, ErrUnsupportedSchema) {
			t.Fatalf("error=%v want ErrUnsupportedSchema", err)
		}
	})

	t.Run("generator", func(t *testing.T) {
		bad := env
		bad.Manifest.GeneratorVersion = "future-generator"
		p := filepath.Join(dir, "bad-generator.geoworld")
		writeEnvelopeForTest(t, p, bad)
		if _, err := Load(p); !errors.Is(err, ErrUnsupportedGeneratorVersion) {
			t.Fatalf("error=%v want ErrUnsupportedGeneratorVersion", err)
		}
	})
}

func TestReplaceExistingTargetPublishesCompleteNewModel(t *testing.T) {
	first := testGeneratedWorld(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "replace.geoworld")
	if err := Save(path, first); err != nil {
		t.Fatal(err)
	}

	g := first.Geometry
	second, err := generator.GenerateWorld(context.Background(), generator.Config{
		Seed:          10400,
		Geometry:      g,
		PlateCount:    12,
		OceanFraction: 0.68,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if err := Replace(path, second); err != nil {
		t.Fatal(err)
	}
	reopened, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Manifest.Seed != second.Config.Seed {
		t.Fatalf("reopened seed=%d want=%d", reopened.Manifest.Seed, second.Config.Seed)
	}

	x, z := g.NodePosition(17, 11)
	want := second.RuntimeProfile.Sample(x+31.25, z+47.5)
	got := reopened.RuntimeProfile.Sample(x+31.25, z+47.5)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("replacement runtime sample mismatch\ngot=%+v\nwant=%+v", got, want)
	}

	temps, err := filepath.Glob(filepath.Join(dir, ".replace.geoworld.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temps) != 0 {
		t.Fatalf("replacement left temporary files: %v", temps)
	}
}

func TestReplaceMissingTargetDoesNotCreateFile(t *testing.T) {
	model := testGeneratedWorld(t)
	path := filepath.Join(t.TempDir(), "missing.geoworld")
	if err := Replace(path, model); err == nil {
		t.Fatal("expected replace of missing target to fail")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("replace unexpectedly created target: %v", err)
	}
}

func TestInspectAndValidateSeparateHeaderFromPayloadIntegrity(t *testing.T) {
	model := testGeneratedWorld(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.geoworld")
	if err := Save(path, model); err != nil {
		t.Fatal(err)
	}

	inspection, err := Inspect(path)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Manifest.Seed != model.Config.Seed || inspection.FileBytes <= 0 {
		t.Fatalf("unexpected inspection: %+v", inspection)
	}
	validated, err := Validate(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(validated, inspection) {
		t.Fatalf("validate inspection differs\ninspect=%+v\nvalidate=%+v", inspection, validated)
	}

	env, err := readEnvelope(path)
	if err != nil {
		t.Fatal(err)
	}
	env.Payload.Surface.LandMask[0] = !env.Payload.Surface.LandMask[0]
	corrupt := filepath.Join(dir, "corrupt-payload.geoworld")
	writeEnvelopeForTest(t, corrupt, env)

	// Inspect stops after the manifest, so payload corruption does not prevent
	// version/seed/geometry discovery.
	corruptInspection, err := Inspect(corrupt)
	if err != nil {
		t.Fatal(err)
	}
	if corruptInspection.Manifest != inspection.Manifest {
		t.Fatalf("corrupt file manifest changed: %+v", corruptInspection.Manifest)
	}
	if _, err := Validate(corrupt); !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("validate error=%v want ErrChecksumMismatch", err)
	}
}

func TestSaveRejectsIncompleteWorldModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "incomplete.geoworld")
	if err := Save(path, &generator.WorldModel{}); err == nil {
		t.Fatal("expected incomplete model save to fail")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("incomplete save created target: %v", err)
	}
}

func firstMixedCoastPoint(m *generator.WorldModel) ([2]float64, bool) {
	g := m.Geometry
	for z := 0; z < g.Height-1; z++ {
		for x := 0; x < g.Width-1; x++ {
			ids := [4]int{
				z*g.Width + x,
				z*g.Width + x + 1,
				(z+1)*g.Width + x,
				(z+1)*g.Width + x + 1,
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
			return [2]float64{
				g.OriginX + (float64(x)+0.5)*g.SpacingX(),
				g.OriginZ + (float64(z)+0.5)*g.SpacingZ(),
			}, true
		}
	}
	return [2]float64{}, false
}

func writeEnvelopeForTest(t *testing.T, path string, env fileEnvelope) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	if err := json.NewEncoder(gz).Encode(env); err != nil {
		_ = gz.Close()
		_ = f.Close()
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
