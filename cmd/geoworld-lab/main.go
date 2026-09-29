package main

import (
	"flag"
	"fmt"
	"log"

	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	plates := flag.Int("plates", 18, "plate count")
	ocean := flag.Float64("ocean", 0.68, "oceanic plate fraction")
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

	continental := 0
	for _, p := range world.Plates {
		if p.Crust == phase1.Continental {
			continental++
		}
	}

	fmt.Printf("GeoWorld Phase 1 Lab\n")
	fmt.Printf("seed=%d grid=%dx%d plates=%d continental=%d oceanic=%d\n",
		cfg.Seed, cfg.Width, cfg.Height, len(world.Plates), continental, len(world.Plates)-continental)
	fmt.Printf("output=%s\n", *out)
}
