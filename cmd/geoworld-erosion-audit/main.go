package main

import (
	"flag"
	"fmt"

	"geoworld/internal/climate"
	"geoworld/internal/erosion"
	"geoworld/internal/hydrology"
	"geoworld/internal/phase1"
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
	count := flag.Int("count", 8, "number of seeds")
	width := flag.Int("width", 240, "macro grid width")
	height := flag.Int("height", 150, "macro grid height")
	flag.Parse()

	var avgErosion, maxErosion stat
	var valleyPct, hillslopePct stat
	var highReliefPct, avgRelief stat
	var strongErosionPct stat

	for n := 0; n < *count; n++ {
		seed := int64(5000 + n*6997)
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

		landN, valleyN, hillslopeN, highReliefN, strongN := 0, 0, 0, 0, 0
		sumE, sumR := 0.0, 0.0
		thisMaxE := 0.0

		for i, land := range e.LandMask {
			if !land {
				continue
			}
			landN++
			sumE += e.TotalErosion[i]
			sumR += e.LocalRelief[i]
			if e.TotalErosion[i] > thisMaxE {
				thisMaxE = e.TotalErosion[i]
			}
			if e.ValleyIncision[i] > 1e-4 {
				valleyN++
			}
			if e.HillslopeErosion[i] > 1e-4 {
				hillslopeN++
			}
			if e.ReliefIndex[i] >= 0.65 {
				highReliefN++
			}
			if e.TotalErosion[i] >= 0.015 {
				strongN++
			}
		}

		avgE := safeAvg(sumE, landN)
		avgR := safeAvg(sumR, landN)
		vPct := percent(valleyN, landN)
		hPct := percent(hillslopeN, landN)
		rPct := percent(highReliefN, landN)
		sPct := percent(strongN, landN)

		avgErosion.add(avgE)
		maxErosion.add(thisMaxE)
		valleyPct.add(vPct)
		hillslopePct.add(hPct)
		highReliefPct.add(rPct)
		avgRelief.add(avgR)
		strongErosionPct.add(sPct)

		fmt.Printf("seed=%d avg-erode=%.5f max=%.5f valley=%.2f%% hillslope=%.2f%% relief=%.2f%% strong=%.2f%%\n",
			seed, avgE, thisMaxE, vPct, hPct, rPct, sPct)
	}

	fmt.Printf("\nGeoWorld Phase 4.1 Erosion Audit: seeds=%d grid=%dx%d\n", *count, *width, *height)
	printStat("average erosion", avgErosion)
	printStat("maximum erosion", maxErosion)
	printStat("valley incision land %", valleyPct)
	printStat("hillslope erosion land %", hillslopePct)
	printStat("high relief land %", highReliefPct)
	printStat("average local relief", avgRelief)
	printStat("strong erosion >=0.015 land %", strongErosionPct)
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

func printStat(name string, s stat) {
	fmt.Printf("%-36s min=%8.5f avg=%8.5f max=%8.5f\n", name, s.min, s.avg(), s.max)
}
