package mining

import (
	"math"
	"testing"
)

func TestAffinityMatchesKnownGeoWorldRules(t *testing.T) {
	cases := []struct {
		mineral, terrain, biome string
		y, want                 float64
	}{
		{"iron", "Mountain", "Alpine", 80, 19.8},
		{"coal", "Rolling Plain", "Temperate Grassland", 80, 20.475},
		{"copper", "Plateau", "Savanna", 30, 12.0},
		{"gold", "Canyon", "Hot Desert", 20, 6.72},
		{"emerald", "Mountain", "Alpine", 80, 3.456},
		{"diamond", "Mountain", "Alpine", -40, 0.5 + 40.0/63.0},
	}
	for _, tc := range cases {
		got := Affinity(tc.mineral, tc.terrain, tc.biome, tc.y)
		if math.Abs(got-tc.want) > 1e-9 {
			t.Fatalf("%s affinity=%f want=%f", tc.mineral, got, tc.want)
		}
	}
}

func TestWaterHasNoMiningDistrict(t *testing.T) {
	s := Evaluate(Inputs{
		Seed: 123456, MineralRegionalization: 1,
		X: 10, Z: 20, Land: true, WaterKind: "Ocean",
		Terrain: "Plain", Biome: "Temperate Grassland",
	})
	if s.MiningSuitability != 0 || s.MiningClass != "none" ||
		s.DominantResource != "none" {
		t.Fatalf("water should not be mining district: %+v", s)
	}
}

func TestMountainFavorsSurfaceMiningAndIron(t *testing.T) {
	s := Evaluate(Inputs{
		Seed: 123456, MineralRegionalization: 1,
		X: 1024, Z: -2048, Land: true, WaterKind: "None",
		Terrain: "Mountain", Biome: "Alpine",
	})
	if s.SurfaceMiningSuitability < 0.65 {
		t.Fatalf("mountain surface mining too low: %+v", s)
	}
	if s.ResourceProfile["iron"] <= s.ResourceProfile["coal"] {
		t.Fatalf("mountain should favor iron over coal: %+v", s.ResourceProfile)
	}
	if s.MiningSuitability <= 0 || s.MiningSuitability > 1 {
		t.Fatalf("invalid mining score: %+v", s)
	}
}

func TestSedimentaryPlainFavorsCoal(t *testing.T) {
	s := Evaluate(Inputs{
		Seed: 123456, MineralRegionalization: 1,
		X: 4096, Z: 4096, Land: true, WaterKind: "None",
		Terrain: "Rolling Plain", Biome: "Temperate Grassland",
	})
	if s.ResourceProfile["coal"] <= s.ResourceProfile["copper"] {
		t.Fatalf("sedimentary plain should favor coal: %+v", s.ResourceProfile)
	}
	if s.IndustrialValue <= 0 || s.ResourceDiversity <= 0 {
		t.Fatalf("expected useful industrial profile: %+v", s)
	}
}

func TestProvinceStrengthDeterministic(t *testing.T) {
	a := ProvinceStrength(123456, 1.8, 4000, -2000)
	b := ProvinceStrength(123456, 1.8, 4000, -2000)
	if a != b || a < 0 || a >= 1 {
		t.Fatalf("province strength invalid a=%f b=%f", a, b)
	}
	c := ProvinceStrength(123456, 1.8, 8000, -2000)
	if c == a {
		t.Fatalf("different province unexpectedly identical: %f", a)
	}
}
