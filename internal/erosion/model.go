package erosion

type Config struct {
	SeaLevelNormalized      float64
	ReferencePrecipMM       float64
	SlopeScale              float64
	MaxValleyIncision       float64
	MaxHillslopeErosion     float64
	TectonicSupportStrength float64
	ReliefRadiusCells       int
	ReliefScale             float64
}

type Input struct {
	Width  int
	Height int

	LandMask          []bool
	Elevation         []float64
	CrustThickness    []float64
	TectonicIntensity []float64
	TectonicInfluence []float64
	AnnualPrecipMM    []float64

	FlowAccumulation []float64
	RiverStrength    []float64
	LakeID           []int
	RiverThreshold   float64
}

type Model struct {
	Config Config
	Width  int
	Height int

	LandMask []bool

	OriginalElevation []float64
	Slope             []float64
	Wetness           []float64
	TectonicSupport   []float64
	TerrainMaturity   []float64
	ErosionResistance []float64

	FluvialPower     []float64
	ValleyIncision   []float64
	HillslopeErosion []float64
	TotalErosion     []float64
	ErodedElevation  []float64

	LocalRelief []float64
	ReliefIndex []float64
}
