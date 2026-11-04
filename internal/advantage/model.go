package advantage

import (
	"math"
	"sort"
)

const highAdvantageThreshold = 0.68

const Macro05CoreTripleOverlapLimit = 0.15

type Inputs struct {
	Land bool

	SettlementSuitability  float64
	Buildability           float64
	FreshWaterAccess       float64
	FloodRisk              float64
	AgricultureSuitability float64
	TransportAccess        float64
	ResourceAccess         float64

	WaterwayNavigability     float64
	PortSuitability          float64
	NavigationBottleneckRisk float64

	MiningSuitability float64
	IndustrialValue   float64
	ResourceDiversity float64
}

type Score struct {
	BuildingAdvantage      float64 `json:"buildingAdvantage"`
	AgricultureAdvantage   float64 `json:"agricultureAdvantage"`
	PortAdvantage          float64 `json:"portAdvantage"`
	MiningAdvantage        float64 `json:"miningAdvantage"`
	TransportAdvantage     float64 `json:"transportAdvantage"`
	SpecializationStrength float64 `json:"specializationStrength"`
	AdvantageOverlapRisk   float64 `json:"advantageOverlapRisk"`
	HighAdvantageCount     int     `json:"highAdvantageCount"`
	DominantAdvantage      string  `json:"dominantAdvantage"`
	RegionalClass          string  `json:"regionalClass"`
}

type Summary struct {
	LandCells                 int            `json:"landCells"`
	PotentialCells            int            `json:"potentialCells"`
	HighOverlapCells          int            `json:"highOverlapCells"`
	CoreTripleOverlapCells    int            `json:"coreTripleOverlapCells"`
	HighOverlapFraction       float64        `json:"highOverlapFraction"`
	CoreTripleOverlapFraction float64        `json:"coreTripleOverlapFraction"`
	MeanOverlapRisk           float64        `json:"meanOverlapRisk"`
	RoleCoverage              int            `json:"roleCoverage"`
	DominantCounts            map[string]int `json:"dominantCounts"`
	Macro05Satisfied          bool           `json:"macro05Satisfied"`
	Macro05CoreTripleLimit    float64        `json:"macro05CoreTripleLimit"`
}

func Evaluate(in Inputs) Score {
	if !in.Land {
		return Score{
			DominantAdvantage: "none",
			RegionalClass:     "none",
		}
	}

	settlement := clamp01(in.SettlementSuitability)
	build := clamp01(in.Buildability)
	fresh := clamp01(in.FreshWaterAccess)
	floodSafety := 1 - clamp01(in.FloodRisk)
	agriculture := clamp01(in.AgricultureSuitability)
	transport := clamp01(in.TransportAccess)
	resource := clamp01(in.ResourceAccess)
	waterway := clamp01(in.WaterwayNavigability)
	port := clamp01(in.PortSuitability)
	navigationSafety := 1 - clamp01(in.NavigationBottleneckRisk)
	mining := clamp01(in.MiningSuitability)
	industrial := clamp01(in.IndustrialValue)
	diversity := clamp01(in.ResourceDiversity)

	buildingAdvantage := clamp01(
		0.45*build +
			0.30*settlement +
			0.15*floodSafety +
			0.10*transport,
	)
	agricultureAdvantage := clamp01(
		0.50*agriculture +
			0.18*fresh +
			0.12*floodSafety +
			0.10*settlement +
			0.10*transport,
	)
	portAdvantage := clamp01(
		0.65*port +
			0.12*waterway +
			0.08*transport +
			0.07*settlement +
			0.08*navigationSafety,
	)
	miningAdvantage := clamp01(
		0.48*mining +
			0.22*industrial +
			0.15*resource +
			0.08*transport +
			0.07*diversity,
	)
	transportAdvantage := clamp01(
		0.55*transport +
			0.22*build +
			0.08*waterway +
			0.08*settlement +
			0.07*navigationSafety,
	)

	values := []float64{
		buildingAdvantage,
		agricultureAdvantage,
		portAdvantage,
		miningAdvantage,
		transportAdvantage,
	}
	labels := []string{
		"building",
		"agriculture",
		"port",
		"mining",
		"transport",
	}
	order := []int{0, 1, 2, 3, 4}
	sort.SliceStable(order, func(i, j int) bool {
		return values[order[i]] > values[order[j]]
	})

	best := values[order[0]]
	second := values[order[1]]
	third := values[order[2]]
	highCount := 0
	for _, value := range values {
		if value >= highAdvantageThreshold {
			highCount++
		}
	}

	specialization := clamp01((best - second) / 0.25)
	overlap := clamp01((third - 0.45) / 0.35)
	dominant := labels[order[0]]

	class := dominant + "_region"
	switch {
	case best < 0.45:
		class = "low_potential"
	case highCount >= 4 && best-second <= 0.05:
		class = "multi_advantage_region"
	}

	return Score{
		BuildingAdvantage:      buildingAdvantage,
		AgricultureAdvantage:   agricultureAdvantage,
		PortAdvantage:          portAdvantage,
		MiningAdvantage:        miningAdvantage,
		TransportAdvantage:     transportAdvantage,
		SpecializationStrength: specialization,
		AdvantageOverlapRisk:   overlap,
		HighAdvantageCount:     highCount,
		DominantAdvantage:      dominant,
		RegionalClass:          class,
	}
}

func Summarize(scores []Score) Summary {
	out := Summary{
		DominantCounts: make(map[string]int, 5),
	}
	overlapSum := 0.0
	for _, score := range scores {
		if score.RegionalClass == "none" {
			continue
		}
		out.LandCells++
		best := max5(
			score.BuildingAdvantage,
			score.AgricultureAdvantage,
			score.PortAdvantage,
			score.MiningAdvantage,
			score.TransportAdvantage,
		)
		if best < 0.55 {
			continue
		}
		out.PotentialCells++
		out.DominantCounts[score.DominantAdvantage]++
		overlapSum += score.AdvantageOverlapRisk
		if score.AdvantageOverlapRisk >= 0.70 {
			out.HighOverlapCells++
		}
		if score.BuildingAdvantage >= highAdvantageThreshold &&
			score.PortAdvantage >= highAdvantageThreshold &&
			score.MiningAdvantage >= highAdvantageThreshold {
			out.CoreTripleOverlapCells++
		}
	}
	if out.PotentialCells > 0 {
		denom := float64(out.PotentialCells)
		out.HighOverlapFraction = float64(out.HighOverlapCells) / denom
		out.CoreTripleOverlapFraction =
			float64(out.CoreTripleOverlapCells) / denom
		out.MeanOverlapRisk = overlapSum / denom
	}
	for _, count := range out.DominantCounts {
		if count > 0 {
			out.RoleCoverage++
		}
	}
	out.Macro05CoreTripleLimit = Macro05CoreTripleOverlapLimit
	out.Macro05Satisfied = out.PotentialCells > 0 &&
		out.RoleCoverage >= 3 &&
		out.CoreTripleOverlapFraction <= Macro05CoreTripleOverlapLimit
	return out
}

func clamp01(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 1
	}
	return value
}

func max5(a, b, c, d, e float64) float64 {
	return math.Max(math.Max(a, b), math.Max(c, math.Max(d, e)))
}
