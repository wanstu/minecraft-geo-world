package specialfeatures

import (
	"geoworld/internal/reconcile"
	"math"
	"reflect"
	"testing"
)

func TestR81rFlowIsOneConnectedChannelAndApron(t *testing.T) {
	f := Feature{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744, RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626}
	phase := float64(f.ID) * 0.413
	c, s := math.Cos(phase), math.Sin(phase)
	onAxis := func(u float64) (x, z, radial float64) {
		v := 0.14*math.Sin(3.1*u+phase) + 0.035*math.Sin(7.2*u-phase*0.53)
		dx, dz := c*u-s*v, s*u+c*v
		return f.CenterX + dx*f.RadiusX, f.CenterZ + dz*f.RadiusZ, math.Hypot(dx, dz)
	}
	upstream := 0
	for _, u := range []float64{-0.75, -0.6, -0.45, -0.30} {
		x, z, r := onAxis(u)
		if sinkholeFlowAlignedTransfer(f, x, z, r) >= 0 {
			t.Fatalf("source not eroded at u=%g", u)
		}
		upstream++
	}
	for _, u := range []float64{0.22, 0.35, 0.45} {
		x, z, r := onAxis(u)
		if sinkholeFlowAlignedTransfer(f, x, z, r) <= 0 {
			t.Fatalf("fan not deposited at u=%g", u)
		}
	}
	if upstream < 4 {
		t.Fatal("insufficient connected upstream samples")
	}
	if sinkholeFlowAlignedTransfer(f, f.CenterX, f.CenterZ, 0) != 0 {
		t.Fatal("center changed")
	}
	for _, r := range []float64{0.97, 1.0, 1.10} {
		if sinkholeFlowAlignedTransfer(f, f.CenterX+r*f.RadiusX, f.CenterZ, r) != 0 {
			t.Fatal("effect escaped rim")
		}
	}
}

func TestR81rOldRulesAndWaterUnaffected(t *testing.T) {
	if usesFlowAlignedSinkhole("geoworld-rules-v1.1.30") || !usesFlowAlignedSinkhole("geoworld-rules-v1.1.31") {
		t.Fatal("flow algorithm version not isolated")
	}
	f := Feature{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744, RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626}
	base := testHabitableDepressionBase()
	base.World.GroundHeight = f.BaseHeight
	const fresh = "geoworld-rules-v1.1.31"
	const old = "geoworld-rules-v1.1.29"
	centerA := Apply([]Feature{f}, base, 0.49, f.CenterX, f.CenterZ, old)
	centerB := Apply([]Feature{f}, base, 0.49, f.CenterX, f.CenterZ, fresh)
	if centerA.World.GroundHeight != centerB.World.GroundHeight {
		t.Fatal("planned center level changed")
	}
	if out := Apply([]Feature{f}, base, 0.49, f.CenterX+1.1*f.RadiusX, f.CenterZ, fresh); !reflect.DeepEqual(out, base) {
		t.Fatal("outside footprint changed")
	}
	river := base
	river.World.WaterKind = reconcile.WaterRiver
	river.World.WaterSurfaceDefined = true
	river.World.WaterSurfaceElevation = 0.61
	if out := Apply([]Feature{f}, river, 0.49, f.CenterX+60, f.CenterZ, fresh); !reflect.DeepEqual(out, river) {
		t.Fatal("changed river")
	}
	for _, kind := range []Kind{BasinCherry, BasinSnow, VolcanicCraterCherry, VolcanicCraterSnow} {
		other := f
		other.Kind = kind
		for _, p := range [][2]float64{{f.CenterX, f.CenterZ}, {f.CenterX + 70, f.CenterZ + 10}} {
			a := Apply([]Feature{other}, base, 0.49, p[0], p[1], old)
			b := Apply([]Feature{other}, base, 0.49, p[0], p[1], fresh)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("changed unrelated landform %s", kind)
			}
		}
	}
}

func TestR81rNoSecondaryPointSink(t *testing.T) {
	f := Feature{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744, RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626}
	base := testHabitableDepressionBase()
	base.World.GroundHeight = f.BaseHeight
	const v = "geoworld-rules-v1.1.31"
	center := Apply([]Feature{f}, base, 0.49, f.CenterX, f.CenterZ, v).World.GroundHeight
	for i := 0; i < 24; i++ {
		a := 2 * math.Pi * float64(i) / 24
		for d := 6.0; d <= 0.80*f.RadiusX; d += 6 {
			x := f.CenterX + d*math.Cos(a)
			z := f.CenterZ + d*f.RadiusZ/f.RadiusX*math.Sin(a)
			y := Apply([]Feature{f}, base, 0.49, x, z, v).World.GroundHeight
			if y+1.0/514.0 < center {
				t.Fatalf("new sink below planned center at ray=%d d=%.0f", i, d)
			}
		}
	}
}
