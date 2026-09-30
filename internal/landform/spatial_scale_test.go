package landform

import (
	"math"
	"testing"

	"geoworld/internal/world"
)

func TestMountainInfluencePhysicalScaleStableAcrossResolution(t *testing.T) {
	makeInput := func(width, height int, spanX, spanZ float64) Input {
		in := syntheticInput(width, height)
		in.Geometry = world.GridGeometry{
			Width: width, Height: height,
			SpanXBlocks: spanX, SpanZBlocks: spanZ,
		}
		cx, cz := (width-1)/2, (height-1)/2
		i := cz*width + cx
		in.Elevation[i] = 0.88
		in.ReliefIndex[i] = 0.95
		in.Slope[i] = 0.09
		in.TectonicSupport[i] = 0.95
		return in
	}

	coarse, err := Generate(makeInput(17, 17, 2048, 2048), Config{})
	if err != nil {
		t.Fatal(err)
	}
	fine, err := Generate(makeInput(33, 33, 2048, 2048), Config{})
	if err != nil {
		t.Fatal(err)
	}

	for z := 0; z < coarse.Height; z++ {
		for x := 0; x < coarse.Width; x++ {
			ci := z*coarse.Width + x
			fi := (z*2)*fine.Width + x*2
			if math.Abs(coarse.MountainDistance[ci]-fine.MountainDistance[fi]) > 1e-12 {
				t.Fatalf("mountain distance changed at physical node %d,%d: coarse=%.12f fine=%.12f",
					x, z, coarse.MountainDistance[ci], fine.MountainDistance[fi])
			}
			if math.Abs(coarse.MountainInfluence[ci]-fine.MountainInfluence[fi]) > 1e-12 {
				t.Fatalf("mountain influence changed at physical node %d,%d: coarse=%.12f fine=%.12f",
					x, z, coarse.MountainInfluence[ci], fine.MountainInfluence[fi])
			}
		}
	}
}

func TestValleySpreadPhysicalScaleStableAcrossResolution(t *testing.T) {
	makeInput := func(width, height int, spanX, spanZ float64) Input {
		in := syntheticInput(width, height)
		in.Geometry = world.GridGeometry{
			Width: width, Height: height,
			SpanXBlocks: spanX, SpanZBlocks: spanZ,
		}
		cx, cz := (width-1)/2, (height-1)/2
		i := cz*width + cx
		in.ValleyIncision[i] = 0.01
		in.FluvialPower[i] = 0.8
		return in
	}

	coarse, err := Generate(makeInput(17, 17, 2048, 2048), Config{})
	if err != nil {
		t.Fatal(err)
	}
	fine, err := Generate(makeInput(33, 33, 2048, 2048), Config{})
	if err != nil {
		t.Fatal(err)
	}

	for z := 0; z < coarse.Height; z++ {
		for x := 0; x < coarse.Width; x++ {
			ci := z*coarse.Width + x
			fi := (z*2)*fine.Width + x*2
			if math.Abs(coarse.ValleyWeight[ci]-fine.ValleyWeight[fi]) > 1e-12 {
				t.Fatalf("valley spread changed at physical node %d,%d: coarse=%.12f fine=%.12f",
					x, z, coarse.ValleyWeight[ci], fine.ValleyWeight[fi])
			}
		}
	}
}

func TestLandformRejectsGeometryDimensionMismatch(t *testing.T) {
	in := syntheticInput(5, 5)
	in.Geometry = world.GridGeometry{Width: 6, Height: 5, SpanXBlocks: 640, SpanZBlocks: 512}
	if _, err := Generate(in, Config{}); err == nil {
		t.Fatal("expected mismatched landform geometry to be rejected")
	}
}
