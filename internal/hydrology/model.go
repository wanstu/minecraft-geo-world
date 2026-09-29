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
}

type Input struct {
	Width               int
	Height              int
	Elevation           []float64
	LandMask            []bool
	AnnualPrecipMM      []float64
	RainfallSeasonality []float64
}

type Model struct {
	Config Config
	Width  int
	Height int

	LandMask          []bool
	OriginalElevation []float64
	FilledElevation   []float64
	FillDepth         []float64

	FlowTo           []int
	LocalRunoffMM    []float64
	FlowAccumulation []float64

	BasinID          []int
	RiverStrength    []float64
	LakePotential    []float64
	WetlandPotential []float64

	RiverThreshold  float64
	BasinCount      int
	MajorBasinCount int
}
