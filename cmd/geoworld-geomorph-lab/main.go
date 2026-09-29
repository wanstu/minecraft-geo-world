package main

import (
	"flag"
	"fmt"
	"log"

	"geoworld/internal/climate"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	out := flag.String("out", "prototype/generated/geomorph", "output directory")
	flag.Parse()

	world := phase1.Generate(phase1.Config{
		Seed:          *seed,
		Width:         *width,
		Height:        *height,
		PlateCount:    18,
		OceanFraction: 0.68,
	})
	c, err := climate.Generate(climate.InputFromPhase1(world), climate.Config{})
	if err != nil {
		log.Fatal(err)
	}
	h, err := hydrology.Generate(hydrology.InputFromWorld(world, c), hydrology.Config{})
	if err != nil {
		log.Fatal(err)
	}
	e, err := erosion.Generate(erosion.InputFromWorld(world, c, h), erosion.Config{})
	if err != nil {
		log.Fatal(err)
	}
	l, err := landform.Generate(landform.InputFromWorld(world, e), landform.Config{})
	if err != nil {
		log.Fatal(err)
	}
	g, err := geomorph.Generate(geomorph.InputFromModels(h, e, l), geomorph.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := geomorph.RenderAll(g, *out); err != nil {
		log.Fatal(err)
	}

	landN := 0
	floodN := 0
	alluvialN := 0
	fanN := 0
	canyonN := 0
	wideN := 0
	depositionN := 0

	maxFlood := 0.0
	maxAlluvial := 0.0
	maxFan := 0.0
	maxCanyon := 0.0
	maxWidth := 0.0
	maxDeposit := 0.0
	maxSlopeDrop := 0.0
	maxFanIndex := -1
	maxCanyonIndex := -1

	for i, land := range g.LandMask {
		if !land {
			continue
		}
		landN++

		if g.FloodplainWeight[i] >= 0.35 {
			floodN++
		}
		if g.AlluvialPlainWeight[i] >= 0.30 {
			alluvialN++
		}
		if g.AlluvialFanWeight[i] >= 0.20 {
			fanN++
		}
		if g.CanyonWeight[i] >= 0.30 {
			canyonN++
		}
		if g.ValleyFloorWidth[i] >= 0.45 {
			wideN++
		}
		if g.DepositionPotential[i] >= 0.30 {
			depositionN++
		}

		maxFlood = max(maxFlood, g.FloodplainWeight[i])
		maxAlluvial = max(maxAlluvial, g.AlluvialPlainWeight[i])
		if g.AlluvialFanWeight[i] > maxFan {
			maxFan = g.AlluvialFanWeight[i]
			maxFanIndex = i
		}
		if g.CanyonWeight[i] > maxCanyon {
			maxCanyon = g.CanyonWeight[i]
			maxCanyonIndex = i
		}
		maxWidth = max(maxWidth, g.ValleyFloorWidth[i])
		maxDeposit = max(maxDeposit, g.DepositionPotential[i])
		maxSlopeDrop = max(maxSlopeDrop, g.SlopeDrop[i])
	}

	fmt.Printf("GeoWorld Phase 4.4 Geomorph Lab\n")
	fmt.Printf("seed=%d grid=%dx%d land=%d\n", *seed, *width, *height, landN)
	fmt.Printf("floodplain >=0.35  %6.3f%% max=%.3f\n", percent(floodN, landN), maxFlood)
	fmt.Printf("alluvial  >=0.30  %6.3f%% max=%.3f\n", percent(alluvialN, landN), maxAlluvial)
	fmt.Printf("fan       >=0.20  %6.3f%% max=%.3f\n", percent(fanN, landN), maxFan)
	fmt.Printf("canyon    >=0.30  %6.3f%% max=%.3f\n", percent(canyonN, landN), maxCanyon)
	fmt.Printf("wide-floor>=0.45  %6.3f%% max=%.3f\n", percent(wideN, landN), maxWidth)
	fmt.Printf("deposit   >=0.30  %6.3f%% max=%.3f\n", percent(depositionN, landN), maxDeposit)
	fmt.Printf("max slope-drop=%.3f\n", maxSlopeDrop)
	if maxFanIndex >= 0 {
		i := maxFanIndex
		fmt.Printf("max-fan i=%d river=%.3f accum=%.1f slope=%.4f up-slope=%.4f drop=%.3f supply=%.3f foot=%.3f mount=%.3f relief=%.3f fan=%.4f\n",
			i, h.RiverStrength[i], h.FlowAccumulation[i], e.Slope[i], g.UpstreamSlope[i],
			g.SlopeDrop[i], g.SedimentSupply[i], l.FoothillWeight[i], l.MountainWeight[i],
			e.ReliefIndex[i], g.AlluvialFanWeight[i])
		fmt.Printf("fan-memory steep=%.3f mountain=%.3f core=%.3f\n",
			g.UpstreamSteepMemory[i], g.UpstreamMountainMemory[i], g.AlluvialFanCore[i])
	}
	if maxCanyonIndex >= 0 {
		i := maxCanyonIndex
		fmt.Printf("max-canyon i=%d river=%.3f accum=%.1f slope=%.4f relief=%.3f incision=%.5f power=%.3f canyon=%.4f\n",
			i, h.RiverStrength[i], h.FlowAccumulation[i], e.Slope[i], e.ReliefIndex[i],
			e.ValleyIncision[i], e.FluvialPower[i], g.CanyonWeight[i])
	}
	fmt.Printf("output=%s\n", *out)
}

func percent(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
