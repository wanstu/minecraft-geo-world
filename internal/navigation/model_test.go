package navigation

import "testing"

func TestMajorLowGradientRiverIsNavigable(t *testing.T) {
	s := Evaluate(Inputs{
		WaterKind:            "River",
		RiverAvailable:       true,
		RiverHalfWidth:       28,
		RiverGradient:        0.00002,
		WaterDepthNormalized: 0.012,
	})
	if s.WaterwayNavigability < 0.85 {
		t.Fatalf("major river should be highly navigable: %+v", s)
	}
	if s.NavigationBottleneckRisk > 0.25 {
		t.Fatalf("major low-gradient river should not be a bottleneck: %+v", s)
	}
	if s.WaterwayClass != "major_river" {
		t.Fatalf("class=%s want major_river", s.WaterwayClass)
	}
}

func TestNarrowSteepRiverIsBottleneck(t *testing.T) {
	s := Evaluate(Inputs{
		WaterKind:      "River",
		RiverAvailable: true,
		RiverHalfWidth: 7,
		RiverGradient:  0.00042,
	})
	if s.WaterwayNavigability > 0.45 {
		t.Fatalf("narrow steep reach unexpectedly navigable: %+v", s)
	}
	if s.NavigationBottleneckRisk < 0.80 {
		t.Fatalf("narrow steep reach should be a bottleneck: %+v", s)
	}
}

func TestSafeBuildableRiverPortBeatsFloodedBank(t *testing.T) {
	base := Inputs{
		Land: true, WaterKind: "None",
		RiverAvailable:        true,
		RiverHalfWidth:        24,
		RiverGradient:         0.00003,
		DistanceToRiverBlocks: 80,
		HasRiverDistance:      true,
		SettlementSuitability: 0.82,
	}
	safe := Evaluate(func() Inputs {
		v := base
		v.Buildability = 0.92
		v.FloodRisk = 0.08
		return v
	}())
	flooded := Evaluate(func() Inputs {
		v := base
		v.DistanceToRiverBlocks = 4
		v.Buildability = 0.90
		v.FloodRisk = 0.94
		return v
	}())
	if safe.PortSuitability < 0.70 {
		t.Fatalf("safe bank should be a good river port: %+v", safe)
	}
	if flooded.PortSuitability >= safe.PortSuitability {
		t.Fatalf("flooded bank should not beat safe bank: safe=%+v flooded=%+v", safe, flooded)
	}
}

func TestSeaPortUsesCoastAccess(t *testing.T) {
	s := Evaluate(Inputs{
		Land: true, WaterKind: "None",
		DistanceToCoastBlocks: 96, HasCoastDistance: true,
		Buildability: 0.86, FloodRisk: 0.12, SettlementSuitability: 0.78,
	})
	if s.PortType != "sea" || s.PortSuitability < 0.70 {
		t.Fatalf("expected strong sea-port site: %+v", s)
	}
}

func TestWaterIsNotReportedAsPortSite(t *testing.T) {
	s := Evaluate(Inputs{
		Land: true, WaterKind: "Ocean",
		DistanceToCoastBlocks: 0, HasCoastDistance: true,
		Buildability: 1, FloodRisk: 0, SettlementSuitability: 1,
	})
	if s.PortSuitability != 0 {
		t.Fatalf("water must not have port suitability: %+v", s)
	}
	if s.PortType != "none" {
		t.Fatalf("water port type=%s want=none", s.PortType)
	}
}

func TestFarLandHasNoRiverWaterwayClass(t *testing.T) {
	s := Evaluate(Inputs{
		Land: true, WaterKind: "None",
		RiverAvailable:        true,
		RiverHalfWidth:        30,
		RiverGradient:         0,
		DistanceToRiverBlocks: 900,
		HasRiverDistance:      true,
	})
	if s.WaterwayNavigability != 0 {
		t.Fatalf("far land navigability=%f want=0", s.WaterwayNavigability)
	}
	if s.WaterwayClass != "none" {
		t.Fatalf("far land class=%s want=none", s.WaterwayClass)
	}
}

func TestScoresNormalized(t *testing.T) {
	s := Evaluate(Inputs{
		Land: true, WaterKind: "River", RiverAvailable: true,
		RiverHalfWidth: 999, RiverGradient: -2,
		DistanceToRiverBlocks: -10, HasRiverDistance: true,
		DistanceToLakeBlocks: -10, HasLakeDistance: true,
		DistanceToCoastBlocks: -10, HasCoastDistance: true,
		Buildability: 99, FloodRisk: -4, SettlementSuitability: 7,
	})
	for _, v := range []float64{s.WaterwayNavigability, s.PortSuitability, s.NavigationBottleneckRisk} {
		if v < 0 || v > 1 {
			t.Fatalf("score outside [0,1]: %+v", s)
		}
	}
}
