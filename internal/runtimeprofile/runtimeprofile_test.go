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

func TestBiomeBoundaryCompetitionLetsBothSidesInvade(t *testing.T) {
	m := &Model{
		Profile: &mcprofile.Model{
			Width: 2, Height: 1,
			LandMask: []bool{true, true},
		},
	}
	candidates := []candidate{
		{node: 0, weight: 0.51},
		{node: 1, weight: 0.49},
	}

	seen := map[int]bool{}
	for x := 0.0; x <= 1024; x += 8 {
		node, _, ok := m.selectErodedCandidate(
			candidates,
			true,
			731894205,
			x,
			64,
			128,
		)
		if !ok {
			t.Fatal("expected competing land candidates to be selectable")
		}
		seen[node] = true
	}
	if !seen[0] || !seen[1] {
		t.Fatalf("expected mutual biome invasion near boundary, winners=%v", seen)
	}
}

func TestElevationZoningThresholdsFollowClimateAndTerrain(t *testing.T) {
	if !(terrainElevationContext(mcprofile.TerrainMountain) >
		terrainElevationContext(mcprofile.TerrainFoothills) &&
		terrainElevationContext(mcprofile.TerrainFoothills) >
			terrainElevationContext(mcprofile.TerrainHills)) {
		t.Fatal("terrain elevation context should decrease from mountain to foothills to hills")
	}

	if !(biomeSnowLine(mcprofile.BiomeTropicalRainforest) >
		biomeSnowLine(mcprofile.BiomeTemperateForest) &&
		biomeSnowLine(mcprofile.BiomeTemperateForest) >
			biomeSnowLine(mcprofile.BiomeBorealForest)) {
		t.Fatal("warmer climate families should require higher elevation before alpine snow")
	}
}

func TestExposedAlpineRockTracksSlopeRatherThanAltitudeBand(t *testing.T) {
	for _, patch := range []float64{0, 0.5, 1} {
		if exposedAlpineRock(0, patch) || exposedAlpineRock(0.25, patch) {
			t.Fatalf("flat or gently sloped snowfields must not become bare rock, patch=%v", patch)
		}
		if !exposedAlpineRock(1.1, patch) {
			t.Fatalf("steep alpine face must show rock, patch=%v", patch)
		}
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
