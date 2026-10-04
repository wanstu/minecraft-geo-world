package worldsample

import (
	"geoworld/internal/reconcile"
	"geoworld/internal/sampling"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

// Config defines Runtime-facing water semantics. RiverCenterDepthNormalized is
// the headwater/base center depth; larger Waterbed channels scale it upward
// deterministically while keeping one level cross-section water datum.
type Config struct {
	RiverCenterDepthNormalized   float64
	MinRiverWaterDepthNormalized float64 `json:",omitempty"`
	OceanEligibilityThreshold    float64
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
