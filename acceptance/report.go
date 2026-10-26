package acceptance

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/runtimeworld"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
)

type ResultStatus string

const (
	StatusPass           ResultStatus = "PASS"
	StatusFail           ResultStatus = "FAIL"
	StatusNotImplemented ResultStatus = "NOT_IMPLEMENTED"
	StatusManualRequired ResultStatus = "MANUAL_REQUIRED"
)

type Point struct {
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Z       int    `json:"z"`
	TP      string `json:"tp"`
	Role    string `json:"role,omitempty"`
	Reason  string `json:"reason"`
	Water   string `json:"water"`
	Biome   string `json:"biome"`
	Terrain string `json:"terrain"`
}

type Result struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Requirement RequirementKind `json:"requirement"`
	Inspection  InspectionKind  `json:"inspection"`
	Method      string          `json:"method"`
	Status      ResultStatus    `json:"status"`
	Observe     string          `json:"observe"`
	Expected    string          `json:"expected"`
	Failure     string          `json:"failure"`
	Points      []Point         `json:"points,omitempty"`
	Note        string          `json:"note,omitempty"`
}

type Report struct {
	RegistryVersion         string   `json:"registryVersion"`
	Seed                    int64    `json:"seed"`
	WorldProfileVersion     string   `json:"worldProfileVersion"`
	GenerationRulesVersion  string   `json:"generationRulesVersion"`
	WorldModelVersion       string   `json:"worldModelVersion"`
	Preset                  string   `json:"preset"`
	ParameterSnapshotSHA256 string   `json:"parameterSnapshotSha256"`
	Results                 []Result `json:"results"`
}

func Plan(model *worldmodelio.RuntimeModel) (Report, error) {
	if model == nil {
		return Report{}, fmt.Errorf("acceptance: runtime model is required")
	}
	runtimeWorld, err := runtimeworld.NewReopened(model)
	if err != nil {
		return Report{}, err
	}
	entries, err := Entries()
	if err != nil {
		return Report{}, err
	}
	catalog := buildCatalog(model, runtimeWorld)
	results := make([]Result, 0, len(entries))
	for _, entry := range entries {
		status := StatusManualRequired
		note := ""
		implementedLater := entry.Requirement == RequirementLater &&
			laterFeatureImplemented(entry.ID)
		if entry.Requirement == RequirementLater && !implementedLater {
			status = StatusNotImplemented
			note = "v1.1 将此项标记为后期；当前明确报告 NOT_IMPLEMENTED，不静默省略。"
		}
		points := candidatePointsForEntry(catalog, entry)
		if !entryUsesWorldSamples(entry) {
			points = nil
		}
		if len(points) > 6 {
			points = points[:6]
		}
		points = assignPointRoles(entry, points)
		if implementedLater {
			if len(points) == 0 {
				note = "该后期功能已实现，但此 Seed 未实例化可定位候选；稀有功能允许单 Seed 缺席。"
			} else {
				note = "该后期功能已实现；候选点直接来自当前生成规则/持久化模型证据，仍需按通过标准人工或专项 verifier 验收。"
			}
		}
		if entry.Requirement != RequirementLater &&
			entryUsesWorldSamples(entry) &&
			len(points) == 0 {
			status = StatusFail
			note = "MISSING_SAMPLE：该 Seed 未定位到此普通核心/人工验收项的候选样本；按 v1.1 不能静默跳过。"
		}
		results = append(results, Result{
			ID:          entry.ID,
			Name:        entry.Name,
			Requirement: entry.Requirement,
			Inspection:  entry.Inspection,
			Method:      entry.Method,
			Status:      status,
			Observe:     "在候选点/路线中观察「" + entry.Name + "」是否满足文档定义的地理关系与连续性。",
			Expected:    entry.PassStandard,
			Failure:     "未满足以下通过标准，或稳定出现其相反现象：" + entry.PassStandard,
			Points:      points,
			Note:        note,
		})
	}
	return Report{
		RegistryVersion:         "geoworld-acceptance-v1.1",
		Seed:                    model.Manifest.Seed,
		WorldProfileVersion:     model.Manifest.WorldProfileVersion,
		GenerationRulesVersion:  model.Manifest.GenerationRulesVersion,
		WorldModelVersion:       model.Manifest.WorldModelVersion,
		Preset:                  model.Manifest.Preset,
		ParameterSnapshotSHA256: model.Manifest.ParameterSnapshotSHA256,
		Results:                 results,
	}, nil
}

var implementedLaterFeatures = map[string]struct{}{
	"FLOOD-01": {},
	"FLOOD-02": {},
	"FLOOD-03": {},
	"FLOOD-04": {},
	"RARE-01":  {},
	"RARE-02":  {},
	"RARE-03":  {},
	"RARE-04":  {},
	"RARE-05":  {},
}

func laterFeatureImplemented(id string) bool {
	_, ok := implementedLaterFeatures[id]
	return ok
}

func inspectionUsesWorldSamples(kind InspectionKind) bool {
	switch kind {
	case InspectionPoint, InspectionRoute, InspectionRegion, InspectionRelation,
		InspectionAutomaticAndManual:
		return true
	default:
		return false
	}
}

func entryUsesWorldSamples(entry Entry) bool {
	// PARAM-09 is a cross-preset/cross-seed regression gate. A coordinate from
	// one model cannot satisfy it, so it is handled by the batch verifier.
	if entry.ID == "PARAM-09" {
		return false
	}
	return inspectionUsesWorldSamples(entry.Inspection)
}

func candidatePointsForEntry(catalog candidateCatalog, entry Entry) []Point {
	if points, ok := catalog[entry.ID]; ok {
		return append([]Point(nil), points...)
	}
	prefix := strings.SplitN(entry.ID, "-", 2)[0]
	return append([]Point(nil), catalog[prefix]...)
}

func assignPointRoles(entry Entry, points []Point) []Point {
	if len(points) == 0 {
		return points
	}
	out := append([]Point(nil), points...)
	prefix := strings.SplitN(entry.ID, "-", 2)[0]
	if entry.Inspection != InspectionRoute {
		for i := range out {
			out[i].Role = fmt.Sprintf("候选样本 P%d", i+1)
		}
		return out
	}
	for i := range out {
		role := "过渡/中段"
		if i == 0 {
			role = "起点/上游或内侧对照"
		} else if i == len(out)-1 {
			role = "终点/下游或外侧对照"
		}
		if prefix == "RIVER" || prefix == "EST" {
			switch {
			case i == 0:
				role = "上游/河源侧"
			case i == len(out)-1:
				role = "下游/河口侧"
			case i*3 < len(out):
				role = "中上游"
			case i*3 < 2*len(out):
				role = "中游/汇流观察"
			default:
				role = "下游"
			}
		} else if prefix == "EDGE" || prefix == "POLAR" || prefix == "SEA" {
			switch {
			case i == 0:
				role = "近侧/过渡起点"
			case i == len(out)-1:
				role = "远侧/终点"
			default:
				role = "过渡带"
			}
		}
		out[i].Role = fmt.Sprintf("P%d %s", i+1, role)
	}
	return out
}

type candidateCatalog map[string][]Point

func buildCatalog(
	model *worldmodelio.RuntimeModel,
	runtimeWorld *runtimeworld.Model,
) candidateCatalog {
	out := make(candidateCatalog)
	g := model.Geometry
	representativeTerrain := map[mcprofile.TerrainVariant]Point{}
	representativeBiome := map[mcprofile.BiomeFamily]Point{}
	representativeSurface := map[mcprofile.SurfaceProfile]Point{}
	representativeVegetation := map[mcprofile.VegetationProfile]Point{}

	var spawn Point
	spawnSet := false
	centerX := g.OriginX + g.SpanXBlocks*0.5
	centerZ := g.OriginZ + g.SpanZBlocks*0.5
	bestSpawn := math.Inf(1)

	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			wx, wz := g.NodePosition(x, z)
			s := runtimeWorld.Sample(wx, wz)
			p := makePoint(runtimeWorld, wx, wz, "macro-model representative")
			if _, ok := representativeTerrain[s.TerrainVariant]; !ok {
				representativeTerrain[s.TerrainVariant] = p
			}
			if _, ok := representativeBiome[s.BiomeFamily]; !ok {
				representativeBiome[s.BiomeFamily] = p
			}
			if _, ok := representativeSurface[s.SurfaceProfile]; !ok {
				representativeSurface[s.SurfaceProfile] = p
			}
			if _, ok := representativeVegetation[s.VegetationProfile]; !ok {
				representativeVegetation[s.VegetationProfile] = p
			}
			if s.SourceLand && s.World.WaterKind == reconcile.WaterNone {
				terrainPenalty := 1.0
				if s.TerrainVariant == mcprofile.TerrainPlain ||
					s.TerrainVariant == mcprofile.TerrainRollingPlain {
					terrainPenalty = 0
				}
				dx := (wx - centerX) / math.Max(1, g.SpanXBlocks)
				dz := (wz - centerZ) / math.Max(1, g.SpanZBlocks)
				score := dx*dx + dz*dz + terrainPenalty*0.05
				if score < bestSpawn {
					bestSpawn = score
					spawn = makePoint(runtimeWorld, wx, wz, "habitable mainland spawn candidate")
					spawnSet = true
				}
			}
		}
	}

	add := func(prefix string, points ...Point) {
		for _, p := range points {
			if p.TP != "" {
				out[prefix] = appendUniquePoint(out[prefix], p)
			}
		}
	}
	if spawnSet {
		add("CORE", spawn)
		add("MACRO", spawn)
		add("SPAWN", spawn)
		add("VSTRUCT", spawn)
		add("CAVE", spawn)
	}

	terrain := func(kind mcprofile.TerrainVariant) Point {
		return representativeTerrain[kind]
	}
	add("MTN", terrain(mcprofile.TerrainMountain), terrain(mcprofile.TerrainFoothills))
	add("HILL", terrain(mcprofile.TerrainHills), terrain(mcprofile.TerrainRollingPlain))
	add("PLATEAU", findPlateauRegionPoints(model, runtimeWorld)...)
	add("BASIN", terrain(mcprofile.TerrainPlain), terrain(mcprofile.TerrainFloodplain))
	add("PLAIN", terrain(mcprofile.TerrainPlain), terrain(mcprofile.TerrainAlluvialPlain), terrain(mcprofile.TerrainFloodplain))
	add("CORRIDOR", terrain(mcprofile.TerrainFloodplain), terrain(mcprofile.TerrainHills), terrain(mcprofile.TerrainFoothills))

	if coast, ok := findCoast(model, runtimeWorld); ok {
		add("COAST", coast)
		add("MACRO", coast)
		add("EDGE", coast)
	}
	if riverPoints := findRiverPoints(model, runtimeWorld); len(riverPoints) > 0 {
		add("RIVER", riverPoints...)
		add("CORRIDOR", riverPoints...)
		add("NODE", riverPoints[len(riverPoints)-1])
	}
	for _, candidate := range model.Waterbed.AcceptanceFluvialCandidates() {
		point := makePoint(
			runtimeWorld,
			candidate.X,
			candidate.Z,
			fmt.Sprintf(
				"v1.1.7 fluvial %s candidate from runtime generation rule (signal=%.4f)",
				candidate.Kind,
				candidate.Strength,
			),
		)
		switch candidate.Kind {
		case "floodplain":
			add("FLOOD-01", point)
		case "terrace":
			add("FLOOD-02", point)
		case "oxbow":
			add("FLOOD-03", point)
		case "levee":
			add("FLOOD-04", point)
		}
	}
	if mouth, ok := findOceanMouth(model, runtimeWorld); ok {
		add("EST", mouth)
		add("NODE", mouth)
		add("COAST", mouth)
	}
	if lake, ok := findLargestLake(model, runtimeWorld); ok {
		add("LAKE", lake)
		add("NODE", lake)
	}
	wetlandPoints := make([]Point, 0, 5)
	if wet, ok := representativeBiome[mcprofile.BiomeWetland]; ok {
		wet.Reason = "wetland biome representative"
		wetlandPoints = appendUniquePoint(wetlandPoints, wet)
	}
	for _, surfaceKind := range []mcprofile.SurfaceProfile{
		mcprofile.SurfacePeat,
		mcprofile.SurfaceWetMud,
	} {
		if wet, ok := representativeSurface[surfaceKind]; ok {
			wet.Reason = "natural wetland surface representative: " + surfaceKind.String()
			wetlandPoints = appendUniquePoint(wetlandPoints, wet)
		}
	}
	if wet, ok := representativeVegetation[mcprofile.VegetationWetland]; ok {
		wet.Reason = "wetland vegetation representative"
		wetlandPoints = appendUniquePoint(wetlandPoints, wet)
	}
	if len(wetlandPoints) > 0 {
		if floodplain := terrain(mcprofile.TerrainFloodplain); floodplain.TP != "" {
			floodplain.Reason = "water-system lowland comparison near wetland candidates"
			wetlandPoints = appendUniquePoint(wetlandPoints, floodplain)
		}
		add("WET", wetlandPoints...)
	}
	if forest := forestPoints(representativeVegetation); len(forest) > 0 {
		add("FOREST", forest...)
	}
	if polar, ok := representativeBiome[mcprofile.BiomeTundra]; ok {
		polar.Reason = "high-latitude tundra representative"
		add("POLAR", polar)
	}

	biomeKeys := make([]int, 0, len(representativeBiome))
	for k := range representativeBiome {
		biomeKeys = append(biomeKeys, int(k))
	}
	sort.Ints(biomeKeys)
	for _, key := range biomeKeys {
		p := representativeBiome[mcprofile.BiomeFamily(key)]
		p.Reason = "biome family representative: " + p.Biome
		add("BIOME", p)
	}

	surfaceKeys := make([]int, 0, len(representativeSurface))
	for k := range representativeSurface {
		surfaceKeys = append(surfaceKeys, int(k))
	}
	sort.Ints(surfaceKeys)
	for _, key := range surfaceKeys {
		p := representativeSurface[mcprofile.SurfaceProfile(key)]
		p.Reason = "surface-profile representative"
		add("SURFACE", p)
	}

	if climatePoints := climateExtremes(model, runtimeWorld); len(climatePoints) > 0 {
		add("CLIMATE", climatePoints...)
	}
	if confluence, ok := findConfluence(model, runtimeWorld); ok {
		add("NODE", confluence)
		add("RIVER", confluence)
	}
	if crossing, ok := findWatershedCrossing(model, runtimeWorld); ok {
		add("CORRIDOR", crossing)
		add("NODE", crossing)
	}

	transition := math.Max(
		2048,
		0.5*(g.SpacingX()+g.SpacingZ())*24*
			model.Config.WorldProfile.Parameters.TransitionWidth,
	)
	nearOcean := makePoint(
		runtimeWorld,
		g.OriginX+g.SpanXBlocks+0.30*transition,
		g.OriginZ+g.SpanZBlocks*0.5,
		"near exterior-ocean transition sample",
	)
	midOcean := makePoint(
		runtimeWorld,
		g.OriginX+g.SpanXBlocks+1.20*transition,
		g.OriginZ+g.SpanZBlocks*0.5,
		"mid exterior-ocean transition sample",
	)
	farOcean := makePoint(
		runtimeWorld,
		g.OriginX+g.SpanXBlocks+3*transition,
		g.OriginZ+g.SpanZBlocks*0.5,
		"far-ocean exterior sample",
	)
	polarOcean := makePoint(
		runtimeWorld,
		g.OriginX+g.SpanXBlocks*0.5,
		g.OriginZ-2*transition,
		"north frozen-ocean exterior sample",
	)
	add("SEA", nearOcean, midOcean, farOcean, polarOcean)
	add("POLAR", polarOcean)
	add("EDGE", nearOcean, midOcean, farOcean, polarOcean)
	if island, ok := findOuterIsland(runtimeWorld, g, transition); ok {
		add("ISLAND-01", island)
		add("ISLAND-02", island)
		add("EDGE", island)
	}
	// Density/far-ocean checks remain valid even when a particular seed has no
	// outer-island sample; absence is itself evidence of sparsity, not a missing
	// sample for ISLAND-02/03.
	add("ISLAND-02", nearOcean, midOcean, farOcean)
	add("ISLAND-03", midOcean, farOcean, polarOcean)
	add("MACRO", farOcean)

	if point, ok := findRareRiftPoint(model, runtimeWorld); ok {
		add("RARE-01", point)
	}
	if point, ok := findRareTerrainPoint(
		model,
		runtimeWorld,
		"giant valley",
		func(s rareTerrainSignals) float64 { return s.Carve },
		0.0025,
	); ok {
		add("RARE-02", point)
	}
	if point, ok := findRareMacroLakePoint(model, runtimeWorld); ok {
		add("RARE-03", point)
	}
	if point, ok := findRareTerrainPoint(
		model,
		runtimeWorld,
		"isolated peak",
		func(s rareTerrainSignals) float64 { return s.Lift },
		0.0030,
	); ok {
		add("RARE-04", point)
	}
	if point, ok := findRareTerrainPoint(
		model,
		runtimeWorld,
		"large escarpment",
		func(s rareTerrainSignals) float64 { return math.Abs(s.Escarpment) },
		0.0025,
	); ok {
		add("RARE-05", point)
	}

	// Ore placement is Minecraft-side, but these natural regions are useful
	// starting locations for the dedicated Java mineral verifier and manual TP.
	add("ORE", terrain(mcprofile.TerrainMountain), terrain(mcprofile.TerrainPlain))

	return out
}

type rareTerrainSignals struct {
	Lift       float64
	Carve      float64
	Escarpment float64
}

func findRareTerrainPoint(
	model *worldmodelio.RuntimeModel,
	rw *runtimeworld.Model,
	label string,
	score func(rareTerrainSignals) float64,
	minScore float64,
) (Point, bool) {
	if model == nil || model.CoastTerrain == nil || rw == nil || score == nil {
		return Point{}, false
	}
	g := model.Geometry
	bestScore := 0.0
	bestX := 0.0
	bestZ := 0.0
	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			wx, wz := g.NodePosition(x, z)
			s := model.CoastTerrain.Sample(wx, wz)
			value := score(rareTerrainSignals{
				Lift:       s.RareMacroLift,
				Carve:      s.RareMacroCarve,
				Escarpment: s.RareEscarpment,
			})
			if value > bestScore {
				bestScore = value
				bestX = wx
				bestZ = wz
			}
		}
	}
	if bestScore < minScore {
		return Point{}, false
	}
	return makePoint(
		rw,
		bestX,
		bestZ,
		fmt.Sprintf(
			"v1.1.7 rare macro %s candidate from persisted Terrain signal=%.5f",
			label,
			bestScore,
		),
	), true
}

func findRareRiftPoint(
	model *worldmodelio.RuntimeModel,
	rw *runtimeworld.Model,
) (Point, bool) {
	if model == nil || model.TectonicQuery == nil || rw == nil {
		return Point{}, false
	}
	q := model.TectonicQuery
	best := -1
	bestScore := 0.0
	for i := range q.Feature {
		score := 0.0
		if q.Feature[i] == phase1.ContinentalRift {
			score = q.Intensity[i]
		}
		if q.InfluenceFeature[i] == phase1.ContinentalRift {
			score = math.Max(score, q.Influence[i])
		}
		// Rare-macro strengthening only begins materially on the upper part of
		// the ordinary rift-intensity range. Lower values are normal rifts, not
		// RARE-01 landmarks.
		if score >= 0.68 && score > bestScore {
			best = i
			bestScore = score
		}
	}
	if best < 0 {
		return Point{}, false
	}
	x := best % model.Geometry.Width
	z := best / model.Geometry.Width
	wx, wz := model.Geometry.NodePosition(x, z)
	return makePoint(
		rw,
		wx,
		wz,
		fmt.Sprintf(
			"high-intensity ContinentalRift candidate from persisted Phase1 tectonic query (signal=%.4f)",
			bestScore,
		),
	), true
}

func findRareMacroLakePoint(
	model *worldmodelio.RuntimeModel,
	rw *runtimeworld.Model,
) (Point, bool) {
	if model == nil || model.Waterbed == nil || rw == nil {
		return Point{}, false
	}
	baseline := model.Geometry.SpacingX() * 0.72
	if baseline <= 0 {
		return Point{}, false
	}
	bestScale := 1.0
	best := -1
	for i, node := range model.Waterbed.Lakes {
		scale := node.OuterRadius / baseline
		if scale > bestScale {
			bestScale = scale
			best = i
		}
	}
	if best < 0 || bestScale < 1.10 {
		return Point{}, false
	}
	node := model.Waterbed.Lakes[best]
	point, ok := findRenderedLakePoint(
		rw,
		node,
		fmt.Sprintf(
			"rare macro lake/inland-sea candidate from persisted radius scale %.2fx (LakeID=%d)",
			bestScale,
			node.LakeID,
		),
	)
	return point, ok
}

func makePoint(runtimeWorld *runtimeworld.Model, x, z float64, reason string) Point {
	s := runtimeWorld.Sample(x, z)
	top := s.World.GroundHeight
	if s.World.WaterSurfaceDefined && s.World.WaterSurfaceElevation > top {
		top = s.World.WaterSurfaceElevation
	}
	y := boundaryY(top) + 8
	bx := int(math.Floor(x))
	bz := int(math.Floor(z))
	return Point{
		X: bx, Y: y, Z: bz,
		TP:      fmt.Sprintf("/tp @s %d %d %d", bx, y, bz),
		Reason:  reason,
		Water:   waterName(s.World.WaterKind),
		Biome:   s.BiomeFamily.String(),
		Terrain: s.TerrainVariant.String(),
	}
}

func boundaryY(h float64) int {
	h = math.Max(0, math.Min(1, h))
	var mapped float64
	if h <= 0.5 {
		mapped = -64 + (63-(-64))*(h/0.5)
	} else {
		mapped = 63 + (320-63)*((h-0.5)/0.5)
	}
	y := int(math.Floor(mapped + 0.5))
	if y < -64 {
		return -64
	}
	if y > 320 {
		return 320
	}
	return y
}

func findCoast(model *worldmodelio.RuntimeModel, rw *runtimeworld.Model) (Point, bool) {
	g := model.Geometry
	for z := 0; z < g.Height; z++ {
		for x := 0; x+1 < g.Width; x++ {
			a := z*g.Width + x
			b := a + 1
			if model.Surface.LandMask[a] == model.Surface.LandMask[b] {
				continue
			}
			wx := g.OriginX + (float64(x)+0.5)*g.SpacingX()
			wz := g.OriginZ + float64(z)*g.SpacingZ()
			return makePoint(rw, wx, wz, "formal land/ocean coast transition"), true
		}
	}
	return Point{}, false
}

func findLargestLake(model *worldmodelio.RuntimeModel, rw *runtimeworld.Model) (Point, bool) {
	if len(model.Waterbed.Lakes) == 0 {
		return Point{}, false
	}

	// A persisted lake node is only a macro hydrology candidate. Detailed
	// terrain and spill-capping decide which parts of the footprint are
	// actually water. Rank by connected LakeID area, then return a point that
	// Natural Model itself classifies as Lake instead of blindly using a node
	// center that may be occupied by an outlet river or dry high ground.
	groups := make(map[int][]int)
	for i, lake := range model.Waterbed.Lakes {
		groups[lake.LakeID] = append(groups[lake.LakeID], i)
	}
	type lakeGroup struct {
		id    int
		nodes []int
	}
	ordered := make([]lakeGroup, 0, len(groups))
	for id, nodes := range groups {
		ordered = append(ordered, lakeGroup{id: id, nodes: nodes})
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if len(ordered[i].nodes) != len(ordered[j].nodes) {
			return len(ordered[i].nodes) > len(ordered[j].nodes)
		}
		return ordered[i].id < ordered[j].id
	})

	for _, group := range ordered {
		for _, nodeID := range group.nodes {
			node := model.Waterbed.Lakes[nodeID]
			if point, ok := findRenderedLakePoint(
				rw,
				node,
				fmt.Sprintf(
					"rendered lake-water sample from largest persisted lake group LakeID=%d nodes=%d",
					group.id,
					len(group.nodes),
				),
			); ok {
				return point, true
			}
		}
	}
	return Point{}, false
}

func findRenderedLakePoint(
	rw *runtimeworld.Model,
	node waterbed.LakeNode,
	reason string,
) (Point, bool) {
	if rw == nil {
		return Point{}, false
	}

	// Center first, then concentric rings inside the persisted footprint. The
	// ring search handles a river/outlet crossing the macro center and detailed
	// terrain trimming one side of the lake.
	candidates := [][2]float64{{node.X, node.Z}}
	for _, radiusFraction := range []float64{0.20, 0.40, 0.60, 0.78} {
		radius := node.InnerRadius * radiusFraction
		if radius <= 0 {
			continue
		}
		for i := 0; i < 16; i++ {
			angle := 2 * math.Pi * float64(i) / 16
			candidates = append(candidates, [2]float64{
				node.X + radius*math.Cos(angle),
				node.Z + radius*math.Sin(angle),
			})
		}
	}

	for _, candidate := range candidates {
		sample := rw.Sample(candidate[0], candidate[1])
		if sample.World.WaterKind == reconcile.WaterLake {
			return makePoint(rw, candidate[0], candidate[1], reason), true
		}
	}
	return Point{}, false
}

func findRiverPoints(model *worldmodelio.RuntimeModel, rw *runtimeworld.Model) []Point {
	rivers := model.Waterbed.Rivers
	if len(rivers) == 0 {
		return nil
	}

	terminal := -1
	for i := range rivers {
		if rivers[i].OceanMouth {
			if terminal < 0 || rivers[i].HalfWidth > rivers[terminal].HalfWidth {
				terminal = i
			}
		}
	}
	if terminal < 0 {
		best := 0
		bestLen := math.Hypot(rivers[0].X1-rivers[0].X0, rivers[0].Z1-rivers[0].Z0)
		for i := 1; i < len(rivers); i++ {
			l := math.Hypot(rivers[i].X1-rivers[i].X0, rivers[i].Z1-rivers[i].Z0)
			if l > bestLen {
				best, bestLen = i, l
			}
		}
		seg := rivers[best]
		return []Point{
			makePoint(rw, seg.X0, seg.Z0, "main-river fallback route start"),
			makePoint(rw, 0.5*(seg.X0+seg.X1), 0.5*(seg.Z0+seg.Z1), "main-river fallback route middle"),
			makePoint(rw, seg.X1, seg.Z1, "main-river fallback route end"),
		}
	}

	chain := []int{terminal}
	used := map[int]bool{terminal: true}
	currentNode := rivers[terminal].From
	for len(chain) < 128 {
		best := -1
		for i := range rivers {
			if used[i] || rivers[i].To != currentNode {
				continue
			}
			if best < 0 || rivers[i].HalfWidth > rivers[best].HalfWidth {
				best = i
			}
		}
		if best < 0 {
			break
		}
		used[best] = true
		chain = append(chain, best)
		currentNode = rivers[best].From
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	lengths := make([]float64, len(chain))
	totalLength := 0.0
	for i, id := range chain {
		seg := rivers[id]
		length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
		lengths[i] = length
		totalLength += length
	}

	out := make([]Point, 0, 5)
	first := rivers[chain[0]]
	x, z := riverCenterPoint(first, 0)
	out = appendUniquePoint(out, makePoint(rw, x, z, "main-river chain source/upstream start"))
	for _, f := range []float64{0.25, 0.50, 0.75} {
		target := totalLength * f
		accumulated := 0.0
		segIndex := len(chain) - 1
		localT := 1.0
		for i, length := range lengths {
			if length <= 1e-9 {
				continue
			}
			if target <= accumulated+length || i == len(chain)-1 {
				segIndex = i
				localT = (target - accumulated) / length
				localT = math.Max(0, math.Min(1, localT))
				break
			}
			accumulated += length
		}
		seg := rivers[chain[segIndex]]
		x, z := riverCenterPoint(seg, localT)
		out = appendUniquePoint(out, makePoint(
			rw,
			x,
			z,
			fmt.Sprintf("main-river chain %.0f%% curved-centerline station", f*100),
		))
	}
	mouth := rivers[terminal]
	x, z = riverCenterPoint(mouth, 1)
	out = appendUniquePoint(out, makePoint(rw, x, z, "main-river ocean mouth"))
	return out
}

func riverCenterPoint(seg waterbed.RiverSegment, t float64) (float64, float64) {
	return waterbed.RiverCenterPoint(seg, t)
}

func findOceanMouth(model *worldmodelio.RuntimeModel, rw *runtimeworld.Model) (Point, bool) {
	for _, seg := range model.Waterbed.Rivers {
		if !seg.OceanMouth {
			continue
		}
		return makePoint(rw, seg.X1, seg.Z1, "ocean-mouth / estuary candidate"), true
	}
	return Point{}, false
}

func findConfluence(model *worldmodelio.RuntimeModel, rw *runtimeworld.Model) (Point, bool) {
	rivers := model.Waterbed.Rivers
	if len(rivers) == 0 || rw == nil {
		return Point{}, false
	}

	incoming := make(map[int][]int)
	outgoing := make(map[int][]int)
	for i, seg := range rivers {
		incoming[seg.To] = append(incoming[seg.To], i)
		outgoing[seg.From] = append(outgoing[seg.From], i)
	}

	type candidate struct {
		node     int
		incoming []int
		score    float64
	}
	candidates := make([]candidate, 0)
	for node, ids := range incoming {
		if len(ids) < 2 || len(outgoing[node]) == 0 {
			continue
		}
		score := float64(len(ids)) * 1000
		for _, id := range ids {
			score += rivers[id].HalfWidth
		}
		for _, id := range outgoing[node] {
			score += 0.5 * rivers[id].HalfWidth
		}
		candidates = append(candidates, candidate{
			node:     node,
			incoming: ids,
			score:    score,
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	// Prefer a confluence whose final rendered Natural Model is still River,
	// not a coarse-flow junction already swallowed by an ocean or lake.
	for _, want := range []reconcile.WaterKind{
		reconcile.WaterRiver,
		reconcile.WaterLake,
	} {
		for _, candidate := range candidates {
			if candidate.node < 0 ||
				candidate.node >= model.Geometry.Width*model.Geometry.Height {
				continue
			}
			x := candidate.node % model.Geometry.Width
			z := candidate.node / model.Geometry.Width
			wx, wz := model.Geometry.NodePosition(x, z)
			if rw.Sample(wx, wz).World.WaterKind != want {
				continue
			}
			return makePoint(
				rw,
				wx,
				wz,
				fmt.Sprintf(
					"rendered confluence with %d incoming river channels",
					len(candidate.incoming),
				),
			), true
		}
	}
	return Point{}, false
}

func findWatershedCrossing(model *worldmodelio.RuntimeModel, rw *runtimeworld.Model) (Point, bool) {
	h := model.HydrologyQuery
	if h == nil {
		return Point{}, false
	}
	g := model.Geometry
	for z := 0; z < g.Height; z++ {
		for x := 0; x+1 < g.Width; x++ {
			a := z*g.Width + x
			b := a + 1
			if h.BasinID[a] > 0 && h.BasinID[b] > 0 && h.BasinID[a] != h.BasinID[b] {
				wx := g.OriginX + (float64(x)+0.5)*g.SpacingX()
				wz := g.OriginZ + float64(z)*g.SpacingZ()
				return makePoint(rw, wx, wz, "adjacent persisted watershed IDs; low-crossing inspection candidate"), true
			}
		}
	}
	return Point{}, false
}

func climateExtremes(model *worldmodelio.RuntimeModel, rw *runtimeworld.Model) []Point {
	c := model.ClimatePreview
	if c == nil || len(c.AnnualMeanTempC) == 0 {
		return nil
	}
	type ext struct {
		idx int
		v   float64
	}
	hot, cold := ext{0, c.AnnualMeanTempC[0]}, ext{0, c.AnnualMeanTempC[0]}
	wet, dry := ext{0, c.AnnualPrecipMM[0]}, ext{0, c.AnnualPrecipMM[0]}
	for i := 1; i < len(c.AnnualMeanTempC); i++ {
		if c.AnnualMeanTempC[i] > hot.v {
			hot = ext{i, c.AnnualMeanTempC[i]}
		}
		if c.AnnualMeanTempC[i] < cold.v {
			cold = ext{i, c.AnnualMeanTempC[i]}
		}
		if c.AnnualPrecipMM[i] > wet.v {
			wet = ext{i, c.AnnualPrecipMM[i]}
		}
		if c.AnnualPrecipMM[i] < dry.v {
			dry = ext{i, c.AnnualPrecipMM[i]}
		}
	}
	out := []Point{}
	for _, item := range []struct {
		e      ext
		reason string
	}{
		{hot, "hottest persisted climate node"},
		{cold, "coldest persisted climate node"},
		{wet, "wettest persisted climate node"},
		{dry, "driest persisted climate node"},
	} {
		x := item.e.idx % model.Geometry.Width
		z := item.e.idx / model.Geometry.Width
		wx, wz := model.Geometry.NodePosition(x, z)
		out = appendUniquePoint(out, makePoint(rw, wx, wz, item.reason))
	}
	return out
}

func forestPoints(m map[mcprofile.VegetationProfile]Point) []Point {
	var out []Point
	for _, kind := range []mcprofile.VegetationProfile{
		mcprofile.VegetationConiferForest,
		mcprofile.VegetationMixedForest,
		mcprofile.VegetationBroadleafForest,
		mcprofile.VegetationDenseRainforest,
		mcprofile.VegetationMonsoonForest,
	} {
		if p, ok := m[kind]; ok {
			p.Reason = "forest vegetation representative"
			out = append(out, p)
		}
	}
	return out
}

func findPlateauRegionPoints(
	model *worldmodelio.RuntimeModel,
	rw *runtimeworld.Model,
) []Point {
	if model == nil || model.CoastTerrain == nil {
		return nil
	}
	g := model.Geometry
	values := model.CoastTerrain.PlateauWeight.Values
	if len(values) != g.Width*g.Height {
		return nil
	}

	type candidate struct {
		index  int
		weight float64
	}
	candidates := make([]candidate, 0, 64)
	for i, weight := range values {
		// PlateauWeight is a normalized blend weight, not a final terrain label.
		// Keep the candidate floor below the old 0.22 cliff so low-intensity but
		// spatially coherent plateaus remain inspectable across official presets.
		if weight < 0.15 || i >= len(model.Surface.LandMask) || !model.Surface.LandMask[i] {
			continue
		}
		candidates = append(candidates, candidate{index: i, weight: weight})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].weight > candidates[j].weight
	})

	minSpacing := 4 * math.Max(g.SpacingX(), g.SpacingZ())
	minSpacingSq := minSpacing * minSpacing
	out := make([]Point, 0, 6)
	for _, candidate := range candidates {
		x := candidate.index % g.Width
		z := candidate.index / g.Width
		wx, wz := g.NodePosition(x, z)
		tooClose := false
		for _, existing := range out {
			dx := wx - float64(existing.X)
			dz := wz - float64(existing.Z)
			if dx*dx+dz*dz < minSpacingSq {
				tooClose = true
				break
			}
		}
		if tooClose {
			continue
		}
		p := makePoint(rw, wx, wz, fmt.Sprintf(
			"strong continuous plateau influence (weight %.3f)",
			candidate.weight,
		))
		out = append(out, p)
		if len(out) == 6 {
			break
		}
	}
	return out
}

func findOuterIsland(rw *runtimeworld.Model, g world.GridGeometry, transition float64) (Point, bool) {
	step := math.Max(64, math.Min(g.SpacingX(), g.SpacingZ())*0.5)
	minZ := g.OriginZ + 0.10*g.SpanZBlocks
	maxZ := g.OriginZ + 0.90*g.SpanZBlocks
	for d := 0.20 * transition; d <= 2.40*transition; d += step {
		for z := minZ; z <= maxZ; z += step {
			for _, x := range []float64{g.OriginX - d, g.OriginX + g.SpanXBlocks + d} {
				s := rw.Sample(x, z)
				if s.SourceLand && s.World.WaterKind == reconcile.WaterNone {
					return makePoint(rw, x, z, "deterministic sparse island outside finite mainland model"), true
				}
			}
		}
	}
	return Point{}, false
}

func appendUniquePoint(points []Point, p Point) []Point {
	if p.TP == "" {
		return points
	}
	for _, existing := range points {
		if existing.X == p.X && existing.Z == p.Z {
			return points
		}
	}
	return append(points, p)
}

func waterName(kind reconcile.WaterKind) string {
	switch kind {
	case reconcile.WaterOcean:
		return "Ocean"
	case reconcile.WaterLake:
		return "Lake"
	case reconcile.WaterRiver:
		return "River"
	default:
		return "None"
	}
}

// Keep runtimeprofile referenced in this package's public planning boundary:
// report coordinates are derived from the same immutable Runtime sample type.
var _ runtimeprofile.Sample
