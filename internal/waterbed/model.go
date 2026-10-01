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
}

type RiverSegment struct {
	From int
	To   int

	X0 float64
	Z0 float64
	X1 float64
	Z1 float64

	Bed0      float64
	Bed1      float64
	HalfWidth float64

	// Profile is a carve-only, monotonically non-increasing centerline bed from
	// From to To. It is sampled from the actual broad terrain, not only endpoints.
	Profile []float64

	// ReceiverSurface is the downstream lake/ocean datum reachable from this
	// segment. It is propagated upstream through the river DAG so Runtime water
	// can form a flat backwater where the natural shallow river surface would
	// otherwise sit below the receiving waterbody.
	ReceiverSurfaceDefined bool
	ReceiverSurface        float64
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
	RiverReceiverSurfaceDefined bool
	RiverReceiverSurface        float64
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
