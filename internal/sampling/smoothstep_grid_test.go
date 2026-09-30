package sampling

import (
	"math"
	"testing"

	"geoworld/internal/world"
)

func TestSmoothstepBilinearHitsControlPointsAndStaysBounded(t *testing.T) {
	g := world.GridGeometry{
		Width: 3, Height: 3,
		OriginX: -128, OriginZ: 256,
		SpanXBlocks: 256, SpanZBlocks: 256,
	}
	values := []float64{
		0.0, 1.0, 0.2,
		0.4, 0.8, 0.1,
		0.3, 0.6, 0.5,
	}
	grid, err := NewScalarGridWithGeometry(g, values)
	if err != nil {
		t.Fatal(err)
	}

	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			wx, wz := g.NodePosition(x, z)
			got := grid.SampleSmoothstepBilinear(wx, wz)
			want := values[z*g.Width+x]
			if math.Abs(got-want) > 1e-12 {
				t.Fatalf("node %d,%d got %.12f want %.12f", x, z, got, want)
			}
		}
	}

	for z := g.OriginZ; z <= g.OriginZ+g.SpanZBlocks; z += 13 {
		for x := g.OriginX; x <= g.OriginX+g.SpanXBlocks; x += 11 {
			got := grid.SampleSmoothstepBilinear(x, z)
			if got < -1e-12 || got > 1+1e-12 {
				t.Fatalf("smoothstep bilinear overshot convex hull at %.1f,%.1f: %.12f", x, z, got)
			}
		}
	}
}

func TestSmoothstepBilinearHasMatchingSlopeAtMacroBoundary(t *testing.T) {
	g := world.GridGeometry{
		Width: 3, Height: 3,
		SpanXBlocks: 256, SpanZBlocks: 256,
	}
	values := []float64{
		0.0, 0.8, 0.2,
		0.3, 1.0, 0.1,
		0.2, 0.7, 0.4,
	}
	grid, err := NewScalarGridWithGeometry(g, values)
	if err != nil {
		t.Fatal(err)
	}

	x := 128.0
	z := 96.0
	h := 1e-3
	at := grid.SampleSmoothstepBilinear(x, z)
	leftSlope := (at - grid.SampleSmoothstepBilinear(x-h, z)) / h
	rightSlope := (grid.SampleSmoothstepBilinear(x+h, z) - at) / h
	if math.Abs(leftSlope-rightSlope) > 1e-6 {
		t.Fatalf("slope mismatch at macro boundary: left=%.12g right=%.12g", leftSlope, rightSlope)
	}
	if math.Abs(leftSlope) > 1e-6 || math.Abs(rightSlope) > 1e-6 {
		t.Fatalf("correction slope should approach zero at boundary: left=%.12g right=%.12g", leftSlope, rightSlope)
	}
}
