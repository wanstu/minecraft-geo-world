package mcprofile

import (
	"math"
	"testing"

	"geoworld/internal/ecology"
)

func baseInput(width, height int) Input {
	n := width * height
	in := Input{
		Width:               width,
		Height:              height,
		LandMask:            make([]bool, n),
		EcoRegion:           make([]uint8, n),
		RegionConfidence:    make([]float64, n),
		TreeCoverPotential:  make([]float64, n),
		GrassCoverPotential: make([]float64, n),
		ShrubCoverPotential: make([]float64, n),
		BareGroundPotential: make([]float64, n),
		SnowCoverPotential:  make([]float64, n),
		SoilMoisture:        make([]float64, n),
		WetlandEcology:      make([]float64, n),
		AlpineEcology:       make([]float64, n),
		RollingPlainWeight:  make([]float64, n),
		HillWeight:          make([]float64, n),
		FoothillWeight:      make([]float64, n),
		MountainWeight:      make([]float64, n),
		PlateauWeight:       make([]float64, n),
		FloodplainWeight:    make([]float64, n),
		AlluvialPlainWeight: make([]float64, n),
		AlluvialFanWeight:   make([]float64, n),
		CanyonWeight:        make([]float64, n),
	}
	for i := 0; i < n; i++ {
		in.LandMask[i] = true
		in.EcoRegion[i] = uint8(ecology.RegionTemperateForest)
		in.RegionConfidence[i] = 0.8
		in.TreeCoverPotential[i] = 0.7
		in.GrassCoverPotential[i] = 0.3
		in.SoilMoisture[i] = 0.55
		in.RollingPlainWeight[i] = 1
	}
	return in
}

func TestSameTemperateForestGetsDifferentTerrainVariants(t *testing.T) {
	in := baseInput(2, 2)

	plain := 0
	hills := 1
	flood := 2

	in.RollingPlainWeight[hills] = 0.12
	in.HillWeight[hills] = 0.88

	in.RollingPlainWeight[flood] = 0.16
	in.FloodplainWeight[flood] = 0.92
	in.AlluvialPlainWeight[flood] = 0.62
	in.SoilMoisture[flood] = 0.78

	m, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}

	if m.BiomeFamily[plain] != BiomeTemperateForest ||
		m.BiomeFamily[hills] != BiomeTemperateForest ||
		m.BiomeFamily[flood] != BiomeTemperateForest {
		t.Fatal("terrain variant changed the ecological biome family")
	}
	if m.TerrainVariant[plain] != TerrainRollingPlain {
		t.Fatalf("plain forest variant = %s", m.TerrainVariant[plain])
	}
	if m.TerrainVariant[hills] != TerrainHills {
		t.Fatalf("hilly forest variant = %s", m.TerrainVariant[hills])
	}
	if m.TerrainVariant[flood] != TerrainFloodplain {
		t.Fatalf("floodplain forest variant = %s", m.TerrainVariant[flood])
	}
}

func TestWetlandChangesSurfaceAndVegetation(t *testing.T) {
	in := baseInput(2, 2)
	i := 0
	in.EcoRegion[i] = uint8(ecology.RegionWetland)
	in.WetlandEcology[i] = 0.95
	in.SoilMoisture[i] = 0.88
	in.FloodplainWeight[i] = 0.85
	in.RollingPlainWeight[i] = 0.15

	m, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	if m.BiomeFamily[i] != BiomeWetland {
		t.Fatalf("wetland family = %s", m.BiomeFamily[i])
	}
	if m.SurfaceProfile[i] != SurfacePeat {
		t.Fatalf("wetland surface = %s", m.SurfaceProfile[i])
	}
	if m.VegetationProfile[i] != VegetationWetland {
		t.Fatalf("wetland vegetation = %s", m.VegetationProfile[i])
	}
	if m.TerrainVariant[i] != TerrainFloodplain {
		t.Fatalf("wetland terrain = %s", m.TerrainVariant[i])
	}
}

func TestHotDesertGetsAridSurfaceAndSparseVegetation(t *testing.T) {
	in := baseInput(2, 2)
	i := 0
	in.EcoRegion[i] = uint8(ecology.RegionHotDesert)
	in.TreeCoverPotential[i] = 0
	in.GrassCoverPotential[i] = 0.05
	in.ShrubCoverPotential[i] = 0.12
	in.BareGroundPotential[i] = 0.92
	in.SoilMoisture[i] = 0.08

	m, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	if m.BiomeFamily[i] != BiomeHotDesert {
		t.Fatalf("desert family = %s", m.BiomeFamily[i])
	}
	if m.SurfaceProfile[i] != SurfaceDesertSand {
		t.Fatalf("desert surface = %s", m.SurfaceProfile[i])
	}
	if m.VegetationProfile[i] != VegetationDesertSparse {
		t.Fatalf("desert vegetation = %s", m.VegetationProfile[i])
	}
}

func TestTemperateDesertBareGroundUsesSandNotGravel(t *testing.T) {
	in := baseInput(2, 2)
	i := 0
	in.EcoRegion[i] = uint8(ecology.RegionTemperateDesert)
	in.TreeCoverPotential[i] = 0
	in.GrassCoverPotential[i] = 0.05
	in.ShrubCoverPotential[i] = 0.12
	in.BareGroundPotential[i] = 0.92
	in.SoilMoisture[i] = 0.10

	m, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	if m.SurfaceProfile[i] != SurfaceDesertSand {
		t.Fatalf("temperate desert bare ground should be sand, got %s", m.SurfaceProfile[i])
	}
}

func TestSnowyAlpineMountainGetsRockSnowProfile(t *testing.T) {
	in := baseInput(2, 2)
	i := 0
	in.EcoRegion[i] = uint8(ecology.RegionAlpine)
	in.RollingPlainWeight[i] = 0
	in.MountainWeight[i] = 0.88
	in.AlpineEcology[i] = 0.9
	in.SnowCoverPotential[i] = 0.86
	in.BareGroundPotential[i] = 0.62

	m, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	if m.BiomeFamily[i] != BiomeAlpine {
		t.Fatalf("alpine family = %s", m.BiomeFamily[i])
	}
	if m.TerrainVariant[i] != TerrainMountain {
		t.Fatalf("alpine terrain = %s", m.TerrainVariant[i])
	}
	if m.SurfaceProfile[i] != SurfaceAlpineSnow {
		t.Fatalf("alpine surface = %s", m.SurfaceProfile[i])
	}
	if m.VegetationProfile[i] != VegetationAlpineSparse {
		t.Fatalf("alpine vegetation = %s", m.VegetationProfile[i])
	}
}

func TestOceanProfilesDoNotReceiveLandVegetation(t *testing.T) {
	in := baseInput(2, 2)
	cases := []struct {
		index   int
		region  ecology.Region
		family  BiomeFamily
		surface SurfaceProfile
	}{
		{0, ecology.RegionWarmOcean, BiomeWarmOcean, SurfaceOceanSand},
		{1, ecology.RegionColdOcean, BiomeColdOcean, SurfaceOceanGravel},
		{2, ecology.RegionFrozenOcean, BiomeFrozenOcean, SurfaceFrozenOcean},
	}
	for _, tc := range cases {
		in.LandMask[tc.index] = false
		in.EcoRegion[tc.index] = uint8(tc.region)
	}

	m, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		if m.BiomeFamily[tc.index] != tc.family {
			t.Fatalf("ocean %d family = %s", tc.index, m.BiomeFamily[tc.index])
		}
		if m.SurfaceProfile[tc.index] != tc.surface {
			t.Fatalf("ocean %d surface = %s", tc.index, m.SurfaceProfile[tc.index])
		}
		if m.VegetationProfile[tc.index] != VegetationNone {
			t.Fatalf("ocean %d received vegetation %s", tc.index, m.VegetationProfile[tc.index])
		}
		if m.TerrainVariant[tc.index] != TerrainOcean {
			t.Fatalf("ocean %d terrain = %s", tc.index, m.TerrainVariant[tc.index])
		}
	}
}

func TestAllEcoRegionsMapToKnownBiomeFamily(t *testing.T) {
	regions := []ecology.Region{
		ecology.RegionWarmOcean,
		ecology.RegionTemperateOcean,
		ecology.RegionColdOcean,
		ecology.RegionFrozenOcean,
		ecology.RegionIceSheet,
		ecology.RegionTundra,
		ecology.RegionAlpine,
		ecology.RegionBorealForest,
		ecology.RegionTemperateForest,
		ecology.RegionTemperateGrassland,
		ecology.RegionTemperateDesert,
		ecology.RegionMediterraneanShrubland,
		ecology.RegionSubtropicalForest,
		ecology.RegionHotDesert,
		ecology.RegionSavanna,
		ecology.RegionTropicalMonsoonForest,
		ecology.RegionTropicalRainforest,
		ecology.RegionWetland,
	}

	for _, r := range regions {
		if got := familyFromEcoRegion(r); got == BiomeUnknown {
			t.Fatalf("eco region %s mapped to unknown family", r)
		}
	}
}

func TestProfileOutputDeterministicBoundedAndComplete(t *testing.T) {
	in := baseInput(11, 7)
	regions := []ecology.Region{
		ecology.RegionTundra,
		ecology.RegionAlpine,
		ecology.RegionBorealForest,
		ecology.RegionTemperateForest,
		ecology.RegionTemperateGrassland,
		ecology.RegionTemperateDesert,
		ecology.RegionSubtropicalForest,
		ecology.RegionHotDesert,
		ecology.RegionSavanna,
		ecology.RegionTropicalMonsoonForest,
		ecology.RegionTropicalRainforest,
		ecology.RegionWetland,
	}

	for i := range in.LandMask {
		in.EcoRegion[i] = uint8(regions[i%len(regions)])
		in.RegionConfidence[i] = 0.35 + 0.6*float64(i%7)/6
		in.RollingPlainWeight[i] = float64((i*3)%11) / 10
		in.HillWeight[i] = float64((i*5)%13) / 12
		in.FoothillWeight[i] = float64((i*7)%17) / 16
		in.MountainWeight[i] = float64((i*11)%19) / 18
		in.PlateauWeight[i] = float64((i*13)%23) / 22
		in.FloodplainWeight[i] = float64(i%5) / 4
		in.AlluvialPlainWeight[i] = float64(i%4) / 3
		in.AlluvialFanWeight[i] = float64(i%3) / 2
		in.CanyonWeight[i] = float64(i%6) / 5
		in.TreeCoverPotential[i] = float64((i*2)%9) / 8
		in.GrassCoverPotential[i] = float64((i*4)%11) / 10
		in.ShrubCoverPotential[i] = float64((i*6)%13) / 12
		in.BareGroundPotential[i] = float64((i*8)%15) / 14
		in.SnowCoverPotential[i] = float64((i*10)%17) / 16
		in.SoilMoisture[i] = float64((i*12)%19) / 18
		in.WetlandEcology[i] = float64((i*14)%21) / 20
		in.AlpineEcology[i] = float64((i*16)%23) / 22
	}

	a, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(in)
	if err != nil {
		t.Fatal(err)
	}

	for i := range a.BiomeFamily {
		if a.BiomeFamily[i] != b.BiomeFamily[i] ||
			a.TerrainVariant[i] != b.TerrainVariant[i] ||
			a.SurfaceProfile[i] != b.SurfaceProfile[i] ||
			a.VegetationProfile[i] != b.VegetationProfile[i] ||
			a.ProfileConfidence[i] != b.ProfileConfidence[i] {
			t.Fatalf("non-deterministic profile at %d", i)
		}
		if a.BiomeFamily[i] == BiomeUnknown ||
			a.TerrainVariant[i] == TerrainUnknown ||
			a.SurfaceProfile[i] == SurfaceUnknown ||
			a.VegetationProfile[i] == VegetationUnknown {
			t.Fatalf("incomplete profile at %d: %s / %s / %s / %s",
				i, a.BiomeFamily[i], a.TerrainVariant[i],
				a.SurfaceProfile[i], a.VegetationProfile[i])
		}
		if math.IsNaN(a.ProfileConfidence[i]) ||
			math.IsInf(a.ProfileConfidence[i], 0) ||
			a.ProfileConfidence[i] < 0 ||
			a.ProfileConfidence[i] > 1 {
			t.Fatalf("confidence out of range at %d: %v", i, a.ProfileConfidence[i])
		}
	}
}
