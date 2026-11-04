package acceptance

import (
	"bufio"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const RegionalAdvantageSupportRegistryVersion = "geoworld-acceptance-regional-advantage-support-v1"

//go:embed resources/geoworld/acceptance-regional-advantage-support-v1.tsv
var regionalAdvantageSupportTSV string

var (
	regionalAdvantageSupportOnce sync.Once
	regionalAdvantageSupport     []SupplementalPoint
	regionalAdvantageSupportErr  error
)

func RegionalAdvantageSupportPoints() ([]SupplementalPoint, error) {
	regionalAdvantageSupportOnce.Do(loadRegionalAdvantageSupport)
	if regionalAdvantageSupportErr != nil {
		return nil, regionalAdvantageSupportErr
	}
	return append(
		[]SupplementalPoint(nil),
		regionalAdvantageSupport...,
	), nil
}

func loadRegionalAdvantageSupport() {
	scanner := bufio.NewScanner(
		strings.NewReader(regionalAdvantageSupportTSV),
	)
	seen := make(map[string]struct{}, 6)
	points := make([]SupplementalPoint, 0, 6)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "	", 11)
		if len(parts) != 11 {
			regionalAdvantageSupportErr = fmt.Errorf(
				"regional advantage support line %d: expected 11 columns",
				lineNumber,
			)
			return
		}
		seed, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			regionalAdvantageSupportErr = fmt.Errorf(
				"regional advantage support line %d: invalid seed",
				lineNumber,
			)
			return
		}
		x, err := strconv.Atoi(parts[6])
		if err != nil {
			regionalAdvantageSupportErr = fmt.Errorf(
				"regional advantage support line %d: invalid x",
				lineNumber,
			)
			return
		}
		y, err := strconv.Atoi(parts[7])
		if err != nil {
			regionalAdvantageSupportErr = fmt.Errorf(
				"regional advantage support line %d: invalid y",
				lineNumber,
			)
			return
		}
		z, err := strconv.Atoi(parts[8])
		if err != nil {
			regionalAdvantageSupportErr = fmt.Errorf(
				"regional advantage support line %d: invalid z",
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
		if point.ID == "" ||
			point.GenerationRulesVersion == "" ||
			point.Name == "" ||
			point.PassStandard == "" ||
			point.Method == "" {
			regionalAdvantageSupportErr = fmt.Errorf(
				"regional advantage support line %d: empty required field",
				lineNumber,
			)
			return
		}
		if !validRequirement(point.Requirement) ||
			!validInspection(point.Inspection) {
			regionalAdvantageSupportErr = fmt.Errorf(
				"regional advantage support line %d: invalid requirement/inspection",
				lineNumber,
			)
			return
		}
		if _, exists := seen[point.ID]; exists {
			regionalAdvantageSupportErr = fmt.Errorf(
				"regional advantage support line %d: duplicate id %s",
				lineNumber,
				point.ID,
			)
			return
		}
		seen[point.ID] = struct{}{}
		points = append(points, point)
	}
	if err := scanner.Err(); err != nil {
		regionalAdvantageSupportErr = err
		return
	}
	if len(points) != 6 {
		regionalAdvantageSupportErr = fmt.Errorf(
			"regional advantage support contains %d points, want 6",
			len(points),
		)
		return
	}
	regionalAdvantageSupport = points
}
