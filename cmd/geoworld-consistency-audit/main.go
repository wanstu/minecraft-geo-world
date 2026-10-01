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
	"geoworld/internal/reconcile"
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
	RuntimeOceanNodeViolations, RuntimeEnclosedOceanLeaks                          int
	RuntimeLakeNodeViolations, RuntimeRiverUndefinedSteps                          int
	RuntimeRiverDrySteps, RuntimeRiverSurfaceUpslopeSteps                          int
	RuntimeMaxRiverSurfaceRise                                                     float64
	RuntimeMaxRisePreviousKind, RuntimeMaxRiseCurrentKind                          int
	RuntimeMaxRiseSegment                                                          int
	RuntimeMaxRiseT, RuntimeMaxRisePreviousSurface, RuntimeMaxRiseCurrentSurface   float64
	RuntimeMaxRiseReceiverDefined                                                  bool
	RuntimeMaxRiseReceiverSurface, RuntimeMaxRiseRiverWeight                       float64
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
	if model.Runtime != nil {
		d := auditRuntimeWater(model)
		r.RuntimeOceanNodeViolations = d.oceanNodeViolations
		r.RuntimeEnclosedOceanLeaks = d.enclosedOceanLeaks
		r.RuntimeLakeNodeViolations = d.lakeNodeViolations
		r.RuntimeRiverUndefinedSteps = d.riverUndefinedSteps
		r.RuntimeRiverDrySteps = d.riverDrySteps
		r.RuntimeRiverSurfaceUpslopeSteps = d.riverSurfaceUpslopeSteps
		r.RuntimeMaxRiverSurfaceRise = d.maxRiverSurfaceRise
		r.RuntimeMaxRisePreviousKind = int(d.maxRisePreviousKind)
		r.RuntimeMaxRiseCurrentKind = int(d.maxRiseCurrentKind)
		r.RuntimeMaxRiseSegment = d.maxRiseSegment
		r.RuntimeMaxRiseT = d.maxRiseT
		r.RuntimeMaxRisePreviousSurface = d.maxRisePreviousSurface
		r.RuntimeMaxRiseCurrentSurface = d.maxRiseCurrentSurface
		r.RuntimeMaxRiseReceiverDefined = d.maxRiseReceiverDefined
		r.RuntimeMaxRiseReceiverSurface = d.maxRiseReceiverSurface
		r.RuntimeMaxRiseRiverWeight = d.maxRiseRiverWeight
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

type runtimeWaterDiagnostics struct {
	oceanNodeViolations      int
	enclosedOceanLeaks       int
	lakeNodeViolations       int
	riverUndefinedSteps      int
	riverDrySteps            int
	riverSurfaceUpslopeSteps int
	maxRiverSurfaceRise      float64
	maxRisePreviousKind      reconcile.WaterKind
	maxRiseCurrentKind       reconcile.WaterKind
	maxRiseSegment           int
	maxRiseT                 float64
	maxRisePreviousSurface   float64
	maxRiseCurrentSurface    float64
	maxRiseReceiverDefined   bool
	maxRiseReceiverSurface   float64
	maxRiseRiverWeight       float64
}

func auditRuntimeWater(model *generator.WorldModel) runtimeWaterDiagnostics {
	var d runtimeWaterDiagnostics
	if model == nil || model.Runtime == nil || model.Surface == nil ||
		model.Hydrology == nil || model.Reconciliation == nil || model.Waterbed == nil {
		return d
	}

	geometry := model.Geometry
	sea := model.Config.SeaLevelNormalized
	for i, land := range model.Surface.LandMask {
		x, z := geometry.NodePosition(i%geometry.Width, i/geometry.Width)
		s := model.Runtime.Sample(x, z)

		if !land {
			if s.WaterKind != reconcile.WaterOcean ||
				!s.WaterSurfaceDefined ||
				math.Abs(s.WaterSurfaceElevation-sea) > 1e-12 ||
				s.WaterDepth <= 0 {
				d.oceanNodeViolations++
			}
		}
		if model.Surface.EnclosedBelowSea[i] && s.WaterKind == reconcile.WaterOcean {
			d.enclosedOceanLeaks++
		}
		if model.Hydrology.LakeID[i] > 0 {
			if s.WaterKind != reconcile.WaterLake ||
				!s.WaterSurfaceDefined ||
				s.WaterDepth <= 0 {
				d.lakeNodeViolations++
			}
		}
	}

	epsilon := model.Reconciliation.Config.FinalUpslopeEpsilon
	for segID, seg := range model.Waterbed.Rivers {
		steps := 16
		if n := len(seg.Profile) - 1; n > steps {
			steps = n
		}
		var prevSurface float64
		var prevKind reconcile.WaterKind
		var prevReceiverDefined bool
		var prevReceiverSurface float64
		var prevRiverWeight float64
		havePrev := false
		for j := 0; j <= steps; j++ {
			t := float64(j) / float64(steps)
			x := seg.X0 + (seg.X1-seg.X0)*t
			z := seg.Z0 + (seg.Z1-seg.Z0)*t
			ground := model.Waterbed.Sample(x, z)
			s := model.Runtime.Sample(x, z)

			if !s.WaterSurfaceDefined || s.WaterKind == reconcile.WaterNone {
				d.riverUndefinedSteps++
				havePrev = false
				continue
			}
			if s.WaterDepth <= 1e-12 {
				d.riverDrySteps++
			}
			if havePrev {
				rise := s.WaterSurfaceElevation - prevSurface
				if rise > epsilon {
					d.riverSurfaceUpslopeSteps++
					if rise > d.maxRiverSurfaceRise {
						d.maxRiverSurfaceRise = rise
						d.maxRisePreviousKind = prevKind
						d.maxRiseCurrentKind = s.WaterKind
						d.maxRiseSegment = segID
						d.maxRiseT = t
						d.maxRisePreviousSurface = prevSurface
						d.maxRiseCurrentSurface = s.WaterSurfaceElevation
						d.maxRiseReceiverDefined = prevReceiverDefined
						d.maxRiseReceiverSurface = prevReceiverSurface
						d.maxRiseRiverWeight = prevRiverWeight
					}
				}
			}
			prevSurface = s.WaterSurfaceElevation
			prevKind = s.WaterKind
			prevReceiverDefined = ground.RiverReceiverSurfaceDefined
			prevReceiverSurface = ground.RiverReceiverSurface
			prevRiverWeight = ground.RiverWeight
			havePrev = true
		}
	}
	return d
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
	strict := flag.Bool("strict-final", false, "also require all Runtime-facing ocean/lake/river water checks to pass")
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
	failed, runtimeFailed, rawBaselineDifferences := false, false, false
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
		if r.RuntimeOceanNodeViolations != 0 ||
			r.RuntimeEnclosedOceanLeaks != 0 ||
			r.RuntimeLakeNodeViolations != 0 ||
			r.RuntimeRiverUndefinedSteps != 0 ||
			r.RuntimeRiverDrySteps != 0 ||
			r.RuntimeRiverSurfaceUpslopeSteps != 0 {
			runtimeFailed = true
		}
		if r.FinalSurfaceMaskChanges != 0 || r.FinalUpslopeEdges != 0 {
			rawBaselineDifferences = true
		}
	}
	if rawBaselineDifferences {
		fmt.Fprintln(os.Stderr, "RAW BASELINE: uncorrected Terrain retains historical coast/river mismatches by design; applied CoastTerrain + Waterbed + Runtime are the final-world checks.")
	}
	if runtimeFailed {
		fmt.Fprintln(os.Stderr, "FINAL RUNTIME: ocean/lake/river water semantics still contain violations.")
	}
	if failed || (*strict && runtimeFailed) {
		os.Exit(1)
	}
}
