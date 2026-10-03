package mcprofile

import (
	"errors"
	"math"

	"geoworld/internal/ecology"
)

func Generate(input Input) (*Model, error) {
	if input.Width < 2 || input.Height < 2 {
		return nil, errors.New("mcprofile: grid must be at least 2x2")
	}
	n := input.Width * input.Height
	if len(input.LandMask) != n || len(input.EcoRegion) != n {
		return nil, errors.New("mcprofile: discrete field size does not match dimensions")
	}

	fields := []struct {
		name string
		data []float64
	}{
		{"region confidence", input.RegionConfidence},
		{"tree cover", input.TreeCoverPotential},
		{"grass cover", input.GrassCoverPotential},
		{"shrub cover", input.ShrubCoverPotential},
		{"bare ground", input.BareGroundPotential},
		{"snow cover", input.SnowCoverPotential},
		{"soil moisture", input.SoilMoisture},
		{"wetland ecology", input.WetlandEcology},
		{"alpine ecology", input.AlpineEcology},
		{"rolling plain", input.RollingPlainWeight},
		{"hill", input.HillWeight},
		{"foothill", input.FoothillWeight},
		{"mountain", input.MountainWeight},
		{"plateau", input.PlateauWeight},
		{"floodplain", input.FloodplainWeight},
		{"alluvial plain", input.AlluvialPlainWeight},
		{"alluvial fan", input.AlluvialFanWeight},
		{"canyon", input.CanyonWeight},
	}
	for _, f := range fields {
		if len(f.data) != n {
			return nil, errors.New("mcprofile: " + f.name + " size does not match dimensions")
		}
	}

	m := &Model{
		Width:             input.Width,
		Height:            input.Height,
		LandMask:          append([]bool(nil), input.LandMask...),
		BiomeFamily:       make([]BiomeFamily, n),
		TerrainVariant:    make([]TerrainVariant, n),
		SurfaceProfile:    make([]SurfaceProfile, n),
		VegetationProfile: make([]VegetationProfile, n),
		ProfileConfidence: make([]float64, n),
	}

	for i := 0; i < n; i++ {
		region := ecology.Region(input.EcoRegion[i])
		m.BiomeFamily[i] = familyFromEcoRegion(region)
		m.TerrainVariant[i] = terrainVariant(i, input)
		m.SurfaceProfile[i] = surfaceProfile(i, region, input)
		m.VegetationProfile[i] = vegetationProfile(i, region, input)
		m.ProfileConfidence[i] = profileConfidence(i, input, m)
	}
	return m, nil
}

func familyFromEcoRegion(r ecology.Region) BiomeFamily {
	switch r {
	case ecology.RegionWarmOcean:
		return BiomeWarmOcean
	case ecology.RegionTemperateOcean:
		return BiomeTemperateOcean
	case ecology.RegionColdOcean:
		return BiomeColdOcean
	case ecology.RegionFrozenOcean:
		return BiomeFrozenOcean
	case ecology.RegionIceSheet:
		return BiomeIceSheet
	case ecology.RegionTundra:
		return BiomeTundra
	case ecology.RegionAlpine:
		return BiomeAlpine
	case ecology.RegionBorealForest:
		return BiomeBorealForest
	case ecology.RegionTemperateForest:
		return BiomeTemperateForest
	case ecology.RegionTemperateGrassland:
		return BiomeTemperateGrassland
	case ecology.RegionTemperateDesert:
		return BiomeTemperateDesert
	case ecology.RegionMediterraneanShrubland:
		return BiomeMediterraneanShrubland
	case ecology.RegionSubtropicalForest:
		return BiomeSubtropicalForest
	case ecology.RegionHotDesert:
		return BiomeHotDesert
	case ecology.RegionSavanna:
		return BiomeSavanna
	case ecology.RegionTropicalMonsoonForest:
		return BiomeTropicalMonsoonForest
	case ecology.RegionTropicalRainforest:
		return BiomeTropicalRainforest
	case ecology.RegionWetland:
		return BiomeWetland
	default:
		return BiomeUnknown
	}
}

func terrainVariant(i int, in Input) TerrainVariant {
	if !in.LandMask[i] {
		return TerrainOcean
	}

	// Fluvial/depositional forms are overlays already derived from hydrology.
	// Once they cross their meaningful threshold they should not be erased by
	// a stronger underlying rolling-plain weight.
	switch {
	case in.CanyonWeight[i] >= 0.30:
		return TerrainCanyon
	case in.AlluvialFanWeight[i] >= 0.20:
		return TerrainAlluvialFan
	case in.FloodplainWeight[i] >= 0.35:
		return TerrainFloodplain
	case in.AlluvialPlainWeight[i] >= 0.30:
		return TerrainAlluvialPlain
	}

	candidates := []struct {
		variant TerrainVariant
		score   float64
	}{
		{TerrainMountain, in.MountainWeight[i]},
		{TerrainFoothills, in.FoothillWeight[i]},
		{TerrainPlateau, in.PlateauWeight[i]},
		{TerrainHills, in.HillWeight[i]},
		{TerrainRollingPlain, in.RollingPlainWeight[i]},
	}

	best := TerrainPlain
	bestScore := 0.0
	for _, c := range candidates {
		if c.score > bestScore {
			best = c.variant
			bestScore = c.score
		}
	}
	if bestScore < 0.18 {
		return TerrainPlain
	}
	return best
}

func surfaceProfile(i int, region ecology.Region, in Input) SurfaceProfile {
	if !in.LandMask[i] {
		switch region {
		case ecology.RegionFrozenOcean:
			return SurfaceFrozenOcean
		case ecology.RegionColdOcean:
			return SurfaceOceanGravel
		default:
			return SurfaceOceanSand
		}
	}

	if in.SnowCoverPotential[i] > 0.78 {
		if in.MountainWeight[i]+in.AlpineEcology[i] > 0.75 {
			return SurfaceAlpineSnow
		}
		return SurfaceIceSnow
	}

	if in.WetlandEcology[i] > 0.62 {
		if in.SoilMoisture[i] > 0.78 {
			return SurfacePeat
		}
		return SurfaceWetMud
	}

	if in.FloodplainWeight[i] > 0.42 ||
		in.AlluvialPlainWeight[i] > 0.48 ||
		in.AlluvialFanWeight[i] > 0.42 {
		return SurfaceAlluvialSilt
	}

	switch region {
	case ecology.RegionIceSheet:
		return SurfaceIceSnow
	case ecology.RegionTundra:
		return SurfaceTundraSoil
	case ecology.RegionAlpine:
		if in.SnowCoverPotential[i] > 0.48 {
			return SurfaceAlpineSnow
		}
		return SurfaceAlpineRock
	case ecology.RegionBorealForest:
		if in.TreeCoverPotential[i] > 0.55 {
			return SurfacePodzol
		}
		return SurfaceForestLoam
	case ecology.RegionTemperateForest, ecology.RegionSubtropicalForest,
		ecology.RegionTropicalMonsoonForest, ecology.RegionTropicalRainforest:
		return SurfaceForestLoam
	case ecology.RegionTemperateGrassland:
		return SurfaceGrassLoam
	case ecology.RegionMediterraneanShrubland, ecology.RegionSavanna:
		return SurfaceDryGrass
	case ecology.RegionTemperateDesert:
		if in.BareGroundPotential[i] > 0.72 {
			return SurfaceDesertSand
		}
		return SurfaceShrubSoil
	case ecology.RegionHotDesert:
		return SurfaceDesertSand
	case ecology.RegionWetland:
		return SurfaceWetMud
	default:
		return SurfaceGrassLoam
	}
}

func vegetationProfile(i int, region ecology.Region, in Input) VegetationProfile {
	if !in.LandMask[i] {
		return VegetationNone
	}

	switch region {
	case ecology.RegionIceSheet:
		return VegetationNone
	case ecology.RegionTundra:
		return VegetationTundraSparse
	case ecology.RegionAlpine:
		return VegetationAlpineSparse
	case ecology.RegionBorealForest:
		return VegetationConiferForest
	case ecology.RegionTemperateForest:
		if in.TreeCoverPotential[i] >= 0.48 {
			return VegetationMixedForest
		}
		return VegetationGrassland
	case ecology.RegionSubtropicalForest:
		return VegetationBroadleafForest
	case ecology.RegionTropicalRainforest:
		return VegetationDenseRainforest
	case ecology.RegionTropicalMonsoonForest:
		return VegetationMonsoonForest
	case ecology.RegionSavanna:
		return VegetationSavanna
	case ecology.RegionTemperateGrassland:
		if in.ShrubCoverPotential[i] > in.GrassCoverPotential[i] {
			return VegetationShrubland
		}
		return VegetationGrassland
	case ecology.RegionMediterraneanShrubland:
		return VegetationShrubland
	case ecology.RegionTemperateDesert, ecology.RegionHotDesert:
		return VegetationDesertSparse
	case ecology.RegionWetland:
		return VegetationWetland
	default:
		if in.WetlandEcology[i] > 0.65 {
			return VegetationWetland
		}
		if in.TreeCoverPotential[i] > 0.55 {
			return VegetationMixedForest
		}
		if in.GrassCoverPotential[i] > 0.40 {
			return VegetationGrassland
		}
		return VegetationShrubland
	}
}

func profileConfidence(i int, in Input, m *Model) float64 {
	base := clamp(in.RegionConfidence[i], 0, 1)

	terrainStrength := 0.0
	if in.LandMask[i] {
		terrainStrength = maxSlice(
			in.RollingPlainWeight[i],
			in.HillWeight[i],
			in.FoothillWeight[i],
			in.MountainWeight[i],
			in.PlateauWeight[i],
			in.FloodplainWeight[i],
			in.AlluvialPlainWeight[i],
			in.AlluvialFanWeight[i],
			in.CanyonWeight[i],
		)
	} else {
		terrainStrength = 1
	}

	surfaceSignal := maxSlice(
		in.SnowCoverPotential[i],
		in.SoilMoisture[i],
		in.BareGroundPotential[i],
		in.TreeCoverPotential[i],
		in.GrassCoverPotential[i],
	)

	conf := 0.62*base + 0.23*clamp(terrainStrength, 0, 1) + 0.15*clamp(surfaceSignal, 0, 1)

	// Unknown output is always low-confidence so audits catch it immediately.
	if m.BiomeFamily[i] == BiomeUnknown ||
		m.TerrainVariant[i] == TerrainUnknown ||
		m.SurfaceProfile[i] == SurfaceUnknown ||
		m.VegetationProfile[i] == VegetationUnknown {
		conf *= 0.20
	}
	return clamp(conf, 0, 1)
}

func maxSlice(v ...float64) float64 {
	m := math.Inf(-1)
	for _, x := range v {
		if x > m {
			m = x
		}
	}
	if math.IsInf(m, -1) {
		return 0
	}
	return m
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
