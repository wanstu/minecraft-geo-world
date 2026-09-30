package reconcile

import "geoworld/internal/world"

type WaterKind uint8

const (
	WaterNone WaterKind = iota
	WaterOcean
	WaterLake
	WaterRiver
)

// Config controls only reconciliation tolerances and target constraints. It does
// not alter the upstream climate/hydrology simulation.
type Config struct {
	CoastMarginNormalized      float64
	RiverStrengthThreshold     float64
	OriginalDownhillEpsilon    float64
	FinalUpslopeEpsilon        float64
	MinRiverBedDropNormalized  float64
	LakeBedClearanceNormalized float64
}

type RiverEdgeConstraint struct {
	From int
	To   int

	RiverStrength  float64
	OriginalDrop   float64
	RawTerrainRise float64
	TargetBedDrop  float64
}

// Model translates current cross-stage inconsistencies into explicit targets.
// CoastTargetElevation is a macro control-point target for the broad terrain
// surface. RiverBedTarget is a separate channel-bed target and must not be
// confused with the broad terrain control point.
type Model struct {
	Config   Config
	Geometry world.GridGeometry

	SeaLevelNormalized float64

	RawTerrainElevation  []float64
	CoastTargetElevation []float64
	CoastCorrection      []float64
	RawTerrainSurface    *world.Surface
	ConstrainedSurface   *world.Surface

	AppliedCoastElevation       []float64
	AppliedCoastSurface         *world.Surface
	AppliedCoastSurfaceMismatch int
	AppliedCoastMaxNodeError    float64

	SurfaceMismatchBefore int
	SurfaceMismatchAfter  int
	CoastAdjustedNodes    int
	CoastRaisedNodes      int
	CoastCarvedNodes      int
	MaxCoastRaise         float64
	MaxCoastCarve         float64

	WaterKind             []WaterKind
	WaterSurfaceDefined   []bool
	WaterSurfaceElevation []float64

	LakeBedCeilingDefined []bool
	LakeBedCeiling        []float64
	RawLakeBedViolations  int
	LakeBedViolations     int

	RiverBaseElevation      []float64
	RiverBedActive          []bool
	RiverBedTarget          []float64
	RiverEdges              []RiverEdgeConstraint
	QualifyingRiverEdges    int
	OriginallyDownhillEdges int
	RawUpslopeRiverEdges    int
	TargetUpslopeRiverEdges int
	RiverBedCarvedNodes     int
	MaxRiverBedCarve        float64
}
