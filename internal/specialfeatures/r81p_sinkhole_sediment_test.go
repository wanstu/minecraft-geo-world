package specialfeatures

import (
	"math"
	"reflect"
	"testing"

	"geoworld/internal/reconcile"
)

func TestR81pSedimentTransferHasBothSourceAndToeslopeDeposit(t *testing.T) {
	f := Feature{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744, RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626}
	if usesSedimentFanSinkhole("geoworld-rules-v1.1.28") || !usesSedimentFanSinkhole("geoworld-rules-v1.1.29") {
		t.Fatal("candidate v1.1.29 must be the only sediment version")
	}
	erodedVolume, depositedVolume := 0.0, 0.0
	peakCut, peakFan := 0.0, 0.0
	for i := 0; i < 180; i++ {
		theta := 2 * math.Pi * (float64(i) + 0.5) / 180
		for j := 0; j < 160; j++ {
			r := (float64(j) + 0.5) / 160
			x := f.CenterX + r*f.RadiusX*math.Cos(theta)
			z := f.CenterZ + r*f.RadiusZ*math.Sin(theta)
			_, cut, deposit := sinkholeSedimentTransfer(f, x, z, r)
			erodedVolume += cut * r
			depositedVolume += deposit * r
			peakCut = math.Max(peakCut, cut)
			peakFan = math.Max(peakFan, deposit)
		}
	}
	if peakCut < 0.003 || peakFan < 0.003 || erodedVolume <= 0 {
		t.Fatalf("missing source/fan: cut=%g fan=%g total eroded=%g", peakCut, peakFan, erodedVolume)
	}
	ratio := depositedVolume / erodedVolume
	// Approximately matched *plan-view volume proxy*: not a physical
	// mass-conserving erosion or hydrology solver.
	if ratio < 0.65 || ratio > 1.65 {
		t.Fatalf("unbalanced transport proxy, deposited/eroded=%.3f", ratio)
	}
	for _, r := range []float64{0, 0.02, 0.98, 1.0, 1.10} {
		net, cut, fan := sinkholeSedimentTransfer(f, f.CenterX+r*f.RadiusX, f.CenterZ, r)
		if net != 0 || cut != 0 || fan != 0 {
			t.Fatalf("transport boundary r=%g: %g", r, net)
		}
	}
}

func TestR81pOldWorldAndWaterIsolation(t *testing.T) {
	f := Feature{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744, RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626}
	base := testHabitableDepressionBase()
	base.World.GroundHeight = f.BaseHeight
	const old = "geoworld-rules-v1.1.28"
	const next = "geoworld-rules-v1.1.29"
	centerOld := Apply([]Feature{f}, base, 0.49, f.CenterX, f.CenterZ, old)
	centerNew := Apply([]Feature{f}, base, 0.49, f.CenterX, f.CenterZ, next)
	if centerOld.World.GroundHeight != centerNew.World.GroundHeight {
		t.Fatalf("changed exact planned center: %g vs %g", centerOld.World.GroundHeight, centerNew.World.GroundHeight)
	}
	outside := Apply([]Feature{f}, base, 0.49, f.CenterX+1.1*f.RadiusX, f.CenterZ, next)
	if !reflect.DeepEqual(outside, base) {
		t.Fatal("candidate escaped old footprint")
	}
	wet := base
	wet.World.WaterKind = reconcile.WaterRiver
	wet.World.WaterSurfaceDefined = true
	wet.World.WaterSurfaceElevation = 0.61
	if v := Apply([]Feature{f}, wet, 0.49, f.CenterX+70, f.CenterZ, next); !reflect.DeepEqual(v, wet) {
		t.Fatal("candidate touched existing river")
	}
	for _, kind := range []Kind{BasinCherry, BasinSnow, VolcanicCraterCherry, VolcanicCraterSnow} {
		other := f
		other.Kind = kind
		for _, p := range [][2]float64{{f.CenterX, f.CenterZ}, {f.CenterX + 75, f.CenterZ + 15}} {
			a := Apply([]Feature{other}, base, 0.49, p[0], p[1], old)
			b := Apply([]Feature{other}, base, 0.49, p[0], p[1], next)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("changed unrelated %s", kind)
			}
		}
	}
}

func TestR81pRemovesBroadFlatCoreWithoutNewCliff(t *testing.T) {
	const old = "geoworld-rules-v1.1.28"
	const next = "geoworld-rules-v1.1.29"
	for _, f := range []Feature{
		{ID: 9, Kind: Sinkhole, CenterX: 6080, CenterZ: 7744, RadiusX: 190, RadiusZ: 174, BaseHeight: 0.626, FloorHeight: 0.532, RimHeight: 0.626},
		{ID: 11, Kind: Sinkhole, CenterX: 320, CenterZ: 6592, RadiusX: 153.631, RadiusZ: 130.49, BaseHeight: 0.60437718, FloorHeight: 0.52937718, RimHeight: 0.60437718},
	} {
		base := testHabitableDepressionBase()
		base.World.GroundHeight = f.BaseHeight
		at := func(v string, x, z float64) float64 {
			return Apply([]Feature{f}, base, 0.49, x, z, v).World.GroundHeight
		}
		// Test the center region at physical block step across many rays,
		// not only that some model point differs from the old rules.
		center := at(next, f.CenterX, f.CenterZ)
		oldCenter := at(old, f.CenterX, f.CenterZ)
		oldFlats, newFlats := 0, 0
		const blockHeight = 1.0 / 514.0
		for angleID := 0; angleID < 16; angleID++ {
			angle := 2 * math.Pi * float64(angleID) / 16
			for d := 4.0; d <= 40; d += 4 {
				x := f.CenterX + d*math.Cos(angle)
				z := f.CenterZ + d*math.Sin(angle)
				if math.Abs(at(old, x, z)-oldCenter) < 0.5*blockHeight {
					oldFlats++
				}
				if math.Abs(at(next, x, z)-center) < 0.5*blockHeight {
					newFlats++
				}
			}
			last := center
			for d := 4.0; d <= f.RadiusX; d += 4 {
				x := f.CenterX + d*math.Cos(angle)
				z := f.CenterZ + d*math.Sin(angle)
				y := at(next, x, z)
				grade := math.Abs(y-last) * 514 / 4
				if grade > 0.80 {
					t.Fatalf("candidate steepens slope id=%d ray=%d dist=%.0f grade=%.3f", f.ID, angleID, d, grade)
				}
				last = y
			}
		}
		if newFlats >= oldFlats {
			t.Fatalf("candidate did not reduce flat core id=%d previous=%d new=%d", f.ID, oldFlats, newFlats)
		}
	}
}
