package terrain

import (
	"geoworld/internal/sampling"
	"geoworld/internal/world"
)

type Config struct {
	Seed int64

	// Legacy compatibility only. Formal model adapters provide Input.Geometry.
	// When Geometry is present, non-zero legacy values must agree with it.
	CellSizeBlocks float64
	OriginX        float64
	OriginZ        float64

	PlainAmplitude    float64
	HillAmplitude     float64
	FoothillAmplitude float64
	MountainAmplitude float64
	// MountainStrengthScale is zero for locked pre-v1.1.7 worlds. Current
	// rules persist the semantic mountain-strength multiplier here so it can
	// raise/lower the whole mountain envelope while MountainAmplitude continues
	// to control internal relief. Zero deliberately preserves legacy sampling.
	MountainStrengthScale float64 `json:",omitempty"`
	PlateauAmplitude      float64
	ValleyDepth           float64

	DepositionLift   float64
	AlluvialFanLift  float64
	CanyonExtraDepth float64
	CliffFrequency   float64
	// RareMacroStrength is zero for locked pre-v1.1.7 worlds. Current
	// profiles map RareMacroTerrain here so rare landmarks can scale without
	// changing the ordinary terrain language.
	RareMacroStrength float64
}

type Input struct {
	Width    int
	Height   int
	Geometry world.GridGeometry

	BaseElevation []float64

	RollingPlainWeight []float64
	HillWeight         []float64
	FoothillWeight     []float64
	MountainWeight     []float64
	PlateauWeight      []float64
	ValleyWeight       []float64

	FloodplainWeight    []float64
	AlluvialPlainWeight []float64
	AlluvialFanWeight   []float64
	CanyonWeight        []float64
	ValleyFloorWidth    []float64
	DepositionPotential []float64
}

type Model struct {
	Config   Config
	Geometry world.GridGeometry

	BaseElevation sampling.ScalarGrid

	RollingPlainWeight sampling.ScalarGrid
	HillWeight         sampling.ScalarGrid
	FoothillWeight     sampling.ScalarGrid
	MountainWeight     sampling.ScalarGrid
	PlateauWeight      sampling.ScalarGrid
	ValleyWeight       sampling.ScalarGrid

	FloodplainWeight    sampling.ScalarGrid
	AlluvialPlainWeight sampling.ScalarGrid
	AlluvialFanWeight   sampling.ScalarGrid
	CanyonWeight        sampling.ScalarGrid
	ValleyFloorWidth    sampling.ScalarGrid
	DepositionPotential sampling.ScalarGrid

	// HeightCorrection is an optional continuous broad-surface overlay. It is
	// intentionally separate from river/lake bed geometry.
	HeightCorrection *sampling.ScalarGrid
}

type Sample struct {
	BaseElevation float64

	RollingPlainWeight float64
	HillWeight         float64
	FoothillWeight     float64
	MountainWeight     float64
	PlateauWeight      float64
	ValleyWeight       float64

	FloodplainWeight    float64
	AlluvialPlainWeight float64
	AlluvialFanWeight   float64
	CanyonWeight        float64
	ValleyFloorWidth    float64
	DepositionPotential float64

	RegionalDetail   float64
	DepositionLift   float64
	ValleyCarve      float64
	CanyonCarve      float64
	RareMacroLift    float64
	RareMacroCarve   float64
	RareEscarpment   float64
	HeightCorrection float64
	FinalHeight      float64
}
