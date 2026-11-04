package mineralmodel

import (
	"math"
	"sort"
	"strings"

	"geoworld/internal/world"
)

const (
	districtCellXZ = 4096
	districtMargin = 1700
	districtSalt   = uint64(0x4D494E4544495354)
	primarySalt    = uint64(0x5052494D4152594C)
	companionSalt  = uint64(0x434F4D50414E494F)
	replaceSalt    = uint64(0x5245504C4143454D)
)

var vanillaFamilies = []string{
	"coal", "iron", "copper", "gold",
	"lapis", "redstone", "diamond", "emerald",
}

var defaultDictionary = []string{
	"aluminum", "coal", "cobalt", "copper", "diamond", "emerald",
	"gold", "iridium", "iron", "lapis", "lead", "nickel",
	"osmium", "platinum", "redstone", "silver", "sulfur", "tin",
	"titanium", "tungsten", "uranium", "zinc",
}

type DistrictAssignment struct {
	ID               int64  `json:"id"`
	PlannedPrimary   string `json:"plannedPrimary"`
	PlannedCompanion string `json:"plannedCompanion"`
	Primary          string `json:"primary"`
	Companion        string `json:"companion"`
}

type Model struct {
	Dictionary  []string             `json:"dictionary"`
	Available   []string             `json:"available"`
	Assignments []DistrictAssignment `json:"assignments"`
}

func Build(seed int64, geometry world.GridGeometry, dictionary, available []string) *Model {
	dictionary = normalizeFamilies(dictionary)
	available = normalizeFamilies(available)
	if len(dictionary) == 0 {
		dictionary = append([]string(nil), defaultDictionary...)
	}
	if len(available) == 0 {
		available = intersection(dictionary, vanillaFamilies)
		if len(available) == 0 {
			available = append([]string(nil), vanillaFamilies...)
		}
	}
	// Every actually available ore family is also part of the dictionary used
	// to plan new models. This lets future mods participate without any
	// GeoWorld-specific hard dependency.
	dictionary = normalizeFamilies(append(dictionary, available...))

	minX := int(math.Floor(geometry.OriginX))
	maxX := int(math.Ceil(geometry.OriginX + geometry.SpanXBlocks))
	minZ := int(math.Floor(geometry.OriginZ))
	maxZ := int(math.Ceil(geometry.OriginZ + geometry.SpanZBlocks))
	firstX := floorDiv(minX-districtMargin, districtCellXZ)
	lastX := floorDiv(maxX+districtMargin, districtCellXZ)
	firstZ := floorDiv(minZ-districtMargin, districtCellXZ)
	lastZ := floorDiv(maxZ+districtMargin, districtCellXZ)

	assignments := make([]DistrictAssignment, 0, (lastX-firstX+1)*(lastZ-firstZ+1))
	for cellZ := firstZ; cellZ <= lastZ; cellZ++ {
		for cellX := firstX; cellX <= lastX; cellX++ {
			id := int64(cellSeed(uint64(seed)^districtSalt, cellX, 0, cellZ))
			plannedPrimary := pick(dictionary, mix(uint64(id)^primarySalt))
			plannedCompanion := pickCompanion(dictionary, plannedPrimary, uint64(id)^companionSalt)

			primary := plannedPrimary
			if !contains(available, primary) {
				primary = pick(available, mix(uint64(id)^hashString(plannedPrimary)^replaceSalt))
			}

			companionCandidate := plannedCompanion
			if primary != plannedPrimary {
				// The primary changed because the originally planned ore is not
				// present in this world. Re-plan the companion from the new
				// primary before checking companion availability.
				companionCandidate = pickCompanion(
					dictionary,
					primary,
					uint64(id)^hashString(primary)^companionSalt,
				)
			}

			companion := companionCandidate
			if companion == primary || !contains(available, companion) {
				companion = pickAvailableCompanion(
					available,
					primary,
					uint64(id)^hashString(primary)^replaceSalt^companionSalt,
				)
			}

			assignments = append(assignments, DistrictAssignment{
				ID:               id,
				PlannedPrimary:   plannedPrimary,
				PlannedCompanion: plannedCompanion,
				Primary:          primary,
				Companion:        companion,
			})
		}
	}

	return &Model{
		Dictionary:  append([]string(nil), dictionary...),
		Available:   append([]string(nil), available...),
		Assignments: assignments,
	}
}

func normalizeFamilies(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || strings.Contains(value, "/") {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func intersection(values, allowed []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(allowed))
	for _, value := range allowed {
		if _, ok := set[value]; ok {
			out = append(out, value)
		}
	}
	return out
}

func pick(values []string, seed uint64) string {
	if len(values) == 0 {
		return ""
	}
	return values[int(seed%uint64(len(values)))]
}

func pickCompanion(values []string, primary string, seed uint64) string {
	if len(values) <= 1 {
		return primary
	}
	start := int(mix(seed^hashString(primary)) % uint64(len(values)))
	for offset := 0; offset < len(values); offset++ {
		value := values[(start+offset)%len(values)]
		if value != primary {
			return value
		}
	}
	return primary
}

func pickAvailableCompanion(values []string, primary string, seed uint64) string {
	return pickCompanion(values, primary, seed)
}

func contains(values []string, needle string) bool {
	index := sort.SearchStrings(values, needle)
	return index < len(values) && values[index] == needle
}

func floorDiv(value, divisor int) int {
	q := value / divisor
	r := value % divisor
	if r != 0 && ((r < 0) != (divisor < 0)) {
		q--
	}
	return q
}

func cellSeed(seed uint64, x, y, z int) uint64 {
	h := seed ^ 0x9E3779B97F4A7C15
	h ^= uint64(int64(x)) * 0xBF58476D1CE4E5B9
	h ^= uint64(int64(y)) * 0x94D049BB133111EB
	h ^= uint64(int64(z)) * 0xD6E8FEB86659FD93
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	return h ^ (h >> 31)
}

func mix(value uint64) uint64 {
	value ^= value >> 30
	value *= 0xBF58476D1CE4E5B9
	value ^= value >> 27
	value *= 0x94D049BB133111EB
	return value ^ (value >> 31)
}

func hashString(value string) uint64 {
	const offset = uint64(1469598103934665603)
	const prime = uint64(1099511628211)
	h := offset
	for i := 0; i < len(value); i++ {
		h ^= uint64(value[i])
		h *= prime
	}
	return h
}
