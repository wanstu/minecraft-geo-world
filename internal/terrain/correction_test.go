package terrain

import (
	"math"
	"testing"

	"geoworld/internal/world"
)

func TestHeightCorrectionHitsMacroTargetsExactly(t *testing.T) {
	in := uniformInput(3, 3, 0.60)
	in.Geometry = world.GridGeometry{
		Width: 3, Height: 3,
		OriginX: -128, OriginZ: 256,
		SpanXBlocks: 256, SpanZBlocks: 256,
	}
	base, err := Generate(in, Config{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	correction := []float64{
		0.00, -0.05, 0.00,
		0.02, 0.00, -0.03,
		0.00, 0.04, 0.00,
	}
	adjusted, err := WithHeightCorrection(base, correction)
	if err != nil {
		t.Fatal(err)
	}

	for z := 0; z < in.Height; z++ {
		for x := 0; x < in.Width; x++ {
			wx, wz := in.Geometry.NodePosition(x, z)
			raw := base.SampleNormalized(wx, wz)
			got := adjusted.Sample(wx, wz)
			want := raw + correction[z*in.Width+x]
			if math.Abs(got.HeightCorrection-correction[z*in.Width+x]) > 1e-12 {
				t.Fatalf("node %d,%d correction=%.12f want=%.12f",
					x, z, got.HeightCorrection, correction[z*in.Width+x])
			}
			if math.Abs(got.FinalHeight-want) > 1e-12 {
				t.Fatalf("node %d,%d final=%.12f want=%.12f", x, z, got.FinalHeight, want)
			}
		}
	}
}

func TestHeightCorrectionDoesNotAddMacroBoundarySlopeKink(t *testing.T) {
	in := uniformInput(3, 3, 0.60)
	in.Geometry = world.GridGeometry{
		Width: 3, Height: 3,
		SpanXBlocks: 256, SpanZBlocks: 256,
	}
	base, err := Generate(in, Config{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	adjusted, err := WithHeightCorrection(base, []float64{
		0.00, 0.05, -0.02,
		0.01, -0.04, 0.03,
		0.00, 0.02, 0.00,
	})
	if err != nil {
		t.Fatal(err)
	}

	x := 128.0
	z := 80.0
	h := 1e-3
	rawAt := base.SampleNormalized(x, z)
	rawLeft := (rawAt - base.SampleNormalized(x-h, z)) / h
	rawRight := (base.SampleNormalized(x+h, z) - rawAt) / h

	at := adjusted.SampleNormalized(x, z)
	left := (at - adjusted.SampleNormalized(x-h, z)) / h
	right := (adjusted.SampleNormalized(x+h, z) - at) / h

	if math.Abs((left-right)-(rawLeft-rawRight)) > 1e-6 {
		t.Fatalf("correction added slope kink: raw mismatch=%.12g adjusted mismatch=%.12g",
			rawLeft-rawRight, left-right)
	}
}

func TestWithHeightCorrectionRejectsWrongSize(t *testing.T) {
	in := uniformInput(3, 3, 0.60)
	base, err := Generate(in, Config{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WithHeightCorrection(base, make([]float64, 8)); err == nil {
		t.Fatal("expected wrong correction size to fail")
	}
}
