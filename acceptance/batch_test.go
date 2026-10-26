package acceptance

import "testing"

func TestBuildBatchRequiresRepresentativePresetMatrix(t *testing.T) {
	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}

	const randomSeed int64 = 777777777777777777
	seeds := append(append([]int64(nil), FixedAcceptanceSeeds...), randomSeed)
	profiles := RequiredAcceptanceProfiles()
	reports := make([]Report, 0, len(profiles)*len(seeds))
	for profileIndex, profile := range profiles {
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
				WorldProfileVersion:     "geoworld-profile-v1.1",
				GenerationRulesVersion:  "geoworld-rules-test",
				WorldModelVersion:       "geoworld-model-v1",
				Preset:                  profile,
				ParameterSnapshotSHA256: string(rune('a' + profileIndex)),
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

func TestBuildBatchRequiresScaleStressProfiles(t *testing.T) {
	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}

	const randomSeed int64 = 888888888888888888
	seeds := append(append([]int64(nil), FixedAcceptanceSeeds...), randomSeed)
	profiles := []string{
		"standard",
		"natural",
		"civilization",
		"epic",
		"stress_large",
	}
	reports := make([]Report, 0, len(profiles)*len(seeds))
	for profileIndex, profile := range profiles {
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
					(entry.Requirement != RequirementLater ||
						laterFeatureImplemented(entry.ID)) {
					result.Points = []Point{{TP: "/tp @s 0 80 0"}}
				}
				results = append(results, result)
			}
			reports = append(reports, Report{
				RegistryVersion:         "geoworld-acceptance-v1.1",
				Seed:                    seed,
				WorldProfileVersion:     "geoworld-profile-v1.1",
				GenerationRulesVersion:  "geoworld-rules-test",
				WorldModelVersion:       "geoworld-model-v1",
				Preset:                  profile,
				ParameterSnapshotSHA256: string(rune('a' + profileIndex)),
				Results:                 results,
			})
		}
	}

	batch, err := BuildBatch(reports)
	if err != nil {
		t.Fatal(err)
	}
	if batch.MatrixComplete {
		t.Fatal("matrix without stress_small must not be complete")
	}
	wantMissing := "stress_small/123456"
	found := false
	for _, missing := range batch.MissingCombinations {
		if missing == wantMissing {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("missing combinations do not include %q: %v", wantMissing, batch.MissingCombinations)
	}
	if got := batchResultByID(t, batch.Results, "PARAM-09").CoverageStatus; got != BatchMissingMatrix {
		t.Fatalf("PARAM-09 coverage=%s want=%s", got, BatchMissingMatrix)
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
