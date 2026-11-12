package specialfeatures

import (
	"math"
	"reflect"
	"testing"

	"geoworld/internal/reconcile"
)

func TestR81oAsymmetricFloorVersionIsolationAndOtherKinds(t *testing.T) {
	if usesAsymmetricSinkholeFloor("geoworld-rules-v1.1.27") ||
		!usesAsymmetricSinkholeFloor("geoworld-rules-v1.1.28") {
		t.Fatal("v1.1.28-only gate broken")
	}
	f := Feature{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744,
		RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626}
	base := testHabitableDepressionBase()
	base.World.GroundHeight = f.BaseHeight
	old, next := "geoworld-rules-v1.1.27", "geoworld-rules-v1.1.28"
	at := func(v string, x, z float64) float64 {
		return Apply([]Feature{f}, base, 0.49, x, z, v).World.GroundHeight
	}
	if a, b := at(old, f.CenterX, f.CenterZ), at(next, f.CenterX, f.CenterZ); a != b {
		t.Fatalf("floor center moved old=%g next=%g", a, b)
	}
	if out := Apply([]Feature{f}, base, 0.49, f.CenterX+f.RadiusX*1.1, f.CenterZ, next); !reflect.DeepEqual(out, base) {
		t.Fatal("new floor escaped the planned feature")
	}
	wet := base
	wet.World.WaterKind = reconcile.WaterRiver
	wet.World.WaterSurfaceDefined = true
	wet.World.WaterSurfaceElevation = 0.61
	if got := Apply([]Feature{f}, wet, 0.49, f.CenterX+60, f.CenterZ, next); !reflect.DeepEqual(got, wet) {
		t.Fatal("new floor touched river source")
	}
	for _, kind := range []Kind{BasinCherry, BasinSnow, VolcanicCraterCherry, VolcanicCraterSnow} {
		copy := f
		copy.Kind = kind
		for _, p := range [][2]float64{{f.CenterX, f.CenterZ}, {f.CenterX + 70, f.CenterZ + 30}} {
			a := Apply([]Feature{copy}, base, 0.49, p[0], p[1], old)
			b := Apply([]Feature{copy}, base, 0.49, p[0], p[1], next)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("changed unrelated %s", kind)
			}
		}
	}
}

func TestR81oCoreBreaksCircularMeadowAndIsNotSteep(t *testing.T) {
	old, next := "geoworld-rules-v1.1.27", "geoworld-rules-v1.1.28"
	for _, f := range []Feature{
		{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744,
			RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626},
		{ID: 11, Kind: Sinkhole, CenterX: 320, CenterZ: 6592,
			RadiusX: 153.631, RadiusZ: 130.49, BaseHeight: 0.60437718,
			FloorHeight: 0.52937718, RimHeight: 0.60437718},
	} {
		base := testHabitableDepressionBase()
		base.World.GroundHeight = f.BaseHeight
		at := func(v string, x, z float64) float64 {
			return Apply([]Feature{f}, base, 0.49, x, z, v).World.GroundHeight
		}
		changed := 0
		meadowSwitch := 0
		samples := 72
		for i := 0; i < samples; i++ {
			angle := float64(i) * 2 * math.Pi / float64(samples)
			x := f.CenterX + 0.3*f.RadiusX*math.Cos(angle)
			z := f.CenterZ + 0.3*f.RadiusZ*math.Sin(angle)
			a := Apply([]Feature{f}, base, 0.49, x, z, old)
			b := Apply([]Feature{f}, base, 0.49, x, z, next)
			if math.Abs(a.World.GroundHeight-b.World.GroundHeight) > .001 {
				changed++
			}
			if a.TerrainVariant != b.TerrainVariant {
				meadowSwitch++
			}
			if b.World.GroundHeight > base.World.GroundHeight+1e-10 {
				t.Fatalf("raised original natural terrain id=%d", f.ID)
			}
		}
		if changed < 30 || meadowSwitch < 10 {
			t.Fatalf("new core did not break circle id=%d changed=%d vegetation=%d", f.ID, changed, meadowSwitch)
		}
		for i := 0; i < 16; i++ {
			a := float64(i) * math.Pi / 8
			last := at(next, f.CenterX, f.CenterZ)
			for d := 4.0; d < f.RadiusX*1.08; d += 4 {
				x := f.CenterX + d*math.Cos(a)
				z := f.CenterZ + d*f.RadiusZ/f.RadiusX*math.Sin(a)
				y := at(next, x, z)
				grade := math.Abs(y-last) * 514 / 4
				if grade > 0.96 {
					t.Fatalf("new local cliff id=%d angle=%d distance=%.0f grade=%.3f", f.ID, i, d, grade)
				}
				last = y
			}
		}
	}
}
