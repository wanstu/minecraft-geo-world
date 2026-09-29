package main

import (
	"flag"
	"fmt"

	"geoworld/internal/climate"
	"geoworld/internal/erosion"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
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

	var dominant [5]stat
	var mixedPct, valleyPct stat
	var plateau20Pct, foothillAvg stat

	for n := 0; n < *count; n++ {
		seed := int64(6000 + n*6637)
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
		m, err := landform.Generate(landform.InputFromWorld(world, e), landform.Config{})
		if err != nil {
			panic(err)
		}

		landN := 0
		counts := [5]int{}
		mixedN := 0
		valleyN := 0
		plateau20N := 0
		foothillSum := 0.0

		for i, land := range m.LandMask {
			if !land {
				continue
			}
			landN++
			weights := [5]float64{
				m.RollingPlainWeight[i],
				m.HillWeight[i],
				m.FoothillWeight[i],
				m.MountainWeight[i],
				m.PlateauWeight[i],
			}
			best := 0
			second := 0.0
			for j := 1; j < len(weights); j++ {
				if weights[j] > weights[best] {
					best = j
				}
			}
			for j, w := range weights {
				if j != best && w > second {
					second = w
				}
			}
			counts[best]++
			if weights[best]-second < 0.20 {
				mixedN++
			}
			if m.ValleyWeight[i] >= 0.45 {
				valleyN++
			}
			if m.PlateauWeight[i] >= 0.20 {
				plateau20N++
			}
			foothillSum += m.FoothillWeight[i]
		}

		for i := range dominant {
			dominant[i].add(percent(counts[i], landN))
		}
		mixedPct.add(percent(mixedN, landN))
		valleyPct.add(percent(valleyN, landN))
		plateau20Pct.add(percent(plateau20N, landN))
		foothillAvg.add(foothillSum / float64(landN))

		fmt.Printf("seed=%d plain=%.1f%% hill=%.1f%% foot=%.1f%% mountain=%.1f%% plateau=%.1f%% mixed=%.1f%% valley=%.2f%%\n",
			seed,
			percent(counts[0], landN), percent(counts[1], landN), percent(counts[2], landN),
			percent(counts[3], landN), percent(counts[4], landN),
			percent(mixedN, landN), percent(valleyN, landN))
	}

	names := []string{"plain dominant %", "hill dominant %", "foothill dominant %", "mountain dominant %", "plateau dominant %"}
	fmt.Printf("\nGeoWorld Phase 4.2 Landform Audit: seeds=%d grid=%dx%d\n", *count, *width, *height)
	for i, name := range names {
		printStat(name, dominant[i])
	}
	printStat("mixed transition %", mixedPct)
	printStat("strong valley overlay %", valleyPct)
	printStat("plateau weight >=0.20 %", plateau20Pct)
	printStat("average foothill weight", foothillAvg)
}

func percent(n, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(n) / float64(total) * 100
}

func printStat(name string, s stat) {
	fmt.Printf("%-34s min=%8.3f avg=%8.3f max=%8.3f\n", name, s.min, s.avg(), s.max)
}
