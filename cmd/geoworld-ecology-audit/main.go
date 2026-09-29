package main

import (
	"flag"
	"fmt"
	"os"

	"geoworld/internal/climate"
	"geoworld/internal/ecology"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
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
	count := flag.Int("count", 4, "number of seeds")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	flag.Parse()

	var coldPct, forestPct, dryPct, grassPct, tropicalPct, wetlandPct stat
	var lowConfidencePct, avgConfidence stat
	var avgTree, avgGrass, avgBare, avgSoil stat
	var frozenOceanPct stat

	for n := 0; n < *count; n++ {
		seed := int64(9200 + n*5479)

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
		eco, err := ecology.Generate(ecology.InputFromModels(c, h, e, l, g), ecology.Config{})
		if err != nil {
			panic(err)
		}

		landN := 0
		oceanN := 0
		coldN, forestN, dryN, grassN, tropicalN, wetlandN := 0, 0, 0, 0, 0, 0
		lowConfN := 0
		frozenOceanN := 0
		confSum, treeSum, grassSum, bareSum, soilSum := 0.0, 0.0, 0.0, 0.0, 0.0

		for i, region := range eco.EcoRegion {
			if !eco.LandMask[i] {
				oceanN++
				if region == ecology.RegionFrozenOcean {
					frozenOceanN++
				}
				continue
			}

			landN++
			confSum += eco.RegionConfidence[i]
			treeSum += eco.TreeCoverPotential[i]
			grassSum += eco.GrassCoverPotential[i]
			bareSum += eco.BareGroundPotential[i]
			soilSum += eco.SoilMoisture[i]
			if eco.RegionConfidence[i] < 0.20 {
				lowConfN++
			}

			switch region {
			case ecology.RegionIceSheet, ecology.RegionTundra, ecology.RegionAlpine:
				coldN++
			case ecology.RegionBorealForest, ecology.RegionTemperateForest,
				ecology.RegionSubtropicalForest, ecology.RegionTropicalMonsoonForest,
				ecology.RegionTropicalRainforest:
				forestN++
			case ecology.RegionTemperateDesert, ecology.RegionHotDesert:
				dryN++
			case ecology.RegionTemperateGrassland, ecology.RegionSavanna,
				ecology.RegionMediterraneanShrubland:
				grassN++
			case ecology.RegionWetland:
				wetlandN++
			}

			switch region {
			case ecology.RegionHotDesert, ecology.RegionSavanna,
				ecology.RegionTropicalMonsoonForest, ecology.RegionTropicalRainforest:
				tropicalN++
			}
		}

		cold := percent(coldN, landN)
		forest := percent(forestN, landN)
		dry := percent(dryN, landN)
		grass := percent(grassN, landN)
		tropical := percent(tropicalN, landN)
		wetland := percent(wetlandN, landN)
		lowConf := percent(lowConfN, landN)
		frozenOcean := percent(frozenOceanN, oceanN)

		coldPct.add(cold)
		forestPct.add(forest)
		dryPct.add(dry)
		grassPct.add(grass)
		tropicalPct.add(tropical)
		wetlandPct.add(wetland)
		lowConfidencePct.add(lowConf)
		avgConfidence.add(safeAvg(confSum, landN))
		avgTree.add(safeAvg(treeSum, landN))
		avgGrass.add(safeAvg(grassSum, landN))
		avgBare.add(safeAvg(bareSum, landN))
		avgSoil.add(safeAvg(soilSum, landN))
		frozenOceanPct.add(frozenOcean)

		fmt.Printf("seed=%d cold=%.1f%% forest=%.1f%% dry=%.1f%% grass=%.1f%% tropical=%.1f%% wetland=%.2f%% conf=%.3f low-conf=%.1f%% frozen-ocean=%.1f%%\n",
			seed, cold, forest, dry, grass, tropical, wetland,
			safeAvg(confSum, landN), lowConf, frozenOcean)
	}

	fmt.Printf("\nGeoWorld Phase 5.1 Ecology Audit: seeds=%d grid=%dx%d\n", *count, *width, *height)
	printStat("cold ecology land %", coldPct)
	printStat("forest ecology land %", forestPct)
	printStat("dry ecology land %", dryPct)
	printStat("grass/shrub ecology land %", grassPct)
	printStat("tropical ecology land %", tropicalPct)
	printStat("wetland ecology land %", wetlandPct)
	printStat("low confidence land %", lowConfidencePct)
	printStat("average region confidence", avgConfidence)
	printStat("average tree potential", avgTree)
	printStat("average grass potential", avgGrass)
	printStat("average bare-ground potential", avgBare)
	printStat("average soil moisture", avgSoil)
	printStat("frozen ocean %", frozenOceanPct)

	failed := false
	if coldPct.max > 40 {
		fmt.Fprintln(os.Stderr, "ecology audit failed: cold ecology dominates too much land")
		failed = true
	}
	if dryPct.max > 50 {
		fmt.Fprintln(os.Stderr, "ecology audit failed: dry ecology dominates too much land")
		failed = true
	}
	if forestPct.min < 10 {
		fmt.Fprintln(os.Stderr, "ecology audit failed: forest ecology collapsed across acceptance seed")
		failed = true
	}
	if grassPct.min < 10 {
		fmt.Fprintln(os.Stderr, "ecology audit failed: grass/shrub ecology collapsed across acceptance seed")
		failed = true
	}
	if lowConfidencePct.max > 10 || avgConfidence.min < 0.45 {
		fmt.Fprintln(os.Stderr, "ecology audit failed: too much ambiguous ecological classification")
		failed = true
	}
	if avgTree.max > 0.80 || avgBare.max > 0.80 {
		fmt.Fprintln(os.Stderr, "ecology audit failed: vegetation potentials saturated globally")
		failed = true
	}
	if failed {
		os.Exit(1)
	}
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
	fmt.Printf("%-34s min=%8.3f avg=%8.3f max=%8.3f\n", name, s.min, s.avg(), s.max)
}
