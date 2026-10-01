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
	gx := clamp((worldX-m.Geometry.OriginX)/spacingX, 0, float64(m.Geometry.Width-1))
	gz := clamp((worldZ-m.Geometry.OriginZ)/spacingZ, 0, float64(m.Geometry.Height-1))

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

	best := -1
	bestWeight := -1.0
	for _, c := range candidates {
		if m.Profile.LandMask[c.node] != wantLand {
			continue
		}
		if c.weight > bestWeight {
			best = c.node
			bestWeight = c.weight
		}
	}
	if best >= 0 {
		return best, bestWeight, false
	}

	// A valid final sample should always have at least one matching corner:
	// continuous ocean eligibility cannot produce Ocean inside an all-land cell,
	// and non-ocean samples inside mixed/all-land cells have a land corner.
	// Keep a deterministic fallback so diagnostics can expose violations without
	// making Runtime queries panic.
	for _, c := range candidates {
		if c.weight > bestWeight {
			best = c.node
			bestWeight = c.weight
		}
	}
	return best, bestWeight, true
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
