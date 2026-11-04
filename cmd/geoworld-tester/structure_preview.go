package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"

	"geoworld/internal/mcprofile"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
)

//go:embed structure_data/minecraft-26.3.json
var structureSnapshotJSON []byte

type structureSnapshot struct {
	MinecraftVersion    string                               `json:"minecraftVersion"`
	StructureSets       map[string]structureSetSnapshot      `json:"structureSets"`
	VectorSeed          int64                                `json:"vectorSeed"`
	RandomSpreadVectors map[string][]structureVectorSnapshot `json:"randomSpreadVectors"`
}

type structureSetSnapshot struct {
	Structures []structureSelectionSnapshot `json:"structures"`
	Placement  structurePlacementSnapshot   `json:"placement"`
}

type structureSelectionSnapshot struct {
	Structure string `json:"structure"`
	Weight    int    `json:"weight"`
}

type structurePlacementSnapshot struct {
	Type                     string                      `json:"type"`
	LocateOffset             []int                       `json:"locate_offset,omitempty"`
	FrequencyReductionMethod string                      `json:"frequency_reduction_method,omitempty"`
	Frequency                float64                     `json:"frequency,omitempty"`
	Salt                     int                         `json:"salt"`
	Spacing                  int                         `json:"spacing,omitempty"`
	Separation               int                         `json:"separation,omitempty"`
	SpreadType               string                      `json:"spread_type,omitempty"`
	ExclusionZone            *structureExclusionSnapshot `json:"exclusion_zone,omitempty"`
	Distance                 int                         `json:"distance,omitempty"`
	Spread                   int                         `json:"spread,omitempty"`
	Count                    int                         `json:"count,omitempty"`
	PreferredBiomes          string                      `json:"preferred_biomes,omitempty"`
}

type structureExclusionSnapshot struct {
	OtherSet   string `json:"other_set"`
	ChunkCount int    `json:"chunk_count"`
}

type structureVectorSnapshot struct {
	QueryChunkX      int  `json:"queryChunkX"`
	QueryChunkZ      int  `json:"queryChunkZ"`
	CandidateChunkX  int  `json:"candidateChunkX"`
	CandidateChunkZ  int  `json:"candidateChunkZ"`
	FrequencyAllowed bool `json:"frequencyAllowed"`
}

type structureCandidate struct {
	SetID     string `json:"setId"`
	ChunkX    int    `json:"chunkX"`
	ChunkZ    int    `json:"chunkZ"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Z         int    `json:"z"`
	GeoStatus string `json:"geoStatus,omitempty"`
	GeoReason string `json:"geoReason,omitempty"`
}

type structureSetPreviewInfo struct {
	Structures []string `json:"structures"`
}

type structurePreviewResponse struct {
	Seed              int64                              `json:"seed"`
	MinecraftVersion  string                             `json:"minecraftVersion"`
	Mode              string                             `json:"mode"`
	Sets              map[string]structureSetPreviewInfo `json:"sets"`
	Candidates        []structureCandidate               `json:"candidates"`
	Total             int                                `json:"total"`
	Returned          int                                `json:"returned"`
	Truncated         bool                               `json:"truncated"`
	SkippedStronghold bool                               `json:"skippedStronghold"`
	GeoFiltered       int                                `json:"geoFiltered"`
	GeoPassed         int                                `json:"geoPassed"`
	GeoUnknown        int                                `json:"geoUnknown"`
	Exact             int                                `json:"exact"`
	Note              string                             `json:"note"`
}

type structureSearchResponse struct {
	Seed            int64              `json:"seed"`
	QueryX          int                `json:"queryX"`
	QueryZ          int                `json:"queryZ"`
	RequestedSetID  string             `json:"requestedSetId,omitempty"`
	Candidate       structureCandidate `json:"candidate"`
	Distance        float64            `json:"distance"`
	ExactStronghold bool               `json:"exactStronghold"`
}

var (
	structureSnapshotOnce sync.Once
	structureSnapshotData structureSnapshot
	structureSnapshotErr  error
)

func loadStructureSnapshot() (*structureSnapshot, error) {
	structureSnapshotOnce.Do(func() {
		structureSnapshotErr = json.Unmarshal(structureSnapshotJSON, &structureSnapshotData)
		if structureSnapshotErr == nil {
			structureSnapshotErr = validateStructureSnapshot(&structureSnapshotData)
		}
	})
	if structureSnapshotErr != nil {
		return nil, structureSnapshotErr
	}
	return &structureSnapshotData, nil
}

func validateStructureSnapshot(snapshot *structureSnapshot) error {
	if snapshot == nil || snapshot.MinecraftVersion == "" || len(snapshot.StructureSets) == 0 {
		return errors.New("empty Minecraft structure snapshot")
	}
	for id, vectors := range snapshot.RandomSpreadVectors {
		set, ok := snapshot.StructureSets[id]
		if !ok || set.Placement.Type != "minecraft:random_spread" {
			return fmt.Errorf("structure vector references invalid set %q", id)
		}
		for _, vector := range vectors {
			x, z := randomSpreadPotentialChunk(
				snapshot.VectorSeed,
				vector.QueryChunkX,
				vector.QueryChunkZ,
				set.Placement,
			)
			if x != vector.CandidateChunkX || z != vector.CandidateChunkZ {
				return fmt.Errorf(
					"structure placement parity mismatch %s query=%d,%d got=%d,%d want=%d,%d",
					id, vector.QueryChunkX, vector.QueryChunkZ,
					x, z, vector.CandidateChunkX, vector.CandidateChunkZ,
				)
			}
			allowed := structureFrequencyAllowed(
				snapshot.VectorSeed, x, z, set.Placement,
			)
			if allowed != vector.FrequencyAllowed {
				return fmt.Errorf(
					"structure frequency parity mismatch %s chunk=%d,%d got=%t want=%t",
					id, x, z, allowed, vector.FrequencyAllowed,
				)
			}
		}
	}
	return nil
}

var overworldRandomSpreadSets = map[string]bool{
	"minecraft:abandoned_camp":    true,
	"minecraft:ancient_cities":    true,
	"minecraft:buried_treasures":  true,
	"minecraft:desert_pyramids":   true,
	"minecraft:igloos":            true,
	"minecraft:jungle_temples":    true,
	"minecraft:mineshafts":        true,
	"minecraft:ocean_monuments":   true,
	"minecraft:ocean_ruins":       true,
	"minecraft:pillager_outposts": true,
	"minecraft:ruined_portals":    true,
	"minecraft:shipwrecks":        true,
	"minecraft:swamp_huts":        true,
	"minecraft:trail_ruins":       true,
	"minecraft:trial_chambers":    true,
	"minecraft:villages":          true,
	"minecraft:woodland_mansions": true,
}

var denseStructureSets = map[string]bool{
	"minecraft:buried_treasures": true,
	"minecraft:mineshafts":       true,
}

func (s *appState) handleStructureCandidates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	snapshot, err := loadStructureSnapshot()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active == nil {
		writeError(w, http.StatusConflict, errNoActiveWorld)
		return
	}

	mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
	if mode == "" {
		mode = "major"
	}
	if mode != "major" && mode != "all" {
		writeError(w, http.StatusBadRequest, errors.New("structure mode must be major or all"))
		return
	}

	g := active.geometry
	modelMinX := int(math.Floor(g.OriginX))
	modelMaxX := int(math.Ceil(g.OriginX + g.SpanXBlocks))
	modelMinZ := int(math.Floor(g.OriginZ))
	modelMaxZ := int(math.Ceil(g.OriginZ + g.SpanZBlocks))
	minX, err := queryInt(r, "minX", modelMinX)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	maxX, err := queryInt(r, "maxX", modelMaxX)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	minZ, err := queryInt(r, "minZ", modelMinZ)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	maxZ, err := queryInt(r, "maxZ", modelMaxZ)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	minX = maxInt(minX, modelMinX)
	maxX = minInt(maxX, modelMaxX)
	minZ = maxInt(minZ, modelMinZ)
	maxZ = minInt(maxZ, modelMaxZ)
	if minX > maxX || minZ > maxZ {
		writeError(w, http.StatusBadRequest, errors.New("structure viewport is outside the model"))
		return
	}

	maxSpan := 32000
	if mode == "all" {
		maxSpan = 7000
	}
	if maxX-minX > maxSpan || maxZ-minZ > maxSpan {
		writeError(w, http.StatusRequestEntityTooLarge,
			fmt.Errorf("%s structure viewport exceeds %d blocks; zoom in", mode, maxSpan))
		return
	}

	limit := parseBoundedInt(r.URL.Query().Get("limit"), 30000, 100, 50000)
	accept := func(candidate structureCandidate) bool {
		return structureGeoCompatible(
			candidate.SetID,
			active.sample(float64(candidate.X), float64(candidate.Z)),
		)
	}
	candidates, total, geoFiltered := predictStructureCandidates(
		active.seed, snapshot, mode, minX, maxX, minZ, maxZ, limit, accept,
	)
	randomTotal := total
	randomReturned := len(candidates)
	strongholds, strongholdErr := s.strongholdPreviewCandidates(active)
	if strongholdErr == nil {
		for _, candidate := range strongholds {
			if candidate.X < minX || candidate.X > maxX ||
				candidate.Z < minZ || candidate.Z > maxZ {
				continue
			}
			candidates = append(candidates, candidate)
			total++
		}
	}

	geoPassed, geoUnknown, exactCount := 0, 0, 0
	for i := range candidates {
		sample := active.sample(
			float64(candidates[i].X),
			float64(candidates[i].Z),
		)
		candidates[i].Y = minecraft26TeleportY(sample)
		if candidates[i].SetID == "minecraft:strongholds" {
			candidates[i].GeoStatus = "exact"
			candidates[i].GeoReason = "Minecraft 26.3 同心环位置，并应用 GeoWorld 群系偏移搜索。"
			exactCount++
			continue
		}
		candidates[i].GeoStatus, candidates[i].GeoReason =
			structureGeoAssessment(candidates[i].SetID, sample)
		switch candidates[i].GeoStatus {
		case "pass":
			geoPassed++
		case "unknown":
			geoUnknown++
		}
	}

	sets := make(map[string]structureSetPreviewInfo)
	for _, candidate := range candidates {
		if _, ok := sets[candidate.SetID]; ok {
			continue
		}
		set := snapshot.StructureSets[candidate.SetID]
		structures := make([]string, 0, len(set.Structures))
		for _, entry := range set.Structures {
			structures = append(structures, entry.Structure)
		}
		sets[candidate.SetID] = structureSetPreviewInfo{Structures: structures}
	}

	note := "普通结构点与 Minecraft 26.3 原版 StructureSet placement 候选 chunk 同源；已按 GeoWorld 当前水体/生态剔除可高置信判断为不兼容的候选，复杂地下或多条件结构仍保持候选身份。"
	if strongholdErr == nil {
		note += " 强要塞使用 Minecraft 26.3 concentric-rings 规则与 GeoWorld 群系偏移搜索，构建时会与 Minecraft 原生实现逐点校验。"
	} else {
		note += " 强要塞精确预览需要从可执行 GeoWorld JAR 启动 Web 工具。"
	}

	writeJSON(w, http.StatusOK, structurePreviewResponse{
		Seed: active.seed, MinecraftVersion: snapshot.MinecraftVersion,
		Mode: mode, Sets: sets, Candidates: candidates,
		Total: total, Returned: len(candidates),
		Truncated:         randomReturned < randomTotal,
		SkippedStronghold: strongholdErr != nil,
		GeoFiltered:       geoFiltered,
		GeoPassed:         geoPassed,
		GeoUnknown:        geoUnknown,
		Exact:             exactCount,
		Note:              note,
	})
}

func (s *appState) handleStructureSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	snapshot, err := loadStructureSnapshot()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active == nil {
		writeError(w, http.StatusConflict, errNoActiveWorld)
		return
	}

	queryX, err := queryInt(r, "x", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	queryZ, err := queryInt(r, "z", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	setID := strings.TrimSpace(r.URL.Query().Get("setId"))
	if setID != "" && setID != "minecraft:strongholds" && !overworldRandomSpreadSets[setID] {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported overworld structure set %q", setID))
		return
	}

	g := active.geometry
	modelMinX := int(math.Floor(g.OriginX))
	modelMaxX := int(math.Ceil(g.OriginX + g.SpanXBlocks))
	modelMinZ := int(math.Floor(g.OriginZ))
	modelMaxZ := int(math.Ceil(g.OriginZ + g.SpanZBlocks))
	if queryX < modelMinX || queryX > modelMaxX || queryZ < modelMinZ || queryZ > modelMaxZ {
		writeError(w, http.StatusBadRequest, errors.New("search coordinate is outside the current World Model"))
		return
	}

	bestDistance := math.Inf(1)
	best := structureCandidate{}
	found := false
	exactStronghold := false

	if setID == "" || setID == "minecraft:strongholds" {
		strongholds, strongholdErr := s.strongholdPreviewCandidates(active)
		if strongholdErr != nil {
			if setID == "minecraft:strongholds" {
				writeError(w, http.StatusServiceUnavailable, strongholdErr)
				return
			}
		} else {
			for _, candidate := range strongholds {
				if candidate.X < modelMinX || candidate.X > modelMaxX ||
					candidate.Z < modelMinZ || candidate.Z > modelMaxZ {
					continue
				}
				distance := math.Hypot(float64(candidate.X-queryX), float64(candidate.Z-queryZ))
				if distance < bestDistance {
					bestDistance = distance
					best = candidate
					found = true
					exactStronghold = true
				}
			}
		}
	}

	setIDs := make([]string, 0, len(overworldRandomSpreadSets))
	for id := range overworldRandomSpreadSets {
		if setID != "" && id != setID {
			continue
		}
		setIDs = append(setIDs, id)
	}
	sort.Strings(setIDs)
	for _, id := range setIDs {
		candidate, distance, ok := findNearestRandomSpreadCandidate(
			active.seed, snapshot, id,
			queryX, queryZ,
			modelMinX, modelMaxX, modelMinZ, modelMaxZ,
			func(candidate structureCandidate) bool {
				return structureGeoCompatible(
					candidate.SetID,
					active.sample(float64(candidate.X), float64(candidate.Z)),
				)
			},
		)
		if ok && distance < bestDistance {
			bestDistance = distance
			best = candidate
			found = true
			exactStronghold = false
		}
	}

	if !found {
		writeError(w, http.StatusNotFound, errors.New("no matching structure candidate inside the current World Model"))
		return
	}
	bestSample := active.sample(float64(best.X), float64(best.Z))
	best.Y = minecraft26TeleportY(bestSample)
	if exactStronghold {
		best.GeoStatus = "exact"
		best.GeoReason = "Minecraft 26.3 同心环位置，并应用 GeoWorld 群系偏移搜索。"
	} else {
		best.GeoStatus, best.GeoReason =
			structureGeoAssessment(best.SetID, bestSample)
	}
	writeJSON(w, http.StatusOK, structureSearchResponse{
		Seed: active.seed, QueryX: queryX, QueryZ: queryZ,
		RequestedSetID: setID, Candidate: best,
		Distance: bestDistance, ExactStronghold: exactStronghold,
	})
}

func findNearestRandomSpreadCandidate(
	seed int64,
	snapshot *structureSnapshot,
	setID string,
	queryX, queryZ int,
	modelMinX, modelMaxX, modelMinZ, modelMaxZ int,
	accept func(structureCandidate) bool,
) (structureCandidate, float64, bool) {
	set, ok := snapshot.StructureSets[setID]
	if !ok || set.Placement.Type != "minecraft:random_spread" {
		return structureCandidate{}, 0, false
	}
	placement := set.Placement
	if placement.Spacing <= 0 || placement.Separation < 0 ||
		placement.Separation >= placement.Spacing {
		return structureCandidate{}, 0, false
	}

	minChunkX := floorDiv(modelMinX, 16)
	maxChunkX := floorDiv(modelMaxX, 16)
	minChunkZ := floorDiv(modelMinZ, 16)
	maxChunkZ := floorDiv(modelMaxZ, 16)
	minRegionX := floorDiv(minChunkX, placement.Spacing) - 1
	maxRegionX := floorDiv(maxChunkX, placement.Spacing) + 1
	minRegionZ := floorDiv(minChunkZ, placement.Spacing) - 1
	maxRegionZ := floorDiv(maxChunkZ, placement.Spacing) + 1
	centerRegionX := floorDiv(floorDiv(queryX, 16), placement.Spacing)
	centerRegionZ := floorDiv(floorDiv(queryZ, 16), placement.Spacing)

	maxRadius := 0
	for _, d := range []int{
		centerRegionX - minRegionX, maxRegionX - centerRegionX,
		centerRegionZ - minRegionZ, maxRegionZ - centerRegionZ,
	} {
		if d < 0 {
			d = -d
		}
		if d > maxRadius {
			maxRadius = d
		}
	}

	bestDistance := math.Inf(1)
	best := structureCandidate{}
	found := false
	consider := func(regionX, regionZ int) {
		if regionX < minRegionX || regionX > maxRegionX ||
			regionZ < minRegionZ || regionZ > maxRegionZ {
			return
		}
		baseChunkX := regionX * placement.Spacing
		baseChunkZ := regionZ * placement.Spacing
		chunkX, chunkZ := randomSpreadPotentialChunk(seed, baseChunkX, baseChunkZ, placement)
		if chunkX < minChunkX || chunkX > maxChunkX ||
			chunkZ < minChunkZ || chunkZ > maxChunkZ {
			return
		}
		if !structureFrequencyAllowed(seed, chunkX, chunkZ, placement) {
			return
		}
		if placement.ExclusionZone != nil &&
			structureExcludedByOtherSet(snapshot, seed, chunkX, chunkZ, *placement.ExclusionZone) {
			return
		}
		offsetX, offsetZ := 0, 0
		if len(placement.LocateOffset) >= 3 {
			offsetX = placement.LocateOffset[0]
			offsetZ = placement.LocateOffset[2]
		}
		x := chunkX*16 + offsetX
		z := chunkZ*16 + offsetZ
		if x < modelMinX || x > modelMaxX || z < modelMinZ || z > modelMaxZ {
			return
		}
		candidate := structureCandidate{SetID: setID, ChunkX: chunkX, ChunkZ: chunkZ, X: x, Z: z}
		if accept != nil && !accept(candidate) {
			return
		}
		distance := math.Hypot(float64(x-queryX), float64(z-queryZ))
		if distance < bestDistance {
			bestDistance = distance
			best = candidate
			found = true
		}
	}

	for radius := 0; radius <= maxRadius; radius++ {
		if radius == 0 {
			consider(centerRegionX, centerRegionZ)
		} else {
			minRX, maxRX := centerRegionX-radius, centerRegionX+radius
			minRZ, maxRZ := centerRegionZ-radius, centerRegionZ+radius
			for regionX := minRX; regionX <= maxRX; regionX++ {
				consider(regionX, minRZ)
				consider(regionX, maxRZ)
			}
			for regionZ := minRZ + 1; regionZ < maxRZ; regionZ++ {
				consider(minRX, regionZ)
				consider(maxRX, regionZ)
			}
		}
		if found {
			lowerBound := float64(radius*placement.Spacing*16 - 64)
			if lowerBound > 0 && lowerBound >= bestDistance {
				break
			}
		}
	}
	return best, bestDistance, found
}

func predictStructureCandidates(
	seed int64,
	snapshot *structureSnapshot,
	mode string,
	minX, maxX, minZ, maxZ, limit int,
	accept func(structureCandidate) bool,
) ([]structureCandidate, int, int) {
	minChunkX := floorDiv(minX, 16)
	maxChunkX := floorDiv(maxX, 16)
	minChunkZ := floorDiv(minZ, 16)
	maxChunkZ := floorDiv(maxZ, 16)

	setIDs := make([]string, 0, len(overworldRandomSpreadSets))
	for id := range overworldRandomSpreadSets {
		if mode == "major" && denseStructureSets[id] {
			continue
		}
		setIDs = append(setIDs, id)
	}
	sort.Strings(setIDs)

	perSet := make(map[string][]structureCandidate, len(setIDs))
	total := 0
	geoFiltered := 0
	for _, id := range setIDs {
		set, ok := snapshot.StructureSets[id]
		if !ok || set.Placement.Type != "minecraft:random_spread" {
			continue
		}
		placement := set.Placement
		if placement.Spacing <= 0 || placement.Separation < 0 ||
			placement.Separation >= placement.Spacing {
			continue
		}

		minRegionX := floorDiv(minChunkX, placement.Spacing) - 1
		maxRegionX := floorDiv(maxChunkX, placement.Spacing) + 1
		minRegionZ := floorDiv(minChunkZ, placement.Spacing) - 1
		maxRegionZ := floorDiv(maxChunkZ, placement.Spacing) + 1

		list := make([]structureCandidate, 0, 128)
		for regionZ := minRegionZ; regionZ <= maxRegionZ; regionZ++ {
			for regionX := minRegionX; regionX <= maxRegionX; regionX++ {
				queryX := regionX * placement.Spacing
				queryZ := regionZ * placement.Spacing
				chunkX, chunkZ := randomSpreadPotentialChunk(seed, queryX, queryZ, placement)
				if chunkX < minChunkX || chunkX > maxChunkX ||
					chunkZ < minChunkZ || chunkZ > maxChunkZ {
					continue
				}
				if !structureFrequencyAllowed(seed, chunkX, chunkZ, placement) {
					continue
				}
				if placement.ExclusionZone != nil &&
					structureExcludedByOtherSet(snapshot, seed, chunkX, chunkZ, *placement.ExclusionZone) {
					continue
				}

				offsetX, offsetZ := 0, 0
				if len(placement.LocateOffset) >= 3 {
					offsetX = placement.LocateOffset[0]
					offsetZ = placement.LocateOffset[2]
				}
				x := chunkX*16 + offsetX
				z := chunkZ*16 + offsetZ
				if x < minX || x > maxX || z < minZ || z > maxZ {
					continue
				}

				candidate := structureCandidate{
					SetID: id, ChunkX: chunkX, ChunkZ: chunkZ, X: x, Z: z,
				}
				if accept != nil && !accept(candidate) {
					geoFiltered++
					continue
				}
				list = append(list, candidate)
			}
		}
		if len(list) > 0 {
			perSet[id] = list
			total += len(list)
		}
	}
	if total <= limit {
		return flattenStructureCandidates(setIDs, perSet, nil), total, geoFiltered
	}

	nonEmpty := make([]string, 0, len(perSet))
	for _, id := range setIDs {
		if len(perSet[id]) > 0 {
			nonEmpty = append(nonEmpty, id)
		}
	}
	if len(nonEmpty) == 0 {
		return nil, 0, geoFiltered
	}

	allocation := make(map[string]int, len(nonEmpty))
	base := limit / len(nonEmpty)
	if base < 1 {
		base = 1
	}
	used := 0
	for _, id := range nonEmpty {
		n := minInt(base, len(perSet[id]))
		allocation[id] = n
		used += n
	}
	for used < limit {
		progressed := false
		for _, id := range nonEmpty {
			if used >= limit {
				break
			}
			if allocation[id] >= len(perSet[id]) {
				continue
			}
			allocation[id]++
			used++
			progressed = true
		}
		if !progressed {
			break
		}
	}
	return flattenStructureCandidates(nonEmpty, perSet, allocation), total, geoFiltered
}

func flattenStructureCandidates(
	setIDs []string,
	perSet map[string][]structureCandidate,
	allocation map[string]int,
) []structureCandidate {
	out := make([]structureCandidate, 0)
	for _, id := range setIDs {
		list := perSet[id]
		if len(list) == 0 {
			continue
		}
		n := len(list)
		if allocation != nil {
			n = allocation[id]
		}
		if n <= 0 {
			continue
		}
		if n >= len(list) {
			out = append(out, list...)
			continue
		}
		for i := 0; i < n; i++ {
			index := ((2*i + 1) * len(list)) / (2 * n)
			if index >= len(list) {
				index = len(list) - 1
			}
			out = append(out, list[index])
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Z != out[j].Z {
			return out[i].Z < out[j].Z
		}
		if out[i].X != out[j].X {
			return out[i].X < out[j].X
		}
		return out[i].SetID < out[j].SetID
	})
	return out
}

func structureGeoAssessment(
	setID string,
	sample runtimeprofile.Sample,
) (string, string) {
	water := sample.World.WaterKind
	family := sample.BiomeFamily
	landCandidate := water != reconcile.WaterOcean && water != reconcile.WaterLake

	pass := func(ok bool, accepted, rejected string) (string, string) {
		if ok {
			return "pass", accepted
		}
		return "reject", rejected
	}

	switch setID {
	case "minecraft:ocean_monuments", "minecraft:ocean_ruins":
		return pass(
			water == reconcile.WaterOcean,
			"GeoWorld 水体条件通过：当前位置属于海洋。",
			"GeoWorld 水体条件拒绝：当前位置不是海洋。",
		)

	case "minecraft:desert_pyramids":
		return pass(
			landCandidate &&
				(family == mcprofile.BiomeHotDesert ||
					family == mcprofile.BiomeTemperateDesert),
			"GeoWorld 地表与荒漠生态条件通过。",
			"GeoWorld 条件拒绝：当前位置不是兼容的陆地荒漠生态。",
		)

	case "minecraft:igloos":
		return pass(
			landCandidate &&
				(family == mcprofile.BiomeTundra ||
					family == mcprofile.BiomeIceSheet ||
					family == mcprofile.BiomeAlpine ||
					family == mcprofile.BiomeBorealForest),
			"GeoWorld 地表与寒冷生态条件通过。",
			"GeoWorld 条件拒绝：当前位置不是兼容的寒冷陆地生态。",
		)

	case "minecraft:jungle_temples":
		return pass(
			landCandidate &&
				(family == mcprofile.BiomeTropicalRainforest ||
					family == mcprofile.BiomeTropicalMonsoonForest),
			"GeoWorld 地表与热带森林生态条件通过。",
			"GeoWorld 条件拒绝：当前位置不是兼容的热带森林陆地生态。",
		)

	case "minecraft:swamp_huts":
		return pass(
			landCandidate && family == mcprofile.BiomeWetland,
			"GeoWorld 地表与湿地生态条件通过。",
			"GeoWorld 条件拒绝：当前位置不是兼容的陆地湿地生态。",
		)

	case "minecraft:woodland_mansions":
		return pass(
			landCandidate &&
				(family == mcprofile.BiomeTemperateForest ||
					family == mcprofile.BiomeBorealForest ||
					family == mcprofile.BiomeSubtropicalForest),
			"GeoWorld 地表与森林生态条件通过。",
			"GeoWorld 条件拒绝：当前位置不是兼容的森林陆地生态。",
		)

	case "minecraft:villages", "minecraft:pillager_outposts":
		return pass(
			landCandidate &&
				(family == mcprofile.BiomeTemperateGrassland ||
					family == mcprofile.BiomeTemperateDesert ||
					family == mcprofile.BiomeHotDesert ||
					family == mcprofile.BiomeSavanna ||
					family == mcprofile.BiomeBorealForest ||
					family == mcprofile.BiomeTundra),
			"GeoWorld 地表与聚落生态条件通过。",
			"GeoWorld 条件拒绝：当前位置不是兼容的聚落陆地生态。",
		)

	default:
		// These sets depend on underground conditions, coast/beach details,
		// structure-specific terrain checks, or broad biome tags that the
		// current surface World Model cannot classify with enough confidence.
		// Keep them as candidates rather than creating false negatives.
		return "unknown", "GeoWorld 当前只能确认原版 placement；地下、海岸或结构自身条件仍需 Minecraft 运行时判定。"
	}
}

func structureGeoCompatible(
	setID string,
	sample runtimeprofile.Sample,
) bool {
	status, _ := structureGeoAssessment(setID, sample)
	return status != "reject"
}

func structureExcludedByOtherSet(
	snapshot *structureSnapshot,
	seed int64,
	chunkX, chunkZ int,
	exclusion structureExclusionSnapshot,
) bool {
	other, ok := snapshot.StructureSets[exclusion.OtherSet]
	if !ok || other.Placement.Type != "minecraft:random_spread" {
		return false
	}
	for x := chunkX - exclusion.ChunkCount; x <= chunkX+exclusion.ChunkCount; x++ {
		for z := chunkZ - exclusion.ChunkCount; z <= chunkZ+exclusion.ChunkCount; z++ {
			if isRandomSpreadPlacementChunk(seed, x, z, other.Placement, false, snapshot) {
				return true
			}
		}
	}
	return false
}

func isRandomSpreadPlacementChunk(
	seed int64,
	chunkX, chunkZ int,
	placement structurePlacementSnapshot,
	checkExclusion bool,
	snapshot *structureSnapshot,
) bool {
	x, z := randomSpreadPotentialChunk(seed, chunkX, chunkZ, placement)
	if x != chunkX || z != chunkZ {
		return false
	}
	if !structureFrequencyAllowed(seed, chunkX, chunkZ, placement) {
		return false
	}
	if checkExclusion && placement.ExclusionZone != nil &&
		structureExcludedByOtherSet(snapshot, seed, chunkX, chunkZ, *placement.ExclusionZone) {
		return false
	}
	return true
}

func randomSpreadPotentialChunk(
	seed int64,
	chunkX, chunkZ int,
	placement structurePlacementSnapshot,
) (int, int) {
	regionX := floorDiv(chunkX, placement.Spacing)
	regionZ := floorDiv(chunkZ, placement.Spacing)
	random := newJavaRandom(0)
	random.setLargeFeatureWithSalt(seed, regionX, regionZ, placement.Salt)
	bound := placement.Spacing - placement.Separation
	offsetX := 0
	offsetZ := 0
	if placement.SpreadType == "triangular" {
		offsetX = (random.nextInt(bound) + random.nextInt(bound)) / 2
		offsetZ = (random.nextInt(bound) + random.nextInt(bound)) / 2
	} else {
		offsetX = random.nextInt(bound)
		offsetZ = random.nextInt(bound)
	}
	return regionX*placement.Spacing + offsetX,
		regionZ*placement.Spacing + offsetZ
}

func structureFrequencyAllowed(
	seed int64,
	chunkX, chunkZ int,
	placement structurePlacementSnapshot,
) bool {
	frequency := float32(placement.Frequency)
	if placement.Frequency == 0 {
		frequency = 1
	}
	if frequency >= 1 {
		return true
	}
	method := placement.FrequencyReductionMethod
	if method == "" {
		method = "default"
	}
	switch method {
	case "legacy_type_1":
		regionX := int32(chunkX >> 4)
		regionZ := int32(chunkZ >> 4)
		mixed := int64(regionX ^ (regionZ << 4))
		random := newJavaRandom(mixed ^ seed)
		_ = random.nextInt32()
		bound := int(float32(1.0) / frequency)
		return bound > 0 && random.nextInt(bound) == 0
	case "legacy_type_2":
		random := newJavaRandom(0)
		random.setLargeFeatureSeed(seed, chunkX, chunkZ)
		return random.nextDouble() < float64(frequency)
	case "legacy_type_3":
		random := newJavaRandom(0)
		random.setLargeFeatureWithSalt(seed, chunkX, chunkZ, 10387320)
		return random.nextFloat() < frequency
	default:
		random := newJavaRandom(0)
		random.setLargeFeatureWithSalt(seed, chunkX, chunkZ, placement.Salt)
		return random.nextFloat() < frequency
	}
}

func floorDiv(a, b int) int {
	q := a / b
	r := a % b
	if r != 0 && ((r < 0) != (b < 0)) {
		q--
	}
	return q
}

const (
	javaRandomMultiplier uint64 = 0x5DEECE66D
	javaRandomAddend     uint64 = 0xB
	javaRandomMask       uint64 = (1 << 48) - 1
)

type javaRandom struct {
	seed uint64
}

func newJavaRandom(seed int64) *javaRandom {
	r := &javaRandom{}
	r.setSeed(seed)
	return r
}

func (r *javaRandom) setSeed(seed int64) {
	r.seed = (uint64(seed) ^ javaRandomMultiplier) & javaRandomMask
}

func (r *javaRandom) next(bits uint) uint32 {
	r.seed = (r.seed*javaRandomMultiplier + javaRandomAddend) & javaRandomMask
	return uint32(r.seed >> (48 - bits))
}

func (r *javaRandom) nextInt(bound int) int {
	if bound <= 0 {
		panic("javaRandom.nextInt bound must be positive")
	}
	if bound&(bound-1) == 0 {
		return int((int64(bound) * int64(r.next(31))) >> 31)
	}
	for {
		bits := int32(r.next(31))
		value := bits % int32(bound)
		check := int32(int64(bits) - int64(value) + int64(bound-1))
		if check >= 0 {
			return int(value)
		}
	}
}

func (r *javaRandom) nextInt32() int32 {
	return int32(r.next(32))
}

func (r *javaRandom) nextLong() int64 {
	high := int64(int32(r.next(32)))
	low := int64(int32(r.next(32)))
	return int64(uint64(high<<32) + uint64(low))
}

func (r *javaRandom) nextFloat() float32 {
	return float32(r.next(24)) / float32(1<<24)
}

func (r *javaRandom) nextDouble() float64 {
	high := uint64(r.next(26))
	low := uint64(r.next(27))
	return float64((high<<27)+low) / float64(uint64(1)<<53)
}

func (r *javaRandom) setLargeFeatureWithSalt(
	worldSeed int64,
	chunkX, chunkZ, salt int,
) {
	r.setSeed(
		int64(chunkX)*341873128712 +
			int64(chunkZ)*132897987541 +
			worldSeed +
			int64(salt),
	)
}

func (r *javaRandom) setLargeFeatureSeed(
	worldSeed int64,
	chunkX, chunkZ int,
) {
	r.setSeed(worldSeed)
	xSeed := r.nextLong()
	zSeed := r.nextLong()
	mixed := (int64(chunkX) * xSeed) ^ (int64(chunkZ) * zSeed) ^ worldSeed
	r.setSeed(mixed)
}
