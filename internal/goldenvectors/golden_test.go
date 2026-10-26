package goldenvectors

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"geoworld/internal/runtimeworld"
	"geoworld/internal/worldmodelio"
)

func loadGoldenDocument(t *testing.T) Document {
	t.Helper()
	return loadGoldenDocumentFile(t, "runtime-v1.json")
}

func loadGoldenDocumentFile(t *testing.T, name string) Document {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Format != Format || doc.FormatVersion != FormatVersion {
		t.Fatalf("unexpected golden format: %q v%d", doc.Format, doc.FormatVersion)
	}
	if doc.ModelSchemaVersion != worldmodelio.CurrentSchemaVersion {
		t.Fatalf("golden model schema=%d current=%d", doc.ModelSchemaVersion, worldmodelio.CurrentSchemaVersion)
	}
	if doc.GeneratorVersion != worldmodelio.CurrentGeneratorVersion {
		t.Fatalf("golden generator=%q current=%q", doc.GeneratorVersion, worldmodelio.CurrentGeneratorVersion)
	}
	if doc.FloatTolerance <= 0 {
		t.Fatalf("invalid tolerance: %g", doc.FloatTolerance)
	}
	return doc
}

func TestPROFILE06NaturalResultChangesRequireVersionChange(t *testing.T) {
	want := loadGoldenDocumentFile(t, "runtime-rules-v1.1.json")
	committed, err := worldmodelio.Load(filepath.Join("testdata", "runtime-rules-v1.1.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	freshModel, err := GenerateFixture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := Build(freshModel)
	if err != nil {
		t.Fatal(err)
	}

	changed := !sameGoldenResults(fresh, want)
	versionChanged :=
		freshModel.Config.WorldProfile.WorldProfileVersion != committed.Manifest.WorldProfileVersion ||
			freshModel.Config.WorldProfile.GenerationRulesVersion != committed.Manifest.GenerationRulesVersion ||
			freshModel.Config.WorldProfile.WorldModelVersion != committed.Manifest.WorldModelVersion
	if changed && !versionChanged {
		t.Fatalf(
			"natural results changed without profile/rules/model version change: profile=%s rules=%s model=%s",
			committed.Manifest.WorldProfileVersion,
			committed.Manifest.GenerationRulesVersion,
			committed.Manifest.WorldModelVersion,
		)
	}
}

func TestPROFILE07PureOptimizationKeepsLockedProfileResults(t *testing.T) {
	want := loadGoldenDocument(t)
	committed, err := worldmodelio.Load(filepath.Join("testdata", "runtime-v1.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	freshModel, err := GenerateFixture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := Build(freshModel)
	if err != nil {
		t.Fatal(err)
	}

	sameVersion :=
		freshModel.Config.WorldProfile.WorldProfileVersion == committed.Manifest.WorldProfileVersion &&
			freshModel.Config.WorldProfile.GenerationRulesVersion == committed.Manifest.GenerationRulesVersion &&
			freshModel.Config.WorldProfile.WorldModelVersion == committed.Manifest.WorldModelVersion
	if !sameVersion {
		t.Skip("current generator intentionally uses a different locked profile/rules/model version")
	}
	if !sameGoldenResults(fresh, want) {
		t.Fatal("same locked profile/rules/model version changed natural results")
	}
}

func sameGoldenResults(got, want Document) bool {
	if len(got.Cases) != len(want.Cases) {
		return false
	}
	for i := range want.Cases {
		gotCase, wantCase := got.Cases[i], want.Cases[i]
		if gotCase.Name != wantCase.Name ||
			gotCase.Category != wantCase.Category ||
			gotCase.X != wantCase.X ||
			gotCase.Z != wantCase.Z {
			return false
		}
		if CompareSample(gotCase.Expected, wantCase.Expected, want.FloatTolerance) != nil {
			return false
		}
	}
	return true
}

func TestCommittedGoldenDocumentMatchesFreshGeneration(t *testing.T) {
	want := loadGoldenDocument(t)
	model, err := GenerateFixture(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := Build(model)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != want.Format ||
		got.FormatVersion != want.FormatVersion ||
		got.ModelSchemaVersion != want.ModelSchemaVersion ||
		got.GeneratorVersion != want.GeneratorVersion ||
		!reflect.DeepEqual(got.Source, want.Source) ||
		len(got.Cases) != len(want.Cases) {
		t.Fatalf("fresh golden metadata differs from committed golden; regenerate intentionally with go run ./cmd/geoworld-golden-vectors -force")
	}
	for i := range want.Cases {
		gotCase, wantCase := got.Cases[i], want.Cases[i]
		if gotCase.Name != wantCase.Name ||
			gotCase.Category != wantCase.Category ||
			gotCase.Description != wantCase.Description ||
			gotCase.X != wantCase.X ||
			gotCase.Z != wantCase.Z {
			t.Fatalf("golden case %d metadata changed: got=%+v want=%+v", i, gotCase, wantCase)
		}
		if err := CompareSample(gotCase.Expected, wantCase.Expected, want.FloatTolerance); err != nil {
			t.Fatalf("fresh golden case %s differs: %v", wantCase.Name, err)
		}
	}
}

func TestCommittedGoldenModelMatchesGoldenDocument(t *testing.T) {
	doc := loadGoldenDocument(t)
	modelPath := filepath.Join("testdata", "runtime-v1.geoworld")
	model, err := worldmodelio.Load(modelPath)
	if err != nil {
		t.Fatal(err)
	}
	if model.Manifest.SchemaVersion != doc.ModelSchemaVersion ||
		model.Manifest.GeneratorVersion != doc.GeneratorVersion ||
		model.Manifest.Seed != doc.Source.Seed ||
		model.Geometry != doc.Source.Geometry.World() {
		t.Fatalf("golden model metadata does not match golden document: manifest=%+v source=%+v", model.Manifest, doc.Source)
	}

	runtimeWorld, err := runtimeworld.NewReopened(model)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range doc.Cases {
		got := FromRuntimeSample(runtimeWorld.Sample(testCase.X, testCase.Z))
		if err := CompareSample(got, testCase.Expected, doc.FloatTolerance); err != nil {
			t.Fatalf("%s (%s) at %.6f,%.6f: %v", testCase.Name, testCase.Category, testCase.X, testCase.Z, err)
		}
	}
}

func TestLegacyV11GoldenModelRemainsLocked(t *testing.T) {
	doc := loadGoldenDocumentFile(t, "runtime-rules-v1.1.json")
	model, err := worldmodelio.Load(filepath.Join("testdata", "runtime-rules-v1.1.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	runtimeWorld, err := runtimeworld.NewReopened(model)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range doc.Cases {
		got := FromRuntimeSample(runtimeWorld.Sample(testCase.X, testCase.Z))
		if err := CompareSample(got, testCase.Expected, doc.FloatTolerance); err != nil {
			t.Fatalf(
				"legacy v1.1 changed for %s (%s) at %.6f,%.6f: %v",
				testCase.Name,
				testCase.Category,
				testCase.X,
				testCase.Z,
				err,
			)
		}
	}
}

func TestLegacyV113GoldenModelRemainsLocked(t *testing.T) {
	doc := loadGoldenDocumentFile(t, "runtime-rules-v1.1.3.json")
	model, err := worldmodelio.Load(filepath.Join("testdata", "runtime-rules-v1.1.3.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	if got := model.Manifest.GenerationRulesVersion; got != "geoworld-rules-v1.1.3" {
		t.Fatalf("legacy v1.1.3 rules=%q want geoworld-rules-v1.1.3", got)
	}
	runtimeWorld, err := runtimeworld.NewReopened(model)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range doc.Cases {
		got := FromRuntimeSample(runtimeWorld.Sample(testCase.X, testCase.Z))
		if err := CompareSample(got, testCase.Expected, doc.FloatTolerance); err != nil {
			t.Fatalf(
				"legacy v1.1.3 changed for %s (%s) at %.6f,%.6f: %v",
				testCase.Name,
				testCase.Category,
				testCase.X,
				testCase.Z,
				err,
			)
		}
	}
}

func TestLegacyV114GoldenModelRemainsLocked(t *testing.T) {
	doc := loadGoldenDocumentFile(t, "runtime-rules-v1.1.4.json")
	model, err := worldmodelio.Load(filepath.Join("testdata", "runtime-rules-v1.1.4.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	if got := model.Manifest.GenerationRulesVersion; got != "geoworld-rules-v1.1.4" {
		t.Fatalf("legacy v1.1.4 rules=%q want geoworld-rules-v1.1.4", got)
	}
	runtimeWorld, err := runtimeworld.NewReopened(model)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range doc.Cases {
		got := FromRuntimeSample(runtimeWorld.Sample(testCase.X, testCase.Z))
		if err := CompareSample(got, testCase.Expected, doc.FloatTolerance); err != nil {
			t.Fatalf(
				"legacy v1.1.4 changed for %s (%s) at %.6f,%.6f: %v",
				testCase.Name,
				testCase.Category,
				testCase.X,
				testCase.Z,
				err,
			)
		}
	}
}

func TestLegacyV115GoldenModelRemainsLocked(t *testing.T) {
	doc := loadGoldenDocumentFile(t, "runtime-rules-v1.1.5.json")
	model, err := worldmodelio.Load(filepath.Join("testdata", "runtime-rules-v1.1.5.geoworld"))
	if err != nil {
		t.Fatal(err)
	}
	if got := model.Manifest.GenerationRulesVersion; got != "geoworld-rules-v1.1.5" {
		t.Fatalf("legacy v1.1.5 rules=%q want geoworld-rules-v1.1.5", got)
	}
	runtimeWorld, err := runtimeworld.NewReopened(model)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range doc.Cases {
		got := FromRuntimeSample(runtimeWorld.Sample(testCase.X, testCase.Z))
		if err := CompareSample(got, testCase.Expected, doc.FloatTolerance); err != nil {
			t.Fatalf(
				"legacy v1.1.5 changed for %s (%s) at %.6f,%.6f: %v",
				testCase.Name,
				testCase.Category,
				testCase.X,
				testCase.Z,
				err,
			)
		}
	}
}

func TestGoldenCoverageIncludesRequiredCrossLanguageCases(t *testing.T) {
	doc := loadGoldenDocument(t)
	required := map[string]bool{
		"outer-corner":      false,
		"outer-edge":        false,
		"chunk-boundary":    false,
		"coordinate-origin": false,
		"macro-grid":        false,
		"coast":             false,
		"river-mouth":       false,
		"lake":              false,
		"outer-extension":   false,
	}
	byName := make(map[string]Case, len(doc.Cases))
	for _, testCase := range doc.Cases {
		byName[testCase.Name] = testCase
		if _, ok := required[testCase.Category]; ok {
			required[testCase.Category] = true
		}
	}
	for category, present := range required {
		if !present {
			t.Fatalf("golden vector category %q is missing", category)
		}
	}
	if got := byName["river_backwater"].Expected.WaterKind; got != "River" {
		t.Fatalf("river_backwater water kind=%q want River", got)
	}
	if got := byName["lake_center"].Expected.WaterKind; got != "Lake" {
		t.Fatalf("lake_center water kind=%q want Lake", got)
	}
	if p := byName["negative_chunk_boundary"]; p.X >= 0 || p.Z >= 0 || int64(p.X)%16 != 0 || int64(p.Z)%16 != 0 {
		t.Fatalf("negative_chunk_boundary is not a negative 16-block boundary: %.3f,%.3f", p.X, p.Z)
	}
}
