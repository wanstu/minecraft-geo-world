package sampling

import (
	"math"
	"testing"
)

// A chunk's block coordinates are converted to absolute world coordinates
// before sampling. Exercise negative chunks and macro-grid boundaries as well
// as ordinary 16-block boundaries; no chunk-relative sampling state is allowed.
func TestScalarGridChunkCoordinateSamplingIsContinuous(t *testing.T) {
	g, err := NewScalarGrid(6, 6, 64, -128, -128, []float64{
		0.12, 0.25, 0.41, 0.52, 0.63, 0.72,
		0.17, 0.26, 0.39, 0.48, 0.58, 0.68,
		0.21, 0.37, 0.45, 0.57, 0.69, 0.79,
		0.32, 0.40, 0.53, 0.62, 0.73, 0.81,
		0.28, 0.36, 0.49, 0.59, 0.65, 0.75,
		0.20, 0.33, 0.44, 0.51, 0.61, 0.70,
	})
	if err != nil {
		t.Fatal(err)
	}

	samplers := []struct {
		name   string
		sample func(float64, float64) float64
	}{
		{"height", g.SampleBilinear},
		{"correction", g.SampleSmoothstepBilinear},
	}
	// Some boundaries are exactly on macro-grid nodes; others are not.
	boundaries := []int{-112, -64, -16, 0, 16, 64, 112, 128}
	for _, sampler := range samplers {
		t.Run(sampler.name, func(t *testing.T) {
			for _, boundary := range boundaries {
				for _, crossX := range []bool{true, false} {
					// The non-crossing coordinate stays away from a grid node.
					for _, fixed := range []float64{-87.5, -12.25, 47.75, 103.5} {
						worldX, worldZ := fixed, fixed
						if crossX {
							worldX = float64(boundary)
						} else {
							worldZ = float64(boundary)
						}
						// From each side the chunk conversion is evaluated independently.
						fromChunk := func(chunk, local int) float64 { return float64(chunk*16 + local) }
						chunkBefore := (boundary - 1) >> 4
						chunkAfter := boundary >> 4
						before := fromChunk(chunkBefore, (boundary-1)-chunkBefore*16)
						after := fromChunk(chunkAfter, boundary-chunkAfter*16)
						if before != float64(boundary-1) || after != float64(boundary) {
							t.Fatalf("invalid chunk split at %d", boundary)
						}
						var left, right, at float64
						if crossX {
							left = sampler.sample(before, worldZ)
							at = sampler.sample(after, worldZ)
							right = sampler.sample(after+1, worldZ)
						} else {
							left = sampler.sample(worldX, before)
							at = sampler.sample(worldX, after)
							right = sampler.sample(worldX, after+1)
						}
						if math.IsNaN(at) || math.IsInf(at, 0) {
							t.Fatalf("non-finite value near %d", boundary)
						}
						// A one-block variation must not make a jump out of proportion to the
						// steepest interpolation supported by these 64-block control points.
						if math.Abs(at-left) > 0.045 || math.Abs(right-at) > 0.045 {
							t.Fatalf("%s axisX=%v boundary=%d fixed=%.2f jump left=%.6f right=%.6f", sampler.name, crossX, boundary, fixed, at-left, right-at)
						}
						// Continuous sampling close to the same boundary is required too.
						const epsilon = 1e-6
						if crossX {
							if math.Abs(sampler.sample(worldX-epsilon, worldZ)-at) > 1e-7 || math.Abs(sampler.sample(worldX+epsilon, worldZ)-at) > 1e-7 {
								t.Fatalf("%s discontinuity at x=%d z=%.2f", sampler.name, boundary, fixed)
							}
						} else if math.Abs(sampler.sample(worldX, worldZ-epsilon)-at) > 1e-7 || math.Abs(sampler.sample(worldX, worldZ+epsilon)-at) > 1e-7 {
							t.Fatalf("%s discontinuity at z=%d x=%.2f", sampler.name, boundary, fixed)
						}
					}
				}
			}
		})
	}
}
