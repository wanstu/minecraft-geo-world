package generator

import (
	"reflect"
	"testing"
)

// Slice E v1.1.26 changes Runtime river navigation / mouth semantics only.
// Macro terrain, hydrology and Waterbed river geometry stay locked to v1.1.25.
func TestSliceEV126KeepsV125MacroAndHydrologySettings(t *testing.T) {
	current := testConfig(20261117)
	current.WorldProfile = StandardWorldProfile()
	current.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.26"

	locked := current
	locked.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.25"

	a, err := resolveTopLevelConfig(current)
	if err != nil {
		t.Fatal(err)
	}
	b, err := resolveTopLevelConfig(locked)
	if err != nil {
		t.Fatal(err)
	}

	if a.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.26" ||
		b.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.25" {
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
		{"waterbed", a.Waterbed, b.Waterbed},
		{"terrain", a.Terrain, b.Terrain},
		{"geomorph", a.Geomorph, b.Geomorph},
		{"reconcile", a.Reconcile, b.Reconcile},
		{"hydrology", a.Hydrology, b.Hydrology},
		{"ecology", a.Ecology, b.Ecology},
	}
	for _, c := range comparisons {
		if !reflect.DeepEqual(c.a, c.b) {
			t.Errorf("v1.1.26 unexpectedly changes %s defaults", c.name)
		}
	}
	if a.WorldSample.RiverCenterDepthNormalized != b.WorldSample.RiverCenterDepthNormalized ||
		a.WorldSample.MinRiverWaterDepthNormalized != b.WorldSample.MinRiverWaterDepthNormalized ||
		a.WorldSample.OceanEligibilityThreshold != b.WorldSample.OceanEligibilityThreshold {
		t.Fatalf("Slice E changed locked river water scalar defaults: current=%+v locked=%+v", a.WorldSample, b.WorldSample)
	}
	if a.WorldSample.CoastalBackwaterVersion != 2 ||
		a.WorldSample.RiverNavigationVersion != 1 ||
		a.WorldSample.MinNavigableRiverDepthBlocks != 3 {
		t.Fatalf("v1.1.26 navigation defaults wrong: %+v", a.WorldSample)
	}
	if b.WorldSample.CoastalBackwaterVersion != 1 ||
		b.WorldSample.RiverNavigationVersion != 0 ||
		b.WorldSample.MinNavigableRiverDepthBlocks != 0 {
		t.Fatalf("locked v1.1.25 navigation semantics changed: %+v", b.WorldSample)
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

func TestSliceFV127InheritsV126HydrologySettings(t *testing.T) {
	current := testConfig(20261118)
	current.WorldProfile = StandardWorldProfile()
	current.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.27"
	locked := current
	locked.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.26"

	changed, err := resolveTopLevelConfig(current)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := resolveTopLevelConfig(locked)
	if err != nil {
		t.Fatal(err)
	}
	if changed.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.27" ||
		changed.Ecology.DistributionVersion != 1 || baseline.Ecology.DistributionVersion != 0 {
		t.Fatalf("biome distribution settings not versioned: current=%+v legacy=%+v", changed.Ecology, baseline.Ecology)
	}
	changed.WorldProfile.GenerationRulesVersion = baseline.WorldProfile.GenerationRulesVersion
	changed.Ecology.DistributionVersion = baseline.Ecology.DistributionVersion
	if !reflect.DeepEqual(changed, baseline) {
		t.Fatal("v1.1.27 unexpectedly changes generation settings outside biome distribution")
	}
}
