package acceptance

import "testing"

func TestRegionalAdvantageSupportPoints(t *testing.T) {
	points, err := RegionalAdvantageSupportPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 6 {
		t.Fatalf("points=%d want=6", len(points))
	}
	want := map[string]string{
		"ADVANTAGE-01": "/tp @s 3648 139 7872",
		"ADVANTAGE-02": "/tp @s 10688 166 -2240",
		"ADVANTAGE-03": "/tp @s -13120 161 -6336",
		"ADVANTAGE-04": "/tp @s 7104 143 -960",
		"ADVANTAGE-05": "/tp @s 10560 160 -2240",
		"ADVANTAGE-06": "/tp @s -3520 101 -2752",
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
		t.Fatalf("missing regional advantage support IDs: %v", want)
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
	civilization, err := CivilizationSupportPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(civilization) != 5 {
		t.Fatalf("civilization support count=%d want=5", len(civilization))
	}
}
