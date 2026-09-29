// Package world defines shared coordinate and surface contracts.
// It does not depend on simulation stages or Minecraft loaders.
package world

import (
	"errors"
	"math"
)

// DefaultSeaLevelNormalized is the legacy macro height datum. It is not
// Minecraft Y or a physical elevation in metres.
const DefaultSeaLevelNormalized = 0.5

// DefaultMacroSpacingBlocks is the physical spacing represented by one legacy
// macro-grid step. New code should carry GridGeometry explicitly and use this
// constant only when translating old cell-based defaults.
const DefaultMacroSpacingBlocks = 128.0

// GridGeometry describes node locations, not cell-centred pixels.
// Spans are in Minecraft blocks and remain independent of sampling resolution.
type GridGeometry struct {
	Width       int
	Height      int
	OriginX     float64
	OriginZ     float64
	SpanXBlocks float64
	SpanZBlocks float64
}

// NodeCount validates dimensions before multiplying or allocating.
func NodeCount(width, height int) (int, error) {
	if width < 2 || height < 2 {
		return 0, errors.New("world: grid must be at least 2x2")
	}
	maxInt := int(^uint(0) >> 1)
	if width > maxInt/height {
		return 0, errors.New("world: grid dimensions overflow")
	}
	return width * height, nil
}

func (g GridGeometry) Validate() error {
	if _, err := NodeCount(g.Width, g.Height); err != nil {
		return err
	}
	for _, v := range []float64{g.OriginX, g.OriginZ, g.SpanXBlocks, g.SpanZBlocks} {
		if !finite(v) {
			return errors.New("world: geometry must be finite")
		}
	}
	if g.SpanXBlocks <= 0 || g.SpanZBlocks <= 0 {
		return errors.New("world: spans must be positive")
	}
	endX, endZ := g.OriginX+g.SpanXBlocks, g.OriginZ+g.SpanZBlocks
	dx, dz := g.SpacingX(), g.SpacingZ()
	if !finite(endX) || !finite(endZ) || dx <= 0 || dz <= 0 ||
		g.OriginX+dx == g.OriginX || g.OriginZ+dz == g.OriginZ ||
		endX-dx == endX || endZ-dz == endZ {
		return errors.New("world: node spacing cannot be represented at this origin")
	}
	return nil
}

func (g GridGeometry) SpacingX() float64 { return g.SpanXBlocks / float64(g.Width-1) }
func (g GridGeometry) SpacingZ() float64 { return g.SpanZBlocks / float64(g.Height-1) }

// NodePosition requires validated geometry and in-range node indices.
func (g GridGeometry) NodePosition(x, z int) (float64, float64) {
	return g.OriginX + float64(x)*g.SpacingX(), g.OriginZ + float64(z)*g.SpacingZ()
}

// SquareCellSize is the explicit compatibility gate for legacy scalar grids.
// Rectangular worlds are supported when X and Z node spacing is the same.
// Anisotropic spacing must not silently stretch one axis.
func (g GridGeometry) SquareCellSize() (float64, error) {
	if err := g.Validate(); err != nil {
		return 0, err
	}
	dx, dz := g.SpacingX(), g.SpacingZ()
	if math.Abs(dx-dz) > 1e-12*math.Max(dx, dz) {
		return 0, errors.New("world: legacy sampler requires equal X/Z node spacing")
	}
	return dx, nil
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
