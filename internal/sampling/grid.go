package sampling

import (
	"errors"
	"math"
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
	if width < 2 || height < 2 {
		return ScalarGrid{}, errors.New("sampling: grid must be at least 2x2")
	}
	if cellSize <= 0 {
		return ScalarGrid{}, errors.New("sampling: cell size must be positive")
	}
	if len(values) != width*height {
		return ScalarGrid{}, errors.New("sampling: value count does not match grid dimensions")
	}

	copyValues := append([]float64(nil), values...)
	return ScalarGrid{
		Width:    width,
		Height:   height,
		CellSize: cellSize,
		OriginX:  originX,
		OriginZ:  originZ,
		Values:   copyValues,
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
