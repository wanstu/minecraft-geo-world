package terrain

import (
	"math"
	"testing"

	"geoworld/internal/world"
)

func terrainInputForGeometry(g world.GridGeometry) Input {
	in := uniformInput(g.Width, g.Height, 0)
	in.Geometry = g
	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			i := z*g.Width + x
			wx, wz := g.NodePosition(x, z)
			nx := (wx - g.OriginX) / g.SpanXBlocks
			nz := (wz - g.OriginZ) / g.SpanZBlocks
			in.BaseElevation[i] = 0.55 + 0.05*nx + 0.03*nz
			in.RollingPlainWeight[i] = 1
		}
	}
	return in
}

func TestTerrainGeometryStableAcrossResolution(t *testing.T) {
	coarseGeometry := world.GridGeometry{
		Width: 17, Height: 17,
		OriginX: -1024, OriginZ: 2048,
		SpanXBlocks: 2048, SpanZBlocks: 2048,
	}
	fineGeometry := world.GridGeometry{
		Width: 33, Height: 33,
		OriginX: coarseGeometry.OriginX, OriginZ: coarseGeometry.OriginZ,
		SpanXBlocks: coarseGeometry.SpanXBlocks, SpanZBlocks: coarseGeometry.SpanZBlocks,
	}

	coarse, err := Generate(terrainInputForGeometry(coarseGeometry), Config{Seed: 2026})
	if err != nil {
		t.Fatal(err)
	}
	fine, err := Generate(terrainInputForGeometry(fineGeometry), Config{Seed: 2026})
	if err != nil {
		t.Fatal(err)
	}

	if coarse.Geometry.SpacingX() != 128 || fine.Geometry.SpacingX() != 64 {
		t.Fatalf("unexpected spacing: coarse=%.1f fine=%.1f", coarse.Geometry.SpacingX(), fine.Geometry.SpacingX())
	}
	if coarse.Config.OriginX != coarseGeometry.OriginX || coarse.Config.OriginZ != coarseGeometry.OriginZ {
		t.Fatalf("resolved config did not inherit geometry origin: %+v", coarse.Config)
	}

	for _, p := range [][2]float64{
		{coarseGeometry.OriginX + 512, coarseGeometry.OriginZ + 640},
		{coarseGeometry.OriginX + 896.25, coarseGeometry.OriginZ + 1117.5},
		{coarseGeometry.OriginX + 1408, coarseGeometry.OriginZ + 1536},
	} {
		a := coarse.Sample(p[0], p[1])
		b := fine.Sample(p[0], p[1])
		if math.Abs(a.BaseElevation-b.BaseElevation) > 1e-10 {
			t.Fatalf("base sampling changed with resolution at %.2f,%.2f: coarse=%.12f fine=%.12f",
				p[0], p[1], a.BaseElevation, b.BaseElevation)
		}
		if math.Abs(a.RegionalDetail-b.RegionalDetail) > 1e-12 {
			t.Fatalf("world-coordinate detail changed with resolution at %.2f,%.2f: coarse=%.12f fine=%.12f",
				p[0], p[1], a.RegionalDetail, b.RegionalDetail)
		}
		if math.Abs(a.FinalHeight-b.FinalHeight) > 1e-10 {
			t.Fatalf("final terrain changed with resolution at %.2f,%.2f: coarse=%.12f fine=%.12f",
				p[0], p[1], a.FinalHeight, b.FinalHeight)
		}
	}
}

func TestTerrainRejectsGeometryDimensionMismatch(t *testing.T) {
	in := uniformInput(5, 5, 0.6)
	in.Geometry = world.GridGeometry{Width: 6, Height: 5, SpanXBlocks: 640, SpanZBlocks: 512}
	if _, err := Generate(in, Config{}); err == nil {
		t.Fatal("expected terrain geometry dimension mismatch to fail")
	}
}

func TestTerrainRejectsLegacyCellSizeConflict(t *testing.T) {
	in := uniformInput(5, 5, 0.6)
	in.Geometry = world.GridGeometry{Width: 5, Height: 5, SpanXBlocks: 256, SpanZBlocks: 256}
	if _, err := Generate(in, Config{CellSizeBlocks: 128}); err == nil {
		t.Fatal("expected legacy cell size conflict to fail")
	}
}

func TestTerrainRejectsLegacyOriginConflict(t *testing.T) {
	in := uniformInput(5, 5, 0.6)
	in.Geometry = world.GridGeometry{
		Width: 5, Height: 5,
		OriginX: 100, OriginZ: 200,
		SpanXBlocks: 512, SpanZBlocks: 512,
	}
	if _, err := Generate(in, Config{OriginX: 50}); err == nil {
		t.Fatal("expected legacy origin conflict to fail")
	}
}

func TestTerrainLegacyInputFallsBackToReferenceGrid(t *testing.T) {
	in := uniformInput(5, 4, 0.6)
	m, err := Generate(in, Config{Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(m.Geometry.SpacingX()-world.DefaultMacroSpacingBlocks) > 1e-12 ||
		math.Abs(m.Geometry.SpacingZ()-world.DefaultMacroSpacingBlocks) > 1e-12 {
		t.Fatalf("legacy terrain input did not use reference spacing: x=%.1f z=%.1f",
			m.Geometry.SpacingX(), m.Geometry.SpacingZ())
	}
	if m.BaseElevation.OriginX != 0 || m.BaseElevation.OriginZ != 0 {
		t.Fatalf("unexpected legacy origin: %.1f,%.1f", m.BaseElevation.OriginX, m.BaseElevation.OriginZ)
	}
}
