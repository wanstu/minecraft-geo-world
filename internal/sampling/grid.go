package sampling

import (
	"errors"
	"math"

	"geoworld/internal/world"
)

// ScalarGrid represents a continuous scalar field backed by a regular macro grid.
//
// Values are stored row-major. CellSize is expressed in world-coordinate units
// (Minecraft blocks later). OriginX/OriginZ specify the world coordinate of
// grid node (0,0).
//
// SampleBilinear is intentionally independent from Chunk coordinates. A caller
// always asks for a world position, so neighbouring Chunks sample exactly the
// same field at their shared boundary.
type ScalarGrid struct {
	Width    int
	Height   int
	CellSize float64
	OriginX  float64
	OriginZ  float64
	Values   []float64
}

func NewScalarGrid(width, height int, cellSize, originX, originZ float64, values []float64) (ScalarGrid, error) {
	return NewScalarGridWithGeometry(world.GridGeometry{
		Width: width, Height: height, OriginX: originX, OriginZ: originZ,
		SpanXBlocks: float64(width-1) * cellSize, SpanZBlocks: float64(height-1) * cellSize,
	}, values)
}

// NewScalarGridWithGeometry binds a physical extent to sampling resolution.
// This legacy sampler intentionally rejects anisotropic spacing.
func NewScalarGridWithGeometry(geometry world.GridGeometry, values []float64) (ScalarGrid, error) {
	cellSize, err := geometry.SquareCellSize()
	if err != nil {
		return ScalarGrid{}, err
	}
	n, err := world.NodeCount(geometry.Width, geometry.Height)
	if err != nil {
		return ScalarGrid{}, err
	}
	if len(values) != n {
		return ScalarGrid{}, errors.New("sampling: value count does not match grid dimensions")
	}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ScalarGrid{}, errors.New("sampling: values must be finite")
		}
	}
	return ScalarGrid{
		Width: geometry.Width, Height: geometry.Height, CellSize: cellSize,
		OriginX: geometry.OriginX, OriginZ: geometry.OriginZ,
		Values: append([]float64(nil), values...),
	}, nil
}

// SampleBilinear samples the scalar field at an absolute world coordinate.
//
// The current V1 behaviour clamps positions to the finite grid edge. Runtime
// topology code will later decide whether positions outside the GeoWorld model
// should instead be handled by storm/frozen-ocean generators.
func (g ScalarGrid) SampleBilinear(worldX, worldZ float64) float64 {
	gx := (worldX - g.OriginX) / g.CellSize
	gz := (worldZ - g.OriginZ) / g.CellSize

	gx = clamp(gx, 0, float64(g.Width-1))
	gz = clamp(gz, 0, float64(g.Height-1))

	x0 := int(math.Floor(gx))
	z0 := int(math.Floor(gz))
	x1 := minInt(x0+1, g.Width-1)
	z1 := minInt(z0+1, g.Height-1)

	tx := gx - float64(x0)
	tz := gz - float64(z0)

	v00 := g.at(x0, z0)
	v10 := g.at(x1, z0)
	v01 := g.at(x0, z1)
	v11 := g.at(x1, z1)

	top := lerp(v00, v10, tx)
	bottom := lerp(v01, v11, tx)
	return lerp(top, bottom, tz)
}

// SampleSmoothstepBilinear is an interpolating correction sampler.
//
// It keeps the convex-hull/no-overshoot property of bilinear interpolation,
// passes through every macro control point exactly, and applies smoothstep to
// both local coordinates. The correction derivative therefore approaches zero
// on every macro-cell boundary from both sides. This is useful for sparse
// correction overlays: the original C2 terrain carries the broad slope while
// the overlay can hit exact control-point constraints without adding a visible
// first-derivative kink at macro boundaries.
func (g ScalarGrid) SampleSmoothstepBilinear(worldX, worldZ float64) float64 {
	gx := (worldX - g.OriginX) / g.CellSize
	gz := (worldZ - g.OriginZ) / g.CellSize

	gx = clamp(gx, 0, float64(g.Width-1))
	gz = clamp(gz, 0, float64(g.Height-1))

	x0 := int(math.Floor(gx))
	z0 := int(math.Floor(gz))
	x1 := minInt(x0+1, g.Width-1)
	z1 := minInt(z0+1, g.Height-1)

	tx := smoothstepUnit(gx - float64(x0))
	tz := smoothstepUnit(gz - float64(z0))

	v00 := g.at(x0, z0)
	v10 := g.at(x1, z0)
	v01 := g.at(x0, z1)
	v11 := g.at(x1, z1)

	top := lerp(v00, v10, tx)
	bottom := lerp(v01, v11, tx)
	return lerp(top, bottom, tz)
}

func (g ScalarGrid) at(x, z int) float64 {
	return g.Values[z*g.Width+x]
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func smoothstepUnit(t float64) float64 {
	t = clamp(t, 0, 1)
	return t * t * (3 - 2*t)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SampleCubicBSpline samples the field with a uniform cubic B-spline.
//
// Unlike interpolating cubics such as Catmull-Rom, the cubic B-spline basis is
// non-negative and sums to one. The result therefore remains inside the convex
// hull of the neighbouring control values and cannot create interpolation
// spikes. In the grid interior the field is C2 continuous.
//
// Macro values are treated as control points rather than mandatory pass-through
// points. This is intentional for final terrain shaping.
func (g ScalarGrid) SampleCubicBSpline(worldX, worldZ float64) float64 {
	gx := (worldX - g.OriginX) / g.CellSize
	gz := (worldZ - g.OriginZ) / g.CellSize

	gx = clamp(gx, 0, float64(g.Width-1))
	gz = clamp(gz, 0, float64(g.Height-1))

	xBase := int(math.Floor(gx))
	zBase := int(math.Floor(gz))
	tx := gx - float64(xBase)
	tz := gz - float64(zBase)

	wx := cubicBSplineWeights(tx)
	wz := cubicBSplineWeights(tz)

	sum := 0.0
	for dz := 0; dz < 4; dz++ {
		z := clampInt(zBase+dz-1, 0, g.Height-1)
		for dx := 0; dx < 4; dx++ {
			x := clampInt(xBase+dx-1, 0, g.Width-1)
			sum += g.at(x, z) * wx[dx] * wz[dz]
		}
	}
	return sum
}

func cubicBSplineWeights(t float64) [4]float64 {
	t = clamp(t, 0, 1)
	t2 := t * t
	t3 := t2 * t
	oneMinus := 1 - t

	return [4]float64{
		oneMinus * oneMinus * oneMinus / 6,
		(3*t3 - 6*t2 + 4) / 6,
		(-3*t3 + 3*t2 + 3*t + 1) / 6,
		t3 / 6,
	}
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
