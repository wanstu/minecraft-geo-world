package ecology

import (
	"math"
	"testing"
)

func baseInput(width, height int) Input {
	n := width * height
	in := Input{
		Width:                   width,
		Height:                  height,
		LandMask:                make([]bool, n),
		AnnualTempC:             make([]float64, n),
		SummerTempC:             make([]float64, n),
		WinterTempC:             make([]float64, n),
		AnnualPrecipMM:          make([]float64, n),
		WarmSeasonPrecipMM:      make([]float64, n),
		ColdSeasonPrecipMM:      make([]float64, n),
		RainfallSeasonality:     make([]float64, n),
		MonsoonIndex:            make([]float64, n),
		SeaSurfaceTempC:         make([]float64, n),
		Elevation:               make([]float64, n),
		ReliefIndex:             make([]float64, n),
		RollingPlainWeight:      make([]float64, n),
		HillWeight:              make([]float64, n),
		FoothillWeight:          make([]float64, n),
		MountainWeight:          make([]float64, n),
		PlateauWeight:           make([]float64, n),
		WetlandPotential:        make([]float64, n),
		PerennialRiverPotential: make([]float64, n),
		SeasonalRiverPotential:  make([]float64, n),
		FloodplainWeight:        make([]float64, n),
		AlluvialPlainWeight:     make([]float64, n),
		ValleyFloorWidth:        make([]float64, n),
	}
	for i := 0; i < n; i++ {
		in.LandMask[i] = true
		in.AnnualTempC[i] = 14
		in.SummerTempC[i] = 23
		in.WinterTempC[i] = 5
		in.AnnualPrecipMM[i] = 1100
		in.WarmSeasonPrecipMM[i] = 600
		in.ColdSeasonPrecipMM[i] = 500
		in.RainfallSeasonality[i] = 0.22
		in.Elevation[i] = 0.62
		in.ReliefIndex[i] = 0.15
		in.RollingPlainWeight[i] = 1
	}
	return in
}

func singleCellInput() Input {
	return baseInput(2, 2)
}

func TestHotWetAseasonalClimatePrefersTropicalRainforest(t *testing.T) {
	in := singleCellInput()
	i := 0
	in.AnnualTempC[i] = 27
	in.SummerTempC[i] = 30
	in.WinterTempC[i] = 24
	in.AnnualPrecipMM[i] = 3200
	in.WarmSeasonPrecipMM[i] = 1700
	in.ColdSeasonPrecipMM[i] = 1500
	in.RainfallSeasonality[i] = 0.12
	in.MonsoonIndex[i] = 0.15

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.EcoRegion[i] != RegionTropicalRainforest {
		t.Fatalf("expected tropical rainforest, got %s", m.EcoRegion[i])
	}
	if m.TreeCoverPotential[i] < 0.55 {
		t.Fatalf("rainforest tree cover too low: %.3f", m.TreeCoverPotential[i])
	}
}

func TestHotSeasonalClimateCanBecomeSavanna(t *testing.T) {
	in := singleCellInput()
	i := 0
	in.AnnualTempC[i] = 25
	in.SummerTempC[i] = 31
	in.WinterTempC[i] = 19
	in.AnnualPrecipMM[i] = 850
	in.WarmSeasonPrecipMM[i] = 730
	in.ColdSeasonPrecipMM[i] = 120
	in.RainfallSeasonality[i] = 0.78
	in.MonsoonIndex[i] = 0.62

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.EcoRegion[i] != RegionSavanna && m.EcoRegion[i] != RegionTropicalMonsoonForest {
		t.Fatalf("expected savanna/monsoon ecology, got %s", m.EcoRegion[i])
	}
	if m.GrassCoverPotential[i] <= 0 {
		t.Fatal("seasonal tropical climate produced no grass potential")
	}
}

func TestHotDryClimatePrefersHotDesert(t *testing.T) {
	in := singleCellInput()
	i := 0
	in.AnnualTempC[i] = 29
	in.SummerTempC[i] = 36
	in.WinterTempC[i] = 21
	in.AnnualPrecipMM[i] = 120
	in.WarmSeasonPrecipMM[i] = 70
	in.ColdSeasonPrecipMM[i] = 50
	in.RainfallSeasonality[i] = 0.35

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.EcoRegion[i] != RegionHotDesert {
		t.Fatalf("expected hot desert, got %s", m.EcoRegion[i])
	}
	if m.BareGroundPotential[i] < 0.45 {
		t.Fatalf("hot desert bare-ground potential too low: %.3f", m.BareGroundPotential[i])
	}
}

func TestTemperateWetClimatePrefersForest(t *testing.T) {
	in := singleCellInput()
	i := 0
	in.AnnualTempC[i] = 11
	in.SummerTempC[i] = 21
	in.WinterTempC[i] = 1
	in.AnnualPrecipMM[i] = 1500
	in.WarmSeasonPrecipMM[i] = 800
	in.ColdSeasonPrecipMM[i] = 700
	in.RainfallSeasonality[i] = 0.20

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.EcoRegion[i] != RegionTemperateForest {
		t.Fatalf("expected temperate forest, got %s", m.EcoRegion[i])
	}
}

func TestVeryColdClimatePrefersIceOrTundra(t *testing.T) {
	in := singleCellInput()
	i := 0
	in.AnnualTempC[i] = -12
	in.SummerTempC[i] = -2
	in.WinterTempC[i] = -24
	in.AnnualPrecipMM[i] = 300
	in.WarmSeasonPrecipMM[i] = 180
	in.ColdSeasonPrecipMM[i] = 120

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.EcoRegion[i] != RegionIceSheet && m.EcoRegion[i] != RegionTundra {
		t.Fatalf("expected ice/tundra, got %s", m.EcoRegion[i])
	}
	if m.SnowCoverPotential[i] < 0.6 {
		t.Fatalf("very cold climate snow potential too low: %.3f", m.SnowCoverPotential[i])
	}
}

func TestColdMountainCanBecomeAlpine(t *testing.T) {
	in := singleCellInput()
	i := 0
	in.AnnualTempC[i] = 3
	in.SummerTempC[i] = 12
	in.WinterTempC[i] = -7
	in.AnnualPrecipMM[i] = 1000
	in.MountainWeight[i] = 0.85
	in.RollingPlainWeight[i] = 0.05
	in.FoothillWeight[i] = 0.10
	in.ReliefIndex[i] = 0.88

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.AlpineEcology[i] < 0.4 {
		t.Fatalf("mountain alpine tendency too low: %.3f", m.AlpineEcology[i])
	}
	if m.EcoRegion[i] != RegionAlpine && m.EcoRegion[i] != RegionTundra {
		t.Fatalf("expected alpine/tundra, got %s", m.EcoRegion[i])
	}
}

func TestWetFloodplainCanBecomeWetland(t *testing.T) {
	in := singleCellInput()
	i := 0
	in.AnnualTempC[i] = 16
	in.SummerTempC[i] = 25
	in.WinterTempC[i] = 7
	in.AnnualPrecipMM[i] = 1700
	in.WarmSeasonPrecipMM[i] = 900
	in.ColdSeasonPrecipMM[i] = 800
	in.WetlandPotential[i] = 1
	in.FloodplainWeight[i] = 1
	in.AlluvialPlainWeight[i] = 0.7
	in.ValleyFloorWidth[i] = 0.8
	in.PerennialRiverPotential[i] = 0.9

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.WetlandEcology[i] < 0.65 {
		t.Fatalf("wetland ecology too low: %.3f", m.WetlandEcology[i])
	}
	if m.EcoRegion[i] != RegionWetland {
		t.Fatalf("expected wetland, got %s", m.EcoRegion[i])
	}
}

func TestWetlandGuaranteePromotesBestConnectedNaturalLowland(t *testing.T) {
	in := baseInput(6, 6)
	anchor := 3*in.Width + 3

	// Keep the world below the ordinary wetland classification threshold but
	// give one lowland a clearly stronger natural hydrologic context.
	for i := range in.LandMask {
		in.AnnualPrecipMM[i] = 850
		in.WarmSeasonPrecipMM[i] = 480
		in.ColdSeasonPrecipMM[i] = 370
		in.Elevation[i] = 0.66
		in.ReliefIndex[i] = 0.20
	}
	for z := 2; z <= 4; z++ {
		for x := 2; x <= 4; x++ {
			i := z*in.Width + x
			in.WetlandPotential[i] = 0.20
			in.FloodplainWeight[i] = 0.25
			in.AlluvialPlainWeight[i] = 0.18
			in.ValleyFloorWidth[i] = 0.55
			in.PerennialRiverPotential[i] = 0.24
			in.Elevation[i] = 0.54
			in.ReliefIndex[i] = 0.06
		}
	}
	in.WetlandPotential[anchor] = 0.34
	in.FloodplainWeight[anchor] = 0.42
	in.AlluvialPlainWeight[anchor] = 0.30
	in.ValleyFloorWidth[anchor] = 0.70
	in.PerennialRiverPotential[anchor] = 0.38

	withoutGuarantee, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	for i, region := range withoutGuarantee.EcoRegion {
		if region == RegionWetland {
			t.Fatalf("fixture unexpectedly has a natural wetland at %d", i)
		}
	}

	withGuarantee, err := Generate(in, Config{EnsureWetlandRegion: true})
	if err != nil {
		t.Fatal(err)
	}
	wetlands := 0
	anchorNearby := false
	for i, region := range withGuarantee.EcoRegion {
		if region != RegionWetland {
			continue
		}
		wetlands++
		if withGuarantee.WetlandEcology[i] < 0.68 {
			t.Fatalf("promoted wetland ecology[%d]=%.3f want >= 0.68", i, withGuarantee.WetlandEcology[i])
		}
		x := i % in.Width
		z := i / in.Width
		ax := anchor % in.Width
		az := anchor / in.Width
		if absInt(x-ax) <= 1 && absInt(z-az) <= 1 {
			anchorNearby = true
		}
	}
	if wetlands < 2 {
		t.Fatalf("wetland guarantee produced only %d cells", wetlands)
	}
	if !anchorNearby {
		t.Fatal("wetland guarantee ignored the strongest hydrologic lowland")
	}
}

func TestWetlandGuaranteeDoesNotChangeExistingWetlandWorld(t *testing.T) {
	in := singleCellInput()
	i := 0
	in.AnnualTempC[i] = 16
	in.SummerTempC[i] = 25
	in.WinterTempC[i] = 7
	in.AnnualPrecipMM[i] = 1700
	in.WarmSeasonPrecipMM[i] = 900
	in.ColdSeasonPrecipMM[i] = 800
	in.WetlandPotential[i] = 1
	in.FloodplainWeight[i] = 1
	in.AlluvialPlainWeight[i] = 0.7
	in.ValleyFloorWidth[i] = 0.8
	in.PerennialRiverPotential[i] = 0.9

	baseline, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	guaranteed, err := Generate(in, Config{EnsureWetlandRegion: true})
	if err != nil {
		t.Fatal(err)
	}
	for j := range baseline.EcoRegion {
		if baseline.EcoRegion[j] != guaranteed.EcoRegion[j] {
			t.Fatalf("existing wetland world region[%d] changed: %s -> %s", j, baseline.EcoRegion[j], guaranteed.EcoRegion[j])
		}
		if math.Abs(baseline.WetlandEcology[j]-guaranteed.WetlandEcology[j]) > 1e-12 {
			t.Fatalf("existing wetland ecology[%d] changed", j)
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func TestOceanClassificationUsesSST(t *testing.T) {
	cases := []struct {
		sst  float64
		want Region
	}{
		{-1.7, RegionFrozenOcean},
		{5, RegionColdOcean},
		{16, RegionTemperateOcean},
		{28, RegionWarmOcean},
	}
	for _, tc := range cases {
		in := singleCellInput()
		in.LandMask[0] = false
		in.SeaSurfaceTempC[0] = tc.sst

		m, err := Generate(in, Config{})
		if err != nil {
			t.Fatal(err)
		}
		if m.EcoRegion[0] != tc.want {
			t.Fatalf("sst %.1f expected %s, got %s", tc.sst, tc.want, m.EcoRegion[0])
		}
	}
}

func TestContinuousFieldsBoundedAndDeterministic(t *testing.T) {
	in := baseInput(11, 7)
	for i := range in.LandMask {
		in.AnnualTempC[i] = -15 + float64((i*7)%48)
		in.SummerTempC[i] = in.AnnualTempC[i] + 8
		in.WinterTempC[i] = in.AnnualTempC[i] - 11
		in.AnnualPrecipMM[i] = float64((i * 193) % 3500)
		in.WarmSeasonPrecipMM[i] = in.AnnualPrecipMM[i] * (0.25 + 0.5*float64(i%5)/4)
		in.ColdSeasonPrecipMM[i] = math.Max(0, in.AnnualPrecipMM[i]-in.WarmSeasonPrecipMM[i])
		in.RainfallSeasonality[i] = float64((i*11)%23) / 22
		in.MonsoonIndex[i] = float64((i*5)%17) / 16
		in.WetlandPotential[i] = float64(i%7) / 6
		in.FloodplainWeight[i] = float64(i%5) / 4
		in.MountainWeight[i] = float64(i%4) / 3
		in.ReliefIndex[i] = float64((i*3)%13) / 12
		if i%9 == 0 {
			in.LandMask[i] = false
			in.SeaSurfaceTempC[i] = -2 + float64(i%30)
		}
	}

	a, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	fields := [][]float64{
		a.MoistureIndex,
		a.AridityIndex,
		a.WarmSeasonRainShare,
		a.ColdSeasonRainShare,
		a.ThermalSeasonality,
		a.WetlandEcology,
		a.AlpineEcology,
		a.TreeCoverPotential,
		a.GrassCoverPotential,
		a.ShrubCoverPotential,
		a.BareGroundPotential,
		a.SnowCoverPotential,
		a.SoilMoisture,
		a.VegetationDensity,
		a.RegionConfidence,
	}
	for f, field := range fields {
		for i, v := range field {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < -1e-12 || v > 1+1e-12 {
				t.Fatalf("field %d cell %d out of range: %v", f, i, v)
			}
		}
	}
	for i := range a.EcoRegion {
		if a.EcoRegion[i] != b.EcoRegion[i] || a.RegionConfidence[i] != b.RegionConfidence[i] {
			t.Fatalf("non-deterministic ecology at %d", i)
		}
		if a.EcoRegion[i] == RegionUnknown {
			t.Fatalf("unclassified ecology at %d", i)
		}
	}
}

func TestSmallClimateChangeDoesNotExplodeContinuousEcology(t *testing.T) {
	aIn := singleCellInput()
	bIn := singleCellInput()
	aIn.AnnualTempC[0] = 14.99
	bIn.AnnualTempC[0] = 15.01

	a, err := Generate(aIn, Config{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(bIn, Config{})
	if err != nil {
		t.Fatal(err)
	}

	diff := math.Abs(a.MoistureIndex[0]-b.MoistureIndex[0]) +
		math.Abs(a.TreeCoverPotential[0]-b.TreeCoverPotential[0]) +
		math.Abs(a.GrassCoverPotential[0]-b.GrassCoverPotential[0]) +
		math.Abs(a.ShrubCoverPotential[0]-b.ShrubCoverPotential[0])

	if diff > 0.05 {
		t.Fatalf("tiny temperature change caused abrupt continuous ecology jump: %.6f", diff)
	}
}
