package civilization

import "testing"

func TestEvaluateRoleSpecializations(t *testing.T) {
	agri := Evaluate(Inputs{
		Land:                  true,
		SettlementSuitability: .88, Buildability: .9,
		FreshWaterAccess: .9, FloodRisk: .08,
		AgricultureSuitability: .95, TransportAccess: .72,
		ResourceAccess: .4,
	})
	if agri.AgriculturalSettlementSuitability <= agri.MiningTownSuitability ||
		agri.NodeClass != "agricultural_settlement" {
		t.Fatalf("agricultural site misclassified: %+v", agri)
	}

	port := Evaluate(Inputs{
		Land:                  true,
		SettlementSuitability: .82, Buildability: .85,
		FreshWaterAccess: .9, FloodRisk: .10,
		TransportAccess: .88, WaterwayNavigability: .92,
		PortSuitability: .94, NavigationBottleneckRisk: .05,
		MiningSuitability: .35, IndustrialValue: .55,
	})
	if port.PortCitySuitability <= port.AgriculturalSettlementSuitability ||
		(port.NodeClass != "port_city" && port.NodeClass != "multi_role_hub") {
		t.Fatalf("port site misclassified: %+v", port)
	}

	mine := Evaluate(Inputs{
		Land:                  true,
		SettlementSuitability: .72, Buildability: .7,
		TransportAccess: .72, ResourceAccess: .95,
		MiningSuitability: .94, IndustrialValue: .91,
		ResourceDiversity: .82,
	})
	if mine.MiningTownSuitability <= mine.PortCitySuitability ||
		mine.NodeClass != "mining_town" {
		t.Fatalf("mining site misclassified: %+v", mine)
	}
}

func TestWaterHasNoCivilizationNode(t *testing.T) {
	s := Evaluate(Inputs{
		Land:                  false,
		SettlementSuitability: 1,
		PortSuitability:       1,
		MiningSuitability:     1,
	})
	if s.NodeSuitability != 0 || s.NodeClass != "none" {
		t.Fatalf("water became civilization node: %+v", s)
	}
}

func TestBuildNetworkDeterministicAndSeparated(t *testing.T) {
	const w, h = 7, 7
	cells := make([]Cell, w*h)
	for z := 0; z < h; z++ {
		for x := 0; x < w; x++ {
			i := z*w + x
			score := .20
			if (x == 1 && z == 1) || (x == 5 && z == 1) ||
				(x == 1 && z == 5) || (x == 5 && z == 5) {
				score = .82
			}
			cells[i] = Cell{
				X: float64(x * 256), Z: float64(z * 256),
				GridX: x, GridZ: z, Land: true,
				Score:        Score{NodeSuitability: score, TransportNodeSuitability: score, NodeClass: "transport_node"},
				Buildability: .8, TransportAccess: .8,
			}
		}
	}
	cfg := NetworkConfig{
		Width: w, Height: h, MinNodeSuitability: .6,
		LocalMaximumRadius: 1, MinSeparationBlocks: 300,
		MaxNodes: 10, MaxRelationsPerNode: 2, MaxRelationDistance: 1300,
	}
	a := BuildNetwork(cells, cfg)
	b := BuildNetwork(cells, cfg)
	if len(a.Nodes) != 4 || len(a.Relations) == 0 {
		t.Fatalf("unexpected network: %+v", a)
	}
	if len(a.Nodes) != len(b.Nodes) || len(a.Relations) != len(b.Relations) {
		t.Fatalf("network not deterministic: a=%+v b=%+v", a, b)
	}
	for _, rel := range a.Relations {
		if rel.CorridorSuitability <= 0 || rel.CorridorSuitability > 1 {
			t.Fatalf("invalid relation: %+v", rel)
		}
	}
}
