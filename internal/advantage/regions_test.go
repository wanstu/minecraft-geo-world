package advantage

import (
	"math"
	"reflect"
	"testing"
)

func TestBuildRegionsGroupsFourConnectedCellsAndSummarizes(t *testing.T) {
	scores := []Score{
		regionTestScore("building", "building_region", .90, .10),
		regionTestScore("building", "building_region", .80, .20),
		regionTestScore("building", "low_potential", .30, .90),
		regionTestScore("port", "port_region", .88, .12),

		regionTestScore("building", "building_region", .85, .15),
		regionTestScore("building", "building_region", .82, .18),
		regionTestScore("building", "low_potential", .25, .95),
		regionTestScore("port", "port_region", .92, .08),

		regionTestScore("agriculture", "agriculture_region", .75, .25),
		regionTestScore("agriculture", "agriculture_region", .78, .22),
		regionTestScore("mining", "mining_region", .81, .30),
		regionTestScore("port", "port_region", .89, .11),
	}
	cfg := RegionConfig{
		Width: 4, Height: 3,
		OriginX: -64, OriginZ: -64,
		SpacingX: 128, SpacingZ: 128,
	}
	got := BuildRegions(scores, cfg)
	if len(got.Regions) != 4 {
		t.Fatalf("region count=%d want=4: %+v", len(got.Regions), got.Summary)
	}
	if got.Summary.EligibleCells != 10 ||
		got.Summary.LargestRegionID != 1 ||
		got.Summary.LargestRegionCells != 4 ||
		math.Abs(got.Summary.LargestRegionShare-.4) > 1e-9 {
		t.Fatalf("unexpected region summary: %+v", got.Summary)
	}
	building, ok := got.Region(1)
	if !ok {
		t.Fatal("region 1 missing")
	}
	if building.Key != "building-g0000-0000" ||
		building.DominantAdvantage != "building" ||
		building.CellCount != 4 ||
		building.AreaBlocks2 != 4*128*128 ||
		building.CentroidX != 0 ||
		building.CentroidZ != 0 {
		t.Fatalf("unexpected building region: %+v", building)
	}
	if building.MeanSpecializationStrength <= 0 ||
		building.MeanOverlapRisk <= 0 {
		t.Fatalf("region means not accumulated: %+v", building)
	}
	if got.CellRegionIDs[2] != 0 || got.CellRegionIDs[6] != 0 {
		t.Fatalf("low-potential cells should not be assigned: %v", got.CellRegionIDs)
	}
	if got.Summary.SmallRegions != 3 || got.Summary.SmallRegionCells != 6 {
		t.Fatalf("unexpected small-region counts: %+v", got.Summary)
	}

	again := BuildRegions(scores, cfg)
	if !reflect.DeepEqual(got, again) {
		t.Fatal("region IDs or statistics changed for identical model input")
	}
}

func TestBuildRegionsDoesNotConnectDiagonalCells(t *testing.T) {
	scores := []Score{
		regionTestScore("building", "building_region", .90, .10),
		{DominantAdvantage: "none", RegionalClass: "none"},
		{DominantAdvantage: "none", RegionalClass: "none"},
		regionTestScore("building", "building_region", .92, .12),
	}
	got := BuildRegions(scores, RegionConfig{
		Width: 2, Height: 2, SpacingX: 1, SpacingZ: 1,
	})
	if len(got.Regions) != 2 {
		t.Fatalf("diagonal cells must be separate four-connected regions: %+v", got.Regions)
	}
	if got.CellRegionIDs[0] != 1 || got.CellRegionIDs[3] != 2 {
		t.Fatalf("unstable or unexpected region IDs: %v", got.CellRegionIDs)
	}
}

func TestBuildRegionsRejectsInvalidGridShape(t *testing.T) {
	got := BuildRegions([]Score{{DominantAdvantage: "building", RegionalClass: "building_region"}}, RegionConfig{
		Width: 2, Height: 2,
	})
	if got.Width != 0 || len(got.Regions) != 0 || len(got.CellRegionIDs) != 0 {
		t.Fatalf("invalid grid shape should produce an empty model: %+v", got)
	}
}

func regionTestScore(dominant, class string, best, overlap float64) Score {
	score := Score{
		DominantAdvantage:      dominant,
		RegionalClass:          class,
		SpecializationStrength: best / 2,
		AdvantageOverlapRisk:   overlap,
	}
	switch dominant {
	case "building":
		score.BuildingAdvantage = best
	case "agriculture":
		score.AgricultureAdvantage = best
	case "port":
		score.PortAdvantage = best
	case "mining":
		score.MiningAdvantage = best
	case "transport":
		score.TransportAdvantage = best
	}
	return score
}
