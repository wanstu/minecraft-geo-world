package main

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"geoworld/internal/generator"
	"geoworld/internal/mcprofile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
)

func TestBrowserURL(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"0.0.0.0:8788", "http://127.0.0.1:8788"},
		{":8788", "http://127.0.0.1:8788"},
		{"127.0.0.1:8788", "http://127.0.0.1:8788"},
		{"192.168.1.41:8788", "http://192.168.1.41:8788"},
		{"[::]:8788", "http://127.0.0.1:8788"},
	}
	for _, tt := range tests {
		if got := browserURL(tt.addr); got != tt.want {
			t.Fatalf("browserURL(%q) = %q, want %q", tt.addr, got, tt.want)
		}
	}
}

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"GeoWorld 世界 QA", "GeoWorld 世界 QA"},
		{"GeoWorld 世界:QA?", "GeoWorld 世界_QA_"},
		{"  preview.  ", "preview"},
		{"CON", "_CON"},
		{"a/b\\c|d*e", "a_b_c_d_e"},
	}
	for _, tt := range tests {
		if got := sanitizeFolderName(tt.in); got != tt.want {
			t.Fatalf("sanitizeFolderName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestQualityLayerPalettesDistinguishTerrainAndSurface(t *testing.T) {
	plain := runtimeprofile.Sample{TerrainVariant: mcprofile.TerrainPlain}
	mountain := runtimeprofile.Sample{TerrainVariant: mcprofile.TerrainMountain}
	if landformColor(plain) == landformColor(mountain) {
		t.Fatal("plain and mountain must use different landform colors")
	}

	sand := runtimeprofile.Sample{SurfaceProfile: mcprofile.SurfaceDesertSand}
	gravel := runtimeprofile.Sample{SurfaceProfile: mcprofile.SurfaceDesertGravel}
	snow := runtimeprofile.Sample{SurfaceProfile: mcprofile.SurfaceAlpineSnow}
	if surfaceColor(sand) == surfaceColor(gravel) {
		t.Fatal("desert sand and legacy desert gravel must be visually distinguishable")
	}
	if surfaceColor(snow) == surfaceColor(sand) {
		t.Fatal("alpine snow and desert sand must be visually distinguishable")
	}
}

func TestAnomalyReportMatchesAfterModelReopen(t *testing.T) {
	const (
		width    = 80
		height   = 50
		cellSize = 128.0
	)
	geometry := world.GridGeometry{
		Width:       width,
		Height:      height,
		OriginX:     -float64(width-1) * cellSize / 2,
		OriginZ:     -float64(height-1) * cellSize / 2,
		SpanXBlocks: float64(width-1) * cellSize,
		SpanZBlocks: float64(height-1) * cellSize,
	}
	model, err := generator.GenerateWorld(context.Background(), generator.Config{
		Seed:     731894205,
		Geometry: geometry,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	generated := &activeWorld{
		geometry: model.Geometry,
		sample:   model.RuntimeProfile.Sample,
		model:    model,
	}
	generatedReport := buildAnomalyReport(generated)

	path := filepath.Join(t.TempDir(), "world.geoworld")
	if err := worldmodelio.Save(path, model); err != nil {
		t.Fatal(err)
	}
	reopenedModel, err := worldmodelio.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := &activeWorld{
		geometry: reopenedModel.Geometry,
		sample:   reopenedModel.RuntimeProfile.Sample,
		reopened: reopenedModel,
	}
	reopenedReport := buildAnomalyReport(reopened)

	if !reflect.DeepEqual(generatedReport, reopenedReport) {
		t.Fatalf("anomaly report changed after reopen\ngenerated: %#v\nreopened: %#v", generatedReport, reopenedReport)
	}

	waterbedAudit := model.Waterbed.Audit(16, model.Config.Reconcile.FinalUpslopeEpsilon)
	if got := generatedReport.Counts[anomalyRiverUpslope]; got != waterbedAudit.RiverUpslopeSteps {
		t.Fatalf("river upslope count = %d, audit = %d", got, waterbedAudit.RiverUpslopeSteps)
	}
	if got := generatedReport.Counts[anomalyLakeBed]; got != waterbedAudit.LakeBedViolations {
		t.Fatalf("lake bed count = %d, audit = %d", got, waterbedAudit.LakeBedViolations)
	}
	if got := generatedReport.Counts[anomalyCoastMismatch]; got != model.Reconciliation.AppliedCoastSurfaceMismatch {
		t.Fatalf("coast mismatch count = %d, reconciliation = %d", got, model.Reconciliation.AppliedCoastSurfaceMismatch)
	}
	fallbacks, mismatches := summarizeRuntimeSemantics(model.Geometry, model.RuntimeProfile.Sample)
	if got := generatedReport.Counts[anomalyRuntimeFallback]; got != fallbacks {
		t.Fatalf("runtime fallback count = %d, macro audit = %d", got, fallbacks)
	}
	if got := generatedReport.Counts[anomalyRuntimeMismatch]; got != mismatches {
		t.Fatalf("runtime mismatch count = %d, macro audit = %d", got, mismatches)
	}
}
