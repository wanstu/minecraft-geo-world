package acceptance

import "testing"

func TestSupplementalPointsV111(t *testing.T) {
	points, err := SupplementalPointsForRules("geoworld-rules-v1.1.11")
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 10 {
		t.Fatalf("points=%d want=10", len(points))
	}

	want := map[string]string{
		"SPECIAL-01":    "/tp @s -1088 160 1856",
		"SPECIAL-02":    "/tp @s -4800 160 3008",
		"SPECIAL-03":    "/tp @s -5312 160 -1728",
		"SPECIAL-04":    "/tp @s 1984 160 -576",
		"SPECIAL-05":    "/tp @s -1088 160 -1344",
		"SPECIAL-06":    "/tp @s 4544 160 -1088",
		"NAVIGATION-01": "/tp @s 8032 140 -711",
		"NAVIGATION-02": "/tp @s 5957 107 3360",
		"NAVIGATION-03": "/tp @s 3520 140 -2624",
		"NAVIGATION-04": "/tp @s -14272 108 -7616",
	}
	for _, point := range points {
		if point.Seed != 123456 {
			t.Fatalf("%s seed=%d want=123456", point.ID, point.Seed)
		}
		if point.Requirement != RequirementCore {
			t.Fatalf("%s requirement=%s want=%s", point.ID, point.Requirement, RequirementCore)
		}
		if point.Inspection != InspectionPoint {
			t.Fatalf("%s inspection=%s want=%s", point.ID, point.Inspection, InspectionPoint)
		}
		expectedTP, ok := want[point.ID]
		if !ok {
			t.Fatalf("unexpected point id %s", point.ID)
		}
		if point.TP() != expectedTP {
			t.Fatalf("%s tp=%q want=%q", point.ID, point.TP(), expectedTP)
		}
		delete(want, point.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing supplemental IDs: %v", want)
	}

	base, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(base) != 177 {
		t.Fatalf("canonical v1.1 registry changed: %d", len(base))
	}
}
