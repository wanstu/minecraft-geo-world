package worldsample

import (
	"errors"
	"fmt"
	"math"

	"geoworld/internal/reconcile"
	"geoworld/internal/sampling"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

func Build(surface *world.Surface, ground *waterbed.Model, cfg Config) (*Model, error) {
	if surface == nil {
		return nil, errors.New("worldsample: macro surface is required")
	}
	if ground == nil {
		return nil, errors.New("worldsample: waterbed ground is required")
	}
	if _, err := ground.Geometry.SquareCellSize(); err != nil {
		return nil, fmt.Errorf("worldsample: geometry: %w", err)
	}
	n, err := world.NodeCount(ground.Geometry.Width, ground.Geometry.Height)
	if err != nil {
		return nil, err
	}
	if surface.Width != ground.Geometry.Width ||
		surface.Height != ground.Geometry.Height ||
		len(surface.LandMask) != n ||
		len(surface.EnclosedBelowSea) != n {
		return nil, errors.New("worldsample: surface dimensions do not match ground geometry")
	}
	if !finite(surface.SeaLevelNormalized) ||
		surface.SeaLevelNormalized <= 0 ||
		surface.SeaLevelNormalized >= 1 {
		return nil, errors.New("worldsample: invalid sea level")
	}

	cfg, err = normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}

	ocean := make([]float64, n)
	for i, land := range surface.LandMask {
		if !land {
			ocean[i] = 1
		}
	}
	oceanGrid, err := sampling.NewScalarGridWithGeometry(ground.Geometry, ocean)
	if err != nil {
		return nil, err
	}

	return &Model{
		Config:             cfg,
		Geometry:           ground.Geometry,
		SeaLevelNormalized: surface.SeaLevelNormalized,
		Ground:             ground,
		OceanEligibility:   oceanGrid,
	}, nil
}

func (m *Model) Sample(worldX, worldZ float64) Sample {
	ground := m.Ground.Sample(worldX, worldZ)
	oceanEligibility := m.OceanEligibility.SampleSmoothstepBilinear(worldX, worldZ)

	out := Sample{
		GroundHeight:     ground.FinalHeight,
		WaterKind:        reconcile.WaterNone,
		OceanEligibility: oceanEligibility,
		RiverWeight:      ground.RiverWeight,
	}

	// Explicit lakes are hydrology-owned basins and override the broad ocean
	// eligibility transition if their local bed is actually below lake datum.
	if ground.WaterKind == reconcile.WaterLake &&
		ground.WaterSurfaceDefined &&
		ground.FinalHeight < ground.WaterSurfaceElevation {
		out.WaterKind = reconcile.WaterLake
		out.WaterSurfaceDefined = true
		out.WaterSurfaceElevation = ground.WaterSurfaceElevation
		out.WaterDepth = ground.WaterSurfaceElevation - ground.FinalHeight
		return out
	}

	// Ocean requires both continuous below-sea terrain and continuous macro
	// ocean eligibility. This keeps enclosed below-sea land dry unless Hydrology
	// explicitly turns it into a lake.
	if oceanEligibility >= m.Config.OceanEligibilityThreshold &&
		ground.FinalHeight < m.SeaLevelNormalized {
		out.WaterKind = reconcile.WaterOcean
		out.WaterSurfaceDefined = true
		out.WaterSurfaceElevation = m.SeaLevelNormalized
		out.WaterDepth = m.SeaLevelNormalized - ground.FinalHeight
		return out
	}

	// River water starts with a constant center depth, smoothly tapered to zero
	// at the bank. If this channel eventually drains into a lake/ocean whose
	// datum sits above that shallow surface, Waterbed exposes the receiving
	// datum and Runtime applies a backwater floor. The floor is also tapered by
	// RiverWeight, so it raises water depth inside the channel without flooding
	// the surrounding broad terrain.
	if ground.WaterKind == reconcile.WaterRiver && ground.RiverWeight > 0 {
		depth := m.Config.RiverCenterDepthNormalized * ground.RiverWeight
		if ground.RiverReceiverSurfaceDefined && ground.RiverReceiverSurface > ground.FinalHeight {
			backwaterDepth := (ground.RiverReceiverSurface - ground.FinalHeight) * ground.RiverWeight
			if backwaterDepth > depth {
				depth = backwaterDepth
			}
		}
		if depth > 1e-12 {
			surface := math.Min(1, ground.FinalHeight+depth)
			if surface > ground.FinalHeight {
				out.WaterKind = reconcile.WaterRiver
				out.WaterSurfaceDefined = true
				out.WaterSurfaceElevation = surface
				out.WaterDepth = surface - ground.FinalHeight
			}
		}
	}
	return out
}

func normalizeConfig(cfg Config) (Config, error) {
	if cfg.RiverCenterDepthNormalized == 0 {
		cfg.RiverCenterDepthNormalized = 0.0025
	}
	if !finite(cfg.RiverCenterDepthNormalized) ||
		cfg.RiverCenterDepthNormalized <= 0 ||
		cfg.RiverCenterDepthNormalized >= 0.05 {
		return Config{}, errors.New("worldsample: river center depth must be finite and inside (0,0.05)")
	}

	if cfg.OceanEligibilityThreshold == 0 {
		cfg.OceanEligibilityThreshold = 0.5
	}
	if !finite(cfg.OceanEligibilityThreshold) ||
		cfg.OceanEligibilityThreshold <= 0 ||
		cfg.OceanEligibilityThreshold >= 1 {
		return Config{}, errors.New("worldsample: ocean eligibility threshold must be finite and inside (0,1)")
	}
	return cfg, nil
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
