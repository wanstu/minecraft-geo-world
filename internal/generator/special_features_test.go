package generator

import (
	"context"
	"testing"

	"geoworld/internal/reconcile"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/world"
)

func TestV111SpecialLandscapePlanAndRuntimeOverlay(t *testing.T) {
	cfg := testConfig(123456)
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
