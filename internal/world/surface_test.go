package world

import (
	"math"
	"reflect"
	"testing"
)

func TestSurfaceDistinguishesIslandOceanAndEnclosedDepression(t *testing.T) {
	values := []float64{
		.2, .2, .2, .2, .2, .2, .2,
		.2, .8, .8, .8, .8, .8, .2,
		.2, .8, .1, .1, .1, .8, .2,
		.2, .8, .1, .9, .1, .8, .2,
		.2, .8, .8, .8, .8, .8, .2,
		.2, .2, .2, .2, .2, .2, .2,
	}
	before := append([]float64(nil), values...)
	closed, err := ClassifySurface(7, 6, values, .5)
	if err != nil {
		t.Fatal(err)
	}
	if closed.LandMask[0] || !closed.LandMask[8] || !closed.EnclosedBelowSea[16] {
		t.Fatal("ocean, island and enclosed lowland not separated")
	}
	if !reflect.DeepEqual(before, values) {
		t.Fatal("classification modified elevations")
	}
	// A real below-sea channel connects the basin to the external sea.
	values[9] = .3
	open, err := ClassifySurface(7, 6, values, .5)
	if err != nil {
		t.Fatal(err)
	}
	if open.LandMask[16] || open.EnclosedBelowSea[16] {
		t.Fatal("connected channel did not inundate basin")
	}
	if !open.LandMask[24] {
		t.Fatal("emerged island was flooded")
	}
}

func TestOceanDoesNotLeakThroughDiagonalOrSeaLevelSill(t *testing.T) {
	values := []float64{.2, .8, .8, .8, .2, .8, .8, .8, .8}
	s, err := ClassifySurface(3, 3, values, .5)
	if err != nil {
		t.Fatal(err)
	}
	if !s.EnclosedBelowSea[4] {
		t.Fatal("diagonal corner incorrectly flooded")
	}
	values[1] = .5
	s, err = ClassifySurface(3, 3, values, .5)
	if err != nil {
		t.Fatal(err)
	}
	if !s.LandMask[1] || !s.EnclosedBelowSea[4] {
		t.Fatal("zero-depth sill became an open passage")
	}
}

func TestSurfaceWithoutOceanAndInputValidation(t *testing.T) {
	s, err := ClassifySurface(3, 3, []float64{.8, .8, .8, .8, .1, .8, .8, .8, .8}, .5)
	if err != nil {
		t.Fatal(err)
	}
	for _, land := range s.LandMask {
		if !land {
			t.Fatal("created ocean without boundary outlet")
		}
	}
	if !s.EnclosedBelowSea[4] {
		t.Fatal("lost enclosed lowland")
	}
	for _, sea := range []float64{0, 1, math.NaN(), math.Inf(1)} {
		if _, err := ClassifySurface(2, 2, []float64{.2, .2, .2, .2}, sea); err == nil {
			t.Fatal("accepted invalid sea level")
		}
	}
	for _, v := range []float64{math.NaN(), math.Inf(1), -.1, 1.1} {
		if _, err := ClassifySurface(2, 2, []float64{v, .2, .2, .2}, .5); err == nil {
			t.Fatal("accepted invalid elevation")
		}
	}
	if _, err := ClassifySurface(2, 2, []float64{.2}, .5); err == nil {
		t.Fatal("accepted short elevation array")
	}
}
