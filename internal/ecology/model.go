package ecology

type Region uint8

const (
	RegionUnknown Region = iota
	RegionWarmOcean
	RegionTemperateOcean
	RegionColdOcean
	RegionFrozenOcean
	RegionIceSheet
	RegionTundra
	RegionAlpine
	RegionBorealForest
	RegionTemperateForest
	RegionTemperateGrassland
	RegionTemperateDesert
	RegionMediterraneanShrubland
	RegionSubtropicalForest
	RegionHotDesert
	RegionSavanna
	RegionTropicalMonsoonForest
	RegionTropicalRainforest
	RegionWetland
)

func (r Region) String() string {
	switch r {
	case RegionWarmOcean:
		return "Warm Ocean"
	case RegionTemperateOcean:
		return "Temperate Ocean"
	case RegionColdOcean:
		return "Cold Ocean"
	case RegionFrozenOcean:
		return "Frozen Ocean"
	case RegionIceSheet:
		return "Ice Sheet"
	case RegionTundra:
		return "Tundra"
	case RegionAlpine:
		return "Alpine"
	case RegionBorealForest:
		return "Boreal Forest"
	case RegionTemperateForest:
		return "Temperate Forest"
	case RegionTemperateGrassland:
		return "Temperate Grassland"
	case RegionTemperateDesert:
		return "Temperate Desert"
	case RegionMediterraneanShrubland:
		return "Mediterranean Shrubland"
	case RegionSubtropicalForest:
		return "Subtropical Forest"
	case RegionHotDesert:
		return "Hot Desert"
	case RegionSavanna:
		return "Savanna"
	case RegionTropicalMonsoonForest:
		return "Tropical Monsoon Forest"
	case RegionTropicalRainforest:
		return "Tropical Rainforest"
	case RegionWetland:
		return "Wetland"
	default:
		return "Unknown"
	}
}

type Config struct {
	ReferencePrecipMM float64
}

type Input struct {
	Width  int
	Height int

	LandMask []bool

	AnnualTempC         []float64
	SummerTempC         []float64
	WinterTempC         []float64
	AnnualPrecipMM      []float64
	WarmSeasonPrecipMM  []float64
	ColdSeasonPrecipMM  []float64
	RainfallSeasonality []float64
	MonsoonIndex        []float64
	SeaSurfaceTempC     []float64

	Elevation          []float64
	ReliefIndex        []float64
	RollingPlainWeight []float64
	HillWeight         []float64
	FoothillWeight     []float64
	MountainWeight     []float64
	PlateauWeight      []float64

	WetlandPotential        []float64
	PerennialRiverPotential []float64
	SeasonalRiverPotential  []float64
	FloodplainWeight        []float64
	AlluvialPlainWeight     []float64
	ValleyFloorWidth        []float64
}

type Model struct {
	Config Config
	Width  int
	Height int

	LandMask []bool

	MoistureIndex       []float64
	AridityIndex        []float64
	WarmSeasonRainShare []float64
	ColdSeasonRainShare []float64
	ThermalSeasonality  []float64
	WetlandEcology      []float64
	AlpineEcology       []float64

	TreeCoverPotential  []float64
	GrassCoverPotential []float64
	ShrubCoverPotential []float64
	BareGroundPotential []float64
	SnowCoverPotential  []float64
	SoilMoisture        []float64
	VegetationDensity   []float64

	EcoRegion        []Region
	RegionConfidence []float64
}
