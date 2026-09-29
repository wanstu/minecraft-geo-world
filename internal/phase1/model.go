package phase1

import "geoworld/internal/world"

type CrustType uint8

const (
	Continental CrustType = iota
	Oceanic
)

type BoundaryType uint8

const (
	NoBoundary BoundaryType = iota
	Convergent
	Divergent
	Transform
)

type TectonicFeature uint8

const (
	NoFeature TectonicFeature = iota
	CollisionOrogen
	VolcanicArc
	Trench
	IslandArc
	MidOceanRidge
	ContinentalRift
	TransformFault
)

type MacroTerrain uint8

const (
	TerrainUnknown MacroTerrain = iota
	ContinentalInterior
	ContinentalCoast
	ContinentalShelf
	ContinentalSlope
	OceanBasin
	MountainBelt
	Plateau
	TrenchZone
	IslandArcZone
	MidOceanRidgeZone
	RiftValleyZone
)

type Vec2 struct {
	X float64
	Y float64
}

type Plate struct {
	ID       int
	Center   Vec2
	Velocity Vec2
}

const DefaultReferenceCellSizeBlocks = 128.0

type Config struct {
	Seed       int64
	Width      int
	Height     int
	PlateCount int

	// Physical world extent is independent from macro-grid resolution. Zero spans
	// preserve the legacy default of 128 blocks between neighbouring macro nodes.
	OriginX     float64
	OriginZ     float64
	SpanXBlocks float64
	SpanZBlocks float64

	// OceanFraction currently controls oceanic crust area, not inundated ocean area.
	OceanFraction           float64
	EdgeOceanBufferFraction float64
}

type World struct {
	// Surface is derived from elevation and ocean connectivity, never crust type.
	Surface           *world.Surface
	Geometry          world.GridGeometry
	Config            Config
	Plates            []Plate
	RawPlateID        []int
	PlateID           []int
	Crust             []CrustType
	CrustThickness    []float64
	CoastDistance     []float64
	BaseElevation     []float64
	TerrainZone       []MacroTerrain
	LandPotential     []float64
	Boundary          []BoundaryType
	BoundaryStrength  []float64
	OceanAge          []float64
	Feature           []TectonicFeature
	TectonicIntensity []float64
	TectonicInfluence []float64
	InfluenceFeature  []TectonicFeature
	Elevation         []float64
}
