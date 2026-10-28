package runtimeworld

import (
	"errors"
	"math"

	"geoworld/internal/ecology"
	"geoworld/internal/generator"
	"geoworld/internal/mcprofile"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/sampling"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
	"geoworld/internal/worldsample"
)

// Model extends the finite locked natural model into GeoWorld's infinite
// exterior ocean. Both acceptance tooling and cross-language golden vectors use
// this sampler so the Java Minecraft runtime cannot silently diverge at edges.
type Model struct {
	Geometry               world.GridGeometry
	Seed                   int64
	SeaLevelNormalized     float64
	Legacy                 bool
	GenerationRulesVersion string

	TransitionWidth      float64
	OuterIslandAbundance float64
	PolarTransitionWidth float64

	Finite          *runtimeprofile.Model
	SpecialFeatures []specialfeatures.Feature

	climateTemp    sampling.ScalarGrid
	climatePrecip  sampling.ScalarGrid
	climateMonsoon sampling.ScalarGrid
	climateWindX   sampling.ScalarGrid
	climateWindZ   sampling.ScalarGrid
}

type ClimateFields struct {
	AnnualMeanTempC []float64
	AnnualPrecipMM  []float64
	MonsoonIndex    []float64
	WindX           []float64
	WindZ           []float64
}

func NewGenerated(model *generator.WorldModel) (*Model, error) {
	if model == nil || model.Climate == nil || model.RuntimeProfile == nil {
		return nil, errors.New("runtimeworld: complete generated model is required")
	}
	p := model.Config.WorldProfile.Parameters
	return New(Config{
		Geometry:               model.Geometry,
		Seed:                   model.Config.Seed,
		SeaLevelNormalized:     model.Config.SeaLevelNormalized,
		Legacy:                 model.Config.WorldProfile.WorldProfileVersion == "",
		GenerationRulesVersion: model.Config.WorldProfile.GenerationRulesVersion,
		TransitionWidth:        p.TransitionWidth,
		OuterIslandAbundance:   p.OuterIslandAbundance,
		PolarTransitionWidth:   p.PolarTransitionWidth,
		Finite:                 model.RuntimeProfile,
		SpecialFeatures:        append([]specialfeatures.Feature(nil), model.SpecialFeatures...),
		Climate: ClimateFields{
			AnnualMeanTempC: model.Climate.AdjustedAnnualTempC,
			AnnualPrecipMM:  model.Climate.AnnualPrecipMM,
			MonsoonIndex:    model.Climate.MonsoonIndex,
			WindX:           model.Climate.WindX,
			WindZ:           model.Climate.WindZ,
		},
	})
}

func NewReopened(model *worldmodelio.RuntimeModel) (*Model, error) {
	if model == nil || model.RuntimeProfile == nil {
		return nil, errors.New("runtimeworld: reopened runtime model is required")
	}
	if model.Config.WorldProfile.WorldProfileVersion == "" {
		return &Model{
			Geometry:           model.Geometry,
			Seed:               model.Manifest.Seed,
			SeaLevelNormalized: model.Manifest.SeaLevelNormalized,
			Legacy:             true,
			Finite:             model.RuntimeProfile,
		}, nil
	}
	if model.ClimatePreview == nil {
		return nil, errors.New("runtimeworld: climate preview is required")
	}
	p := model.Config.WorldProfile.Parameters
	return New(Config{
		Geometry:               model.Geometry,
		Seed:                   model.Manifest.Seed,
		SeaLevelNormalized:     model.Manifest.SeaLevelNormalized,
		Legacy:                 model.Config.WorldProfile.WorldProfileVersion == "",
		GenerationRulesVersion: model.Config.WorldProfile.GenerationRulesVersion,
		TransitionWidth:        p.TransitionWidth,
		OuterIslandAbundance:   p.OuterIslandAbundance,
		PolarTransitionWidth:   p.PolarTransitionWidth,
		Finite:                 model.RuntimeProfile,
		SpecialFeatures:        append([]specialfeatures.Feature(nil), model.SpecialFeatures...),
		Climate: ClimateFields{
			AnnualMeanTempC: model.ClimatePreview.AnnualMeanTempC,
			AnnualPrecipMM:  model.ClimatePreview.AnnualPrecipMM,
			MonsoonIndex:    model.ClimatePreview.MonsoonIndex,
			WindX:           model.ClimatePreview.WindX,
			WindZ:           model.ClimatePreview.WindZ,
		},
	})
}

type Config struct {
	Geometry               world.GridGeometry
	Seed                   int64
	SeaLevelNormalized     float64
	Legacy                 bool
	GenerationRulesVersion string
	TransitionWidth        float64
	OuterIslandAbundance   float64
	PolarTransitionWidth   float64
	Finite                 *runtimeprofile.Model
	SpecialFeatures        []specialfeatures.Feature
	Climate                ClimateFields
}

func New(cfg Config) (*Model, error) {
	if cfg.Finite == nil {
		return nil, errors.New("runtimeworld: finite runtime profile is required")
	}
	if err := cfg.Geometry.Validate(); err != nil {
		return nil, err
	}
	n, err := world.NodeCount(cfg.Geometry.Width, cfg.Geometry.Height)
	if err != nil {
		return nil, err
	}
	fields := [][]float64{
		cfg.Climate.AnnualMeanTempC,
		cfg.Climate.AnnualPrecipMM,
		cfg.Climate.MonsoonIndex,
		cfg.Climate.WindX,
		cfg.Climate.WindZ,
	}
	for _, values := range fields {
		if len(values) != n {
			return nil, errors.New("runtimeworld: climate field size does not match geometry")
		}
	}
	if cfg.TransitionWidth <= 0 {
		cfg.TransitionWidth = 1
	}
	if cfg.OuterIslandAbundance <= 0 {
		cfg.OuterIslandAbundance = 1
	}
	if cfg.PolarTransitionWidth <= 0 {
		cfg.PolarTransitionWidth = 1
	}
	if cfg.GenerationRulesVersion == "" && !cfg.Legacy {
		cfg.GenerationRulesVersion = generator.CurrentGenerationRulesVersion
	}
	temp, err := sampling.NewScalarGridWithGeometry(cfg.Geometry, cfg.Climate.AnnualMeanTempC)
	if err != nil {
		return nil, err
	}
	precip, err := sampling.NewScalarGridWithGeometry(cfg.Geometry, cfg.Climate.AnnualPrecipMM)
	if err != nil {
		return nil, err
	}
	monsoon, err := sampling.NewScalarGridWithGeometry(cfg.Geometry, cfg.Climate.MonsoonIndex)
	if err != nil {
		return nil, err
	}
	windX, err := sampling.NewScalarGridWithGeometry(cfg.Geometry, cfg.Climate.WindX)
	if err != nil {
		return nil, err
	}
	windZ, err := sampling.NewScalarGridWithGeometry(cfg.Geometry, cfg.Climate.WindZ)
	if err != nil {
		return nil, err
	}

	return &Model{
		Geometry:               cfg.Geometry,
		Seed:                   cfg.Seed,
		SeaLevelNormalized:     cfg.SeaLevelNormalized,
		Legacy:                 cfg.Legacy,
		GenerationRulesVersion: cfg.GenerationRulesVersion,
		TransitionWidth:        cfg.TransitionWidth,
		OuterIslandAbundance:   cfg.OuterIslandAbundance,
		PolarTransitionWidth:   cfg.PolarTransitionWidth,
		Finite:                 cfg.Finite,
		SpecialFeatures:        append([]specialfeatures.Feature(nil), cfg.SpecialFeatures...),
		climateTemp:            temp,
		climatePrecip:          precip,
		climateMonsoon:         monsoon,
		climateWindX:           windX,
		climateWindZ:           windZ,
	}, nil
}

func (m *Model) Sample(worldX, worldZ float64) runtimeprofile.Sample {
	var out runtimeprofile.Sample
	if m.Legacy || !m.outside(worldX, worldZ) {
		out = m.Finite.Sample(worldX, worldZ)
	} else {
		out = m.sampleOuterOcean(worldX, worldZ)
	}
	if m.GenerationRulesVersion == "geoworld-rules-v1.1.11" && len(m.SpecialFeatures) > 0 {
		out = m.applySpecialFeatures(out, worldX, worldZ)
	}
	return out
}

func (m *Model) applySpecialFeatures(
	base runtimeprofile.Sample,
	x, z float64,
) runtimeprofile.Sample {
	return specialfeatures.Apply(
		m.SpecialFeatures,
		base,
		m.SeaLevelNormalized,
		x,
		z,
	)
}

type ClimateSample struct {
	AnnualMeanTempC float64
	AnnualPrecipMM  float64
	MonsoonIndex    float64
	WindX           float64
	WindZ           float64
}

func (m *Model) SampleClimate(worldX, worldZ float64) ClimateSample {
	x := worldX
	z := worldZ
	if m.outside(worldX, worldZ) {
		x = clamp(worldX, m.minX(), m.maxX())
		z = clamp(worldZ, m.minZ(), m.maxZ())
	}
	polar := 0.0
	if m.outside(worldX, worldZ) {
		polar = m.polarFactor(worldZ)
	}
	temp := m.climateTemp.SampleSmoothstepBilinear(x, z)
	precip := m.climatePrecip.SampleSmoothstepBilinear(x, z)
	monsoon := m.climateMonsoon.SampleSmoothstepBilinear(x, z)
	if polar > 0 {
		temp = lerp(temp, -18.0, polar)
		monsoon *= 1 - 0.75*polar
	}
	return ClimateSample{
		AnnualMeanTempC: temp,
		AnnualPrecipMM:  precip,
		MonsoonIndex:    monsoon,
		WindX:           m.climateWindX.SampleSmoothstepBilinear(x, z),
		WindZ:           m.climateWindZ.SampleSmoothstepBilinear(x, z),
	}
}

func (m *Model) sampleOuterOcean(x, z float64) runtimeprofile.Sample {
	edgeX := clamp(x, m.minX(), m.maxX())
	edgeZ := clamp(z, m.minZ(), m.maxZ())
	edge := m.Finite.Sample(edgeX, edgeZ)
	distance := m.outsideDistance(x, z)
	cellSize := 0.5 * (m.Geometry.SpacingX() + m.Geometry.SpacingZ())
	transition := math.Max(2048, cellSize*24*m.TransitionWidth)
	t := smoothstepUnit(clamp(distance/transition, 0, 1))
	sea := m.SeaLevelNormalized
	deep := math.Max(0.08, sea-0.18)
	seafloor := m.outerOceanSeafloor(x, z, sea, deep)
	edgeFloor := math.Min(edge.World.GroundHeight, sea-0.004)
	ground := lerp(edgeFloor, seafloor, t)

	polar := m.polarFactor(z)
	climate := m.SampleClimate(x, z)
	legacyOuterIslands := m.GenerationRulesVersion == "" ||
		m.GenerationRulesVersion == "geoworld-rules-v1.1"
	radialOuterIslands := m.GenerationRulesVersion == "geoworld-rules-v1.1.1"
	naturalOuterIslandsV112 := m.GenerationRulesVersion == "geoworld-rules-v1.1.2"
	naturalOuterIslands := !legacyOuterIslands &&
		!radialOuterIslands &&
		!naturalOuterIslandsV112
	naturalOuterIslandFamily := naturalOuterIslandsV112 || naturalOuterIslands

	islandNoise := 0.68*valueNoise(m.Seed+77101, x, z, 1800) +
		0.32*valueNoise(m.Seed+77123, x, z, 720)
	islandSignal := 0.5 + 0.5*islandNoise
	islandThreshold := clamp(0.86-0.08*(m.OuterIslandAbundance-1), 0.70, 0.94)
	islandShape := smoothstep(islandThreshold, math.Min(0.995, islandThreshold+0.10), islandSignal)
	// Two deterministic shelf-linked anchor islands guarantee that the normal
	// island system is observable for every seed without filling the exterior
	// ocean. The broad noise field still creates the rest of the sparse islands.
	islandShape = math.Max(
		islandShape,
		m.guaranteedOuterIslandShape(x, z, transition),
	)
	distanceFade := 1 - smoothstepUnit(clamp(
		(distance-transition*0.20)/(transition*2.20), 0, 1,
	))
	polarSuppression := 1 - smoothstepUnit(clamp((polar-0.35)/0.45, 0, 1))
	islandWeight := clamp(islandShape*distanceFade*polarSuppression, 0, 1)
	islandRelief := 0.0
	if islandWeight > 0 {
		switch {
		case naturalOuterIslands:
			// v1.1.3: most of an offshore island approaches the sea through a
			// low, broad coastal plain. Only a minority of broad coastline
			// sectors become cliff/high-coast segments. Elevation therefore no
			// longer rises as one continuous high ring around the whole island.
			terrainWarpX := 110 * valueNoise(m.Seed+77301, x, z, 780)
			terrainWarpZ := 95 * valueNoise(m.Seed+77303, x, z, 690)
			tx := x + terrainWarpX
			tz := z + terrainWarpZ
			broad := 0.56*valueNoise(m.Seed+77311, tx, tz, 980) +
				0.29*valueNoise(m.Seed+77317, tx, tz, 470) +
				0.15*valueNoise(m.Seed+77319, tx, tz, 235)
			ridge := 1 - math.Abs(valueNoise(m.Seed+77323, tx, tz, 720))
			ridge = ridge * ridge
			islandRelief = clamp(0.44+0.30*broad+0.20*ridge, 0, 1)
			detail := valueNoise(m.Seed+77329, tx, tz, 170)
			coastField := 0.72*valueNoise(m.Seed+77331, x, z, 920) +
				0.28*valueNoise(m.Seed+77337, x, z, 430)
			cliffSector := smoothstep(0.58, 0.84, coastField)
			gentleRise := clamp(
				0.004+0.030*islandRelief+0.004*detail,
				0.003,
				0.040,
			)
			cliffRise := clamp(
				0.020+0.070*islandRelief+0.008*detail,
				0.016,
				0.095,
			)
			gentleRamp := smoothstep(0.12, 0.80, islandWeight)
			cliffRamp := smoothstep(0.05, 0.34, islandWeight)
			coastRamp := lerp(gentleRamp, cliffRamp, cliffSector)
			rise := lerp(gentleRise, cliffRise, cliffSector)
			target := sea + rise*coastRamp
			blend := smoothstep(0.06, 0.48, islandWeight)
			ground = lerp(ground, target, blend)
		case naturalOuterIslandsV112:
			// v1.1.2: footprint and elevation are deliberately independent.
			// The coastline mask only raises terrain out of the sea; the island
			// interior comes from broad rolling fields, ridges and local detail.
			// This prevents the old radial "hemisphere / flat cap / contour ring"
			// failure mode.
			terrainWarpX := 110 * valueNoise(m.Seed+77301, x, z, 780)
			terrainWarpZ := 95 * valueNoise(m.Seed+77303, x, z, 690)
			tx := x + terrainWarpX
			tz := z + terrainWarpZ
			broad := 0.56*valueNoise(m.Seed+77311, tx, tz, 980) +
				0.29*valueNoise(m.Seed+77317, tx, tz, 470) +
				0.15*valueNoise(m.Seed+77319, tx, tz, 235)
			ridge := 1 - math.Abs(valueNoise(m.Seed+77323, tx, tz, 720))
			ridge = ridge * ridge
			islandRelief = clamp(0.48+0.34*broad+0.24*ridge, 0, 1)
			detail := valueNoise(m.Seed+77329, tx, tz, 170)
			rise := clamp(
				0.008+0.038*islandRelief+0.006*detail,
				0.006,
				0.060,
			)
			coastCharacter := 0.5 + 0.5*valueNoise(
				m.Seed+77331, tx, tz, 1100,
			)
			cliffSector := smoothstep(0.72, 0.90, coastCharacter)
			coastStart := lerp(0.20, 0.08, cliffSector)
			coastFull := lerp(0.72, 0.48, cliffSector)
			coastRamp := smoothstep(coastStart, coastFull, islandWeight)
			target := sea + rise*coastRamp
			blend := smoothstep(0.08, 0.52, islandWeight)
			ground = lerp(ground, target, blend)
		case radialOuterIslands:
			relief := 0.5 + 0.5*valueNoise(m.Seed+77147, x, z, 260)
			islandTop := sea + 0.012 + 0.060*math.Pow(islandWeight, 1.15) +
				0.018*relief*islandWeight
			ground = lerp(ground, islandTop, islandWeight)
		default:
			islandTop := sea + 0.018 + 0.082*islandWeight
			ground = lerp(ground, islandTop, islandWeight)
		}
	}
	islandLand := islandWeight >= 0.58 && ground > sea+0.002
	if radialOuterIslands {
		islandLand = islandWeight > 0.18 && ground > sea+0.001
	} else if naturalOuterIslandsV112 {
		islandLand = islandWeight > 0.14 && ground > sea+0.001
	} else if naturalOuterIslands {
		// v1.1.3 uses a slightly more conservative coastline mask so tiny
		// one-cell lobes do not become a saw-tooth land/water boundary.
		islandLand = islandWeight > 0.20 && ground > sea+0.0015
	}
	if islandLand {
		biome := mcprofile.BiomeTemperateGrassland
		surface := mcprofile.SurfaceGrassLoam
		vegetation := mcprofile.VegetationGrassland
		switch {
		case polar > 0.28 || climate.AnnualMeanTempC < 1:
			biome = mcprofile.BiomeTundra
			surface = mcprofile.SurfaceTundraSoil
			vegetation = mcprofile.VegetationTundraSparse
		case climate.AnnualMeanTempC < 9:
			biome = mcprofile.BiomeBorealForest
			surface = mcprofile.SurfacePodzol
			vegetation = mcprofile.VegetationConiferForest
		case climate.AnnualMeanTempC > 22 && climate.AnnualPrecipMM > 1400:
			biome = mcprofile.BiomeTropicalRainforest
			surface = mcprofile.SurfaceForestLoam
			vegetation = mcprofile.VegetationDenseRainforest
		case climate.AnnualMeanTempC > 18:
			biome = mcprofile.BiomeSavanna
			surface = mcprofile.SurfaceDryGrass
			vegetation = mcprofile.VegetationSavanna
		}
		terrain := mcprofile.TerrainPlain
		if naturalOuterIslandFamily {
			if islandRelief > 0.60 {
				terrain = mcprofile.TerrainHills
			} else if islandRelief > 0.38 {
				terrain = mcprofile.TerrainRollingPlain
			}
		} else if islandWeight > 0.82 {
			terrain = mcprofile.TerrainHills
		}
		maxIslandHeight := 0.72
		if naturalOuterIslandFamily {
			maxIslandHeight = 0.90
		}
		return runtimeprofile.Sample{
			World: worldsample.Sample{
				GroundHeight: clamp(ground, sea+0.002, maxIslandHeight),
				WaterKind:    reconcile.WaterNone,
			},
			SourceNode:        -1,
			SourceWeight:      islandWeight,
			SourceLand:        true,
			EcoRegion:         ecology.Region(biome),
			RegionConfidence:  islandWeight,
			BiomeFamily:       biome,
			TerrainVariant:    terrain,
			SurfaceProfile:    surface,
			VegetationProfile: vegetation,
			ProfileConfidence: islandWeight,
		}
	}

	switch {
	case naturalOuterIslandFamily:
		ground = math.Min(ground, sea-0.004)
	case radialOuterIslands:
		ground = math.Min(ground, sea-0.002)
	default:
		ground = math.Min(ground, sea-0.012)
	}
	node, weight, _ := m.Finite.SampleSource(edgeX, edgeZ, false)
	biome := mcprofile.BiomeTemperateOcean
	surface := mcprofile.SurfaceOceanGravel
	switch {
	case polar >= 0.72:
		biome = mcprofile.BiomeFrozenOcean
		surface = mcprofile.SurfaceFrozenOcean
	case polar >= 0.34 || climate.AnnualMeanTempC < 8:
		biome = mcprofile.BiomeColdOcean
		surface = mcprofile.SurfaceOceanGravel
	case climate.AnnualMeanTempC > 20:
		biome = mcprofile.BiomeWarmOcean
		surface = mcprofile.SurfaceOceanSand
	}
	return runtimeprofile.Sample{
		World: worldsample.Sample{
			GroundHeight:          ground,
			WaterKind:             reconcile.WaterOcean,
			WaterSurfaceDefined:   true,
			WaterSurfaceElevation: sea,
			WaterDepth:            sea - ground,
			OceanEligibility:      1,
		},
		SourceNode:        node,
		SourceWeight:      weight,
		SourceLand:        false,
		EcoRegion:         ecology.Region(biome),
		RegionConfidence:  1,
		BiomeFamily:       biome,
		TerrainVariant:    mcprofile.TerrainOcean,
		SurfaceProfile:    surface,
		VegetationProfile: mcprofile.VegetationNone,
		ProfileConfidence: 1,
		// The exterior itself is authoritative. A clamped edge source is
		// only a categorical seed, so finite-grid source fallback must not mark
		// the outer sample as a semantic fallback.
		SemanticFallback: false,
	}
}

// outerOceanSeafloor extends the finite model with deterministic large-scale
// ocean-floor geography. v1.1.7 adds basin-scale relief plus sparse ridge,
// seamount and trench structures while keeping every feature safely submerged.
// Older locked rule versions retain the exact historical single-noise formula.
func (m *Model) outerOceanSeafloor(x, z, sea, deep float64) float64 {
	legacy := deep + 0.018*valueNoise(m.Seed+77003, x, z, 5200)
	if m.GenerationRulesVersion != "geoworld-rules-v1.1.7" &&
		m.GenerationRulesVersion != "geoworld-rules-v1.1.8" &&
		m.GenerationRulesVersion != "geoworld-rules-v1.1.9" &&
		m.GenerationRulesVersion != "geoworld-rules-v1.1.10" &&
		m.GenerationRulesVersion != "geoworld-rules-v1.1.11" {
		return legacy
	}

	// Broad basin relief is intentionally much larger in wavelength than
	// ordinary terrain detail so the far ocean reads as regions, not noise.
	basin :=
		0.020*valueNoise(m.Seed+77011, x, z, 16000) +
			0.010*valueNoise(m.Seed+77017, x, z, 7200)

	// Long, sparse ridge belts follow narrow zero-crossing bands. A second,
	// broader field suppresses most candidate bands so ridges do not fill the
	// whole ocean.
	ridgeLine := 1 - math.Abs(valueNoise(m.Seed+77021, x, z, 11500))
	ridgeGate := smoothstep(0.08, 0.58, valueNoise(m.Seed+77023, x, z, 24000))
	ridge :=
		0.030 *
			smoothstep(0.82, 0.98, ridgeLine) *
			ridgeGate

	// Seamounts are compact intersections of two independent broad fields.
	// They remain submarine; outer islands continue to be owned exclusively by
	// the dedicated island system below.
	seamountSignal :=
		(0.5 + 0.5*valueNoise(m.Seed+77031, x, z, 4800)) *
			(0.5 + 0.5*valueNoise(m.Seed+77033, x, z, 2600))
	seamountGate := smoothstep(
		0.20,
		0.70,
		valueNoise(m.Seed+77037, x, z, 18000),
	)
	seamount :=
		0.045 *
			smoothstep(0.78, 0.95, seamountSignal) *
			seamountGate

	// Trenches use a separate narrow-band field so they can cross basin floors
	// without being the negative image of the ridge network.
	trenchLine := 1 - math.Abs(valueNoise(m.Seed+77041, x, z, 13000))
	trenchGate := smoothstep(0.18, 0.70, valueNoise(m.Seed+77043, x, z, 22000))
	trench :=
		0.035 *
			smoothstep(0.86, 0.985, trenchLine) *
			trenchGate

	detail := 0.006 * valueNoise(m.Seed+77051, x, z, 2800)
	floor := deep + basin + ridge + seamount - trench + detail

	// Never let ordinary ocean-floor geography become emergent land. The lower
	// bound also prevents trenches from collapsing onto the world bottom.
	return clamp(
		floor,
		math.Max(0.06, deep-0.09),
		sea-0.035,
	)
}

func (m *Model) guaranteedOuterIslandShape(x, z, transition float64) float64 {
	switch m.GenerationRulesVersion {
	case "geoworld-rules-v1.1":
		return m.guaranteedOuterIslandShapeV11(x, z, transition)
	case "geoworld-rules-v1.1.1":
		return m.guaranteedOuterIslandShapeV111(x, z, transition)
	}

	centerZ := m.minZ() + m.Geometry.SpanZBlocks*0.5
	best := 0.0
	for side := 0; side < 2; side++ {
		key := int64(side + 1)
		distanceJitter := 0.5 + 0.5*hashValue(m.Seed+77201, key, 11)
		zJitter := hashValue(m.Seed+77203, key, 17)
		radiusJitter := 0.5 + 0.5*hashValue(m.Seed+77207, key, 23)
		aspectJitter := 0.5 + 0.5*hashValue(m.Seed+77211, key, 29)
		rotation := math.Pi * hashValue(m.Seed+77213, key, 31)

		offshore := transition * (0.26 + 0.12*distanceJitter)
		anchorX := m.minX() - offshore
		if side == 1 {
			anchorX = m.maxX() + offshore
		}
		anchorZ := centerZ + zJitter*0.22*m.Geometry.SpanZBlocks

		radiusScale := clamp(0.75+0.25*m.OuterIslandAbundance, 0.60, 1.40)
		radius := (420 + 300*radiusJitter) * radiusScale
		aspect := 0.64 + 0.48*aspectJitter

		warpX := radius * 0.16 * valueNoise(m.Seed+77217+key*97, x, z, 540)
		warpZ := radius * 0.13 * valueNoise(m.Seed+77219+key*101, x, z, 470)
		dx := x - anchorX + warpX
		dz := z - anchorZ + warpZ
		c := math.Cos(rotation)
		s := math.Sin(rotation)
		rx := dx*c + dz*s
		rz := -dx*s + dz*c

		mainScore := 1 - math.Hypot(rx/radius, rz/(radius*aspect))

		lobe1X := radius * (0.24 + 0.10*hashValue(m.Seed+77223, key, 37))
		lobe1Z := radius * aspect * 0.26 * hashValue(m.Seed+77229, key, 41)
		lobe1Score := 0.78 - math.Hypot(
			(rx-lobe1X)/(radius*0.62),
			(rz-lobe1Z)/(radius*aspect*0.58),
		)

		lobe2X := -radius * (0.20 + 0.12*(0.5+0.5*hashValue(m.Seed+77231, key, 43)))
		lobe2Z := radius * aspect * 0.30 * hashValue(m.Seed+77237, key, 47)
		lobe2Score := 0.73 - math.Hypot(
			(rx-lobe2X)/(radius*0.55),
			(rz-lobe2Z)/(radius*aspect*0.52),
		)

		score := math.Max(mainScore, math.Max(lobe1Score, lobe2Score))
		score += 0.12*valueNoise(m.Seed+77241+key*109, x, z, 260) +
			0.05*valueNoise(m.Seed+77243+key*113, x, z, 125)
		// v1.1.3 widens the coastline transition so the whole island rim
		// does not climb at once. Most coasts become low/gradual while local
		// relief can still create isolated cliff sectors.
		shape := smoothstep(-0.24, 0.24, score)
		if shape > best {
			best = shape
		}
	}
	return clamp(best, 0, 1)
}

func (m *Model) guaranteedOuterIslandShapeV111(x, z, transition float64) float64 {
	centerZ := m.minZ() + m.Geometry.SpanZBlocks*0.5
	best := 0.0
	for side := 0; side < 2; side++ {
		key := int64(side + 1)
		distanceJitter := 0.5 + 0.5*hashValue(m.Seed+77201, key, 11)
		zJitter := hashValue(m.Seed+77203, key, 17)
		radiusJitter := 0.5 + 0.5*hashValue(m.Seed+77207, key, 23)
		aspectJitter := 0.5 + 0.5*hashValue(m.Seed+77211, key, 29)
		phase := math.Pi * hashValue(m.Seed+77213, key, 31)

		offshore := transition * (0.26 + 0.12*distanceJitter)
		anchorX := m.minX() - offshore
		if side == 1 {
			anchorX = m.maxX() + offshore
		}
		anchorZ := centerZ + zJitter*0.22*m.Geometry.SpanZBlocks

		radiusScale := clamp(0.75+0.25*m.OuterIslandAbundance, 0.60, 1.40)
		radius := (360 + 260*radiusJitter) * radiusScale
		aspect := 0.68 + 0.42*aspectJitter
		dx := x - anchorX
		dz := z - anchorZ
		angle := math.Atan2(dz, dx)
		coastWarp := 1 +
			0.14*valueNoise(m.Seed+77217+key*97, x, z, 420) +
			0.08*math.Sin(3*angle+phase) +
			0.05*math.Sin(5*angle-0.7*phase)
		coastWarp = clamp(coastWarp, 0.72, 1.28)
		normalizedDistance := math.Hypot(
			dx/radius,
			dz/(radius*aspect),
		) / coastWarp
		shape := 1 - smoothstep(0.38, 1, normalizedDistance)
		if shape > best {
			best = shape
		}
	}
	return clamp(best, 0, 1)
}

func (m *Model) guaranteedOuterIslandShapeV11(x, z, transition float64) float64 {
	centerZ := m.minZ() + m.Geometry.SpanZBlocks*0.5
	best := 0.0
	for side := 0; side < 2; side++ {
		key := int64(side + 1)
		distanceJitter := 0.5 + 0.5*hashValue(m.Seed+77201, key, 11)
		zJitter := hashValue(m.Seed+77203, key, 17)
		radiusJitter := 0.5 + 0.5*hashValue(m.Seed+77207, key, 23)

		offshore := transition * (0.26 + 0.12*distanceJitter)
		anchorX := m.minX() - offshore
		if side == 1 {
			anchorX = m.maxX() + offshore
		}
		anchorZ := centerZ + zJitter*0.22*m.Geometry.SpanZBlocks
		radiusScale := clamp(0.75+0.25*m.OuterIslandAbundance, 0.60, 1.40)
		radius := (360 + 260*radiusJitter) * radiusScale
		distance := math.Hypot(x-anchorX, z-anchorZ)
		shape := 1 - smoothstep(radius*0.35, radius, distance)
		if shape > best {
			best = shape
		}
	}
	return clamp(best, 0, 1)
}

func (m *Model) outside(x, z float64) bool {
	return x < m.minX() || x > m.maxX() || z < m.minZ() || z > m.maxZ()
}
func (m *Model) minX() float64 { return m.Geometry.OriginX }
func (m *Model) maxX() float64 { return m.Geometry.OriginX + m.Geometry.SpanXBlocks }
func (m *Model) minZ() float64 { return m.Geometry.OriginZ }
func (m *Model) maxZ() float64 { return m.Geometry.OriginZ + m.Geometry.SpanZBlocks }
func (m *Model) outsideDistance(x, z float64) float64 {
	dx := 0.0
	if x < m.minX() {
		dx = m.minX() - x
	} else if x > m.maxX() {
		dx = x - m.maxX()
	}
	dz := 0.0
	if z < m.minZ() {
		dz = m.minZ() - z
	} else if z > m.maxZ() {
		dz = z - m.maxZ()
	}
	return math.Hypot(dx, dz)
}
func (m *Model) polarFactor(z float64) float64 {
	center := m.minZ() + m.Geometry.SpanZBlocks*0.5
	half := math.Max(1, m.Geometry.SpanZBlocks*0.5)
	latitude := math.Abs((z - center) / half)
	return smoothstepUnit(clamp((latitude-0.72)/(0.40*m.PolarTransitionWidth), 0, 1))
}

func valueNoise(seed int64, x, z, wavelength float64) float64 {
	if wavelength <= 0 {
		return 0
	}
	gx, gz := x/wavelength, z/wavelength
	x0, z0 := int64(math.Floor(gx)), int64(math.Floor(gz))
	tx, tz := gx-float64(x0), gz-float64(z0)
	fx, fz := quinticFade(tx), quinticFade(tz)
	v00 := hashValue(seed, x0, z0)
	v10 := hashValue(seed, x0+1, z0)
	v01 := hashValue(seed, x0, z0+1)
	v11 := hashValue(seed, x0+1, z0+1)
	return lerp(lerp(v00, v10, fx), lerp(v01, v11, fx), fz)
}
func hashValue(seed, x, z int64) float64 {
	h := uint64(seed) ^ uint64(x)*0x9E3779B185EBCA87 ^ uint64(z)*0xC2B2AE3D27D4EB4F
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	h ^= h >> 31
	unit := float64(h>>11) / float64(uint64(1)<<53)
	return unit*2 - 1
}
func quinticFade(t float64) float64 {
	t = clamp(t, 0, 1)
	return t * t * t * (t*(t*6-15) + 10)
}
func smoothstep(edge0, edge1, x float64) float64 {
	if edge1 <= edge0 {
		if x >= edge1 {
			return 1
		}
		return 0
	}
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}
func smoothstepUnit(t float64) float64 {
	t = clamp(t, 0, 1)
	return t * t * (3 - 2*t)
}
func lerp(a, b, t float64) float64    { return a + (b-a)*t }
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
