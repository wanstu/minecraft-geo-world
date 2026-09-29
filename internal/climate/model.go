package climate

type Config struct {
	MaxLatitudeDeg            float64
	EquatorTempC              float64
	PolarTempC                float64
	LapseRateCPerKm           float64
	MaxLandElevationM         float64
	SeaLevelNormalized        float64
	OceanCurrentDeflectionDeg float64
	OceanCurrentWindScale     float64
	SSTTransportIterations    int
	SSTAdvection              float64
	SSTDiffusion              float64
	CoastalCurrentInfluence   float64
	CoastalTempDecayCells     float64
}

type Input struct {
	Width     int
	Height    int
	Elevation []float64
	LandMask  []bool
}

type Model struct {
	Config               Config
	Width                int
	Height               int
	LandMask             []bool
	LatitudeDeg          []float64
	LatitudeBaseTempC    []float64
	LandElevationM       []float64
	AnnualMeanTempC      []float64
	PressureIndex        []float64
	WindX                []float64
	WindZ                []float64
	SeaSurfaceTempC      []float64
	CurrentX             []float64
	CurrentZ             []float64
	OceanDistanceCells   []float64
	OceanTempCorrectionC []float64
	AdjustedAnnualTempC  []float64
}
