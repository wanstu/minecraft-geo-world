package hydrology

import "geoworld/internal/world"

type Config struct {
	SeaLevelNormalized float64
	// FillEpsilon is the legacy normalized rise per 128-block D8 hop.
	FillEpsilon            float64
	RunoffLossMM           float64
	BaseRunoffCoefficient  float64
	SeasonalityRunoffBoost float64
	RiverQuantile          float64
	// Accumulation scales remain in reference-cell-equivalent runoff units.
	MinRiverAccumulation     float64
	LakeDepthScale           float64
	WetlandAccumulationScale float64
	LakePotentialThreshold   float64
	MinLakeAreaBlocks2       float64
}

type Input struct {
	Width               int
	Height              int
	Geometry            world.GridGeometry
	Elevation           []float64
	LandMask            []bool
	AnnualPrecipMM      []float64
	RainfallSeasonality []float64
	WarmSeasonPrecipMM  []float64
	ColdSeasonPrecipMM  []float64
}

type Lake struct {
	ID                 int
	CellCount          int
	AreaBlocks2        float64
	SurfaceElevation   float64
	MaxDepth           float64
	SpillOutlet        int
	BasinID            int
	InflowAccumulation float64
}

type Model struct {
	Config   Config
	Geometry world.GridGeometry
	Width    int
	Height   int

	LandMask          []bool
	OriginalElevation []float64
	FilledElevation   []float64
	FillDepth         []float64

	FlowTo []int
	// LocalRunoffMM keeps its historical name, but after 6.1b values are
	// reference-cell-equivalent runoff contributions so accumulation is not
	// multiplied merely by increasing macro-grid resolution.
	LocalRunoffMM        []float64
	FlowAccumulation     []float64
	WarmFlowAccumulation []float64
	ColdFlowAccumulation []float64

	BasinID                 []int
	RiverStrength           []float64
	RiverOrder              []uint8
	PerennialRiverPotential []float64
	SeasonalRiverPotential  []float64
	LakePotential           []float64
	WetlandPotential        []float64
	LakeID                  []int
	Lakes                   []Lake

	RiverThreshold  float64
	BasinCount      int
	MajorBasinCount int
}
