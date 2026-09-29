package hydrology

type Config struct {
	SeaLevelNormalized       float64
	FillEpsilon              float64
	RunoffLossMM             float64
	BaseRunoffCoefficient    float64
	SeasonalityRunoffBoost   float64
	RiverQuantile            float64
	MinRiverAccumulation     float64
	LakeDepthScale           float64
	WetlandAccumulationScale float64
	LakePotentialThreshold   float64
	MinLakeCells             int
}

type Input struct {
	Width               int
	Height              int
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
	SurfaceElevation   float64
	MaxDepth           float64
	SpillOutlet        int
	BasinID            int
	InflowAccumulation float64
}

type Model struct {
	Config Config
	Width  int
	Height int

	LandMask          []bool
	OriginalElevation []float64
	FilledElevation   []float64
	FillDepth         []float64

	FlowTo               []int
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
