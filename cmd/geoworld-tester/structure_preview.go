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
	SetID  string `json:"setId"`
	ChunkX int    `json:"chunkX"`
	ChunkZ int    `json:"chunkZ"`
	X      int    `json:"x"`
	Z      int    `json:"z"`
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
	Note              string                             `json:"note"`
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
	candidates, total := predictStructureCandidates(
		active.seed, snapshot, mode, minX, maxX, minZ, maxZ, limit,
	)
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

	writeJSON(w, http.StatusOK, structurePreviewResponse{
		Seed: active.seed, MinecraftVersion: snapshot.MinecraftVersion,
		Mode: mode, Sets: sets, Candidates: candidates,
		Total: total, Returned: len(candidates), Truncated: total > len(candidates),
		SkippedStronghold: true,
		Note:              "这些点与 Minecraft 26.3 原版 StructureSet placement 候选 chunk 同源；最终 StructureStart 仍由 Minecraft 的群系、结构合法性与生成检查决定。强要塞因还需 GeoWorld 群系偏移搜索，暂不伪造。",
	})
}

func predictStructureCandidates(
	seed int64,
	snapshot *structureSnapshot,
	mode string,
	minX, maxX, minZ, maxZ, limit int,
) ([]structureCandidate, int) {
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

				list = append(list, structureCandidate{
					SetID: id, ChunkX: chunkX, ChunkZ: chunkZ, X: x, Z: z,
				})
			}
		}
		if len(list) > 0 {
			perSet[id] = list
			total += len(list)
		}
	}
	if total <= limit {
		return flattenStructureCandidates(setIDs, perSet, nil), total
	}

	nonEmpty := make([]string, 0, len(perSet))
	for _, id := range setIDs {
		if len(perSet[id]) > 0 {
			nonEmpty = append(nonEmpty, id)
		}
	}
	if len(nonEmpty) == 0 {
		return nil, 0
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
	return flattenStructureCandidates(nonEmpty, perSet, allocation), total
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
