package climate

import "geoworld/internal/world"

type Config struct {
	MaxLatitudeDeg                  float64
	EquatorTempC                    float64
	PolarTempC                      float64
	LatitudeTempCurvePower          float64
	LapseRateCPerKm                 float64
	MaxLandElevationM               float64
	ElevationCurvePower             float64
	SeaLevelNormalized              float64
	OceanCurrentDeflectionDeg       float64
	OceanCurrentWindScale           float64
	SSTTransportIterations          int
	SSTAdvection                    float64
	SSTDiffusion                    float64
	CoastalCurrentInfluence         float64
	CoastalTempDecayBlocks          float64
	MoistureIterations              int
	MoistureAdvection               float64
	MoistureDiffusion               float64
	OrographicRainStrength          float64
	PrecipitationScaleMM            float64
	LandSeasonalityBaseC            float64
	LandSeasonalityLatitudeC        float64
	OceanSeasonalityBaseC           float64
	OceanSeasonalityLatitudeC       float64
	MonsoonWindStrength             float64
	PrevailingWindStrength          float64
	SeasonalMoistureTempCoupling    float64
	SeasonalSeaSurfaceTempCoupling  float64
	SeasonalBackgroundMoistureBlend float64
}

type Input struct {
	Width     int
	Height    int
	Geometry  world.GridGeometry
	Elevation []float64
	LandMask  []bool
}

type Model struct {
	Config                 Config
	Geometry               world.GridGeometry
	Width                  int
	Height                 int
	LandMask               []bool
	LatitudeDeg            []float64
	LatitudeBaseTempC      []float64
	LandElevationM         []float64
	AnnualMeanTempC        []float64
	PressureIndex          []float64
	WindX                  []float64
	WindZ                  []float64
	SeaSurfaceTempC        []float64
	CurrentX               []float64
	CurrentZ               []float64
	OceanDistanceBlocks    []float64
	OceanTempCorrectionC   []float64
	AdjustedAnnualTempC    []float64
	Moisture               []float64
	OrographicLift         []float64
	AnnualPrecipMM         []float64
	SeasonalAmplitudeC     []float64
	SummerTempC            []float64
	WinterTempC            []float64
	MonsoonIndex           []float64
	SummerWindX            []float64
	SummerWindZ            []float64
	WinterWindX            []float64
	WinterWindZ            []float64
	WarmSeasonPrecipMM     []float64
	ColdSeasonPrecipMM     []float64
	SeasonalAnnualPrecipMM []float64
	RainfallSeasonality    []float64
	MonsoonRainRatio       []float64
}
