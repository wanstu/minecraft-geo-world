package specialfeatures

import (
	"math"
	"testing"

	"geoworld/internal/reconcile"
)

// A basin must retain the river's incision below its dry floor, and the
// common water datum must not depend on the depth of each cross-section column.
func TestBasinRiverWaterStaysBelowDryFloor(t *testing.T) {
	for _, kind := range []Kind{BasinCherry, BasinSnow} {
		feature := Feature{ID: 1, Kind: kind, RadiusX: 1000, RadiusZ: 1000,
			BaseHeight: 0.68, FloorHeight: 0.54, RimHeight: 0.68}
		land := testHabitableDepressionBase()
		land.World.GroundHeight = 0.68
		for _, radius := range []float64{0, 300, 600, 750, 880, 1000} {
			floor := Apply([]Feature{feature}, land, 0.5, radius, 0, "geoworld-rules-v1.1.16")
			surface := math.NaN()
			for _, depth := range []float64{0.004, 0.020, 0.036} {
				river := land
				river.World.WaterKind = reconcile.WaterRiver
				river.World.WaterSurfaceDefined = true
				river.World.WaterSurfaceElevation = 0.67
				river.World.WaterDepth = depth
				river.World.GroundHeight = 0.67 - depth
				river.World.RiverWeight = 1
				got := Apply([]Feature{feature}, river, 0.5, radius, 0, "geoworld-rules-v1.1.16")
				if got.World.WaterSurfaceElevation > floor.World.GroundHeight+1e-9 {
					t.Fatalf("raised river kind=%s radius=%.0f depth=%.3f water=%.6f dryFloor=%.6f bed=%.6f",
						kind, radius, depth, got.World.WaterSurfaceElevation, floor.World.GroundHeight, got.World.GroundHeight)
				}
				if !math.IsNaN(surface) && math.Abs(surface-got.World.WaterSurfaceElevation) > 1e-9 {
					t.Fatalf("cross-section water varies with bed depth: first=%.6f current=%.6f", surface, got.World.WaterSurfaceElevation)
				}
				surface = got.World.WaterSurfaceElevation
				if got.World.GroundHeight > river.World.GroundHeight+1e-9 {
					t.Fatal("basin raised the original river bed")
				}
				if math.Abs(got.World.WaterDepth-depth) > 1e-9 {
					t.Fatalf("river lost depth: got=%.6f want=%.6f", got.World.WaterDepth, depth)
				}
			}
		}
	}
}

func TestV1118BasinDrainageKeepsRiverDatumAndShapesOutletValley(t *testing.T) {
	feature := Feature{
		ID:          1,
		Kind:        BasinCherry,
		CenterX:     0,
		CenterZ:     0,
		RadiusX:     1000,
		RadiusZ:     1000,
		BaseHeight:  0.68,
		FloorHeight: 0.54,
		RimHeight:   0.68,
		Drainage: []BasinDrainageReach{{
			X0: -1000, Z0: 0,
			X1: 1000, Z1: 0,
			Surface0:  0.67,
			Surface1:  0.65,
			HalfWidth: 12,
		}},
	}
	baseLand := testHabitableDepressionBase()
	baseLand.World.GroundHeight = 0.68

	x := 600.0
	expectedSurface := 0.654
	river := baseLand
	river.World.WaterKind = reconcile.WaterRiver
	river.World.WaterSurfaceDefined = true
	river.World.WaterSurfaceElevation = expectedSurface
	river.World.WaterDepth = 0.008
	river.World.GroundHeight = expectedSurface - river.World.WaterDepth
	river.World.RiverWeight = 1

	gotRiver := Apply(
		[]Feature{feature},
		river,
		0.5,
		x,
		0,
		"geoworld-rules-v1.1.18",
	)
	if math.Abs(gotRiver.World.WaterSurfaceElevation-expectedSurface) > 1e-12 {
		t.Fatalf(
			"v1.1.18 basin changed authoritative river datum: got=%.9f want=%.9f",
			gotRiver.World.WaterSurfaceElevation,
			expectedSurface,
		)
	}
	if math.Abs(gotRiver.World.GroundHeight-river.World.GroundHeight) > 1e-12 {
		t.Fatalf(
			"v1.1.18 basin changed authoritative river bed: got=%.9f want=%.9f",
			gotRiver.World.GroundHeight,
			river.World.GroundHeight,
		)
	}

	nearBank := Apply(
		[]Feature{feature},
		baseLand,
		0.5,
		x,
		32,
		"geoworld-rules-v1.1.18",
	)
	if nearBank.World.GroundHeight <= expectedSurface {
		t.Fatalf(
			"v1.1.18 outlet valley bank below river datum: ground=%.9f water=%.9f",
			nearBank.World.GroundHeight,
			expectedSurface,
		)
	}
	if nearBank.World.GroundHeight >= baseLand.World.GroundHeight-1e-9 {
		t.Fatalf(
			"v1.1.18 outlet valley did not participate in basin carving: ground=%.9f base=%.9f",
			nearBank.World.GroundHeight,
			baseLand.World.GroundHeight,
		)
	}

	outsideCorridor := Apply(
		[]Feature{feature},
		baseLand,
		0.5,
		x,
		260,
		"geoworld-rules-v1.1.18",
	)
	if outsideCorridor.World.GroundHeight >= nearBank.World.GroundHeight {
		t.Fatalf(
			"v1.1.18 drainage corridor did not form a river-centered valley: near=%.9f outside=%.9f",
			nearBank.World.GroundHeight,
			outsideCorridor.World.GroundHeight,
		)
	}

	legacy := Apply(
		[]Feature{feature},
		river,
		0.5,
		x,
		0,
		"geoworld-rules-v1.1.16",
	)
	if math.Abs(legacy.World.WaterSurfaceElevation-expectedSurface) <= 1e-9 {
		t.Fatal("v1.1.16 lock unexpectedly adopted v1.1.18 drainage semantics")
	}
}
