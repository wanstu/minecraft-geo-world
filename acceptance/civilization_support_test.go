package acceptance

import "testing"

func TestCivilizationSupportPoints(t *testing.T) {
	points, err := CivilizationSupportPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 5 {
		t.Fatalf("points=%d want=5", len(points))
	}
	want := map[string]string{
		"CIVILIZATION-01": "/tp @s 10560 160 -2240",
		"CIVILIZATION-02": "/tp @s 6464 99 10944",
		"CIVILIZATION-03": "/tp @s 6080 143 -4672",
		"CIVILIZATION-04": "/tp @s 8896 141 -192",
		"CIVILIZATION-05": "/tp @s 2496 102 -6464",
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
		t.Fatalf("missing civilization support IDs: %v", want)
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
	mining, err := MiningSupportPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(mining) != 4 {
		t.Fatalf("mining support count=%d want=4", len(mining))
	}
}
