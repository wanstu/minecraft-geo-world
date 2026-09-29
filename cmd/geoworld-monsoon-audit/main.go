package main

import (
	"flag"
	"fmt"

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
	count := flag.Int("count", 12, "number of seeds")
	width := flag.Int("width", 240, "macro grid width")
	height := flag.Int("height", 150, "macro grid height")
	flag.Parse()

	var annualRatio stat
	var seasonality stat
	var monsoonRain stat
	var weakWarmColdRatio stat
	var strongWarmColdRatio stat
	var strongMonsoonArea stat

	for n := 0; n < *count; n++ {
		seed := int64(3000 + n*8089)
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

		landN := 0
		annualSum := 0.0
		seasonalAnnualSum := 0.0
		seasonalitySum := 0.0
		monsoonRainSum := 0.0
		monsoonRainN := 0

		weakWarm, weakCold := 0.0, 0.0
		strongWarm, strongCold := 0.0, 0.0
		weakN, strongN := 0, 0
		seasonalitySaturated := 0

		for i := range model.AnnualPrecipMM {
			if world.Crust[i] != phase1.Continental {
				continue
			}
			landN++
			annualSum += model.AnnualPrecipMM[i]
			seasonalAnnualSum += model.SeasonalAnnualPrecipMM[i]
			seasonalitySum += model.RainfallSeasonality[i]
			if model.RainfallSeasonality[i] >= 0.999 {
				seasonalitySaturated++
			}
			if model.MonsoonRainRatio[i] > 0 {
				monsoonRainSum += model.MonsoonRainRatio[i]
				monsoonRainN++
			}

			switch {
			case model.MonsoonIndex[i] < 0.20:
				weakWarm += model.WarmSeasonPrecipMM[i]
				weakCold += model.ColdSeasonPrecipMM[i]
				weakN++
			case model.MonsoonIndex[i] >= 0.60:
				strongWarm += model.WarmSeasonPrecipMM[i]
				strongCold += model.ColdSeasonPrecipMM[i]
				strongN++
			}
		}

		if landN == 0 {
			continue
		}

		baseAvg := annualSum / float64(landN)
		seasonalAvg := seasonalAnnualSum / float64(landN)
		ratio := 0.0
		if baseAvg > 1 {
			ratio = seasonalAvg / baseAvg
		}
		avgSeasonality := seasonalitySum / float64(landN)
		avgMonsoonRain := 0.0
		if monsoonRainN > 0 {
			avgMonsoonRain = monsoonRainSum / float64(monsoonRainN)
		}

		weakRatio := 0.0
		if weakN > 0 && weakCold > 1 {
			weakRatio = (weakWarm / float64(weakN)) / (weakCold / float64(weakN))
		}
		strongRatio := 0.0
		if strongN > 0 && strongCold > 1 {
			strongRatio = (strongWarm / float64(strongN)) / (strongCold / float64(strongN))
		}
		strongArea := float64(strongN) / float64(landN) * 100

		annualRatio.add(ratio)
		seasonality.add(avgSeasonality)
		monsoonRain.add(avgMonsoonRain)
		if weakRatio > 0 {
			weakWarmColdRatio.add(weakRatio)
		}
		if strongRatio > 0 {
			strongWarmColdRatio.add(strongRatio)
		}
		strongMonsoonArea.add(strongArea)

		fmt.Printf("seed=%d seasonal/base=%.2f seasonality=%.3f sat=%.1f%% weakW/C=%.2f strongW/C=%.2f strong-area=%.1f%%\n",
			seed, ratio, avgSeasonality,
			float64(seasonalitySaturated)/float64(landN)*100,
			weakRatio, strongRatio, strongArea)
	}

	fmt.Printf("\nGeoWorld Phase 3.2 Monsoon Audit: seeds=%d grid=%dx%d\n", *count, *width, *height)
	printStat("seasonal annual / phase2 annual", annualRatio)
	printStat("rainfall seasonality avg", seasonality)
	printStat("monsoon rain ratio avg", monsoonRain)
	printStat("weak monsoon warm/cold ratio", weakWarmColdRatio)
	printStat("strong monsoon warm/cold ratio", strongWarmColdRatio)
	printStat("strong monsoon land area %", strongMonsoonArea)
}

func printStat(name string, s stat) {
	fmt.Printf("%-36s min=%8.3f avg=%8.3f max=%8.3f\n", name, s.min, s.avg(), s.max)
}
