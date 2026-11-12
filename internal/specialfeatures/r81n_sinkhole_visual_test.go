package specialfeatures

import (
	"math"
	"reflect"
	"testing"

	"geoworld/internal/reconcile"
)

func TestR81nSinkholeDirectionalErosionOnlyInV127(t *testing.T) {
	feature := Feature{
		ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744,
		RadiusX: 190, RadiusZ: 174,
		BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626,
	}
	base := testHabitableDepressionBase()
	base.World.GroundHeight = feature.BaseHeight
	sample := func(version string, x, z float64) float64 {
		return Apply([]Feature{feature}, base, 0.49, x, z, version).World.GroundHeight
	}
	const old = "geoworld-rules-v1.1.26"
	const next = "geoworld-rules-v1.1.27"
	if !usesDirectionalSinkholeErosion(next) || usesDirectionalSinkholeErosion(old) {
		t.Fatal("R8.1n erosion version gate no longer preserves locked worlds")
	}
	for _, point := range [][2]float64{
		{6080, 7744}, {6150, 7755}, {6015, 7820}, {6080, 7580},
	} {
		got := sample(old, point[0], point[1])
		expected := Apply([]Feature{feature}, base, 0.49, point[0], point[1], old)
		if got != expected.World.GroundHeight {
			t.Fatal("v1.1.26 not deterministic")
		}
	}
	if centerOld, centerNew := sample(old, feature.CenterX, feature.CenterZ),
		sample(next, feature.CenterX, feature.CenterZ); centerOld != centerNew {
		t.Fatalf("new erosion changed exact planned center: old=%g new=%g", centerOld, centerNew)
	}
	outside := Apply([]Feature{feature}, base, 0.49,
		feature.CenterX+feature.RadiusX*1.1, feature.CenterZ, next)
	if !reflect.DeepEqual(outside, base) {
		t.Fatal("v1.1.27 carving escaped planned feature boundary")
	}
	wet := base
	wet.World.WaterKind = reconcile.WaterRiver
	wet.World.WaterSurfaceDefined = true
	wet.World.WaterSurfaceElevation = 0.61
	if got := Apply([]Feature{feature}, wet, 0.49, feature.CenterX+100, feature.CenterZ, next); !reflect.DeepEqual(got, wet) {
		t.Fatal("new sinkhole algorithm touched existing river water")
	}
	for _, kind := range []Kind{BasinCherry, BasinSnow, VolcanicCraterCherry, VolcanicCraterSnow} {
		other := feature
		other.Kind = kind
		for _, at := range [][2]float64{{feature.CenterX, feature.CenterZ}, {feature.CenterX + 90, feature.CenterZ + 40}} {
			a := Apply([]Feature{other}, base, 0.49, at[0], at[1], old)
			b := Apply([]Feature{other}, base, 0.49, at[0], at[1], next)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("new slope unexpectedly changed %s", kind)
			}
		}
	}
}

func TestR81nNaturalSlopeBreaksConcentricContoursWithoutCliffs(t *testing.T) {
	const old = "geoworld-rules-v1.1.26"
	const next = "geoworld-rules-v1.1.27"
	for _, feature := range []Feature{
		{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744,
			RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626},
		{ID: 11, Kind: Sinkhole, CenterX: 320, CenterZ: 6592,
			RadiusX: 153.631, RadiusZ: 130.49, BaseHeight: 0.60437718,
			FloorHeight: 0.52937718, RimHeight: 0.60437718},
	} {
		base := testHabitableDepressionBase()
		base.World.GroundHeight = feature.BaseHeight
		sample := func(v string, x, z float64) float64 {
			return Apply([]Feature{feature}, base, 0.49, x, z, v).World.GroundHeight
		}
		// Compare the same distorted old rim coordinates: a new natural
		// channel must actually change sampled heights, not just metadata.
		changed := 0
		difference := 0.0
		for i := 0; i < 72; i++ {
			theta := float64(i) * 2 * math.Pi / 72
			x := feature.CenterX + 0.57*feature.RadiusX*math.Cos(theta)
			z := feature.CenterZ + 0.57*feature.RadiusZ*math.Sin(theta)
			before, after := sample(old, x, z), sample(next, x, z)
			if math.Abs(after-before) > 0.002 {
				changed++
			}
			difference = math.Max(difference, math.Abs(after-before))
			if after > base.World.GroundHeight+1e-9 {
				t.Fatalf("new natural sinkhole raised original land id=%d", feature.ID)
			}
		}
		if changed < 25 || difference < 0.006 {
			t.Fatalf("new sinkhole remains radially uniform id=%d: changed=%d max difference=%.5f",
				feature.ID, changed, difference)
		}
		// Neighboring samples of the same synthetic world must not contain
		// sheer cliffs; proxy translates normalized heights by 514 blocks.
		for i := 0; i < 8; i++ {
			theta := float64(i) * math.Pi / 4
			last := sample(next, feature.CenterX, feature.CenterZ)
			for dist := 4.0; dist <= feature.RadiusX*1.08; dist += 4 {
				x := feature.CenterX + dist*math.Cos(theta)
				z := feature.CenterZ + dist*feature.RadiusZ/feature.RadiusX*math.Sin(theta)
				cur := sample(next, x, z)
				grade := math.Abs(cur-last) * 514 / 4
				if grade > 0.95 {
					t.Fatalf("unacceptable wall cliff id=%d angle=%d at r=%.0f: grade=%.3f",
						feature.ID, i, dist, grade)
				}
				last = cur
			}
		}
	}
}
