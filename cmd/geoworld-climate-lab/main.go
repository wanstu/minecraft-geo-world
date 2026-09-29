package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"geoworld/internal/climate"
	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	plates := flag.Int("plates", 18, "plate count")
	ocean := flag.Float64("ocean", 0.68, "target ocean area fraction")
	out := flag.String("out", "prototype/generated/climate", "output directory")
	flag.Parse()

	world := phase1.Generate(phase1.Config{
		Seed:          *seed,
		Width:         *width,
		Height:        *height,
		PlateCount:    *plates,
		OceanFraction: *ocean,
	})

	model, err := climate.Generate(climate.InputFromPhase1(world), climate.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := climate.RenderAll(model, *out); err != nil {
		log.Fatal(err)
	}

	minTemp := math.Inf(1)
	maxTemp := math.Inf(-1)
	minPressure := math.Inf(1)
	maxPressure := math.Inf(-1)
	minWind := math.Inf(1)
	maxWind := math.Inf(-1)
	landSum, oceanSum := 0.0, 0.0
	landN, oceanN := 0, 0
	for i, temp := range model.AnnualMeanTempC {
		if temp < minTemp {
			minTemp = temp
		}
		if temp > maxTemp {
			maxTemp = temp
		}
		if model.PressureIndex[i] < minPressure {
			minPressure = model.PressureIndex[i]
		}
		if model.PressureIndex[i] > maxPressure {
			maxPressure = model.PressureIndex[i]
		}
		wind := math.Hypot(model.WindX[i], model.WindZ[i])
		if wind < minWind {
			minWind = wind
		}
		if wind > maxWind {
			maxWind = wind
		}
		if world.Crust[i] == phase1.Continental {
			landSum += temp
			landN++
		} else {
			oceanSum += temp
			oceanN++
		}
	}

	fmt.Printf("GeoWorld Phase 2.2 Climate Lab\n")
	fmt.Printf("seed=%d grid=%dx%d latitude=±%.1f°\n", *seed, *width, *height, model.Config.MaxLatitudeDeg)
	fmt.Printf("annual-temp=%.2f..%.2f C land-avg=%.2f C ocean-avg=%.2f C\n",
		minTemp, maxTemp, landSum/float64(landN), oceanSum/float64(oceanN))
	fmt.Printf("equator=%.1f C polar=%.1f C lapse=%.1f C/km max-land-elevation=%.0f m\n",
		model.Config.EquatorTempC, model.Config.PolarTempC, model.Config.LapseRateCPerKm, model.Config.MaxLandElevationM)
	fmt.Printf("pressure-index=%.3f..%.3f wind-magnitude=%.3f..%.3f\n",
		minPressure, maxPressure, minWind, maxWind)
	fmt.Printf("output=%s\n", *out)
}
