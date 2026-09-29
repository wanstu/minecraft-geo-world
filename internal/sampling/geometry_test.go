package sampling

import (
	"geoworld/internal/world"
	"math"
	"testing"
)

func TestExtentBasedSamplingIsStableAcrossResolutionAndNegativeCoordinates(t *testing.T) {
	field := func(x, z float64) float64 { return .5 + x/4096 + z/4096 }
	makeGrid := func(width, height int) ScalarGrid {
		geometry := world.GridGeometry{Width: width, Height: height, OriginX: -512, OriginZ: -256, SpanXBlocks: 1024, SpanZBlocks: 512}
		values := make([]float64, width*height)
		for z := 0; z < height; z++ {
			for x := 0; x < width; x++ {
				wx, wz := geometry.NodePosition(x, z)
				values[z*width+x] = field(wx, wz)
			}
		}
		g, err := NewScalarGridWithGeometry(geometry, values)
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	low, high := makeGrid(9, 5), makeGrid(17, 9)
	for _, p := range [][2]float64{{-255, -64}, {-16, 32}, {0, 0}, {123.5, 47.25}} {
		for _, g := range []ScalarGrid{low, high} {
			for _, sample := range []func(float64, float64) float64{g.SampleBilinear, g.SampleCubicBSpline} {
				if got := sample(p[0], p[1]); math.Abs(got-field(p[0], p[1])) > 1e-12 {
					t.Fatalf("resolution changed plane: %v got %.12f", p, got)
				}
			}
		}
	}
	if _, err := NewScalarGridWithGeometry(world.GridGeometry{Width: 3, Height: 3, SpanXBlocks: 128, SpanZBlocks: 256}, make([]float64, 9)); err == nil {
		t.Fatal("silently accepted different X/Z spacing")
	}
	if _, err := NewScalarGrid(2, 2, 128, 0, 0, []float64{0, 0, math.NaN(), 0}); err == nil {
		t.Fatal("accepted non-finite field")
	}
}
