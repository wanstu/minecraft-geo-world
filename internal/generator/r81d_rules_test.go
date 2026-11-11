package generator

import (
	"reflect"
	"testing"
)

// R8.1d only changes the selection of new sinkholes. The hydrology,
// waterbed, terrain and coastal semantics of the last locked rules stay
// identical; the old version is still a supported persisted world.
func TestR81dV124KeepsV123MacroAndCoastalSettings(t *testing.T) {
	current := testConfig(20261111)
	current.WorldProfile = StandardWorldProfile()
	legacy := current
	legacy.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.23"

	a, err := resolveTopLevelConfig(current)
	if err != nil {
		t.Fatal(err)
	}
	b, err := resolveTopLevelConfig(legacy)
	if err != nil {
		t.Fatal(err)
	}

	if a.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.24" ||
		b.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.23" {
		t.Fatalf("wrong profile rules: v124=%s old=%s",
			a.WorldProfile.GenerationRulesVersion, b.WorldProfile.GenerationRulesVersion)
	}

	// Snapshot and tectonic settings are set by user profile, not by the
	// sinkhole-site implementation.
	comparisons := []struct {
		name string
		a, b any
	}{
		{"worldsample", a.WorldSample, b.WorldSample},
		{"waterbed", a.Waterbed, b.Waterbed},
		{"terrain", a.Terrain, b.Terrain},
		{"geomorph", a.Geomorph, b.Geomorph},
		{"reconcile", a.Reconcile, b.Reconcile},
		{"hydrology", a.Hydrology, b.Hydrology},
		{"ecology", a.Ecology, b.Ecology},
	}
	for _, c := range comparisons {
		if !reflect.DeepEqual(c.a, c.b) {
			t.Errorf("v1.1.24 unexpectedly changes %s defaults", c.name)
		}
	}
	if a.WorldSample.CoastalBackwaterVersion != 1 ||
		b.WorldSample.CoastalBackwaterVersion != 1 ||
		a.Waterbed.RiverGeometryVersion != 7 ||
		b.Waterbed.RiverGeometryVersion != 7 {
		t.Fatal("v124 must preserve v123 river and coastal settings")
	}
}
