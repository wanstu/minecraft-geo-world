package waterbed

import (
	"math"
	"testing"
)

func TestNavigationSampleAtUsesRenderedWidthAndLocalGradient(t *testing.T) {
	m := &Model{
		Rivers: []RiverSegment{{
			X0: 0, Z0: 0,
			X1: 128, Z1: 0,
			Bed0: 0.60, Bed1: 0.58,
			HalfWidth: 12,
			Profile:   []float64{0.60, 0.595, 0.59, 0.585, 0.58},
		}},
	}
	s, ok := m.NavigationSampleAt(64, 4)
	if !ok {
		t.Fatal("expected navigation sample")
	}
	if s.RiverID != 1 {
		t.Fatalf("river id=%d want=1", s.RiverID)
	}
	if math.Abs(s.BankDistance) > 1e-9 {
		t.Fatalf("inside channel bank distance=%f want=0", s.BankDistance)
	}
	if math.Abs(s.HalfWidth-12) > 1e-9 {
		t.Fatalf("half width=%f want=12", s.HalfWidth)
	}
	if s.Gradient <= 0 || !finite(s.Gradient) {
		t.Fatalf("gradient=%f want positive finite", s.Gradient)
	}
}

func TestNavigationSampleAtChoosesNearestBank(t *testing.T) {
	m := &Model{
		Rivers: []RiverSegment{
			{X0: 0, Z0: 0, X1: 128, Z1: 0, Bed0: 0.60, Bed1: 0.60, HalfWidth: 8},
			{X0: 0, Z0: 80, X1: 128, Z1: 80, Bed0: 0.60, Bed1: 0.60, HalfWidth: 20},
		},
	}
	s, ok := m.NavigationSampleAt(64, 58)
	if !ok {
		t.Fatal("expected navigation sample")
	}
	if s.RiverID != 2 {
		t.Fatalf("river id=%d want=2", s.RiverID)
	}
	if math.Abs(s.BankDistance-2) > 1e-6 {
		t.Fatalf("bank distance=%f want=2", s.BankDistance)
	}
}
