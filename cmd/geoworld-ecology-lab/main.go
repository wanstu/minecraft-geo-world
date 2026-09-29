package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"sort"

	"geoworld/internal/climate"
	"geoworld/internal/ecology"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	out := flag.String("out", "prototype/generated/ecology", "output directory")
	flag.Parse()

	world := phase1.Generate(phase1.Config{
		Seed:          *seed,
		Width:         *width,
		Height:        *height,
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
	eco, err := ecology.Generate(ecology.InputFromModels(c, h, e, l, g), ecology.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := ecology.RenderAll(eco, *out); err != nil {
		log.Fatal(err)
	}

	counts := map[ecology.Region]int{}
	landN := 0
	oceanN := 0
	lowConfidence := 0
	sumConfidence := 0.0
	sumTree := 0.0
	sumGrass := 0.0
	sumBare := 0.0
	sumSoil := 0.0
	minTemp := 1e9
	maxTemp := -1e9
	sumTemp := 0.0
	sumLatitudeBaseTemp := 0.0
	sumLandElevationM := 0.0
	belowMinus10 := 0
	belowMinus5 := 0
	belowZero := 0
	above20 := 0
	highLatitudeLand := 0
	veryDryLand := 0

	for i, r := range eco.EcoRegion {
		counts[r]++
		if eco.LandMask[i] {
			landN++
			sumConfidence += eco.RegionConfidence[i]
			sumTree += eco.TreeCoverPotential[i]
			sumGrass += eco.GrassCoverPotential[i]
			sumBare += eco.BareGroundPotential[i]
			sumSoil += eco.SoilMoisture[i]
			if eco.RegionConfidence[i] < 0.20 {
				lowConfidence++
			}
			temp := c.AdjustedAnnualTempC[i]
			if temp < minTemp {
				minTemp = temp
			}
			if temp > maxTemp {
				maxTemp = temp
			}
			sumTemp += temp
			sumLatitudeBaseTemp += c.LatitudeBaseTempC[i]
			sumLandElevationM += c.LandElevationM[i]
			if temp < -10 {
				belowMinus10++
			}
			if temp < -5 {
				belowMinus5++
			}
			if temp < 0 {
				belowZero++
			}
			if temp > 20 {
				above20++
			}
			if math.Abs(c.LatitudeDeg[i]) >= 60 {
				highLatitudeLand++
			}
			if eco.MoistureIndex[i] < 0.25 {
				veryDryLand++
			}
		} else {
			oceanN++
		}
	}

	fmt.Printf("GeoWorld Phase 5.1 Ecology Lab\n")
	fmt.Printf("seed=%d grid=%dx%d land=%d ocean=%d\n", *seed, *width, *height, landN, oceanN)
	fmt.Printf("land avg confidence=%.3f low-confidence<0.20=%.2f%%\n",
		safeAvg(sumConfidence, landN), percent(lowConfidence, landN))
	fmt.Printf("land avg tree=%.3f grass=%.3f bare=%.3f soil-moisture=%.3f\n",
		safeAvg(sumTree, landN), safeAvg(sumGrass, landN), safeAvg(sumBare, landN), safeAvg(sumSoil, landN))
	fmt.Printf("land temp min=%.2f avg=%.2f max=%.2f C latitude-only-avg=%.2f C avg-elevation=%.0f m\n",
		minTemp, safeAvg(sumTemp, landN), maxTemp,
		safeAvg(sumLatitudeBaseTemp, landN), safeAvg(sumLandElevationM, landN))
	fmt.Printf("land temp <-10=%.2f%% <-5=%.2f%% <0=%.2f%% >20=%.2f%% high-lat>=60=%.2f%% moisture<0.25=%.2f%%\n",
		percent(belowMinus10, landN), percent(belowMinus5, landN), percent(belowZero, landN),
		percent(above20, landN), percent(highLatitudeLand, landN), percent(veryDryLand, landN))

	regions := make([]ecology.Region, 0, len(counts))
	for r := range counts {
		regions = append(regions, r)
	}
	sort.Slice(regions, func(i, j int) bool { return regions[i] < regions[j] })

	for _, r := range regions {
		n := counts[r]
		denom := landN
		if r <= ecology.RegionFrozenOcean {
			denom = oceanN
		}
		fmt.Printf("%-28s %6d %6.2f%%\n", r.String(), n, percent(n, denom))
	}
	fmt.Printf("output=%s\n", *out)
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
