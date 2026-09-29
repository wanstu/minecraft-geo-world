// geoworld-consistency-audit checks cross-stage contracts. It reports unresolved
// final-terrain differences separately from the enforced macro surface contract.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"geoworld/internal/climate"
	"geoworld/internal/ecology"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
	"geoworld/internal/terrain"
	"geoworld/internal/world"
)

type report struct {
	Seed                                                                     int64
	Width, Height                                                            int
	CellSizeBlocks                                                           float64
	SeaLevelNormalized                                                       float64
	CrustOceanPercent, ActualMacroOceanPercent                               float64
	EmergedOceanicCrust, EnclosedBelowSea                                    int
	MacroOceanAtOrAboveSea, MacroMaskDisagreements                           int
	FinalOceanLabelAboveSea, FinalLandLabelBelowSea, FinalSurfaceMaskChanges int
	NonLakeStrongRiverEdges, OriginallyDownhillEdges, FinalUpslopeEdges      int
	MaxNormalizedRise                                                        float64
}

func checked[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func inspect(seed int64, width, height int, cellSize float64) report {
	w := phase1.Generate(phase1.Config{Seed: seed, Width: width, Height: height, PlateCount: 18, OceanFraction: .68})
	c := checked(climate.Generate(climate.InputFromPhase1(w), climate.Config{}))
	h := checked(hydrology.Generate(hydrology.InputFromWorld(w, c), hydrology.Config{}))
	e := checked(erosion.Generate(erosion.InputFromWorld(w, c, h), erosion.Config{}))
	l := checked(landform.Generate(landform.InputFromWorld(w, e), landform.Config{}))
	g := checked(geomorph.Generate(geomorph.InputFromModels(h, e, l), geomorph.Config{}))
	eco := checked(ecology.Generate(ecology.InputFromModels(c, h, e, l, g), ecology.Config{}))
	p := checked(mcprofile.Generate(mcprofile.InputFromModels(eco, l, g)))
	t := checked(terrain.Generate(terrain.InputFromModels(e, l, g), terrain.Config{Seed: seed, CellSizeBlocks: cellSize}))
	r := report{Seed: seed, Width: width, Height: height, CellSizeBlocks: cellSize, SeaLevelNormalized: w.Surface.SeaLevelNormalized}
	final := make([]float64, len(w.Elevation))
	crustOcean, actualOcean := 0, 0
	masks := [][]bool{c.LandMask, h.LandMask, e.LandMask, l.LandMask, g.LandMask, eco.LandMask, p.LandMask}
	for i, land := range w.Surface.LandMask {
		if w.Crust[i] == phase1.Oceanic {
			crustOcean++
			if land && w.Elevation[i] >= r.SeaLevelNormalized {
				r.EmergedOceanicCrust++
			}
		}
		if w.Surface.EnclosedBelowSea[i] {
			r.EnclosedBelowSea++
		}
		if !land {
			actualOcean++
			if w.Elevation[i] >= r.SeaLevelNormalized {
				r.MacroOceanAtOrAboveSea++
			}
		}
		for _, mask := range masks {
			if mask[i] != land {
				r.MacroMaskDisagreements++
				break
			}
		}
		final[i] = t.SampleNormalized(float64(i%width)*cellSize, float64(i/width)*cellSize)
		if !land && final[i] >= r.SeaLevelNormalized {
			r.FinalOceanLabelAboveSea++
		}
		if land && final[i] < r.SeaLevelNormalized {
			r.FinalLandLabelBelowSea++
		}
	}
	r.CrustOceanPercent = 100 * float64(crustOcean) / float64(len(final))
	r.ActualMacroOceanPercent = 100 * float64(actualOcean) / float64(len(final))
	finalSurface := checked(world.ClassifySurface(width, height, final, r.SeaLevelNormalized))
	for i, land := range finalSurface.LandMask {
		if land != w.Surface.LandMask[i] {
			r.FinalSurfaceMaskChanges++
		}
	}
	for i, to := range h.FlowTo {
		if to < 0 || !h.LandMask[i] || !h.LandMask[to] || h.LakeID[i] != 0 || h.LakeID[to] != 0 || h.RiverStrength[i] < .30 {
			continue
		}
		r.NonLakeStrongRiverEdges++
		if w.Elevation[i] <= w.Elevation[to]+1e-6 {
			continue
		}
		r.OriginallyDownhillEdges++
		if rise := final[to] - final[i]; rise > 1e-5 {
			r.FinalUpslopeEdges++
			r.MaxNormalizedRise = math.Max(r.MaxNormalizedRise, rise)
		}
	}
	return r
}

func main() {
	seedsArg := flag.String("seeds", "731894205,10400", "comma-separated signed 64-bit seeds")
	width := flag.Int("width", 640, "macro grid nodes X (at least 16)")
	height := flag.Int("height", 400, "macro grid nodes Z (at least 16)")
	cellSize := flag.Float64("cell-size", 128, "blocks per node interval")
	strict := flag.Bool("strict-final", false, "also fail on unresolved final-surface or river-direction differences")
	flag.Parse()
	geometry := world.GridGeometry{Width: *width, Height: *height, SpanXBlocks: float64(*width-1) * *cellSize, SpanZBlocks: float64(*height-1) * *cellSize}
	if err := geometry.Validate(); err != nil || *width < 16 || *height < 16 {
		fmt.Fprintln(os.Stderr, "invalid geometry: require finite positive spacing and at least 16x16 nodes")
		os.Exit(2)
	}
	var seeds []int64
	for _, s := range strings.Split(*seedsArg, ",") {
		seed, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid seed:", s)
			os.Exit(2)
		}
		seeds = append(seeds, seed)
	}
	failed, unresolved := false, false
	for _, seed := range seeds {
		r := inspect(seed, *width, *height, *cellSize)
		if err := json.NewEncoder(os.Stdout).Encode(r); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if r.MacroOceanAtOrAboveSea != 0 || r.MacroMaskDisagreements != 0 {
			failed = true
		}
		if r.FinalSurfaceMaskChanges != 0 || r.FinalUpslopeEdges != 0 {
			unresolved = true
		}
	}
	if unresolved {
		fmt.Fprintln(os.Stderr, "UNRESOLVED: final continuous coast/water and river-bed reconciliation belongs to Phase 6.2; macro checks do not certify a playable world.")
	}
	if failed || (*strict && unresolved) {
		os.Exit(1)
	}
}
