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
	"geoworld/internal/mcprofile"
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

	var lowConfPct, avgConfidence stat
	var rollingPct, hillPct, mountainPct, fluvialPct stat
	var forestSurfacePct, grassSurfacePct, aridSurfacePct, snowSurfacePct stat
	var forestVegPct, openVegPct, desertVegPct, coldVegPct, wetVegPct stat

	totalUnknown := 0

	for n := 0; n < *count; n++ {
		seed := int64(10400 + n*5323)

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
		m, err := mcprofile.Generate(mcprofile.InputFromModels(eco, l, g))
		if err != nil {
			panic(err)
		}

		landN := 0
		lowConfN := 0
		confSum := 0.0
		rollingN, hillN, mountainN, fluvialN := 0, 0, 0, 0
		forestSurfaceN, grassSurfaceN, aridSurfaceN, snowSurfaceN := 0, 0, 0, 0
		forestVegN, openVegN, desertVegN, coldVegN, wetVegN := 0, 0, 0, 0, 0
		unknownN := 0

		for i := range m.BiomeFamily {
			if m.BiomeFamily[i] == mcprofile.BiomeUnknown ||
				m.TerrainVariant[i] == mcprofile.TerrainUnknown ||
				m.SurfaceProfile[i] == mcprofile.SurfaceUnknown ||
				m.VegetationProfile[i] == mcprofile.VegetationUnknown {
				unknownN++
			}
			if !m.LandMask[i] {
				continue
			}

			landN++
			confSum += m.ProfileConfidence[i]
			if m.ProfileConfidence[i] < 0.30 {
				lowConfN++
			}

			switch m.TerrainVariant[i] {
			case mcprofile.TerrainPlain, mcprofile.TerrainRollingPlain:
				rollingN++
			case mcprofile.TerrainHills, mcprofile.TerrainFoothills:
				hillN++
			case mcprofile.TerrainMountain, mcprofile.TerrainPlateau:
				mountainN++
			case mcprofile.TerrainFloodplain, mcprofile.TerrainAlluvialPlain,
				mcprofile.TerrainAlluvialFan, mcprofile.TerrainCanyon:
				fluvialN++
			}

			switch m.SurfaceProfile[i] {
			case mcprofile.SurfaceForestLoam, mcprofile.SurfacePodzol:
				forestSurfaceN++
			case mcprofile.SurfaceGrassLoam, mcprofile.SurfaceDryGrass, mcprofile.SurfaceShrubSoil:
				grassSurfaceN++
			case mcprofile.SurfaceDesertSand, mcprofile.SurfaceDesertGravel:
				aridSurfaceN++
			case mcprofile.SurfaceIceSnow, mcprofile.SurfaceAlpineSnow, mcprofile.SurfaceAlpineRock,
				mcprofile.SurfaceTundraSoil:
				snowSurfaceN++
			}

			switch m.VegetationProfile[i] {
			case mcprofile.VegetationConiferForest, mcprofile.VegetationMixedForest,
				mcprofile.VegetationBroadleafForest, mcprofile.VegetationDenseRainforest,
				mcprofile.VegetationMonsoonForest:
				forestVegN++
			case mcprofile.VegetationGrassland, mcprofile.VegetationDryGrassland,
				mcprofile.VegetationShrubland, mcprofile.VegetationSavanna:
				openVegN++
			case mcprofile.VegetationDesertSparse:
				desertVegN++
			case mcprofile.VegetationNone, mcprofile.VegetationTundraSparse,
				mcprofile.VegetationAlpineSparse:
				coldVegN++
			case mcprofile.VegetationWetland:
				wetVegN++
			}
		}

		totalUnknown += unknownN

		lowConf := percent(lowConfN, landN)
		conf := safeAvg(confSum, landN)
		rolling := percent(rollingN, landN)
		hills := percent(hillN, landN)
		mountains := percent(mountainN, landN)
		fluvial := percent(fluvialN, landN)
		forestSurface := percent(forestSurfaceN, landN)
		grassSurface := percent(grassSurfaceN, landN)
		aridSurface := percent(aridSurfaceN, landN)
		snowSurface := percent(snowSurfaceN, landN)
		forestVeg := percent(forestVegN, landN)
		openVeg := percent(openVegN, landN)
		desertVeg := percent(desertVegN, landN)
		coldVeg := percent(coldVegN, landN)
		wetVeg := percent(wetVegN, landN)

		lowConfPct.add(lowConf)
		avgConfidence.add(conf)
		rollingPct.add(rolling)
		hillPct.add(hills)
		mountainPct.add(mountains)
		fluvialPct.add(fluvial)
		forestSurfacePct.add(forestSurface)
		grassSurfacePct.add(grassSurface)
		aridSurfacePct.add(aridSurface)
		snowSurfacePct.add(snowSurface)
		forestVegPct.add(forestVeg)
		openVegPct.add(openVeg)
		desertVegPct.add(desertVeg)
		coldVegPct.add(coldVeg)
		wetVegPct.add(wetVeg)

		fmt.Printf("seed=%d conf=%.3f low=%.2f%% terrain[p=%.1f h=%.1f m=%.1f f=%.2f] surface[forest=%.1f grass=%.1f arid=%.1f snow=%.1f] veg[forest=%.1f open=%.1f desert=%.1f cold=%.1f wet=%.2f] unknown=%d\n",
			seed, conf, lowConf,
			rolling, hills, mountains, fluvial,
			forestSurface, grassSurface, aridSurface, snowSurface,
			forestVeg, openVeg, desertVeg, coldVeg, wetVeg, unknownN)
	}

	fmt.Printf("\nGeoWorld Phase 5.2 Minecraft Profile Audit: seeds=%d grid=%dx%d\n", *count, *width, *height)
	printStat("low-confidence land %", lowConfPct)
	printStat("average profile confidence", avgConfidence)
	printStat("plain terrain land %", rollingPct)
	printStat("hill/foothill terrain land %", hillPct)
	printStat("mountain/plateau terrain land %", mountainPct)
	printStat("fluvial terrain land %", fluvialPct)
	printStat("forest surface land %", forestSurfacePct)
	printStat("grass/shrub surface land %", grassSurfacePct)
	printStat("arid surface land %", aridSurfacePct)
	printStat("snow/alpine surface land %", snowSurfacePct)
	printStat("forest vegetation land %", forestVegPct)
	printStat("open vegetation land %", openVegPct)
	printStat("desert vegetation land %", desertVegPct)
	printStat("cold/sparse vegetation land %", coldVegPct)
	printStat("wetland vegetation land %", wetVegPct)

	failed := false
	if totalUnknown != 0 {
		fmt.Fprintf(os.Stderr, "mcprofile audit failed: unknown outputs=%d\n", totalUnknown)
		failed = true
	}
	if avgConfidence.min < 0.55 || lowConfPct.max > 5 {
		fmt.Fprintln(os.Stderr, "mcprofile audit failed: profile confidence degraded")
		failed = true
	}
	if rollingPct.max > 85 || hillPct.min < 5 || mountainPct.min < 5 {
		fmt.Fprintln(os.Stderr, "mcprofile audit failed: terrain variants collapsed")
		failed = true
	}
	if forestVegPct.min < 5 || openVegPct.min < 10 {
		fmt.Fprintln(os.Stderr, "mcprofile audit failed: vegetation diversity collapsed")
		failed = true
	}
	if desertVegPct.max > 55 || coldVegPct.max > 45 {
		fmt.Fprintln(os.Stderr, "mcprofile audit failed: one sparse vegetation family dominates")
		failed = true
	}
	if forestSurfacePct.min < 5 || grassSurfacePct.min < 10 {
		fmt.Fprintln(os.Stderr, "mcprofile audit failed: surface diversity collapsed")
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
