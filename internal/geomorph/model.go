package geomorph

import "geoworld/internal/world"

type Config struct {
	SeaLevelNormalized float64
	AccumulationScale  float64
	SlopeScale         float64
	ReliefScale        float64

	// LongitudinalGradeVersion enables causal lowland river-profile grading.
	// Zero preserves all locked worlds exactly. Version 1 smooths only mature,
	// alluvial river corridors using the already-resolved FlowTo network.
	LongitudinalGradeVersion uint8 `json:",omitempty"`
}

type Input struct {
	Width    int
	Height   int
	Geometry world.GridGeometry

	LandMask        []bool
	Elevation       []float64
	Slope           []float64
	ReliefIndex     []float64
	ValleyIncision  []float64
	TotalErosion    []float64
	FluvialPower    []float64
	TectonicSupport []float64

	FlowTo                  []int
	FlowAccumulation        []float64
	RiverStrength           []float64
	PerennialRiverPotential []float64
	SeasonalRiverPotential  []float64
	WetlandPotential        []float64
	LakeID                  []int
	RiverThreshold          float64

	RollingPlainWeight []float64
	HillWeight         []float64
	FoothillWeight     []float64
	MountainWeight     []float64
	PlateauWeight      []float64
	ValleyWeight       []float64
}

type Model struct {
	Config   Config
	Geometry world.GridGeometry
	Width    int
	Height   int

	LandMask []bool

	UpstreamSlope               []float64
	UpstreamSteepMemory         []float64
	UpstreamMountainMemory      []float64
	SlopeDrop                   []float64
	SedimentSupply              []float64
	SedimentProduction          []float64
	SedimentFlux                []float64
	TransportCapacity           []float64
	SedimentAccommodation       []float64
	SedimentDeposition          []float64
	SedimentExport              []float64
	FloodplainWeight            []float64
	AlluvialPlainWeight         []float64
	AlluvialFanCore             []float64
	AlluvialFanWeight           []float64
	CanyonWeight                []float64
	ValleyFloorWidth            []float64
	DepositionPotential         []float64
	LongitudinalGradeCorrection []float64
}
