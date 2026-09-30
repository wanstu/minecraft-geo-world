// geoworld-consistency-audit checks cross-stage contracts. It reports unresolved
// final-terrain differences separately from the enforced macro surface contract.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"geoworld/internal/generator"
	"geoworld/internal/phase1"
	"geoworld/internal/terrain"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

type report struct {
	Seed                                                                           int64
	Width, Height                                                                  int
	CellSizeBlocks                                                                 float64
	SeaLevelNormalized                                                             float64
	CrustOceanPercent, ActualMacroOceanPercent                                     float64
	EmergedOceanicCrust, EnclosedBelowSea                                          int
	MacroOceanAtOrAboveSea, MacroMaskDisagreements                                 int
	FinalOceanLabelAboveSea, FinalLandLabelBelowSea, FinalSurfaceMaskChanges       int
	NonLakeStrongRiverEdges, OriginallyDownhillEdges, FinalUpslopeEdges            int
	MaxNormalizedRise                                                              float64
	ConstraintSurfaceMismatches                                                    int
	ConstraintCoastAdjustedNodes, ConstraintCoastRaisedNodes                       int
	ConstraintCoastCarvedNodes, ConstraintLakeBedViolations                        int
	ConstraintRiverUpslopeEdges, ConstraintRiverBedCarvedNodes                     int
	AppliedCoastSurfaceMismatches, CoastTerrainUpslopeEdges                        int
	AppliedCoastMaxNodeError, CoastTerrainMaxNormalizedRise                        float64
	CoastTerrainChunkJump, CoastTerrainMacroSlopeMismatch                          float64
	CoastInteriorOceanAboveSea, CoastInteriorLandBelowSea                          int
	CoastInteriorMaxOceanExcess, CoastInteriorMaxLandDeficit                       float64
	WaterbedRiverSampleSteps, WaterbedRiverUpslopeSteps, WaterbedLakeBedViolations int
	WaterbedMaxRiverRise, WaterbedMaxLakeBedExcess, WaterbedChunkJump              float64
	WaterbedMaxRiseSegment, WaterbedMaxRiseFrom, WaterbedMaxRiseTo                 int
	WaterbedMaxRiseT, WaterbedMaxRiseOwnProfileRise                                float64
	WaterbedMaxRisePreviousOverlaps, WaterbedMaxRiseCurrentOverlaps                int
}

func checked[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

func inspect(seed int64, width, height int, cellSize float64) report {
	geometry := world.GridGeometry{
		Width: width, Height: height,
		SpanXBlocks: float64(width-1) * cellSize,
		SpanZBlocks: float64(height-1) * cellSize,
	}
	model := checked(generator.GenerateWorld(context.Background(), generator.Config{
		Seed: seed, Geometry: geometry,
		PlateCount: 18, OceanFraction: .68,
	}, nil))
	w := model.Phase1
	c := model.Climate
	h := model.Hydrology
	e := model.Erosion
	l := model.Landform
	g := model.Geomorph
	eco := model.Ecology
	p := model.Profile
	t := model.Terrain
	r := report{Seed: seed, Width: width, Height: height, CellSizeBlocks: cellSize, SeaLevelNormalized: model.Config.SeaLevelNormalized}
	if rec := model.Reconciliation; rec != nil {
		r.ConstraintSurfaceMismatches = rec.SurfaceMismatchAfter
		r.ConstraintCoastAdjustedNodes = rec.CoastAdjustedNodes
		r.ConstraintCoastRaisedNodes = rec.CoastRaisedNodes
		r.ConstraintCoastCarvedNodes = rec.CoastCarvedNodes
		r.ConstraintLakeBedViolations = rec.LakeBedViolations
		r.ConstraintRiverUpslopeEdges = rec.TargetUpslopeRiverEdges
		r.ConstraintRiverBedCarvedNodes = rec.RiverBedCarvedNodes
		r.AppliedCoastSurfaceMismatches = rec.AppliedCoastSurfaceMismatch
		r.AppliedCoastMaxNodeError = rec.AppliedCoastMaxNodeError
		r.CoastTerrainUpslopeEdges = rec.RawUpslopeRiverEdges
		for _, edge := range rec.RiverEdges {
			if edge.RawTerrainRise > r.CoastTerrainMaxNormalizedRise {
				r.CoastTerrainMaxNormalizedRise = edge.RawTerrainRise
			}
		}
	}
	if model.CoastTerrain != nil {
		r.CoastTerrainChunkJump = auditChunkSeams(model.CoastTerrain, model.Geometry)
		r.CoastTerrainMacroSlopeMismatch = auditMacroSlopes(model.CoastTerrain, model.Geometry)
		coast := auditCoastInteriors(model.CoastTerrain, w.Surface, model.Geometry, r.SeaLevelNormalized)
		r.CoastInteriorOceanAboveSea = coast.oceanAboveSea
		r.CoastInteriorLandBelowSea = coast.landBelowSea
		r.CoastInteriorMaxOceanExcess = coast.maxOceanExcess
		r.CoastInteriorMaxLandDeficit = coast.maxLandDeficit
	}
	if model.Waterbed != nil {
		d := model.Waterbed.Audit(16, model.Reconciliation.Config.FinalUpslopeEpsilon)
		r.WaterbedRiverSampleSteps = d.RiverSampleSteps
		r.WaterbedRiverUpslopeSteps = d.RiverUpslopeSteps
		r.WaterbedMaxRiverRise = d.MaxRiverRise
		r.WaterbedMaxRiseSegment = d.MaxRiseSegment
		r.WaterbedMaxRiseFrom = d.MaxRiseFrom
		r.WaterbedMaxRiseTo = d.MaxRiseTo
		r.WaterbedMaxRiseT = d.MaxRiseT
		r.WaterbedMaxRiseOwnProfileRise = d.MaxRiseOwnProfileRise
		r.WaterbedMaxRisePreviousOverlaps = d.MaxRisePreviousOverlaps
		r.WaterbedMaxRiseCurrentOverlaps = d.MaxRiseCurrentOverlaps
		r.WaterbedLakeBedViolations = d.LakeBedViolations
		r.WaterbedMaxLakeBedExcess = d.MaxLakeBedExcess
		r.WaterbedChunkJump = auditWaterbedChunkSeams(model.Waterbed, model.Geometry)
	}
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
		x, z := w.Geometry.NodePosition(i%width, i/width)
		final[i] = t.SampleNormalized(x, z)
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

func auditChunkSeams(m *terrain.Model, geometry world.GridGeometry) float64 {
	maxJump := 0.0
	eps := 1e-4
	endX := geometry.OriginX + geometry.SpanXBlocks
	endZ := geometry.OriginZ + geometry.SpanZBlocks

	firstBoundary := math.Floor(geometry.OriginX/16)*16 + 16
	for firstBoundary <= geometry.OriginX {
		firstBoundary += 16
	}
	stepZ := math.Max(256, geometry.SpanZBlocks/40)
	for x := firstBoundary; x < endX; x += 512 {
		for z := geometry.OriginZ; z <= endZ; z += stepZ {
			at := m.SampleNormalized(x, z)
			maxJump = math.Max(maxJump, math.Abs(m.SampleNormalized(x-eps, z)-at))
			maxJump = math.Max(maxJump, math.Abs(m.SampleNormalized(x+eps, z)-at))
		}
	}
	return maxJump
}

func auditWaterbedChunkSeams(m *waterbed.Model, geometry world.GridGeometry) float64 {
	maxJump := 0.0
	eps := 1e-4
	endX := geometry.OriginX + geometry.SpanXBlocks
	endZ := geometry.OriginZ + geometry.SpanZBlocks

	firstBoundary := math.Floor(geometry.OriginX/16)*16 + 16
	for firstBoundary <= geometry.OriginX {
		firstBoundary += 16
	}
	stepZ := math.Max(256, geometry.SpanZBlocks/40)
	for x := firstBoundary; x < endX; x += 512 {
		for z := geometry.OriginZ; z <= endZ; z += stepZ {
			at := m.SampleNormalized(x, z)
			maxJump = math.Max(maxJump, math.Abs(m.SampleNormalized(x-eps, z)-at))
			maxJump = math.Max(maxJump, math.Abs(m.SampleNormalized(x+eps, z)-at))
		}
	}
	return maxJump
}

func auditMacroSlopes(m *terrain.Model, geometry world.GridGeometry) float64 {
	maxMismatch := 0.0
	cellSize := geometry.SpacingX()
	h := 0.25
	endX := geometry.OriginX + geometry.SpanXBlocks
	endZ := geometry.OriginZ + geometry.SpanZBlocks

	for x := geometry.OriginX + cellSize; x < endX; x += cellSize * 8 {
		for z := geometry.OriginZ + cellSize; z < endZ-cellSize; z += cellSize * 4 {
			at := m.SampleNormalized(x, z)
			leftSlope := (at - m.SampleNormalized(x-h, z)) / h
			rightSlope := (m.SampleNormalized(x+h, z) - at) / h
			maxMismatch = math.Max(maxMismatch, math.Abs(leftSlope-rightSlope))
		}
	}
	return maxMismatch
}

type coastInteriorDiagnostics struct {
	oceanAboveSea  int
	landBelowSea   int
	maxOceanExcess float64
	maxLandDeficit float64
}

func auditCoastInteriors(m *terrain.Model, surface *world.Surface, geometry world.GridGeometry, sea float64) coastInteriorDiagnostics {
	var d coastInteriorDiagnostics
	offsets := [][2]float64{
		{0.25, 0.25},
		{0.75, 0.25},
		{0.50, 0.50},
		{0.25, 0.75},
		{0.75, 0.75},
	}
	spacing := geometry.SpacingX()

	for z := 0; z < geometry.Height-1; z++ {
		for x := 0; x < geometry.Width-1; x++ {
			indices := [4]int{
				z*geometry.Width + x,
				z*geometry.Width + x + 1,
				(z+1)*geometry.Width + x,
				(z+1)*geometry.Width + x + 1,
			}

			allOcean := true
			allNormalLand := true
			for _, i := range indices {
				if surface.LandMask[i] {
					allOcean = false
				} else {
					allNormalLand = false
				}
				if !surface.LandMask[i] || surface.EnclosedBelowSea[i] {
					allNormalLand = false
				}
			}
			if !allOcean && !allNormalLand {
				continue
			}

			for _, off := range offsets {
				wx := geometry.OriginX + (float64(x)+off[0])*spacing
				wz := geometry.OriginZ + (float64(z)+off[1])*spacing
				h := m.SampleNormalized(wx, wz)
				if allOcean && h >= sea {
					d.oceanAboveSea++
					d.maxOceanExcess = math.Max(d.maxOceanExcess, h-sea)
				}
				if allNormalLand && h < sea {
					d.landBelowSea++
					d.maxLandDeficit = math.Max(d.maxLandDeficit, sea-h)
				}
			}
		}
	}
	return d
}

func main() {
	seedsArg := flag.String("seeds", "731894205,10400", "comma-separated signed 64-bit seeds")
	width := flag.Int("width", 640, "macro grid nodes X (at least 16)")
	height := flag.Int("height", 400, "macro grid nodes Z (at least 16)")
	cellSize := flag.Float64("cell-size", 128, "blocks per node interval")
	strict := flag.Bool("strict-final", false, "fail until Runtime-facing river water-surface/depth semantics are implemented")
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
	failed, rawBaselineDifferences := false, false
	for _, seed := range seeds {
		r := inspect(seed, *width, *height, *cellSize)
		if err := json.NewEncoder(os.Stdout).Encode(r); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if r.MacroOceanAtOrAboveSea != 0 ||
			r.MacroMaskDisagreements != 0 ||
			r.ConstraintSurfaceMismatches != 0 ||
			r.ConstraintRiverUpslopeEdges != 0 ||
			r.AppliedCoastSurfaceMismatches != 0 ||
			r.AppliedCoastMaxNodeError > 1e-10 ||
			r.CoastTerrainChunkJump > 1e-5 ||
			r.CoastTerrainMacroSlopeMismatch > 1e-4 ||
			r.CoastInteriorOceanAboveSea != 0 ||
			r.CoastInteriorLandBelowSea != 0 ||
			r.WaterbedRiverUpslopeSteps != 0 ||
			r.WaterbedLakeBedViolations != 0 ||
			r.WaterbedChunkJump > 1e-5 {
			failed = true
		}
		if r.FinalSurfaceMaskChanges != 0 || r.FinalUpslopeEdges != 0 {
			rawBaselineDifferences = true
		}
	}
	if rawBaselineDifferences {
		fmt.Fprintln(os.Stderr, "RAW BASELINE: uncorrected Terrain still retains the historical coast/river mismatches; applied CoastTerrain + Waterbed are audited separately above.")
	}
	// Phase 6.2 has now applied broad coast and local bed geometry, but the
	// Runtime-facing water-surface sampler is not complete yet: ocean/lake datum
	// exists, while river water depth/surface still needs an explicit rule.
	runtimeWaterSurfaceIncomplete := true
	if runtimeWaterSurfaceIncomplete {
		fmt.Fprintln(os.Stderr, "UNRESOLVED: Runtime river water-surface/depth semantics are not implemented yet; -strict-final remains intentionally failing.")
	}
	if failed || (*strict && runtimeWaterSurfaceIncomplete) {
		os.Exit(1)
	}
}
