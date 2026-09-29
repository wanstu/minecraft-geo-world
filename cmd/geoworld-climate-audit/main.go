package main

import (
	"flag"
	"fmt"
	"math"

	"geoworld/internal/climate"
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
	count := flag.Int("count", 16, "number of seeds")
	width := flag.Int("width", 320, "macro grid width")
	height := flag.Int("height", 200, "macro grid height")
	flag.Parse()

	var landAvg, dryPct, wetPct, satPct stat
	var tropicalAvg, temperateAvg, polarAvg stat

	for n := 0; n < *count; n++ {
		seed := int64(2000 + n*6151)
		world := phase1.Generate(phase1.Config{
			Seed:          seed,
			Width:         *width,
			Height:        *height,
			PlateCount:    18,
			OceanFraction: 0.68,
		})
		model, err := climate.Generate(climate.InputFromPhase1(world), climate.Config{})
		if err != nil {
			panic(err)
		}

		landSum := 0.0
		landN, dryN, wetN := 0, 0, 0
		saturated := 0

		zoneSum := [3]float64{}
		zoneN := [3]int{}

		for i, p := range model.AnnualPrecipMM {
			if math.IsNaN(p) || math.IsInf(p, 0) {
				panic(fmt.Sprintf("seed %d invalid precipitation at %d", seed, i))
			}
			if p >= 5999 {
				saturated++
			}
			if world.Crust[i] != phase1.Continental {
				continue
			}

			landSum += p
			landN++
			if p < 250 {
				dryN++
			}
			if p > 1500 {
				wetN++
			}

			absLat := math.Abs(model.LatitudeDeg[i])
			band := 2
			if absLat < 30 {
				band = 0
			} else if absLat < 60 {
				band = 1
			}
			zoneSum[band] += p
			zoneN[band]++
		}

		if landN == 0 {
			continue
		}
		landAvg.add(landSum / float64(landN))
		dryPct.add(float64(dryN) / float64(landN) * 100)
		wetPct.add(float64(wetN) / float64(landN) * 100)
		satPct.add(float64(saturated) / float64(len(model.AnnualPrecipMM)) * 100)

		if zoneN[0] > 0 {
			tropicalAvg.add(zoneSum[0] / float64(zoneN[0]))
		}
		if zoneN[1] > 0 {
			temperateAvg.add(zoneSum[1] / float64(zoneN[1]))
		}
		if zoneN[2] > 0 {
			polarAvg.add(zoneSum[2] / float64(zoneN[2]))
		}

		fmt.Printf("seed=%d land=%.0fmm dry=%.1f%% wet=%.1f%% sat=%.2f%%\n",
			seed,
			landSum/float64(landN),
			float64(dryN)/float64(landN)*100,
			float64(wetN)/float64(landN)*100,
			float64(saturated)/float64(len(model.AnnualPrecipMM))*100,
		)
	}

	fmt.Printf("\nGeoWorld Climate Audit: seeds=%d grid=%dx%d\n", *count, *width, *height)
	printStat("land precip mm", landAvg)
	printStat("dry land <250mm %", dryPct)
	printStat("wet land >1500mm %", wetPct)
	printStat("precip saturation % all cells", satPct)
	printStat("tropical land precip mm", tropicalAvg)
	printStat("temperate land precip mm", temperateAvg)
	printStat("polar land precip mm", polarAvg)
}

func printStat(name string, s stat) {
	fmt.Printf("%-34s min=%8.2f avg=%8.2f max=%8.2f\n", name, s.min, s.avg(), s.max)
}
