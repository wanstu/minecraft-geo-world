package runtimeprofile

import (
	"geoworld/internal/ecology"
	"geoworld/internal/mcprofile"
	"geoworld/internal/reconcile"
	"geoworld/internal/world"
	"geoworld/internal/worldsample"
)

// Model resolves macro ecology/profile labels for one final Runtime coordinate.
// Climate/ecology remain macro-scale; this layer only ensures that discrete
// Minecraft-facing labels are selected from the same land/ocean semantic side
// as the final Ground/Water sample.
type Model struct {
	Geometry        world.GridGeometry
	Ecology         *ecology.Model
	Profile         *mcprofile.Model
	Runtime         *worldsample.Model
	BiomeContinuity float64
}

type Sample struct {
	World worldsample.Sample

	SourceNode   int
	SourceWeight float64
	SourceLand   bool

	EcoRegion        ecology.Region
	RegionConfidence float64

	BiomeFamily       mcprofile.BiomeFamily
	TerrainVariant    mcprofile.TerrainVariant
	SurfaceProfile    mcprofile.SurfaceProfile
	VegetationProfile mcprofile.VegetationProfile
	ProfileConfidence float64

	SpecialFeature         string
	SpecialFeatureStrength float64

	// SemanticFallback means no macro corner on the desired land/ocean side was
	// available and the strongest corner had to be used. It should remain false
	// for valid in-domain final-world samples.
	SemanticFallback bool
	// SemanticMismatch means the chosen macro profile itself contradicts the
	// final Runtime side (for example Ocean water with a terrestrial biome).
	SemanticMismatch bool
}

func (s Sample) WaterKind() reconcile.WaterKind { return s.World.WaterKind }
