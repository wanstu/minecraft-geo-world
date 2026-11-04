package acceptance

import (
	"bufio"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

const SettlementSupportRegistryVersion = "geoworld-acceptance-settlement-support-v1"

//go:embed resources/geoworld/acceptance-settlement-support-v1.tsv
var settlementSupportTSV string

var (
	settlementSupportOnce sync.Once
	settlementSupport     []SupplementalPoint
	settlementSupportErr  error
)

// SettlementSupportPoints returns deterministic manual checkpoints for the
// civilization-support geography layer. It stays separate from the canonical
// v1.1 registry and the special-landscape supplement.
func SettlementSupportPoints() ([]SupplementalPoint, error) {
	settlementSupportOnce.Do(loadSettlementSupport)
	if settlementSupportErr != nil {
		return nil, settlementSupportErr
	}
	return append([]SupplementalPoint(nil), settlementSupport...), nil
}

func loadSettlementSupport() {
	scanner := bufio.NewScanner(strings.NewReader(settlementSupportTSV))
	seen := make(map[string]struct{}, 3)
	points := make([]SupplementalPoint, 0, 3)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "	", 11)
		if len(parts) != 11 {
			settlementSupportErr = fmt.Errorf(
				"settlement support line %d: expected 11 columns",
				lineNumber,
			)
			return
		}
		seed, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			settlementSupportErr = fmt.Errorf(
				"settlement support line %d: invalid seed",
				lineNumber,
			)
			return
		}
		x, err := strconv.Atoi(parts[6])
		if err != nil {
			settlementSupportErr = fmt.Errorf(
				"settlement support line %d: invalid x",
				lineNumber,
			)
			return
		}
		y, err := strconv.Atoi(parts[7])
		if err != nil {
			settlementSupportErr = fmt.Errorf(
				"settlement support line %d: invalid y",
				lineNumber,
			)
			return
		}
		z, err := strconv.Atoi(parts[8])
		if err != nil {
			settlementSupportErr = fmt.Errorf(
				"settlement support line %d: invalid z",
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
			settlementSupportErr = fmt.Errorf(
				"settlement support line %d: empty required field",
				lineNumber,
			)
			return
		}
		if !validRequirement(point.Requirement) || !validInspection(point.Inspection) {
			settlementSupportErr = fmt.Errorf(
				"settlement support line %d: invalid requirement/inspection",
				lineNumber,
			)
			return
		}
		if _, exists := seen[point.ID]; exists {
			settlementSupportErr = fmt.Errorf(
				"settlement support line %d: duplicate id %s",
				lineNumber,
				point.ID,
			)
			return
		}
		seen[point.ID] = struct{}{}
		points = append(points, point)
	}
	if err := scanner.Err(); err != nil {
		settlementSupportErr = err
		return
	}
	if len(points) != 3 {
		settlementSupportErr = fmt.Errorf(
			"settlement support contains %d points, want 3",
			len(points),
		)
		return
	}
	settlementSupport = points
}
