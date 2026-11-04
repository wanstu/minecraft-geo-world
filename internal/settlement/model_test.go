package settlement

import "testing"

func TestEvaluatePrefersTemperatePlainNearFreshWater(t *testing.T) {
	good := Evaluate(Inputs{
		Land: true, WaterKind: "None", Slope: 0.0008,
		Terrain: "Plain", Surface: "Grass Loam", Biome: "Temperate Grassland",
		AnnualMeanTempC: 16, AnnualPrecipMM: 900,
		DistanceToRiverBlocks: 180, HasRiverDistance: true,
		DistanceToLakeBlocks: 900, HasLakeDistance: true,
	})
	if good.SettlementSuitability < 0.70 {
		t.Fatalf("expected strong settlement suitability, got %+v", good)
	}
	if good.Buildability < 0.85 || good.FreshWaterAccess < 0.80 {
		t.Fatalf("expected strong buildability/water access, got %+v", good)
	}
}

func TestEvaluatePenalizesImmediateFloodplainBank(t *testing.T) {
	safe := Evaluate(Inputs{
		Land: true, WaterKind: "None", Slope: 0.0008,
		Terrain: "Plain", Surface: "Grass Loam", Biome: "Temperate Grassland",
		AnnualMeanTempC: 16, AnnualPrecipMM: 900,
		DistanceToRiverBlocks: 180, HasRiverDistance: true,
	})
	flooded := Evaluate(Inputs{
		Land: true, WaterKind: "None", Slope: 0.0008,
		Terrain: "Floodplain", Surface: "Alluvial Silt", Biome: "Wetland",
		AnnualMeanTempC: 16, AnnualPrecipMM: 1100,
		DistanceToRiverBlocks: 4, HasRiverDistance: true,
	})
	if flooded.FreshWaterAccess <= safe.FreshWaterAccess {
		t.Fatalf("bank should have at least as much fresh-water access: safe=%+v flooded=%+v", safe, flooded)
	}
	if flooded.FloodRisk < 0.85 {
		t.Fatalf("expected high flood risk, got %+v", flooded)
	}
	if flooded.SettlementSuitability >= safe.SettlementSuitability {
		t.Fatalf("flood danger must prevent river bank from dominating safer plain: safe=%+v flooded=%+v", safe, flooded)
	}
}

func TestEvaluatePenalizesSteepAlpineTerrain(t *testing.T) {
	score := Evaluate(Inputs{
		Land: true, WaterKind: "None", Slope: 0.020,
		Terrain: "Mountain", Surface: "Alpine Rock", Biome: "Alpine",
		AnnualMeanTempC: -7, AnnualPrecipMM: 620,
		DistanceToRiverBlocks: 700, HasRiverDistance: true,
	})
	if score.Buildability > 0.10 {
		t.Fatalf("steep mountain unexpectedly buildable: %+v", score)
	}
	if score.SettlementSuitability > 0.35 {
		t.Fatalf("steep alpine site unexpectedly suitable: %+v", score)
	}
	if score.ResourceAccess < 0.70 {
		t.Fatalf("mountain should still expose useful resource access: %+v", score)
	}
}

func TestEvaluateWaterIsUnsuitable(t *testing.T) {
	score := Evaluate(Inputs{
		Land: false, WaterKind: "Ocean",
		AnnualMeanTempC: 18, AnnualPrecipMM: 1000,
	})
	if score.SettlementSuitability != 0 || score.Buildability != 0 {
		t.Fatalf("water cannot be a land settlement site: %+v", score)
	}
	if score.FloodRisk != 1 {
		t.Fatalf("water should expose maximum flood/water occupation risk: %+v", score)
	}
}

func TestEvaluateScoresRemainNormalized(t *testing.T) {
	cases := []Inputs{
		{Land: true, WaterKind: "None", Slope: -2, Terrain: "Plain", Surface: "Grass Loam", Biome: "Forest", AnnualMeanTempC: 20, AnnualPrecipMM: 700},
		{Land: true, WaterKind: "None", Slope: 99, Terrain: "Canyon", Surface: "Desert Sand", Biome: "Hot Desert", AnnualMeanTempC: 90, AnnualPrecipMM: -50, DistanceToRiverBlocks: -12, HasRiverDistance: true},
	}
	for _, in := range cases {
		s := Evaluate(in)
		values := []float64{
			s.Buildability, s.FreshWaterAccess, s.FloodRisk,
			s.AgricultureSuitability, s.ClimateComfort, s.TransportAccess,
			s.ResourceAccess, s.SettlementSuitability,
		}
		for _, v := range values {
			if v < 0 || v > 1 {
				t.Fatalf("score outside [0,1]: %+v", s)
			}
		}
	}
}
