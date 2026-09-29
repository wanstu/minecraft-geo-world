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

func TestLandSeasonalityExceedsOceanAtSameLatitude(t *testing.T) {
	input := Input{
		Width:     4,
		Height:    5,
		Elevation: make([]float64, 20),
		LandMask:  make([]bool, 20),
	}
	// Row 1 is about 41°N. Keep ocean at x=0 and land inland.
	for x := 1; x < 4; x++ {
		input.LandMask[1*input.Width+x] = true
		input.Elevation[1*input.Width+x] = 0.5
	}
	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	oceanAmp := m.SeasonalAmplitudeC[1*input.Width+0]
	landAmp := m.SeasonalAmplitudeC[1*input.Width+2]
	if !(landAmp > oceanAmp) {
		t.Fatalf("land seasonality should exceed ocean at same latitude: land=%.2f ocean=%.2f", landAmp, oceanAmp)
	}
}

func TestSummerTemperatureExceedsWinterTemperature(t *testing.T) {
	input := Input{
		Width:     4,
		Height:    5,
		Elevation: make([]float64, 20),
		LandMask:  make([]bool, 20),
	}
	for i := range input.LandMask {
		input.LandMask[i] = i%4 >= 2
		input.Elevation[i] = 0.55
	}
	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	for i := range m.SummerTempC {
		if !(m.SummerTempC[i] > m.WinterTempC[i]) {
			t.Fatalf("cell %d summer %.2f not warmer than winter %.2f", i, m.SummerTempC[i], m.WinterTempC[i])
		}
	}
}

func TestMonsoonIndexTargetsLowMidLatitudeLand(t *testing.T) {
	cfg := normalizeConfig(Config{})
	m := &Model{
		Config:              cfg,
		Width:               3,
		Height:              3,
		LandMask:            []bool{false, true, true, false, true, true, false, true, true},
		LatitudeDeg:         []float64{20, 20, 20, 65, 65, 65, 0, 0, 0},
		AdjustedAnnualTempC: make([]float64, 9),
		OceanDistanceCells:  []float64{0, 1, 2, 0, 1, 2, 0, 1, 2},
		WindX:               make([]float64, 9),
		WindZ:               make([]float64, 9),
		SeasonalAmplitudeC:  make([]float64, 9),
		SummerTempC:         make([]float64, 9),
		WinterTempC:         make([]float64, 9),
		MonsoonIndex:        make([]float64, 9),
		SummerWindX:         make([]float64, 9),
		SummerWindZ:         make([]float64, 9),
		WinterWindX:         make([]float64, 9),
		WinterWindZ:         make([]float64, 9),
	}
	m.computeSeasonalStates()

	if m.MonsoonIndex[1] <= 0 {
		t.Fatalf("expected subtropical land monsoon signal, got %.3f", m.MonsoonIndex[1])
	}
	if m.MonsoonIndex[4] > 0.05 {
		t.Fatalf("high-latitude monsoon signal too strong: %.3f", m.MonsoonIndex[4])
	}
	if m.MonsoonIndex[0] != 0 {
		t.Fatalf("ocean cell must not receive land monsoon index: %.3f", m.MonsoonIndex[0])
	}
}

func TestMonsoonSummerWindOnshoreWinterWindOffshore(t *testing.T) {
	cfg := normalizeConfig(Config{MonsoonWindStrength: 1})
	m := &Model{
		Config:              cfg,
		Width:               5,
		Height:              1,
		LandMask:            []bool{false, true, true, true, true},
		LatitudeDeg:         []float64{20, 20, 20, 20, 20},
		AdjustedAnnualTempC: []float64{20, 20, 20, 20, 20},
		OceanDistanceCells:  []float64{0, 1, 2, 3, 4},
		WindX:               make([]float64, 5),
		WindZ:               make([]float64, 5),
		SeasonalAmplitudeC:  make([]float64, 5),
		SummerTempC:         make([]float64, 5),
		WinterTempC:         make([]float64, 5),
		MonsoonIndex:        make([]float64, 5),
		SummerWindX:         make([]float64, 5),
		SummerWindZ:         make([]float64, 5),
		WinterWindX:         make([]float64, 5),
		WinterWindZ:         make([]float64, 5),
	}
	m.computeSeasonalStates()

	i := 2
	if m.MonsoonIndex[i] <= 0 {
		t.Fatalf("expected monsoon at inland test cell")
	}
	if !(m.SummerWindX[i] > 0) {
		t.Fatalf("summer monsoon should point inland (+X), got %.3f", m.SummerWindX[i])
	}
	if !(m.WinterWindX[i] < 0) {
		t.Fatalf("winter monsoon should point seaward (-X), got %.3f", m.WinterWindX[i])
	}
}

func syntheticSeasonalPrecipModel(t *testing.T) *Model {
	t.Helper()

	width, height := 24, 5
	n := width * height
	cfg := normalizeConfig(Config{
		MoistureIterations:   96,
		PrecipitationScaleMM: 40000,
	})

	m := &Model{
		Config:                 cfg,
		Width:                  width,
		Height:                 height,
		LandMask:               make([]bool, n),
		LatitudeDeg:            make([]float64, n),
		LandElevationM:         make([]float64, n),
		PressureIndex:          make([]float64, n),
		AdjustedAnnualTempC:    make([]float64, n),
		SeaSurfaceTempC:        make([]float64, n),
		SeasonalAmplitudeC:     make([]float64, n),
		SummerTempC:            make([]float64, n),
		WinterTempC:            make([]float64, n),
		MonsoonIndex:           make([]float64, n),
		SummerWindX:            make([]float64, n),
		SummerWindZ:            make([]float64, n),
		WinterWindX:            make([]float64, n),
		WinterWindZ:            make([]float64, n),
		WarmSeasonPrecipMM:     make([]float64, n),
		ColdSeasonPrecipMM:     make([]float64, n),
		SeasonalAnnualPrecipMM: make([]float64, n),
		RainfallSeasonality:    make([]float64, n),
		MonsoonRainRatio:       make([]float64, n),
	}

	for z := 0; z < height; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			m.LatitudeDeg[i] = 20
			m.PressureIndex[i] = 0.46
			m.AdjustedAnnualTempC[i] = 23
			m.SummerTempC[i] = 29
			m.WinterTempC[i] = 17
			m.SeasonalAmplitudeC[i] = 6

			if x < 4 {
				m.SeaSurfaceTempC[i] = 26
				m.SummerTempC[i] = 27
				m.WinterTempC[i] = 23
				m.SeasonalAmplitudeC[i] = 2
				m.SummerWindX[i] = 1
				m.WinterWindX[i] = -1
				continue
			}

			m.LandMask[i] = true
			m.MonsoonIndex[i] = 0.8
			// Warm season carries marine air eastward into the continent.
			m.SummerWindX[i] = 1
			// Cold season reverses and exports dry continental air westward.
			m.WinterWindX[i] = -1
		}
	}

	return m
}

func TestSeasonalMonsoonCreatesWarmSeasonRainMaximum(t *testing.T) {
	m := syntheticSeasonalPrecipModel(t)
	m.computeSeasonalPrecipitation()

	z := 2
	i := z*m.Width + 9
	warm := m.WarmSeasonPrecipMM[i]
	cold := m.ColdSeasonPrecipMM[i]

	if !(warm > cold*1.20) {
		t.Fatalf("warm-season monsoon not wetter enough inland: warm=%.1f cold=%.1f", warm, cold)
	}
	if m.RainfallSeasonality[i] <= 0.08 {
		t.Fatalf("expected meaningful rainfall seasonality, got %.3f", m.RainfallSeasonality[i])
	}
	if m.MonsoonRainRatio[i] <= 0 {
		t.Fatalf("expected positive monsoon rain ratio, got %.3f", m.MonsoonRainRatio[i])
	}
}

func TestSeasonalPrecipitationFieldsAreBoundedAndConsistent(t *testing.T) {
	input := Input{
		Width:     40,
		Height:    24,
		Elevation: make([]float64, 40*24),
		LandMask:  make([]bool, 40*24),
	}
	for z := 4; z < 20; z++ {
		for x := 8; x < 32; x++ {
			i := z*input.Width + x
			input.LandMask[i] = true
			input.Elevation[i] = 0.56
		}
	}

	m, err := Generate(input, Config{})
	if err != nil {
		t.Fatal(err)
	}

	for i := range m.WarmSeasonPrecipMM {
		warm := m.WarmSeasonPrecipMM[i]
		cold := m.ColdSeasonPrecipMM[i]
		total := m.SeasonalAnnualPrecipMM[i]
		seasonality := m.RainfallSeasonality[i]
		ratio := m.MonsoonRainRatio[i]

		if math.IsNaN(warm) || math.IsInf(warm, 0) ||
			math.IsNaN(cold) || math.IsInf(cold, 0) ||
			math.IsNaN(total) || math.IsInf(total, 0) {
			t.Fatalf("invalid seasonal precipitation at %d", i)
		}
		if warm < 0 || warm > 3000.001 || cold < 0 || cold > 3000.001 {
			t.Fatalf("season precipitation out of half-year bounds at %d: warm=%.2f cold=%.2f", i, warm, cold)
		}
		if math.Abs(total-(warm+cold)) > 1e-9 {
			t.Fatalf("seasonal annual mismatch at %d: total=%.6f warm+cold=%.6f", i, total, warm+cold)
		}
		if seasonality < 0 || seasonality > 1 {
			t.Fatalf("rainfall seasonality out of bounds at %d: %.4f", i, seasonality)
		}
		if ratio < 0 || ratio > 1 {
			t.Fatalf("monsoon rain ratio out of bounds at %d: %.4f", i, ratio)
		}
		if !m.LandMask[i] && ratio != 0 {
			t.Fatalf("ocean cell %d has monsoon rain ratio %.4f", i, ratio)
		}
	}
}

func TestMonsoonRainRatioRequiresWarmSeasonExcess(t *testing.T) {
	m := syntheticSeasonalPrecipModel(t)

	// Reverse the representative winds so the cold season is the marine inflow.
	for i := range m.SummerWindX {
		m.SummerWindX[i], m.WinterWindX[i] = m.WinterWindX[i], m.SummerWindX[i]
	}
	m.computeSeasonalPrecipitation()

	i := 2*m.Width + 9
	if m.WarmSeasonPrecipMM[i] >= m.ColdSeasonPrecipMM[i] {
		t.Fatalf("test setup failed to create cold-season maximum: warm=%.1f cold=%.1f",
			m.WarmSeasonPrecipMM[i], m.ColdSeasonPrecipMM[i])
	}
	if m.MonsoonRainRatio[i] != 0 {
		t.Fatalf("warm-season monsoon rain ratio should be zero without warm-season excess, got %.4f",
			m.MonsoonRainRatio[i])
	}
}

func TestSeasonalPrecipitationSeedSweep(t *testing.T) {
	seeds := []int64{3000, 19178, 43445, 75801}

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
		baseAnnual := 0.0
		seasonalAnnual := 0.0
		seasonalitySum := 0.0
		seasonalitySaturated := 0

		weakWarm, weakCold := 0.0, 0.0
		strongWarm, strongCold := 0.0, 0.0
		weakN, strongN := 0, 0

		for i := range m.AnnualPrecipMM {
			if world.Crust[i] != phase1.Continental {
				continue
			}
			landN++
			baseAnnual += m.AnnualPrecipMM[i]
			seasonalAnnual += m.SeasonalAnnualPrecipMM[i]
			seasonalitySum += m.RainfallSeasonality[i]
			if m.RainfallSeasonality[i] >= 0.999 {
				seasonalitySaturated++
			}

			switch {
			case m.MonsoonIndex[i] < 0.20:
				weakWarm += m.WarmSeasonPrecipMM[i]
				weakCold += m.ColdSeasonPrecipMM[i]
				weakN++
			case m.MonsoonIndex[i] >= 0.60:
				strongWarm += m.WarmSeasonPrecipMM[i]
				strongCold += m.ColdSeasonPrecipMM[i]
				strongN++
			}
		}

		if landN == 0 {
			t.Fatalf("seed %d has no land", seed)
		}

		baseAvg := baseAnnual / float64(landN)
		seasonalAvg := seasonalAnnual / float64(landN)
		ratio := seasonalAvg / math.Max(baseAvg, 1)
		if ratio < 0.60 || ratio > 1.60 {
			t.Fatalf("seed %d seasonal annual precipitation drifted too far: ratio %.3f", seed, ratio)
		}

		avgSeasonality := seasonalitySum / float64(landN)
		if avgSeasonality < 0.10 || avgSeasonality > 0.85 {
			t.Fatalf("seed %d rainfall seasonality suspicious: %.3f", seed, avgSeasonality)
		}

		saturatedPct := float64(seasonalitySaturated) / float64(landN)
		if saturatedPct > 0.10 {
			t.Fatalf("seed %d too much fully seasonal land: %.2f%%", seed, saturatedPct*100)
		}

		if weakN > 0 && weakCold > 1 {
			weakRatio := (weakWarm / float64(weakN)) / (weakCold / float64(weakN))
			if weakRatio > 3.0 {
				t.Fatalf("seed %d weak-monsoon seasonal contrast too strong: %.2f", seed, weakRatio)
			}
		}

		if strongN > 0 && strongCold > 1 {
			strongRatio := (strongWarm / float64(strongN)) / (strongCold / float64(strongN))
			if strongRatio > 50 {
				t.Fatalf("seed %d strong-monsoon seasonal contrast numerically unstable: %.2f", seed, strongRatio)
			}
		}
	}
}

func TestElevationCurveCompressesMidElevation(t *testing.T) {
	input := Input{
		Width:     2,
		Height:    2,
		Elevation: []float64{0.625, 0.75, 0.90, 1.0},
		LandMask:  []bool{true, true, true, true},
	}
	cfg := Config{
		MaxLatitudeDeg:      10,
		EquatorTempC:        28,
		PolarTempC:          24,
		LapseRateCPerKm:     6.5,
		MaxLandElevationM:   6000,
		ElevationCurvePower: 1.70,
		SeaLevelNormalized:  0.5,
	}
	m, err := Generate(input, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if m.LandElevationM[0] < 500 || m.LandElevationM[0] > 650 {
		t.Fatalf("quarter normalized land height mapped unexpectedly: %.1fm", m.LandElevationM[0])
	}
	if m.LandElevationM[1] < 1750 || m.LandElevationM[1] > 1950 {
		t.Fatalf("half normalized land height mapped unexpectedly: %.1fm", m.LandElevationM[1])
	}
	if !(m.LandElevationM[0] < m.LandElevationM[1] &&
		m.LandElevationM[1] < m.LandElevationM[2] &&
		m.LandElevationM[2] < m.LandElevationM[3]) {
		t.Fatalf("elevation curve is not monotonic: %v", m.LandElevationM)
	}
	if math.Abs(m.LandElevationM[3]-6000) > 1e-9 {
		t.Fatalf("maximum land height changed: %.3f", m.LandElevationM[3])
	}
}

func TestLatitudeTemperatureCurveKeepsMidLatitudesTemperate(t *testing.T) {
	cfg := normalizeConfig(Config{})
	maxLatRad := cfg.MaxLatitudeDeg * math.Pi / 180
	maxLatSin := math.Sin(maxLatRad)

	tempAt := func(lat float64) float64 {
		latFactor := math.Sin(math.Abs(lat)*math.Pi/180) / maxLatSin
		latFactor = clamp(latFactor, 0, 1)
		latFactor = math.Pow(latFactor, cfg.LatitudeTempCurvePower)
		return cfg.EquatorTempC + (cfg.PolarTempC-cfg.EquatorTempC)*latFactor
	}

	t30 := tempAt(30)
	t45 := tempAt(45)
	t60 := tempAt(60)

	if t30 < 20 || t30 > 27 {
		t.Fatalf("30-degree sea-level climate too cold/hot: %.2fC", t30)
	}
	if t45 < 10 || t45 > 18 {
		t.Fatalf("45-degree sea-level climate too cold/hot: %.2fC", t45)
	}
	if t60 < -3 || t60 > 5 {
		t.Fatalf("60-degree sea-level climate outside expected cool range: %.2fC", t60)
	}
	if !(t30 > t45 && t45 > t60) {
		t.Fatalf("latitude temperature curve not monotonic: 30=%.2f 45=%.2f 60=%.2f", t30, t45, t60)
	}
}
