package geomorphaudit

import (
	"context"
	"encoding/json"
	"testing"

	"geoworld/internal/generator"
	"geoworld/internal/world"
)

func TestAnalyzeGeneratedWorldIsReadOnlyAndJSONSafe(t *testing.T) {
	g := world.GridGeometry{
		Width:       96,
		Height:      60,
		OriginX:     -95 * 64,
		OriginZ:     -59 * 64,
		SpanXBlocks: 95 * 128,
		SpanZBlocks: 59 * 128,
	}
	model, err := generator.GenerateWorld(context.Background(), generator.Config{
		Seed:                   123456,
		Geometry:               g,
		PlateCount:             18,
		OceanFraction:          0.68,
		ContinentFragmentation: 0.35,
		ContinentSeparation:    0.65,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	report, err := Analyze(model, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if report.EvidenceClass != EvidenceClassInvestigation {
		t.Fatalf("evidence class=%q", report.EvidenceClass)
	}
	if report.River.Segments != len(model.Waterbed.Rivers) {
		t.Fatalf("segments=%d want=%d", report.River.Segments, len(model.Waterbed.Rivers))
	}
	if report.River.Routes == 0 {
		t.Fatal("expected at least one source-to-sink river route")
	}
	if report.Transition.Samples == 0 || len(report.Transition.Classes) == 0 {
		t.Fatal("expected terrain transition samples")
	}
	if _, err := json.Marshal(report); err != nil {
		t.Fatalf("report must remain JSON-safe: %v", err)
	}

	t.Logf(
		"signals: routes=%d traversed=%d unique=%d internal=%d receiver=%d internal>=0.25=%d internal>=1=%d maxInternal=%.4f legacyDownstreamSteeper=%d longFree=%d broadSteepening=%d outlets=%d longDrainages=%d broadDrainages=%d confluences=%d confluenceRises=%d widthRegressions=%d mouths=%d mouthRises=%d mouthNarrow=%d transitions=%d",
		report.River.Routes,
		report.River.TraversedSurfaceRiseSteps,
		report.River.UniqueSurfaceRiseSteps,
		report.River.UniqueInternalRiseSteps,
		report.River.UniqueReceiverRiseSteps,
		report.River.InternalRiseAboveQuarterBlock,
		report.River.InternalRiseAboveOneBlock,
		report.River.MaxInternalRiseBlocks,
		report.River.DownstreamSteeperRoutes,
		report.River.LongFreeFlowRoutes,
		report.River.BroadDownstreamSteepeningSignals,
		report.River.DrainageOutlets,
		report.River.LongRepresentativeDrainages,
		report.River.BroadDownstreamSteepeningDrainages,
		report.River.Confluences,
		report.River.ConfluenceSurfaceRises,
		report.River.ConfluenceWidthRegressions,
		report.River.OceanMouths,
		report.River.MouthApproachRises,
		report.River.MouthNarrowingSignals,
		report.Transition.Samples,
	)
}

func TestAnalyzeRejectsIncompleteWorld(t *testing.T) {
	if _, err := Analyze(nil, DefaultConfig()); err == nil {
		t.Fatal("expected incomplete-world error")
	}
}
