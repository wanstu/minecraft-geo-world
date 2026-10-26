package generator

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"geoworld/internal/world"
)

const (
	CurrentWorldProfileVersion    = "geoworld-profile-v1.1"
	CurrentGenerationRulesVersion = "geoworld-rules-v1.1.9"
	CurrentWorldModelVersion      = "geoworld-model-v1"
)

// WorldProfile is the immutable public identity of a generated natural world.
// Presets only select ParameterSnapshot values; they never select another
// generator implementation.
type WorldProfile struct {
	WorldProfileVersion    string
	GenerationRulesVersion string
	WorldModelVersion      string
	Preset                 string
	Parameters             ParameterSnapshot
}

// ParameterSnapshot contains player-facing semantic controls. Values are
// dimensionless multipliers where 1 is the Standard baseline unless noted.
// Keeping these semantics separate from low-level stage configs lets algorithms
// evolve without exposing implementation knobs to the creation UI.
type ParameterSnapshot struct {
	MainlandScale          float64
	RegionScale            float64
	TransitionWidth        float64
	MountainStrength       float64
	PlainContinuity        float64
	PlateauScale           float64
	CliffRarity            float64
	RareMacroTerrain       float64
	RiverDensity           float64
	MainRiverScale         float64
	LakeAbundance          float64
	WetlandAbundance       float64
	ClimateRegionScale     float64
	ClimateTransitionWidth float64
	RainShadowStrength     float64
	BiomeContinuity        float64
	CaveRichness           float64
	LargeCaveRarity        float64
	MineralRegionalization float64
	SpawnPreference        float64
	StructureInfluence     float64
	OuterIslandAbundance   float64
	PolarTransitionWidth   float64
	OceanClimateStrength   float64
	PrevailingWindStrength float64
}

func StandardWorldProfile() WorldProfile {
	return profileForPreset("standard")
}

// ParseParameterSnapshotCSV decodes the stable 25-value transport used by the
// Minecraft creation UI when invoking the native engine.
func ParseParameterSnapshotCSV(value string) (ParameterSnapshot, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 25 {
		return ParameterSnapshot{}, errors.New("generator: parameter snapshot must contain 25 values")
	}
	values := make([]float64, len(parts))
	for i, part := range parts {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return ParameterSnapshot{}, errors.New("generator: invalid parameter snapshot value")
		}
		values[i] = parsed
	}
	p := ParameterSnapshot{
		MainlandScale: values[0], RegionScale: values[1], TransitionWidth: values[2],
		MountainStrength: values[3], PlainContinuity: values[4], PlateauScale: values[5],
		CliffRarity: values[6], RareMacroTerrain: values[7],
		RiverDensity: values[8], MainRiverScale: values[9],
		LakeAbundance: values[10], WetlandAbundance: values[11],
		ClimateRegionScale: values[12], ClimateTransitionWidth: values[13],
		RainShadowStrength: values[14], BiomeContinuity: values[15],
		CaveRichness: values[16], LargeCaveRarity: values[17],
		MineralRegionalization: values[18], SpawnPreference: values[19],
		StructureInfluence: values[20], OuterIslandAbundance: values[21],
		PolarTransitionWidth: values[22], OceanClimateStrength: values[23],
		PrevailingWindStrength: values[24],
	}
	profile := WorldProfile{
		WorldProfileVersion:    CurrentWorldProfileVersion,
		GenerationRulesVersion: CurrentGenerationRulesVersion,
		WorldModelVersion:      CurrentWorldModelVersion,
		Preset:                 "custom",
		Parameters:             p,
	}
	normalized, err := normalizeWorldProfile(profile)
	if err != nil {
		return ParameterSnapshot{}, err
	}
	return normalized.Parameters, nil
}

func WorldProfileForPreset(id string) WorldProfile {
	return profileForPreset(id)
}

func profileForPreset(id string) WorldProfile {
	p := standardParameters()
	switch id {
	case "", "standard":
		id = "standard"
	case "natural":
		p.MainlandScale = 1.15
		p.RegionScale = 1.20
		p.TransitionWidth = 1.20
		p.MountainStrength = 0.92
		p.PlainContinuity = 0.92
		p.CliffRarity = 1.28
		p.RiverDensity = 0.92
		p.ClimateRegionScale = 1.20
		p.ClimateTransitionWidth = 1.20
		p.BiomeContinuity = 1.18
		p.SpawnPreference = 0.85
		p.StructureInfluence = 0.75
		p.OuterIslandAbundance = 0.75
	case "civilization":
		p.MainlandScale = 1.08
		p.RegionScale = 1.05
		p.MountainStrength = 0.86
		p.PlainContinuity = 1.34
		p.PlateauScale = 1.08
		p.CliffRarity = 1.45
		p.RiverDensity = 0.95
		p.MainRiverScale = 1.08
		p.LakeAbundance = 0.90
		p.WetlandAbundance = 0.82
		p.SpawnPreference = 1.30
		p.StructureInfluence = 1.20
	case "expansive":
		p.MainlandScale = 1.35
		p.RegionScale = 1.40
		p.TransitionWidth = 1.35
		p.ClimateRegionScale = 1.35
		p.ClimateTransitionWidth = 1.35
		p.BiomeContinuity = 1.25
		p.OuterIslandAbundance = 0.75
	case "epic":
		p.MainlandScale = 1.15
		p.MountainStrength = 1.30
		p.PlateauScale = 1.18
		p.CliffRarity = 0.78
		p.RareMacroTerrain = 1.45
		p.RiverDensity = 1.10
		p.MainRiverScale = 1.35
		p.LakeAbundance = 1.20
		p.RainShadowStrength = 1.15
	case "near_vanilla":
		p.MainlandScale = 0.88
		p.RegionScale = 0.80
		p.TransitionWidth = 0.82
		p.MountainStrength = 0.92
		p.PlainContinuity = 0.82
		p.RiverDensity = 0.88
		p.MainRiverScale = 0.82
		p.LakeAbundance = 0.90
		p.ClimateRegionScale = 0.82
		p.ClimateTransitionWidth = 0.82
		p.BiomeContinuity = 0.80
		p.CaveRichness = 1.35
		p.StructureInfluence = 1.35
	default:
		// Custom/unknown profile IDs preserve Standard values. Callers that use
		// "custom" replace Parameters explicitly before generation.
	}
	return WorldProfile{
		WorldProfileVersion:    CurrentWorldProfileVersion,
		GenerationRulesVersion: CurrentGenerationRulesVersion,
		WorldModelVersion:      CurrentWorldModelVersion,
		Preset:                 id,
		Parameters:             p,
	}
}

func standardParameters() ParameterSnapshot {
	return ParameterSnapshot{
		MainlandScale: 1, RegionScale: 1, TransitionWidth: 1,
		MountainStrength: 1, PlainContinuity: 1, PlateauScale: 1,
		CliffRarity: 1, RareMacroTerrain: 1,
		RiverDensity: 1, MainRiverScale: 1, LakeAbundance: 1, WetlandAbundance: 1,
		ClimateRegionScale: 1, ClimateTransitionWidth: 1,
		RainShadowStrength: 1, BiomeContinuity: 1,
		CaveRichness: 1, LargeCaveRarity: 1, MineralRegionalization: 1,
		SpawnPreference: 1, StructureInfluence: 1,
		OuterIslandAbundance: 1, PolarTransitionWidth: 1,
		OceanClimateStrength: 1, PrevailingWindStrength: 1,
	}
}

func normalizeWorldProfile(profile WorldProfile) (WorldProfile, error) {
	if profile.WorldProfileVersion == "" &&
		profile.GenerationRulesVersion == "" &&
		profile.WorldModelVersion == "" {
		profile = StandardWorldProfile()
	}
	if profile.WorldProfileVersion == "" ||
		profile.GenerationRulesVersion == "" ||
		profile.WorldModelVersion == "" {
		return WorldProfile{}, errors.New("generator: incomplete world profile identity")
	}
	if profile.Preset == "" {
		profile.Preset = "custom"
	}
	p := profile.Parameters
	defaults := standardParameters()
	fill := func(value, fallback float64) float64 {
		if value == 0 {
			return fallback
		}
		return value
	}
	p.MainlandScale = fill(p.MainlandScale, defaults.MainlandScale)
	p.RegionScale = fill(p.RegionScale, defaults.RegionScale)
	p.TransitionWidth = fill(p.TransitionWidth, defaults.TransitionWidth)
	p.MountainStrength = fill(p.MountainStrength, defaults.MountainStrength)
	p.PlainContinuity = fill(p.PlainContinuity, defaults.PlainContinuity)
	p.PlateauScale = fill(p.PlateauScale, defaults.PlateauScale)
	p.CliffRarity = fill(p.CliffRarity, defaults.CliffRarity)
	p.RareMacroTerrain = fill(p.RareMacroTerrain, defaults.RareMacroTerrain)
	p.RiverDensity = fill(p.RiverDensity, defaults.RiverDensity)
	p.MainRiverScale = fill(p.MainRiverScale, defaults.MainRiverScale)
	p.LakeAbundance = fill(p.LakeAbundance, defaults.LakeAbundance)
	p.WetlandAbundance = fill(p.WetlandAbundance, defaults.WetlandAbundance)
	p.ClimateRegionScale = fill(p.ClimateRegionScale, defaults.ClimateRegionScale)
	p.ClimateTransitionWidth = fill(p.ClimateTransitionWidth, defaults.ClimateTransitionWidth)
	p.RainShadowStrength = fill(p.RainShadowStrength, defaults.RainShadowStrength)
	p.BiomeContinuity = fill(p.BiomeContinuity, defaults.BiomeContinuity)
	p.CaveRichness = fill(p.CaveRichness, defaults.CaveRichness)
	p.LargeCaveRarity = fill(p.LargeCaveRarity, defaults.LargeCaveRarity)
	p.MineralRegionalization = fill(p.MineralRegionalization, defaults.MineralRegionalization)
	// Spawn/structure influence intentionally allow 0: zero means GeoWorld
	// does not deliberately bias that compatibility decision.
	p.OuterIslandAbundance = fill(p.OuterIslandAbundance, defaults.OuterIslandAbundance)
	p.PolarTransitionWidth = fill(p.PolarTransitionWidth, defaults.PolarTransitionWidth)
	p.OceanClimateStrength = fill(p.OceanClimateStrength, defaults.OceanClimateStrength)
	p.PrevailingWindStrength = fill(p.PrevailingWindStrength, defaults.PrevailingWindStrength)

	values := []float64{
		p.MainlandScale, p.RegionScale, p.TransitionWidth,
		p.MountainStrength, p.PlainContinuity, p.PlateauScale, p.CliffRarity,
		p.RareMacroTerrain, p.RiverDensity, p.MainRiverScale, p.LakeAbundance,
		p.WetlandAbundance, p.ClimateRegionScale, p.ClimateTransitionWidth,
		p.RainShadowStrength, p.BiomeContinuity, p.CaveRichness,
		p.LargeCaveRarity, p.MineralRegionalization,
		p.OuterIslandAbundance, p.PolarTransitionWidth,
		p.OceanClimateStrength, p.PrevailingWindStrength,
	}
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0.25 || value > 2.5 {
			return WorldProfile{}, errors.New("generator: world profile parameter outside 0.25..2.5")
		}
	}
	for _, value := range []float64{p.SpawnPreference, p.StructureInfluence} {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 2.5 {
			return WorldProfile{}, errors.New("generator: compatibility influence parameter outside 0..2.5")
		}
	}
	profile.Parameters = p
	return profile, nil
}

func usesV117Features(version string) bool {
	return version == "geoworld-rules-v1.1.7" ||
		version == "geoworld-rules-v1.1.8" ||
		version == "geoworld-rules-v1.1.9"
}

func usesV119Features(version string) bool {
	return version == "geoworld-rules-v1.1.9"
}

func applyWorldProfile(cfg *Config, geometry world.GridGeometry) {
	p := cfg.WorldProfile.Parameters

	// Standard (all multipliers == 1) intentionally resolves to the exact
	// historical defaults used by rc.1, preserving locked fingerprints.
	if cfg.Terrain.PlainAmplitude == 0 {
		cfg.Terrain.PlainAmplitude = 0.0035 / p.PlainContinuity
	}
	if cfg.Terrain.HillAmplitude == 0 {
		cfg.Terrain.HillAmplitude = 0.025
	}
	if cfg.Terrain.FoothillAmplitude == 0 {
		cfg.Terrain.FoothillAmplitude = 0.040
	}
	if cfg.Terrain.MountainAmplitude == 0 {
		cfg.Terrain.MountainAmplitude = 0.075 * p.MountainStrength
	}
	if usesV117Features(cfg.WorldProfile.GenerationRulesVersion) &&
		cfg.Terrain.MountainStrengthScale == 0 {
		cfg.Terrain.MountainStrengthScale = p.MountainStrength
	}
	if cfg.Terrain.PlateauAmplitude == 0 {
		cfg.Terrain.PlateauAmplitude = 0.010 * p.PlateauScale
	}
	if cfg.Terrain.ValleyDepth == 0 {
		cfg.Terrain.ValleyDepth = 0.007
	}
	if cfg.Terrain.DepositionLift == 0 {
		cfg.Terrain.DepositionLift = 0.0035
	}
	if cfg.Terrain.AlluvialFanLift == 0 {
		cfg.Terrain.AlluvialFanLift = 0.0045
	}
	if cfg.Terrain.CanyonExtraDepth == 0 {
		cfg.Terrain.CanyonExtraDepth = 0.006 * p.RareMacroTerrain
	}
	if cfg.Terrain.CliffFrequency == 0 {
		cfg.Terrain.CliffFrequency = 1.0 / p.CliffRarity
	}
	if usesV117Features(cfg.WorldProfile.GenerationRulesVersion) &&
		cfg.Terrain.RareMacroStrength == 0 {
		cfg.Terrain.RareMacroStrength = p.RareMacroTerrain
	}

	if cfg.Hydrology.RiverQuantile == 0 {
		cfg.Hydrology.RiverQuantile = clampProfile(0.95-(p.RiverDensity-1)*0.04, 0.86, 0.975)
	}
	if cfg.Hydrology.MinRiverAccumulation == 0 {
		cfg.Hydrology.MinRiverAccumulation = 5000 / p.RiverDensity
	}
	if cfg.Hydrology.LakePotentialThreshold == 0 {
		cfg.Hydrology.LakePotentialThreshold = clampProfile(0.35/p.LakeAbundance, 0.16, 0.70)
	}
	if cfg.Hydrology.WetlandAccumulationScale == 0 {
		cfg.Hydrology.WetlandAccumulationScale = 12000 / p.WetlandAbundance
	}
	if usesV119Features(cfg.WorldProfile.GenerationRulesVersion) {
		cfg.Ecology.EnsureWetlandRegion = true
	}

	if cfg.Waterbed.MinRiverHalfWidthBlocks == 0 {
		cfg.Waterbed.MinRiverHalfWidthBlocks = 4 * math.Sqrt(p.MainRiverScale)
	}
	if cfg.Waterbed.MaxRiverHalfWidthBlocks == 0 {
		cfg.Waterbed.MaxRiverHalfWidthBlocks = 18 * p.MainRiverScale
	}
	if cfg.Waterbed.RiverProfileStepBlocks == 0 {
		cfg.Waterbed.RiverProfileStepBlocks = 1
	}
	if cfg.Waterbed.RiverGeometryVersion == 0 {
		switch cfg.WorldProfile.GenerationRulesVersion {
		case "geoworld-rules-v1.1.8", "geoworld-rules-v1.1.9":
			cfg.Waterbed.RiverGeometryVersion = 3
		case "geoworld-rules-v1.1.7":
			cfg.Waterbed.RiverGeometryVersion = 2
		case "geoworld-rules-v1.1.5", "geoworld-rules-v1.1.6":
			cfg.Waterbed.RiverGeometryVersion = 1
		}
	}
	if cfg.Waterbed.LakeGeometryVersion == 0 &&
		usesV117Features(cfg.WorldProfile.GenerationRulesVersion) {
		cfg.Waterbed.LakeGeometryVersion = 1
	}
	// v1.1.4 deepens the river channel and its water column together. Old
	// rule versions keep the exact v1.1.3 depth defaults when regenerated from
	// a locked World Profile.
	modernRiverDepth :=
		cfg.WorldProfile.GenerationRulesVersion == "geoworld-rules-v1.1.4" ||
			cfg.WorldProfile.GenerationRulesVersion == "geoworld-rules-v1.1.5" ||
			cfg.WorldProfile.GenerationRulesVersion == "geoworld-rules-v1.1.6" ||
			usesV117Features(cfg.WorldProfile.GenerationRulesVersion)
	if cfg.Waterbed.MinRiverBedDepthNormalized == 0 {
		if modernRiverDepth {
			cfg.Waterbed.MinRiverBedDepthNormalized = 0.006
		} else {
			cfg.Waterbed.MinRiverBedDepthNormalized = 0.004
		}
	}
	if cfg.WorldSample.RiverCenterDepthNormalized == 0 {
		if modernRiverDepth {
			cfg.WorldSample.RiverCenterDepthNormalized = 0.004
		} else {
			cfg.WorldSample.RiverCenterDepthNormalized = 0.0025
		}
	}
	if cfg.Waterbed.MinLakeWaterDepthNormalized == 0 {
		cfg.Waterbed.MinLakeWaterDepthNormalized = 0.002
	}

	if cfg.Climate.OrographicRainStrength == 0 {
		// RainShadowStrength was persisted before v1.1.7 but the climate solver
		// historically used a fixed 0.22 orographic coefficient. Keep locked
		// older rules bit-compatible; v1.1.7 is the first ruleset where this
		// semantic parameter actively scales windward rainout / leeward drying.
		if usesV117Features(cfg.WorldProfile.GenerationRulesVersion) {
			cfg.Climate.OrographicRainStrength = 0.22 * p.RainShadowStrength
		} else {
			cfg.Climate.OrographicRainStrength = 0.22
		}
	}
	if cfg.Climate.CoastalCurrentInfluence == 0 {
		cfg.Climate.CoastalCurrentInfluence = clampProfile(0.75*p.OceanClimateStrength, 0.20, 1)
	}
	if cfg.Climate.CoastalTempDecayBlocks == 0 {
		cfg.Climate.CoastalTempDecayBlocks =
			18 * world.DefaultMacroSpacingBlocks * p.ClimateTransitionWidth
	}
	if cfg.Climate.MoistureIterations == 0 {
		referenceMaxDim := math.Max(geometry.SpanXBlocks, geometry.SpanZBlocks)/world.DefaultMacroSpacingBlocks + 1
		baseIterations := int(referenceMaxDim / 3)
		if baseIterations < 48 {
			baseIterations = 48
		}
		if baseIterations > 192 {
			baseIterations = 192
		}
		iterations := int(math.Round(float64(baseIterations) * math.Sqrt(p.ClimateRegionScale)))
		if iterations < 48 {
			iterations = 48
		}
		if iterations > 192 {
			iterations = 192
		}
		cfg.Climate.MoistureIterations = iterations
	}
	if cfg.Climate.MoistureDiffusion == 0 {
		cfg.Climate.MoistureDiffusion = clampProfile(0.08*math.Sqrt(p.ClimateRegionScale), 0.04, 0.16)
	}
	if cfg.Climate.MonsoonWindStrength == 0 {
		cfg.Climate.MonsoonWindStrength = 0.75 * p.PrevailingWindStrength
	}
	if cfg.Climate.PrevailingWindStrength == 0 {
		cfg.Climate.PrevailingWindStrength = p.PrevailingWindStrength
	}

	minSpan := math.Min(geometry.SpanXBlocks, geometry.SpanZBlocks)
	referenceMinDim := minSpan/world.DefaultMacroSpacingBlocks + 1
	if cfg.Landform.MountainInfluenceBlocks == 0 {
		cfg.Landform.MountainInfluenceBlocks =
			math.Max(2, referenceMinDim*0.025) *
				world.DefaultMacroSpacingBlocks * p.RegionScale
	}
	if cfg.Landform.ValleyRadiusBlocks == 0 {
		referenceCells := int(math.Round(referenceMinDim * 0.01))
		if referenceCells < 1 {
			referenceCells = 1
		}
		cfg.Landform.ValleyRadiusBlocks =
			float64(referenceCells) *
				world.DefaultMacroSpacingBlocks * p.RegionScale
	}
	if cfg.Landform.ValleyDecayBlocks == 0 {
		cfg.Landform.ValleyDecayBlocks =
			math.Max(0.75, referenceMinDim*0.0055) *
				world.DefaultMacroSpacingBlocks * p.TransitionWidth
	}
}

func clampProfile(value, lo, hi float64) float64 {
	if value < lo {
		return lo
	}
	if value > hi {
		return hi
	}
	return value
}
