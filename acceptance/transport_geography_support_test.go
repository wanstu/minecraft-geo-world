package acceptance

import "testing"

func TestTransportGeographySupportPoints(t *testing.T) {
	points, err := TransportGeographySupportPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 10 {
		t.Fatalf("points=%d want=10", len(points))
	}
	want := map[string]string{
		"TRANSPORT-01": "/tp @s 8896 142 -832",
		"TRANSPORT-02": "/tp @s -12224 201 -7744",
		"TRANSPORT-03": "/tp @s 11584 167 -1600",
		"TRANSPORT-04": "/tp @s -14016 138 -5952",
		"TRANSPORT-05": "/tp @s -14912 149 5440",
		"TRANSPORT-06": "/tp @s -10176 87 -5184",
		"TRANSPORT-07": "/tp @s 7616 96 -9152",
		"TRANSPORT-08": "/tp @s -3264 107 -2880",
		"TRANSPORT-09": "/tp @s 4032 133 -1728",
		"TRANSPORT-10": "/tp @s 3776 131 -1984",
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
		if point.Y < -56 || point.Y > 320 {
			t.Fatalf("%s y=%d outside Minecraft 26.3 acceptance range", point.ID, point.Y)
		}
		delete(want, point.ID)
	}
	if len(want) != 0 {
		t.Fatalf("missing transport geography support IDs: %v", want)
	}

	base, err := Entries()
	if err != nil {
		t.Fatal(err)
	}
	if len(base) != 177 {
		t.Fatalf("canonical v1.1 registry changed: %d", len(base))
	}
	regional, err := RegionalAdvantageSupportPoints()
	if err != nil {
		t.Fatal(err)
	}
	if len(regional) != 6 {
		t.Fatalf("regional advantage support count=%d want=6", len(regional))
	}
}
