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
