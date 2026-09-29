package main

import (
	"flag"
	"fmt"
	"os"

	"geoworld/internal/climate"
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

	var floodPct, alluvialPct, fanPct, canyonPct, widePct, depositPct stat
	var maxFlood, maxAlluvial, maxFan, maxCanyon, maxWidth, maxDeposit stat

	for n := 0; n < *count; n++ {
		seed := int64(8100 + n*5923)

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

		landN := 0
		floodN, alluvialN, fanN, canyonN, wideN, depositN := 0, 0, 0, 0, 0, 0
		mf, ma, mfan, mc, mw, md := 0.0, 0.0, 0.0, 0.0, 0.0, 0.0

		for i, land := range g.LandMask {
			if !land {
				continue
			}
			landN++

			if g.FloodplainWeight[i] >= 0.35 {
				floodN++
			}
			if g.AlluvialPlainWeight[i] >= 0.30 {
				alluvialN++
			}
			if g.AlluvialFanWeight[i] >= 0.20 {
				fanN++
			}
			if g.CanyonWeight[i] >= 0.30 {
				canyonN++
			}
			if g.ValleyFloorWidth[i] >= 0.45 {
				wideN++
			}
			if g.DepositionPotential[i] >= 0.20 {
				depositN++
			}

			mf = max(mf, g.FloodplainWeight[i])
			ma = max(ma, g.AlluvialPlainWeight[i])
			mfan = max(mfan, g.AlluvialFanWeight[i])
			mc = max(mc, g.CanyonWeight[i])
			mw = max(mw, g.ValleyFloorWidth[i])
			md = max(md, g.DepositionPotential[i])
		}

		fp := percent(floodN, landN)
		ap := percent(alluvialN, landN)
		fnp := percent(fanN, landN)
		cp := percent(canyonN, landN)
		wp := percent(wideN, landN)
		dp := percent(depositN, landN)

		floodPct.add(fp)
		alluvialPct.add(ap)
		fanPct.add(fnp)
		canyonPct.add(cp)
		widePct.add(wp)
		depositPct.add(dp)

		maxFlood.add(mf)
		maxAlluvial.add(ma)
		maxFan.add(mfan)
		maxCanyon.add(mc)
		maxWidth.add(mw)
		maxDeposit.add(md)

		fmt.Printf("seed=%d flood=%.3f%% alluvial=%.3f%% fan=%.3f%% canyon=%.3f%% wide=%.3f%% deposit=%.3f%% max=[%.2f %.2f %.2f %.2f %.2f %.2f]\n",
			seed, fp, ap, fnp, cp, wp, dp, mf, ma, mfan, mc, mw, md)
	}

	fmt.Printf("\nGeoWorld Phase 4.4 Geomorph Audit: seeds=%d grid=%dx%d\n", *count, *width, *height)
	printStat("floodplain >=0.35 land %", floodPct)
	printStat("alluvial plain >=0.30 land %", alluvialPct)
	printStat("alluvial fan >=0.20 land %", fanPct)
	printStat("canyon >=0.30 land %", canyonPct)
	printStat("wide valley floor >=0.45 land %", widePct)
	printStat("deposition >=0.20 land %", depositPct)
	printStat("max floodplain weight", maxFlood)
	printStat("max alluvial plain weight", maxAlluvial)
	printStat("max alluvial fan weight", maxFan)
	printStat("max canyon weight", maxCanyon)
	printStat("max valley floor width", maxWidth)
	printStat("max deposition potential", maxDeposit)

	failed := false

	// Broad depositional forms should remain local/regional modifiers, not
	// replace most of the continent.
	if floodPct.max > 2.0 || alluvialPct.max > 8.0 || widePct.max > 3.0 || depositPct.max > 3.0 {
		fmt.Fprintln(os.Stderr, "geomorph audit failed: depositional landforms spread too broadly")
		failed = true
	}

	// A single seed is allowed to lack strong fans or canyons. Across the fixed
	// acceptance batch, however, both mechanisms must still be alive.
	if maxFan.max < 0.20 {
		fmt.Fprintln(os.Stderr, "geomorph audit failed: alluvial fans disappeared across acceptance seeds")
		failed = true
	}
	if maxCanyon.max < 0.30 {
		fmt.Fprintln(os.Stderr, "geomorph audit failed: canyon tendency disappeared across acceptance seeds")
		failed = true
	}

	for _, st := range []stat{maxFlood, maxAlluvial, maxFan, maxCanyon, maxWidth, maxDeposit} {
		if st.min < 0 || st.max > 1.000001 {
			fmt.Fprintln(os.Stderr, "geomorph audit failed: normalized field escaped 0..1")
			failed = true
			break
		}
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

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func printStat(name string, s stat) {
	fmt.Printf("%-38s min=%8.3f avg=%8.3f max=%8.3f\n", name, s.min, s.avg(), s.max)
}
