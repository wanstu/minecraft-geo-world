package specialfeatures

import (
	"math"
	"reflect"
	"testing"

	"geoworld/internal/reconcile"
)

func TestR81qVersionAndWaterIsolation(t *testing.T) {
	if usesNonConcentricSinkhole("geoworld-rules-v1.1.29") ||
		!usesNonConcentricSinkhole("geoworld-rules-v1.1.30") {
		t.Fatal("compound trough must be v1.1.30 only")
	}
	f := Feature{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744, RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626}
	base := testHabitableDepressionBase()
	base.World.GroundHeight = f.BaseHeight
	center0 := Apply([]Feature{f}, base, 0.49, f.CenterX, f.CenterZ, "geoworld-rules-v1.1.29")
	center1 := Apply([]Feature{f}, base, 0.49, f.CenterX, f.CenterZ, "geoworld-rules-v1.1.30")
	if center0.World.GroundHeight != center1.World.GroundHeight {
		t.Fatalf("center moved: old=%g new=%g", center0.World.GroundHeight, center1.World.GroundHeight)
	}
	if got := Apply([]Feature{f}, base, 0.49, f.CenterX+f.RadiusX*1.04, f.CenterZ, "geoworld-rules-v1.1.30"); !reflect.DeepEqual(got, base) {
		t.Fatal("compound trough escaped footprint")
	}
	river := base
	river.World.WaterKind = reconcile.WaterRiver
	river.World.WaterSurfaceDefined = true
	river.World.WaterSurfaceElevation = 0.61
	if got := Apply([]Feature{f}, river, 0.49, f.CenterX+60, f.CenterZ, "geoworld-rules-v1.1.30"); !reflect.DeepEqual(got, river) {
		t.Fatal("compound trough touched protected river")
	}
	for _, kind := range []Kind{BasinCherry, BasinSnow, VolcanicCraterCherry, VolcanicCraterSnow} {
		feature := f
		feature.Kind = kind
		for _, p := range [][2]float64{{f.CenterX, f.CenterZ}, {f.CenterX + 75, f.CenterZ + 15}} {
			a := Apply([]Feature{feature}, base, 0.49, p[0], p[1], "geoworld-rules-v1.1.29")
			b := Apply([]Feature{feature}, base, 0.49, p[0], p[1], "geoworld-rules-v1.1.30")
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("changed unrelated %s", kind)
			}
		}
	}
}

func TestR81qCompoundContourAndNoNewCliff(t *testing.T) {
	for _, f := range []Feature{
		{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744, RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626},
		{ID: 11, Kind: Sinkhole, CenterX: 320, CenterZ: 6592, RadiusX: 153.631, RadiusZ: 130.49, BaseHeight: 0.60437718, FloorHeight: 0.52937718, RimHeight: 0.60437718},
	} {
		base := testHabitableDepressionBase()
		base.World.GroundHeight = f.BaseHeight
		at := func(ver string, x, z float64) float64 {
			return Apply([]Feature{f}, base, 0.49, x, z, ver).World.GroundHeight
		}
		radiusFieldChanges := 0
		for _, r := range []float64{0.22, 0.37, 0.51, 0.63} {
			oldMin, oldMax := math.Inf(1), math.Inf(-1)
			newMin, newMax := math.Inf(1), math.Inf(-1)
			for i := 0; i < 72; i++ {
				a := float64(i) * 2 * math.Pi / 72
				x, z := f.CenterX+r*f.RadiusX*math.Cos(a), f.CenterZ+r*f.RadiusZ*math.Sin(a)
				oldY, newY := at("geoworld-rules-v1.1.29", x, z), at("geoworld-rules-v1.1.30", x, z)
				oldMin = math.Min(oldMin, oldY)
				oldMax = math.Max(oldMax, oldY)
				newMin = math.Min(newMin, newY)
				newMax = math.Max(newMax, newY)
			}
			if newMax-newMin > oldMax-oldMin+0.002 {
				radiusFieldChanges++
			}
		}
		if radiusFieldChanges < 2 {
			t.Fatalf("compound trough did not increase angular contour variation id=%d changes=%d", f.ID, radiusFieldChanges)
		}
		for i := 0; i < 24; i++ {
			a := float64(i) * 2 * math.Pi / 24
			last := at("geoworld-rules-v1.1.30", f.CenterX, f.CenterZ)
			for d := 4.0; d <= f.RadiusX; d += 4 {
				x := f.CenterX + d*math.Cos(a)
				z := f.CenterZ + d*f.RadiusZ/f.RadiusX*math.Sin(a)
				height := at("geoworld-rules-v1.1.30", x, z)
				grade := math.Abs(height-last) * 514 / 4
				if grade > 0.96 {
					t.Fatalf("cliff id=%d ray=%d dist=%g grade=%.3f", f.ID, i, d, grade)
				}
				last = height
			}
		}
	}
}
