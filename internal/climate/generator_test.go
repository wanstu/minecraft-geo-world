package climate

import (
	"math"
	"testing"

	"geoworld/internal/phase1"
)

func TestLatitudeRunsNorthToSouth(t *testing.T) {
	input := Input{
		Width:     2,
		Height:    5,
		Elevation: make([]float64, 10),
		LandMask:  make([]bool, 10),
	}
	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if math.Abs(m.LatitudeDeg[0]-82) > 1e-9 {
		t.Fatalf("north latitude %.3f, want 82", m.LatitudeDeg[0])
	}
	if math.Abs(m.LatitudeDeg[4*2]+82) > 1e-9 {
		t.Fatalf("south latitude %.3f, want -82", m.LatitudeDeg[4*2])
	}
	if math.Abs(m.LatitudeDeg[2*2]) > 1e-9 {
		t.Fatalf("equator latitude %.3f, want 0", m.LatitudeDeg[2*2])
	}
}

func TestAnnualTemperatureIsSymmetricByLatitude(t *testing.T) {
	input := Input{
		Width:  1 + 1,
		Height: 7,
	}
	input.Elevation = make([]float64, input.Width*input.Height)
	input.LandMask = make([]bool, len(input.Elevation))

	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	for y := 0; y < input.Height; y++ {
		mirror := input.Height - 1 - y
		a := m.AnnualMeanTempC[y*input.Width]
		b := m.AnnualMeanTempC[mirror*input.Width]
		if math.Abs(a-b) > 1e-9 {
			t.Fatalf("temperature not symmetric at rows %d/%d: %.6f vs %.6f", y, mirror, a, b)
		}
	}
}

func TestElevationCoolsLand(t *testing.T) {
	input := Input{
		Width:     2,
		Height:    2,
		Elevation: []float64{0.5, 1.0, 0.5, 1.0},
		LandMask:  []bool{true, true, true, true},
	}
	cfg := Config{
		MaxLatitudeDeg:     10,
		EquatorTempC:       28,
		PolarTempC:         24,
		LapseRateCPerKm:    6.5,
		MaxLandElevationM:  6000,
		SeaLevelNormalized: 0.5,
	}
	m, err := Generate(input, cfg)
	if err != nil {
		t.Fatal(err)
	}

	low := m.AnnualMeanTempC[0]
	high := m.AnnualMeanTempC[1]
	gotDrop := low - high
	wantDrop := 39.0
	if math.Abs(gotDrop-wantDrop) > 1e-9 {
		t.Fatalf("lapse cooling %.3fC, want %.3fC", gotDrop, wantDrop)
	}
}

func TestOceanDoesNotUseSeafloorElevationForLapseRate(t *testing.T) {
	input := Input{
		Width:     2,
		Height:    2,
		Elevation: []float64{0.10, 0.49, 0.10, 0.49},
		LandMask:  []bool{false, false, false, false},
	}
	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if math.Abs(m.AnnualMeanTempC[0]-m.AnnualMeanTempC[1]) > 1e-12 {
		t.Fatalf("same-latitude ocean temperatures differ by seafloor depth: %.6f vs %.6f",
			m.AnnualMeanTempC[0], m.AnnualMeanTempC[1])
	}
	if m.LandElevationM[0] != 0 || m.LandElevationM[1] != 0 {
		t.Fatal("ocean cells must not receive land elevation lapse correction")
	}
}

func TestHigherLatitudeIsColderAtSeaLevel(t *testing.T) {
	input := Input{
		Width:     2,
		Height:    5,
		Elevation: make([]float64, 10),
		LandMask:  make([]bool, 10),
	}
	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	north := m.AnnualMeanTempC[0]
	equator := m.AnnualMeanTempC[2*2]
	if !(north < equator) {
		t.Fatalf("expected polar region colder than equator: north %.2f equator %.2f", north, equator)
	}
}

func TestPressureBeltsFollowIdealizedCirculation(t *testing.T) {
	p0, _, _ := circulationAtLatitude(0)
	p30, _, _ := circulationAtLatitude(30)
	p60, _, _ := circulationAtLatitude(60)
	p82, _, _ := circulationAtLatitude(82)

	if !(p0 < p30) {
		t.Fatalf("equatorial low missing: p0=%.3f p30=%.3f", p0, p30)
	}
	if !(p60 < p30) {
		t.Fatalf("subpolar low missing: p60=%.3f p30=%.3f", p60, p30)
	}
	if !(p82 > p60) {
		t.Fatalf("polar high missing: p82=%.3f p60=%.3f", p82, p60)
	}
}

func TestWindBeltsHaveExpectedDirections(t *testing.T) {
	_, x15, z15 := circulationAtLatitude(15)
	if !(x15 < 0 && z15 > 0) {
		t.Fatalf("NH trade wind wrong direction: x=%.3f z=%.3f", x15, z15)
	}

	_, xm15, zm15 := circulationAtLatitude(-15)
	if !(xm15 < 0 && zm15 < 0) {
		t.Fatalf("SH trade wind wrong direction: x=%.3f z=%.3f", xm15, zm15)
	}

	_, x45, z45 := circulationAtLatitude(45)
	if !(x45 > 0 && z45 < 0) {
		t.Fatalf("NH westerly wrong direction: x=%.3f z=%.3f", x45, z45)
	}

	_, xm45, zm45 := circulationAtLatitude(-45)
	if !(xm45 > 0 && zm45 > 0) {
		t.Fatalf("SH westerly wrong direction: x=%.3f z=%.3f", xm45, zm45)
	}

	_, x75, z75 := circulationAtLatitude(75)
	if !(x75 < 0 && z75 > 0) {
		t.Fatalf("NH polar easterly wrong direction: x=%.3f z=%.3f", x75, z75)
	}
}

func TestWindFieldIsHemisphereSymmetric(t *testing.T) {
	for _, lat := range []float64{5, 15, 30, 45, 60, 75} {
		pN, xN, zN := circulationAtLatitude(lat)
		pS, xS, zS := circulationAtLatitude(-lat)
		if math.Abs(pN-pS) > 1e-12 {
			t.Fatalf("pressure asymmetric at %.1f°", lat)
		}
		if math.Abs(xN-xS) > 1e-12 {
			t.Fatalf("zonal wind asymmetric at %.1f°: %.6f %.6f", lat, xN, xS)
		}
		if math.Abs(zN+zS) > 1e-12 {
			t.Fatalf("meridional wind not mirrored at %.1f°: %.6f %.6f", lat, zN, zS)
		}
	}
}

func TestWindTransitionsAreContinuous(t *testing.T) {
	for _, boundary := range []float64{30, 60} {
		_, xA, zA := circulationAtLatitude(boundary - 0.001)
		_, xB, zB := circulationAtLatitude(boundary + 0.001)
		if math.Hypot(xA-xB, zA-zB) > 0.01 {
			t.Fatalf("wind discontinuity near %.1f°: before=(%.4f,%.4f) after=(%.4f,%.4f)",
				boundary, xA, zA, xB, zB)
		}
	}
}

func TestSurfaceCurrentCoriolisDeflectsOppositeByHemisphere(t *testing.T) {
	cfg := normalizeConfig(Config{})
	nx, nz := surfaceCurrentFromWind(1, 0, 45, cfg)
	sx, sz := surfaceCurrentFromWind(1, 0, -45, cfg)

	if !(nx > 0 && nz > 0) {
		t.Fatalf("NH current should deflect right/down in x-z coordinates: %.3f %.3f", nx, nz)
	}
	if !(sx > 0 && sz < 0) {
		t.Fatalf("SH current should deflect left/up in x-z coordinates: %.3f %.3f", sx, sz)
	}
	if math.Abs(nx-sx) > 1e-12 || math.Abs(nz+sz) > 1e-12 {
		t.Fatalf("hemisphere current deflection is not mirrored")
	}
}

func TestLandCellsHaveNoOceanCurrent(t *testing.T) {
	input := Input{
		Width:     4,
		Height:    4,
		Elevation: make([]float64, 16),
		LandMask: []bool{
			false, false, false, false,
			false, true, true, false,
			false, true, true, false,
			false, false, false, false,
		},
	}
	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}
	for i, land := range input.LandMask {
		if land && (m.CurrentX[i] != 0 || m.CurrentZ[i] != 0) {
			t.Fatalf("land cell %d has ocean current %.4f %.4f", i, m.CurrentX[i], m.CurrentZ[i])
		}
	}
}

func TestCoastRemovesOnshoreCurrentComponent(t *testing.T) {
	input := Input{
		Width:     5,
		Height:    5,
		Elevation: make([]float64, 25),
		LandMask:  make([]bool, 25),
	}
	// East-side wall. Row 1 is ~41°N and has a westerly background,
	// so without coastal blocking the current has a strong +X component.
	for y := 0; y < input.Height; y++ {
		input.LandMask[y*input.Width+3] = true
		input.LandMask[y*input.Width+4] = true
	}

	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	i := 1*input.Width + 2
	if m.CurrentX[i] > 1e-9 {
		t.Fatalf("coastal current still points into east-side land: x=%.6f z=%.6f", m.CurrentX[i], m.CurrentZ[i])
	}
}

func TestSeaSurfaceTemperatureStaysBounded(t *testing.T) {
	input := Input{
		Width:     12,
		Height:    12,
		Elevation: make([]float64, 144),
		LandMask:  make([]bool, 144),
	}
	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	for i, temp := range m.SeaSurfaceTempC {
		if temp < -1.800001 || temp > 30.000001 {
			t.Fatalf("SST out of bounds at %d: %.4f", i, temp)
		}
	}
}

func TestSeaSurfaceTransportChangesLatitudeBaseline(t *testing.T) {
	input := Input{
		Width:     24,
		Height:    24,
		Elevation: make([]float64, 576),
		LandMask:  make([]bool, 576),
	}
	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	changed := 0
	for i, temp := range m.SeaSurfaceTempC {
		baseline := clamp(m.LatitudeBaseTempC[i]+1.2, -1.8, 30)
		if math.Abs(temp-baseline) > 1e-3 {
			changed++
		}
	}
	if changed == 0 {
		t.Fatal("surface current transport did not modify SST anywhere")
	}
}

func TestWarmCurrentInfluenceDecaysInland(t *testing.T) {
	cfg := normalizeConfig(Config{
		CoastalCurrentInfluence: 0.75,
		CoastalTempDecayCells:   2,
	})
	m := &Model{
		Config:               cfg,
		Width:                6,
		Height:               1,
		LandMask:             []bool{false, true, true, true, true, true},
		LatitudeBaseTempC:    []float64{10, 10, 10, 10, 10, 10},
		AnnualMeanTempC:      []float64{10, 10, 10, 10, 10, 10},
		SeaSurfaceTempC:      []float64{15.2, 0, 0, 0, 0, 0}, // +4°C SST anomaly at ocean source
		OceanDistanceCells:   make([]float64, 6),
		OceanTempCorrectionC: make([]float64, 6),
		AdjustedAnnualTempC:  make([]float64, 6),
	}
	m.applyOceanTemperatureInfluence()

	if !(m.OceanTempCorrectionC[1] > m.OceanTempCorrectionC[2] &&
		m.OceanTempCorrectionC[2] > m.OceanTempCorrectionC[3] &&
		m.OceanTempCorrectionC[3] > 0) {
		t.Fatalf("warm-current correction does not decay inland: %v", m.OceanTempCorrectionC)
	}
}

func TestColdCurrentProducesNegativeCoastalCorrection(t *testing.T) {
	cfg := normalizeConfig(Config{CoastalTempDecayCells: 3})
	m := &Model{
		Config:               cfg,
		Width:                3,
		Height:               1,
		LandMask:             []bool{false, true, true},
		LatitudeBaseTempC:    []float64{10, 10, 10},
		AnnualMeanTempC:      []float64{10, 10, 10},
		SeaSurfaceTempC:      []float64{7.2, 0, 0}, // -4°C anomaly from 11.2°C baseline
		OceanDistanceCells:   make([]float64, 3),
		OceanTempCorrectionC: make([]float64, 3),
		AdjustedAnnualTempC:  make([]float64, 3),
	}
	m.applyOceanTemperatureInfluence()

	if m.OceanTempCorrectionC[1] >= 0 {
		t.Fatalf("cold current should cool coast, got %.3f", m.OceanTempCorrectionC[1])
	}
}

func TestOceanCorrectionPreservesExistingLandTemperatureDifference(t *testing.T) {
	cfg := normalizeConfig(Config{
		CoastalCurrentInfluence: 0.75,
		CoastalTempDecayCells:   1000, // nearly equal ocean influence on both land cells
	})
	m := &Model{
		Config:               cfg,
		Width:                3,
		Height:               1,
		LandMask:             []bool{false, true, true},
		LatitudeBaseTempC:    []float64{10, 10, 10},
		AnnualMeanTempC:      []float64{10, 5, -5},
		SeaSurfaceTempC:      []float64{15.2, 0, 0},
		OceanDistanceCells:   make([]float64, 3),
		OceanTempCorrectionC: make([]float64, 3),
		AdjustedAnnualTempC:  make([]float64, 3),
	}
	m.applyOceanTemperatureInfluence()

	before := m.AnnualMeanTempC[1] - m.AnnualMeanTempC[2]
	after := m.AdjustedAnnualTempC[1] - m.AdjustedAnnualTempC[2]
	if math.Abs(before-after) > 0.01 {
		t.Fatalf("ocean correction erased existing land temperature structure: before %.3f after %.3f", before, after)
	}
}

func TestAdjustedTemperatureFinite(t *testing.T) {
	input := Input{
		Width:     16,
		Height:    12,
		Elevation: make([]float64, 192),
		LandMask:  make([]bool, 192),
	}
	for z := 3; z < 9; z++ {
		for x := 5; x < 11; x++ {
			input.LandMask[z*input.Width+x] = true
			input.Elevation[z*input.Width+x] = 0.7
		}
	}

	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}
	for i, temp := range m.AdjustedAnnualTempC {
		if math.IsNaN(temp) || math.IsInf(temp, 0) {
			t.Fatalf("invalid adjusted temperature at %d: %v", i, temp)
		}
	}
}

func syntheticRainShadowModel(t *testing.T) *Model {
	t.Helper()
	width, height := 24, 5
	n := width * height
	cfg := normalizeConfig(Config{
		MoistureIterations:     80,
		MoistureAdvection:      0.82,
		MoistureDiffusion:      0.08,
		OrographicRainStrength: 0.28,
		PrecipitationScaleMM:   18000,
	})

	m := &Model{
		Config:              cfg,
		Width:               width,
		Height:              height,
		LandMask:            make([]bool, n),
		LandElevationM:      make([]float64, n),
		AdjustedAnnualTempC: make([]float64, n),
		PressureIndex:       make([]float64, n),
		WindX:               make([]float64, n),
		WindZ:               make([]float64, n),
		SeaSurfaceTempC:     make([]float64, n),
		Moisture:            make([]float64, n),
		OrographicLift:      make([]float64, n),
		AnnualPrecipMM:      make([]float64, n),
	}

	for z := 0; z < height; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			m.LandMask[i] = x >= 4
			m.AdjustedAnnualTempC[i] = 20
			m.PressureIndex[i] = 0.52
			m.WindX[i] = 1
			m.WindZ[i] = 0
			if x < 4 {
				m.SeaSurfaceTempC[i] = 25
			}

			// West-to-east wind crosses a ridge centered near x=12.
			switch x {
			case 8:
				m.LandElevationM[i] = 500
			case 9:
				m.LandElevationM[i] = 1200
			case 10:
				m.LandElevationM[i] = 2200
			case 11:
				m.LandElevationM[i] = 3400
			case 12:
				m.LandElevationM[i] = 4500
			case 13:
				m.LandElevationM[i] = 3200
			case 14:
				m.LandElevationM[i] = 1800
			case 15:
				m.LandElevationM[i] = 800
			}
		}
	}
	return m
}

func TestOrographicLiftAppearsOnWindwardSlope(t *testing.T) {
	m := syntheticRainShadowModel(t)
	m.computeMoistureAndPrecipitation()

	z := 2
	windward := m.OrographicLift[z*m.Width+10]
	leeward := m.OrographicLift[z*m.Width+14]
	if windward <= 0.02 {
		t.Fatalf("expected positive windward lift, got %.4f", windward)
	}
	if leeward > 0.01 {
		t.Fatalf("leeward descending slope should not have strong uplift, got %.4f", leeward)
	}
}

func TestMountainProducesRainShadow(t *testing.T) {
	m := syntheticRainShadowModel(t)
	m.computeMoistureAndPrecipitation()

	z := 2
	windward := m.AnnualPrecipMM[z*m.Width+10]
	leeward := m.AnnualPrecipMM[z*m.Width+15]
	if !(windward > leeward*1.20) {
		t.Fatalf("rain shadow too weak: windward %.1f mm leeward %.1f mm", windward, leeward)
	}
}

func TestMoistureComesFromOceanAndDeclinesInland(t *testing.T) {
	m := syntheticRainShadowModel(t)
	m.computeMoistureAndPrecipitation()

	z := 2
	coast := m.Moisture[z*m.Width+4]
	farInland := m.Moisture[z*m.Width+20]
	if !(coast > farInland) {
		t.Fatalf("expected moisture to decline inland: coast %.4f inland %.4f", coast, farInland)
	}
}

func TestPrecipitationFiniteAndBounded(t *testing.T) {
	input := Input{
		Width:     16,
		Height:    12,
		Elevation: make([]float64, 192),
		LandMask:  make([]bool, 192),
	}
	for z := 3; z < 9; z++ {
		for x := 5; x < 11; x++ {
			i := z*input.Width + x
			input.LandMask[i] = true
			input.Elevation[i] = 0.72
		}
	}
	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range m.AnnualPrecipMM {
		if math.IsNaN(p) || math.IsInf(p, 0) || p < 0 || p > 6000 {
			t.Fatalf("invalid precipitation at %d: %.3f", i, p)
		}
	}
}

func TestClimateSeedSweepDoesNotCollapsePrecipitation(t *testing.T) {
	seeds := []int64{2000, 14302, 32755, 94265}
	for _, seed := range seeds {
		world := phase1.Generate(phase1.Config{
			Seed:          seed,
			Width:         160,
			Height:        100,
			PlateCount:    18,
			OceanFraction: 0.68,
		})
		m, err := Generate(InputFromPhase1(world), Config{})
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}

		landN := 0
		landSum := 0.0
		dryN := 0
		wetN := 0
		saturated := 0
		for i, p := range m.AnnualPrecipMM {
			if math.IsNaN(p) || math.IsInf(p, 0) {
				t.Fatalf("seed %d invalid precip at %d", seed, i)
			}
			if p >= 5999 {
				saturated++
			}
			if world.Crust[i] != phase1.Continental {
				continue
			}
			landN++
			landSum += p
			if p < 250 {
				dryN++
			}
			if p > 1500 {
				wetN++
			}
		}
		if landN == 0 {
			t.Fatalf("seed %d has no land", seed)
		}

		avg := landSum / float64(landN)
		if avg < 250 || avg > 2500 {
			t.Fatalf("seed %d land precipitation collapsed: avg=%.1f mm", seed, avg)
		}
		dryPct := float64(dryN) / float64(landN)
		wetPct := float64(wetN) / float64(landN)
		if dryPct < 0.05 || dryPct > 0.90 {
			t.Fatalf("seed %d dry-land fraction suspicious: %.1f%%", seed, dryPct*100)
		}
		if wetPct < 0.01 || wetPct > 0.60 {
			t.Fatalf("seed %d wet-land fraction suspicious: %.1f%%", seed, wetPct*100)
		}
		satPct := float64(saturated) / float64(len(m.AnnualPrecipMM))
		if satPct > 0.05 {
			t.Fatalf("seed %d precipitation saturation too high: %.2f%%", seed, satPct*100)
		}
	}
}
