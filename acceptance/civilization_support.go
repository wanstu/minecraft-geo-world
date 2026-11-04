package acceptance

import (
	"bufio"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const CivilizationSupportRegistryVersion = "geoworld-acceptance-civilization-support-v1"

//go:embed resources/geoworld/acceptance-civilization-support-v1.tsv
var civilizationSupportTSV string

var (
	civilizationSupportOnce sync.Once
	civilizationSupport     []SupplementalPoint
	civilizationSupportErr  error
)

func CivilizationSupportPoints() ([]SupplementalPoint, error) {
	civilizationSupportOnce.Do(loadCivilizationSupport)
	if civilizationSupportErr != nil {
		return nil, civilizationSupportErr
	}
	return append([]SupplementalPoint(nil), civilizationSupport...), nil
}

func loadCivilizationSupport() {
	scanner := bufio.NewScanner(strings.NewReader(civilizationSupportTSV))
	seen := make(map[string]struct{}, 5)
	points := make([]SupplementalPoint, 0, 5)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "	", 11)
		if len(parts) != 11 {
			civilizationSupportErr = fmt.Errorf(
				"civilization support line %d: expected 11 columns",
				lineNumber,
			)
			return
		}
		seed, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			civilizationSupportErr = fmt.Errorf(
				"civilization support line %d: invalid seed",
				lineNumber,
			)
			return
		}
		x, err := strconv.Atoi(parts[6])
		if err != nil {
			civilizationSupportErr = fmt.Errorf(
				"civilization support line %d: invalid x",
				lineNumber,
			)
			return
		}
		y, err := strconv.Atoi(parts[7])
		if err != nil {
			civilizationSupportErr = fmt.Errorf(
				"civilization support line %d: invalid y",
				lineNumber,
			)
			return
		}
		z, err := strconv.Atoi(parts[8])
		if err != nil {
			civilizationSupportErr = fmt.Errorf(
				"civilization support line %d: invalid z",
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
			civilizationSupportErr = fmt.Errorf(
				"civilization support line %d: empty required field",
				lineNumber,
			)
			return
		}
		if !validRequirement(point.Requirement) || !validInspection(point.Inspection) {
			civilizationSupportErr = fmt.Errorf(
				"civilization support line %d: invalid requirement/inspection",
				lineNumber,
			)
			return
		}
		if _, exists := seen[point.ID]; exists {
			civilizationSupportErr = fmt.Errorf(
				"civilization support line %d: duplicate id %s",
				lineNumber,
				point.ID,
			)
			return
		}
		seen[point.ID] = struct{}{}
		points = append(points, point)
	}
	if err := scanner.Err(); err != nil {
		civilizationSupportErr = err
		return
	}
	if len(points) != 5 {
		civilizationSupportErr = fmt.Errorf(
			"civilization support contains %d points, want 5",
			len(points),
		)
		return
	}
	civilizationSupport = points
}
