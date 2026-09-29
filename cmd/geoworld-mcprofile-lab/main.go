package main

import (
	"flag"
	"fmt"
	"log"
	"sort"

	"geoworld/internal/climate"
	"geoworld/internal/ecology"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
)

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid width")
	height := flag.Int("height", 400, "macro grid height")
	out := flag.String("out", "prototype/generated/mcprofile", "output directory")
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
	m, err := mcprofile.Generate(mcprofile.InputFromModels(eco, l, g))
	if err != nil {
		log.Fatal(err)
	}
	if err := mcprofile.RenderAll(m, *out); err != nil {
		log.Fatal(err)
	}

	landN := 0
	unknownN := 0
	lowConfidenceN := 0
	confSum := 0.0

	terrainCounts := map[mcprofile.TerrainVariant]int{}
	surfaceCounts := map[mcprofile.SurfaceProfile]int{}
	vegetationCounts := map[mcprofile.VegetationProfile]int{}

	for i := range m.BiomeFamily {
		if m.LandMask[i] {
			landN++
			confSum += m.ProfileConfidence[i]
			if m.ProfileConfidence[i] < 0.30 {
				lowConfidenceN++
			}
		}
		if m.BiomeFamily[i] == mcprofile.BiomeUnknown ||
			m.TerrainVariant[i] == mcprofile.TerrainUnknown ||
			m.SurfaceProfile[i] == mcprofile.SurfaceUnknown ||
			m.VegetationProfile[i] == mcprofile.VegetationUnknown {
			unknownN++
		}
		terrainCounts[m.TerrainVariant[i]]++
		surfaceCounts[m.SurfaceProfile[i]]++
		if m.LandMask[i] {
			vegetationCounts[m.VegetationProfile[i]]++
		}
	}

	fmt.Printf("GeoWorld Phase 5.2 Minecraft Profile Lab\n")
	fmt.Printf("seed=%d grid=%dx%d land=%d unknown=%d\n", *seed, *width, *height, landN, unknownN)
	fmt.Printf("land avg profile confidence=%.3f low-confidence<0.30=%.2f%%\n",
		safeAvg(confSum, landN), percent(lowConfidenceN, landN))

	printTerrain(terrainCounts, landN)
	printSurface(surfaceCounts, landN)
	printVegetation(vegetationCounts, landN)
	fmt.Printf("output=%s\n", *out)
}

func printTerrain(counts map[mcprofile.TerrainVariant]int, landN int) {
	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	fmt.Println("\nTerrain variants:")
	for _, k := range keys {
		v := mcprofile.TerrainVariant(k)
		if v == mcprofile.TerrainOcean {
			continue
		}
		fmt.Printf("  %-18s %6.2f%%\n", v.String(), percent(counts[v], landN))
	}
}

func printSurface(counts map[mcprofile.SurfaceProfile]int, landN int) {
	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	fmt.Println("\nLand surface profiles:")
	for _, k := range keys {
		v := mcprofile.SurfaceProfile(k)
		if v == mcprofile.SurfaceOceanSand ||
			v == mcprofile.SurfaceOceanGravel ||
			v == mcprofile.SurfaceFrozenOcean {
			continue
		}
		fmt.Printf("  %-18s %6.2f%%\n", v.String(), percent(counts[v], landN))
	}
}

func printVegetation(counts map[mcprofile.VegetationProfile]int, landN int) {
	keys := make([]int, 0, len(counts))
	for k := range counts {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)
	fmt.Println("\nLand vegetation profiles:")
	for _, k := range keys {
		v := mcprofile.VegetationProfile(k)
		fmt.Printf("  %-18s %6.2f%%\n", v.String(), percent(counts[v], landN))
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
