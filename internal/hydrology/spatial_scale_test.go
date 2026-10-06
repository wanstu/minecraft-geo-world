package hydrology

import (
	"math"
	"testing"

	"geoworld/internal/world"
)

func TestFillEpsilonUsesPhysicalStepAcrossResolution(t *testing.T) {
	makeInput := func(width int, span float64) Input {
		n := width * width
		in := Input{
			Width:               width,
			Height:              width,
			Geometry:            world.GridGeometry{Width: width, Height: width, SpanXBlocks: span, SpanZBlocks: span},
			Elevation:           make([]float64, n),
			LandMask:            make([]bool, n),
			AnnualPrecipMM:      make([]float64, n),
			RainfallSeasonality: make([]float64, n),
		}
		for z := 0; z < width; z++ {
			for x := 0; x < width; x++ {
				i := z*width + x
				in.LandMask[i] = true
				in.Elevation[i] = 0.60
				if x == 0 || z == 0 || x == width-1 || z == width-1 {
					in.Elevation[i] = 0.80
				}
			}
		}
		return in
	}

	coarse, err := Generate(makeInput(9, 1024), Config{})
	if err != nil {
		t.Fatal(err)
	}
	fine, err := Generate(makeInput(17, 1024), Config{})
	if err != nil {
		t.Fatal(err)
	}

	coarseCenter := coarse.FilledElevation[4*coarse.Width+4]
	fineCenter := fine.FilledElevation[8*fine.Width+8]
	if math.Abs(coarseCenter-fineCenter) > 1e-12 {
		t.Fatalf("filled center changed with resolution: coarse=%.12f fine=%.12f", coarseCenter, fineCenter)
	}
}

func TestDownstreamSlopeUsesPhysicalDistanceAcrossResolution(t *testing.T) {
	makeModel := func(width int, cellSize float64) *Model {
		geometry := world.GridGeometry{
			Width:       width,
			Height:      3,
			SpanXBlocks: float64(width-1) * cellSize,
			SpanZBlocks: 2 * cellSize,
		}
		n := width * 3
		m := &Model{
			Config:          normalizeConfig(Config{}),
			Geometry:        geometry,
			Width:           width,
			Height:          3,
			LandMask:        make([]bool, n),
			FilledElevation: make([]float64, n),
			FlowTo:          make([]int, n),
		}
		for i := range m.LandMask {
			m.LandMask[i] = true
			m.FlowTo[i] = -1
		}
		for z := 0; z < 3; z++ {
			for x := 0; x < width; x++ {
				i := z*width + x
				worldX, _ := geometry.NodePosition(x, z)
				m.FilledElevation[i] = 0.90 - worldX/10240
			}
		}
		return m
	}

	coarse := makeModel(9, 128)
	fine := makeModel(17, 64)
	ci := coarse.Width + 4
	fi := fine.Width + 8
	coarse.FlowTo[ci] = ci + 1
	fine.FlowTo[fi] = fi + 1

	coarseSlope := coarse.downstreamSlope(ci)
	fineSlope := fine.downstreamSlope(fi)
	if math.Abs(coarseSlope-fineSlope) > 1e-12 {
		t.Fatalf("downstream slope changed with resolution: coarse=%.12f fine=%.12f", coarseSlope, fineSlope)
	}
}

func TestLocalRunoffScalesWithPhysicalCellArea(t *testing.T) {
	makeModel := func(cellSize float64) *Model {
		return &Model{
			Config: normalizeConfig(Config{}),
			Geometry: world.GridGeometry{
				Width: 3, Height: 3,
				SpanXBlocks: 2 * cellSize,
				SpanZBlocks: 2 * cellSize,
			},
			Width:    3,
			Height:   3,
			LandMask: []bool{true, true, true, true, true, true, true, true, true},
			FlowTo:   make([]int, 9),
		}
	}

	precip := []float64{1200, 1200, 1200, 1200, 1200, 1200, 1200, 1200, 1200}
	seasonality := []float64{0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25, 0.25}

	coarse := makeModel(128)
	fine := makeModel(64)
	coarseLocal := coarse.runoffField(precip, seasonality, coarse.Config.RunoffLossMM)[4]
	fineLocal := fine.runoffField(precip, seasonality, fine.Config.RunoffLossMM)[4]
	if math.Abs(fineLocal*4-coarseLocal) > 1e-12 {
		t.Fatalf("runoff contribution did not scale with physical cell area: coarse=%.12f fine=%.12f", coarseLocal, fineLocal)
	}
}

func TestMinimumLakeAreaUsesPhysicalArea(t *testing.T) {
	makeModel := func(cellSize float64, candidateCount int) *Model {
		width, height := 12, 3
		geometry := world.GridGeometry{
			Width: width, Height: height,
			SpanXBlocks: float64(width-1) * cellSize,
			SpanZBlocks: float64(height-1) * cellSize,
		}
		n := width * height
		m := &Model{
			Config:           normalizeConfig(Config{}),
			Geometry:         geometry,
			Width:            width,
			Height:           height,
			LandMask:         make([]bool, n),
			FilledElevation:  make([]float64, n),
			FillDepth:        make([]float64, n),
			FlowTo:           make([]int, n),
			FlowAccumulation: make([]float64, n),
			BasinID:          make([]int, n),
			LakePotential:    make([]float64, n),
			LakeID:           make([]int, n),
		}
		for i := range m.FlowTo {
			m.FlowTo[i] = -1
		}
		for x := 1; x <= candidateCount; x++ {
			i := width + x
			m.LandMask[i] = true
			m.FilledElevation[i] = 0.70
			m.FillDepth[i] = 0.05
			m.LakePotential[i] = 1
			m.FlowAccumulation[i] = 1000
			m.BasinID[i] = 1
		}
		return m
	}

	coarse := makeModel(128, 2)
	coarse.identifyLakes()
	if len(coarse.Lakes) != 1 {
		t.Fatalf("two 128-block cells should satisfy default lake area, got %d lakes", len(coarse.Lakes))
	}

	fineTooSmall := makeModel(64, 2)
	fineTooSmall.identifyLakes()
	if len(fineTooSmall.Lakes) != 0 {
		t.Fatalf("two 64-block cells should be below default lake area, got %d lakes", len(fineTooSmall.Lakes))
	}

	fineEnough := makeModel(64, 8)
	fineEnough.identifyLakes()
	if len(fineEnough.Lakes) != 1 {
		t.Fatalf("eight 64-block cells should satisfy default lake area, got %d lakes", len(fineEnough.Lakes))
	}

	wantArea := 2 * world.DefaultMacroSpacingBlocks * world.DefaultMacroSpacingBlocks
	if math.Abs(coarse.Lakes[0].AreaBlocks2-wantArea) > 1e-12 ||
		math.Abs(fineEnough.Lakes[0].AreaBlocks2-wantArea) > 1e-12 {
		t.Fatalf("lake physical area mismatch: coarse=%.1f fine=%.1f want=%.1f",
			coarse.Lakes[0].AreaBlocks2, fineEnough.Lakes[0].AreaBlocks2, wantArea)
	}
}

func TestLakeComponentsDoNotMergeAcrossDifferentWaterLevels(t *testing.T) {
	width, height := 6, 3
	g := world.GridGeometry{
		Width: width, Height: height,
		SpanXBlocks: float64(width-1) * world.DefaultMacroSpacingBlocks,
		SpanZBlocks: float64(height-1) * world.DefaultMacroSpacingBlocks,
	}
	n := width * height
	m := &Model{
		Config:           normalizeConfig(Config{}),
		Geometry:         g,
		Width:            width,
		Height:           height,
		LandMask:         make([]bool, n),
		FilledElevation:  make([]float64, n),
		FillDepth:        make([]float64, n),
		FlowTo:           make([]int, n),
		FlowAccumulation: make([]float64, n),
		BasinID:          make([]int, n),
		LakePotential:    make([]float64, n),
		LakeID:           make([]int, n),
	}
	for i := range m.FlowTo {
		m.FlowTo[i] = -1
	}
	for x := 1; x <= 4; x++ {
		i := width + x
		m.LandMask[i] = true
		m.FillDepth[i] = 0.05
		m.LakePotential[i] = 1
		m.FlowAccumulation[i] = 1000
		m.BasinID[i] = 7
		if x <= 2 {
			m.FilledElevation[i] = 0.70
		} else {
			m.FilledElevation[i] = 0.72
		}
	}

	m.identifyLakes()
	if len(m.Lakes) != 2 {
		t.Fatalf(
			"adjacent depressions at different spill levels merged: lakes=%d",
			len(m.Lakes),
		)
	}
	for _, lake := range m.Lakes {
		if lake.CellCount != 2 {
			t.Fatalf("lake %d cell count=%d want=2", lake.ID, lake.CellCount)
		}
	}
}

func TestHydrologyRejectsGeometryDimensionMismatch(t *testing.T) {
	in := makeBorderOceanInput(5, 5, 0.7)
	in.Geometry = world.GridGeometry{Width: 6, Height: 5, SpanXBlocks: 640, SpanZBlocks: 512}
	if _, err := Generate(in, Config{}); err == nil {
		t.Fatal("expected mismatched hydrology geometry to be rejected")
	}
}
