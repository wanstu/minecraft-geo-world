package main

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"geoworld/internal/generator"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
)

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
