package main

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"geoworld/internal/specialfeatures"
	"geoworld/internal/waterbed"
	"geoworld/internal/worldmodelio"
)

func TestR81TerminalCategories(t *testing.T) {
	cases := []struct {
		segment waterbed.RiverSegment
		want    string
	}{
		{waterbed.RiverSegment{OceanMouth: true}, "ocean"},
		{waterbed.RiverSegment{LakeMouth: true}, "lake"},
		{waterbed.RiverSegment{}, "unclassified"},
	}
	for _, c := range cases {
		if got := terminalCategory(c.segment); got != c.want {
			t.Errorf("category %q want %q", got, c.want)
		}
	}
}

func TestR81MinecraftNormalizedHeightConversion(t *testing.T) {
	if y := projectedY(0.5); y != 63 {
		t.Fatalf("sea water level %d want 63", y)
	}
	if y := projectedGroundY(.5); y != 63 {
		t.Fatalf("sea ground level %d want 63", y)
	}
	if y := projectedY(.54675); y <= 63 {
		t.Fatalf("river above sea unexpected y=%d", y)
	}
	if projectedY(.54675) < projectedY(.52) {
		t.Fatal("mapping not monotonic")
	}
}

func TestR81FixedCasesPreserveExactSignedSeed(t *testing.T) {
	data, err := os.ReadFile("../../docs/acceptance/r81_hydrology_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture Fixture
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	const wanted int64 = -1818694639628404530
	if fixture.Seed != wanted {
		t.Fatalf("seed precision loss: %d want %d", fixture.Seed, wanted)
	}
	if len(fixture.Cases) != 5 {
		t.Fatalf("fixed cases=%d want=5", len(fixture.Cases))
	}
	if fixture.RulesVersion != "geoworld-rules-v1.1.22" {
		t.Fatalf("unexpected rules %q", fixture.RulesVersion)
	}
	m := &worldmodelio.RuntimeModel{}
	m.Manifest.Seed = wanted
	m.Manifest.GenerationRulesVersion = fixture.RulesVersion
	if err := validateCases(fixture, m); err != nil {
		t.Fatal(err)
	}
	fixture.Seed = 123456
	if err := validateCases(fixture, m); err == nil {
		t.Fatal("incompatible seed did not fail closed")
	}
}

func TestR81DuplicateCaseAndInvalidCoordinatesRejected(t *testing.T) {
	m := &worldmodelio.RuntimeModel{}
	m.Manifest.Seed = 8
	m.Manifest.GenerationRulesVersion = "r8"
	f := Fixture{SchemaVersion: 1, Seed: 8, RulesVersion: "r8", Cases: []Case{{ID: "a", X: 5, Z: 6}, {ID: "a", X: 7, Z: 8}}}
	if err := validateCases(f, m); err == nil {
		t.Fatal("duplicate case not rejected")
	}
	f.Cases = f.Cases[:1]
	f.Cases[0].X = math.NaN()
	if err := validateCases(f, m); err == nil {
		t.Fatal("non-finite location not rejected")
	}
}

func TestR81NoOverconfidentPassOnModelOnlyReport(t *testing.T) {
	m := &worldmodelio.RuntimeModel{Waterbed: &waterbed.Model{}}
	m.Manifest.Seed = 5
	m.Manifest.GenerationRulesVersion = "r7"
	f := Fixture{SchemaVersion: 1, Seed: 5, RulesVersion: "r7"}
	result, err := audit(m, f, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result.EvidenceClass != evidenceClass {
		t.Fatalf("unexpected class %q", result.EvidenceClass)
	}
	if result.Counts.RiverSegments != 0 || result.Counts.TerminalSegments != 0 {
		t.Fatalf("empty river graph fabricated counts %+v", result.Counts)
	}
	if len(result.Limitation) == 0 {
		t.Fatal("missing model-only limitation")
	}
}

func TestR81SinkholeIntersectionSignalIsNotAnOutletProof(t *testing.T) {
	m := &worldmodelio.RuntimeModel{Waterbed: &waterbed.Model{Rivers: []waterbed.RiverSegment{
		{From: 1, To: 2, X0: 0, Z0: 0, X1: 128, Z1: 0, HalfWidth: 8},
	}}}
	sf := specialfeatures.Feature{ID: 9, Kind: specialfeatures.Sinkhole, CenterX: 64, CenterZ: 4, RadiusX: 70, RadiusZ: 35}
	got := sinkhole(m, sf)
	if !got.RiverIntersectsFootprint || got.HasPersistedDrainage {
		t.Fatalf("unexpected unplanned sinkhole overlap %+v", got)
	}
	sf.CenterZ = 400
	got = sinkhole(m, sf)
	if got.RiverIntersectsFootprint {
		t.Fatalf("distant sinkhole intersects river %+v", got)
	}
}

func TestR81RiverOceanBoundaryOneBlockRiseDetected(t *testing.T) {
	y62, y63, y59, y64 := 62, 63, 59, 64
	for _, tc := range []struct {
		name  string
		pts   []Point
		found bool
		rise  int
	}{
		{"one-block-water-wall", []Point{{Kind: "river", WaterY: &y62}, {Kind: "ocean", WaterY: &y63}}, true, 1},
		{"four-block-water-wall", []Point{{Kind: "river", WaterY: &y59}, {Kind: "ocean", WaterY: &y63}}, true, 4},
		{"downward-handoff", []Point{{Kind: "river", WaterY: &y64}, {Kind: "ocean", WaterY: &y63}}, true, -1},
		{"no-transition", []Point{{Kind: "river", WaterY: &y62}, {Kind: "river", WaterY: &y63}}, false, 0},
		{"dry-gap-not-fabricated", []Point{{Kind: "river", WaterY: &y62}, {Kind: "none"}, {Kind: "ocean", WaterY: &y63}}, false, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, _, rise := detectRiverOceanBoundary(tc.pts)
			if got != tc.found || rise != tc.rise {
				t.Fatalf("got found=%v rise=%d want found=%v rise=%d", got, rise, tc.found, tc.rise)
			}
		})
	}
}
