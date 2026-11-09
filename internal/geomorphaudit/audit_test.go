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
	if report.Transition.MountainPlainLowBufferEdges >
		report.Transition.DirectMountainPlainEdges {
		t.Fatalf(
			"low-buffer mountain/plain edges exceed direct edges: low=%d direct=%d",
			report.Transition.MountainPlainLowBufferEdges,
			report.Transition.DirectMountainPlainEdges,
		)
	}
	if report.Transition.MountainPlainLowBufferEdgeFraction < 0 ||
		report.Transition.MountainPlainLowBufferEdgeFraction > 1 {
		t.Fatalf(
			"mountain/plain low-buffer fraction out of range: %g",
			report.Transition.MountainPlainLowBufferEdgeFraction,
		)
	}
	if report.Sediment.Production <= 0 {
		t.Fatal("expected actual erosion to produce sediment proxy mass")
	}
	massTolerance := report.Sediment.Production * 1e-10
	if massTolerance < 1e-12 {
		massTolerance = 1e-12
	}
	if report.Sediment.MassBalanceError > massTolerance {
		t.Fatalf(
			"sediment budget is not conserved: production=%g deposition=%g export=%g error=%g",
			report.Sediment.Production,
			report.Sediment.Deposition,
			report.Sediment.Export,
			report.Sediment.MassBalanceError,
		)
	}
	if report.Sediment.HillslopeMassBalanceError > massTolerance {
		t.Fatalf(
			"hillslope sediment budget is not conserved: error=%g tolerance=%g",
			report.Sediment.HillslopeMassBalanceError,
			massTolerance,
		)
	}
	if report.Sediment.NetworkMassBalanceError > massTolerance {
		t.Fatalf(
			"network sediment budget is not conserved: error=%g tolerance=%g",
			report.Sediment.NetworkMassBalanceError,
			massTolerance,
		)
	}
	if report.Sediment.DepositionPartitionError > massTolerance {
		t.Fatalf(
			"sediment deposition partition is not conserved: error=%g tolerance=%g",
			report.Sediment.DepositionPartitionError,
			massTolerance,
		)
	}
	if report.Sediment.OceanReceiverLoad > report.Sediment.Export+massTolerance {
		t.Fatalf(
			"ocean receiver load exceeds terminal export: ocean=%g export=%g",
			report.Sediment.OceanReceiverLoad,
			report.Sediment.Export,
		)
	}
	if report.Sediment.HillslopeToNetworkDelivery <= 0 ||
		report.Sediment.NetworkActiveCells == 0 {
		t.Fatalf(
			"expected routed sediment to reach an active drainage network: delivery=%g active=%d",
			report.Sediment.HillslopeToNetworkDelivery,
			report.Sediment.NetworkActiveCells,
		)
	}
	if _, err := json.Marshal(report); err != nil {
		t.Fatalf("report must remain JSON-safe: %v", err)
	}

	t.Logf(
		"sediment: production=%.6f deposition=%.6f export=%.6f depositedFraction=%.3f exportFraction=%.3f lakeFraction=%.3f maxFlux=%.6f active=%d capacity=%.3f",
		report.Sediment.Production,
		report.Sediment.Deposition,
		report.Sediment.Export,
		report.Sediment.DepositedFraction,
		report.Sediment.ExportFraction,
		report.Sediment.LakeDepositionFraction,
		report.Sediment.MaxFlux,
		report.Sediment.ActiveCells,
		report.Sediment.MeanTransportCapacity,
	)
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
