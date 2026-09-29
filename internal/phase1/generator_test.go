package phase1

import (
	"math"
	"testing"

	"geoworld/internal/world"
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

func TestOceanicCrustAreaMatchesTarget(t *testing.T) {
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

func TestDefaultSpatialConfigUsesReferenceCellSize(t *testing.T) {
	w := Generate(testConfig())
	if got := w.Geometry.SpacingX(); math.Abs(got-DefaultReferenceCellSizeBlocks) > 1e-12 {
		t.Fatalf("default X spacing %.3f, want %.3f blocks", got, DefaultReferenceCellSizeBlocks)
	}
	if got := w.Geometry.SpacingZ(); math.Abs(got-DefaultReferenceCellSizeBlocks) > 1e-12 {
		t.Fatalf("default Z spacing %.3f, want %.3f blocks", got, DefaultReferenceCellSizeBlocks)
	}
	wantX := float64(w.Config.Width-1) * DefaultReferenceCellSizeBlocks
	wantZ := float64(w.Config.Height-1) * DefaultReferenceCellSizeBlocks
	if w.Config.SpanXBlocks != wantX || w.Config.SpanZBlocks != wantZ {
		t.Fatalf("resolved spans %.1fx%.1f, want %.1fx%.1f blocks", w.Config.SpanXBlocks, w.Config.SpanZBlocks, wantX, wantZ)
	}
}

func TestCoastDistancePhysicalScaleStableAcrossResolution(t *testing.T) {
	makeWorld := func(width, height int, spanX, spanZ float64) *World {
		geometry := world.GridGeometry{Width: width, Height: height, SpanXBlocks: spanX, SpanZBlocks: spanZ}
		w := &World{
			Geometry:      geometry,
			Config:        Config{Width: width, Height: height, SpanXBlocks: spanX, SpanZBlocks: spanZ},
			Crust:         make([]CrustType, width*height),
			CoastDistance: make([]float64, width*height),
		}
		for z := 0; z < height; z++ {
			for x := 0; x < width; x++ {
				worldX, _ := geometry.NodePosition(x, z)
				if worldX >= 1024 {
					w.Crust[w.index(x, z)] = Continental
				} else {
					w.Crust[w.index(x, z)] = Oceanic
				}
			}
		}
		w.computeCoastDistance()
		return w
	}

	coarse := makeWorld(33, 17, 4096, 2048) // 128 blocks/node
	fine := makeWorld(65, 33, 4096, 2048)   // 64 blocks/node

	coarseDistance := coarse.CoastDistance[coarse.index(16, 8)] // world X=2048
	fineDistance := fine.CoastDistance[fine.index(32, 16)]      // world X=2048
	if math.Abs(coarseDistance-fineDistance) > 1e-12 {
		t.Fatalf("coast distance changed with resolution: coarse=%.3f fine=%.3f blocks", coarseDistance, fineDistance)
	}
	if math.Abs(coarseDistance-1024) > 1e-12 {
		t.Fatalf("coast distance %.3f, want 1024 blocks", coarseDistance)
	}
}

func TestTectonicInfluencePhysicalScaleStableAcrossResolution(t *testing.T) {
	makeWorld := func(width, height int, spanX, spanZ float64, sourceX, sourceZ int) *World {
		geometry := world.GridGeometry{Width: width, Height: height, SpanXBlocks: spanX, SpanZBlocks: spanZ}
		n := width * height
		w := &World{
			Geometry:          geometry,
			Config:            Config{Width: width, Height: height, SpanXBlocks: spanX, SpanZBlocks: spanZ},
			Crust:             make([]CrustType, n),
			Feature:           make([]TectonicFeature, n),
			TectonicIntensity: make([]float64, n),
			TectonicInfluence: make([]float64, n),
			InfluenceFeature:  make([]TectonicFeature, n),
		}
		for i := range w.Crust {
			w.Crust[i] = Continental
		}
		i := w.index(sourceX, sourceZ)
		w.Feature[i] = CollisionOrogen
		w.TectonicIntensity[i] = 0.5
		w.computeTectonicInfluence()
		return w
	}

	coarse := makeWorld(65, 33, 8192, 4096, 32, 16) // 128 blocks/node
	fine := makeWorld(129, 65, 8192, 4096, 64, 32)  // 64 blocks/node

	for _, offsetBlocks := range []int{0, 512, 1024, 1536} {
		coarseX := 32 + offsetBlocks/128
		fineX := 64 + offsetBlocks/64
		coarseV := coarse.TectonicInfluence[coarse.index(coarseX, 16)]
		fineV := fine.TectonicInfluence[fine.index(fineX, 32)]
		if math.Abs(coarseV-fineV) > 1e-12 {
			t.Fatalf("tectonic influence at +%d blocks changed with resolution: coarse=%.12f fine=%.12f", offsetBlocks, coarseV, fineV)
		}
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

	if maxLand < 5*DefaultReferenceCellSizeBlocks {
		t.Fatalf("continental interior too shallow: %.2f blocks", maxLand)
	}
	if minOcean > -10*DefaultReferenceCellSizeBlocks {
		t.Fatalf("ocean basin too close to coast: %.2f blocks", minOcean)
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
		case d <= continentalShelfWidthBlocks:
			shelfSum += w.BaseElevation[i]
			shelfN++
		case d <= continentalSlopeWidthBlocks:
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

func TestMultipleSeedsPhase1Invariants(t *testing.T) {
	seeds := []int64{1, 2, 42, 2026, 123456789, 731894205, -99, -987654321}
	for _, seed := range seeds {
		cfg := testConfig()
		cfg.Seed = seed
		w := Generate(cfg)

		ocean := 0
		saturatedElevation := 0
		saturatedIntensity := 0
		featureCells := 0
		for i, crust := range w.Crust {
			if crust == Oceanic {
				ocean++
			}
			if math.IsNaN(w.Elevation[i]) || math.IsInf(w.Elevation[i], 0) {
				t.Fatalf("seed %d produced invalid elevation at %d", seed, i)
			}
			if w.Elevation[i] < 0 || w.Elevation[i] > 1 {
				t.Fatalf("seed %d elevation out of range at %d: %.3f", seed, i, w.Elevation[i])
			}
			if w.Elevation[i] >= 0.999 {
				saturatedElevation++
			}
			if w.Feature[i] != NoFeature {
				featureCells++
				if w.TectonicIntensity[i] >= 0.999 {
					saturatedIntensity++
				}
			}
		}

		elevationSatRatio := float64(saturatedElevation) / float64(len(w.Elevation))
		if elevationSatRatio > 0.03 {
			t.Fatalf("seed %d has too much saturated elevation: %.3f%%", seed, elevationSatRatio*100)
		}
		if featureCells > 0 {
			intensitySatRatio := float64(saturatedIntensity) / float64(featureCells)
			if intensitySatRatio > 0.20 {
				t.Fatalf("seed %d has too much saturated tectonic intensity: %.3f%%", seed, intensitySatRatio*100)
			}
		}

		gotOcean := float64(ocean) / float64(len(w.Crust))
		if math.Abs(gotOcean-cfg.OceanFraction) > 0.002 {
			t.Fatalf("seed %d ocean fraction %.4f, want %.4f", seed, gotOcean, cfg.OceanFraction)
		}

		buffer := edgeOceanBufferCells(w.Config)
		for y := 0; y < cfg.Height; y++ {
			for x := 0; x < cfg.Width; x++ {
				if x < buffer || y < buffer || x >= cfg.Width-buffer || y >= cfg.Height-buffer {
					if w.Crust[w.index(x, y)] != Oceanic {
						t.Fatalf("seed %d has land inside %d-cell edge ocean buffer at %d,%d", seed, buffer, x, y)
					}
				}
			}
		}
	}
}

func BenchmarkGenerate640x400(b *testing.B) {
	cfg := Config{
		Seed:          731894205,
		Width:         640,
		Height:        400,
		PlateCount:    18,
		OceanFraction: 0.68,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.Seed = 731894205 + int64(i)
		_ = Generate(cfg)
	}
}

func BenchmarkGenerate1024x640(b *testing.B) {
	cfg := Config{
		Seed:          731894205,
		Width:         1024,
		Height:        640,
		PlateCount:    18,
		OceanFraction: 0.68,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.Seed = 731894205 + int64(i)
		_ = Generate(cfg)
	}
}
