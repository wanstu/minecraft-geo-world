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
	if cfg.MoistureIterations <= 0 {
		maxDim := input.Width
		if input.Height > maxDim {
			maxDim = input.Height
		}
		cfg.MoistureIterations = maxDim / 3
		if cfg.MoistureIterations < 48 {
			cfg.MoistureIterations = 48
		}
		if cfg.MoistureIterations > 192 {
			cfg.MoistureIterations = 192
		}
	}
	m := &Model{
		Config:                 cfg,
		Width:                  input.Width,
		Height:                 input.Height,
		LandMask:               append([]bool(nil), input.LandMask...),
		LatitudeDeg:            make([]float64, n),
		LatitudeBaseTempC:      make([]float64, n),
		LandElevationM:         make([]float64, n),
		AnnualMeanTempC:        make([]float64, n),
		PressureIndex:          make([]float64, n),
		WindX:                  make([]float64, n),
		WindZ:                  make([]float64, n),
		SeaSurfaceTempC:        make([]float64, n),
		CurrentX:               make([]float64, n),
		CurrentZ:               make([]float64, n),
		OceanDistanceCells:     make([]float64, n),
		OceanTempCorrectionC:   make([]float64, n),
		AdjustedAnnualTempC:    make([]float64, n),
		Moisture:               make([]float64, n),
		OrographicLift:         make([]float64, n),
		AnnualPrecipMM:         make([]float64, n),
		SeasonalAmplitudeC:     make([]float64, n),
		SummerTempC:            make([]float64, n),
		WinterTempC:            make([]float64, n),
		MonsoonIndex:           make([]float64, n),
		SummerWindX:            make([]float64, n),
		SummerWindZ:            make([]float64, n),
		WinterWindX:            make([]float64, n),
		WinterWindZ:            make([]float64, n),
		WarmSeasonPrecipMM:     make([]float64, n),
		ColdSeasonPrecipMM:     make([]float64, n),
		SeasonalAnnualPrecipMM: make([]float64, n),
		RainfallSeasonality:    make([]float64, n),
		MonsoonRainRatio:       make([]float64, n),
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

	m.initializeOceanSurface()
	m.computeSurfaceCurrents()
	m.transportSeaSurfaceHeat()
	m.applyOceanTemperatureInfluence()
	m.computeSeasonalStates()
	m.computeMoistureAndPrecipitation()
	m.computeSeasonalPrecipitation()

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
	if cfg.OceanCurrentDeflectionDeg <= 0 || cfg.OceanCurrentDeflectionDeg > 60 {
		cfg.OceanCurrentDeflectionDeg = 25
	}
	if cfg.OceanCurrentWindScale <= 0 {
		cfg.OceanCurrentWindScale = 0.62
	}
	if cfg.SSTTransportIterations <= 0 {
		cfg.SSTTransportIterations = 24
	}
	if cfg.SSTAdvection <= 0 || cfg.SSTAdvection >= 0.5 {
		cfg.SSTAdvection = 0.14
	}
	if cfg.SSTDiffusion <= 0 || cfg.SSTDiffusion >= 0.3 {
		cfg.SSTDiffusion = 0.06
	}
	if cfg.CoastalCurrentInfluence <= 0 || cfg.CoastalCurrentInfluence > 1 {
		cfg.CoastalCurrentInfluence = 0.75
	}
	if cfg.CoastalTempDecayCells <= 0 {
		cfg.CoastalTempDecayCells = 18
	}
	if cfg.MoistureAdvection <= 0 || cfg.MoistureAdvection >= 1 {
		cfg.MoistureAdvection = 0.82
	}
	if cfg.MoistureDiffusion <= 0 || cfg.MoistureDiffusion >= 0.3 {
		cfg.MoistureDiffusion = 0.08
	}
	if cfg.OrographicRainStrength <= 0 {
		cfg.OrographicRainStrength = 0.22
	}
	if cfg.PrecipitationScaleMM <= 0 {
		cfg.PrecipitationScaleMM = 40000
	}
	if cfg.LandSeasonalityBaseC <= 0 {
		cfg.LandSeasonalityBaseC = 3.5
	}
	if cfg.LandSeasonalityLatitudeC <= 0 {
		cfg.LandSeasonalityLatitudeC = 15
	}
	if cfg.OceanSeasonalityBaseC <= 0 {
		cfg.OceanSeasonalityBaseC = 1.5
	}
	if cfg.OceanSeasonalityLatitudeC <= 0 {
		cfg.OceanSeasonalityLatitudeC = 5
	}
	if cfg.MonsoonWindStrength <= 0 {
		cfg.MonsoonWindStrength = 0.75
	}
	if cfg.SeasonalMoistureTempCoupling <= 0 || cfg.SeasonalMoistureTempCoupling > 1 {
		cfg.SeasonalMoistureTempCoupling = 0.35
	}
	if cfg.SeasonalSeaSurfaceTempCoupling <= 0 || cfg.SeasonalSeaSurfaceTempCoupling > 1 {
		cfg.SeasonalSeaSurfaceTempCoupling = 0.35
	}
	if cfg.SeasonalBackgroundMoistureBlend <= 0 || cfg.SeasonalBackgroundMoistureBlend > 0.8 {
		cfg.SeasonalBackgroundMoistureBlend = 0.20
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

func (m *Model) initializeOceanSurface() {
	for i := range m.SeaSurfaceTempC {
		if m.LandMask[i] {
			m.SeaSurfaceTempC[i] = 0
			continue
		}
		// Ocean surface is slightly more thermally buffered than the idealized
		// sea-level air field. Clamp at approximate seawater freezing.
		m.SeaSurfaceTempC[i] = clamp(m.LatitudeBaseTempC[i]+1.2, -1.8, 30)
	}
}

func (m *Model) computeSurfaceCurrents() {
	for i := range m.CurrentX {
		if m.LandMask[i] {
			m.CurrentX[i] = 0
			m.CurrentZ[i] = 0
			continue
		}
		m.CurrentX[i], m.CurrentZ[i] = surfaceCurrentFromWind(
			m.WindX[i],
			m.WindZ[i],
			m.LatitudeDeg[i],
			m.Config,
		)
	}

	// Remove the component that points directly into adjacent land.
	// The remaining vector naturally tends to run along the coastline.
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			i := y*m.Width + x
			if m.LandMask[i] {
				continue
			}

			nx, nz := 0.0, 0.0
			if x+1 < m.Width && m.LandMask[i+1] {
				nx += 1
			}
			if x > 0 && m.LandMask[i-1] {
				nx -= 1
			}
			if y+1 < m.Height && m.LandMask[i+m.Width] {
				nz += 1
			}
			if y > 0 && m.LandMask[i-m.Width] {
				nz -= 1
			}

			norm := math.Hypot(nx, nz)
			if norm == 0 {
				continue
			}
			nx /= norm
			nz /= norm

			dot := m.CurrentX[i]*nx + m.CurrentZ[i]*nz
			if dot > 0 {
				m.CurrentX[i] -= dot * nx
				m.CurrentZ[i] -= dot * nz
			}
		}
	}
}

func surfaceCurrentFromWind(windX, windZ, latitudeDeg float64, cfg Config) (float64, float64) {
	if math.Abs(latitudeDeg) < 0.25 {
		return windX * cfg.OceanCurrentWindScale, windZ * cfg.OceanCurrentWindScale
	}

	sign := 1.0
	if latitudeDeg < 0 {
		sign = -1
	}
	angle := cfg.OceanCurrentDeflectionDeg * math.Pi / 180 * sign
	cosA, sinA := math.Cos(angle), math.Sin(angle)

	x := windX*cosA - windZ*sinA
	z := windX*sinA + windZ*cosA
	return x * cfg.OceanCurrentWindScale, z * cfg.OceanCurrentWindScale
}

func (m *Model) transportSeaSurfaceHeat() {
	prev := append([]float64(nil), m.SeaSurfaceTempC...)
	next := make([]float64, len(prev))

	const relaxation = 0.04
	selfWeight := 1 - m.Config.SSTAdvection - m.Config.SSTDiffusion - relaxation
	if selfWeight < 0.2 {
		selfWeight = 0.2
	}

	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for iter := 0; iter < m.Config.SSTTransportIterations; iter++ {
		copy(next, prev)

		for y := 0; y < m.Height; y++ {
			for x := 0; x < m.Width; x++ {
				i := y*m.Width + x
				if m.LandMask[i] {
					next[i] = 0
					continue
				}

				self := prev[i]
				advected := self

				mag := math.Hypot(m.CurrentX[i], m.CurrentZ[i])
				if mag > 1e-6 {
					ux := m.CurrentX[i] / mag
					uz := m.CurrentZ[i] / mag
					sx := x - int(math.Round(ux))
					sz := y - int(math.Round(uz))
					if sx >= 0 && sx < m.Width && sz >= 0 && sz < m.Height {
						si := sz*m.Width + sx
						if !m.LandMask[si] {
							advected = prev[si]
						}
					}
				}

				sum, count := 0.0, 0
				for _, d := range dirs {
					nx, nz := x+d[0], y+d[1]
					if nx < 0 || nx >= m.Width || nz < 0 || nz >= m.Height {
						continue
					}
					ni := nz*m.Width + nx
					if m.LandMask[ni] {
						continue
					}
					sum += prev[ni]
					count++
				}
				diffused := self
				if count > 0 {
					diffused = sum / float64(count)
				}

				baseline := clamp(m.LatitudeBaseTempC[i]+1.2, -1.8, 30)
				value := self*selfWeight +
					advected*m.Config.SSTAdvection +
					diffused*m.Config.SSTDiffusion +
					baseline*relaxation

				next[i] = clamp(value, -1.8, 30)
			}
		}
		prev, next = next, prev
	}

	copy(m.SeaSurfaceTempC, prev)
}

func (m *Model) applyOceanTemperatureInfluence() {
	const unvisited = -1
	n := len(m.LandMask)
	dist := make([]int, n)
	sourceAnomaly := make([]float64, n)
	for i := range dist {
		dist[i] = unvisited
	}

	type point struct{ x, z int }
	queue := make([]point, 0, n/2)

	// Every ocean cell is a source. The propagated value is not the absolute SST,
	// but the anomaly relative to the same-latitude SST baseline.
	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			if m.LandMask[i] {
				continue
			}
			baselineSST := clamp(m.LatitudeBaseTempC[i]+1.2, -1.8, 30)
			sourceAnomaly[i] = m.SeaSurfaceTempC[i] - baselineSST
			dist[i] = 0
			queue = append(queue, point{x: x, z: z})
		}
	}

	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	head := 0
	for head < len(queue) {
		p := queue[head]
		head++
		pi := p.z*m.Width + p.x

		for _, d := range dirs {
			nx, nz := p.x+d[0], p.z+d[1]
			if nx < 0 || nx >= m.Width || nz < 0 || nz >= m.Height {
				continue
			}
			ni := nz*m.Width + nx
			if dist[ni] != unvisited {
				continue
			}
			// Ocean cells were already initialized as local sources, so this
			// propagation naturally proceeds only through land.
			dist[ni] = dist[pi] + 1
			sourceAnomaly[ni] = sourceAnomaly[pi]
			queue = append(queue, point{x: nx, z: nz})
		}
	}

	for i := range m.AdjustedAnnualTempC {
		m.AdjustedAnnualTempC[i] = m.AnnualMeanTempC[i]
		if dist[i] < 0 {
			m.OceanDistanceCells[i] = float64(maxInt(m.Width, m.Height))
			continue
		}

		m.OceanDistanceCells[i] = float64(dist[i])
		if !m.LandMask[i] {
			// Air over the sea follows only part of the SST anomaly.
			m.OceanTempCorrectionC[i] = sourceAnomaly[i] * 0.45
			m.AdjustedAnnualTempC[i] += m.OceanTempCorrectionC[i]
			continue
		}

		influence := math.Exp(-float64(dist[i]) / m.Config.CoastalTempDecayCells)
		correction := sourceAnomaly[i] * m.Config.CoastalCurrentInfluence * influence
		m.OceanTempCorrectionC[i] = correction
		m.AdjustedAnnualTempC[i] += correction
	}
}

func (m *Model) computeMoistureAndPrecipitation() {
	moisture, lift, precip := m.solveMoistureAndPrecipitation(
		m.AdjustedAnnualTempC,
		m.WindX,
		m.WindZ,
		m.SeaSurfaceTempC,
		0,
	)
	copy(m.Moisture, moisture)
	copy(m.OrographicLift, lift)
	copy(m.AnnualPrecipMM, precip)
}

func (m *Model) computeSeasonalPrecipitation() {
	n := m.Width * m.Height
	warmSST := make([]float64, n)
	coldSST := make([]float64, n)
	warmMoistureTemp := make([]float64, n)
	coldMoistureTemp := make([]float64, n)

	for i := 0; i < n; i++ {
		// Full warm/cold temperature fields remain available as climate outputs,
		// but moisture capacity only follows part of that thermal swing. This
		// prevents ordinary non-monsoon climates from becoming almost dry in the
		// cold representative season solely because air temperature dropped.
		warmMoistureTemp[i] = lerp(
			m.AdjustedAnnualTempC[i],
			m.SummerTempC[i],
			m.Config.SeasonalMoistureTempCoupling,
		)
		coldMoistureTemp[i] = lerp(
			m.AdjustedAnnualTempC[i],
			m.WinterTempC[i],
			m.Config.SeasonalMoistureTempCoupling,
		)

		if m.LandMask[i] {
			continue
		}
		sstDelta := m.SeasonalAmplitudeC[i] * m.Config.SeasonalSeaSurfaceTempCoupling
		warmSST[i] = clamp(m.SeaSurfaceTempC[i]+sstDelta, -1.8, 32)
		coldSST[i] = clamp(m.SeaSurfaceTempC[i]-sstDelta, -1.8, 30)
	}

	_, _, warmPotential := m.solveMoistureAndPrecipitation(
		warmMoistureTemp,
		m.SummerWindX,
		m.SummerWindZ,
		warmSST,
		m.Config.SeasonalBackgroundMoistureBlend,
	)
	_, _, coldPotential := m.solveMoistureAndPrecipitation(
		coldMoistureTemp,
		m.WinterWindX,
		m.WinterWindZ,
		coldSST,
		m.Config.SeasonalBackgroundMoistureBlend,
	)

	for i := 0; i < n; i++ {
		// Each representative state stands for roughly half a year.
		warm := warmPotential[i] * 0.5
		cold := coldPotential[i] * 0.5
		total := warm + cold

		m.WarmSeasonPrecipMM[i] = warm
		m.ColdSeasonPrecipMM[i] = cold
		m.SeasonalAnnualPrecipMM[i] = total

		denom := total + 1
		m.RainfallSeasonality[i] = clamp(math.Abs(warm-cold)/denom, 0, 1)

		if m.LandMask[i] && warm > cold {
			warmExcessShare := (warm - cold) / denom
			m.MonsoonRainRatio[i] = clamp(warmExcessShare*m.MonsoonIndex[i], 0, 1)
		}
	}
}

func (m *Model) solveMoistureAndPrecipitation(
	tempC, windX, windZ, oceanSurfaceTemp []float64,
	backgroundMoistureBlend float64,
) (moisture, lift, precip []float64) {
	n := m.Width * m.Height
	if len(tempC) != n || len(windX) != n || len(windZ) != n || len(oceanSurfaceTemp) != n {
		return make([]float64, n), make([]float64, n), make([]float64, n)
	}

	prev := make([]float64, n)
	next := make([]float64, n)
	lift = make([]float64, n)
	precip = make([]float64, n)

	for i := 0; i < n; i++ {
		if !m.LandMask[i] {
			warmth := clamp((oceanSurfaceTemp[i]+1.8)/(32+1.8), 0, 1)
			prev[i] = 0.35 + warmth*0.65
		}
		lift[i] = m.computeOrographicLiftAt(i, windX, windZ)
	}

	for iter := 0; iter < m.Config.MoistureIterations; iter++ {
		for z := 0; z < m.Height; z++ {
			for x := 0; x < m.Width; x++ {
				i := z*m.Width + x
				wx, wz := windX[i], windZ[i]
				mag := math.Hypot(wx, wz)

				advected := prev[i]
				if mag > 1e-6 {
					ux, uz := wx/mag, wz/mag
					advected = sampleScalar(prev, m.Width, m.Height, float64(x)-ux, float64(z)-uz)
				}

				diffused := neighborAverage(prev, m.Width, m.Height, x, z)
				lowPressureLift := clamp((0.58-m.PressureIndex[i])/0.36, 0, 1)

				rainout := 0.003 + lowPressureLift*0.008 + lift[i]*0.10
				rainout = clamp(rainout, 0.002, 0.18)

				oceanSource := 0.0
				landRecycling := 0.0
				if !m.LandMask[i] {
					warmth := clamp((oceanSurfaceTemp[i]+1.8)/(32+1.8), 0, 1)
					oceanSource = (0.20 + 0.35*warmth) * 0.10
				} else {
					warmth := clamp((tempC[i]+20)/40, 0, 1)
					recyclable := math.Min(prev[i], 0.8)
					landRecycling = recyclable * (0.008 + 0.028*warmth)
				}

				selfWeight := 1 - m.Config.MoistureAdvection - m.Config.MoistureDiffusion
				if selfWeight < 0.05 {
					selfWeight = 0.05
				}

				available := prev[i]*selfWeight +
					advected*m.Config.MoistureAdvection +
					diffused*m.Config.MoistureDiffusion +
					oceanSource +
					landRecycling
				next[i] = clamp(available*(1-rainout), 0, 1.6)
			}
		}
		prev, next = next, prev
	}

	moisture = append([]float64(nil), prev...)
	if backgroundMoistureBlend > 0 && len(m.Moisture) == n {
		blend := clamp(backgroundMoistureBlend, 0, 0.8)
		for i := range moisture {
			moisture[i] = lerp(moisture[i], m.Moisture[i], blend)
		}
	}

	for i := 0; i < n; i++ {
		lowPressureLift := clamp((0.58-m.PressureIndex[i])/0.36, 0, 1)
		condensation := 0.018 + lowPressureLift*0.045 + lift[i]*0.30

		tempFactor := clamp((tempC[i]+25)/45, 0.05, 1)
		rate := moisture[i] * condensation * tempFactor
		precip[i] = clamp(rate*m.Config.PrecipitationScaleMM, 0, 6000)
	}

	return moisture, lift, precip
}

func (m *Model) computeOrographicLiftAt(i int, windX, windZ []float64) float64 {
	x := i % m.Width
	z := i / m.Width

	left := m.LandElevationM[i]
	right := m.LandElevationM[i]
	up := m.LandElevationM[i]
	down := m.LandElevationM[i]

	if x > 0 {
		left = m.LandElevationM[i-1]
	}
	if x+1 < m.Width {
		right = m.LandElevationM[i+1]
	}
	if z > 0 {
		up = m.LandElevationM[i-m.Width]
	}
	if z+1 < m.Height {
		down = m.LandElevationM[i+m.Width]
	}

	gradX := (right - left) / (2 * math.Max(1, m.Config.MaxLandElevationM))
	gradZ := (down - up) / (2 * math.Max(1, m.Config.MaxLandElevationM))

	wx, wz := windX[i], windZ[i]
	mag := math.Hypot(wx, wz)
	if mag < 1e-6 {
		return 0
	}
	wx /= mag
	wz /= mag

	upslope := wx*gradX + wz*gradZ
	return clamp(upslope*12, 0, 1)
}

func sampleScalar(values []float64, width, height int, x, z float64) float64 {
	x = clamp(x, 0, float64(width-1))
	z = clamp(z, 0, float64(height-1))

	x0 := int(math.Floor(x))
	z0 := int(math.Floor(z))
	x1 := x0 + 1
	z1 := z0 + 1
	if x1 >= width {
		x1 = width - 1
	}
	if z1 >= height {
		z1 = height - 1
	}
	tx := x - float64(x0)
	tz := z - float64(z0)

	v00 := values[z0*width+x0]
	v10 := values[z0*width+x1]
	v01 := values[z1*width+x0]
	v11 := values[z1*width+x1]

	top := v00 + (v10-v00)*tx
	bottom := v01 + (v11-v01)*tx
	return top + (bottom-top)*tz
}

func neighborAverage(values []float64, width, height, x, z int) float64 {
	sum := 0.0
	count := 0
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, nz := x+d[0], z+d[1]
		if nx < 0 || nx >= width || nz < 0 || nz >= height {
			continue
		}
		sum += values[nz*width+nx]
		count++
	}
	if count == 0 {
		return values[z*width+x]
	}
	return sum / float64(count)
}

func (m *Model) computeSeasonalStates() {
	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			absLat := math.Abs(m.LatitudeDeg[i])
			latFactor := math.Sin(absLat * math.Pi / 180)

			oceanAmplitude := m.Config.OceanSeasonalityBaseC +
				m.Config.OceanSeasonalityLatitudeC*latFactor

			amplitude := oceanAmplitude
			monsoon := 0.0

			if m.LandMask[i] {
				distance := m.OceanDistanceCells[i]
				continentality := 1 - math.Exp(-distance/28.0)
				landBase := m.Config.LandSeasonalityBaseC +
					m.Config.LandSeasonalityLatitudeC*latFactor
				amplitude = landBase * (0.72 + 0.72*continentality)

				thermalContrast := math.Max(0, amplitude-oceanAmplitude)
				latWeight := smoothstep(5, 14, absLat) *
					(1 - smoothstep(38, 52, absLat))
				oceanAccess := math.Exp(-distance / 95.0)
				monsoon = clamp((thermalContrast/11.0)*latWeight*(0.45+0.55*oceanAccess), 0, 1)
			}

			m.SeasonalAmplitudeC[i] = amplitude
			m.SummerTempC[i] = m.AdjustedAnnualTempC[i] + amplitude
			m.WinterTempC[i] = m.AdjustedAnnualTempC[i] - amplitude
			m.MonsoonIndex[i] = monsoon

			m.SummerWindX[i] = m.WindX[i]
			m.SummerWindZ[i] = m.WindZ[i]
			m.WinterWindX[i] = m.WindX[i]
			m.WinterWindZ[i] = m.WindZ[i]

			if !m.LandMask[i] || monsoon <= 0 {
				continue
			}

			dx, dz := m.oceanDistanceGradient(x, z)
			mag := math.Hypot(dx, dz)
			if mag < 1e-6 {
				continue
			}
			dx /= mag
			dz /= mag

			anomaly := monsoon * m.Config.MonsoonWindStrength

			// Distance-to-ocean increases inland, so +gradient points from sea
			// toward the continental interior. Local summer strengthens onshore
			// flow; local winter reverses toward the sea.
			m.SummerWindX[i] += dx * anomaly
			m.SummerWindZ[i] += dz * anomaly
			m.WinterWindX[i] -= dx * anomaly
			m.WinterWindZ[i] -= dz * anomaly
		}
	}
}

func (m *Model) oceanDistanceGradient(x, z int) (dx, dz float64) {
	i := z*m.Width + x
	left := m.OceanDistanceCells[i]
	right := m.OceanDistanceCells[i]
	up := m.OceanDistanceCells[i]
	down := m.OceanDistanceCells[i]

	if x > 0 {
		left = m.OceanDistanceCells[i-1]
	}
	if x+1 < m.Width {
		right = m.OceanDistanceCells[i+1]
	}
	if z > 0 {
		up = m.OceanDistanceCells[i-m.Width]
	}
	if z+1 < m.Height {
		down = m.OceanDistanceCells[i+m.Width]
	}

	return (right - left) * 0.5, (down - up) * 0.5
}
