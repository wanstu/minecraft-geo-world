package acceptance

import "testing"

func TestBuildBatchRequiresRepresentativePresetMatrix(t *testing.T) {
	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}

	const randomSeed int64 = 777777777777777777
	seeds := append(append([]int64(nil), FixedAcceptanceSeeds...), randomSeed)
	reports := make([]Report, 0, len(RepresentativePresets)*len(seeds))
	for presetIndex, preset := range RepresentativePresets {
		for _, seed := range seeds {
			results := make([]Result, 0, len(entries))
			for _, entry := range entries {
				status := StatusManualRequired
				if entry.Requirement == RequirementLater &&
					!laterFeatureImplemented(entry.ID) {
					status = StatusNotImplemented
				}
				result := Result{
					ID:          entry.ID,
					Name:        entry.Name,
					Requirement: entry.Requirement,
					Inspection:  entry.Inspection,
					Status:      status,
				}
				if entryUsesWorldSamples(entry) &&
					(entry.Requirement != RequirementLater || laterFeatureImplemented(entry.ID)) {
					result.Points = []Point{{TP: "/tp @s 0 80 0"}}
				}
				results = append(results, result)
			}
			reports = append(reports, Report{
				RegistryVersion:         "geoworld-acceptance-v1.1",
				Seed:                    seed,
				Preset:                  preset,
				ParameterSnapshotSHA256: string(rune('a' + presetIndex)),
				Results:                 results,
			})
		}
	}

	batch, err := BuildBatch(reports)
	if err != nil {
		t.Fatal(err)
	}
	if !batch.MatrixComplete {
		t.Fatalf("expected complete matrix, missing=%v anomalies=%v", batch.MissingCombinations, batch.Anomalies)
	}
	if batch.RandomSeed == nil || *batch.RandomSeed != randomSeed {
		t.Fatalf("random seed = %v, want %d", batch.RandomSeed, randomSeed)
	}
	param09 := batchResultByID(t, batch.Results, "PARAM-09")
	if param09.CoverageStatus != BatchCovered {
		t.Fatalf("PARAM-09 coverage = %s, want %s", param09.CoverageStatus, BatchCovered)
	}
}

func TestBuildBatchReportsMissingMatrixAndSample(t *testing.T) {
	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	results := make([]Result, 0, len(entries))
	for _, entry := range entries {
		result := Result{
			ID:          entry.ID,
			Name:        entry.Name,
			Requirement: entry.Requirement,
			Inspection:  entry.Inspection,
			Status:      StatusManualRequired,
		}
		if entry.Requirement == RequirementLater && !laterFeatureImplemented(entry.ID) {
			result.Status = StatusNotImplemented
		} else if entry.Requirement != RequirementLater &&
			entryUsesWorldSamples(entry) && entry.ID != "PLATEAU-01" {
			result.Points = []Point{{TP: "/tp @s 0 80 0"}}
		} else if entry.ID == "PLATEAU-01" {
			result.Status = StatusFail
			result.Note = "MISSING_SAMPLE"
		}
		results = append(results, result)
	}
	batch, err := BuildBatch([]Report{{
		RegistryVersion:         "geoworld-acceptance-v1.1",
		Seed:                    FixedAcceptanceSeeds[0],
		Preset:                  "standard",
		ParameterSnapshotSHA256: "snapshot",
		Results:                 results,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if batch.MatrixComplete {
		t.Fatal("single model must not satisfy representative preset matrix")
	}
	if batchResultByID(t, batch.Results, "PARAM-09").CoverageStatus != BatchMissingMatrix {
		t.Fatal("PARAM-09 should report missing matrix")
	}
	if batchResultByID(t, batch.Results, "PLATEAU-01").CoverageStatus != BatchMissingSample {
		t.Fatal("PLATEAU-01 should report missing sample")
	}
}

func batchResultByID(t *testing.T, results []BatchResult, id string) BatchResult {
	t.Helper()
	for _, result := range results {
		if result.ID == id {
			return result
		}
	}
	t.Fatalf("missing batch result %s", id)
	return BatchResult{}
}
