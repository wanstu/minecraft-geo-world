package acceptance

import (
	"testing"
)

func syntheticManualReport(t *testing.T) Report {
	t.Helper()
	entries := MustEntries()
	results := make([]Result, 0, len(entries))
	for i, entry := range entries {
		points := []Point(nil)
		if i%3 == 0 {
			points = []Point{{TP: "/tp @s 0 80 0", X: 0, Y: 80, Z: 0}}
		}
		status := StatusManualRequired
		if entry.Requirement == RequirementLater && !laterFeatureImplemented(entry.ID) {
			status = StatusNotImplemented
		}
		results = append(results, Result{
			ID: entry.ID, Name: entry.Name, Requirement: entry.Requirement,
			Inspection: entry.Inspection, Method: entry.Method, Status: status,
			Expected: entry.PassStandard, Failure: "fail", Points: points,
		})
	}
	return Report{
		RegistryVersion:         "geoworld-acceptance-v1.1",
		Seed:                    123456,
		WorldProfileVersion:     "geoworld-profile-v1.1",
		GenerationRulesVersion:  "geoworld-rules-v1.1.10",
		WorldModelVersion:       "geoworld-model-v1",
		Preset:                  "standard",
		ParameterSnapshotSHA256: "test-snapshot",
		Results:                 results,
	}
}

func TestBuildManualTableContainsEveryRegistryItemExactlyOnce(t *testing.T) {
	table, err := BuildManualTable(syntheticManualReport(t))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	count := 0
	for _, big := range table.BigCategories {
		if len(big.SmallCategories) == 0 {
			t.Fatalf("big category %s is empty", big.ID)
		}
		for _, small := range big.SmallCategories {
			for _, item := range small.Items {
				count++
				if seen[item.ID] {
					t.Fatalf("duplicate item %s", item.ID)
				}
				seen[item.ID] = true
			}
		}
	}
	if got, want := count, len(MustEntries()); got != want {
		t.Fatalf("manual item count=%d want=%d", got, want)
	}
}

func TestBuildSpotTableTakesTwoFromEveryBigCategory(t *testing.T) {
	full, err := BuildManualTable(syntheticManualReport(t))
	if err != nil {
		t.Fatal(err)
	}
	spot, err := BuildSpotTable(full, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(spot.BigCategories) != len(full.BigCategories) {
		t.Fatalf("spot big categories=%d want=%d", len(spot.BigCategories), len(full.BigCategories))
	}
	fullIDs := map[string]bool{}
	for _, big := range full.BigCategories {
		for _, small := range big.SmallCategories {
			for _, item := range small.Items {
				fullIDs[item.ID] = true
			}
		}
	}
	for _, big := range spot.BigCategories {
		if len(big.Items) != 2 {
			t.Fatalf("%s selected=%d want=2", big.ID, len(big.Items))
		}
		for _, item := range big.Items {
			if !fullIDs[item.Item.ID] {
				t.Fatalf("spot item %s does not come from full table", item.Item.ID)
			}
			if item.Item.Status == StatusNotImplemented {
				t.Fatalf("spot selected NOT_IMPLEMENTED item %s", item.Item.ID)
			}
		}
	}
}

func TestBuildSpotTableIsDeterministic(t *testing.T) {
	full, err := BuildManualTable(syntheticManualReport(t))
	if err != nil {
		t.Fatal(err)
	}
	a, err := BuildSpotTable(full, 2)
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildSpotTable(full, 2)
	if err != nil {
		t.Fatal(err)
	}
	for i := range a.BigCategories {
		for j := range a.BigCategories[i].Items {
			if a.BigCategories[i].Items[j].Item.ID != b.BigCategories[i].Items[j].Item.ID {
				t.Fatalf("selection not deterministic")
			}
		}
	}
}
