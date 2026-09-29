package main

import (
	"flag"
	"fmt"
	"math"

	"geoworld/internal/phase1"
)

type rangeStat struct {
	min float64
	max float64
	sum float64
	n   int
}

func (r *rangeStat) add(v float64) {
	if r.n == 0 || v < r.min {
		r.min = v
	}
	if r.n == 0 || v > r.max {
		r.max = v
	}
	r.sum += v
	r.n++
}

func (r rangeStat) avg() float64 {
	if r.n == 0 {
		return 0
	}
	return r.sum / float64(r.n)
}

func main() {
	count := flag.Int("count", 32, "number of seeds")
	width := flag.Int("width", 320, "macro grid width")
	height := flag.Int("height", 200, "macro grid height")
	plates := flag.Int("plates", 18, "plate count")
	ocean := flag.Float64("ocean", 0.68, "target ocean fraction")
	flag.Parse()

	var edgeBuffer rangeStat
	var mountains, plateau, trench, ridge, rift rangeStat
	var elevationSat, intensitySat rangeStat

	failures := 0

	for n := 0; n < *count; n++ {
		seed := int64(1000 + n*7919)
		cfg := phase1.Config{
			Seed:          seed,
			Width:         *width,
			Height:        *height,
			PlateCount:    *plates,
			OceanFraction: *ocean,
		}
		w := phase1.Generate(cfg)
		total := float64(len(w.Crust))

		minLandEdge := math.MaxInt
		zoneCounts := map[phase1.MacroTerrain]int{}
		satElevation := 0
		satIntensity := 0
		featureCells := 0

		for y := 0; y < cfg.Height; y++ {
			for x := 0; x < cfg.Width; x++ {
				i := y*cfg.Width + x
				if w.Crust[i] == phase1.Continental {
					d := min4(x, y, cfg.Width-1-x, cfg.Height-1-y)
					if d < minLandEdge {
						minLandEdge = d
					}
				}
				zoneCounts[w.TerrainZone[i]]++
				if w.Elevation[i] >= 0.999 {
					satElevation++
				}
				if w.Feature[i] != phase1.NoFeature {
					featureCells++
					if w.TectonicIntensity[i] >= 0.999 {
						satIntensity++
					}
				}
			}
		}

		if minLandEdge == math.MaxInt {
			minLandEdge = 0
		}
		edgeBuffer.add(float64(minLandEdge))
		mountains.add(float64(zoneCounts[phase1.MountainBelt]) / total * 100)
		plateau.add(float64(zoneCounts[phase1.Plateau]) / total * 100)
		trench.add(float64(zoneCounts[phase1.TrenchZone]) / total * 100)
		ridge.add(float64(zoneCounts[phase1.MidOceanRidgeZone]) / total * 100)
		rift.add(float64(zoneCounts[phase1.RiftValleyZone]) / total * 100)
		elevationSat.add(float64(satElevation) / total * 100)

		if featureCells > 0 {
			intensitySat.add(float64(satIntensity) / float64(featureCells) * 100)
		} else {
			intensitySat.add(0)
		}

		if minLandEdge < 3 {
			fmt.Printf("WARN seed=%d land-edge-buffer=%d cells\n", seed, minLandEdge)
			failures++
		}
	}

	fmt.Printf("GeoWorld Phase 1 audit: seeds=%d grid=%dx%d\n", *count, *width, *height)
	printStat("land edge buffer cells", edgeBuffer)
	printStat("mountain %%", mountains)
	printStat("plateau %%", plateau)
	printStat("trench %%", trench)
	printStat("ridge %%", ridge)
	printStat("rift %%", rift)
	printStat("elevation saturation %%", elevationSat)
	printStat("tectonic saturation %% feature cells", intensitySat)
	fmt.Printf("edge-buffer-warnings=%d\n", failures)
}

func printStat(name string, s rangeStat) {
	fmt.Printf("%-36s min=%8.3f avg=%8.3f max=%8.3f\n", name, s.min, s.avg(), s.max)
}

func min4(a, b, c, d int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	if d < m {
		m = d
	}
	return m
}
