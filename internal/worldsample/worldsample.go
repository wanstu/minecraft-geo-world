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
	if math.Abs(
		ground.Config.RiverCenterDepthNormalized-cfg.RiverCenterDepthNormalized,
	) > 1e-12 {
		return nil, errors.New(
			"worldsample: river center depth conflicts with waterbed profile",
		)
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

	// River water starts with a constant center depth. If a downstream
	// lake/ocean stands above the natural river surface, Waterbed propagates that
	// receiving datum upstream only as far as needed and Runtime applies it as a
	// flat backwater floor. The channel cross-section still owns where that water
	// can exist, so the floor does not flood the surrounding broad terrain.
	if ground.WaterKind == reconcile.WaterRiver && ground.RiverWeight > 0 {
		// River size is already encoded by Waterbed half-width. Make the main
		// stem physically deeper than headwater tributaries while preserving the
		// same smoothly tapered banks.
		widthSpan := math.Max(
			1e-9,
			m.Ground.Config.MaxRiverHalfWidthBlocks-
				m.Ground.Config.MinRiverHalfWidthBlocks,
		)
		widthT := math.Max(
			0,
			math.Min(
				1,
				(ground.RiverHalfWidth-m.Ground.Config.MinRiverHalfWidthBlocks)/
					widthSpan,
			),
		)
		centerDepth := m.Config.RiverCenterDepthNormalized *
			(1 + 2*math.Pow(widthT, 1.15))
		// A real river cross-section has an approximately level water surface;
		// depth changes toward the bank because the bed rises, not because the
		// water plane itself tilts sideways. Use the centerline bed to define one
		// local datum and let the carved bank decide where that datum intersects
		// terrain.
		if ground.RiverBedDefined {
			surface := ground.RiverBedElevation + centerDepth
			if ground.RiverReceiverSurfaceDefined &&
				ground.RiverReceiverSurface > surface &&
				ground.RiverReceiverWeight > 0 {
				// Receiver backwater is a floor, not a downstream ramp.
				// Waterbed has already propagated the datum only through
				// segments whose natural surface lies below the receiver.
				surface = ground.RiverReceiverSurface
			}
			surface = math.Min(1, surface)
			localDepth := surface - ground.FinalHeight
			// Minecraft renders whole water blocks. Keep sub-block-thin fringe
			// dry, but never slope the water plane toward the bank.
			if localDepth >= m.Config.MinRiverWaterDepthNormalized {
				out.WaterKind = reconcile.WaterRiver
				out.WaterSurfaceDefined = true
				out.WaterSurfaceElevation = surface
				out.WaterDepth = localDepth
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
	if cfg.MinRiverWaterDepthNormalized == 0 {
		cfg.MinRiverWaterDepthNormalized = 0.0015
	}
	if !finite(cfg.MinRiverWaterDepthNormalized) ||
		cfg.MinRiverWaterDepthNormalized <= 0 ||
		cfg.MinRiverWaterDepthNormalized >= cfg.RiverCenterDepthNormalized {
		return Config{}, errors.New("worldsample: minimum river water depth must be finite, positive and below center depth")
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
