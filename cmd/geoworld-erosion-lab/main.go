package main

import (
	"flag"
	"fmt"
	"log"

	"geoworld/internal/climate"
	"geoworld/internal/erosion"
	"geoworld/internal/hydrology"
	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	out := flag.String("out", "prototype/generated/erosion", "output directory")
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
	hydro, err := hydrology.Generate(hydrology.InputFromWorld(world, climateModel), hydrology.Config{})
	if err != nil {
		log.Fatal(err)
	}
	model, err := erosion.Generate(erosion.InputFromWorld(world, climateModel, hydro), erosion.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := erosion.RenderAll(model, *out); err != nil {
		log.Fatal(err)
	}

	landN := 0
	erodedN := 0
	valleyN := 0
	hillslopeN := 0
	highReliefN := 0
	sumErosion := 0.0
	sumRelief := 0.0
	maxErosion := 0.0
	maxValley := 0.0
	maxHillslope := 0.0
	maxRelief := 0.0

	for i, land := range model.LandMask {
		if !land {
			continue
		}
		landN++
		sumErosion += model.TotalErosion[i]
		sumRelief += model.LocalRelief[i]
		if model.TotalErosion[i] > 1e-4 {
			erodedN++
		}
		if model.ValleyIncision[i] > 1e-4 {
			valleyN++
		}
		if model.HillslopeErosion[i] > 1e-4 {
			hillslopeN++
		}
		if model.ReliefIndex[i] >= 0.65 {
			highReliefN++
		}
		if model.TotalErosion[i] > maxErosion {
			maxErosion = model.TotalErosion[i]
		}
		if model.ValleyIncision[i] > maxValley {
			maxValley = model.ValleyIncision[i]
		}
		if model.HillslopeErosion[i] > maxHillslope {
			maxHillslope = model.HillslopeErosion[i]
		}
		if model.LocalRelief[i] > maxRelief {
			maxRelief = model.LocalRelief[i]
		}
	}

	fmt.Printf("GeoWorld Phase 4.1 Erosion Lab\n")
	fmt.Printf("seed=%d grid=%dx%d\n", *seed, *width, *height)
	fmt.Printf("erosion avg=%.5f max=%.5f affected=%.2f%% land\n",
		safeAvg(sumErosion, landN), maxErosion, percent(erodedN, landN))
	fmt.Printf("valley affected=%.2f%% max-incision=%.5f hillslope affected=%.2f%% max=%.5f\n",
		percent(valleyN, landN), maxValley, percent(hillslopeN, landN), maxHillslope)
	fmt.Printf("relief avg=%.5f max=%.5f high-relief=%.2f%% land\n",
		safeAvg(sumRelief, landN), maxRelief, percent(highReliefN, landN))
	fmt.Printf("output=%s\n", *out)
}

func percent(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}

func safeAvg(sum float64, n int) float64 {
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}
