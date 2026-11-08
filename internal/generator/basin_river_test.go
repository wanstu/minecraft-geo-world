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
	preservedDatumRaised := 0
	protectedCorridorRaised := 0
	maxExcess := 0.0
	maxPreservedDatumExcess := 0.0
	maxProtectedCorridorExcess := 0.0
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
			protectedBank := 0.0
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
						protectedBank = bank
						ground := model.Waterbed.Sample(bx, bz)
						if ground.RiverCarve > 1e-12 {
							protectedBank = b.World.GroundHeight
						}
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
			if base.World.WaterSurfaceDefined {
				preservedExcess := base.World.WaterSurfaceElevation - bank
				if preservedExcess > 0.002 {
					preservedDatumRaised++
					maxPreservedDatumExcess = math.Max(
						maxPreservedDatumExcess,
						preservedExcess,
					)
				}
				protectedExcess := base.World.WaterSurfaceElevation - protectedBank
				if protectedExcess > 0.002 {
					protectedCorridorRaised++
					maxProtectedCorridorExcess = math.Max(
						maxProtectedCorridorExcess,
						protectedExcess,
					)
				}
			}
		}
	}
	t.Logf(
		"reported basin seed=123456 checks=%d raised=%d maxNormalizedExcess=%.6f preserveOriginalDatumRaised=%d maxPreservedExcess=%.6f protectedCorridorRaised=%d maxProtectedCorridorExcess=%.6f",
		checked,
		raised,
		maxExcess,
		preservedDatumRaised,
		maxPreservedDatumExcess,
		protectedCorridorRaised,
		maxProtectedCorridorExcess,
	)
	if checked == 0 {
		t.Fatal("regression did not cover a river/basin intersection")
	}
	if raised > 0 {
		t.Fatalf("river water above nearest dry bank: %d/%d samples", raised, checked)
	}

	visibleUpslopes := 0
	maxVisibleRiseBlocks := 0.0
	maxVisibleRiseX, maxVisibleRiseZ := 0.0, 0.0
	baseVisibleUpslopes := 0
	maxBaseRiseBlocks := 0.0
	for _, seg := range model.Waterbed.Rivers {
		pathUpperBound := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0) +
			math.Pi*math.Abs(seg.MeanderOffset)
		steps := int(math.Ceil(pathUpperBound / 8.0))
		if steps < 2 {
			steps = 2
		}
		finalPrevSet := false
		basePrevSet := false
		finalPrevY := 0.0
		basePrevY := 0.0
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
				finalPrevSet = false
				basePrevSet = false
				continue
			}

			finalY := basinTestMinecraftY(got.World.WaterSurfaceElevation)
			if finalPrevSet {
				rise := finalY - finalPrevY
				if rise > 1.0 {
					visibleUpslopes++
				}
				if rise > maxVisibleRiseBlocks {
					maxVisibleRiseBlocks = rise
					maxVisibleRiseX = x
					maxVisibleRiseZ = z
				}
			}
			finalPrevY = finalY
			finalPrevSet = true

			if base.World.WaterKind == reconcile.WaterRiver &&
				base.World.WaterSurfaceDefined {
				baseY := basinTestMinecraftY(base.World.WaterSurfaceElevation)
				if basePrevSet {
					rise := baseY - basePrevY
					if rise > 1.0 {
						baseVisibleUpslopes++
					}
					maxBaseRiseBlocks = math.Max(maxBaseRiseBlocks, rise)
				}
				basePrevY = baseY
				basePrevSet = true
			} else {
				basePrevSet = false
			}
		}
	}
	t.Logf(
		"basin longitudinal monotonicity: finalVisibleUpslopes=%d maxFinalRiseBlocks=%.3f at=(%.1f,%.1f) baseVisibleUpslopes=%d maxBaseRiseBlocks=%.3f",
		visibleUpslopes,
		maxVisibleRiseBlocks,
		maxVisibleRiseX,
		maxVisibleRiseZ,
		baseVisibleUpslopes,
		maxBaseRiseBlocks,
	)
	if visibleUpslopes > 0 {
		t.Fatalf(
			"special basin overlay creates %d downstream river rises above one Minecraft block; max=%.3f blocks",
			visibleUpslopes,
			maxVisibleRiseBlocks,
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
