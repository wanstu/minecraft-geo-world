package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	plates := flag.Int("plates", 18, "plate count")
	ocean := flag.Float64("ocean", 0.68, "target ocean area fraction")
	out := flag.String("out", "prototype/generated", "output directory")
	flag.Parse()

	cfg := phase1.Config{
		Seed:          *seed,
		Width:         *width,
		Height:        *height,
		PlateCount:    *plates,
		OceanFraction: *ocean,
	}
	world := phase1.Generate(cfg)
	if err := phase1.RenderAll(world, *out); err != nil {
		log.Fatal(err)
	}

	oceanCells := 0
	minAge := 1.0
	maxAge := 0.0
	influenced := 0
	for i, c := range world.Crust {
		if c == phase1.Oceanic {
			oceanCells++
			if world.OceanAge[i] < minAge {
				minAge = world.OceanAge[i]
			}
			if world.OceanAge[i] > maxAge {
				maxAge = world.OceanAge[i]
			}
		}
		if math.Abs(world.TectonicInfluence[i]) > 0.005 {
			influenced++
		}
	}
	oceanPct := float64(oceanCells) / float64(len(world.Crust)) * 100
	influencePct := float64(influenced) / float64(len(world.Crust)) * 100

	fmt.Printf("GeoWorld Phase 1 Lab\n")
	fmt.Printf("seed=%d grid=%dx%d plates=%d ocean-area=%.2f%%\n",
		cfg.Seed, cfg.Width, cfg.Height, len(world.Plates), oceanPct)
	fmt.Printf("ocean-age=%.3f..%.3f tectonic-influence=%.2f%% cells\n",
		minAge, maxAge, influencePct)
	fmt.Printf("output=%s\n", *out)
}
