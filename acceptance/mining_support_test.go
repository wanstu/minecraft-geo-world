package acceptance

import "testing"

func TestMiningSupportPoints(t *testing.T) {
	points, err := MiningSupportPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 4 {
		t.Fatalf("points=%d want=4", len(points))
	}
	want := map[string]string{
		"MINING-01": "/tp @s -11968 274 -5184",
		"MINING-02": "/tp @s 13888 107 4544",
		"MINING-03": "/tp @s 13376 172 -1472",
		"MINING-04": "/tp @s 3648 151 8384",
	}
	for _, point := range points {
		if point.GenerationRulesVersion != "geoworld-rules-v1.1.11" {
			t.Fatalf("%s rules=%s", point.ID, point.GenerationRulesVersion)
		}
		if point.Seed != 123456 {
			t.Fatalf("%s seed=%d want=123456", point.ID, point.Seed)
		}
		if point.Requirement != RequirementCore ||
			point.Inspection != InspectionPoint {
			t.Fatalf(
				"%s requirement=%s inspection=%s",
				point.ID,
				point.Requirement,
				point.Inspection,
			)
		}
		expected, ok := want[point.ID]
		if !ok {
			t.Fatalf("unexpected point id %s", point.ID)
		}
		if point.TP() != expected {
			t.Fatalf("%s tp=%q want=%q", point.ID, point.TP(), expected)
		}
		delete(want, point.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing mining support IDs: %v", want)
	}

	base, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(base) != 177 {
		t.Fatalf("canonical v1.1 registry changed: %d", len(base))
	}
	supplement, err := SupplementalPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(supplement) != 10 {
		t.Fatalf("supplemental registry count=%d want=10", len(supplement))
	}
	settlement, err := SettlementSupportPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(settlement) != 3 {
		t.Fatalf("settlement support count=%d want=3", len(settlement))
	}
}
