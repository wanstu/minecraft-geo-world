package world_test

import (
	"geoworld/internal/climate"
	"geoworld/internal/ecology"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
	"geoworld/internal/world"
	"reflect"
	"testing"
)

func TestMacroSurfacePropagatesThroughEntirePipeline(t *testing.T) {
	for _, seed := range []int64{731894205, 10400} {
		w := phase1.Generate(phase1.Config{Seed: seed, Width: 240, Height: 150, PlateCount: 18, OceanFraction: .68})
		input := climate.InputFromPhase1(w)
		c, err := climate.Generate(input, climate.Config{})
		if err != nil {
			t.Fatal(err)
		}
		h, err := hydrology.Generate(hydrology.InputFromWorld(w, c), hydrology.Config{})
		if err != nil {
			t.Fatal(err)
		}
		e, err := erosion.Generate(erosion.InputFromWorld(w, c, h), erosion.Config{})
		if err != nil {
			t.Fatal(err)
		}
		l, err := landform.Generate(landform.InputFromWorld(w, e), landform.Config{})
		if err != nil {
			t.Fatal(err)
		}
		g, err := geomorph.Generate(geomorph.InputFromModels(h, e, l), geomorph.Config{})
		if err != nil {
			t.Fatal(err)
		}
		eco, err := ecology.Generate(ecology.InputFromModels(c, h, e, l, g), ecology.Config{})
		if err != nil {
			t.Fatal(err)
		}
		p, err := mcprofile.Generate(mcprofile.InputFromModels(eco, l, g))
		if err != nil {
			t.Fatal(err)
		}
		for name, mask := range map[string][]bool{"climate": c.LandMask, "hydrology": h.LandMask, "erosion": e.LandMask, "landform": l.LandMask, "geomorph": g.LandMask, "ecology": eco.LandMask, "profile": p.LandMask} {
			if !reflect.DeepEqual(mask, w.Surface.LandMask) {
				t.Fatalf("seed %d: %s reconstructed another mask", seed, name)
			}
		}
		emergedIslands := 0
		for i, land := range w.Surface.LandMask {
			if !land && w.Elevation[i] >= world.DefaultSeaLevelNormalized {
				t.Fatalf("ocean above sea at %d", i)
			}
			if land && w.Crust[i] == phase1.Oceanic && w.Elevation[i] >= world.DefaultSeaLevelNormalized {
				emergedIslands++
				if c.CurrentX[i] != 0 || c.CurrentZ[i] != 0 {
					t.Fatal("ocean current on emerged island")
				}
				if p.TerrainVariant[i] == mcprofile.TerrainOcean {
					t.Fatal("island mapped to ocean terrain")
				}
			}
		}
		if emergedIslands == 0 {
			t.Fatalf("seed %d did not exercise emerged oceanic crust", seed)
		}
		// Downstream callers receive copies and cannot change the upstream model.
		before := w.Surface.LandMask[0]
		input.LandMask[0] = !before
		if w.Surface.LandMask[0] != before {
			t.Fatal("adapter aliases shared surface")
		}
	}
}
