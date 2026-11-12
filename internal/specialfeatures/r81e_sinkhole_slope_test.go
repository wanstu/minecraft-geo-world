package specialfeatures

import (
	"math"
	"reflect"
	"testing"
)

func TestR81eSinkholeBroadSlopeReplacesFlatCenterOnlyForV125(t *testing.T) {
	feature := Feature{
		ID: 11, Kind: Sinkhole,
		CenterX: 320, CenterZ: 6592,
		RadiusX: 153.631, RadiusZ: 130.49,
		BaseHeight:  0.60437718,
		FloorHeight: 0.52937718,
		RimHeight:   0.60437718,
	}
	base := testHabitableDepressionBase()
	base.World.GroundHeight = feature.BaseHeight
	oldVersion := "geoworld-rules-v1.1.24"
	newVersion := "geoworld-rules-v1.1.25"
	sample := func(x float64, version string) float64 {
		return Apply([]Feature{feature}, base, 0.50, x, feature.CenterZ, version).World.GroundHeight
	}
	oldCenter, oldInner := sample(320, oldVersion), sample(380, oldVersion)
	newCenter, newInner := sample(320, newVersion), sample(380, newVersion)
	if math.Abs(oldInner-oldCenter) > 0.004 {
		t.Fatalf("locked v1.1.24 should have its old flat floor: center=%g at +60=%g", oldCenter, oldInner)
	}
	if newInner-newCenter < 0.014 {
		t.Fatalf("v1.1.25 new bowl not gradually rising: center=%g at +60=%g", newCenter, newInner)
	}
	if math.Abs(newCenter-oldCenter) > 0.0002 {
		t.Fatalf("new slope unexpectedly shifted the bottom: %.7f vs %.7f", newCenter, oldCenter)
	}
	maxSlope := 0.0
	last := sample(320, newVersion)
	for x := 324.0; x <= 480; x += 4 {
		got := sample(x, newVersion)
		if got > base.World.GroundHeight+1e-10 {
			t.Fatalf("sinkhole raised natural terrain at %.0f: %.7f", x, got)
		}
		rise := math.Abs(got-last) * 514 / 4
		maxSlope = math.Max(maxSlope, rise)
		last = got
	}
	if maxSlope > 1.0 {
		t.Fatalf("v1.1.25 sinkhole still has an abrupt wall: maxSlope=%.3f Y/block", maxSlope)
	}
	outside := Apply([]Feature{feature}, base, 0.50, 500, 6592, newVersion)
	if !reflect.DeepEqual(outside, base) {
		t.Fatal("sinkhole modified terrain outside its footprint")
	}
}

func TestR81eOldSinkholeVersionsRemainFlatAndOtherFeatureKindsUnchanged(t *testing.T) {
	base := testHabitableDepressionBase()
	base.World.GroundHeight = 0.66
	for _, kind := range []Kind{BasinSnow, BasinCherry, VolcanicCraterSnow, VolcanicCraterCherry} {
		feature := Feature{
			ID: 11, Kind: kind, CenterX: 320, CenterZ: 6592,
			RadiusX: 300, RadiusZ: 280, BaseHeight: 0.66,
			FloorHeight: 0.55, RimHeight: 0.66,
		}
		for _, x := range []float64{320, 380, 470, 560, 620} {
			old := Apply([]Feature{feature}, base, 0.50, x, 6592, "geoworld-rules-v1.1.24")
			newer := Apply([]Feature{feature}, base, 0.50, x, 6592, "geoworld-rules-v1.1.25")
			if !reflect.DeepEqual(old, newer) {
				t.Fatalf("v125 changed unrelated feature kind %s at x=%f", kind, x)
			}
		}
	}
	for _, version := range []string{"geoworld-rules-v1.1.22", "geoworld-rules-v1.1.23", "geoworld-rules-v1.1.24"} {
		if usesNaturalSinkholeSlope(version) {
			t.Fatalf("old version %s unexpectedly gained new sinkhole slope", version)
		}
	}
	if !usesNaturalSinkholeSlope("geoworld-rules-v1.1.25") ||
		!usesRiverSafeSinkholeSites("geoworld-rules-v1.1.25") {
		t.Fatal("new version must keep river-safe placement AND new geometry")
	}
}
