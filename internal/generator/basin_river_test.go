package generator

import (
	"context"
	"math"
	"testing"

	"geoworld/internal/reconcile"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/world"
)

func TestReportedBasinRiverBelowBanks(t *testing.T) {
	cfg := Config{
		Seed: 123456,
		Geometry: world.GridGeometry{Width: 320, Height: 200, OriginX: -20416, OriginZ: -12736,
			SpanXBlocks: 40832, SpanZBlocks: 25472},
		WorldProfile: StandardWorldProfile(), PlateCount: 18, OceanFraction: 0.68,
		ContinentFragmentation: 0.35000000000000003, ContinentSeparation: 0.65,
	}
	cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.16"
	model, err := GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	sample := func(x, z float64) float64 {
		base := model.RuntimeProfile.Sample(x, z)
		got := specialfeatures.Apply(model.SpecialFeatures, base, 0.5, x, z, cfg.WorldProfile.GenerationRulesVersion)
		return got.World.GroundHeight
	}
	checked, raised := 0, 0
	maxExcess := 0.0
	for z := 2500.0; z <= 3000; z += 4 {
		for x := 5200.0; x <= 5700; x += 4 {
			base := model.RuntimeProfile.Sample(x, z)
			if base.World.WaterKind != reconcile.WaterRiver {
				continue
			}
			got := specialfeatures.Apply(model.SpecialFeatures, base, 0.5, x, z, cfg.WorldProfile.GenerationRulesVersion)
			if got.SpecialFeature != string(specialfeatures.BasinCherry) && got.SpecialFeature != string(specialfeatures.BasinSnow) {
				continue
			}
			if base.World.GroundHeight-got.World.GroundHeight < 0.005 {
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
	t.Logf("reported basin seed=123456 checks=%d raised=%d maxNormalizedExcess=%.6f", checked, raised, maxExcess)
	if checked == 0 {
		t.Fatal("regression did not cover a river/basin intersection")
	}
	if raised > 0 {
		t.Fatalf("river water above nearest dry bank: %d/%d samples", raised, checked)
	}
}
