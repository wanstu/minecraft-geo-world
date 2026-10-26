package waterbed

import (
	"geoworld/internal/reconcile"
	"geoworld/internal/terrain"
	"geoworld/internal/world"
)

type Config struct {
	// River half-width is expressed directly in Minecraft blocks. RiverStrength
	// maps continuously from Min to Max for the strong-river range selected by
	// reconciliation.
	MinRiverHalfWidthBlocks float64
	MaxRiverHalfWidthBlocks float64

	// RiverProfileStepBlocks controls precomputed centerline profile resolution.
	// It is a physical Minecraft-block distance, independent of macro-grid size.
	RiverProfileStepBlocks float64

	// RiverGeometryVersion versions only horizontal river footprint geometry.
	// Version 1 (v1.1.5+) prevents wide low-gradient D8 diagonal reaches from
	// remaining perfectly grid-locked when the legacy random straight-reach
	// branch fires. Version 3 (v1.1.8+) scales lake-mouth transition breadth
	// with large rivers instead of clipping them at the legacy fixed cap.
	// Omitted/zero preserves all older worlds exactly.
	RiverGeometryVersion uint8 `json:",omitempty"`

	// MinRiverBedDepthNormalized guarantees that a river centerline is actually
	// cut into the broad terrain even when the broad terrain already slopes
	// downhill. Without this minimum trench depth, Runtime water can sit one or
	// two blocks on top of an uncarved surface.
	MinRiverBedDepthNormalized float64 `json:",omitempty"`

	// RiverCenterDepthNormalized mirrors the Runtime river depth basis while
	// Waterbed constructs the longitudinal profile. It lets a wider downstream
	// stem lower its bed enough that bed + water depth cannot jump upward at a
	// confluence. Generator keeps this value identical to WorldSample config.
	RiverCenterDepthNormalized float64 `json:",omitempty"`

	// MinLakeWaterDepthNormalized is persisted because shallow lake-edge
	// ownership affects whether a near river/lake connection is visibly wet.
	// Old v1.1 models omit it and therefore retain the legacy 0.003 threshold.
	MinLakeWaterDepthNormalized float64 `json:",omitempty"`

	// LakeGeometryVersion versions the contract between persisted lake-bed
	// ceilings and Runtime's minimum visible water depth. Version 1 (v1.1.7+)
	// guarantees that every planned lake cell can expose a real water column;
	// omitted/zero preserves historical shallow lake candidates exactly.
	LakeGeometryVersion uint8 `json:",omitempty"`
}

type RiverSegment struct {
	From int
	To   int

	X0 float64
	Z0 float64
	X1 float64
	Z1 float64

	Bed0          float64
	Bed1          float64
	HalfWidth     float64
	MeanderOffset float64

	// Profile is a carve-only, monotonically non-increasing centerline bed from
	// From to To. It is sampled from the actual broad terrain, not only endpoints.
	Profile []float64

	// ReceiverSurface is a lake/ocean backwater floor. Direct receiver
	// segments seed the datum, then Waterbed propagates it upstream only while
	// the natural river surface remains below the receiving water level. Runtime
	// therefore gets a flat backwater reach instead of an upstream-facing ramp.
	ReceiverSurfaceDefined bool
	ReceiverSurface        float64

	// Receiving reaches may expand near their outlet. Version 0 preserves
	// the v1.1 behavior for already-created worlds; version 2 enables the
	// v1.1.1 large-river/lake-mouth rules.
	OceanMouth               bool
	LakeMouth                bool
	LakeOutlet               bool
	LakeOutletSurfaceDefined bool    `json:",omitempty"`
	LakeOutletSurface        float64 `json:",omitempty"`
	MouthExpansionVersion    uint8   `json:",omitempty"`
}

type LakeNode struct {
	Index  int
	LakeID int

	X float64
	Z float64

	BedCeiling   float64
	InnerRadius  float64
	OuterRadius  float64
	WaterSurface float64

	// CoreCarve is set only for v1.1.7+ lake groups whose macro depression
	// would otherwise be completely erased by detailed terrain. At most one
	// low-excess node per LakeID receives it; old persisted worlds omit it.
	CoreCarve bool `json:",omitempty"`
}

type Model struct {
	Config   Config
	Geometry world.GridGeometry
	Base     *terrain.Model

	Rivers []RiverSegment
	Lakes  []LakeNode

	// Buckets are indexed by nearest macro node. Segments/nodes are inserted
	// into every nearby bucket whose query region can intersect their support.
	riverBuckets [][]int
	lakeBuckets  [][]int
}

type Sample struct {
	BaseHeight  float64
	FinalHeight float64

	RiverCarve                  float64
	RiverWeight                 float64
	RiverHalfWidth              float64
	RiverBedDefined             bool
	RiverBedElevation           float64
	RiverReceiverSurfaceDefined bool
	RiverReceiverSurface        float64
	RiverReceiverWeight         float64
	LakeCarve                   float64

	WaterKind             reconcile.WaterKind
	WaterSurfaceDefined   bool
	WaterSurfaceElevation float64
}

type Diagnostics struct {
	RiverSampleSteps  int
	RiverUpslopeSteps int
	MaxRiverRise      float64

	MaxRiseSegment          int
	MaxRiseFrom             int
	MaxRiseTo               int
	MaxRiseT                float64
	MaxRiseOwnProfileRise   float64
	MaxRisePreviousOverlaps int
	MaxRiseCurrentOverlaps  int

	LakeBedViolations int
	MaxLakeBedExcess  float64
}
