package advantage

import (
	"fmt"
	"math"
)

const (
	DefaultRegionPotentialThreshold = 0.55
	smallRegionCellLimit            = 4
)

// RegionConfig describes the macro grid used to aggregate per-cell advantage
// scores into contiguous, deterministic geographic regions.
type RegionConfig struct {
	Width, Height int
	OriginX       float64
	OriginZ       float64
	SpacingX      float64
	SpacingZ      float64
	MinPotential  float64
}

// Region is a connected component of eligible cells with the same dominant
// natural advantage. Its bounds and area are macro-grid estimates, not a
// claim that every block inside the bounding box belongs to the region.
type Region struct {
	ID                         int     `json:"id"`
	Key                        string  `json:"key"`
	DominantAdvantage          string  `json:"dominantAdvantage"`
	CellCount                  int     `json:"cellCount"`
	AreaBlocks2                float64 `json:"areaBlocks2"`
	CentroidX                  float64 `json:"centroidX"`
	CentroidZ                  float64 `json:"centroidZ"`
	MinGridX                   int     `json:"minGridX"`
	MaxGridX                   int     `json:"maxGridX"`
	MinGridZ                   int     `json:"minGridZ"`
	MaxGridZ                   int     `json:"maxGridZ"`
	MinWorldX                  float64 `json:"minWorldX"`
	MaxWorldX                  float64 `json:"maxWorldX"`
	MinWorldZ                  float64 `json:"minWorldZ"`
	MaxWorldZ                  float64 `json:"maxWorldZ"`
	MeanBuildingAdvantage      float64 `json:"meanBuildingAdvantage"`
	MeanAgricultureAdvantage   float64 `json:"meanAgricultureAdvantage"`
	MeanPortAdvantage          float64 `json:"meanPortAdvantage"`
	MeanMiningAdvantage        float64 `json:"meanMiningAdvantage"`
	MeanTransportAdvantage     float64 `json:"meanTransportAdvantage"`
	MeanSpecializationStrength float64 `json:"meanSpecializationStrength"`
	MeanOverlapRisk            float64 `json:"meanOverlapRisk"`
	HighOverlapCellFraction    float64 `json:"highOverlapCellFraction"`
	MultiAdvantageCellFraction float64 `json:"multiAdvantageCellFraction"`
}

// RegionSummary reports the scale and distribution of connected regions.
type RegionSummary struct {
	TotalRegions       int            `json:"totalRegions"`
	EligibleCells      int            `json:"eligibleCells"`
	SmallRegions       int            `json:"smallRegions"`
	SmallRegionCells   int            `json:"smallRegionCells"`
	LargestRegionID    int            `json:"largestRegionId"`
	LargestRegionCells int            `json:"largestRegionCells"`
	LargestRegionShare float64        `json:"largestRegionShare"`
	RegionsByAdvantage map[string]int `json:"regionsByAdvantage"`
}

// RegionModel contains the connected regions and a row-major cell-to-region
// mapping. Region ID 0 means the cell is not part of a potential region.
type RegionModel struct {
	Width         int           `json:"width"`
	Height        int           `json:"height"`
	CellRegionIDs []int         `json:"-"`
	Regions       []Region      `json:"regions"`
	Summary       RegionSummary `json:"summary"`
}

// BuildRegions groups eligible cells by dominant advantage using four-way
// connectivity. IDs are assigned in row-major discovery order, making the
// output stable for repeated analysis of the same locked grid.
func BuildRegions(scores []Score, cfg RegionConfig) RegionModel {
	if cfg.Width <= 0 || cfg.Height <= 0 ||
		len(scores) != cfg.Width*cfg.Height {
		return RegionModel{}
	}
	if !finitePositive(cfg.SpacingX) {
		cfg.SpacingX = 1
	}
	if !finitePositive(cfg.SpacingZ) {
		cfg.SpacingZ = 1
	}
	if math.IsNaN(cfg.OriginX) || math.IsInf(cfg.OriginX, 0) {
		cfg.OriginX = 0
	}
	if math.IsNaN(cfg.OriginZ) || math.IsInf(cfg.OriginZ, 0) {
		cfg.OriginZ = 0
	}
	if !finitePositive(cfg.MinPotential) {
		cfg.MinPotential = DefaultRegionPotentialThreshold
	}
	cfg.MinPotential = clamp01(cfg.MinPotential)

	model := RegionModel{
		Width:         cfg.Width,
		Height:        cfg.Height,
		CellRegionIDs: make([]int, len(scores)),
		Regions:       make([]Region, 0, 32),
		Summary: RegionSummary{
			RegionsByAdvantage: make(map[string]int, 5),
		},
	}
	queue := make([]int, 0, minInt(len(scores), 1024))
	for start, score := range scores {
		if model.CellRegionIDs[start] != 0 ||
			!regionEligible(score, cfg.MinPotential) {
			continue
		}

		id := len(model.Regions) + 1
		dominant := score.DominantAdvantage
		anchorGX := start % cfg.Width
		anchorGZ := start / cfg.Width
		region := Region{
			ID:                id,
			Key:               fmt.Sprintf("%s-g%04d-%04d", dominant, anchorGX, anchorGZ),
			DominantAdvantage: dominant,
			MinGridX:          cfg.Width,
			MinGridZ:          cfg.Height,
			MaxGridX:          -1,
			MaxGridZ:          -1,
			MinWorldX:         math.Inf(1),
			MinWorldZ:         math.Inf(1),
			MaxWorldX:         math.Inf(-1),
			MaxWorldZ:         math.Inf(-1),
		}

		queue = queue[:0]
		queue = append(queue, start)
		model.CellRegionIDs[start] = id
		for head := 0; head < len(queue); head++ {
			index := queue[head]
			gx := index % cfg.Width
			gz := index / cfg.Width
			cellScore := scores[index]
			accumulateRegionCell(&region, cellScore, gx, gz, cfg)

			neighbors := [4]int{-1, -1, -1, -1}
			if gx > 0 {
				neighbors[0] = index - 1
			}
			if gx+1 < cfg.Width {
				neighbors[1] = index + 1
			}
			if gz > 0 {
				neighbors[2] = index - cfg.Width
			}
			if gz+1 < cfg.Height {
				neighbors[3] = index + cfg.Width
			}
			for _, neighbor := range neighbors {
				if neighbor < 0 || model.CellRegionIDs[neighbor] != 0 {
					continue
				}
				next := scores[neighbor]
				if !regionEligible(next, cfg.MinPotential) ||
					next.DominantAdvantage != dominant {
					continue
				}
				model.CellRegionIDs[neighbor] = id
				queue = append(queue, neighbor)
			}
		}

		finalizeRegion(&region, cfg)
		model.Regions = append(model.Regions, region)
	}

	summary := &model.Summary
	summary.TotalRegions = len(model.Regions)
	for _, region := range model.Regions {
		summary.EligibleCells += region.CellCount
		summary.RegionsByAdvantage[region.DominantAdvantage]++
		if region.CellCount < smallRegionCellLimit {
			summary.SmallRegions++
			summary.SmallRegionCells += region.CellCount
		}
		if region.CellCount > summary.LargestRegionCells {
			summary.LargestRegionID = region.ID
			summary.LargestRegionCells = region.CellCount
		}
	}
	if summary.EligibleCells > 0 {
		summary.LargestRegionShare =
			float64(summary.LargestRegionCells) / float64(summary.EligibleCells)
	}
	return model
}

func (m RegionModel) Region(id int) (Region, bool) {
	if id <= 0 || id > len(m.Regions) {
		return Region{}, false
	}
	region := m.Regions[id-1]
	return region, region.ID == id
}

func regionEligible(score Score, minimum float64) bool {
	if score.RegionalClass == "" ||
		score.RegionalClass == "none" ||
		score.RegionalClass == "low_potential" {
		return false
	}
	if !isAdvantageLabel(score.DominantAdvantage) {
		return false
	}
	return max5(
		score.BuildingAdvantage,
		score.AgricultureAdvantage,
		score.PortAdvantage,
		score.MiningAdvantage,
		score.TransportAdvantage,
	) >= minimum
}

func isAdvantageLabel(value string) bool {
	switch value {
	case "building", "agriculture", "port", "mining", "transport":
		return true
	default:
		return false
	}
}

func accumulateRegionCell(
	region *Region,
	score Score,
	gx, gz int,
	cfg RegionConfig,
) {
	x := cfg.OriginX + float64(gx)*cfg.SpacingX
	z := cfg.OriginZ + float64(gz)*cfg.SpacingZ
	region.CellCount++
	region.CentroidX += x
	region.CentroidZ += z
	if gx < region.MinGridX {
		region.MinGridX = gx
		region.MinWorldX = x
	}
	if gx > region.MaxGridX {
		region.MaxGridX = gx
		region.MaxWorldX = x
	}
	if gz < region.MinGridZ {
		region.MinGridZ = gz
		region.MinWorldZ = z
	}
	if gz > region.MaxGridZ {
		region.MaxGridZ = gz
		region.MaxWorldZ = z
	}
	region.MeanBuildingAdvantage += score.BuildingAdvantage
	region.MeanAgricultureAdvantage += score.AgricultureAdvantage
	region.MeanPortAdvantage += score.PortAdvantage
	region.MeanMiningAdvantage += score.MiningAdvantage
	region.MeanTransportAdvantage += score.TransportAdvantage
	region.MeanSpecializationStrength += score.SpecializationStrength
	region.MeanOverlapRisk += score.AdvantageOverlapRisk
	if score.AdvantageOverlapRisk >= 0.70 {
		region.HighOverlapCellFraction++
	}
	if score.RegionalClass == "multi_advantage_region" {
		region.MultiAdvantageCellFraction++
	}
}

func finalizeRegion(region *Region, cfg RegionConfig) {
	if region.CellCount <= 0 {
		return
	}
	count := float64(region.CellCount)
	region.AreaBlocks2 = count * cfg.SpacingX * cfg.SpacingZ
	region.CentroidX /= count
	region.CentroidZ /= count
	region.MeanBuildingAdvantage /= count
	region.MeanAgricultureAdvantage /= count
	region.MeanPortAdvantage /= count
	region.MeanMiningAdvantage /= count
	region.MeanTransportAdvantage /= count
	region.MeanSpecializationStrength /= count
	region.MeanOverlapRisk /= count
	region.HighOverlapCellFraction /= count
	region.MultiAdvantageCellFraction /= count
}

func finitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
