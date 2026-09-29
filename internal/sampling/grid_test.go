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
