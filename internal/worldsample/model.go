package worldsample

import (
	"geoworld/internal/reconcile"
	"geoworld/internal/sampling"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

// Config defines Runtime-facing water semantics. River depth remains normalized
// until Phase 6.3/6.4 freezes the normalized-height <-> Minecraft Y mapping.
type Config struct {
	RiverCenterDepthNormalized float64
	OceanEligibilityThreshold  float64
}

type Model struct {
	Config   Config
	Geometry world.GridGeometry

	SeaLevelNormalized float64
	Ground             *waterbed.Model
	OceanEligibility   sampling.ScalarGrid
}

type Sample struct {
	GroundHeight float64

	WaterKind             reconcile.WaterKind
	WaterSurfaceDefined   bool
	WaterSurfaceElevation float64
	WaterDepth            float64

	OceanEligibility float64
	RiverWeight      float64
}
