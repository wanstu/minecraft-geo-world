package acceptance

import (
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"geoworld/internal/waterbed"
	"geoworld/internal/worldmodelio"
)

func TestPROFILE05AcceptanceUsesLockedProfileAndDeterministicCandidates(t *testing.T) {
	path := filepath.Join("..", "internal", "goldenvectors", "testdata", "runtime-v1.geoworld")
	firstModel, err := worldmodelio.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	secondModel, err := worldmodelio.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	first, err := Plan(firstModel)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Plan(secondModel)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("acceptance candidates changed after reopening the same locked model")
	}

	if first.Seed != firstModel.Manifest.Seed ||
		first.WorldProfileVersion != firstModel.Manifest.WorldProfileVersion ||
		first.GenerationRulesVersion != firstModel.Manifest.GenerationRulesVersion ||
		first.WorldModelVersion != firstModel.Manifest.WorldModelVersion ||
		first.Preset != firstModel.Manifest.Preset ||
		first.ParameterSnapshotSHA256 != firstModel.Manifest.ParameterSnapshotSHA256 {
		t.Fatalf(
			"acceptance identity diverged from locked model: report=%+v manifest=%+v",
			first,
			firstModel.Manifest,
		)
	}

	for _, result := range first.Results {
		for _, point := range result.Points {
			if point.TP == "" || point.Reason == "" {
				t.Fatalf("%s contains an ungrounded acceptance candidate: %+v", result.ID, point)
			}
		}
	}
}

func TestAcceptanceReportCoversEveryRegisteredID(t *testing.T) {
	model, err := worldmodelio.Load(
		filepath.Join("..", "internal", "goldenvectors", "testdata", "runtime-v1.geoworld"),
	)
	if err != nil {
		t.Fatal(err)
	}
	report, err := Plan(model)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 177 {
		t.Fatalf("results=%d want=177", len(report.Results))
	}
	seen := make(map[string]bool, len(report.Results))
	outerCandidate := false
	for _, result := range report.Results {
		if seen[result.ID] {
			t.Fatalf("duplicate result %s", result.ID)
		}
		seen[result.ID] = true
		if result.Requirement == RequirementLater {
			if laterFeatureImplemented(result.ID) {
				if result.Status == StatusNotImplemented {
					t.Fatalf("%s implemented later feature is still NOT_IMPLEMENTED", result.ID)
				}
			} else if result.Status != StatusNotImplemented {
				t.Fatalf("%s unimplemented later status=%s", result.ID, result.Status)
			}
		}
		if result.Requirement != RequirementLater &&
			result.Status == StatusNotImplemented {
			t.Fatalf("%s non-later silently not implemented", result.ID)
		}
		for _, point := range result.Points {
			if point.TP == "" || point.Y < -56 || point.Y > 328 {
				t.Fatalf("%s invalid TP point: %+v", result.ID, point)
			}
			if result.ID == "EDGE-03" &&
				point.Reason == "deterministic sparse island outside finite mainland model" {
				outerCandidate = true
			}
		}
	}
	if !outerCandidate {
		t.Fatal("EDGE acceptance report has no deterministic sparse outer-island candidate")
	}
}

func TestApplyMacro05EvidenceUpdatesOnlyMacro05(t *testing.T) {
	base := Report{
		Seed: 123456,
		Results: []Result{
			{ID: "MACRO-04", Status: StatusManualRequired},
			{ID: "MACRO-05", Status: StatusManualRequired},
		},
	}
	passed := ApplyMacro05Evidence(base, Macro05Evidence{
		PotentialCells:            19000,
		RoleCoverage:              4,
		CoreTripleOverlapFraction: 0.08,
		CoreTripleOverlapLimit:    0.15,
		Satisfied:                 true,
	})
	if passed.Results[0].Status != StatusManualRequired {
		t.Fatalf("unrelated result changed: %+v", passed.Results[0])
	}
	if passed.Results[1].Status != StatusPass ||
		passed.Results[1].Observe == "" ||
		passed.Results[1].Note == "" {
		t.Fatalf("MACRO-05 pass evidence not applied: %+v", passed.Results[1])
	}
	if base.Results[1].Status != StatusManualRequired {
		t.Fatal("ApplyMacro05Evidence mutated the source report")
	}

	failed := ApplyMacro05Evidence(base, Macro05Evidence{
		PotentialCells:            19000,
		RoleCoverage:              2,
		CoreTripleOverlapFraction: 0.22,
		CoreTripleOverlapLimit:    0.15,
		Satisfied:                 false,
	})
	if failed.Results[1].Status != StatusFail {
		t.Fatalf("MACRO-05 failure evidence not applied: %+v", failed.Results[1])
	}
}

func TestApplyMacro06EvidenceUpdatesOnlyMacro06(t *testing.T) {
	base := Report{
		Seed: 123456,
		Results: []Result{
			{ID: "MACRO-05", Status: StatusManualRequired},
			{ID: "MACRO-06", Status: StatusManualRequired},
		},
	}
	passed := ApplyMacro06Evidence(base, Macro06Evidence{
		MinSpanBlocks:            25472,
		MaxScaleBlocks:           5376,
		MaxScaleName:             "phase1_macro_transition",
		ScaleCapacity:            25472.0 / 5376.0,
		InteriorFraction:         1 - 2*5376.0/25472.0,
		RecommendedMinSpanBlocks: 21504,
		ShortfallBlocks:          0,
		Satisfied:                true,
	})
	if passed.Results[0].Status != StatusManualRequired {
		t.Fatalf("unrelated result changed: %+v", passed.Results[0])
	}
	if passed.Results[1].Status != StatusPass ||
		passed.Results[1].Observe == "" ||
		passed.Results[1].Note == "" {
		t.Fatalf("MACRO-06 pass evidence not applied: %+v", passed.Results[1])
	}
	if base.Results[1].Status != StatusManualRequired {
		t.Fatal("ApplyMacro06Evidence mutated the source report")
	}

	failed := ApplyMacro06Evidence(base, Macro06Evidence{
		MinSpanBlocks:            16000,
		MaxScaleBlocks:           5376,
		MaxScaleName:             "phase1_macro_transition",
		ScaleCapacity:            16000.0 / 5376.0,
		InteriorFraction:         1 - 2*5376.0/16000.0,
		RecommendedMinSpanBlocks: 21504,
		ShortfallBlocks:          5504,
		Satisfied:                false,
	})
	if failed.Results[1].Status != StatusFail {
		t.Fatalf("MACRO-06 failure evidence not applied: %+v", failed.Results[1])
	}
}

func TestApplyNodeEvidenceUpdatesNodeAcceptance(t *testing.T) {
	base := Report{
		Seed: 123456,
		Results: []Result{
			{ID: "MACRO-06", Status: StatusManualRequired},
			{ID: "NODE-01", Status: StatusManualRequired},
			{ID: "NODE-02", Status: StatusManualRequired},
			{ID: "NODE-03", Status: StatusManualRequired},
			{ID: "NODE-04", Status: StatusManualRequired},
			{ID: "NODE-05", Status: StatusManualRequired},
			{ID: "NODE-06", Status: StatusManualRequired},
		},
	}
	passed := ApplyNodeEvidence(base, NodeEvidence{
		TotalCells:        64000,
		TotalNodes:        72,
		ConfluenceNodes:   3,
		LakeOutletNodes:   2,
		EstuaryNodes:      2,
		MountainPassNodes: 5,
		StraitNodes:       1,
		IsthmusNodes:      1,
		NodeFraction:      72.0 / 64000.0,
		NodeFractionLimit: 0.01,
	})
	if passed.Results[0].Status != StatusManualRequired {
		t.Fatalf("unrelated result changed: %+v", passed.Results[0])
	}
	for i := 1; i < len(passed.Results); i++ {
		if passed.Results[i].Status != StatusPass ||
			passed.Results[i].Observe == "" ||
			passed.Results[i].Note == "" {
			t.Fatalf("%s pass evidence not applied: %+v",
				passed.Results[i].ID, passed.Results[i])
		}
	}
	if base.Results[1].Status != StatusManualRequired {
		t.Fatal("ApplyNodeEvidence mutated the source report")
	}

	failed := ApplyNodeEvidence(base, NodeEvidence{
		TotalCells:        1000,
		TotalNodes:        20,
		ConfluenceNodes:   0,
		LakeOutletNodes:   1,
		EstuaryNodes:      1,
		MountainPassNodes: 1,
		StraitNodes:       0,
		IsthmusNodes:      0,
		NodeFraction:      0.02,
		NodeFractionLimit: 0.01,
	})
	if failed.Results[1].Status != StatusFail {
		t.Fatalf("NODE-01 missing confluence should fail: %+v", failed.Results[1])
	}
	if failed.Results[5].Status != StatusFail {
		t.Fatalf("NODE-05 missing strait/isthmus should fail: %+v", failed.Results[5])
	}
	if failed.Results[6].Status != StatusFail {
		t.Fatalf("NODE-06 dense nodes should fail: %+v", failed.Results[6])
	}
}

func TestApplyNodeEvidenceUsesDefaultScarcityLimit(t *testing.T) {
	report := Report{Results: []Result{
		{ID: "NODE-06", Status: StatusManualRequired},
	}}
	result := ApplyNodeEvidence(report, NodeEvidence{
		TotalCells:   10000,
		TotalNodes:   50,
		NodeFraction: 0.005,
	})
	if result.Results[0].Status != StatusPass {
		t.Fatalf("default 1%% node fraction limit not applied: %+v", result.Results[0])
	}
}

func TestApplyCorridorEvidenceUpdatesOnlyCorridors(t *testing.T) {
	base := Report{
		Results: []Result{
			{ID: "NODE-06", Status: StatusManualRequired},
			{ID: "CORRIDOR-01", Status: StatusManualRequired},
			{ID: "CORRIDOR-02", Status: StatusManualRequired},
			{ID: "CORRIDOR-03", Status: StatusManualRequired},
			{ID: "CORRIDOR-04", Status: StatusManualRequired},
			{ID: "CORRIDOR-05", Status: StatusManualRequired},
			{ID: "CORRIDOR-06", Status: StatusManualRequired},
		},
	}
	classes := map[string]CorridorClassEvidence{
		"river_valley":   {Routes: 8, CompleteRoutes: 8, LongestCompleteBlocks: 17000},
		"hill_pass":      {Routes: 8, CompleteRoutes: 8, LongestCompleteBlocks: 4200},
		"mountain_pass":  {Routes: 8, CompleteRoutes: 3, LongestCompleteBlocks: 11900},
		"plateau_access": {Routes: 8, CompleteRoutes: 3, LongestCompleteBlocks: 11000},
		"basin_gateway":  {Routes: 8, CompleteRoutes: 8, LongestCompleteBlocks: 9000},
		"low_divide":     {Routes: 8, CompleteRoutes: 8, LongestCompleteBlocks: 10000},
	}
	passed := ApplyCorridorEvidence(base, CorridorEvidence{Classes: classes})
	if passed.Results[0].Status != StatusManualRequired {
		t.Fatalf("unrelated result changed: %+v", passed.Results[0])
	}
	for i := 1; i < len(passed.Results); i++ {
		if passed.Results[i].Status != StatusPass ||
			passed.Results[i].Observe == "" ||
			passed.Results[i].Note == "" {
			t.Fatalf("%s pass evidence not applied: %+v",
				passed.Results[i].ID, passed.Results[i])
		}
	}
	if base.Results[1].Status != StatusManualRequired {
		t.Fatal("ApplyCorridorEvidence mutated the source report")
	}

	failedClasses := make(map[string]CorridorClassEvidence, len(classes))
	for class, item := range classes {
		failedClasses[class] = item
	}
	failedClasses["mountain_pass"] = CorridorClassEvidence{
		Routes:                8,
		CompleteRoutes:        0,
		LongestCompleteBlocks: 0,
	}
	failed := ApplyCorridorEvidence(
		base,
		CorridorEvidence{Classes: failedClasses},
	)
	if failed.Results[3].Status != StatusFail {
		t.Fatalf("CORRIDOR-03 missing complete route should fail: %+v", failed.Results[3])
	}
}

func TestRiverAcceptanceStationFollowsRenderedMeanderCenterline(t *testing.T) {
	seg := waterbed.RiverSegment{
		X0:            0,
		Z0:            0,
		X1:            100,
		Z1:            0,
		MeanderOffset: 20,
	}
	x, z := riverCenterPoint(seg, 0.5)
	if math.Abs(x-50) > 1e-12 || math.Abs(z-20) > 1e-12 {
		t.Fatalf("centerline=(%.12f, %.12f), want=(50, 20)", x, z)
	}

	x, z = riverCenterPoint(seg, 0)
	if math.Abs(x) > 1e-12 || math.Abs(z) > 1e-12 {
		t.Fatalf("start=(%.12f, %.12f), want=(0, 0)", x, z)
	}

	x, z = riverCenterPoint(seg, 1)
	if math.Abs(x-100) > 1e-12 || math.Abs(z) > 1e-12 {
		t.Fatalf("end=(%.12f, %.12f), want=(100, 0)", x, z)
	}
}
