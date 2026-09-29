package ecology

import (
	"errors"
	"math"
)

func Generate(input Input, cfg Config) (*Model, error) {
	if input.Width < 2 || input.Height < 2 {
		return nil, errors.New("ecology: grid must be at least 2x2")
	}
	n := input.Width * input.Height

	fields := []struct {
		name string
		data []float64
	}{
		{"annual temp", input.AnnualTempC},
		{"summer temp", input.SummerTempC},
		{"winter temp", input.WinterTempC},
		{"annual precipitation", input.AnnualPrecipMM},
		{"warm season precipitation", input.WarmSeasonPrecipMM},
		{"cold season precipitation", input.ColdSeasonPrecipMM},
		{"rainfall seasonality", input.RainfallSeasonality},
		{"monsoon index", input.MonsoonIndex},
		{"sea surface temp", input.SeaSurfaceTempC},
		{"elevation", input.Elevation},
		{"relief", input.ReliefIndex},
		{"rolling plain", input.RollingPlainWeight},
		{"hill", input.HillWeight},
		{"foothill", input.FoothillWeight},
		{"mountain", input.MountainWeight},
		{"plateau", input.PlateauWeight},
		{"wetland", input.WetlandPotential},
		{"perennial river", input.PerennialRiverPotential},
		{"seasonal river", input.SeasonalRiverPotential},
		{"floodplain", input.FloodplainWeight},
		{"alluvial plain", input.AlluvialPlainWeight},
		{"valley floor width", input.ValleyFloorWidth},
	}
	for _, field := range fields {
		if len(field.data) != n {
			return nil, errors.New("ecology: " + field.name + " size does not match dimensions")
		}
	}
	if len(input.LandMask) != n {
		return nil, errors.New("ecology: land mask size does not match dimensions")
	}

	if cfg.ReferencePrecipMM <= 0 {
		cfg.ReferencePrecipMM = 1800
	}

	m := &Model{
		Config:              cfg,
		Width:               input.Width,
		Height:              input.Height,
		LandMask:            append([]bool(nil), input.LandMask...),
		MoistureIndex:       make([]float64, n),
		AridityIndex:        make([]float64, n),
		WarmSeasonRainShare: make([]float64, n),
		ColdSeasonRainShare: make([]float64, n),
		ThermalSeasonality:  make([]float64, n),
		WetlandEcology:      make([]float64, n),
		AlpineEcology:       make([]float64, n),
		TreeCoverPotential:  make([]float64, n),
		GrassCoverPotential: make([]float64, n),
		ShrubCoverPotential: make([]float64, n),
		BareGroundPotential: make([]float64, n),
		SnowCoverPotential:  make([]float64, n),
		SoilMoisture:        make([]float64, n),
		VegetationDensity:   make([]float64, n),
		EcoRegion:           make([]Region, n),
		RegionConfidence:    make([]float64, n),
	}

	for i := 0; i < n; i++ {
		m.computeIndices(i, input)
		if input.LandMask[i] {
			m.classifyLand(i, input)
		} else {
			m.classifyOcean(i, input)
		}
	}
	return m, nil
}

func (m *Model) computeIndices(i int, input Input) {
	annualT := input.AnnualTempC[i]
	summerT := input.SummerTempC[i]
	winterT := input.WinterTempC[i]
	annualP := math.Max(0, input.AnnualPrecipMM[i])
	warmP := math.Max(0, input.WarmSeasonPrecipMM[i])
	coldP := math.Max(0, input.ColdSeasonPrecipMM[i])

	// A compact potential-evapotranspiration proxy. It is intentionally not a
	// claim of physical PET; it only turns temperature into a water-demand scale
	// so that the same precipitation means different things in hot and cold areas.
	petProxy := math.Max(120, 420+32*math.Max(annualT, 0))
	moisture := annualP / (annualP + petProxy)
	m.MoistureIndex[i] = clamp(moisture, 0, 1)
	m.AridityIndex[i] = 1 - m.MoistureIndex[i]

	seasonTotal := warmP + coldP
	if seasonTotal > 1e-9 {
		m.WarmSeasonRainShare[i] = clamp(warmP/seasonTotal, 0, 1)
		m.ColdSeasonRainShare[i] = clamp(coldP/seasonTotal, 0, 1)
	} else {
		m.WarmSeasonRainShare[i] = 0.5
		m.ColdSeasonRainShare[i] = 0.5
	}

	m.ThermalSeasonality[i] = clamp(math.Abs(summerT-winterT)/36, 0, 1)

	wetland := clamp(
		0.38*input.WetlandPotential[i]+
			0.26*input.FloodplainWeight[i]+
			0.16*input.AlluvialPlainWeight[i]+
			0.12*input.ValleyFloorWidth[i]+
			0.08*input.PerennialRiverPotential[i],
		0,
		1,
	)
	m.WetlandEcology[i] = wetland

	coldEnough := smoothstep(12, -2, annualT)
	mountainContext := clamp(
		0.66*input.MountainWeight[i]+
			0.20*input.FoothillWeight[i]+
			0.14*input.PlateauWeight[i],
		0,
		1,
	)
	m.AlpineEcology[i] = clamp(
		mountainContext*
			(0.35+0.65*coldEnough)*
			(0.45+0.55*input.ReliefIndex[i]),
		0,
		1,
	)

	if !input.LandMask[i] {
		return
	}

	tempGrowth := smoothstep(-8, 6, annualT) * (1 - 0.45*smoothstep(30, 38, annualT))
	waterGrowth := smoothstep(0.22, 0.56, m.MoistureIndex[i])
	m.VegetationDensity[i] = clamp(tempGrowth*waterGrowth, 0, 1)

	treeClimate := smoothstep(0.42, 0.68, m.MoistureIndex[i]) *
		smoothstep(-5, 10, annualT) *
		(1 - 0.72*smoothstep(0.68, 0.94, input.RainfallSeasonality[i]))
	treeSeasonal := smoothstep(0.48, 0.76, m.MoistureIndex[i]) *
		smoothstep(0.35, 0.72, input.RainfallSeasonality[i]) *
		smoothstep(10, 22, summerT)
	m.TreeCoverPotential[i] = clamp(math.Max(treeClimate, 0.82*treeSeasonal)*(1-0.62*m.AlpineEcology[i]), 0, 1)

	grassMoisture := smoothBand(m.MoistureIndex[i], 0.22, 0.38, 0.68, 0.82)
	grassTemp := smoothstep(-6, 8, summerT)
	m.GrassCoverPotential[i] = clamp(
		grassMoisture*grassTemp*(0.55+0.45*input.RollingPlainWeight[i])+
			0.35*m.AlpineEcology[i],
		0,
		1,
	)

	shrubDry := smoothBand(m.MoistureIndex[i], 0.16, 0.30, 0.48, 0.66)
	shrubSeasonal := smoothstep(0.35, 0.72, input.RainfallSeasonality[i])
	m.ShrubCoverPotential[i] = clamp(
		shrubDry*(0.58+0.42*shrubSeasonal)*
			smoothstep(-4, 9, annualT),
		0,
		1,
	)

	m.BareGroundPotential[i] = clamp(
		0.75*smoothstep(0.62, 0.86, m.AridityIndex[i])+
			0.65*smoothstep(0, -12, annualT)+
			0.25*m.AlpineEcology[i],
		0,
		1,
	)

	m.SnowCoverPotential[i] = clamp(
		0.60*smoothstep(2, -10, winterT)+
			0.28*smoothstep(0, -8, annualT)+
			0.25*m.AlpineEcology[i],
		0,
		1,
	)

	m.SoilMoisture[i] = clamp(
		0.72*m.MoistureIndex[i]+
			0.18*m.WetlandEcology[i]+
			0.10*input.PerennialRiverPotential[i],
		0,
		1,
	)
}

func (m *Model) classifyOcean(i int, input Input) {
	t := input.SeaSurfaceTempC[i]
	scores := []regionScore{
		{RegionFrozenOcean, smoothstep(2, -1.4, t)},
		{RegionColdOcean, smoothBand(t, -1.5, 1.5, 8, 13)},
		{RegionTemperateOcean, smoothBand(t, 7, 12, 20, 25)},
		{RegionWarmOcean, smoothstep(20, 27, t)},
	}
	m.assignBest(i, scores)
}

func (m *Model) classifyLand(i int, input Input) {
	t := input.AnnualTempC[i]
	winter := input.WinterTempC[i]
	summer := input.SummerTempC[i]
	moist := m.MoistureIndex[i]
	arid := m.AridityIndex[i]
	season := clamp(input.RainfallSeasonality[i], 0, 1)
	monsoon := clamp(input.MonsoonIndex[i], 0, 1)
	warmRain := m.WarmSeasonRainShare[i]
	coldRain := m.ColdSeasonRainShare[i]

	tropical := smoothstep(20, 25, t)
	subtropical := smoothBand(t, 12, 17, 23, 29)
	temperate := smoothBand(t, 2, 8, 17, 23)
	boreal := smoothBand(t, -8, -2, 5, 10)
	polar := smoothstep(2, -7, t)

	veryWet := smoothstep(0.62, 0.78, moist)
	wet := smoothstep(0.48, 0.66, moist)
	subhumid := smoothBand(moist, 0.28, 0.40, 0.57, 0.70)
	semiArid := smoothBand(moist, 0.16, 0.26, 0.38, 0.50)
	aridStrong := smoothstep(0.64, 0.82, arid)

	warmSeasonWet := smoothstep(0.57, 0.76, warmRain)
	warmSeasonDry := smoothstep(0.57, 0.74, coldRain)
	strongSeasonality := smoothstep(0.38, 0.72, season)

	ice := polar *
		smoothstep(-4, -14, t) *
		(0.55 + 0.45*m.SnowCoverPotential[i])

	tundra := polar *
		smoothBand(summer, -2, 3, 9, 14) *
		(1 - 0.65*ice)

	alpine := math.Pow(m.AlpineEcology[i], 0.72) *
		smoothstep(13, 2, t)

	borealForest := boreal *
		wet *
		smoothstep(-22, -7, winter) *
		(1 - 0.92*smoothstep(0.22, 0.62, m.AlpineEcology[i]))

	temperateForest := temperate *
		wet *
		(0.72 + 0.28*(1-strongSeasonality))

	temperateGrassland := temperate *
		subhumid *
		(0.65 + 0.35*strongSeasonality)

	temperateDesert := temperate *
		semiArid *
		smoothstep(0.50, 0.74, arid)

	mediterranean := (0.55*subtropical + 0.45*temperate) *
		smoothBand(moist, 0.28, 0.40, 0.56, 0.70) *
		warmSeasonDry *
		strongSeasonality

	subtropicalForest := subtropical *
		wet *
		(0.55 + 0.45*(1-warmSeasonDry))

	hotDesert := math.Max(tropical, subtropical) *
		aridStrong

	savanna := tropical *
		subhumid *
		(0.58 + 0.42*warmSeasonWet) *
		(0.62 + 0.38*strongSeasonality)

	tropicalMonsoon := tropical *
		wet *
		strongSeasonality *
		(0.45 + 0.55*math.Max(monsoon, warmSeasonWet))

	tropicalRainforest := tropical *
		veryWet *
		(1 - 0.62*strongSeasonality) *
		(0.72 + 0.28*(1-monsoon))

	wetland := smoothstep(0.42, 0.78, m.WetlandEcology[i]) *
		smoothstep(0.38, 0.64, moist) *
		smoothstep(-4, 8, summer)

	scores := []regionScore{
		{RegionIceSheet, ice},
		{RegionTundra, tundra},
		{RegionAlpine, alpine},
		{RegionBorealForest, borealForest},
		{RegionTemperateForest, temperateForest},
		{RegionTemperateGrassland, temperateGrassland},
		{RegionTemperateDesert, temperateDesert},
		{RegionMediterraneanShrubland, mediterranean},
		{RegionSubtropicalForest, subtropicalForest},
		{RegionHotDesert, hotDesert},
		{RegionSavanna, savanna},
		{RegionTropicalMonsoonForest, tropicalMonsoon},
		{RegionTropicalRainforest, tropicalRainforest},
		{RegionWetland, wetland},
	}

	// A low-confidence fallback prevents numerical holes while still exposing
	// the confidence value so weak classifications can be audited.
	if maxScore(scores) < 0.015 {
		switch {
		case t < -2:
			scores = append(scores, regionScore{RegionTundra, 0.02})
		case arid > 0.62:
			scores = append(scores, regionScore{RegionTemperateDesert, 0.02})
		case t > 20:
			scores = append(scores, regionScore{RegionSavanna, 0.02})
		default:
			scores = append(scores, regionScore{RegionTemperateGrassland, 0.02})
		}
	}

	m.assignBest(i, scores)
}

type regionScore struct {
	region Region
	score  float64
}

func (m *Model) assignBest(i int, scores []regionScore) {
	best := RegionUnknown
	bestScore := -1.0
	second := 0.0
	sum := 0.0

	for _, s := range scores {
		score := math.Max(0, s.score)
		sum += score
		if score > bestScore {
			second = bestScore
			bestScore = score
			best = s.region
		} else if score > second {
			second = score
		}
	}

	m.EcoRegion[i] = best
	if sum <= 1e-12 || bestScore <= 0 {
		m.RegionConfidence[i] = 0
		return
	}

	// Confidence rewards both a strong best score and separation from the
	// runner-up. It is intentionally descriptive, not a probability.
	separation := clamp((bestScore-second)/(bestScore+1e-9), 0, 1)
	strength := clamp(bestScore, 0, 1)
	m.RegionConfidence[i] = clamp(0.58*strength+0.42*separation, 0, 1)
}

func maxScore(scores []regionScore) float64 {
	maxV := 0.0
	for _, s := range scores {
		if s.score > maxV {
			maxV = s.score
		}
	}
	return maxV
}

// smoothstep supports both ascending and descending edges.
func smoothstep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	if edge1 > edge0 {
		t := clamp((x-edge0)/(edge1-edge0), 0, 1)
		return t * t * (3 - 2*t)
	}
	t := clamp((edge0-x)/(edge0-edge1), 0, 1)
	return t * t * (3 - 2*t)
}

func smoothBand(x, rise0, rise1, fall0, fall1 float64) float64 {
	return smoothstep(rise0, rise1, x) * (1 - smoothstep(fall0, fall1, x))
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
