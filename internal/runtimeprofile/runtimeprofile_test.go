package runtimeprofile

import (
	"testing"

	"geoworld/internal/mcprofile"
	"geoworld/internal/reconcile"
	"geoworld/internal/world"
)

func TestSelectSourceNodeRespectsRequestedSurfaceSide(t *testing.T) {
	g := world.GridGeometry{Width: 2, Height: 2, SpanXBlocks: 128, SpanZBlocks: 128}
	m := &Model{
		Geometry: g,
		Profile: &mcprofile.Model{
			Width: 2, Height: 2,
			LandMask: []bool{false, true, false, true},
		},
	}

	node, _, fallback := m.selectSourceNode(24, 64, false)
	if fallback || m.Profile.LandMask[node] {
		t.Fatalf("ocean-side selection node=%d fallback=%v land=%v", node, fallback, m.Profile.LandMask[node])
	}

	node, _, fallback = m.selectSourceNode(104, 64, true)
	if fallback || !m.Profile.LandMask[node] {
		t.Fatalf("land-side selection node=%d fallback=%v land=%v", node, fallback, m.Profile.LandMask[node])
	}
}

func TestSelectSourceNodeExposesMissingSemanticSide(t *testing.T) {
	g := world.GridGeometry{Width: 2, Height: 2, SpanXBlocks: 128, SpanZBlocks: 128}
	m := &Model{
		Geometry: g,
		Profile: &mcprofile.Model{
			Width: 2, Height: 2,
			LandMask: []bool{true, true, true, true},
		},
	}
	_, _, fallback := m.selectSourceNode(64, 64, false)
	if !fallback {
		t.Fatal("expected missing ocean-side corner to be exposed as fallback")
	}
}

func TestRuntimeSideMatchesProfile(t *testing.T) {
	ocean := Sample{
		SourceLand:        false,
		BiomeFamily:       mcprofile.BiomeTemperateOcean,
		TerrainVariant:    mcprofile.TerrainOcean,
		VegetationProfile: mcprofile.VegetationNone,
	}
	if !runtimeSideMatchesProfile(reconcile.WaterOcean, ocean) {
		t.Fatal("valid ocean profile rejected")
	}

	land := Sample{
		SourceLand:        true,
		BiomeFamily:       mcprofile.BiomeTemperateForest,
		TerrainVariant:    mcprofile.TerrainRollingPlain,
		VegetationProfile: mcprofile.VegetationMixedForest,
	}
	if !runtimeSideMatchesProfile(reconcile.WaterRiver, land) ||
		!runtimeSideMatchesProfile(reconcile.WaterLake, land) ||
		!runtimeSideMatchesProfile(reconcile.WaterNone, land) {
		t.Fatal("valid terrestrial profile rejected for non-ocean runtime semantics")
	}
	if runtimeSideMatchesProfile(reconcile.WaterOcean, land) {
		t.Fatal("terrestrial profile accepted for ocean runtime semantics")
	}
}
