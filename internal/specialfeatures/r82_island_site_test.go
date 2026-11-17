package specialfeatures

import (
	"math"
	"testing"

	"geoworld/internal/ecology"
	"geoworld/internal/mcprofile"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/worldsample"
)

func testOceanIslandBase() runtimeprofile.Sample {
	return runtimeprofile.Sample{
		World: worldsample.Sample{
			GroundHeight:          0.38,
			WaterKind:             reconcile.WaterOcean,
			WaterSurfaceDefined:   true,
			WaterSurfaceElevation: 0.50,
			WaterDepth:            0.12,
			OceanEligibility:      1,
		},
		SourceLand:        false,
		EcoRegion:         ecology.RegionTemperateOcean,
		BiomeFamily:       mcprofile.BiomeTemperateOcean,
		TerrainVariant:    mcprofile.TerrainOcean,
		SurfaceProfile:    mcprofile.SurfaceOceanSand,
		VegetationProfile: mcprofile.VegetationNone,
	}
}

func TestR82V125SubmergedIslandQualification(t *testing.T) {
	tests := []struct {
		name    string
		context string
		metrics submergedIslandMetrics
		want    bool
	}{
		{
			"ocean qualified",
			"ocean",
			submergedIslandMetrics{RootLiftBlocks: 8, CenterDepthBlocks: 48, MatchingWaterSamples: 32, Samples: 32},
			true,
		},
		{
			"ocean flat seabed",
			"ocean",
			submergedIslandMetrics{RootLiftBlocks: 1, CenterDepthBlocks: 14, MatchingWaterSamples: 32, Samples: 32},
			false,
		},
		{
			"ocean too deep",
			"ocean",
			submergedIslandMetrics{RootLiftBlocks: 12, CenterDepthBlocks: 60, MatchingWaterSamples: 32, Samples: 32},
			false,
		},
		{
			"ocean broken moat",
			"ocean",
			submergedIslandMetrics{RootLiftBlocks: 12, CenterDepthBlocks: 20, MatchingWaterSamples: 31, Samples: 32},
			false,
		},
		{
			"lake qualified",
			"lake",
			submergedIslandMetrics{RootLiftBlocks: 4, CenterDepthBlocks: 24, MatchingWaterSamples: 32, Samples: 32},
			true,
		},
		{
			"lake flat bed",
			"lake",
			submergedIslandMetrics{RootLiftBlocks: 0, CenterDepthBlocks: 8, MatchingWaterSamples: 32, Samples: 32},
			false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := qualifiesSubmergedIslandRoot(tc.context, tc.metrics); got != tc.want {
				t.Fatalf("qualified=%t want %t metrics=%+v", got, tc.want, tc.metrics)
			}
		})
	}
}

func TestR82V125IslandShoulderKeepsOuterSlopeSubmerged(t *testing.T) {
	feature := Feature{
		ID: 1, Kind: MushroomIsland,
		CenterX: 0, CenterZ: 0,
		RadiusX: 100, RadiusZ: 100,
		WaterContext: "ocean",
	}
	base := testOceanIslandBase()

	// x=90 gives a small but non-zero influence. v1.1.25 must begin by
	// raising the seabed while keeping Ocean water above it.
	got := Apply(
		[]Feature{feature},
		base,
		0.50,
		90, 0,
		"geoworld-rules-v1.1.25",
	)
	if got.World.WaterKind != reconcile.WaterOcean {
		t.Fatalf("outer shoulder became land: water=%v", got.World.WaterKind)
	}
	if got.World.GroundHeight <= base.World.GroundHeight {
		t.Fatalf(
			"outer shoulder did not raise seabed: got=%.6f base=%.6f",
			got.World.GroundHeight,
			base.World.GroundHeight,
		)
	}
	if got.World.GroundHeight >= base.World.WaterSurfaceElevation {
		t.Fatalf(
			"outer shoulder rose through water: ground=%.6f water=%.6f",
			got.World.GroundHeight,
			base.World.WaterSurfaceElevation,
		)
	}
	if math.Abs(
		got.World.WaterDepth-
			(base.World.WaterSurfaceElevation-got.World.GroundHeight),
	) > 1e-9 {
		t.Fatalf(
			"shoulder water depth mismatch: depth=%.6f ground=%.6f water=%.6f",
			got.World.WaterDepth,
			got.World.GroundHeight,
			base.World.WaterSurfaceElevation,
		)
	}

	center := Apply(
		[]Feature{feature},
		base,
		0.50,
		0, 0,
		"geoworld-rules-v1.1.25",
	)
	if center.World.WaterKind != reconcile.WaterNone || !center.SourceLand {
		t.Fatalf("island center did not emerge as land: %+v", center.World)
	}
}

func TestR82V124IslandGeometryRemainsLocked(t *testing.T) {
	feature := Feature{
		ID: 1, Kind: MushroomIsland,
		CenterX: 0, CenterZ: 0,
		RadiusX: 100, RadiusZ: 100,
		WaterContext: "ocean",
	}
	base := testOceanIslandBase()
	got := Apply(
		[]Feature{feature},
		base,
		0.50,
		90, 0,
		"geoworld-rules-v1.1.24",
	)
	if got.World.WaterKind != reconcile.WaterNone {
		t.Fatalf("v1.1.24 island geometry changed: water=%v", got.World.WaterKind)
	}
}

func TestR82V125IslandSitingVersionGate(t *testing.T) {
	if !usesSubmergedIslandSiting("geoworld-rules-v1.1.25") {
		t.Fatal("v1.1.25 island siting gate is disabled")
	}
	for _, version := range []string{
		"geoworld-rules-v1.1.24",
		"geoworld-rules-v1.1.23",
		"",
	} {
		if usesSubmergedIslandSiting(version) {
			t.Fatalf("historical rules %q adopted v1.1.25 island siting", version)
		}
	}
}
