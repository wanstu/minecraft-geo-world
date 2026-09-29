package mcprofile

type BiomeFamily uint8

const (
	BiomeUnknown BiomeFamily = iota
	BiomeWarmOcean
	BiomeTemperateOcean
	BiomeColdOcean
	BiomeFrozenOcean
	BiomeIceSheet
	BiomeTundra
	BiomeAlpine
	BiomeBorealForest
	BiomeTemperateForest
	BiomeTemperateGrassland
	BiomeTemperateDesert
	BiomeMediterraneanShrubland
	BiomeSubtropicalForest
	BiomeHotDesert
	BiomeSavanna
	BiomeTropicalMonsoonForest
	BiomeTropicalRainforest
	BiomeWetland
)

func (b BiomeFamily) String() string {
	switch b {
	case BiomeWarmOcean:
		return "Warm Ocean"
	case BiomeTemperateOcean:
		return "Temperate Ocean"
	case BiomeColdOcean:
		return "Cold Ocean"
	case BiomeFrozenOcean:
		return "Frozen Ocean"
	case BiomeIceSheet:
		return "Ice Sheet"
	case BiomeTundra:
		return "Tundra"
	case BiomeAlpine:
		return "Alpine"
	case BiomeBorealForest:
		return "Boreal Forest"
	case BiomeTemperateForest:
		return "Temperate Forest"
	case BiomeTemperateGrassland:
		return "Temperate Grassland"
	case BiomeTemperateDesert:
		return "Temperate Desert"
	case BiomeMediterraneanShrubland:
		return "Mediterranean Shrubland"
	case BiomeSubtropicalForest:
		return "Subtropical Forest"
	case BiomeHotDesert:
		return "Hot Desert"
	case BiomeSavanna:
		return "Savanna"
	case BiomeTropicalMonsoonForest:
		return "Tropical Monsoon Forest"
	case BiomeTropicalRainforest:
		return "Tropical Rainforest"
	case BiomeWetland:
		return "Wetland"
	default:
		return "Unknown"
	}
}

type TerrainVariant uint8

const (
	TerrainUnknown TerrainVariant = iota
	TerrainOcean
	TerrainPlain
	TerrainRollingPlain
	TerrainHills
	TerrainFoothills
	TerrainMountain
	TerrainPlateau
	TerrainFloodplain
	TerrainAlluvialPlain
	TerrainAlluvialFan
	TerrainCanyon
)

func (v TerrainVariant) String() string {
	switch v {
	case TerrainOcean:
		return "Ocean"
	case TerrainPlain:
		return "Plain"
	case TerrainRollingPlain:
		return "Rolling Plain"
	case TerrainHills:
		return "Hills"
	case TerrainFoothills:
		return "Foothills"
	case TerrainMountain:
		return "Mountain"
	case TerrainPlateau:
		return "Plateau"
	case TerrainFloodplain:
		return "Floodplain"
	case TerrainAlluvialPlain:
		return "Alluvial Plain"
	case TerrainAlluvialFan:
		return "Alluvial Fan"
	case TerrainCanyon:
		return "Canyon"
	default:
		return "Unknown"
	}
}

type SurfaceProfile uint8

const (
	SurfaceUnknown SurfaceProfile = iota
	SurfaceOceanSand
	SurfaceOceanGravel
	SurfaceFrozenOcean
	SurfaceIceSnow
	SurfaceTundraSoil
	SurfaceAlpineRock
	SurfaceAlpineSnow
	SurfaceForestLoam
	SurfacePodzol
	SurfaceGrassLoam
	SurfaceDryGrass
	SurfaceShrubSoil
	SurfaceDesertSand
	SurfaceDesertGravel
	SurfaceAlluvialSilt
	SurfaceWetMud
	SurfacePeat
)

func (s SurfaceProfile) String() string {
	switch s {
	case SurfaceOceanSand:
		return "Ocean Sand"
	case SurfaceOceanGravel:
		return "Ocean Gravel"
	case SurfaceFrozenOcean:
		return "Frozen Ocean"
	case SurfaceIceSnow:
		return "Ice / Snow"
	case SurfaceTundraSoil:
		return "Tundra Soil"
	case SurfaceAlpineRock:
		return "Alpine Rock"
	case SurfaceAlpineSnow:
		return "Alpine Snow"
	case SurfaceForestLoam:
		return "Forest Loam"
	case SurfacePodzol:
		return "Podzol"
	case SurfaceGrassLoam:
		return "Grass Loam"
	case SurfaceDryGrass:
		return "Dry Grass Soil"
	case SurfaceShrubSoil:
		return "Shrub Soil"
	case SurfaceDesertSand:
		return "Desert Sand"
	case SurfaceDesertGravel:
		return "Desert Gravel"
	case SurfaceAlluvialSilt:
		return "Alluvial Silt"
	case SurfaceWetMud:
		return "Wet Mud"
	case SurfacePeat:
		return "Peat"
	default:
		return "Unknown"
	}
}

type VegetationProfile uint8

const (
	VegetationUnknown VegetationProfile = iota
	VegetationNone
	VegetationTundraSparse
	VegetationAlpineSparse
	VegetationConiferForest
	VegetationMixedForest
	VegetationBroadleafForest
	VegetationDenseRainforest
	VegetationMonsoonForest
	VegetationGrassland
	VegetationDryGrassland
	VegetationShrubland
	VegetationSavanna
	VegetationDesertSparse
	VegetationWetland
)

func (v VegetationProfile) String() string {
	switch v {
	case VegetationNone:
		return "None"
	case VegetationTundraSparse:
		return "Tundra Sparse"
	case VegetationAlpineSparse:
		return "Alpine Sparse"
	case VegetationConiferForest:
		return "Conifer Forest"
	case VegetationMixedForest:
		return "Mixed Forest"
	case VegetationBroadleafForest:
		return "Broadleaf Forest"
	case VegetationDenseRainforest:
		return "Dense Rainforest"
	case VegetationMonsoonForest:
		return "Monsoon Forest"
	case VegetationGrassland:
		return "Grassland"
	case VegetationDryGrassland:
		return "Dry Grassland"
	case VegetationShrubland:
		return "Shrubland"
	case VegetationSavanna:
		return "Savanna"
	case VegetationDesertSparse:
		return "Desert Sparse"
	case VegetationWetland:
		return "Wetland"
	default:
		return "Unknown"
	}
}

type Input struct {
	Width  int
	Height int

	LandMask []bool

	EcoRegion        []uint8
	RegionConfidence []float64

	TreeCoverPotential  []float64
	GrassCoverPotential []float64
	ShrubCoverPotential []float64
	BareGroundPotential []float64
	SnowCoverPotential  []float64
	SoilMoisture        []float64
	WetlandEcology      []float64
	AlpineEcology       []float64

	RollingPlainWeight []float64
	HillWeight         []float64
	FoothillWeight     []float64
	MountainWeight     []float64
	PlateauWeight      []float64

	FloodplainWeight    []float64
	AlluvialPlainWeight []float64
	AlluvialFanWeight   []float64
	CanyonWeight        []float64
}

type Model struct {
	Width  int
	Height int

	LandMask []bool

	BiomeFamily       []BiomeFamily
	TerrainVariant    []TerrainVariant
	SurfaceProfile    []SurfaceProfile
	VegetationProfile []VegetationProfile

	ProfileConfidence []float64
}
