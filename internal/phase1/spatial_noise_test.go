package phase1

import (
	"math"
	"testing"

	"geoworld/internal/world"
)

func TestReferenceWorldNoiseStableAcrossResolution(t *testing.T) {
	coarse := &World{
		Config: Config{Seed: 731894205, Width: 9, Height: 7},
		Geometry: world.GridGeometry{
			Width: 9, Height: 7,
			SpanXBlocks: 1024,
			SpanZBlocks: 768,
		},
	}
	fine := &World{
		Config: Config{Seed: 731894205, Width: 17, Height: 13},
		Geometry: world.GridGeometry{
			Width: 17, Height: 13,
			SpanXBlocks: 1024,
			SpanZBlocks: 768,
		},
	}

	cx, cz := coarse.referenceGridCoordinates(4, 3)
	fx, fz := fine.referenceGridCoordinates(8, 6)
	if math.Abs(cx-fx) > 1e-12 || math.Abs(cz-fz) > 1e-12 {
		t.Fatalf("reference coordinates changed with resolution: coarse=(%.12f,%.12f) fine=(%.12f,%.12f)",
			cx, cz, fx, fz)
	}

	for name, pair := range map[string][2]float64{
		"continent potential": {
			continentalPotentialNoise(cx, cz, coarse.Config.Seed),
			continentalPotentialNoise(fx, fz, fine.Config.Seed),
		},
		"crust thickness": {
			crustThicknessNoise(cx, cz, coarse.Config.Seed),
			crustThicknessNoise(fx, fz, fine.Config.Seed),
		},
		"macro elevation": {
			macroElevationNoise(cx, cz, coarse.Config.Seed),
			macroElevationNoise(fx, fz, fine.Config.Seed),
		},
	} {
		if math.Abs(pair[0]-pair[1]) > 1e-12 {
			t.Fatalf("%s noise changed with resolution: coarse=%.12f fine=%.12f", name, pair[0], pair[1])
		}
	}
}

func TestReferenceWorldNoisePreservesDefault128BlockGrid(t *testing.T) {
	w := &World{
		Config: Config{Seed: 12345, Width: 8, Height: 6},
		Geometry: world.GridGeometry{
			Width: 8, Height: 6,
			SpanXBlocks: 7 * DefaultReferenceCellSizeBlocks,
			SpanZBlocks: 5 * DefaultReferenceCellSizeBlocks,
		},
	}
	x, z := 5, 4
	rx, rz := w.referenceGridCoordinates(x, z)
	if rx != float64(x) || rz != float64(z) {
		t.Fatalf("128-block reference coordinates changed: got=(%.12f,%.12f) want=(%d,%d)", rx, rz, x, z)
	}

	legacyPotential := math.Sin(float64(x)*0.031+float64(w.Config.Seed%401))*0.10 +
		math.Cos(float64(z)*0.037-float64(w.Config.Seed%271))*0.08 +
		math.Sin(float64(x+z)*0.019)*0.05
	if got := continentalPotentialNoise(rx, rz, w.Config.Seed); math.Abs(got-legacyPotential) > 1e-12 {
		t.Fatalf("continent potential changed on default grid: got=%.12f legacy=%.12f", got, legacyPotential)
	}

	legacyThickness := math.Sin(float64(x)*0.021+float64(w.Config.Seed%421))*0.025 +
		math.Cos(float64(z)*0.019-float64(w.Config.Seed%353))*0.02
	if got := crustThicknessNoise(rx, rz, w.Config.Seed); math.Abs(got-legacyThickness) > 1e-12 {
		t.Fatalf("crust thickness noise changed on default grid: got=%.12f legacy=%.12f", got, legacyThickness)
	}

	legacyElevation := math.Sin(float64(x)*0.071+float64(w.Config.Seed%997)) *
		math.Cos(float64(z)*0.083-float64(w.Config.Seed%313))
	if got := macroElevationNoise(rx, rz, w.Config.Seed); math.Abs(got-legacyElevation) > 1e-12 {
		t.Fatalf("macro elevation noise changed on default grid: got=%.12f legacy=%.12f", got, legacyElevation)
	}
}

func TestReferenceWorldPlateAndContinentGeometryStableAcrossResolution(t *testing.T) {
	coarseCfg := Config{
		Seed: 731894205, Width: 41, Height: 25,
		SpanXBlocks: 40 * DefaultReferenceCellSizeBlocks,
		SpanZBlocks: 24 * DefaultReferenceCellSizeBlocks,
		PlateCount:  8, OceanFraction: 0.68,
	}
	fineCfg := Config{
		Seed: 731894205, Width: 81, Height: 49,
		SpanXBlocks: coarseCfg.SpanXBlocks,
		SpanZBlocks: coarseCfg.SpanZBlocks,
		PlateCount:  8, OceanFraction: 0.68,
	}

	coarse := Generate(coarseCfg)
	fine := Generate(fineCfg)

	if len(coarse.Plates) != len(fine.Plates) {
		t.Fatalf("plate count changed with resolution: coarse=%d fine=%d", len(coarse.Plates), len(fine.Plates))
	}
	for i := range coarse.Plates {
		if math.Abs(coarse.Plates[i].Center.X-fine.Plates[i].Center.X) > 1e-12 ||
			math.Abs(coarse.Plates[i].Center.Y-fine.Plates[i].Center.Y) > 1e-12 {
			t.Fatalf("plate %d center changed with resolution: coarse=%+v fine=%+v",
				i, coarse.Plates[i].Center, fine.Plates[i].Center)
		}
	}

	for z := 0; z < coarse.Config.Height; z++ {
		for x := 0; x < coarse.Config.Width; x++ {
			ci := coarse.index(x, z)
			fi := fine.index(x*2, z*2)
			if coarse.RawPlateID[ci] != fine.RawPlateID[fi] {
				t.Fatalf("raw plate changed at physical node %d,%d: coarse=%d fine=%d",
					x, z, coarse.RawPlateID[ci], fine.RawPlateID[fi])
			}
			if coarse.PlateID[ci] != fine.PlateID[fi] {
				t.Fatalf("warped plate changed at physical node %d,%d: coarse=%d fine=%d",
					x, z, coarse.PlateID[ci], fine.PlateID[fi])
			}
			if math.Abs(coarse.LandPotential[ci]-fine.LandPotential[fi]) > 1e-12 {
				t.Fatalf("land potential changed at physical node %d,%d: coarse=%.12f fine=%.12f",
					x, z, coarse.LandPotential[ci], fine.LandPotential[fi])
			}
		}
	}

	coarseBufferBlocks := float64(edgeOceanBufferCells(coarse.Config)) * coarse.Geometry.SpacingX()
	fineBufferBlocks := float64(edgeOceanBufferCells(fine.Config)) * fine.Geometry.SpacingX()
	if math.Abs(coarseBufferBlocks-fineBufferBlocks) > 1e-12 {
		t.Fatalf("edge ocean buffer changed physical width: coarse=%.1f fine=%.1f blocks",
			coarseBufferBlocks, fineBufferBlocks)
	}
}

func TestWarpedPlatePointStableAcrossResolution(t *testing.T) {
	cfg := Config{Seed: 2026, Width: 41, Height: 25}
	coarseGeometry := world.GridGeometry{
		Width: 41, Height: 25,
		SpanXBlocks: 40 * DefaultReferenceCellSizeBlocks,
		SpanZBlocks: 24 * DefaultReferenceCellSizeBlocks,
	}
	fineGeometry := world.GridGeometry{
		Width: 81, Height: 49,
		SpanXBlocks: coarseGeometry.SpanXBlocks,
		SpanZBlocks: coarseGeometry.SpanZBlocks,
	}
	coarseWorld := &World{Geometry: coarseGeometry}
	fineWorld := &World{Geometry: fineGeometry}

	cx, cz := coarseWorld.referenceGridCoordinates(13, 7)
	fx, fz := fineWorld.referenceGridCoordinates(26, 14)
	cwx, cwz := warpedPlatePoint(cx, cz, cfg, coarseGeometry)
	fwx, fwz := warpedPlatePoint(fx, fz, cfg, fineGeometry)
	if math.Abs(cwx-fwx) > 1e-12 || math.Abs(cwz-fwz) > 1e-12 {
		t.Fatalf("plate warp changed with resolution: coarse=(%.12f,%.12f) fine=(%.12f,%.12f)",
			cwx, cwz, fwx, fwz)
	}
}
