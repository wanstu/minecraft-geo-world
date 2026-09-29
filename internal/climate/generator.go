package climate

import (
	"errors"
	"math"
)

func Generate(input Input, cfg Config) (*Model, error) {
	if input.Width < 2 || input.Height < 2 {
		return nil, errors.New("climate: input grid must be at least 2x2")
	}
	n := input.Width * input.Height
	if len(input.Elevation) != n || len(input.LandMask) != n {
		return nil, errors.New("climate: input field sizes do not match dimensions")
	}

	cfg = normalizeConfig(cfg)
	m := &Model{
		Config:            cfg,
		Width:             input.Width,
		Height:            input.Height,
		LatitudeDeg:       make([]float64, n),
		LatitudeBaseTempC: make([]float64, n),
		LandElevationM:    make([]float64, n),
		AnnualMeanTempC:   make([]float64, n),
		PressureIndex:     make([]float64, n),
		WindX:             make([]float64, n),
		WindZ:             make([]float64, n),
	}

	maxLatRad := cfg.MaxLatitudeDeg * math.Pi / 180
	maxLatSin := math.Sin(maxLatRad)

	for y := 0; y < input.Height; y++ {
		t := float64(y) / float64(input.Height-1)
		lat := cfg.MaxLatitudeDeg * (1 - 2*t)
		latRad := math.Abs(lat) * math.Pi / 180

		latFactor := math.Sin(latRad) / maxLatSin
		latFactor = clamp(latFactor, 0, 1)
		latFactor = math.Pow(latFactor, 1.22)

		baseTemp := cfg.EquatorTempC +
			(cfg.PolarTempC-cfg.EquatorTempC)*latFactor

		for x := 0; x < input.Width; x++ {
			i := y*input.Width + x
			m.LatitudeDeg[i] = lat
			m.LatitudeBaseTempC[i] = baseTemp

			elevationM := 0.0
			temp := baseTemp
			if input.LandMask[i] {
				aboveSea := math.Max(0, input.Elevation[i]-cfg.SeaLevelNormalized)
				span := math.Max(1e-9, 1-cfg.SeaLevelNormalized)
				elevationM = clamp(aboveSea/span, 0, 1) * cfg.MaxLandElevationM
				temp -= cfg.LapseRateCPerKm * elevationM / 1000
			}

			m.LandElevationM[i] = elevationM
			m.AnnualMeanTempC[i] = temp

			pressure, wx, wz := circulationAtLatitude(lat)
			m.PressureIndex[i] = pressure
			m.WindX[i] = wx
			m.WindZ[i] = wz
		}
	}

	return m, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.MaxLatitudeDeg <= 0 || cfg.MaxLatitudeDeg > 89 {
		cfg.MaxLatitudeDeg = 82
	}
	if cfg.EquatorTempC == 0 {
		cfg.EquatorTempC = 28
	}
	if cfg.PolarTempC == 0 {
		cfg.PolarTempC = -18
	}
	if cfg.LapseRateCPerKm <= 0 {
		cfg.LapseRateCPerKm = 6.5
	}
	if cfg.MaxLandElevationM <= 0 {
		cfg.MaxLandElevationM = 6000
	}
	if cfg.SeaLevelNormalized <= 0 || cfg.SeaLevelNormalized >= 1 {
		cfg.SeaLevelNormalized = 0.5
	}
	return cfg
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func circulationAtLatitude(latitudeDeg float64) (pressure, windX, windZ float64) {
	absLat := math.Abs(latitudeDeg)

	// Idealized pressure belts:
	// 0° equatorial low, ~30° subtropical high, ~60° subpolar low,
	// polar edge high. Smooth Gaussian bands avoid hard boundaries.
	pressure =
		0.52 -
			0.30*gaussian(absLat, 0, 11) +
			0.28*gaussian(absLat, 30, 10) -
			0.22*gaussian(absLat, 60, 9) +
			0.20*gaussian(absLat, 82, 10)
	pressure = clamp(pressure, 0, 1)

	sign := 1.0
	if latitudeDeg < 0 {
		sign = -1
	}

	// Three idealized circulation vectors. Blend them over broad latitude
	// windows so 30°/60° do not become artificial discontinuity lines.
	tradeX, tradeZ := -0.85, 0.38*sign
	westerlyX, westerlyZ := 0.92, -0.26*sign
	polarX, polarZ := -0.70, 0.22*sign

	toWesterly := smoothstep(25, 35, absLat)
	windX = lerp(tradeX, westerlyX, toWesterly)
	windZ = lerp(tradeZ, westerlyZ, toWesterly)

	toPolar := smoothstep(55, 65, absLat)
	windX = lerp(windX, polarX, toPolar)
	windZ = lerp(windZ, polarZ, toPolar)

	// Fade meridional component close to the equator to avoid a direction
	// discontinuity exactly at 0°.
	equatorFade := smoothstep(0, 8, absLat)
	windZ *= equatorFade

	return pressure, windX, windZ
}

func gaussian(x, center, sigma float64) float64 {
	d := (x - center) / sigma
	return math.Exp(-0.5 * d * d)
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge1 <= edge0 {
		return 0
	}
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}
