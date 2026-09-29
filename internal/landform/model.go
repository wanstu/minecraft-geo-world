package landform

type Config struct {
	SeaLevelNormalized     float64
	MountainInfluenceCells float64
	ValleyRadiusCells      int
	ValleyDecayCells       float64
}

type Input struct {
	Width  int
	Height int

	LandMask        []bool
	Elevation       []float64
	ReliefIndex     []float64
	Slope           []float64
	TerrainMaturity []float64
	TectonicSupport []float64
	CrustThickness  []float64
	ValleyIncision  []float64
	FluvialPower    []float64
}

type Model struct {
	Config Config
	Width  int
	Height int

	LandMask []bool

	MountainCore      []float64
	MountainDistance  []float64
	MountainInfluence []float64

	RollingPlainWeight []float64
	HillWeight         []float64
	FoothillWeight     []float64
	MountainWeight     []float64
	PlateauWeight      []float64
	ValleyCore         []float64
	ValleyWeight       []float64
}
