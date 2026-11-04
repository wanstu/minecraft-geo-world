package main

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"geoworld/internal/generator"
	"geoworld/internal/runtimeworld"
	"geoworld/internal/transportgeo"
	"geoworld/internal/world"
)

func TestTransportGeoAcceptanceScan(t *testing.T) {
	const (
		width  = 320
		height = 200
		cell   = 128.0
	)
	g := world.GridGeometry{
		Width:       width,
		Height:      height,
		OriginX:     -float64(width-1) * cell / 2,
		OriginZ:     -float64(height-1) * cell / 2,
		SpanXBlocks: float64(width-1) * cell,
		SpanZBlocks: float64(height-1) * cell,
	}
	model, err := generator.GenerateWorld(context.Background(), generator.Config{
		Seed:          123456,
		Geometry:      g,
		WorldProfile:  generator.WorldProfileForPreset("standard"),
		PlateCount:    18,
		OceanFraction: 0.68,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rw, err := runtimeworld.NewGenerated(model)
	if err != nil {
		t.Fatal(err)
	}
	active := &activeWorld{
		seed:              model.Config.Seed,
		geometry:          model.Geometry,
		sample:            rw.Sample,
		runtimeWorld:      rw,
		parameterSnapshot: model.Config.WorldProfile.Parameters,
		model:             model,
	}
	active.previewGrid = buildPreviewGrid(active)
	active.settlementGrid = buildSettlementPreviewGrid(active)
	active.navigationGrid = buildNavigationPreviewGrid(active)
	active.transportGeoModel = buildTransportGeography(active)
	lakeOutlets, oceanMouths, lakeMouths := 0, 0, 0
	for _, seg := range model.Waterbed.Rivers {
		if seg.LakeOutlet {
			lakeOutlets++
		}
		if seg.OceanMouth {
			oceanMouths++
		}
		if seg.LakeMouth {
			lakeMouths++
		}
	}
	fmt.Printf("TRANSPORT_WATERBED rivers=%d lakes=%d lakeOutlets=%d oceanMouths=%d lakeMouths=%d hydrologyLakes=%d\n", len(model.Waterbed.Rivers), len(model.Waterbed.Lakes), lakeOutlets, oceanMouths, lakeMouths, len(model.Hydrology.Lakes))

	classCounts := map[string]int{}
	type cellRow struct {
		x, z                                                  float64
		y                                                     int
		class                                                 string
		score                                                 float64
		choke                                                 float64
		river, hill, mountain, plateau, basin, divide, strait float64
	}
	rows := make([]cellRow, 0, len(active.transportGeoModel.Scores))
	for gz := 0; gz < g.Height; gz++ {
		for gx := 0; gx < g.Width; gx++ {
			i := gz*g.Width + gx
			s := active.transportGeoModel.Scores[i]
			classCounts[s.CorridorClass]++
			x, z := g.NodePosition(gx, gz)
			sample := active.sample(x, z)
			rows = append(rows, cellRow{
				x: x, z: z, y: minecraft26TeleportY(sample) + 24,
				class: s.CorridorClass, score: s.CorridorSuitability,
				choke: s.ChokepointStrength,
				river: s.RiverValleyCorridor, hill: s.HillPassCorridor,
				mountain: s.MountainPassCorridor, plateau: s.PlateauAccessCorridor,
				basin: s.BasinGatewayCorridor, divide: s.LowDivideCorridor,
				strait: s.StraitIsthmusPotential,
			})
		}
	}
	fmt.Printf("TRANSPORT_CLASSES %v\n", classCounts)
	nodeCounts := map[string]int{}
	for _, node := range active.transportGeoModel.Nodes {
		nodeCounts[node.Class]++
	}
	fmt.Printf("TRANSPORT_NODES total=%d classes=%v\n", len(active.transportGeoModel.Nodes), nodeCounts)
	for _, class := range []string{
		transportgeo.NodeConfluence,
		transportgeo.NodeLakeOutlet,
		transportgeo.NodeEstuary,
		transportgeo.NodeMountainPass,
		transportgeo.NodeStrait,
		transportgeo.NodeIsthmus,
	} {
		if nodeCounts[class] == 0 {
			t.Fatalf("NODE acceptance class %q missing from fixed seed: %v", class, nodeCounts)
		}
	}
	nodeFraction := float64(len(active.transportGeoModel.Nodes)) /
		float64(len(active.transportGeoModel.Scores))
	if nodeFraction > 0.01 {
		t.Fatalf("NODE-06 density %.4f exceeds 1%% limit", nodeFraction)
	}

	routeSummary := active.transportGeoModel.RouteSummary
	fmt.Printf(
		"TRANSPORT_ROUTES total=%d complete=%d classes=%v completeClasses=%v longest=%.0f longestComplete=%.0f\n",
		routeSummary.TotalRoutes,
		routeSummary.CompleteRoutes,
		routeSummary.RoutesByClass,
		routeSummary.CompleteRoutesByClass,
		routeSummary.LongestRouteBlocks,
		routeSummary.LongestCompleteBlocks,
	)
	for _, route := range active.transportGeoModel.Routes {
		if route.Class == transportgeo.CorridorMountainPass ||
			route.Class == transportgeo.CorridorPlateauAccess {
			fmt.Printf(
				"TRANSPORT_ROUTE class=%s complete=%t cells=%d len=%.0f min=%.3f peak=%.3f coreInterior=%t plateauEdge=%t mountain=%.2f key=%s\n",
				route.Class,
				route.SemanticComplete,
				route.PathCellCount,
				route.LengthBlocks,
				route.MinSuitability,
				route.PeakSuitability,
				route.CoreOnInterior,
				route.CrossesPlateauEdge,
				route.MountainContext,
				route.Key,
			)
		}
	}
	for _, threshold := range []float64{0.50, 0.45, 0.40, 0.35} {
		mountain, plateau := 0, 0
		for _, score := range active.transportGeoModel.Scores {
			if score.MountainPassCorridor >= threshold {
				mountain++
			}
			if score.PlateauAccessCorridor >= threshold {
				plateau++
			}
		}
		fmt.Printf(
			"TRANSPORT_ROUTE_THRESHOLD threshold=%.2f mountain=%d plateau=%d\n",
			threshold,
			mountain,
			plateau,
		)
	}
	for _, class := range []string{
		transportgeo.CorridorRiverValley,
		transportgeo.CorridorHillPass,
		transportgeo.CorridorMountainPass,
		transportgeo.CorridorPlateauAccess,
		transportgeo.CorridorBasinGateway,
		transportgeo.CorridorLowDivide,
	} {
		if routeSummary.CompleteRoutesByClass[class] == 0 {
			t.Fatalf(
				"CORRIDOR acceptance class %q has no semantically complete route: %+v",
				class,
				routeSummary,
			)
		}
	}

	nodeCopy := append([]struct{}{}, []struct{}{}...)
	_ = nodeCopy
	nodes := append([]transportNodeScan(nil), make([]transportNodeScan, 0)...)
	_ = nodes

	type nodeRow struct {
		class       string
		x, z        float64
		suitability float64
		gx, gz      int
		reason      string
	}
	nr := make([]nodeRow, 0, len(active.transportGeoModel.Nodes))
	for _, n := range active.transportGeoModel.Nodes {
		nr = append(nr, nodeRow{class: n.Class, x: n.X, z: n.Z, suitability: n.Suitability, gx: n.GridX, gz: n.GridZ, reason: n.Reason})
	}
	sort.SliceStable(nr, func(i, j int) bool {
		if nr[i].class == nr[j].class {
			return nr[i].suitability > nr[j].suitability
		}
		return nr[i].class < nr[j].class
	})
	seen := map[string]int{}
	for _, n := range nr {
		if seen[n.class] >= 4 {
			continue
		}
		seen[n.class]++
		sample := active.sample(n.x, n.z)
		fmt.Printf("TRANSPORT_NODE class=%s rank=%d x=%.0f y=%d z=%.0f suitability=%.4f reason=%s\n",
			n.class, seen[n.class], n.x, minecraft26TeleportY(sample)+24, n.z, n.suitability, n.reason)
	}

	classes := []string{"river_valley", "hill_pass", "mountain_pass", "plateau_access", "basin_gateway", "low_divide"}
	for _, class := range classes {
		candidates := make([]cellRow, 0)
		for _, r := range rows {
			if r.class == class {
				candidates = append(candidates, r)
			}
		}
		sort.Slice(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
		for i := 0; i < 3 && i < len(candidates); i++ {
			r := candidates[i]
			fmt.Printf("TRANSPORT_CORRIDOR class=%s rank=%d x=%.0f y=%d z=%.0f score=%.4f choke=%.4f river=%.4f hill=%.4f mountain=%.4f plateau=%.4f basin=%.4f divide=%.4f strait=%.4f\n",
				class, i+1, r.x, r.y, r.z, r.score, r.choke, r.river, r.hill, r.mountain, r.plateau, r.basin, r.divide, r.strait)
		}
	}
}

type transportNodeScan struct{}
