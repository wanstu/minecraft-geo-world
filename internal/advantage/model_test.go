package advantage

import "testing"

func TestEvaluateSpecializedRegions(t *testing.T) {
	building := Evaluate(Inputs{
		Land:                   true,
		SettlementSuitability:  .90,
		Buildability:           .96,
		FreshWaterAccess:       .70,
		FloodRisk:              .04,
		AgricultureSuitability: .42,
		TransportAccess:        .70,
		ResourceAccess:         .30,
	})
	if building.DominantAdvantage != "building" ||
		building.RegionalClass != "building_region" {
		t.Fatalf("building site misclassified: %+v", building)
	}

	port := Evaluate(Inputs{
		Land:                     true,
		SettlementSuitability:    .72,
		Buildability:             .72,
		FreshWaterAccess:         .82,
		FloodRisk:                .08,
		TransportAccess:          .85,
		WaterwayNavigability:     .96,
		PortSuitability:          .98,
		NavigationBottleneckRisk: .03,
		MiningSuitability:        .20,
		IndustrialValue:          .25,
		ResourceAccess:           .30,
	})
	if port.DominantAdvantage != "port" ||
		port.RegionalClass != "port_region" {
		t.Fatalf("port site misclassified: %+v", port)
	}

	mining := Evaluate(Inputs{
		Land:                  true,
		SettlementSuitability: .55,
		Buildability:          .48,
		FloodRisk:             .08,
		TransportAccess:       .55,
		ResourceAccess:        .92,
		MiningSuitability:     .96,
		IndustrialValue:       .94,
		ResourceDiversity:     .88,
	})
	if mining.DominantAdvantage != "mining" ||
		mining.RegionalClass != "mining_region" {
		t.Fatalf("mining site misclassified: %+v", mining)
	}
}

func TestEvaluateMultiAdvantageAndWater(t *testing.T) {
	multi := Evaluate(Inputs{
		Land:                     true,
		SettlementSuitability:    .95,
		Buildability:             .94,
		FreshWaterAccess:         .93,
		FloodRisk:                .03,
		AgricultureSuitability:   .92,
		TransportAccess:          .94,
		ResourceAccess:           .90,
		WaterwayNavigability:     .91,
		PortSuitability:          .92,
		NavigationBottleneckRisk: .04,
		MiningSuitability:        .91,
		IndustrialValue:          .90,
		ResourceDiversity:        .86,
	})
	if multi.HighAdvantageCount < 3 ||
		multi.RegionalClass != "multi_advantage_region" ||
		multi.AdvantageOverlapRisk < .70 {
		t.Fatalf("multi-advantage site not detected: %+v", multi)
	}

	none := Evaluate(Inputs{
		Land:                  false,
		SettlementSuitability: 1,
		PortSuitability:       1,
		MiningSuitability:     1,
	})
	if none.RegionalClass != "none" ||
		none.DominantAdvantage != "none" ||
		none.HighAdvantageCount != 0 {
		t.Fatalf("water/non-land gained regional advantage: %+v", none)
	}
}

func TestSummarizeTracksOverlapAndRoleCoverage(t *testing.T) {
	scores := []Score{
		{
			BuildingAdvantage:    .90,
			AgricultureAdvantage: .45,
			PortAdvantage:        .20,
			MiningAdvantage:      .35,
			TransportAdvantage:   .65,
			DominantAdvantage:    "building",
			RegionalClass:        "building_region",
			AdvantageOverlapRisk: .10,
		},
		{
			BuildingAdvantage:    .55,
			AgricultureAdvantage: .58,
			PortAdvantage:        .25,
			MiningAdvantage:      .92,
			TransportAdvantage:   .60,
			DominantAdvantage:    "mining",
			RegionalClass:        "mining_region",
			AdvantageOverlapRisk: .35,
		},
		{
			BuildingAdvantage:    .80,
			AgricultureAdvantage: .79,
			PortAdvantage:        .82,
			MiningAdvantage:      .81,
			TransportAdvantage:   .80,
			DominantAdvantage:    "port",
			RegionalClass:        "multi_advantage_region",
			AdvantageOverlapRisk: 1,
		},
		{RegionalClass: "none", DominantAdvantage: "none"},
	}
	summary := Summarize(scores)
	if summary.LandCells != 3 ||
		summary.PotentialCells != 3 ||
		summary.HighOverlapCells != 1 ||
		summary.CoreTripleOverlapCells != 1 ||
		summary.RoleCoverage != 3 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.HighOverlapFraction <= 0 ||
		summary.CoreTripleOverlapFraction <= 0 {
		t.Fatalf("overlap fractions were not populated: %+v", summary)
	}
	if summary.Macro05Satisfied ||
		summary.Macro05CoreTripleLimit != Macro05CoreTripleOverlapLimit {
		t.Fatalf("MACRO-05 negative verdict unexpected: %+v", summary)
	}
}

func TestSummarizeMacro05PassesDistributedAdvantages(t *testing.T) {
	scores := []Score{
		{
			BuildingAdvantage:    .90,
			AgricultureAdvantage: .45,
			PortAdvantage:        .20,
			MiningAdvantage:      .35,
			TransportAdvantage:   .62,
			DominantAdvantage:    "building",
			RegionalClass:        "building_region",
		},
		{
			BuildingAdvantage:    .46,
			AgricultureAdvantage: .52,
			PortAdvantage:        .28,
			MiningAdvantage:      .91,
			TransportAdvantage:   .55,
			DominantAdvantage:    "mining",
			RegionalClass:        "mining_region",
		},
		{
			BuildingAdvantage:    .62,
			AgricultureAdvantage: .58,
			PortAdvantage:        .88,
			MiningAdvantage:      .42,
			TransportAdvantage:   .67,
			DominantAdvantage:    "port",
			RegionalClass:        "port_region",
		},
	}
	summary := Summarize(scores)
	if !summary.Macro05Satisfied {
		t.Fatalf("distributed advantages should satisfy MACRO-05: %+v", summary)
	}
	if summary.CoreTripleOverlapFraction != 0 || summary.RoleCoverage != 3 {
		t.Fatalf("unexpected distributed summary: %+v", summary)
	}
}
