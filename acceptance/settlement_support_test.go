package acceptance

import "testing"

func TestSettlementSupportPoints(t *testing.T) {
	points, err := SettlementSupportPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 3 {
		t.Fatalf("points=%d want=3", len(points))
	}

	want := map[string]string{
		"SETTLEMENT-01": "/tp @s 10560 160 -2240",
		"SETTLEMENT-02": "/tp @s 2752 130 -2880",
		"SETTLEMENT-03": "/tp @s -12480 184 -7488",
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
		t.Fatalf("missing settlement support IDs: %v", want)
	}

	base, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(base) != 177 {
		t.Fatalf("canonical v1.1 registry changed: %d", len(base))
	}
	special, err := SupplementalPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(special) != 10 {
		t.Fatalf("supplemental registry count=%d want=10", len(special))
	}
}
