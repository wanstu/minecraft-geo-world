package main

import (
	"flag"
	"fmt"
	"math"
	"os"

	"geoworld/internal/climate"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/phase1"
	"geoworld/internal/terrain"
)

type stat struct {
	min float64
	max float64
	sum float64
	n   int
}

func (s *stat) add(v float64) {
	if s.n == 0 || v < s.min {
		s.min = v
	}
	if s.n == 0 || v > s.max {
		s.max = v
	}
	s.sum += v
	s.n++
}

func (s stat) avg() float64 {
	if s.n == 0 {
		return 0
	}
	return s.sum / float64(s.n)
}

func main() {
	count := flag.Int("count", 4, "number of seeds")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	cellSize := flag.Float64("cell-size", 128, "Minecraft blocks per macro cell")
	flag.Parse()

	var minHeight, maxHeight, maxAbsDetail, maxValley stat
	var chunkJump, macroSlopeMismatch stat
	failed := false

	for n := 0; n < *count; n++ {
		seed := int64(7000 + n*6151)

		world := phase1.Generate(phase1.Config{
			Seed:          seed,
			Width:         *width,
			Height:        *height,
			PlateCount:    18,
			OceanFraction: 0.68,
		})
		c, err := climate.Generate(climate.InputFromPhase1(world), climate.Config{})
		if err != nil {
			panic(err)
		}
		h, err := hydrology.Generate(hydrology.InputFromWorld(world, c), hydrology.Config{})
		if err != nil {
			panic(err)
		}
		e, err := erosion.Generate(erosion.InputFromWorld(world, c, h), erosion.Config{})
		if err != nil {
			panic(err)
		}
		l, err := landform.Generate(landform.InputFromWorld(world, e), landform.Config{})
		if err != nil {
			panic(err)
		}
		g, err := geomorph.Generate(geomorph.InputFromModels(h, e, l), geomorph.Config{})
		if err != nil {
			panic(err)
		}
		m, err := terrain.Generate(terrain.InputFromModels(e, l, g), terrain.Config{
			Seed:           seed,
			CellSizeBlocks: *cellSize,
		})
		if err != nil {
			panic(err)
		}

		worldWidth := float64(*width-1) * *cellSize
		worldHeight := float64(*height-1) * *cellSize

		thisMin := math.Inf(1)
		thisMax := math.Inf(-1)
		thisDetail := 0.0
		thisValley := 0.0
		baseTopN := 0
		finalTopN := 0
		sampleN := 0

		for z := 0.0; z <= worldHeight; z += *cellSize {
			for x := 0.0; x <= worldWidth; x += *cellSize {
				s := m.Sample(x, z)
				if math.IsNaN(s.FinalHeight) || math.IsInf(s.FinalHeight, 0) {
					panic(fmt.Sprintf("seed %d invalid terrain at %.1f,%.1f", seed, x, z))
				}
				thisMin = math.Min(thisMin, s.FinalHeight)
				thisMax = math.Max(thisMax, s.FinalHeight)
				thisDetail = math.Max(thisDetail, math.Abs(s.RegionalDetail))
				thisValley = math.Max(thisValley, s.ValleyCarve)
				if s.BaseElevation >= 0.999 {
					baseTopN++
				}
				if s.FinalHeight >= 0.999 {
					finalTopN++
				}
				sampleN++
			}
		}

		jump := auditChunkSeams(m, worldWidth, worldHeight)
		slope := auditMacroSlopes(m, worldWidth, worldHeight, *cellSize)

		minHeight.add(thisMin)
		maxHeight.add(thisMax)
		maxAbsDetail.add(thisDetail)
		maxValley.add(thisValley)
		chunkJump.add(jump)
		macroSlopeMismatch.add(slope)

		if thisDetail > 0.06 || jump > 1e-5 || slope > 1e-4 {
			failed = true
		}
		fmt.Printf("seed=%d height=%.4f..%.4f detail<=%.5f valley<=%.5f top(base/final)=%.3f%%/%.3f%% chunk-jump=%.9f slope-mismatch=%.9f\n",
			seed, thisMin, thisMax, thisDetail, thisValley,
			percent(baseTopN, sampleN), percent(finalTopN, sampleN),
			jump, slope)
	}

	fmt.Printf("\nGeoWorld Phase 4 Terrain Audit: seeds=%d grid=%dx%d cell=%.0f blocks\n",
		*count, *width, *height, *cellSize)
	printStat("minimum height", minHeight)
	printStat("maximum height", maxHeight)
	printStat("maximum |regional detail|", maxAbsDetail)
	printStat("maximum valley carve", maxValley)
	printStat("chunk epsilon jump", chunkJump)
	printStat("macro slope mismatch", macroSlopeMismatch)

	if failed {
		fmt.Fprintln(os.Stderr, "terrain audit failed guardrails")
		os.Exit(1)
	}
}

func auditChunkSeams(m *terrain.Model, worldWidth, worldHeight float64) float64 {
	maxJump := 0.0
	eps := 1e-4

	// Sampling every 512 blocks still checks thousands of actual Minecraft
	// chunk boundaries without making the audit dominated by redundant samples.
	for x := 16.0; x < worldWidth; x += 512 {
		for z := 0.0; z <= worldHeight; z += math.Max(256, worldHeight/40) {
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

	for x := cellSize; x < worldWidth; x += cellSize * 8 {
		for z := cellSize; z < worldHeight-cellSize; z += cellSize * 4 {
			at := m.SampleNormalized(x, z)
			leftSlope := (at - m.SampleNormalized(x-h, z)) / h
			rightSlope := (m.SampleNormalized(x+h, z) - at) / h
			maxMismatch = math.Max(maxMismatch, math.Abs(leftSlope-rightSlope))
		}
	}
	return maxMismatch
}

func printStat(name string, s stat) {
	fmt.Printf("%-32s min=%12.9f avg=%12.9f max=%12.9f\n", name, s.min, s.avg(), s.max)
}

func percent(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}
