package climate

import (
	"math"
	"testing"
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
