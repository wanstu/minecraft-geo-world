package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"geoworld/internal/climate"
	"geoworld/internal/hydrology"
	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	out := flag.String("out", "prototype/generated/hydrology", "output directory")
	flag.Parse()

	world := phase1.Generate(phase1.Config{
		Seed:          *seed,
		Width:         *width,
		Height:        *height,
		PlateCount:    18,
		OceanFraction: 0.68,
	})
	climateModel, err := climate.Generate(climate.InputFromPhase1(world), climate.Config{})
	if err != nil {
		log.Fatal(err)
	}
	model, err := hydrology.Generate(hydrology.InputFromWorld(world, climateModel), hydrology.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := hydrology.RenderAll(model, *out); err != nil {
		log.Fatal(err)
	}

	landN := 0
	riverN := 0
	lakeN := 0
	wetlandN := 0
	filledN := 0
	maxFill := 0.0
	maxAccum := 0.0
	runoffSum := 0.0
	riverStrengthSum := 0.0

	for i := range model.LandMask {
		if !model.LandMask[i] {
			continue
		}
		landN++
		runoffSum += model.LocalRunoffMM[i]
		if model.FillDepth[i] > 1e-6 {
			filledN++
			if model.FillDepth[i] > maxFill {
				maxFill = model.FillDepth[i]
			}
		}
		if model.FlowAccumulation[i] > maxAccum {
			maxAccum = model.FlowAccumulation[i]
		}
		if model.RiverStrength[i] > 0 {
			riverN++
			riverStrengthSum += model.RiverStrength[i]
		}
		if model.LakePotential[i] >= 0.35 {
			lakeN++
		}
		if model.WetlandPotential[i] >= 0.45 {
			wetlandN++
		}
	}

	avgRiverStrength := 0.0
	if riverN > 0 {
		avgRiverStrength = riverStrengthSum / float64(riverN)
	}
	avgRunoff := 0.0
	if landN > 0 {
		avgRunoff = runoffSum / float64(landN)
	}

	fmt.Printf("GeoWorld Phase 3.3 Hydrology Lab\n")
	fmt.Printf("seed=%d grid=%dx%d basins=%d major-basins=%d\n", *seed, *width, *height, model.BasinCount, model.MajorBasinCount)
	fmt.Printf("runoff land-avg=%.1f mm river-threshold=%.1f max-accum=%.1f\n",
		avgRunoff, model.RiverThreshold, maxAccum)
	fmt.Printf("river-cells=%d (%.2f%% land) avg-strength=%.3f\n",
		riverN, percent(riverN, landN), avgRiverStrength)
	fmt.Printf("filled-depressions=%d (%.2f%% land) max-fill-depth=%.4f\n",
		filledN, percent(filledN, landN), maxFill)
	fmt.Printf("lake-candidates=%d (%.2f%% land) wetland-candidates=%d (%.2f%% land)\n",
		lakeN, percent(lakeN, landN), wetlandN, percent(wetlandN, landN))
	fmt.Printf("output=%s\n", *out)

	if math.IsNaN(maxAccum) || math.IsInf(maxAccum, 0) {
		log.Fatal("invalid hydrology accumulation")
	}
}

func percent(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}
