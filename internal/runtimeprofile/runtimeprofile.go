package runtimeprofile

import (
	"errors"
	"math"

	"geoworld/internal/ecology"
	"geoworld/internal/mcprofile"
	"geoworld/internal/reconcile"
	"geoworld/internal/world"
	"geoworld/internal/worldsample"
)

func Build(
	geometry world.GridGeometry,
	eco *ecology.Model,
	profile *mcprofile.Model,
	runtime *worldsample.Model,
) (*Model, error) {
	if _, err := geometry.SquareCellSize(); err != nil {
		return nil, err
	}
	if eco == nil || profile == nil || runtime == nil {
		return nil, errors.New("runtimeprofile: ecology, profile and runtime models are required")
	}
	if eco.Width != geometry.Width || eco.Height != geometry.Height ||
		profile.Width != geometry.Width || profile.Height != geometry.Height ||
		!sameGeometry(runtime.Geometry, geometry) {
		return nil, errors.New("runtimeprofile: model geometry does not match")
	}
	n, err := world.NodeCount(geometry.Width, geometry.Height)
	if err != nil {
		return nil, err
	}
	if len(eco.LandMask) != n || len(eco.EcoRegion) != n || len(eco.RegionConfidence) != n ||
		len(profile.LandMask) != n || len(profile.BiomeFamily) != n ||
		len(profile.TerrainVariant) != n || len(profile.SurfaceProfile) != n ||
		len(profile.VegetationProfile) != n || len(profile.ProfileConfidence) != n {
		return nil, errors.New("runtimeprofile: model fields do not match geometry")
	}
	for i := 0; i < n; i++ {
		if eco.LandMask[i] != profile.LandMask[i] {
			return nil, errors.New("runtimeprofile: ecology/profile land masks disagree")
		}
	}
	return &Model{
		Geometry: geometry,
		Ecology:  eco,
		Profile:  profile,
		Runtime:  runtime,
	}, nil
}

func (m *Model) Sample(worldX, worldZ float64) Sample {
	ws := m.Runtime.Sample(worldX, worldZ)
	wantLand := ws.WaterKind != reconcile.WaterOcean

	node, weight, fallback := m.selectSourceNode(worldX, worldZ, wantLand)
	out := Sample{
		World:             ws,
		SourceNode:        node,
		SourceWeight:      weight,
		SourceLand:        m.Profile.LandMask[node],
		EcoRegion:         m.Ecology.EcoRegion[node],
		RegionConfidence:  m.Ecology.RegionConfidence[node],
		BiomeFamily:       m.Profile.BiomeFamily[node],
		TerrainVariant:    m.Profile.TerrainVariant[node],
		SurfaceProfile:    m.Profile.SurfaceProfile[node],
		VegetationProfile: m.Profile.VegetationProfile[node],
		ProfileConfidence: m.Profile.ProfileConfidence[node],
		SemanticFallback:  fallback,
	}
	m.applyElevationZoning(worldX, worldZ, &out)
	out.SemanticMismatch = !runtimeSideMatchesProfile(ws.WaterKind, out)
	return out
}

type candidate struct {
	node   int
	weight float64
}

func (m *Model) selectSourceNode(worldX, worldZ float64, wantLand bool) (int, float64, bool) {
	spacingX := m.Geometry.SpacingX()
	spacingZ := m.Geometry.SpacingZ()

	// Categorical ecology/profile fields cannot be numerically interpolated like
	// height. Sampling a raw macro cell therefore produces obvious square
	// biome borders. Bend only the categorical lookup coordinates with a
	// deterministic, low-frequency domain warp; Ground/Water still sample the
	// original world coordinate, so coastlines and terrain remain authoritative.
	seed := int64(0)
	if m.Runtime != nil && m.Runtime.Ground != nil && m.Runtime.Ground.Base != nil {
		seed = m.Runtime.Ground.Base.Config.Seed
	}
	cellSize := 0.5 * (spacingX + spacingZ)
	wavelength := cellSize * 5.0
	amplitude := cellSize * 0.42
	warpX := 0.72*valueNoise(seed+12011, worldX, worldZ, wavelength) +
		0.28*valueNoise(seed+12029, worldX, worldZ, wavelength*0.5)
	warpZ := 0.72*valueNoise(seed+22003, worldX, worldZ, wavelength) +
		0.28*valueNoise(seed+22027, worldX, worldZ, wavelength*0.5)
	profileX := worldX + amplitude*warpX
	profileZ := worldZ + amplitude*warpZ

	gx := clamp((profileX-m.Geometry.OriginX)/spacingX, 0, float64(m.Geometry.Width-1))
	gz := clamp((profileZ-m.Geometry.OriginZ)/spacingZ, 0, float64(m.Geometry.Height-1))

	x0 := int(math.Floor(gx))
	z0 := int(math.Floor(gz))
	x1 := minInt(x0+1, m.Geometry.Width-1)
	z1 := minInt(z0+1, m.Geometry.Height-1)

	tx := smoothstep(gx - float64(x0))
	tz := smoothstep(gz - float64(z0))

	candidates := []candidate{
		{node: z0*m.Geometry.Width + x0, weight: (1 - tx) * (1 - tz)},
		{node: z0*m.Geometry.Width + x1, weight: tx * (1 - tz)},
		{node: z1*m.Geometry.Width + x0, weight: (1 - tx) * tz},
		{node: z1*m.Geometry.Width + x1, weight: tx * tz},
	}

	if best, bestWeight, ok := m.selectErodedCandidate(
		candidates,
		wantLand,
		seed,
		worldX,
		worldZ,
		cellSize,
	); ok {
		return best, bestWeight, false
	}

	// Near coastlines the warped categorical coordinate can cross to the other
	// semantic side while Ground/Water (correctly) stays at the original
	// coordinate. Retry the unwarped cell before declaring a semantic fallback.
	originalGX := clamp((worldX-m.Geometry.OriginX)/spacingX, 0, float64(m.Geometry.Width-1))
	originalGZ := clamp((worldZ-m.Geometry.OriginZ)/spacingZ, 0, float64(m.Geometry.Height-1))
	originalX0 := int(math.Floor(originalGX))
	originalZ0 := int(math.Floor(originalGZ))
	originalX1 := minInt(originalX0+1, m.Geometry.Width-1)
	originalZ1 := minInt(originalZ0+1, m.Geometry.Height-1)
	originalTX := smoothstep(originalGX - float64(originalX0))
	originalTZ := smoothstep(originalGZ - float64(originalZ0))
	originalCandidates := []candidate{
		{node: originalZ0*m.Geometry.Width + originalX0, weight: (1 - originalTX) * (1 - originalTZ)},
		{node: originalZ0*m.Geometry.Width + originalX1, weight: originalTX * (1 - originalTZ)},
		{node: originalZ1*m.Geometry.Width + originalX0, weight: (1 - originalTX) * originalTZ},
		{node: originalZ1*m.Geometry.Width + originalX1, weight: originalTX * originalTZ},
	}
	if best, bestWeight, ok := m.selectErodedCandidate(
		originalCandidates,
		wantLand,
		seed,
		worldX,
		worldZ,
		cellSize,
	); ok {
		return best, bestWeight, false
	}

	// A valid final sample should always have at least one matching corner:
	// continuous ocean eligibility cannot produce Ocean inside an all-land cell,
	// and non-ocean samples inside mixed/all-land cells have a land corner.
	// Keep a deterministic fallback so diagnostics can expose violations without
	// making Runtime queries panic.
	best := candidates[0].node
	bestWeight := candidates[0].weight
	for _, c := range candidates[1:] {
		if c.weight > bestWeight {
			best = c.node
			bestWeight = c.weight
		}
	}
	return best, bestWeight, true
}

func (m *Model) selectErodedCandidate(
	candidates []candidate,
	wantLand bool,
	seed int64,
	worldX, worldZ, cellSize float64,
) (int, float64, bool) {
	eligible := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		if m.Profile.LandMask[c.node] != wantLand {
			continue
		}
		duplicate := -1
		for i := range eligible {
			if eligible[i].node == c.node {
				duplicate = i
				break
			}
		}
		if duplicate >= 0 {
			if c.weight > eligible[duplicate].weight {
				eligible[duplicate].weight = c.weight
			}
			continue
		}
		eligible = append(eligible, c)
	}
	if len(eligible) == 0 {
		return -1, -1, false
	}
	if len(eligible) == 1 {
		return eligible[0].node, eligible[0].weight, true
	}

	bestBase := -1.0
	secondBase := -1.0
	for _, c := range eligible {
		if c.weight > bestBase {
			secondBase = bestBase
			bestBase = c.weight
		} else if c.weight > secondBase {
			secondBase = c.weight
		}
	}
	gap := math.Max(0, bestBase-secondBase)
	ambiguity := 1 - smoothstep(clamp(gap/0.38, 0, 1))
	erosionStrength := 0.16 * ambiguity

	best := eligible[0].node
	bestWeight := eligible[0].weight
	bestScore := -math.MaxFloat64
	for _, c := range eligible {
		nodeSeed := int64(c.node)
		competitionNoise :=
			0.66*valueNoise(
				seed+31013+nodeSeed*7919,
				worldX,
				worldZ,
				cellSize*0.34,
			) +
				0.34*valueNoise(
					seed+41017+nodeSeed*1543,
					worldX,
					worldZ,
					cellSize*0.14,
				)
		score := c.weight + erosionStrength*competitionNoise
		if score > bestScore {
			best = c.node
			bestWeight = c.weight
			bestScore = score
		}
	}
	return best, bestWeight, true
}

func (m *Model) applyElevationZoning(worldX, worldZ float64, s *Sample) {
	if s == nil || m.Runtime == nil || !s.SourceLand || s.World.WaterKind == reconcile.WaterOcean {
		return
	}

	context := terrainElevationContext(s.TerrainVariant)
	if context < 0.45 {
		return
	}

	sea := m.Runtime.SeaLevelNormalized
	if sea <= 0 || sea >= 1 {
		sea = 0.5
	}
	relativeHeight := clamp((s.World.GroundHeight-sea)/(1-sea), 0, 1)

	seed := int64(0)
	if m.Runtime.Ground != nil && m.Runtime.Ground.Base != nil {
		seed = m.Runtime.Ground.Base.Config.Seed
	}
	cellSize := 0.5 * (m.Geometry.SpacingX() + m.Geometry.SpacingZ())
	bandNoise :=
		0.028*valueNoise(seed+33071, worldX, worldZ, cellSize*5.5) +
			0.012*valueNoise(seed+33091, worldX, worldZ, cellSize*2.2)
	effectiveHeight := clamp(
		relativeHeight+bandNoise+0.025*(context-0.5),
		0,
		1,
	)

	snowLine := biomeSnowLine(s.BiomeFamily)
	if effectiveHeight >= snowLine {
		s.BiomeFamily = mcprofile.BiomeAlpine
		s.VegetationProfile = mcprofile.VegetationAlpineSparse
		// Exposed rock follows terrain steepness rather than a continuous
		// altitude stripe. Snow stays on walkable shoulders, including the
		// lower alpine transition, while steep faces can be bare even high up.
		baseHeight := m.Runtime.Ground.Base.SampleNormalized(worldX, worldZ)
		dx := (m.Runtime.Ground.Base.SampleNormalized(worldX+16, worldZ) - baseHeight) * 514 / 16
		dz := (m.Runtime.Ground.Base.SampleNormalized(worldX, worldZ+16) - baseHeight) * 514 / 16
		slope := math.Hypot(dx, dz)
		rockPatch := 0.5 + 0.5*valueNoise(seed+33131, worldX, worldZ, cellSize*1.1)
		if exposedAlpineRock(slope, rockPatch) {
			s.SurfaceProfile = mcprofile.SurfaceAlpineRock
		} else {
			s.SurfaceProfile = mcprofile.SurfaceAlpineSnow
		}
		return
	}

	// The band directly below the alpine zone keeps the broad climate family
	// but cools vegetation/ecology with altitude. This makes a mountain read as
	// one landform with ecological belts rather than one rectangular biome slab.
	if context < 0.65 || effectiveHeight < snowLine-0.12 {
		return
	}

	switch s.BiomeFamily {
	case mcprofile.BiomeTropicalRainforest, mcprofile.BiomeTropicalMonsoonForest:
		s.BiomeFamily = mcprofile.BiomeSubtropicalForest
		s.SurfaceProfile = mcprofile.SurfaceForestLoam
		s.VegetationProfile = mcprofile.VegetationBroadleafForest
	case mcprofile.BiomeSubtropicalForest:
		s.BiomeFamily = mcprofile.BiomeTemperateForest
		s.SurfaceProfile = mcprofile.SurfaceForestLoam
		s.VegetationProfile = mcprofile.VegetationMixedForest
	case mcprofile.BiomeTemperateForest:
		s.BiomeFamily = mcprofile.BiomeBorealForest
		s.SurfaceProfile = mcprofile.SurfacePodzol
		s.VegetationProfile = mcprofile.VegetationConiferForest
	case mcprofile.BiomeBorealForest:
		s.BiomeFamily = mcprofile.BiomeTundra
		s.SurfaceProfile = mcprofile.SurfaceTundraSoil
		s.VegetationProfile = mcprofile.VegetationTundraSparse
	case mcprofile.BiomeSavanna, mcprofile.BiomeMediterraneanShrubland:
		s.BiomeFamily = mcprofile.BiomeTemperateGrassland
		s.SurfaceProfile = mcprofile.SurfaceDryGrass
		s.VegetationProfile = mcprofile.VegetationDryGrassland
	case mcprofile.BiomeHotDesert:
		s.BiomeFamily = mcprofile.BiomeTemperateDesert
		s.SurfaceProfile = mcprofile.SurfaceShrubSoil
		s.VegetationProfile = mcprofile.VegetationDesertSparse
	case mcprofile.BiomeWetland:
		s.BiomeFamily = mcprofile.BiomeTemperateGrassland
		s.SurfaceProfile = mcprofile.SurfaceGrassLoam
		s.VegetationProfile = mcprofile.VegetationGrassland
	}
}

// A flat snowfield must remain snow regardless of altitude or patch noise.
// Rock outcrops are reserved for exposed high-gradient alpine faces.
func exposedAlpineRock(slope, rockPatch float64) bool {
	exposure := smoothstep(clamp((slope-0.32)/(0.85-0.32), 0, 1))
	return exposure > 0.52+0.18*(rockPatch-0.5)
}

func terrainElevationContext(v mcprofile.TerrainVariant) float64 {
	switch v {
	case mcprofile.TerrainMountain:
		return 1.0
	case mcprofile.TerrainFoothills:
		return 0.82
	case mcprofile.TerrainPlateau:
		return 0.68
	case mcprofile.TerrainHills:
		return 0.56
	default:
		return 0
	}
}

func biomeSnowLine(b mcprofile.BiomeFamily) float64 {
	switch b {
	case mcprofile.BiomeIceSheet, mcprofile.BiomeTundra:
		return 0.20
	case mcprofile.BiomeAlpine:
		return 0.24
	case mcprofile.BiomeBorealForest:
		return 0.30
	case mcprofile.BiomeTemperateForest,
		mcprofile.BiomeTemperateGrassland,
		mcprofile.BiomeTemperateDesert,
		mcprofile.BiomeWetland:
		return 0.44
	case mcprofile.BiomeMediterraneanShrubland:
		return 0.49
	case mcprofile.BiomeSubtropicalForest, mcprofile.BiomeSavanna:
		return 0.55
	case mcprofile.BiomeHotDesert:
		return 0.60
	case mcprofile.BiomeTropicalMonsoonForest, mcprofile.BiomeTropicalRainforest:
		return 0.62
	default:
		return 0.46
	}
}

func runtimeSideMatchesProfile(kind reconcile.WaterKind, s Sample) bool {
	if kind == reconcile.WaterOcean {
		return !s.SourceLand &&
			isOceanFamily(s.BiomeFamily) &&
			s.TerrainVariant == mcprofile.TerrainOcean &&
			s.VegetationProfile == mcprofile.VegetationNone
	}
	return s.SourceLand && !isOceanFamily(s.BiomeFamily) && s.TerrainVariant != mcprofile.TerrainOcean
}

func isOceanFamily(f mcprofile.BiomeFamily) bool {
	switch f {
	case mcprofile.BiomeWarmOcean,
		mcprofile.BiomeTemperateOcean,
		mcprofile.BiomeColdOcean,
		mcprofile.BiomeFrozenOcean:
		return true
	default:
		return false
	}
}

func valueNoise(seed int64, x, z, wavelength float64) float64 {
	if wavelength <= 0 {
		return 0
	}
	gx := x / wavelength
	gz := z / wavelength
	x0 := int64(math.Floor(gx))
	z0 := int64(math.Floor(gz))
	tx := gx - float64(x0)
	tz := gz - float64(z0)

	fx := quinticFade(tx)
	fz := quinticFade(tz)
	v00 := hashValue(seed, x0, z0)
	v10 := hashValue(seed, x0+1, z0)
	v01 := hashValue(seed, x0, z0+1)
	v11 := hashValue(seed, x0+1, z0+1)
	top := v00 + (v10-v00)*fx
	bottom := v01 + (v11-v01)*fx
	return top + (bottom-top)*fz
}

func hashValue(seed, x, z int64) float64 {
	h := uint64(seed) ^
		(uint64(x) * 0x9E3779B185EBCA87) ^
		(uint64(z) * 0xC2B2AE3D27D4EB4F)
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

func smoothstep(t float64) float64 {
	t = clamp(t, 0, 1)
	return t * t * (3 - 2*t)
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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sameGeometry(a, b world.GridGeometry) bool {
	return a.Width == b.Width &&
		a.Height == b.Height &&
		math.Abs(a.OriginX-b.OriginX) <= 1e-12 &&
		math.Abs(a.OriginZ-b.OriginZ) <= 1e-12 &&
		math.Abs(a.SpanXBlocks-b.SpanXBlocks) <= 1e-9 &&
		math.Abs(a.SpanZBlocks-b.SpanZBlocks) <= 1e-9
}
