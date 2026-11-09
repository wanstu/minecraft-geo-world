package generator

import (
	"context"
	"math"
	"testing"

	"geoworld/internal/reconcile"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

func TestCurrentBasinRiverDrainageIsContainedAndMonotonic(t *testing.T) {
	cfg := Config{
		Seed: 123456,
		Geometry: world.GridGeometry{
			Width: 320, Height: 200,
			OriginX: -20416, OriginZ: -12736,
			SpanXBlocks: 40832, SpanZBlocks: 25472,
		},
		WorldProfile:           StandardWorldProfile(),
		PlateCount:             18,
		OceanFraction:          0.68,
		ContinentFragmentation: 0.35000000000000003,
		ContinentSeparation:    0.65,
	}
	if cfg.WorldProfile.GenerationRulesVersion != "geoworld-rules-v1.1.19" {
		t.Fatalf("current rules=%q want geoworld-rules-v1.1.19", cfg.WorldProfile.GenerationRulesVersion)
	}

	model, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}

	drainedBasins := 0
	totalDrainageReaches := 0
	maxDrainageReaches := 0
	for _, feature := range model.SpecialFeatures {
		if feature.Kind != specialfeatures.BasinCherry &&
			feature.Kind != specialfeatures.BasinSnow {
			continue
		}
		if len(feature.Drainage) > 0 {
			drainedBasins++
			totalDrainageReaches += len(feature.Drainage)
			if len(feature.Drainage) > maxDrainageReaches {
				maxDrainageReaches = len(feature.Drainage)
			}
		}
	}
	if drainedBasins == 0 {
		t.Fatal("current fixed seed produced no basin with a persisted drainage corridor")
	}

	sample := func(x, z float64) float64 {
		base := model.RuntimeProfile.Sample(x, z)
		got := specialfeatures.Apply(
			model.SpecialFeatures,
			base,
			model.Config.SeaLevelNormalized,
			x,
			z,
			cfg.WorldProfile.GenerationRulesVersion,
		)
		return got.World.GroundHeight
	}

	// Preserve the historical cross-section contract: basin rivers may not sit
	// above their nearest dry bank after the basin overlay.
	checked, raised := 0, 0
	maxExcess := 0.0
	for z := 2500.0; z <= 3000; z += 4 {
		for x := 5200.0; x <= 5700; x += 4 {
			base := model.RuntimeProfile.Sample(x, z)
			if base.World.WaterKind != reconcile.WaterRiver {
				continue
			}
			got := specialfeatures.Apply(
				model.SpecialFeatures,
				base,
				model.Config.SeaLevelNormalized,
				x,
				z,
				cfg.WorldProfile.GenerationRulesVersion,
			)
			if got.SpecialFeature != string(specialfeatures.BasinCherry) &&
				got.SpecialFeature != string(specialfeatures.BasinSnow) {
				continue
			}
			nearest := math.Inf(1)
			bank := 0.0
			for a := 0; a < 16; a++ {
				theta := 2 * math.Pi * float64(a) / 16
				for d := 4.0; d <= 64; d += 4 {
					bx, bz := x+math.Cos(theta)*d, z+math.Sin(theta)*d
					b := model.RuntimeProfile.Sample(bx, bz)
					if b.World.WaterKind != reconcile.WaterNone {
						continue
					}
					if d < nearest {
						nearest = d
						bank = sample(bx, bz)
					}
					break
				}
			}
			if math.IsInf(nearest, 1) {
				continue
			}
			checked++
			excess := got.World.WaterSurfaceElevation - bank
			if excess > 0.002 {
				raised++
				maxExcess = math.Max(maxExcess, excess)
			}
		}
	}
	t.Logf(
		"current basin containment: drainedBasins=%d drainageReaches=%d maxPerBasin=%d checks=%d raised=%d maxNormalizedExcess=%.6f",
		drainedBasins,
		totalDrainageReaches,
		maxDrainageReaches,
		checked,
		raised,
		maxExcess,
	)
	if checked == 0 {
		t.Fatal("regression did not cover a river/basin intersection")
	}
	if raised > 0 {
		t.Fatalf("river water above nearest dry bank: %d/%d samples", raised, checked)
	}

	// The base Waterbed profile is already downstream-monotonic. The special
	// basin overlay must not introduce a new, player-visible climb. We sample
	// every eight blocks along real meandering centerlines and allow only a
	// small sub-block numerical/continuous-surface tolerance.
	basinSamples := 0
	risesAboveQuarterBlock := 0
	maxRiseBlocks := 0.0
	maxRiseX, maxRiseZ := 0.0, 0.0
	for _, seg := range model.Waterbed.Rivers {
		pathUpperBound := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0) +
			math.Pi*math.Abs(seg.MeanderOffset)
		steps := int(math.Ceil(pathUpperBound / 8.0))
		if steps < 2 {
			steps = 2
		}
		prevSet := false
		prevY := 0.0
		for j := 0; j <= steps; j++ {
			tt := float64(j) / float64(steps)
			x, z := waterbed.RiverCenterPoint(seg, tt)
			base := model.RuntimeProfile.Sample(x, z)
			got := specialfeatures.Apply(
				model.SpecialFeatures,
				base,
				model.Config.SeaLevelNormalized,
				x,
				z,
				cfg.WorldProfile.GenerationRulesVersion,
			)
			inBasin := got.SpecialFeature == string(specialfeatures.BasinCherry) ||
				got.SpecialFeature == string(specialfeatures.BasinSnow)
			if !inBasin ||
				got.World.WaterKind != reconcile.WaterRiver ||
				!got.World.WaterSurfaceDefined {
				prevSet = false
				continue
			}
			basinSamples++
			y := basinTestMinecraftY(got.World.WaterSurfaceElevation)
			if prevSet {
				rise := y - prevY
				if rise > 0.25 {
					risesAboveQuarterBlock++
				}
				if rise > maxRiseBlocks {
					maxRiseBlocks = rise
					maxRiseX = x
					maxRiseZ = z
				}
			}
			prevY = y
			prevSet = true
		}
	}
	t.Logf(
		"current basin longitudinal profile: samples=%d rises>0.25block=%d maxRiseBlocks=%.3f at=(%.1f,%.1f)",
		basinSamples,
		risesAboveQuarterBlock,
		maxRiseBlocks,
		maxRiseX,
		maxRiseZ,
	)
	if basinSamples == 0 {
		t.Fatal("regression did not sample a river through a special basin")
	}
	if risesAboveQuarterBlock > 0 {
		t.Fatalf(
			"special basin overlay creates %d downstream rises above 0.25 block; max=%.3f blocks",
			risesAboveQuarterBlock,
			maxRiseBlocks,
		)
	}
}

func basinTestMinecraftY(height float64) float64 {
	height = math.Max(0, math.Min(1, height))
	if height <= 0.5 {
		return -64 + 127*(height/0.5)
	}
	return 63 + 257*((height-0.5)/0.5)
}
