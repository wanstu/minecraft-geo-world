package specialfeatures

import (
	"testing"

	"geoworld/internal/waterbed"
)

func TestR81dSinkholeConflictsAuthoritativeRiver(t *testing.T) {
	pit := Feature{Kind: Sinkhole, CenterX: 0, CenterZ: 0, RadiusX: 150, RadiusZ: 130}
	cases := []struct {
		name  string
		river waterbed.RiverSegment
		want  bool
	}{
		{"river through center", waterbed.RiverSegment{X0: 0, Z0: -250, X1: 0, Z1: 250, HalfWidth: 8}, true},
		{"river bank close to edge", waterbed.RiverSegment{X0: 172, Z0: -250, X1: 172, Z1: 250, HalfWidth: 8}, true},
		{"river with safe clearance", waterbed.RiverSegment{X0: 205, Z0: -250, X1: 205, Z1: 250, HalfWidth: 8}, false},
		{"far away river", waterbed.RiverSegment{X0: 400, Z0: 450, X1: 600, Z1: 550, HalfWidth: 12}, false},
		{"curved segment crossing near pit", waterbed.RiverSegment{X0: -180, Z0: -200, X1: 180, Z1: 200, MeanderOffset: 40, HalfWidth: 8}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sinkholeConflictsRiver(pit, []waterbed.RiverSegment{tc.river})
			if got != tc.want {
				t.Fatalf("conflicts=%t want %t", got, tc.want)
			}
		})
	}
}

func TestR81dSinkholeSiteVetoOnlyInV124(t *testing.T) {
	if !usesRiverSafeSinkholeSites("geoworld-rules-v1.1.24") {
		t.Fatal("new sinkhole site veto is not enabled")
	}
	for _, version := range []string{
		"geoworld-rules-v1.1.22",
		"geoworld-rules-v1.1.23",
		"geoworld-rules-v1.1.20",
		"",
	} {
		if usesRiverSafeSinkholeSites(version) {
			t.Errorf("historical rules %q unexpectedly adopted new sinkhole placement", version)
		}
	}
}

func TestR81dSiteVetoDoesNotAffectOtherFeatureKinds(t *testing.T) {
	r := []waterbed.RiverSegment{{X0: -200, Z0: 0, X1: 200, Z1: 0, HalfWidth: 10}}
	for _, kind := range []Kind{BasinSnow, BasinCherry, VolcanicCraterSnow, MushroomIsland} {
		feature := Feature{Kind: kind, CenterX: 0, CenterZ: 0, RadiusX: 100, RadiusZ: 100}
		if sinkholeConflictsRiver(feature, r) {
			t.Fatalf("river-safe sinkhole check accidentally applied to kind %s", kind)
		}
	}
}
