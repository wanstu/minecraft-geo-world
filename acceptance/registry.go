package acceptance

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
	"sync"
)

//go:embed resources/geoworld/acceptance-v1.1.tsv
var registryTSV string

type RequirementKind string

const (
	RequirementCore       RequirementKind = "CORE"
	RequirementLater      RequirementKind = "LATER"
	RequirementConstraint RequirementKind = "CONSTRAINT"
)

type InspectionKind string

const (
	InspectionAutomatic          InspectionKind = "AUTOMATIC"
	InspectionAutomaticAndManual InspectionKind = "AUTOMATIC_AND_MANUAL"
	InspectionPoint              InspectionKind = "POINT"
	InspectionRoute              InspectionKind = "ROUTE"
	InspectionRegion             InspectionKind = "REGION"
	InspectionRelation           InspectionKind = "RELATION"
	InspectionUI                 InspectionKind = "UI"
	InspectionAPI                InspectionKind = "API"
	InspectionPerformance        InspectionKind = "PERFORMANCE"
)

type Entry struct {
	ID           string
	Requirement  RequirementKind
	Inspection   InspectionKind
	Name         string
	PassStandard string
	Method       string
}

var (
	registryOnce sync.Once
	registry     []Entry
	registryErr  error
)

func Entries() ([]Entry, error) {
	registryOnce.Do(load)
	if registryErr != nil {
		return nil, registryErr
	}
	return append([]Entry(nil), registry...), nil
}

func MustEntries() []Entry {
	entries, err := Entries()
	if err != nil {
		panic(err)
	}
	return entries
}

func load() {
	scanner := bufio.NewScanner(strings.NewReader(registryTSV))
	seen := make(map[string]struct{}, 177)
	entries := make([]Entry, 0, 177)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "	", 6)
		if len(parts) != 6 {
			registryErr = fmt.Errorf("acceptance registry line %d: expected 6 columns", lineNumber)
			return
		}
		entry := Entry{
			ID:           parts[0],
			Requirement:  RequirementKind(parts[1]),
			Inspection:   InspectionKind(parts[2]),
			Name:         parts[3],
			PassStandard: parts[4],
			Method:       parts[5],
		}
		if entry.ID == "" || entry.Name == "" || entry.PassStandard == "" || entry.Method == "" {
			registryErr = fmt.Errorf("acceptance registry line %d: empty required field", lineNumber)
			return
		}
		if _, exists := seen[entry.ID]; exists {
			registryErr = fmt.Errorf("acceptance registry line %d: duplicate id %s", lineNumber, entry.ID)
			return
		}
		if !validRequirement(entry.Requirement) {
			registryErr = fmt.Errorf("acceptance registry line %d: invalid requirement %s", lineNumber, entry.Requirement)
			return
		}
		if !validInspection(entry.Inspection) {
			registryErr = fmt.Errorf("acceptance registry line %d: invalid inspection %s", lineNumber, entry.Inspection)
			return
		}
		seen[entry.ID] = struct{}{}
		entries = append(entries, entry)
	}
	if err := scanner.Err(); err != nil {
		registryErr = err
		return
	}
	if len(entries) != 177 {
		registryErr = fmt.Errorf("acceptance registry contains %d entries, want 177", len(entries))
		return
	}
	registry = entries
}

func validRequirement(kind RequirementKind) bool {
	switch kind {
	case RequirementCore, RequirementLater, RequirementConstraint:
		return true
	default:
		return false
	}
}

func validInspection(kind InspectionKind) bool {
	switch kind {
	case InspectionAutomatic, InspectionAutomaticAndManual,
		InspectionPoint, InspectionRoute, InspectionRegion,
		InspectionRelation, InspectionUI, InspectionAPI,
		InspectionPerformance:
		return true
	default:
		return false
	}
}
