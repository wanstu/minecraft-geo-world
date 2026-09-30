package landform

import "geoworld/internal/world"

type Config struct {
	SeaLevelNormalized      float64
	MountainInfluenceBlocks float64
	ValleyRadiusBlocks      float64
	ValleyDecayBlocks       float64
}

type Input struct {
	Width    int
	Height   int
	Geometry world.GridGeometry

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
	Config   Config
	Geometry world.GridGeometry
	Width    int
	Height   int

	LandMask []bool

	MountainCore []float64
	// MountainDistance is a physical world distance in Minecraft blocks.
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
