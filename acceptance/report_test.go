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
