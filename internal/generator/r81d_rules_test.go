package generator

import (
	"reflect"
	"testing"
)

// R8.1d changes only new sinkhole siting. All hydrology, terrain, river
// geometry and coastal semantics remain locked to the v1.1.23 baseline.
func TestR81dV124KeepsV123MacroAndCoastalSettings(t *testing.T) {
	current := testConfig(20261117)
	current.WorldProfile = StandardWorldProfile()
	current.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.24"
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
		t.Fatalf(
			"wrong rules identity: current=%s locked=%s",
			a.WorldProfile.GenerationRulesVersion,
			b.WorldProfile.GenerationRulesVersion,
		)
	}

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
		b.WorldSample.CoastalBackwaterVersion != 1 {
		t.Fatalf(
			"coastal backwater changed: current=%d locked=%d",
			a.WorldSample.CoastalBackwaterVersion,
			b.WorldSample.CoastalBackwaterVersion,
		)
	}
	if a.Waterbed.RiverGeometryVersion != 8 ||
		b.Waterbed.RiverGeometryVersion != 8 {
		t.Fatalf(
			"river geometry changed: current=%d locked=%d",
			a.Waterbed.RiverGeometryVersion,
			b.Waterbed.RiverGeometryVersion,
		)
	}
}
