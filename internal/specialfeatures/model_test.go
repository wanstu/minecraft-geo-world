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

func testHabitableDepressionBase() runtimeprofile.Sample {
	return runtimeprofile.Sample{
		World: worldsample.Sample{
			GroundHeight: 0.66,
			WaterKind:    reconcile.WaterNone,
		},
		SourceLand:        true,
		EcoRegion:         ecology.RegionTemperateForest,
		BiomeFamily:       mcprofile.BiomeTemperateForest,
		TerrainVariant:    mcprofile.TerrainHills,
		SurfaceProfile:    mcprofile.SurfaceForestLoam,
		VegetationProfile: mcprofile.VegetationBroadleafForest,
	}
}

func TestV112BasinCarvesExistingTerrainWithoutArtificialRim(t *testing.T) {
	for _, kind := range []Kind{BasinCherry, BasinSnow} {
		feature := Feature{
			ID:          1,
			Kind:        kind,
			CenterX:     0,
			CenterZ:     0,
			RadiusX:     1000,
			RadiusZ:     1000,
			BaseHeight:  0.66,
			FloorHeight: 0.54,
			RimHeight:   0.66,
		}
		base := testHabitableDepressionBase()
		center := Apply([]Feature{feature}, base, 0.40, 0, 0, "geoworld-rules-v1.1.14")
		inner := Apply([]Feature{feature}, base, 0.40, 300, 0, "geoworld-rules-v1.1.14")
		edge := Apply([]Feature{feature}, base, 0.40, 920, 0, "geoworld-rules-v1.1.14")

		if center.TerrainVariant != mcprofile.TerrainPlain ||
			center.BiomeFamily != mcprofile.BiomeTemperateGrassland {
			t.Fatalf("basin floor is not a habitable plain: kind=%s terrain=%v biome=%v", kind, center.TerrainVariant, center.BiomeFamily)
		}
		if math.Abs(center.World.GroundHeight-inner.World.GroundHeight) > 0.012 {
			t.Fatalf("basin center is not flat enough: kind=%s center=%.6f inner=%.6f", kind, center.World.GroundHeight, inner.World.GroundHeight)
		}
		if center.World.GroundHeight >= edge.World.GroundHeight {
			t.Fatalf("basin did not carve a depression: kind=%s center=%.6f edge=%.6f", kind, center.World.GroundHeight, edge.World.GroundHeight)
		}
		if edge.TerrainVariant == mcprofile.TerrainMountain {
			t.Fatalf("basin invented a mountain rim on hill terrain: kind=%s", kind)
		}
		assertCarvedOnly(t, feature, base)
	}
}

func TestV112SinkholeIsSmallCarvedDepressionWithIrregularWall(t *testing.T) {
	feature := Feature{
		ID:          1,
		Kind:        Sinkhole,
		CenterX:     0,
		CenterZ:     0,
		RadiusX:     1000,
		RadiusZ:     1000,
		BaseHeight:  0.66,
		FloorHeight: 0.52,
		RimHeight:   0.66,
	}
	base := testHabitableDepressionBase()
	center := Apply([]Feature{feature}, base, 0.40, 0, 0, "geoworld-rules-v1.1.14")
	inner := Apply([]Feature{feature}, base, 0.40, 300, 0, "geoworld-rules-v1.1.14")
	if center.TerrainVariant != mcprofile.TerrainPlain ||
		center.BiomeFamily != mcprofile.BiomeTemperateGrassland {
		t.Fatalf("sinkhole floor is not a habitable plain: terrain=%v biome=%v", center.TerrainVariant, center.BiomeFamily)
	}
	if math.Abs(center.World.GroundHeight-inner.World.GroundHeight) > 0.012 {
		t.Fatalf("sinkhole center is not flat enough: center=%.6f inner=%.6f", center.World.GroundHeight, inner.World.GroundHeight)
	}

	var minWall, maxWall = 1.0, 0.0
	for _, angle := range []float64{0, math.Pi / 4, math.Pi / 2, 3 * math.Pi / 4, math.Pi} {
		x := math.Cos(angle) * 680
		z := math.Sin(angle) * 680
		s := Apply([]Feature{feature}, base, 0.40, x, z, "geoworld-rules-v1.1.14")
		minWall = math.Min(minWall, s.World.GroundHeight)
		maxWall = math.Max(maxWall, s.World.GroundHeight)
	}
	if maxWall-minWall < 0.006 {
		t.Fatalf("sinkhole wall remained unnaturally uniform: min=%.6f max=%.6f", minWall, maxWall)
	}
	assertCarvedOnly(t, feature, base)
}

func TestV112VolcanicCraterOnlyCutsExistingPeak(t *testing.T) {
	feature := Feature{
		ID:          3,
		Kind:        VolcanicCraterCherry,
		CenterX:     0,
		CenterZ:     0,
		RadiusX:     1000,
		RadiusZ:     1000,
		BaseHeight:  0.78,
		FloorHeight: 0.70,
		RimHeight:   0.78,
	}
	base := testHabitableDepressionBase()
	base.World.GroundHeight = 0.78
	base.TerrainVariant = mcprofile.TerrainMountain

	center := Apply([]Feature{feature}, base, 0.40, 0, 0, "geoworld-rules-v1.1.14")
	outer := Apply([]Feature{feature}, base, 0.40, 850, 0, "geoworld-rules-v1.1.14")
	if center.World.GroundHeight >= base.World.GroundHeight {
		t.Fatalf("volcanic crater did not cut into the existing peak")
	}
	if outer.World.GroundHeight > base.World.GroundHeight+1e-9 {
		t.Fatalf("volcanic crater created an artificial cone: got=%.6f base=%.6f", outer.World.GroundHeight, base.World.GroundHeight)
	}
	assertCarvedOnly(t, feature, base)
}

func TestV113AndV114DepressionSemanticsStayVersionLocked(t *testing.T) {
	feature := Feature{
		ID:          7,
		Kind:        VolcanicCraterCherry,
		CenterX:     0,
		CenterZ:     0,
		RadiusX:     1000,
		RadiusZ:     1000,
		BaseHeight:  0.66,
		FloorHeight: 0.56,
		RimHeight:   0.82,
	}
	base := testHabitableDepressionBase()

	legacy := Apply(
		[]Feature{feature},
		base,
		0.40,
		840,
		0,
		"geoworld-rules-v1.1.13",
	)
	if legacy.World.GroundHeight <= base.World.GroundHeight {
		t.Fatalf(
			"v1.1.13 crater semantics changed; locked rim no longer raises ground: got=%.6f base=%.6f",
			legacy.World.GroundHeight,
			base.World.GroundHeight,
		)
	}

	carved := Apply(
		[]Feature{feature},
		base,
		0.40,
		840,
		0,
		"geoworld-rules-v1.1.14",
	)
	if carved.World.GroundHeight > base.World.GroundHeight+1e-9 {
		t.Fatalf(
			"v1.1.14 crater raised existing terrain: got=%.6f base=%.6f",
			carved.World.GroundHeight,
			base.World.GroundHeight,
		)
	}
}

func TestV115BasinConformsExistingRiverToDepressionFloor(t *testing.T) {
	feature := Feature{
		ID:          11,
		Kind:        BasinCherry,
		CenterX:     0,
		CenterZ:     0,
		RadiusX:     900,
		RadiusZ:     900,
		BaseHeight:  0.68,
		FloorHeight: 0.54,
		RimHeight:   0.68,
	}
	base := testHabitableDepressionBase()
	base.World.GroundHeight = 0.68
	base.World.WaterKind = reconcile.WaterRiver
	base.World.WaterSurfaceDefined = true
	base.World.WaterSurfaceElevation = 0.70
	base.World.WaterDepth = 0.02
	base.World.RiverWeight = 1

	locked := Apply(
		[]Feature{feature}, base, 0.40, 0, 0,
		"geoworld-rules-v1.1.14",
	)
	if locked.World.GroundHeight != base.World.GroundHeight ||
		locked.World.WaterSurfaceElevation != base.World.WaterSurfaceElevation {
		t.Fatalf("v1.1.14 river/basin behavior changed: got=%+v base=%+v", locked.World, base.World)
	}

	got := Apply(
		[]Feature{feature}, base, 0.40, 0, 0,
		"geoworld-rules-v1.1.15",
	)
	if got.World.WaterKind != reconcile.WaterRiver {
		t.Fatalf("river identity lost in basin: got=%v", got.World.WaterKind)
	}
	if !got.World.WaterSurfaceDefined {
		t.Fatal("river water surface became undefined in basin")
	}
	if got.World.GroundHeight >= base.World.GroundHeight {
		t.Fatalf("basin did not lower river bed: got=%.6f base=%.6f", got.World.GroundHeight, base.World.GroundHeight)
	}
	if got.World.WaterSurfaceElevation >= base.World.WaterSurfaceElevation {
		t.Fatalf("basin did not lower river water surface: got=%.6f base=%.6f", got.World.WaterSurfaceElevation, base.World.WaterSurfaceElevation)
	}
	if math.Abs(got.World.WaterDepth-base.World.WaterDepth) > 1e-9 {
		t.Fatalf("basin changed river depth: got=%.6f base=%.6f", got.World.WaterDepth, base.World.WaterDepth)
	}
	if math.Abs((got.World.WaterSurfaceElevation-got.World.GroundHeight)-base.World.WaterDepth) > 1e-9 {
		t.Fatalf("river surface no longer follows basin floor: ground=%.6f surface=%.6f depth=%.6f", got.World.GroundHeight, got.World.WaterSurfaceElevation, got.World.WaterDepth)
	}
}

func assertCarvedOnly(t *testing.T, feature Feature, base runtimeprofile.Sample) {
	t.Helper()
	for ai := 0; ai < 16; ai++ {
		angle := 2 * math.Pi * float64(ai) / 16
		for ri := 0; ri <= 10; ri++ {
			r := float64(ri) / 10
			x := math.Cos(angle) * feature.RadiusX * r
			z := math.Sin(angle) * feature.RadiusZ * r
			got := Apply([]Feature{feature}, base, 0.40, x, z, "geoworld-rules-v1.1.14")
			if got.World.GroundHeight > base.World.GroundHeight+1e-9 {
				t.Fatalf(
					"non-island special terrain raised ground: kind=%s r=%.2f angle=%.2f got=%.6f base=%.6f",
					feature.Kind, r, angle, got.World.GroundHeight, base.World.GroundHeight,
				)
			}
		}
	}
}
