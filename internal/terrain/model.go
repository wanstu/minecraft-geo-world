package terrain

import "geoworld/internal/sampling"

type Config struct {
	Seed int64

	CellSizeBlocks float64
	OriginX        float64
	OriginZ        float64

	PlainAmplitude    float64
	HillAmplitude     float64
	FoothillAmplitude float64
	MountainAmplitude float64
	PlateauAmplitude  float64
	ValleyDepth       float64
}

type Input struct {
	Width  int
	Height int

	BaseElevation []float64

	RollingPlainWeight []float64
	HillWeight         []float64
	FoothillWeight     []float64
	MountainWeight     []float64
	PlateauWeight      []float64
	ValleyWeight       []float64
}

type Model struct {
	Config Config

	BaseElevation sampling.ScalarGrid

	RollingPlainWeight sampling.ScalarGrid
	HillWeight         sampling.ScalarGrid
	FoothillWeight     sampling.ScalarGrid
	MountainWeight     sampling.ScalarGrid
	PlateauWeight      sampling.ScalarGrid
	ValleyWeight       sampling.ScalarGrid
}

type Sample struct {
	BaseElevation float64

	RollingPlainWeight float64
	HillWeight         float64
	FoothillWeight     float64
	MountainWeight     float64
	PlateauWeight      float64
	ValleyWeight       float64

	RegionalDetail float64
	ValleyCarve    float64
	FinalHeight    float64
}
