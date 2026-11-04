package settlement

import (
	"math"
	"strings"
)

// Inputs are immutable Natural Model facts used to evaluate regional settlement
// suitability. They intentionally contain no loaded-chunk or gameplay state.
type Inputs struct {
	Land      bool
	WaterKind string
	Slope     float64
	Terrain   string
	Surface   string
	Biome     string

	AnnualMeanTempC float64
	AnnualPrecipMM  float64

	DistanceToRiverBlocks float64
	HasRiverDistance      bool
	DistanceToLakeBlocks  float64
	HasLakeDistance       bool
}

// Score keeps every contribution independently inspectable. All values use
// [0,1], except FloodRisk where 1 means highest natural flood exposure.
type Score struct {
	Buildability           float64 `json:"buildability"`
	FreshWaterAccess       float64 `json:"freshWaterAccess"`
	FloodRisk              float64 `json:"floodRisk"`
	AgricultureSuitability float64 `json:"agricultureSuitability"`
	ClimateComfort         float64 `json:"climateComfort"`
	TransportAccess        float64 `json:"transportAccess"`
	ResourceAccess         float64 `json:"resourceAccess"`
	SettlementSuitability  float64 `json:"settlementSuitability"`
}

// Evaluate derives a civilization-planning score only from locked natural facts.
// It does not decide where settlements exist; gameplay/add-on layers may apply
// their own weights to the independent indicators.
func Evaluate(in Inputs) Score {
	if !in.Land || !isDryLand(in.WaterKind) {
		return Score{FloodRisk: 1}
	}

	buildability := buildabilityScore(in.Slope, in.Terrain)
	freshWater := freshWaterScore(in)
	floodRisk := floodRiskScore(in)
	agriculture := agricultureScore(in)
	climate := climateComfortScore(in.AnnualMeanTempC, in.AnnualPrecipMM)
	transport := transportScore(in, buildability, freshWater)
	resources := resourceScore(in)

	// The default composite is deliberately conservative. Severe terrain or
	// flood constraints cannot be completely cancelled by good climate/water.
	safety := 1 - floodRisk
	composite :=
		0.22*buildability +
			0.16*freshWater +
			0.16*safety +
			0.16*agriculture +
			0.10*climate +
			0.12*transport +
			0.08*resources
	composite *= 0.55 + 0.45*buildability
	composite *= 0.62 + 0.38*safety

	return Score{
		Buildability:           clamp01(buildability),
		FreshWaterAccess:       clamp01(freshWater),
		FloodRisk:              clamp01(floodRisk),
		AgricultureSuitability: clamp01(agriculture),
		ClimateComfort:         clamp01(climate),
		TransportAccess:        clamp01(transport),
		ResourceAccess:         clamp01(resources),
		SettlementSuitability:  clamp01(composite),
	}
}

func buildabilityScore(slope float64, terrain string) float64 {
	slope = math.Max(0, slope)
	slopeScore := 1 - smoothstep(0.0015, 0.014, slope)
	terrainScore := 0.58
	switch normalize(terrain) {
	case "plain":
		terrainScore = 1
	case "rolling plain":
		terrainScore = 0.92
	case "alluvial plain":
		terrainScore = 0.98
	case "floodplain":
		terrainScore = 0.90
	case "alluvial fan":
		terrainScore = 0.84
	case "plateau":
		terrainScore = 0.78
	case "hills":
		terrainScore = 0.62
	case "foothills":
		terrainScore = 0.55
	case "mountain":
		terrainScore = 0.18
	case "canyon":
		terrainScore = 0.12
	case "ocean":
		terrainScore = 0
	}
	return slopeScore * terrainScore
}

func freshWaterScore(in Inputs) float64 {
	best := math.Inf(1)
	if in.HasRiverDistance {
		best = math.Min(best, math.Max(0, in.DistanceToRiverBlocks))
	}
	if in.HasLakeDistance {
		best = math.Min(best, math.Max(0, in.DistanceToLakeBlocks))
	}
	if !math.IsInf(best, 1) {
		// Good settlement water access extends well beyond the immediate bank.
		// Flood danger is modelled independently rather than making rivers bad.
		return 1 - smoothstep(128, 1600, best)
	}
	return 0.12
}

func floodRiskScore(in Inputs) float64 {
	risk := 0.04
	switch normalize(in.Terrain) {
	case "floodplain":
		risk = 0.72
	case "alluvial plain":
		risk = 0.34
	case "alluvial fan":
		risk = 0.22
	case "canyon":
		risk = 0.16
	}

	if in.HasRiverDistance {
		d := math.Max(0, in.DistanceToRiverBlocks)
		riverRisk := 0.0
		switch {
		case d <= 24:
			riverRisk = 0.92
		case d <= 64:
			riverRisk = lerp(0.92, 0.62, (d-24)/40)
		case d <= 180:
			riverRisk = lerp(0.62, 0.22, (d-64)/116)
		case d <= 480:
			riverRisk = lerp(0.22, 0.04, (d-180)/300)
		}
		risk = math.Max(risk, riverRisk)
	}
	if in.HasLakeDistance {
		d := math.Max(0, in.DistanceToLakeBlocks)
		lakeRisk := 0.0
		switch {
		case d <= 16:
			lakeRisk = 0.62
		case d <= 96:
			lakeRisk = lerp(0.62, 0.26, (d-16)/80)
		case d <= 320:
			lakeRisk = lerp(0.26, 0.04, (d-96)/224)
		}
		risk = math.Max(risk, lakeRisk)
	}

	surface := normalize(in.Surface)
	if strings.Contains(surface, "wet mud") || strings.Contains(surface, "peat") {
		risk = math.Max(risk, 0.68)
	}
	if normalize(in.Biome) == "wetland" {
		risk = math.Max(risk, 0.80)
	}
	return clamp01(risk)
}

func agricultureScore(in Inputs) float64 {
	temp := bandScore(in.AnnualMeanTempC, -4, 8, 24, 36)
	precip := bandScore(in.AnnualPrecipMM, 120, 450, 1800, 3400)

	soil := 0.52
	switch normalize(in.Surface) {
	case "grass loam":
		soil = 1
	case "alluvial silt":
		soil = 0.98
	case "forest loam":
		soil = 0.86
	case "dry grass soil":
		soil = 0.70
	case "shrub soil":
		soil = 0.56
	case "podzol":
		soil = 0.48
	case "wet mud":
		soil = 0.42
	case "peat":
		soil = 0.38
	case "tundra soil":
		soil = 0.24
	case "desert sand", "desert gravel", "alpine rock", "alpine snow", "ice / snow":
		soil = 0.10
	}

	terrain := buildabilityScore(in.Slope, in.Terrain)
	floodPenalty := 1 - 0.45*floodRiskScore(in)
	return clamp01((0.34*temp + 0.28*precip + 0.24*soil + 0.14*terrain) * floodPenalty)
}

func climateComfortScore(tempC, precipMM float64) float64 {
	temp := bandScore(tempC, -12, 7, 24, 38)
	precip := bandScore(precipMM, 80, 320, 1800, 3600)
	return clamp01(0.72*temp + 0.28*precip)
}

func transportScore(in Inputs, buildability, freshWater float64) float64 {
	waterway := 0.0
	if in.HasRiverDistance {
		d := math.Max(0, in.DistanceToRiverBlocks)
		waterway = math.Max(waterway, 1-smoothstep(96, 1100, d))
	}
	if in.HasLakeDistance {
		d := math.Max(0, in.DistanceToLakeBlocks)
		waterway = math.Max(waterway, 0.88*(1-smoothstep(96, 900, d)))
	}

	terrainBonus := 0.0
	switch normalize(in.Terrain) {
	case "plain", "alluvial plain":
		terrainBonus = 1
	case "rolling plain", "floodplain":
		terrainBonus = 0.88
	case "alluvial fan", "plateau":
		terrainBonus = 0.68
	case "hills", "foothills":
		terrainBonus = 0.44
	case "mountain", "canyon":
		terrainBonus = 0.12
	}
	return clamp01(0.52*buildability + 0.30*waterway + 0.18*terrainBonus + 0.04*freshWater)
}

func resourceScore(in Inputs) float64 {
	score := 0.50
	switch normalize(in.Terrain) {
	case "foothills":
		score = 0.88
	case "hills":
		score = 0.84
	case "mountain":
		score = 0.82
	case "plateau":
		score = 0.78
	case "rolling plain":
		score = 0.66
	case "plain", "alluvial plain", "floodplain":
		score = 0.58
	case "alluvial fan":
		score = 0.64
	case "canyon":
		score = 0.72
	}

	vegetated := normalize(in.Biome)
	if strings.Contains(vegetated, "forest") {
		score += 0.08
	}
	if freshWaterScore(in) > 0.55 {
		score += 0.05
	}
	return clamp01(score)
}

func bandScore(v, hardLow, goodLow, goodHigh, hardHigh float64) float64 {
	switch {
	case v <= hardLow || v >= hardHigh:
		return 0
	case v < goodLow:
		return smoothstep(hardLow, goodLow, v)
	case v <= goodHigh:
		return 1
	default:
		return 1 - smoothstep(goodHigh, hardHigh, v)
	}
}

func isDryLand(waterKind string) bool {
	k := normalize(waterKind)
	return k == "" || k == "none"
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func smoothstep(edge0, edge1, x float64) float64 {
	if edge1 <= edge0 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*clamp01(t)
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
