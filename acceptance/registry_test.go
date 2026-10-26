package acceptance

import "testing"

func TestRegistryIsCompleteAndStable(t *testing.T) {
	entries, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 177 {
		t.Fatalf("entries=%d want=177", len(entries))
	}
	want := map[string]RequirementKind{
		"CORE-01":    RequirementConstraint,
		"RIVER-04":   RequirementCore,
		"FLOOD-01":   RequirementLater,
		"RARE-01":    RequirementLater,
		"MODEL-08":   RequirementConstraint,
		"PROFILE-08": RequirementConstraint,
		"PARAM-09":   RequirementConstraint,
		"CLIMATE-08": RequirementCore,
	}
	found := make(map[string]RequirementKind, len(entries))
	for _, entry := range entries {
		found[entry.ID] = entry.Requirement
	}
	for id, requirement := range want {
		if found[id] != requirement {
			t.Fatalf("%s requirement=%q want=%q", id, found[id], requirement)
		}
	}
}
