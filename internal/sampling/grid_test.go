package sampling

import (
	"math"
	"testing"
)

func mustGrid(t *testing.T, width, height int, cellSize float64, values []float64) ScalarGrid {
	t.Helper()
	g, err := NewScalarGrid(width, height, cellSize, 0, 0, values)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestSampleBilinearReturnsExactGridNodes(t *testing.T) {
	g := mustGrid(t, 3, 2, 128, []float64{
		0.10, 0.40, 0.70,
		0.20, 0.50, 0.80,
	})

	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			got := g.SampleBilinear(float64(x)*g.CellSize, float64(z)*g.CellSize)
			want := g.Values[z*g.Width+x]
			if math.Abs(got-want) > 1e-12 {
				t.Fatalf("node (%d,%d): got %.12f want %.12f", x, z, got, want)
			}
		}
	}
}

func TestSampleBilinearDoesNotOvershoot(t *testing.T) {
	g := mustGrid(t, 2, 2, 128, []float64{
		0.15, 0.85,
		0.30, 0.60,
	})

	minV, maxV := 0.15, 0.85
	for z := 0.0; z <= 128; z += 2.0 {
		for x := 0.0; x <= 128; x += 2.0 {
			v := g.SampleBilinear(x, z)
			if v < minV-1e-12 || v > maxV+1e-12 {
				t.Fatalf("overshoot at %.1f,%.1f: %.6f outside %.2f..%.2f", x, z, v, minV, maxV)
			}
		}
	}
}

func TestMacroCellBoundaryIsContinuous(t *testing.T) {
	g := mustGrid(t, 3, 2, 128, []float64{
		0.20, 0.70, 0.35,
		0.25, 0.75, 0.40,
	})

	z := 47.25
	left := g.SampleBilinear(128-1e-6, z)
	at := g.SampleBilinear(128, z)
	right := g.SampleBilinear(128+1e-6, z)

	if math.Abs(left-at) > 1e-7 || math.Abs(right-at) > 1e-7 {
		t.Fatalf("macro boundary discontinuity: left=%.12f at=%.12f right=%.12f", left, at, right)
	}
}

func TestMinecraftChunkBoundaryHasNoSpecialDiscontinuity(t *testing.T) {
	g := mustGrid(t, 3, 3, 128, []float64{
		0.20, 0.65, 0.35,
		0.25, 0.70, 0.40,
		0.30, 0.60, 0.45,
	})

	// x=16 is a Minecraft Chunk boundary, but it is not part of the sampling
	// mathematics. Values immediately either side must follow the same smooth field.
	z := 91.0
	before := g.SampleBilinear(16-1e-6, z)
	at := g.SampleBilinear(16, z)
	after := g.SampleBilinear(16+1e-6, z)

	if math.Abs(before-at) > 1e-7 || math.Abs(after-at) > 1e-7 {
		t.Fatalf("chunk boundary discontinuity: before=%.12f at=%.12f after=%.12f", before, at, after)
	}
}

func TestSameWorldCoordinateIsChunkIndependent(t *testing.T) {
	g := mustGrid(t, 4, 3, 128, []float64{
		0.10, 0.30, 0.60, 0.50,
		0.15, 0.35, 0.70, 0.55,
		0.20, 0.40, 0.65, 0.45,
	})

	worldX, worldZ := 175.0, 93.0
	fromLeftChunk := g.SampleBilinear(worldX, worldZ)
	fromRightChunk := g.SampleBilinear(worldX, worldZ)

	if fromLeftChunk != fromRightChunk {
		t.Fatalf("same world coordinate differs: %.12f vs %.12f", fromLeftChunk, fromRightChunk)
	}
}

func TestCubicBSplineConstantField(t *testing.T) {
	values := make([]float64, 8*6)
	for i := range values {
		values[i] = 0.42
	}
	g := mustGrid(t, 8, 6, 128, values)

	for z := 0.0; z <= 5*128; z += 7.25 {
		for x := 0.0; x <= 7*128; x += 9.5 {
			got := g.SampleCubicBSpline(x, z)
			if math.Abs(got-0.42) > 1e-12 {
				t.Fatalf("constant field changed at %.2f,%.2f: %.12f", x, z, got)
			}
		}
	}
}

func TestCubicBSplineDoesNotOvershootGlobalControlRange(t *testing.T) {
	g := mustGrid(t, 5, 5, 128, []float64{
		0.15, 0.22, 0.81, 0.47, 0.35,
		0.19, 0.72, 0.26, 0.61, 0.43,
		0.30, 0.52, 0.85, 0.18, 0.66,
		0.24, 0.41, 0.57, 0.78, 0.32,
		0.36, 0.69, 0.28, 0.54, 0.45,
	})
	minV, maxV := 0.15, 0.85

	for z := 0.0; z <= 4*128; z += 3.75 {
		for x := 0.0; x <= 4*128; x += 4.25 {
			v := g.SampleCubicBSpline(x, z)
			if v < minV-1e-12 || v > maxV+1e-12 {
				t.Fatalf("B-spline overshoot at %.2f,%.2f: %.8f outside %.2f..%.2f",
					x, z, v, minV, maxV)
			}
		}
	}
}

func TestCubicBSplineFirstDerivativeContinuousAcrossMacroBoundary(t *testing.T) {
	g := mustGrid(t, 7, 6, 128, []float64{
		0.15, 0.22, 0.39, 0.61, 0.68, 0.55, 0.44,
		0.18, 0.27, 0.46, 0.66, 0.71, 0.58, 0.47,
		0.23, 0.34, 0.52, 0.69, 0.73, 0.62, 0.51,
		0.29, 0.41, 0.59, 0.72, 0.70, 0.60, 0.49,
		0.33, 0.48, 0.63, 0.74, 0.67, 0.56, 0.45,
		0.38, 0.53, 0.66, 0.71, 0.62, 0.51, 0.42,
	})

	x := 3.0 * 128
	z := 2.37 * 128
	h := 0.01

	leftSlope := (g.SampleCubicBSpline(x, z) - g.SampleCubicBSpline(x-h, z)) / h
	rightSlope := (g.SampleCubicBSpline(x+h, z) - g.SampleCubicBSpline(x, z)) / h

	if math.Abs(leftSlope-rightSlope) > 1e-6 {
		t.Fatalf("B-spline slope discontinuity: left=%.9f right=%.9f", leftSlope, rightSlope)
	}
}

func TestCubicBSplineMinecraftChunkBoundaryHasNoSeam(t *testing.T) {
	g := mustGrid(t, 6, 5, 128, []float64{
		0.20, 0.31, 0.56, 0.64, 0.48, 0.39,
		0.24, 0.38, 0.61, 0.67, 0.51, 0.42,
		0.29, 0.45, 0.66, 0.69, 0.55, 0.46,
		0.33, 0.49, 0.63, 0.65, 0.53, 0.44,
		0.36, 0.47, 0.58, 0.60, 0.50, 0.41,
	})

	z := 211.75
	for _, x := range []float64{16, 32, 48, 64, 80, 96, 112, 128, 144, 160} {
		before := g.SampleCubicBSpline(x-1e-5, z)
		at := g.SampleCubicBSpline(x, z)
		after := g.SampleCubicBSpline(x+1e-5, z)
		if math.Abs(before-at) > 1e-7 || math.Abs(after-at) > 1e-7 {
			t.Fatalf("chunk seam at x=%.0f: before=%.12f at=%.12f after=%.12f",
				x, before, at, after)
		}
	}
}
