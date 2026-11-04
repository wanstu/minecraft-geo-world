package acceptance

import (
	"bufio"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const SupplementalRegistryVersion = "geoworld-acceptance-supplement-v1.1.11"

//go:embed resources/geoworld/acceptance-v1.1.11-supplement.tsv
var supplementalTSV string

// SupplementalPoint is a deterministic manual checkpoint added after the
// canonical v1.1 acceptance registry. Keeping it separate preserves the
// original 177-item specification while allowing later generation rules to
// register new visual acceptance points.
type SupplementalPoint struct {
	ID                     string
	GenerationRulesVersion string
	Seed                   int64
	Requirement            RequirementKind
	Inspection             InspectionKind
	Name                   string
	X                      int
	Y                      int
	Z                      int
	PassStandard           string
	Method                 string
}

func (p SupplementalPoint) TP() string {
	return fmt.Sprintf("/tp @s %d %d %d", p.X, p.Y, p.Z)
}

var (
	supplementalOnce sync.Once
	supplemental     []SupplementalPoint
	supplementalErr  error
)

func SupplementalPoints() ([]SupplementalPoint, error) {
	supplementalOnce.Do(loadSupplemental)
	if supplementalErr != nil {
		return nil, supplementalErr
	}
	return append([]SupplementalPoint(nil), supplemental...), nil
}

func SupplementalPointsForRules(version string) ([]SupplementalPoint, error) {
	points, err := SupplementalPoints()
	if err != nil {
		return nil, err
	}
	out := make([]SupplementalPoint, 0, len(points))
	for _, point := range points {
		if point.GenerationRulesVersion == version {
			out = append(out, point)
		}
	}
	return out, nil
}

func loadSupplemental() {
	scanner := bufio.NewScanner(strings.NewReader(supplementalTSV))
	seen := make(map[string]struct{}, 10)
	points := make([]SupplementalPoint, 0, 10)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "	", 11)
		if len(parts) != 11 {
			supplementalErr = fmt.Errorf(
				"acceptance supplement line %d: expected 11 columns",
				lineNumber,
			)
			return
		}
		seed, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			supplementalErr = fmt.Errorf(
				"acceptance supplement line %d: invalid seed",
				lineNumber,
			)
			return
		}
		x, err := strconv.Atoi(parts[6])
		if err != nil {
			supplementalErr = fmt.Errorf(
				"acceptance supplement line %d: invalid x",
				lineNumber,
			)
			return
		}
		y, err := strconv.Atoi(parts[7])
		if err != nil {
			supplementalErr = fmt.Errorf(
				"acceptance supplement line %d: invalid y",
				lineNumber,
			)
			return
		}
		z, err := strconv.Atoi(parts[8])
		if err != nil {
			supplementalErr = fmt.Errorf(
				"acceptance supplement line %d: invalid z",
				lineNumber,
			)
			return
		}
		point := SupplementalPoint{
			ID:                     parts[0],
			GenerationRulesVersion: parts[1],
			Seed:                   seed,
			Requirement:            RequirementKind(parts[3]),
			Inspection:             InspectionKind(parts[4]),
			Name:                   parts[5],
			X:                      x,
			Y:                      y,
			Z:                      z,
			PassStandard:           parts[9],
			Method:                 parts[10],
		}
		if point.ID == "" || point.GenerationRulesVersion == "" ||
			point.Name == "" || point.PassStandard == "" || point.Method == "" {
			supplementalErr = fmt.Errorf(
				"acceptance supplement line %d: empty required field",
				lineNumber,
			)
			return
		}
		if !validRequirement(point.Requirement) {
			supplementalErr = fmt.Errorf(
				"acceptance supplement line %d: invalid requirement %s",
				lineNumber,
				point.Requirement,
			)
			return
		}
		if !validInspection(point.Inspection) {
			supplementalErr = fmt.Errorf(
				"acceptance supplement line %d: invalid inspection %s",
				lineNumber,
				point.Inspection,
			)
			return
		}
		if _, exists := seen[point.ID]; exists {
			supplementalErr = fmt.Errorf(
				"acceptance supplement line %d: duplicate id %s",
				lineNumber,
				point.ID,
			)
			return
		}
		seen[point.ID] = struct{}{}
		points = append(points, point)
	}
	if err := scanner.Err(); err != nil {
		supplementalErr = err
		return
	}
	if len(points) != 10 {
		supplementalErr = fmt.Errorf(
			"acceptance supplement contains %d points, want 10",
			len(points),
		)
		return
	}
	supplemental = points
}
