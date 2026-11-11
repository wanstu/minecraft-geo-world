package generator

import (
	"reflect"
	"testing"
)

func TestR81eV125KeepsV124HydrologyAndMacroProfileDefaults(t *testing.T) {
	candidate := testConfig(20261111)
	candidate.WorldProfile = StandardWorldProfile()
	legacy := candidate
	legacy.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.24"

	a, err := resolveTopLevelConfig(candidate)
	if err != nil {
		t.Fatal(err)
	}
	b, err := resolveTopLevelConfig(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if a.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.25" ||
		b.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.24" {
		t.Fatalf("unexpected rules: new=%s old=%s",
			a.WorldProfile.GenerationRulesVersion, b.WorldProfile.GenerationRulesVersion)
	}

	same := []struct {
		name        string
		left, right any
	}{
		{"waterbed", a.Waterbed, b.Waterbed},
		{"worldsample", a.WorldSample, b.WorldSample},
		{"terrain", a.Terrain, b.Terrain},
		{"geomorph", a.Geomorph, b.Geomorph},
		{"hydrology", a.Hydrology, b.Hydrology},
		{"reconcile", a.Reconcile, b.Reconcile},
		{"ecology", a.Ecology, b.Ecology},
		{"climate", a.Climate, b.Climate},
		{"landform", a.Landform, b.Landform},
	}
	for _, v := range same {
		if !reflect.DeepEqual(v.left, v.right) {
			t.Errorf("v1.1.25 changed %s, but only sinkhole Apply is allowed to change", v.name)
		}
	}
	if a.Waterbed.RiverGeometryVersion != 7 ||
		b.Waterbed.RiverGeometryVersion != 7 ||
		a.WorldSample.CoastalBackwaterVersion != 1 ||
		b.WorldSample.CoastalBackwaterVersion != 1 {
		t.Fatal("river/coastal semantics unexpectedly changed")
	}
}
