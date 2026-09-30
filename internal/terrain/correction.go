package terrain

import (
	"errors"

	"geoworld/internal/sampling"
)

// WithHeightCorrection returns a shallow copy of base with an immutable
// broad-surface correction grid attached. Existing scalar grids are immutable
// value fields and are safely shared by the copy.
//
// The correction affects only Sample.FinalHeight. Landform weights, regional
// detail diagnostics and channel/basin geometry remain separate.
func WithHeightCorrection(base *Model, correction []float64) (*Model, error) {
	if base == nil {
		return nil, errors.New("terrain: base model is required")
	}
	grid, err := sampling.NewScalarGridWithGeometry(base.Geometry, correction)
	if err != nil {
		return nil, err
	}
	clone := *base
	clone.HeightCorrection = &grid
	return &clone, nil
}
