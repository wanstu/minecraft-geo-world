package climate

type Config struct {
	MaxLatitudeDeg     float64
	EquatorTempC       float64
	PolarTempC         float64
	LapseRateCPerKm    float64
	MaxLandElevationM  float64
	SeaLevelNormalized float64
}

type Input struct {
	Width     int
	Height    int
	Elevation []float64
	LandMask  []bool
}

type Model struct {
	Config            Config
	Width             int
	Height            int
	LatitudeDeg       []float64
	LatitudeBaseTempC []float64
	LandElevationM    []float64
	AnnualMeanTempC   []float64
	PressureIndex     []float64
	WindX             []float64
	WindZ             []float64
}
