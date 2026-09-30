package geomorph

import (
	"math"
	"testing"

	"geoworld/internal/world"
)

func TestUpstreamMemoryDecayStableAcrossResolution(t *testing.T) {
	makeInput := func(width int, cellSize float64) Input {
		height := 3
		in := baseInput(width, height)
		in.Geometry = world.GridGeometry{
			Width: width, Height: height,
			SpanXBlocks: float64(width-1) * cellSize,
			SpanZBlocks: float64(height-1) * cellSize,
		}
		for i := range in.LandMask {
			in.LandMask[i] = false
			in.FlowTo[i] = -1
			in.Slope[i] = 0
			in.MountainWeight[i] = 0
			in.FoothillWeight[i] = 0
			in.FlowAccumulation[i] = 0
		}

		z := 1
		for x := 0; x < width; x++ {
			i := z*width + x
			in.LandMask[i] = true
			in.RollingPlainWeight[i] = 1
			in.FlowAccumulation[i] = float64(x+1) * 1000
			if x+1 < width {
				in.FlowTo[i] = i + 1
			}
		}

		source := z * width
		in.Slope[source] = 0.05
		in.MountainWeight[source] = 1
		in.RollingPlainWeight[source] = 0
		return in
	}

	coarse, err := Generate(makeInput(9, 128), Config{})
	if err != nil {
		t.Fatal(err)
	}
	fine, err := Generate(makeInput(17, 64), Config{})
	if err != nil {
		t.Fatal(err)
	}

	for x := 0; x < coarse.Width; x++ {
		ci := coarse.Width + x
		fi := fine.Width + x*2

		if math.Abs(coarse.UpstreamSteepMemory[ci]-fine.UpstreamSteepMemory[fi]) > 1e-12 {
			t.Fatalf("steep memory changed at physical x=%d blocks: coarse=%.12f fine=%.12f",
				x*128, coarse.UpstreamSteepMemory[ci], fine.UpstreamSteepMemory[fi])
		}
		if math.Abs(coarse.UpstreamMountainMemory[ci]-fine.UpstreamMountainMemory[fi]) > 1e-12 {
			t.Fatalf("mountain memory changed at physical x=%d blocks: coarse=%.12f fine=%.12f",
				x*128, coarse.UpstreamMountainMemory[ci], fine.UpstreamMountainMemory[fi])
		}
		if math.Abs(coarse.SlopeDrop[ci]-fine.SlopeDrop[fi]) > 1e-12 {
			t.Fatalf("slope drop changed at physical x=%d blocks: coarse=%.12f fine=%.12f",
				x*128, coarse.SlopeDrop[ci], fine.SlopeDrop[fi])
		}
	}
}

func TestAlluvialFanSpreadStableAcrossResolution(t *testing.T) {
	makeCase := func(width int, cellSize float64) (*Model, Input) {
		height := width
		n := width * height
		geometry := world.GridGeometry{
			Width: width, Height: height,
			SpanXBlocks: float64(width-1) * cellSize,
			SpanZBlocks: float64(height-1) * cellSize,
		}
		input := Input{
			Width:              width,
			Height:             height,
			Geometry:           geometry,
			LandMask:           make([]bool, n),
			Slope:              make([]float64, n),
			ReliefIndex:        make([]float64, n),
			FlowTo:             make([]int, n),
			FlowAccumulation:   make([]float64, n),
			LakeID:             make([]int, n),
			RiverThreshold:     1000,
			RollingPlainWeight: make([]float64, n),
			FoothillWeight:     make([]float64, n),
		}
		for i := 0; i < n; i++ {
			input.LandMask[i] = true
			input.FlowTo[i] = -1
			input.RollingPlainWeight[i] = 1
		}

		model := &Model{
			Config:            normalizeConfig(Config{}),
			Geometry:          geometry,
			Width:             width,
			Height:            height,
			LandMask:          append([]bool(nil), input.LandMask...),
			AlluvialFanCore:   make([]float64, n),
			AlluvialFanWeight: make([]float64, n),
		}

		cx, cz := (width-1)/2, (height-1)/2
		core := cz*width + cx
		input.FlowTo[core] = core + 1
		input.FlowAccumulation[core] = 7000
		model.AlluvialFanCore[core] = 0.8
		model.AlluvialFanWeight[core] = 0.8
		model.spreadAlluvialFans(input)
		return model, input
	}

	coarse, _ := makeCase(17, 128)
	fine, _ := makeCase(33, 64)

	for z := 0; z < coarse.Height; z++ {
		for x := 0; x < coarse.Width; x++ {
			ci := z*coarse.Width + x
			fi := (z*2)*fine.Width + x*2
			if math.Abs(coarse.AlluvialFanWeight[ci]-fine.AlluvialFanWeight[fi]) > 1e-12 {
				t.Fatalf("fan spread changed at physical node %d,%d: coarse=%.12f fine=%.12f",
					x, z, coarse.AlluvialFanWeight[ci], fine.AlluvialFanWeight[fi])
			}
		}
	}
}

func TestGeomorphRejectsGeometryDimensionMismatch(t *testing.T) {
	in := baseInput(5, 5)
	in.Geometry = world.GridGeometry{Width: 6, Height: 5, SpanXBlocks: 640, SpanZBlocks: 512}
	if _, err := Generate(in, Config{}); err == nil {
		t.Fatal("expected mismatched geomorph geometry to be rejected")
	}
}
