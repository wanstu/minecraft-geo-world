package main

import (
	"flag"
	"fmt"
	"log"

	"geoworld/internal/climate"
	"geoworld/internal/erosion"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	out := flag.String("out", "prototype/generated/landform", "output directory")
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
	m, err := landform.Generate(landform.InputFromWorld(world, e), landform.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := landform.RenderAll(m, *out); err != nil {
		log.Fatal(err)
	}

	landN := 0
	dominant := [5]int{}
	avg := [6]float64{}
	mixedN := 0
	valleyN := 0
	highElevN := 0
	lowReliefN := 0
	plateauCandidateN := 0
	thickPlateauCandidateN := 0
	maxNormElevation := 0.0
	maxPlateauWeight := 0.0
	plateauWeight05N := 0
	plateauWeight20N := 0
	maxPlateauIndex := -1

	for i, land := range m.LandMask {
		if !land {
			continue
		}
		landN++
		normElevation := 0.0
		if e.ErodedElevation[i] > 0.5 {
			normElevation = (e.ErodedElevation[i] - 0.5) / 0.5
		}
		if normElevation > maxNormElevation {
			maxNormElevation = normElevation
		}
		if normElevation >= 0.27 {
			highElevN++
		}
		if e.ReliefIndex[i] <= 0.34 {
			lowReliefN++
		}
		if normElevation >= 0.27 && e.ReliefIndex[i] <= 0.34 {
			plateauCandidateN++
			if world.CrustThickness[i] >= 0.62 {
				thickPlateauCandidateN++
			}
		}

		if m.PlateauWeight[i] > maxPlateauWeight {
			maxPlateauWeight = m.PlateauWeight[i]
			maxPlateauIndex = i
		}
		if m.PlateauWeight[i] > 0.05 {
			plateauWeight05N++
		}
		if m.PlateauWeight[i] > 0.20 {
			plateauWeight20N++
		}

		weights := [5]float64{
			m.RollingPlainWeight[i],
			m.HillWeight[i],
			m.FoothillWeight[i],
			m.MountainWeight[i],
			m.PlateauWeight[i],
		}
		best := 0
		second := 0.0
		for j, w := range weights {
			avg[j] += w
			if w > weights[best] {
				best = j
			}
		}
		for j, w := range weights {
			if j != best && w > second {
				second = w
			}
		}
		dominant[best]++
		if weights[best]-second < 0.20 {
			mixedN++
		}
		avg[5] += m.ValleyWeight[i]
		if m.ValleyWeight[i] >= 0.45 {
			valleyN++
		}
	}

	names := []string{"plain", "hill", "foothill", "mountain", "plateau"}
	fmt.Printf("GeoWorld Phase 4.2 Landform Lab\n")
	fmt.Printf("seed=%d grid=%dx%d land=%d\n", *seed, *width, *height, landN)
	for i, name := range names {
		fmt.Printf("%-9s dominant=%6.2f%% avg-weight=%.3f\n",
			name, percent(dominant[i], landN), safeAvg(avg[i], landN))
	}
	fmt.Printf("mixed-transition=%.2f%% valley-overlay>=0.45=%.2f%% avg-valley=%.3f\n",
		percent(mixedN, landN), percent(valleyN, landN), safeAvg(avg[5], landN))
	fmt.Printf("plateau diagnostics high-elev=%.2f%% low-relief=%.2f%% both=%.2f%% thick-both=%.2f%% max-norm-elev=%.3f\n",
		percent(highElevN, landN), percent(lowReliefN, landN),
		percent(plateauCandidateN, landN), percent(thickPlateauCandidateN, landN),
		maxNormElevation)
	fmt.Printf("plateau-weight max=%.6f >0.05=%.2f%% >0.20=%.2f%%\n",
		maxPlateauWeight, percent(plateauWeight05N, landN), percent(plateauWeight20N, landN))
	if maxPlateauIndex >= 0 {
		i := maxPlateauIndex
		normElevation := 0.0
		if e.ErodedElevation[i] > 0.5 {
			normElevation = (e.ErodedElevation[i] - 0.5) / 0.5
		}
		fmt.Printf("max-plateau-cell i=%d elev=%.3f relief=%.3f slope=%.4f crust=%.3f support=%.3f weights=[plain %.3f hill %.3f foot %.3f mountain %.3f plateau %.3f]\n",
			i, normElevation, e.ReliefIndex[i], e.Slope[i], world.CrustThickness[i], e.TectonicSupport[i],
			m.RollingPlainWeight[i], m.HillWeight[i], m.FoothillWeight[i], m.MountainWeight[i], m.PlateauWeight[i])
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
