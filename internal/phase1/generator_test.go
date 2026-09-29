package phase1

import (
	"math"
	"testing"
)

func testConfig() Config {
	return Config{
		Seed:          731894205,
		Width:         160,
		Height:        100,
		PlateCount:    18,
		OceanFraction: 0.68,
	}
}

func TestGenerateDeterministic(t *testing.T) {
	a := Generate(testConfig())
	b := Generate(testConfig())

	for i := range a.PlateID {
		if a.PlateID[i] != b.PlateID[i] ||
			a.Crust[i] != b.Crust[i] ||
			a.Boundary[i] != b.Boundary[i] ||
			a.Feature[i] != b.Feature[i] ||
			math.Abs(a.OceanAge[i]-b.OceanAge[i]) > 1e-12 ||
			math.Abs(a.Elevation[i]-b.Elevation[i]) > 1e-12 {
			t.Fatalf("world differs at cell %d", i)
		}
	}
}

func TestOceanAreaMatchesTarget(t *testing.T) {
	cfg := testConfig()
	w := Generate(cfg)
	ocean := 0
	for _, crust := range w.Crust {
		if crust == Oceanic {
			ocean++
		}
	}
	got := float64(ocean) / float64(len(w.Crust))
	if math.Abs(got-cfg.OceanFraction) > 0.002 {
		t.Fatalf("ocean fraction %.4f, want %.4f", got, cfg.OceanFraction)
	}
}

func TestOceanAgeHasRange(t *testing.T) {
	w := Generate(testConfig())
	minAge := 1.0
	maxAge := 0.0
	count := 0
	for i, crust := range w.Crust {
		if crust != Oceanic {
			continue
		}
		count++
		age := w.OceanAge[i]
		if age < minAge {
			minAge = age
		}
		if age > maxAge {
			maxAge = age
		}
	}
	if count == 0 {
		t.Fatal("expected ocean cells")
	}
	if minAge > 0.1 {
		t.Fatalf("youngest ocean crust too old: %.3f", minAge)
	}
	if maxAge < 0.35 {
		t.Fatalf("oldest ocean crust too young: %.3f", maxAge)
	}
}

func TestTectonicInfluenceHasWidth(t *testing.T) {
	w := Generate(testConfig())
	sourceCount := 0
	influencedCount := 0
	for i := range w.Feature {
		if w.Feature[i] != NoFeature {
			sourceCount++
		}
		if math.Abs(w.TectonicInfluence[i]) > 0.005 {
			influencedCount++
		}
	}
	if sourceCount == 0 {
		t.Fatal("expected tectonic sources")
	}
	if influencedCount <= sourceCount {
		t.Fatalf("influence did not expand beyond boundary sources: sources=%d influenced=%d", sourceCount, influencedCount)
	}
}
