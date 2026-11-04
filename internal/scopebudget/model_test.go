package scopebudget

import "testing"

func TestEvaluateAdequateWorldScope(t *testing.T) {
	got := Evaluate(Inputs{
		SpanXBlocks: 40832,
		SpanZBlocks: 25472,
		Components: []Component{
			{Name: "phase1_macro_transition", Blocks: 5376},
			{Name: "climate_coastal_decay", Blocks: 2304},
			{Name: "mountain_influence", Blocks: 640},
		},
	})
	if !got.Macro06Satisfied {
		t.Fatalf("expected MACRO-06 to pass: %+v", got)
	}
	if got.MaxScaleName != "phase1_macro_transition" ||
		got.MaxScaleBlocks != 5376 {
		t.Fatalf("unexpected max scale: %+v", got)
	}
	if got.ScaleCapacity < 4 || got.InteriorFraction < .50 {
		t.Fatalf("scope budget unexpectedly cramped: %+v", got)
	}
}

func TestEvaluateCrowdedWorldScope(t *testing.T) {
	got := Evaluate(Inputs{
		SpanXBlocks: 16000,
		SpanZBlocks: 16000,
		Components: []Component{
			{Name: "phase1_macro_transition", Blocks: 5376},
			{Name: "climate_coastal_decay", Blocks: 2304},
		},
	})
	if got.Macro06Satisfied {
		t.Fatalf("crowded world should fail MACRO-06: %+v", got)
	}
	if got.Status != "insufficient" {
		t.Fatalf("status=%q want insufficient", got.Status)
	}
	if got.ShortfallBlocks <= 0 {
		t.Fatalf("expected positive shortfall: %+v", got)
	}
}

func TestEvaluateSortsComponentsDeterministically(t *testing.T) {
	got := Evaluate(Inputs{
		SpanXBlocks: 30000,
		SpanZBlocks: 30000,
		Components: []Component{
			{Name: "b", Blocks: 1000},
			{Name: "a", Blocks: 1000},
			{Name: "c", Blocks: 2000},
		},
	})
	if len(got.Components) != 3 ||
		got.Components[0].Name != "c" ||
		got.Components[1].Name != "a" ||
		got.Components[2].Name != "b" {
		t.Fatalf("unexpected component order: %+v", got.Components)
	}
}
