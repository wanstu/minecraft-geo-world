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
	BaseHeight   float64 `json:"baseHeight,omitempty"`
	FloorHeight  float64 `json:"floorHeight,omitempty"`
	RimHeight    float64 `json:"rimHeight,omitempty"`
	WaterContext string  `json:"waterContext,omitempty"`
	Lobes        []Lobe  `json:"lobes,omitempty"`
}

type candidate struct {
	index     int
	x         float64
	z         float64
	score     uint64
	clearance float64
}

func Plan(
	seed int64,
	g world.GridGeometry,
	surface *world.Surface,
	hydro *hydrology.Model,
	profile *mcprofile.Model,
	runtime *worldsample.Model,
	rareMacroStrength float64,
	rulesVersion string,
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
		if usesV112DepressionGeometry(rulesVersion) {
			center := runtime.Sample(c.x, c.z).GroundHeight
			f.BaseHeight = center
			if usesCarvedDepressionGeometry(rulesVersion) {
				switch kind {
				case Sinkhole:
					// v1.1.14+: carve the existing terrain; never build a rim.
					f.FloorHeight = math.Max(runtime.SeaLevelNormalized+0.014, center-0.075)
					f.RimHeight = center
				case VolcanicCraterSnow, VolcanicCraterCherry:
					// Craters cut into an existing peak rather than generating a cone.
					f.FloorHeight = math.Max(runtime.SeaLevelNormalized+0.014, center-0.055)
					f.RimHeight = center
				case BasinCherry, BasinSnow:
					// Basins are broad shallow cuts; surrounding natural relief remains.
					f.FloorHeight = math.Max(runtime.SeaLevelNormalized+0.014, center-0.045)
					f.RimHeight = center
				}
			} else {
				// v1.1.12-v1.1.13 locked geometry. Keep this byte-for-byte
				// semantic shape for old World Models.
				switch kind {
				case Sinkhole:
					f.FloorHeight = math.Max(runtime.SeaLevelNormalized+0.014, center-0.130)
					f.RimHeight = clamp(math.Max(center+0.012, f.FloorHeight+0.120), 0.001, 0.985)
				case VolcanicCraterSnow, VolcanicCraterCherry:
					f.FloorHeight = math.Max(runtime.SeaLevelNormalized+0.014, center-0.100)
					f.RimHeight = clamp(math.Max(center+0.080, f.FloorHeight+0.160), 0.001, 0.985)
				case BasinCherry, BasinSnow:
					f.FloorHeight = math.Max(runtime.SeaLevelNormalized+0.014, center-0.085)
					f.RimHeight = clamp(math.Max(center+0.070, f.FloorHeight+0.145), 0.001, 0.985)
				}
			}
		}
		nextID++
		features = append(features, f)
		used = append(used, c)
		return true
	}

	take := func(kind Kind, candidates []candidate, count int, radius float64, waterContext string, salt uint64) {
		added := 0
		for _, c := range candidates {
			candidateRadius := radius * (0.86 + 0.28*hashUnit(c.score^salt))
			if waterContext == "lake" && usesLakeIslandClearance(rulesVersion) {
				if c.clearance <= 0 {
					continue
				}
				candidateRadius = math.Min(candidateRadius, c.clearance*0.45)
				if candidateRadius < 48 {
					continue
				}
			}
			if add(kind, c, candidateRadius, waterContext, salt) {
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

	carved := usesCarvedDepressionGeometry(rulesVersion)
	sinkRadius := 300.0
	craterRadius := 520.0
	basinRadius := 820.0
	if carved {
		craterCandidates = filterLocalPeakCandidates(craterCandidates, runtime, 220)
		sinkRadius = 185
		craterRadius = 190
		basinRadius = 720
	}

	take(Sinkhole, sinkCandidates, clampInt(base+1, 2, 6), sinkRadius, "", 0x51)

	craterAdded := 0
	for i, c := range craterCandidates {
		kind := VolcanicCraterCherry
		if i%2 == 0 {
			kind = VolcanicCraterSnow
		}
		radius := craterRadius * (0.9 + 0.25*hashUnit(c.score^0x61))
		if carved {
			radius = craterRadius * (0.90 + 0.20*hashUnit(c.score^0x61))
		}
		if add(kind, c, radius, "", 0x61+uint64(i)) {
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
		radius := basinRadius * (0.9 + 0.25*hashUnit(c.score^0x71))
		if carved {
			radius = basinRadius * (0.90 + 0.22*hashUnit(c.score^0x71))
		}
		if add(kind, c, radius, "", 0x71+uint64(i)) {
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
		x  float64
		z  float64
		gx int
		gz int
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
		a.points = append(a.points, point{x: wx, z: wz, gx: x, gz: z})
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
		clearance := lakeClearanceBlocks(g, hydro, runtime, id, best.gx, best.gz)
		out = append(out, candidate{
			index:     id,
			x:         best.x,
			z:         best.z,
			score:     mix64(uint64(seed) ^ uint64(id)*0x94d049bb133111eb ^ 0x8ebc6af09c88c6e3),
			clearance: clearance,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].score < out[j].score })
	return out
}

func lakeClearanceBlocks(g world.GridGeometry, hydro *hydrology.Model, runtime *worldsample.Model, lakeID, centerX, centerZ int) float64 {
	spacingX := math.Abs(g.SpacingX())
	spacingZ := math.Abs(g.SpacingZ())
	if spacingX <= 0 || spacingZ <= 0 {
		return 0
	}
	best := math.Inf(1)
	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			i := z*g.Width + x
			sameLake := i < len(hydro.LakeID) && hydro.LakeID[i] == lakeID
			if sameLake {
				wx, wz := g.NodePosition(x, z)
				sameLake = runtime.Sample(wx, wz).WaterKind == reconcile.WaterLake
			}
			if sameLake {
				continue
			}
			dx := float64(x-centerX) * spacingX
			dz := float64(z-centerZ) * spacingZ
			d := math.Hypot(dx, dz)
			if d < best {
				best = d
			}
		}
	}
	if math.IsInf(best, 1) {
		return 0
	}
	// Node distance overestimates shoreline clearance by roughly half a macro cell.
	return math.Max(0, best-0.5*math.Max(spacingX, spacingZ))
}

func filterLocalPeakCandidates(
	in []candidate,
	runtime *worldsample.Model,
	radius float64,
) []candidate {
	if runtime == nil || radius <= 0 {
		return nil
	}
	out := make([]candidate, 0, len(in))
	const samples = 8
	for _, c := range in {
		center := runtime.Sample(c.x, c.z).GroundHeight
		sum := 0.0
		lower := 0
		valid := 0
		for i := 0; i < samples; i++ {
			angle := 2 * math.Pi * float64(i) / samples
			s := runtime.Sample(
				c.x+math.Cos(angle)*radius,
				c.z+math.Sin(angle)*radius,
			)
			if s.WaterKind != reconcile.WaterNone {
				continue
			}
			valid++
			sum += s.GroundHeight
			if center > s.GroundHeight+0.002 {
				lower++
			}
		}
		if valid < 6 {
			continue
		}
		average := sum / float64(valid)
		if lower >= 5 && center-average >= 0.006 {
			out = append(out, c)
		}
	}
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
	rulesVersion string,
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
	wallVariation := 0.5
	carvedDepression := usesCarvedDepressionGeometry(rulesVersion)
	if carvedDepression && isCarvedDepression(selected.Kind) {
		radial, wallVariation = depressionGeometry(selected, x, z)
	}
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
		if !usesV112DepressionGeometry(rulesVersion) || !hasDepressionHeights(selected) {
			setLand(math.Max(seaLevel+0.008, base.World.GroundHeight-0.030*best))
			out.TerrainVariant = mcprofile.TerrainCanyon
			break
		}
		floor := depressionFloorHeight(selected, base.World.GroundHeight, seaLevel)
		if carvedDepression {
			if radial > 1.0 {
				return base
			}
			inner := 0.56 + 0.05*(wallVariation-0.5)
			outer := inner + 0.14 + 0.10*wallVariation
			wall := smoothstepRange(inner, outer, radial)
			target := math.Min(base.World.GroundHeight, lerp(floor, base.World.GroundHeight, wall))
			setLand(target)
			if radial < inner {
				applyHabitableFloor(&out)
				out.SpecialFeatureStrength = 0.08 * (1 - radial/math.Max(inner, 0.01))
			} else if radial < outer {
				out.TerrainVariant = mcprofile.TerrainCanyon
				out.SpecialFeatureStrength = 1 - wall
			} else {
				return base
			}
			break
		}

		// v1.1.12-v1.1.13 locked interpretation.
		if radial > 0.90 {
			return base
		}
		wall := smoothstepRange(0.68, 0.82, radial)
		setLand(lerp(floor, base.World.GroundHeight, wall))
		if radial < 0.68 {
			applyHabitableFloor(&out)
			out.SpecialFeatureStrength = 0.08 * (1 - radial/0.68)
		} else {
			out.TerrainVariant = mcprofile.TerrainCanyon
			out.SpecialFeatureStrength = 1 - wall
		}
	case VolcanicCraterSnow, VolcanicCraterCherry:
		if base.World.WaterKind != reconcile.WaterNone || !base.SourceLand || radial > 1.05 {
			return base
		}
		if !usesV112DepressionGeometry(rulesVersion) || !hasDepressionHeights(selected) {
			// v1.1.11 locked interpretation.
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
				applySnowRim(&out)
			}
			if selected.Kind == VolcanicCraterCherry && rim > 0.18 {
				applyCherryRim(&out)
			}
			break
		}
		floor := depressionFloorHeight(selected, base.World.GroundHeight, seaLevel)
		if carvedDepression {
			inner := 0.40 + 0.05*(wallVariation-0.5)
			outer := 0.70 + 0.12*wallVariation
			wall := smoothstepRange(inner, outer, radial)
			target := math.Min(base.World.GroundHeight, lerp(floor, base.World.GroundHeight, wall))
			setLand(target)
			theme := clamp(1-math.Abs(radial-(inner+outer)*0.5)/math.Max(0.10, (outer-inner)*0.75), 0, 1)
			if radial < inner {
				applyHabitableFloor(&out)
				out.SpecialFeatureStrength = 0.08 * (1 - radial/math.Max(inner, 0.01))
			} else if radial < outer {
				out.SpecialFeatureStrength = theme
				if selected.Kind == VolcanicCraterSnow && theme > 0.18 {
					applySnowRim(&out)
				}
				if selected.Kind == VolcanicCraterCherry && theme > 0.18 {
					applyCherryRim(&out)
				}
			}
			break
		}

		// v1.1.12-v1.1.13 locked interpretation.
		innerWall := smoothstepRange(0.62, 0.76, radial)
		outerBlend := smoothstepRange(0.91, 1.04, radial)
		ringGround := lerp(floor, selected.RimHeight, innerWall)
		setLand(lerp(ringGround, base.World.GroundHeight, outerBlend))
		theme := clamp(1-math.Abs(radial-0.84)/0.18, 0, 1)
		if radial < 0.62 {
			applyHabitableFloor(&out)
			out.SpecialFeatureStrength = 0.08 * (1 - radial/0.62)
		} else {
			out.TerrainVariant = mcprofile.TerrainMountain
			out.SpecialFeatureStrength = theme
			if selected.Kind == VolcanicCraterSnow && theme > 0.18 {
				applySnowRim(&out)
			}
			if selected.Kind == VolcanicCraterCherry && theme > 0.18 {
				applyCherryRim(&out)
			}
		}
	case BasinCherry, BasinSnow:
		riverConforming := usesHydrologyConformingDepressions(rulesVersion) && base.World.WaterKind == reconcile.WaterRiver
		if (!riverConforming && base.World.WaterKind != reconcile.WaterNone) || !base.SourceLand || radial > 1.08 {
			return base
		}
		if !usesV112DepressionGeometry(rulesVersion) || !hasDepressionHeights(selected) {
			// v1.1.11 locked interpretation.
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
				applySnowRim(&out)
			}
			if selected.Kind == BasinCherry && basin > 0.18 {
				applyCherryRim(&out)
			}
			break
		}
		floor := depressionFloorHeight(selected, base.World.GroundHeight, seaLevel)
		if carvedDepression {
			inner := 0.54 + 0.06*(wallVariation-0.5)
			outer := 0.88 + 0.08*wallVariation
			wall := smoothstepRange(inner, outer, radial)
			target := math.Min(base.World.GroundHeight, lerp(floor, base.World.GroundHeight, wall))
			if riverConforming {
				depth := math.Max(base.World.WaterDepth, 0.002)
				surface := target + depth
				ground := target
				if rulesVersion == "geoworld-rules-v1.1.16" {
					// Deform the shared water datum, then translate the bed by
					// the same amount. Mapping each bed onto the dry floor and
					// adding depth creates a raised, crowned river instead.
					datum := base.World.GroundHeight + depth
					if base.World.WaterSurfaceDefined {
						datum = base.World.WaterSurfaceElevation
					}
					riverFloor := depressionFloorHeight(selected, datum, seaLevel)
					surface = math.Min(datum, lerp(riverFloor-0.002, datum, wall))
					ground = base.World.GroundHeight - math.Max(0, datum-surface)
				} else {
					// v1.1.15 locked interpretation.
					drop := math.Max(0, base.World.GroundHeight-target)
					if base.World.WaterSurfaceDefined {
						surface = math.Max(surface, base.World.WaterSurfaceElevation-drop)
					}
				}
				out.World.GroundHeight = clamp(ground, 0.001, 0.999)
				out.World.WaterKind = reconcile.WaterRiver
				out.World.WaterSurfaceDefined = true
				out.World.WaterSurfaceElevation = surface
				out.World.WaterDepth = math.Max(0.002, surface-out.World.GroundHeight)
			} else {
				setLand(target)
			}
			theme := clamp(1-math.Abs(radial-(inner+outer)*0.5)/math.Max(0.12, (outer-inner)*0.80), 0, 1)
			if radial < inner {
				applyHabitableFloor(&out)
				out.SpecialFeatureStrength = 0.08 * (1 - radial/math.Max(inner, 0.01))
			} else if radial < outer {
				out.SpecialFeatureStrength = theme
				if selected.Kind == BasinSnow && theme > 0.18 {
					applySnowRim(&out)
				}
				if selected.Kind == BasinCherry && theme > 0.18 {
					applyCherryRim(&out)
				}
			}
			break
		}

		// v1.1.12-v1.1.13 locked interpretation.
		innerWall := smoothstepRange(0.68, 0.80, radial)
		outerBlend := smoothstepRange(0.93, 1.07, radial)
		ringGround := lerp(floor, selected.RimHeight, innerWall)
		setLand(lerp(ringGround, base.World.GroundHeight, outerBlend))
		theme := clamp(1-math.Abs(radial-0.86)/0.18, 0, 1)
		if radial < 0.68 {
			applyHabitableFloor(&out)
			out.SpecialFeatureStrength = 0.08 * (1 - radial/0.68)
		} else {
			out.TerrainVariant = mcprofile.TerrainMountain
			out.SpecialFeatureStrength = theme
			if selected.Kind == BasinSnow && theme > 0.18 {
				applySnowRim(&out)
			}
			if selected.Kind == BasinCherry && theme > 0.18 {
				applyCherryRim(&out)
			}
		}
	}
	return out
}

func usesV112DepressionGeometry(version string) bool {
	return version == "geoworld-rules-v1.1.12" ||
		version == "geoworld-rules-v1.1.13" ||
		version == "geoworld-rules-v1.1.14" ||
		(version == "geoworld-rules-v1.1.15" || version == "geoworld-rules-v1.1.16")
}

func usesCarvedDepressionGeometry(version string) bool {
	return version == "geoworld-rules-v1.1.14" ||
		(version == "geoworld-rules-v1.1.15" || version == "geoworld-rules-v1.1.16")
}

func usesHydrologyConformingDepressions(version string) bool {
	return (version == "geoworld-rules-v1.1.15" || version == "geoworld-rules-v1.1.16")
}

func usesLakeIslandClearance(version string) bool {
	return (version == "geoworld-rules-v1.1.15" || version == "geoworld-rules-v1.1.16")
}

func hasDepressionHeights(f Feature) bool {
	return f.BaseHeight > 0 && f.FloorHeight > 0 && f.RimHeight > 0 &&
		f.FloorHeight < f.RimHeight
}

func depressionFloorHeight(f Feature, baseHeight, seaLevel float64) float64 {
	variation := clamp(baseHeight-f.BaseHeight, -0.050, 0.050) * 0.08
	return math.Max(seaLevel+0.014, f.FloorHeight+variation)
}

func applyHabitableFloor(out *runtimeprofile.Sample) {
	out.TerrainVariant = mcprofile.TerrainPlain
	switch out.BiomeFamily {
	case mcprofile.BiomeMediterraneanShrubland:
		out.EcoRegion = ecology.RegionMediterraneanShrubland
		out.SurfaceProfile = mcprofile.SurfaceShrubSoil
		out.VegetationProfile = mcprofile.VegetationShrubland
	case mcprofile.BiomeSavanna:
		out.EcoRegion = ecology.RegionSavanna
		out.SurfaceProfile = mcprofile.SurfaceDryGrass
		out.VegetationProfile = mcprofile.VegetationSavanna
	case mcprofile.BiomeTemperateGrassland:
		out.EcoRegion = ecology.RegionTemperateGrassland
		out.SurfaceProfile = mcprofile.SurfaceGrassLoam
		out.VegetationProfile = mcprofile.VegetationGrassland
	default:
		out.EcoRegion = ecology.RegionTemperateGrassland
		out.BiomeFamily = mcprofile.BiomeTemperateGrassland
		out.SurfaceProfile = mcprofile.SurfaceGrassLoam
		out.VegetationProfile = mcprofile.VegetationGrassland
	}
}

func applySnowRim(out *runtimeprofile.Sample) {
	out.EcoRegion = ecology.RegionAlpine
	out.BiomeFamily = mcprofile.BiomeAlpine
	out.SurfaceProfile = mcprofile.SurfaceAlpineSnow
	out.VegetationProfile = mcprofile.VegetationAlpineSparse
}

func applyCherryRim(out *runtimeprofile.Sample) {
	out.EcoRegion = ecology.RegionTemperateForest
	out.BiomeFamily = mcprofile.BiomeTemperateForest
	out.SurfaceProfile = mcprofile.SurfaceForestLoam
	out.VegetationProfile = mcprofile.VegetationBroadleafForest
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*clamp(t, 0, 1)
}

func smoothstepRange(lo, hi, value float64) float64 {
	if hi <= lo {
		if value >= hi {
			return 1
		}
		return 0
	}
	t := clamp((value-lo)/(hi-lo), 0, 1)
	return t * t * (3 - 2*t)
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

func isCarvedDepression(kind Kind) bool {
	switch kind {
	case Sinkhole, VolcanicCraterSnow, VolcanicCraterCherry, BasinCherry, BasinSnow:
		return true
	default:
		return false
	}
}

func depressionGeometry(f Feature, x, z float64) (float64, float64) {
	dx := (x - f.CenterX) / math.Max(1, f.RadiusX)
	dz := (z - f.CenterZ) / math.Max(1, f.RadiusZ)
	raw := math.Hypot(dx, dz)
	theta := math.Atan2(dz, dx)
	phase := math.Mod(float64(f.ID)*0.7548776662466927, 1.0) * 2 * math.Pi

	// Keep the feature completely inside the planned ellipse while breaking the
	// artificial perfect-ring silhouette. Different harmonics make different
	// sectors of the wall advance/recede by roughly 5-12%.
	shape := 0.94 +
		0.035*math.Sin(3*theta+phase) +
		0.025*math.Sin(5*theta-phase*0.7)
	shape = clamp(shape, 0.88, 1.0)
	radial := raw / shape

	// A second independent field changes wall width, so the entire perimeter
	// does not have one identical slope.
	wallVariation := 0.5 +
		0.30*math.Sin(4*theta+phase*1.37) +
		0.20*math.Sin(7*theta-phase*0.61)
	return radial, clamp(wallVariation, 0, 1)
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
