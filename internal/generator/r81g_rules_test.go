package generator

import (
	"reflect"
	"testing"
)

// R8.1g may alter only the versioned sea-receiver water surface handoff.
// All macro terrain, hydrology, river geometry, and special-feature planners
// remain unchanged for the same creation settings and seed.
func TestR81gV126KeepsV125WorldConfigExceptCoastalHandoff(t *testing.T) {
	current := testConfig(123456)
	current.WorldProfile = StandardWorldProfile()
	previous := current
	previous.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.25"
	a, err := resolveTopLevelConfig(current)
	if err != nil {
		t.Fatal(err)
	}
	b, err := resolveTopLevelConfig(previous)
	if err != nil {
		t.Fatal(err)
	}
	if a.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.26" ||
		b.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.25" {
		t.Fatalf("wrong rules: current=%s locked=%s", a.WorldProfile.GenerationRulesVersion, b.WorldProfile.GenerationRulesVersion)
	}
	if a.WorldSample.CoastalBackwaterVersion != 2 ||
		b.WorldSample.CoastalBackwaterVersion != 1 {
		t.Fatalf("coastal version leaked: current=%d locked=%d", a.WorldSample.CoastalBackwaterVersion, b.WorldSample.CoastalBackwaterVersion)
	}
	a.WorldSample.CoastalBackwaterVersion = 1
	a.WorldProfile.GenerationRulesVersion = b.WorldProfile.GenerationRulesVersion
	if !reflect.DeepEqual(a, b) {
		t.Fatal("new rules unexpectedly changed non-water WorldProfile resolution")
	}
}
