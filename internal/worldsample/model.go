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
	// R8.1c: opt-in sea-connected river and low-bank backwater; zero is legacy.
	// Version 2 keeps the formal river corridor semantically River while still
	// sharing the receiving sea datum.
	CoastalBackwaterVersion uint8 `json:",omitempty"`
	// Slice E: opt-in Minecraft-quantized navigation depth for the inner river
	// corridor. Old persisted worlds omit both fields and retain their exact
	// historical ground/water geometry.
	RiverNavigationVersion       uint8 `json:",omitempty"`
	MinNavigableRiverDepthBlocks uint8 `json:",omitempty"`
	// OceanBathymetryVersion gives new worlds a navigable nearshore seabed.
	OceanBathymetryVersion uint8 `json:",omitempty"`
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
	// CoastalSeaReceiver marks the persisted OceanMouth influence corridor.
	// It is metadata for Minecraft biome realization / navigation and does not
	// by itself change water ownership.
	CoastalSeaReceiver bool
}
