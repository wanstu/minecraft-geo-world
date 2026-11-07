package generator

import (
	"context"
	"math"
	"testing"

	"geoworld/internal/reconcile"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/world"
)

func TestV111SpecialLandscapePlanAndRuntimeOverlay(t *testing.T) {
	cfg := testConfig(123456)
	cfg.WorldProfile = StandardWorldProfile()
	cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.11"
	cfg.Geometry = world.GridGeometry{
		Width: 96, Height: 60,
		OriginX: -6144, OriginZ: -3840,
		SpanXBlocks: 95 * world.DefaultMacroSpacingBlocks,
		SpanZBlocks: 59 * world.DefaultMacroSpacingBlocks,
	}
	cfg.PlateCount = 18
	model, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if model.Config.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.11" {
		t.Fatalf("rules=%q", model.Config.WorldProfile.GenerationRulesVersion)
	}
	if len(model.SpecialFeatures) == 0 {
		t.Fatal("v1.1.11 generated no special landscapes")
	}

	found := map[specialfeatures.Kind]bool{}
	for _, feature := range model.SpecialFeatures {
		found[feature.Kind] = true
		if feature.Kind == specialfeatures.PaleGardenIsland &&
			feature.WaterContext != "ocean" {
			t.Fatalf("pale garden must be ocean-only: %+v", feature)
		}
		if feature.Kind == specialfeatures.CherryIsland &&
			feature.WaterContext != "ocean" &&
			feature.WaterContext != "lake" {
			t.Fatalf("cherry island has invalid water context: %+v", feature)
		}
		if feature.Kind != specialfeatures.MushroomIsland &&
			feature.Kind != specialfeatures.PaleGardenIsland &&
			feature.Kind != specialfeatures.CherryIsland {
			continue
		}
		base := model.RuntimeProfile.Sample(feature.CenterX, feature.CenterZ)
		got := specialfeatures.Apply(
			model.SpecialFeatures,
			base,
			model.Config.SeaLevelNormalized,
			feature.CenterX,
			feature.CenterZ,
			"geoworld-rules-v1.1.11",
		)
		if got.SpecialFeature != string(feature.Kind) {
			t.Fatalf("feature center kind=%q want=%q context=%q baseWater=%v center=(%.0f,%.0f)", got.SpecialFeature, feature.Kind, feature.WaterContext, base.World.WaterKind, feature.CenterX, feature.CenterZ)
		}
		if got.World.WaterKind != reconcile.WaterNone {
			t.Fatalf("feature center remains water: kind=%q water=%v", feature.Kind, got.World.WaterKind)
		}
		if got.World.GroundHeight <= model.Config.SeaLevelNormalized {
			t.Fatalf("feature center not raised above sea: kind=%q ground=%f sea=%f",
				feature.Kind, got.World.GroundHeight, model.Config.SeaLevelNormalized)
		}
	}
	for _, kind := range []specialfeatures.Kind{
		specialfeatures.MushroomIsland,
		specialfeatures.PaleGardenIsland,
		specialfeatures.CherryIsland,
		specialfeatures.Sinkhole,
	} {
		if !found[kind] {
			t.Fatalf("missing special landscape kind %q; got=%v", kind, found)
		}
	}
}

func TestV114DepressionKindsAreDistinctAndCarved(t *testing.T) {
	cfg := testConfig(123456)
	cfg.WorldProfile = StandardWorldProfile()
	cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.14"
	cfg.Geometry = world.GridGeometry{
		Width: 160, Height: 100,
		OriginX: -10240, OriginZ: -6400,
		SpanXBlocks: 159 * world.DefaultMacroSpacingBlocks,
		SpanZBlocks: 99 * world.DefaultMacroSpacingBlocks,
	}
	cfg.PlateCount = 18

	model, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if model.Config.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.14" {
		t.Fatalf(
			"rules=%q want geoworld-rules-v1.1.14",
			model.Config.WorldProfile.GenerationRulesVersion,
		)
	}

	sinkholes := 0
	craters := 0
	basins := 0
	for _, feature := range model.SpecialFeatures {
		switch feature.Kind {
		case specialfeatures.Sinkhole:
			sinkholes++
			if maxFloat(feature.RadiusX, feature.RadiusZ) > 280 {
				t.Fatalf("sinkhole too large: %+v", feature)
			}
		case specialfeatures.VolcanicCraterSnow, specialfeatures.VolcanicCraterCherry:
			craters++
			if maxFloat(feature.RadiusX, feature.RadiusZ) > 300 {
				t.Fatalf("volcanic crater too large: %+v", feature)
			}
		case specialfeatures.BasinCherry, specialfeatures.BasinSnow:
			basins++
			if maxFloat(feature.RadiusX, feature.RadiusZ) < 500 {
				t.Fatalf("basin too small: %+v", feature)
			}
		default:
			continue
		}

		if feature.RimHeight > feature.BaseHeight+1e-9 {
			t.Fatalf("non-island special terrain raised its rim: %+v", feature)
		}
		if feature.FloorHeight >= feature.BaseHeight {
			t.Fatalf("depression does not cut below source terrain: %+v", feature)
		}
	}

	if sinkholes == 0 || craters == 0 || basins == 0 {
		t.Fatalf(
			"v1.1.14 must keep sinkhole/crater/basin distinct; sinkholes=%d craters=%d basins=%d",
			sinkholes,
			craters,
			basins,
		)
	}
}

func TestV115LakeSpecialIslandsKeepWaterMoat(t *testing.T) {
	verifyLakeSpecialIslandsKeepWaterMoat(t, "geoworld-rules-v1.1.15")
}

func TestV1116LakeSpecialIslandsKeepWaterMoat(t *testing.T) {
	verifyLakeSpecialIslandsKeepWaterMoat(t, "geoworld-rules-v1.1.16")
}

func verifyLakeSpecialIslandsKeepWaterMoat(t *testing.T, rulesVersion string) {
	t.Helper()
	seeds := []int64{
		123456,
		71,
		10400,
		-4852173505788415804,
		-563659577384754347,
		4034658472718554561,
		-1273393314970095624,
	}
	lakeIslands := 0
	for _, seed := range seeds {
		cfg := testConfig(seed)
		cfg.WorldProfile = StandardWorldProfile()
		cfg.WorldProfile.GenerationRulesVersion = rulesVersion
		cfg.Geometry = world.GridGeometry{
			Width: 160, Height: 100,
			OriginX: -10240, OriginZ: -6400,
			SpanXBlocks: 159 * world.DefaultMacroSpacingBlocks,
			SpanZBlocks: 99 * world.DefaultMacroSpacingBlocks,
		}
		cfg.PlateCount = 18

		model, err := GenerateWorld(context.Background(), cfg, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, feature := range model.SpecialFeatures {
			if feature.WaterContext != "lake" {
				continue
			}
			if feature.Kind != specialfeatures.MushroomIsland &&
				feature.Kind != specialfeatures.CherryIsland {
				continue
			}
			lakeIslands++
			ring := maxFloat(feature.RadiusX, feature.RadiusZ) * 1.35
			for ai := 0; ai < 24; ai++ {
				angle := 2 * math.Pi * float64(ai) / 24
				x := feature.CenterX + math.Cos(angle)*ring
				z := feature.CenterZ + math.Sin(angle)*ring
				base := model.RuntimeProfile.Sample(x, z)
				if base.World.WaterKind != reconcile.WaterLake {
					t.Fatalf(
						"lake island lost water moat: rules=%s seed=%d kind=%s center=(%.0f,%.0f) angle=%.2f radius=%.1f baseWater=%v",
						rulesVersion, seed, feature.Kind, feature.CenterX, feature.CenterZ,
						angle, ring, base.World.WaterKind,
					)
				}
				got := specialfeatures.Apply(
					model.SpecialFeatures,
					base,
					model.Config.SeaLevelNormalized,
					x,
					z,
					rulesVersion,
				)
				if got.World.WaterKind != reconcile.WaterLake {
					t.Fatalf(
						"lake island overlay reached moat: rules=%s seed=%d kind=%s angle=%.2f water=%v",
						rulesVersion, seed, feature.Kind, angle, got.World.WaterKind,
					)
				}
			}
		}
		if lakeIslands > 0 {
			break
		}
	}
	if lakeIslands == 0 {
		t.Fatalf("%s fixed seed set generated no eligible lake special island", rulesVersion)
	}
}

func TestV1116OceanSpecialIslandsKeepOceanMoat(t *testing.T) {
	seeds := []int64{
		123456,
		71,
		10400,
		-4852173505788415804,
		-563659577384754347,
		4034658472718554561,
		-1273393314970095624,
	}
	oceanIslands := 0
	for _, seed := range seeds {
		cfg := testConfig(seed)
		cfg.WorldProfile = StandardWorldProfile()
		cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.16"
		cfg.Geometry = world.GridGeometry{
			Width: 160, Height: 100,
			OriginX: -10240, OriginZ: -6400,
			SpanXBlocks: 159 * world.DefaultMacroSpacingBlocks,
			SpanZBlocks: 99 * world.DefaultMacroSpacingBlocks,
		}
		cfg.PlateCount = 18

		model, err := GenerateWorld(context.Background(), cfg, nil)
		if err != nil {
			t.Fatal(err)
		}
		for _, feature := range model.SpecialFeatures {
			if feature.WaterContext != "ocean" {
				continue
			}
			if feature.Kind != specialfeatures.MushroomIsland &&
				feature.Kind != specialfeatures.PaleGardenIsland &&
				feature.Kind != specialfeatures.CherryIsland {
				continue
			}
			oceanIslands++
			for ai := 0; ai < 48; ai++ {
				angle := 2 * math.Pi * float64(ai) / 48
				edge := specialIslandInfluenceEdge(feature, angle)
				for _, moat := range []float64{16, 32} {
					radius := edge + moat
					x := feature.CenterX + math.Cos(angle)*radius
					z := feature.CenterZ + math.Sin(angle)*radius
					base := model.RuntimeProfile.Sample(x, z)
					if base.World.WaterKind != reconcile.WaterOcean {
						t.Fatalf(
							"ocean island lost ocean moat: seed=%d kind=%s center=(%.0f,%.0f) angle=%.2f edge=%.1f moat=%.0f baseWater=%v sourceLand=%v",
							seed, feature.Kind, feature.CenterX, feature.CenterZ,
							angle, edge, moat, base.World.WaterKind, base.SourceLand,
						)
					}
					got := specialfeatures.Apply(
						model.SpecialFeatures,
						base,
						model.Config.SeaLevelNormalized,
						x,
						z,
						"geoworld-rules-v1.1.16",
					)
					if got.World.WaterKind != reconcile.WaterOcean {
						t.Fatalf(
							"ocean island overlay reached moat: seed=%d kind=%s angle=%.2f edge=%.1f moat=%.0f water=%v",
							seed, feature.Kind, angle, edge, moat, got.World.WaterKind,
						)
					}
				}
			}
		}
	}
	if oceanIslands == 0 {
		t.Fatal("v1.1.16 fixed seed set generated no ocean special island")
	}
}

func specialIslandInfluenceEdge(feature specialfeatures.Feature, angle float64) float64 {
	dx := math.Cos(angle)
	dz := math.Sin(angle)
	lo := 0.0
	hi := math.Max(feature.RadiusX, feature.RadiusZ) * 3
	for _, lobe := range feature.Lobes {
		reach := math.Hypot(lobe.OffsetX, lobe.OffsetZ) +
			math.Max(lobe.RadiusX, lobe.RadiusZ)
		if reach > hi {
			hi = reach * 1.25
		}
	}
	for specialfeatures.Influence(
		feature,
		feature.CenterX+dx*hi,
		feature.CenterZ+dz*hi,
	) > 0 {
		hi *= 1.5
	}
	for i := 0; i < 40; i++ {
		mid := (lo + hi) * 0.5
		if specialfeatures.Influence(
			feature,
			feature.CenterX+dx*mid,
			feature.CenterZ+dz*mid,
		) > 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	return hi
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func TestLockedV110HasNoSpecialLandscapes(t *testing.T) {
	cfg := testConfig(123456)
	cfg.WorldProfile = StandardWorldProfile()
	cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.10"
	model, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.SpecialFeatures) != 0 {
		t.Fatalf("locked v1.1.10 gained special landscapes: %d", len(model.SpecialFeatures))
	}
}
