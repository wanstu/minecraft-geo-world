package main

import (
	"flag"
	"fmt"

	"geoworld/internal/climate"
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

	var riverPct, fillPct, lakePct, wetlandPct stat
	var basinsPerK, majorBasins stat
	var maxFill stat
	var maxRiverOrder stat
	var order3Pct stat
	var seasonalDominantPct stat
	var explicitLakes stat
	var largestLakeCells stat
	var maxLakeDepth stat

	for n := 0; n < *count; n++ {
		seed := int64(4000 + n*7213)
		world := phase1.Generate(phase1.Config{
			Seed:          seed,
			Width:         *width,
			Height:        *height,
			PlateCount:    18,
			OceanFraction: 0.68,
		})
		climateModel, err := climate.Generate(climate.InputFromPhase1(world), climate.Config{})
		if err != nil {
			panic(err)
		}
		model, err := hydrology.Generate(hydrology.InputFromWorld(world, climateModel), hydrology.Config{})
		if err != nil {
			panic(err)
		}

		landN, riverN, fillN, lakeN, wetlandN := 0, 0, 0, 0, 0
		order3N := 0
		seasonalDominantN := 0
		thisMaxOrder := uint8(0)
		thisMaxFill := 0.0
		basinMaxAccum := map[int]float64{}

		for i := range model.LandMask {
			if !model.LandMask[i] {
				continue
			}
			landN++
			if model.RiverStrength[i] > 0 {
				riverN++
				if model.RiverOrder[i] > thisMaxOrder {
					thisMaxOrder = model.RiverOrder[i]
				}
				if model.RiverOrder[i] >= 3 {
					order3N++
				}
				if model.SeasonalRiverPotential[i] > model.PerennialRiverPotential[i] {
					seasonalDominantN++
				}
			}
			if model.FillDepth[i] > 1e-6 {
				fillN++
				if model.FillDepth[i] > thisMaxFill {
					thisMaxFill = model.FillDepth[i]
				}
			}
			if model.LakePotential[i] >= 0.35 {
				lakeN++
			}
			if model.WetlandPotential[i] >= 0.45 {
				wetlandN++
			}
			basin := model.BasinID[i]
			if basin > 0 && model.FlowAccumulation[i] > basinMaxAccum[basin] {
				basinMaxAccum[basin] = model.FlowAccumulation[i]
			}
		}

		major := 0
		for _, acc := range basinMaxAccum {
			if acc >= model.RiverThreshold {
				major++
			}
		}

		river := percent(riverN, landN)
		filled := percent(fillN, landN)
		lakes := percent(lakeN, landN)
		wetlands := percent(wetlandN, landN)
		bpk := float64(model.BasinCount) / float64(landN) * 1000

		largestLake := 0
		thisMaxLakeDepth := 0.0
		for _, lake := range model.Lakes {
			if lake.CellCount > largestLake {
				largestLake = lake.CellCount
			}
			if lake.MaxDepth > thisMaxLakeDepth {
				thisMaxLakeDepth = lake.MaxDepth
			}
		}

		order3 := percent(order3N, riverN)
		seasonalDominant := percent(seasonalDominantN, riverN)

		riverPct.add(river)
		fillPct.add(filled)
		lakePct.add(lakes)
		wetlandPct.add(wetlands)
		basinsPerK.add(bpk)
		majorBasins.add(float64(major))
		maxFill.add(thisMaxFill)
		maxRiverOrder.add(float64(thisMaxOrder))
		order3Pct.add(order3)
		seasonalDominantPct.add(seasonalDominant)
		explicitLakes.add(float64(len(model.Lakes)))
		largestLakeCells.add(float64(largestLake))
		maxLakeDepth.add(thisMaxLakeDepth)

		fmt.Printf("seed=%d river=%.2f%% fill=%.2f%% basins=%d major=%d order=%d order3=%.1f%% seasonal=%.1f%% lakes=%d largest=%d max-depth=%.3f\n",
			seed, river, filled, model.BasinCount, major, thisMaxOrder, order3, seasonalDominant,
			len(model.Lakes), largestLake, thisMaxLakeDepth)
	}

	fmt.Printf("\nGeoWorld Phase 3.4 Hydrology Audit: seeds=%d grid=%dx%d\n", *count, *width, *height)
	printStat("river land %", riverPct)
	printStat("filled land %", fillPct)
	printStat("lake candidate land %", lakePct)
	printStat("wetland candidate land %", wetlandPct)
	printStat("basins per 1000 land cells", basinsPerK)
	printStat("major river basins", majorBasins)
	printStat("max normalized fill depth", maxFill)
	printStat("max Strahler river order", maxRiverOrder)
	printStat("order >=3 river cells %", order3Pct)
	printStat("seasonal-dominant river cells %", seasonalDominantPct)
	printStat("explicit lakes", explicitLakes)
	printStat("largest lake cells", largestLakeCells)
	printStat("max explicit lake depth", maxLakeDepth)
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
