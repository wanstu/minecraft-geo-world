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
			a.InfluenceFeature[i] != b.InfluenceFeature[i] ||
			a.TerrainZone[i] != b.TerrainZone[i] ||
			math.Abs(a.OceanAge[i]-b.OceanAge[i]) > 1e-12 ||
			math.Abs(a.CoastDistance[i]-b.CoastDistance[i]) > 1e-12 ||
			math.Abs(a.BaseElevation[i]-b.BaseElevation[i]) > 1e-12 ||
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

func TestPlateWarpChangesBoundaries(t *testing.T) {
	w := Generate(testConfig())
	changed := 0
	for i := range w.PlateID {
		if w.PlateID[i] != w.RawPlateID[i] {
			changed++
		}
	}
	ratio := float64(changed) / float64(len(w.PlateID))
	if ratio < 0.01 {
		t.Fatalf("plate warp changed too little: %.3f", ratio)
	}
	if ratio > 0.35 {
		t.Fatalf("plate warp changed too much: %.3f", ratio)
	}
}

func TestCrustThicknessVaries(t *testing.T) {
	w := Generate(testConfig())
	minThickness := 1.0
	maxThickness := 0.0
	count := 0
	for i, crust := range w.Crust {
		if crust != Continental {
			continue
		}
		count++
		v := w.CrustThickness[i]
		if v < minThickness {
			minThickness = v
		}
		if v > maxThickness {
			maxThickness = v
		}
	}
	if count == 0 {
		t.Fatal("expected continental cells")
	}
	if maxThickness-minThickness < 0.12 {
		t.Fatalf("continental crust thickness varies too little: %.3f..%.3f", minThickness, maxThickness)
	}
}

func TestTectonicIntensityVaries(t *testing.T) {
	w := Generate(testConfig())
	minIntensity := 1.0
	maxIntensity := 0.0
	count := 0
	for i, feature := range w.Feature {
		if feature == NoFeature {
			continue
		}
		count++
		v := w.TectonicIntensity[i]
		if v < minIntensity {
			minIntensity = v
		}
		if v > maxIntensity {
			maxIntensity = v
		}
		if v < 0 || v > 1 {
			t.Fatalf("tectonic intensity out of range at %d: %.3f", i, v)
		}
	}
	if count == 0 {
		t.Fatal("expected tectonic features")
	}
	if maxIntensity-minIntensity < 0.15 {
		t.Fatalf("tectonic intensity varies too little: %.3f..%.3f", minIntensity, maxIntensity)
	}
}

func TestCoastDistanceIsSigned(t *testing.T) {
	w := Generate(testConfig())
	maxLand := 0.0
	minOcean := 0.0
	landCoast := 0
	oceanCoast := 0

	for i, crust := range w.Crust {
		d := w.CoastDistance[i]
		if crust == Continental {
			if d < 0 {
				t.Fatalf("continental cell has negative coast distance at %d: %.2f", i, d)
			}
			if d > maxLand {
				maxLand = d
			}
			if d == 0 {
				landCoast++
			}
		} else {
			if d > 0 {
				t.Fatalf("ocean cell has positive coast distance at %d: %.2f", i, d)
			}
			if d < minOcean {
				minOcean = d
			}
			if d == 0 {
				oceanCoast++
			}
		}
	}

	if maxLand < 5 {
		t.Fatalf("continental interior too shallow: %.2f cells", maxLand)
	}
	if minOcean > -10 {
		t.Fatalf("ocean basin too close to coast: %.2f cells", minOcean)
	}
	if landCoast == 0 || oceanCoast == 0 {
		t.Fatalf("expected coast cells on both sides: land=%d ocean=%d", landCoast, oceanCoast)
	}
}

func TestBaseElevationFormsShelfSlopeAndBasin(t *testing.T) {
	w := Generate(testConfig())
	var shelfSum, slopeSum, basinSum float64
	var shelfN, slopeN, basinN int

	for i, crust := range w.Crust {
		if crust != Oceanic {
			continue
		}
		d := -w.CoastDistance[i]
		switch {
		case d <= 6:
			shelfSum += w.BaseElevation[i]
			shelfN++
		case d <= 18:
			slopeSum += w.BaseElevation[i]
			slopeN++
		default:
			basinSum += w.BaseElevation[i]
			basinN++
		}
	}

	if shelfN == 0 || slopeN == 0 || basinN == 0 {
		t.Fatalf("missing ocean depth bands: shelf=%d slope=%d basin=%d", shelfN, slopeN, basinN)
	}

	shelfAvg := shelfSum / float64(shelfN)
	slopeAvg := slopeSum / float64(slopeN)
	basinAvg := basinSum / float64(basinN)
	if !(shelfAvg > slopeAvg && slopeAvg > basinAvg) {
		t.Fatalf("expected shelf > slope > basin elevation, got %.3f %.3f %.3f", shelfAvg, slopeAvg, basinAvg)
	}
}

func TestMacroTerrainHasCoreZones(t *testing.T) {
	w := Generate(testConfig())
	counts := map[MacroTerrain]int{}
	for _, zone := range w.TerrainZone {
		counts[zone]++
	}

	for _, zone := range []MacroTerrain{ContinentalInterior, ContinentalShelf, ContinentalSlope, OceanBasin} {
		if counts[zone] == 0 {
			t.Fatalf("missing macro terrain zone %v", zone)
		}
	}
}
