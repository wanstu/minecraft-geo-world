package specialfeatures

import (
	"math"
	"sort"

	"geoworld/internal/ecology"
	"geoworld/internal/hydrology"
	"geoworld/internal/mcprofile"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/world"
	"geoworld/internal/worldsample"
)

type Kind string

const (
	MushroomIsland       Kind = "mushroom_island"
	PaleGardenIsland     Kind = "pale_garden_island"
	CherryIsland         Kind = "cherry_island"
	Sinkhole             Kind = "sinkhole"
	VolcanicCraterSnow   Kind = "volcanic_crater_snow"
	VolcanicCraterCherry Kind = "volcanic_crater_cherry"
	BasinCherry          Kind = "basin_cherry"
	BasinSnow            Kind = "basin_snow"
)

type Lobe struct {
	OffsetX float64 `json:"offsetX"`
	OffsetZ float64 `json:"offsetZ"`
	RadiusX float64 `json:"radiusX"`
	RadiusZ float64 `json:"radiusZ"`
}

type Feature struct {
	ID           int     `json:"id"`
	Kind         Kind    `json:"kind"`
	CenterX      float64 `json:"centerX"`
	CenterZ      float64 `json:"centerZ"`
	RadiusX      float64 `json:"radiusX"`
	RadiusZ      float64 `json:"radiusZ"`
	WaterContext string  `json:"waterContext,omitempty"`
	Lobes        []Lobe  `json:"lobes,omitempty"`
}

type candidate struct {
	index int
	x     float64
	z     float64
	score uint64
}

func Plan(
	seed int64,
	g world.GridGeometry,
	surface *world.Surface,
	hydro *hydrology.Model,
	profile *mcprofile.Model,
	runtime *worldsample.Model,
	rareMacroStrength float64,
) []Feature {
	if surface == nil || hydro == nil || profile == nil || runtime == nil {
		return nil
	}
	if rareMacroStrength <= 0 {
		rareMacroStrength = 1
	}
	area := math.Abs(g.SpanXBlocks * g.SpanZBlocks)
	scale := clamp(rareMacroStrength, 0.5, 2.5)
	base := clampInt(int(math.Round(area/300_000_000*scale)), 1, 4)

	ocean := collectOceanCandidates(seed, g, surface)
	land := collectLandCandidates(seed, g, surface, profile, runtime)
	lakes := collectLargeLakeCandidates(seed, g, hydro, runtime)

	features := make([]Feature, 0, base*6+8)
	nextID := 1
	used := make([]candidate, 0, 32)

	add := func(kind Kind, c candidate, radius float64, waterContext string, salt uint64) bool {
		minGap := math.Max(1800, radius*5)
		for _, previous := range used {
			if math.Hypot(c.x-previous.x, c.z-previous.z) < minGap {
				return false
			}
		}
		f := makeFeature(nextID, kind, c.x, c.z, radius, seed, salt, waterContext)
		nextID++
		features = append(features, f)
		used = append(used, c)
		return true
	}

	take := func(kind Kind, candidates []candidate, count int, radius float64, waterContext string, salt uint64) {
		added := 0
		for _, c := range candidates {
			if add(kind, c, radius*(0.86+0.28*hashUnit(c.score^salt)), waterContext, salt) {
				added++
				if added >= count {
					return
				}
			}
		}
	}

	if len(lakes) > 0 {
		take(MushroomIsland, lakes, 1, 180, "lake", 0x11)
		if len(lakes) > 1 {
			take(CherryIsland, lakes[1:], 1, 170, "lake", 0x17)
		}
	}
	take(MushroomIsland, ocean, base, 240, "ocean", 0x21)
	take(PaleGardenIsland, ocean, clampInt((base+1)/2, 1, 3), 220, "ocean", 0x31)
	take(CherryIsland, ocean, clampInt(base, 1, 4), 210, "ocean", 0x41)

	sinkCandidates := filterTerrain(land, profile,
		mcprofile.TerrainRollingPlain, mcprofile.TerrainHills, mcprofile.TerrainPlateau)
	craterCandidates := filterTerrain(land, profile,
		mcprofile.TerrainFoothills, mcprofile.TerrainMountain, mcprofile.TerrainPlateau)
	basinCandidates := filterTerrain(land, profile,
		mcprofile.TerrainPlain, mcprofile.TerrainRollingPlain, mcprofile.TerrainHills)

	take(Sinkhole, sinkCandidates, clampInt(base+1, 2, 6), 300, "", 0x51)

	craterAdded := 0
	for i, c := range craterCandidates {
		kind := VolcanicCraterCherry
		if i%2 == 0 {
			kind = VolcanicCraterSnow
		}
		if add(kind, c, 520*(0.9+0.25*hashUnit(c.score^0x61)), "", 0x61+uint64(i)) {
			craterAdded++
			if craterAdded >= clampInt(base, 1, 4) {
				break
			}
		}
	}

	basinAdded := 0
	for i, c := range basinCandidates {
		kind := BasinCherry
		if i%3 == 2 {
			kind = BasinSnow
		}
		if add(kind, c, 820*(0.9+0.25*hashUnit(c.score^0x71)), "", 0x71+uint64(i)) {
			basinAdded++
			if basinAdded >= clampInt(base, 1, 4) {
				break
			}
		}
	}

	sort.Slice(features, func(i, j int) bool { return features[i].ID < features[j].ID })
	return features
}

func collectOceanCandidates(seed int64, g world.GridGeometry, surface *world.Surface) []candidate {
	out := make([]candidate, 0, g.Width*g.Height/8)
	for z := 5; z < g.Height-5; z++ {
		for x := 5; x < g.Width-5; x++ {
			i := z*g.Width + x
			if i >= len(surface.LandMask) || surface.LandMask[i] || !oceanInterior(surface, g, x, z, 4) {
				continue
			}
			wx, wz := g.NodePosition(x, z)
			out = append(out, candidate{index: i, x: wx, z: wz, score: mix64(uint64(seed) ^ uint64(i)*0x9e3779b97f4a7c15 ^ 0xa0761d6478bd642f)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score < out[j].score })
	return out
}

func collectLandCandidates(seed int64, g world.GridGeometry, surface *world.Surface, profile *mcprofile.Model, runtime *worldsample.Model) []candidate {
	out := make([]candidate, 0, g.Width*g.Height/4)
	for z := 4; z < g.Height-4; z++ {
		for x := 4; x < g.Width-4; x++ {
			i := z*g.Width + x
			if i >= len(surface.LandMask) || !surface.LandMask[i] || !landInterior(surface, g, x, z, 3) {
				continue
			}
			wx, wz := g.NodePosition(x, z)
			if runtime.Sample(wx, wz).WaterKind != reconcile.WaterNone {
				continue
			}
			out = append(out, candidate{index: i, x: wx, z: wz, score: mix64(uint64(seed) ^ uint64(i)*0xbf58476d1ce4e5b9 ^ 0xe7037ed1a0b428db)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score < out[j].score })
	return out
}

func collectLargeLakeCandidates(seed int64, g world.GridGeometry, hydro *hydrology.Model, runtime *worldsample.Model) []candidate {
	type point struct {
		x float64
		z float64
	}
	type agg struct {
		points []point
		xsum   float64
		zsum   float64
	}
	groups := map[int]*agg{}
	for i, id := range hydro.LakeID {
		if id <= 0 {
			continue
		}
		x, z := i%g.Width, i/g.Width
		wx, wz := g.NodePosition(x, z)
		if runtime.Sample(wx, wz).WaterKind != reconcile.WaterLake {
			continue
		}
		a := groups[id]
		if a == nil {
			a = &agg{}
			groups[id] = a
		}
		a.points = append(a.points, point{x: wx, z: wz})
		a.xsum += wx
		a.zsum += wz
	}
	out := make([]candidate, 0, len(groups))
	for id, a := range groups {
		if len(a.points) < 8 {
			continue
		}
		cx := a.xsum / float64(len(a.points))
		cz := a.zsum / float64(len(a.points))
		best := a.points[0]
		bestDistance := math.Inf(1)
		for _, p := range a.points {
			d := math.Hypot(p.x-cx, p.z-cz)
			if d < bestDistance {
				best = p
				bestDistance = d
			}
		}
		out = append(out, candidate{
			index: id,
			x:     best.x,
			z:     best.z,
			score: mix64(uint64(seed) ^ uint64(id)*0x94d049bb133111eb ^ 0x8ebc6af09c88c6e3),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score < out[j].score })
	return out
}

func filterTerrain(in []candidate, profile *mcprofile.Model, allowed ...mcprofile.TerrainVariant) []candidate {
	set := map[mcprofile.TerrainVariant]bool{}
	for _, v := range allowed {
		set[v] = true
	}
	out := make([]candidate, 0, len(in))
	for _, c := range in {
		if c.index < 0 || c.index >= len(profile.TerrainVariant) || !set[profile.TerrainVariant[c.index]] {
			continue
		}
		out = append(out, c)
	}
	return out
}

func makeFeature(id int, kind Kind, x, z, radius float64, seed int64, salt uint64, waterContext string) Feature {
	h := mix64(uint64(seed) ^ uint64(id)*0xd6e8feb86659fd93 ^ salt)
	f := Feature{
		ID: id, Kind: kind, CenterX: x, CenterZ: z,
		RadiusX:      radius * (0.90 + 0.20*hashUnit(h)),
		RadiusZ:      radius * (0.82 + 0.30*hashUnit(h>>11)),
		WaterContext: waterContext,
	}
	if kind == MushroomIsland || kind == PaleGardenIsland || kind == CherryIsland {
		for i := 0; i < 3; i++ {
			v := mix64(h ^ uint64(i+1)*0x9e3779b97f4a7c15)
			angle := 2 * math.Pi * hashUnit(v)
			offset := radius * (0.18 + 0.18*hashUnit(v>>7))
			f.Lobes = append(f.Lobes, Lobe{
				OffsetX: math.Cos(angle) * offset,
				OffsetZ: math.Sin(angle) * offset,
				RadiusX: radius * (0.42 + 0.25*hashUnit(v>>17)),
				RadiusZ: radius * (0.38 + 0.28*hashUnit(v>>29)),
			})
		}
	}
	return f
}

func Apply(
	features []Feature,
	base runtimeprofile.Sample,
	seaLevel float64,
	x, z float64,
) runtimeprofile.Sample {
	best := 0.0
	var selected Feature
	for _, feature := range features {
		influence := Influence(feature, x, z)
		if influence > best {
			best = influence
			selected = feature
		}
	}
	if best <= 0 {
		return base
	}

	radial := NormalizedRadius(selected, x, z)
	out := base
	out.SpecialFeature = string(selected.Kind)
	out.SpecialFeatureStrength = best

	setLand := func(height float64) {
		out.World.GroundHeight = clamp(height, 0.001, 0.999)
		out.World.WaterKind = reconcile.WaterNone
		out.World.WaterSurfaceDefined = false
		out.World.WaterSurfaceElevation = 0
		out.World.WaterDepth = 0
		out.World.OceanEligibility = 0
		out.SourceLand = true
		out.SemanticFallback = false
		out.SemanticMismatch = false
	}

	switch selected.Kind {
	case MushroomIsland, PaleGardenIsland, CherryIsland:
		if base.World.WaterKind == reconcile.WaterRiver {
			return out
		}
		if base.World.WaterKind != reconcile.WaterOcean &&
			base.World.WaterKind != reconcile.WaterLake {
			return base
		}
		if best < 0.035 {
			return base
		}
		water := seaLevel
		if base.World.WaterSurfaceDefined {
			water = base.World.WaterSurfaceElevation
		}
		setLand(math.Max(base.World.GroundHeight, water+0.008+0.052*best))
		switch selected.Kind {
		case MushroomIsland:
			out.EcoRegion = ecology.RegionTemperateGrassland
			out.BiomeFamily = mcprofile.BiomeTemperateGrassland
			out.TerrainVariant = mcprofile.TerrainRollingPlain
			out.SurfaceProfile = mcprofile.SurfaceShrubSoil
			out.VegetationProfile = mcprofile.VegetationShrubland
		case PaleGardenIsland:
			out.EcoRegion = ecology.RegionTemperateForest
			out.BiomeFamily = mcprofile.BiomeTemperateForest
			out.TerrainVariant = mcprofile.TerrainHills
			out.SurfaceProfile = mcprofile.SurfaceForestLoam
			out.VegetationProfile = mcprofile.VegetationMixedForest
		case CherryIsland:
			out.EcoRegion = ecology.RegionTemperateForest
			out.BiomeFamily = mcprofile.BiomeTemperateForest
			out.TerrainVariant = mcprofile.TerrainHills
			out.SurfaceProfile = mcprofile.SurfaceForestLoam
			out.VegetationProfile = mcprofile.VegetationBroadleafForest
		}
	case Sinkhole:
		if base.World.WaterKind != reconcile.WaterNone || !base.SourceLand {
			return base
		}
		setLand(math.Max(seaLevel+0.008, base.World.GroundHeight-0.030*best))
		out.TerrainVariant = mcprofile.TerrainCanyon
	case VolcanicCraterSnow, VolcanicCraterCherry:
		if base.World.WaterKind != reconcile.WaterNone || !base.SourceLand || radial > 1.05 {
			return base
		}
		rim := clamp(1-math.Abs(radial-0.68)/0.20, 0, 1)
		core := clamp(1-radial/0.50, 0, 1)
		setLand(math.Max(seaLevel+0.010, base.World.GroundHeight+0.060*rim-0.028*core))
		out.SpecialFeatureStrength = math.Max(rim, core*0.6)
		if radial < 0.48 {
			out.TerrainVariant = mcprofile.TerrainCanyon
		} else {
			out.TerrainVariant = mcprofile.TerrainMountain
		}
		if selected.Kind == VolcanicCraterSnow && rim > 0.15 {
			out.EcoRegion = ecology.RegionAlpine
			out.BiomeFamily = mcprofile.BiomeAlpine
			out.SurfaceProfile = mcprofile.SurfaceAlpineSnow
			out.VegetationProfile = mcprofile.VegetationAlpineSparse
		}
		if selected.Kind == VolcanicCraterCherry && rim > 0.18 {
			out.EcoRegion = ecology.RegionTemperateForest
			out.BiomeFamily = mcprofile.BiomeTemperateForest
			out.SurfaceProfile = mcprofile.SurfaceForestLoam
			out.VegetationProfile = mcprofile.VegetationBroadleafForest
		}
	case BasinCherry, BasinSnow:
		if base.World.WaterKind != reconcile.WaterNone || !base.SourceLand || radial > 1.08 {
			return base
		}
		basin := clamp(1-radial*radial, 0, 1)
		rim := clamp(1-math.Abs(radial-0.90)/0.18, 0, 1)
		setLand(math.Max(seaLevel+0.008, base.World.GroundHeight-0.020*basin+0.030*rim))
		out.SpecialFeatureStrength = math.Max(basin*0.8, rim)
		if rim > 0.22 {
			out.TerrainVariant = mcprofile.TerrainMountain
		} else {
			out.TerrainVariant = mcprofile.TerrainRollingPlain
		}
		if selected.Kind == BasinSnow && rim > 0.20 {
			out.EcoRegion = ecology.RegionAlpine
			out.BiomeFamily = mcprofile.BiomeAlpine
			out.SurfaceProfile = mcprofile.SurfaceAlpineSnow
			out.VegetationProfile = mcprofile.VegetationAlpineSparse
		}
		if selected.Kind == BasinCherry && basin > 0.18 {
			out.EcoRegion = ecology.RegionTemperateForest
			out.BiomeFamily = mcprofile.BiomeTemperateForest
			out.SurfaceProfile = mcprofile.SurfaceForestLoam
			out.VegetationProfile = mcprofile.VegetationBroadleafForest
		}
	}
	return out
}

func Influence(f Feature, x, z float64) float64 {
	best := ellipseInfluence(x-f.CenterX, z-f.CenterZ, f.RadiusX, f.RadiusZ)
	for _, l := range f.Lobes {
		v := ellipseInfluence(x-(f.CenterX+l.OffsetX), z-(f.CenterZ+l.OffsetZ), l.RadiusX, l.RadiusZ)
		if v > best {
			best = v
		}
	}
	return best
}

func NormalizedRadius(f Feature, x, z float64) float64 {
	dx := (x - f.CenterX) / math.Max(1, f.RadiusX)
	dz := (z - f.CenterZ) / math.Max(1, f.RadiusZ)
	return math.Sqrt(dx*dx + dz*dz)
}

func ellipseInfluence(dx, dz, rx, rz float64) float64 {
	if rx <= 0 || rz <= 0 {
		return 0
	}
	q := 1 - dx*dx/(rx*rx) - dz*dz/(rz*rz)
	if q <= 0 {
		return 0
	}
	if q >= 1 {
		return 1
	}
	return q * q * (3 - 2*q)
}

func oceanInterior(surface *world.Surface, g world.GridGeometry, x, z, radius int) bool {
	for dz := -radius; dz <= radius; dz++ {
		for dx := -radius; dx <= radius; dx++ {
			i := (z+dz)*g.Width + (x + dx)
			if i < 0 || i >= len(surface.LandMask) || surface.LandMask[i] {
				return false
			}
		}
	}
	return true
}

func landInterior(surface *world.Surface, g world.GridGeometry, x, z, radius int) bool {
	for dz := -radius; dz <= radius; dz++ {
		for dx := -radius; dx <= radius; dx++ {
			i := (z+dz)*g.Width + (x + dx)
			if i < 0 || i >= len(surface.LandMask) || !surface.LandMask[i] {
				return false
			}
		}
	}
	return true
}

func mix64(x uint64) uint64 {
	x ^= x >> 30
	x *= 0xbf58476d1ce4e5b9
	x ^= x >> 27
	x *= 0x94d049bb133111eb
	x ^= x >> 31
	return x
}

func hashUnit(x uint64) float64 {
	return float64(x>>11) / float64(uint64(1)<<53)
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

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
