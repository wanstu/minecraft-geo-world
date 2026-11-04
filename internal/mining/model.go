package mining

import (
	"math"
	"strings"
)

var Minerals = [...]string{
	"coal", "iron", "copper", "gold",
	"lapis", "redstone", "diamond", "emerald",
}

type Inputs struct {
	Seed                   int64
	MineralRegionalization float64
	X                      float64
	Z                      float64
	Land                   bool
	WaterKind              string
	Terrain                string
	Biome                  string
}

type Score struct {
	MiningSuitability        float64            `json:"miningSuitability"`
	SurfaceMiningSuitability float64            `json:"surfaceMiningSuitability"`
	ShallowMiningSuitability float64            `json:"shallowMiningSuitability"`
	DeepMiningSuitability    float64            `json:"deepMiningSuitability"`
	IndustrialValue          float64            `json:"industrialValue"`
	ResourceDiversity        float64            `json:"resourceDiversity"`
	DominantResource         string             `json:"dominantResource"`
	MiningClass              string             `json:"miningClass"`
	ResourceProfile          map[string]float64 `json:"resourceProfile"`
}

func Evaluate(in Inputs) Score {
	if !in.Land || normalize(in.WaterKind) != "none" {
		return Score{
			DominantResource: "none",
			MiningClass:      "none",
			ResourceProfile:  zeroProfile(),
		}
	}

	surface := normalizedProfile(in.Terrain, in.Biome, 80)
	shallow := normalizedProfile(in.Terrain, in.Biome, 28)
	deep := normalizedProfile(in.Terrain, in.Biome, -40)

	composite := make([]float64, len(Minerals))
	for i := range composite {
		composite[i] = 0.30*surface[i] + 0.40*shallow[i] + 0.30*deep[i]
	}

	industrialSurface := industrial(surface)
	industrialShallow := industrial(shallow)
	industrialDeep := industrial(deep)
	industrialValue := industrial(composite)
	diversity := resourceDiversity(composite)
	province := provinceSignal(
		in.Seed,
		in.MineralRegionalization,
		in.X,
		in.Z,
	)
	exposure := terrainExposure(in.Terrain)
	access := terrainAccess(in.Terrain)
	criticalDeep := clamp01(1.55 * deepStrategicValue(deep))

	surfaceScore := clamp01(
		0.55*exposure +
			0.30*industrialSurface +
			0.15*province,
	)
	shallowScore := clamp01(
		0.40*industrialShallow +
			0.25*diversity +
			0.20*access +
			0.15*province,
	)
	deepScore := clamp01(
		0.48*criticalDeep +
			0.22*industrialDeep +
			0.20*diversity +
			0.10*province,
	)

	maxLayer := math.Max(surfaceScore, math.Max(shallowScore, deepScore))
	mining := clamp01(
		0.60*maxLayer +
			0.25*industrialValue +
			0.15*diversity,
	)

	profile := make(map[string]float64, len(Minerals))
	dominant := "none"
	dominantValue := -1.0
	for i, mineral := range Minerals {
		value := clamp01(composite[i])
		profile[mineral] = value
		if value > dominantValue {
			dominant = mineral
			dominantValue = value
		}
	}

	return Score{
		MiningSuitability:        mining,
		SurfaceMiningSuitability: surfaceScore,
		ShallowMiningSuitability: shallowScore,
		DeepMiningSuitability:    deepScore,
		IndustrialValue:          clamp01(industrialValue),
		ResourceDiversity:        clamp01(diversity),
		DominantResource:         dominant,
		MiningClass: miningClass(
			mining,
			surfaceScore,
			shallowScore,
			deepScore,
		),
		ResourceProfile: profile,
	}
}

func Affinity(mineral, terrain, biome string, y float64) float64 {
	kind := strings.ToUpper(strings.TrimSpace(mineral))
	minY, maxY, weight, ok := mineralRange(kind)
	if !ok || y < minY || y > maxY {
		return 0
	}

	mountains := terrain == "Mountain"
	foothills := terrain == "Foothills" || terrain == "Hills"
	sedimentary := terrain == "Plain" || terrain == "Rolling Plain" ||
		terrain == "Floodplain" || terrain == "Alluvial Plain" ||
		terrain == "Alluvial Fan"
	arid := strings.Contains(biome, "Desert") || biome == "Savanna"
	highlands := mountains || foothills || terrain == "Plateau"

	multiplier := 0.0
	switch kind {
	case "COAL":
		if sedimentary {
			multiplier = 2.1
		} else {
			multiplier = 0.80
		}
	case "IRON":
		if mountains {
			multiplier = 2.2
		} else if foothills {
			multiplier = 1.6
		} else {
			multiplier = 1.0
		}
	case "COPPER":
		if highlands {
			multiplier = 2.0
		} else {
			multiplier = 0.9
		}
	case "GOLD":
		if terrain == "Plateau" || terrain == "Canyon" || arid {
			multiplier = 2.1
		} else {
			multiplier = 0.75
		}
	case "LAPIS":
		if sedimentary {
			multiplier = 1.5
		} else {
			multiplier = 1.0
		}
	case "REDSTONE":
		if highlands {
			multiplier = 1.25
		} else {
			multiplier = 1.0
		}
	case "DIAMOND":
		multiplier = 1.0
	case "EMERALD":
		if mountains {
			multiplier = 2.4
		} else if foothills {
			multiplier = 1.4
		} else {
			multiplier = 0.08
		}
	}

	vertical := 1.0
	switch kind {
	case "COAL":
		if y < 0 {
			vertical = 0.10
		} else if y < 25 {
			vertical = 0.7
		} else {
			vertical = 1.3
		}
	case "DIAMOND":
		if y > 0 {
			vertical = 0.18
		} else {
			vertical = 0.5 + (-y / 63.0)
		}
	case "REDSTONE":
		if y > 0 {
			vertical = 0.3
		} else {
			vertical = 1.0 + (-y / 100.0)
		}
	case "EMERALD":
		if y < 0 {
			vertical = 0.08
		} else if y > 55 {
			vertical = 1.2
		} else {
			vertical = 0.6
		}
	}
	return weight * multiplier * vertical
}

func normalizedProfile(terrain, biome string, y float64) []float64 {
	out := make([]float64, len(Minerals))
	total := 0.0
	for i, mineral := range Minerals {
		out[i] = Affinity(mineral, terrain, biome, y)
		total += out[i]
	}
	if total <= 0 {
		return out
	}
	for i := range out {
		out[i] /= total
	}
	return out
}

func industrial(profile []float64) float64 {
	weights := [...]float64{0.85, 1.0, 0.85, 0.35, 0.20, 0.45, 0.15, 0.15}
	total := 0.0
	for i := range profile {
		total += profile[i] * weights[i]
	}
	return clamp01(total)
}

func deepStrategicValue(profile []float64) float64 {
	weights := [...]float64{0.10, 0.40, 0.15, 0.60, 0.30, 0.75, 1.0, 0.25}
	total := 0.0
	for i := range profile {
		total += profile[i] * weights[i]
	}
	return clamp01(total)
}

func resourceDiversity(profile []float64) float64 {
	entropy := 0.0
	for _, p := range profile {
		if p > 0 {
			entropy -= p * math.Log(p)
		}
	}
	return clamp01(entropy / math.Log(float64(len(Minerals))))
}

func terrainExposure(terrain string) float64 {
	switch terrain {
	case "Mountain":
		return 1.0
	case "Canyon":
		return 0.92
	case "Foothills":
		return 0.85
	case "Hills":
		return 0.75
	case "Plateau":
		return 0.72
	case "Rolling Plain":
		return 0.40
	case "Plain":
		return 0.30
	case "Alluvial Fan":
		return 0.28
	case "Floodplain", "Alluvial Plain":
		return 0.18
	default:
		return 0.25
	}
}

func terrainAccess(terrain string) float64 {
	switch terrain {
	case "Plain", "Rolling Plain", "Plateau":
		return 0.90
	case "Hills", "Foothills":
		return 0.72
	case "Alluvial Fan":
		return 0.68
	case "Mountain", "Canyon":
		return 0.48
	case "Floodplain", "Alluvial Plain":
		return 0.55
	default:
		return 0.60
	}
}

func provinceSignal(seed int64, regionalization, x, z float64) float64 {
	if !finite(regionalization) || regionalization <= 0 {
		regionalization = 1
	}
	if math.Abs(regionalization-1) <= 1e-12 {
		return 0.5
	}
	strength := ProvinceStrength(seed, regionalization, x, z)
	return clamp01(0.5 + 0.5*(regionalization-1)*(strength-0.5))
}

func ProvinceStrength(seed int64, regionalization, x, z float64) float64 {
	if !finite(regionalization) || regionalization <= 0 {
		regionalization = 1
	}
	scale := 1536.0 * math.Sqrt(math.Max(0.25, regionalization))
	cellX := int64(math.Floor(x / scale))
	cellZ := int64(math.Floor(z / scale))
	h := uint64(seed) ^ 0x4F524550524F5631
	h ^= uint64(cellX) * 0x9E3779B185EBCA87
	h ^= uint64(cellZ) * 0xC2B2AE3D27D4EB4F
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	h ^= h >> 31
	return float64(h>>11) / float64(uint64(1)<<53)
}

func miningClass(overall, surface, shallow, deep float64) string {
	if overall < 0.38 {
		return "low_potential"
	}
	if deep >= 0.74 && deep >= shallow-0.02 && deep >= surface+0.02 {
		return "deep_district"
	}
	values := []float64{surface, shallow, deep}
	labels := []string{"surface_district", "shallow_district", "deep_district"}
	best, second := 0, 1
	if values[second] > values[best] {
		best, second = second, best
	}
	for i := 2; i < len(values); i++ {
		if values[i] > values[best] {
			second = best
			best = i
		} else if values[i] > values[second] {
			second = i
		}
	}
	if values[best]-values[second] <= 0.04 {
		return "mixed_district"
	}
	return labels[best]
}

func mineralRange(kind string) (float64, float64, float64, bool) {
	switch kind {
	case "COAL":
		return -63, 215, 7.5, true
	case "IRON":
		return -60, 319, 9.0, true
	case "COPPER":
		return -20, 150, 6.0, true
	case "GOLD":
		return -63, 60, 3.2, true
	case "LAPIS":
		return -63, 80, 2.7, true
	case "REDSTONE":
		return -63, 24, 4.2, true
	case "DIAMOND":
		return -63, 24, 1.0, true
	case "EMERALD":
		return -63, 319, 1.2, true
	default:
		return 0, 0, 0, false
	}
}

func zeroProfile() map[string]float64 {
	out := make(map[string]float64, len(Minerals))
	for _, mineral := range Minerals {
		out[mineral] = 0
	}
	return out
}

func normalize(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func clamp01(v float64) float64 {
	if !finite(v) || v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
