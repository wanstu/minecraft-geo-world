package navigation

import (
	"math"
	"strings"
)

type Inputs struct {
	Land      bool
	WaterKind string

	WaterDepthNormalized float64

	RiverAvailable  bool
	RiverHalfWidth  float64
	RiverGradient   float64
	RiverOceanMouth bool
	RiverLakeMouth  bool
	RiverLakeOutlet bool

	DistanceToRiverBlocks float64
	HasRiverDistance      bool
	DistanceToLakeBlocks  float64
	HasLakeDistance       bool
	DistanceToCoastBlocks float64
	HasCoastDistance      bool

	Buildability          float64
	FloodRisk             float64
	SettlementSuitability float64
}

type Score struct {
	WaterwayNavigability     float64 `json:"waterwayNavigability"`
	PortSuitability          float64 `json:"portSuitability"`
	NavigationBottleneckRisk float64 `json:"navigationBottleneckRisk"`
	WaterwayClass            string  `json:"waterwayClass"`
	PortType                 string  `json:"portType"`
}

func Evaluate(in Inputs) Score {
	kind := normalize(in.WaterKind)
	riverNav, riverBottleneck := riverNavigation(in)
	openWaterNav := 0.0
	switch kind {
	case "ocean":
		openWaterNav = openWaterScore(in.WaterDepthNormalized, 0.002, 0.010)
	case "lake":
		openWaterNav = openWaterScore(in.WaterDepthNormalized, 0.002, 0.008)
	}

	navigability := openWaterNav
	if kind == "river" {
		navigability = riverNav
	}
	if kind == "none" && in.RiverAvailable && in.HasRiverDistance &&
		in.DistanceToRiverBlocks <= 160 {
		navigability = riverNav
	}

	bottleneck := 0.0
	switch kind {
	case "river":
		bottleneck = riverBottleneck
	case "lake", "ocean":
		bottleneck = clamp01(1 - openWaterNav)
	default:
		if in.RiverAvailable && in.HasRiverDistance &&
			in.DistanceToRiverBlocks <= 160 {
			bottleneck = riverBottleneck
		}
	}

	portType, waterAccess, waterQuality := portWaterAccess(in, riverNav)
	isDryLand := in.Land && kind == "none"
	if !isDryLand {
		portType = "none"
	}
	port := 0.0
	if isDryLand && portType != "none" {
		build := clamp01(in.Buildability)
		safety := 1 - clamp01(in.FloodRisk)
		settlement := clamp01(in.SettlementSuitability)
		port = 0.28*waterQuality +
			0.24*build +
			0.18*safety +
			0.16*settlement +
			0.14*waterAccess
		// A port needs both a usable shore/waterway and usable land.
		port *= 0.52 + 0.48*waterAccess
		port *= 0.58 + 0.42*build
		port *= 0.62 + 0.38*safety
	}

	return Score{
		WaterwayNavigability:     clamp01(navigability),
		PortSuitability:          clamp01(port),
		NavigationBottleneckRisk: clamp01(bottleneck),
		WaterwayClass: waterwayClass(
			kind,
			navigability,
			in.RiverAvailable && in.HasRiverDistance &&
				in.DistanceToRiverBlocks <= 160,
		),
		PortType: portType,
	}
}

func riverNavigation(in Inputs) (float64, float64) {
	if !in.RiverAvailable {
		return 0, 0
	}
	width := smoothstep(6.0, 24.0, math.Max(0, in.RiverHalfWidth))
	gradientRisk := smoothstep(0.00008, 0.00036, math.Max(0, in.RiverGradient))
	gradient := 1 - gradientRisk

	mouthBonus := 0.0
	if in.RiverOceanMouth {
		mouthBonus = 0.10
	} else if in.RiverLakeMouth || in.RiverLakeOutlet {
		mouthBonus = 0.06
	}

	navigability := clamp01(0.68*width + 0.32*gradient + mouthBonus)
	widthRisk := 1 - smoothstep(7.5, 18.0, math.Max(0, in.RiverHalfWidth))
	bottleneck := math.Max(widthRisk, gradientRisk)
	if in.RiverOceanMouth || in.RiverLakeMouth || in.RiverLakeOutlet {
		bottleneck *= 0.82
	}
	return navigability, clamp01(bottleneck)
}

func openWaterScore(depth, shallow, good float64) float64 {
	if depth <= 0 {
		// Macro preview/persisted natural models can omit exact local depth.
		// Open lake/ocean remains broadly navigable unless local reality says otherwise.
		return 0.92
	}
	return clamp01(0.72 + 0.28*smoothstep(shallow, good, depth))
}

func portWaterAccess(in Inputs, riverNav float64) (string, float64, float64) {
	bestType := "none"
	bestAccess := 0.0
	bestQuality := 0.0

	if in.HasCoastDistance {
		d := math.Max(0, in.DistanceToCoastBlocks)
		access := shoreBand(d, 16, 220, 640)
		if access > bestAccess {
			bestType = "sea"
			bestAccess = access
			bestQuality = 1.0
		}
	}
	if in.HasLakeDistance {
		d := math.Max(0, in.DistanceToLakeBlocks)
		access := shoreBand(d, 12, 180, 520)
		if access > bestAccess {
			bestType = "lake"
			bestAccess = access
			bestQuality = 0.90
		}
	}
	if in.HasRiverDistance && in.RiverAvailable {
		d := math.Max(0, in.DistanceToRiverBlocks)
		access := shoreBand(d, 8, 150, 420)
		quality := riverNav
		if in.RiverOceanMouth {
			quality = clamp01(quality + 0.12)
		}
		if access*quality > bestAccess*bestQuality {
			bestType = "river"
			bestAccess = access
			bestQuality = quality
		}
	}
	return bestType, clamp01(bestAccess), clamp01(bestQuality)
}

func shoreBand(distance, tooClose, idealFar, maxDistance float64) float64 {
	if distance >= maxDistance {
		return 0
	}
	if distance <= tooClose {
		return 0.72
	}
	if distance <= idealFar {
		return 1
	}
	return 1 - smoothstep(idealFar, maxDistance, distance)
}

func waterwayClass(kind string, nav float64, riverAvailable bool) string {
	if kind == "ocean" {
		if nav >= 0.65 {
			return "open_sea"
		}
		return "shallow_sea"
	}
	if kind == "lake" {
		if nav >= 0.65 {
			return "lake"
		}
		return "shallow_lake"
	}
	if kind == "river" || riverAvailable {
		switch {
		case nav >= 0.82:
			return "major_river"
		case nav >= 0.58:
			return "river"
		case nav >= 0.34:
			return "local_waterway"
		default:
			return "non_navigable"
		}
	}
	return "none"
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

func clamp01(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
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
