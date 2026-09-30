package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"geoworld/internal/climate"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/phase1"
	"geoworld/internal/terrain"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	cellSize := flag.Float64("cell-size", 128, "Minecraft blocks per macro cell")
	out := flag.String("out", "prototype/generated/terrain", "output directory")
	flag.Parse()

	world := phase1.Generate(phase1.Config{
		Seed:          *seed,
		Width:         *width,
		Height:        *height,
		SpanXBlocks:   float64(*width-1) * *cellSize,
		SpanZBlocks:   float64(*height-1) * *cellSize,
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
	m, err := terrain.Generate(terrain.InputFromModels(e, l, g), terrain.Config{
		Seed: *seed,
	})
	if err != nil {
		log.Fatal(err)
	}

	if err := terrain.RenderAll(m, *out, terrain.RenderConfig{WidthPx: 960, HeightPx: 600}); err != nil {
		log.Fatal(err)
	}

	minH, maxH := math.Inf(1), math.Inf(-1)
	minDetail, maxDetail := math.Inf(1), math.Inf(-1)
	maxValley := 0.0
	sumH := 0.0
	samples := 0

	worldWidth := world.Geometry.SpanXBlocks
	worldHeight := world.Geometry.SpanZBlocks

	for z := 0.0; z <= worldHeight; z += *cellSize / 2 {
		for x := 0.0; x <= worldWidth; x += *cellSize / 2 {
			s := m.Sample(x, z)
			minH = math.Min(minH, s.FinalHeight)
			maxH = math.Max(maxH, s.FinalHeight)
			minDetail = math.Min(minDetail, s.RegionalDetail)
			maxDetail = math.Max(maxDetail, s.RegionalDetail)
			maxValley = math.Max(maxValley, s.ValleyCarve)
			sumH += s.FinalHeight
			samples++
		}
	}

	chunkJump := auditChunkSeams(m, worldWidth, worldHeight)
	macroSlopeMismatch := auditMacroSlopes(m, worldWidth, worldHeight, *cellSize)

	fmt.Printf("GeoWorld Phase 4 Integrated Terrain Lab\n")
	fmt.Printf("seed=%d grid=%dx%d cell-size=%.0f blocks world≈%.0fx%.0f blocks\n",
		*seed, *width, *height, *cellSize, worldWidth, worldHeight)
	fmt.Printf("height min=%.5f avg=%.5f max=%.5f\n", minH, sumH/float64(samples), maxH)
	fmt.Printf("regional-detail min=%.5f max=%.5f max-valley-carve=%.5f\n",
		minDetail, maxDetail, maxValley)
	fmt.Printf("max chunk-boundary epsilon jump=%.12f\n", chunkJump)
	fmt.Printf("max macro-boundary slope mismatch=%.12f\n", macroSlopeMismatch)
	fmt.Printf("output=%s\n", *out)
}

func auditChunkSeams(m *terrain.Model, worldWidth, worldHeight float64) float64 {
	maxJump := 0.0
	eps := 1e-4
	stepZ := math.Max(64, worldHeight/80)

	for x := 16.0; x < worldWidth; x += 16 {
		for z := 0.0; z <= worldHeight; z += stepZ {
			at := m.SampleNormalized(x, z)
			maxJump = math.Max(maxJump, math.Abs(m.SampleNormalized(x-eps, z)-at))
			maxJump = math.Max(maxJump, math.Abs(m.SampleNormalized(x+eps, z)-at))
		}
	}
	return maxJump
}

func auditMacroSlopes(m *terrain.Model, worldWidth, worldHeight, cellSize float64) float64 {
	maxMismatch := 0.0
	h := 0.25
	stepZ := math.Max(cellSize, worldHeight/50)

	for x := cellSize; x < worldWidth; x += cellSize {
		for z := cellSize; z < worldHeight-cellSize; z += stepZ {
			at := m.SampleNormalized(x, z)
			leftSlope := (at - m.SampleNormalized(x-h, z)) / h
			rightSlope := (m.SampleNormalized(x+h, z) - at) / h
			maxMismatch = math.Max(maxMismatch, math.Abs(leftSlope-rightSlope))
		}
	}
	return maxMismatch
}
