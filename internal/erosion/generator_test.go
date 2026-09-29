package erosion

import (
	"math"
	"testing"

	"geoworld/internal/climate"
	"geoworld/internal/hydrology"
	"geoworld/internal/phase1"
	"geoworld/internal/world"
)

func baseInput(width, height int) Input {
	n := width * height
	in := Input{
		Width:             width,
		Height:            height,
		LandMask:          make([]bool, n),
		Elevation:         make([]float64, n),
		CrustThickness:    make([]float64, n),
		TectonicIntensity: make([]float64, n),
		TectonicInfluence: make([]float64, n),
		AnnualPrecipMM:    make([]float64, n),
		FlowAccumulation:  make([]float64, n),
		RiverStrength:     make([]float64, n),
		LakeID:            make([]int, n),
		RiverThreshold:    1000,
	}
	for i := 0; i < n; i++ {
		in.LandMask[i] = true
		in.Elevation[i] = 0.70
		in.CrustThickness[i] = 0.60
		in.AnnualPrecipMM[i] = 1000
	}
	return in
}

func TestFlatTerrainWithoutRiverDoesNotErode(t *testing.T) {
	in := baseInput(8, 8)
	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	for i := range m.TotalErosion {
		if math.Abs(m.TotalErosion[i]) > 1e-12 {
			t.Fatalf("flat non-river terrain eroded at %d: %.8f", i, m.TotalErosion[i])
		}
	}
}

func TestStrongerRiverIncisesMore(t *testing.T) {
	in := baseInput(5, 5)
	for z := 0; z < 5; z++ {
		for x := 0; x < 5; x++ {
			i := z*5 + x
			in.Elevation[i] = 0.82 - float64(x)*0.035
		}
	}

	weak := 2*5 + 2
	strong := 2*5 + 3
	in.RiverStrength[weak] = 0.25
	in.FlowAccumulation[weak] = 2500
	in.RiverStrength[strong] = 0.95
	in.FlowAccumulation[strong] = 25000

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !(m.ValleyIncision[strong] > m.ValleyIncision[weak]) {
		t.Fatalf("strong river did not incise more: strong=%.5f weak=%.5f",
			m.ValleyIncision[strong], m.ValleyIncision[weak])
	}
}

func TestWetSlopeErodesMoreThanDrySlope(t *testing.T) {
	in := baseInput(6, 4)
	for z := 0; z < in.Height; z++ {
		for x := 0; x < in.Width; x++ {
			i := z*in.Width + x
			in.Elevation[i] = 0.80 - float64(x)*0.03
		}
	}
	dry := 1*in.Width + 2
	wet := 2*in.Width + 2
	in.AnnualPrecipMM[dry] = 200
	in.AnnualPrecipMM[wet] = 2200

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !(m.HillslopeErosion[wet] > m.HillslopeErosion[dry]) {
		t.Fatalf("wet slope did not erode more: wet=%.6f dry=%.6f",
			m.HillslopeErosion[wet], m.HillslopeErosion[dry])
	}
}

func TestTectonicSupportResistsHillslopeFlattening(t *testing.T) {
	in := baseInput(6, 4)
	for z := 0; z < in.Height; z++ {
		for x := 0; x < in.Width; x++ {
			i := z*in.Width + x
			in.Elevation[i] = 0.82 - float64(x)*0.035
			in.AnnualPrecipMM[i] = 1600
		}
	}
	stable := 1*in.Width + 2
	active := 2*in.Width + 2
	in.TectonicIntensity[stable] = 0.05
	in.TectonicIntensity[active] = 0.95

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !(m.HillslopeErosion[stable] > m.HillslopeErosion[active]) {
		t.Fatalf("tectonic support did not resist flattening: stable=%.6f active=%.6f",
			m.HillslopeErosion[stable], m.HillslopeErosion[active])
	}
	if !(m.TerrainMaturity[stable] > m.TerrainMaturity[active]) {
		t.Fatalf("stable terrain should have higher maturity proxy")
	}
}

func TestLakeSuppressesValleyIncision(t *testing.T) {
	in := baseInput(5, 3)
	for z := 0; z < in.Height; z++ {
		for x := 0; x < in.Width; x++ {
			i := z*in.Width + x
			in.Elevation[i] = 0.80 - float64(x)*0.04
		}
	}
	normal := 1*in.Width + 2
	lake := 1*in.Width + 3
	for _, i := range []int{normal, lake} {
		in.RiverStrength[i] = 0.9
		in.FlowAccumulation[i] = 15000
	}
	in.LakeID[lake] = 1

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !(m.ValleyIncision[lake] < m.ValleyIncision[normal]*0.25) {
		t.Fatalf("lake did not suppress valley incision: lake=%.6f normal=%.6f",
			m.ValleyIncision[lake], m.ValleyIncision[normal])
	}
}

func TestOceanUnchangedAndLandStaysAboveSeaLevel(t *testing.T) {
	in := baseInput(6, 4)
	for z := 0; z < in.Height; z++ {
		for x := 0; x < in.Width; x++ {
			i := z*in.Width + x
			if x == 0 {
				in.LandMask[i] = false
				in.Elevation[i] = 0.2
				continue
			}
			in.Elevation[i] = 0.5015 + float64(x)*0.005
			in.RiverStrength[i] = 1
			in.FlowAccumulation[i] = 100000
			in.AnnualPrecipMM[i] = 3000
		}
	}

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	for i := range m.ErodedElevation {
		if !in.LandMask[i] {
			if m.ErodedElevation[i] != in.Elevation[i] {
				t.Fatalf("ocean changed at %d: got %.6f want %.6f", i, m.ErodedElevation[i], in.Elevation[i])
			}
			continue
		}
		if m.ErodedElevation[i] < m.Config.SeaLevelNormalized+0.001-1e-12 {
			t.Fatalf("land eroded below phase4.1 floor at %d: %.6f", i, m.ErodedElevation[i])
		}
	}
}

func TestReliefMeasuresNeighborhoodRange(t *testing.T) {
	in := baseInput(7, 7)
	center := 3*in.Width + 3
	in.Elevation[center] = 0.90

	m, err := Generate(in, Config{ReliefRadiusBlocks: 2 * world.DefaultMacroSpacingBlocks})
	if err != nil {
		t.Fatal(err)
	}
	if m.LocalRelief[center] < 0.19 {
		t.Fatalf("expected local relief around isolated high point, got %.4f", m.LocalRelief[center])
	}
	if m.ReliefIndex[center] <= 0 {
		t.Fatalf("expected positive relief index")
	}
}

func TestSlopeAndReliefPhysicalScaleStableAcrossResolution(t *testing.T) {
	makeInput := func(width, height int, span float64) Input {
		in := baseInput(width, height)
		geometry := world.GridGeometry{Width: width, Height: height, SpanXBlocks: span, SpanZBlocks: span}
		in.Geometry = geometry
		for z := 0; z < height; z++ {
			for x := 0; x < width; x++ {
				i := z*width + x
				worldX, _ := geometry.NodePosition(x, z)
				in.Elevation[i] = 0.82 - 0.10*(worldX/span)
			}
		}
		return in
	}

	coarse, err := Generate(makeInput(17, 17, 2048), Config{})
	if err != nil {
		t.Fatal(err)
	}
	fine, err := Generate(makeInput(33, 33, 2048), Config{})
	if err != nil {
		t.Fatal(err)
	}

	ci := 8*coarse.Width + 8
	fi := 16*fine.Width + 16
	if math.Abs(coarse.Slope[ci]-fine.Slope[fi]) > 1e-12 {
		t.Fatalf("slope changed with resolution: coarse=%.12f fine=%.12f", coarse.Slope[ci], fine.Slope[fi])
	}
	if math.Abs(coarse.HillslopeErosion[ci]-fine.HillslopeErosion[fi]) > 1e-12 {
		t.Fatalf("hillslope erosion changed with resolution: coarse=%.12f fine=%.12f",
			coarse.HillslopeErosion[ci], fine.HillslopeErosion[fi])
	}
	if math.Abs(coarse.LocalRelief[ci]-fine.LocalRelief[fi]) > 1e-12 {
		t.Fatalf("local relief changed with resolution: coarse=%.12f fine=%.12f",
			coarse.LocalRelief[ci], fine.LocalRelief[fi])
	}
	if coarse.Config.ReliefRadiusBlocks != fine.Config.ReliefRadiusBlocks {
		t.Fatalf("default relief radius changed with resolution: coarse=%.1f fine=%.1f blocks",
			coarse.Config.ReliefRadiusBlocks, fine.Config.ReliefRadiusBlocks)
	}
	if coarse.Config.ReliefRadiusBlocks != world.DefaultMacroSpacingBlocks {
		t.Fatalf("unexpected default relief radius %.1f blocks", coarse.Config.ReliefRadiusBlocks)
	}
}

func TestExplicitReliefRadiusUsesPhysicalBlocksAcrossResolution(t *testing.T) {
	makeModel := func(width, height int, span float64) *Model {
		geometry := world.GridGeometry{Width: width, Height: height, SpanXBlocks: span, SpanZBlocks: span}
		n := width * height
		m := &Model{
			Config: Config{
				ReliefRadiusBlocks: 2 * world.DefaultMacroSpacingBlocks,
				ReliefScale:        0.16,
			},
			Geometry:        geometry,
			Width:           width,
			Height:          height,
			LandMask:        make([]bool, n),
			ErodedElevation: make([]float64, n),
			LocalRelief:     make([]float64, n),
			ReliefIndex:     make([]float64, n),
		}
		for i := range m.LandMask {
			m.LandMask[i] = true
			m.ErodedElevation[i] = 0.70
		}
		cx, cz := (width-1)/2, (height-1)/2
		m.ErodedElevation[cz*width+cx] = 0.90
		m.computeRelief()
		return m
	}

	coarse := makeModel(9, 9, 1024)
	fine := makeModel(17, 17, 1024)

	coarseInside := coarse.LocalRelief[4*coarse.Width+6] // world x=768, exactly +256 blocks.
	fineInside := fine.LocalRelief[8*fine.Width+12]
	if math.Abs(coarseInside-fineInside) > 1e-12 || coarseInside < 0.19 {
		t.Fatalf("relief radius differs at physical boundary: coarse=%.4f fine=%.4f", coarseInside, fineInside)
	}

	coarseOutside := coarse.LocalRelief[4*coarse.Width+7] // world x=896, +384 blocks.
	fineOutside := fine.LocalRelief[8*fine.Width+14]
	if math.Abs(coarseOutside-fineOutside) > 1e-12 || coarseOutside > 1e-12 {
		t.Fatalf("relief leaked beyond physical radius: coarse=%.4f fine=%.4f", coarseOutside, fineOutside)
	}
}

func TestErosionRejectsGeometryDimensionMismatch(t *testing.T) {
	in := baseInput(4, 4)
	in.Geometry = world.GridGeometry{Width: 5, Height: 4, SpanXBlocks: 512, SpanZBlocks: 384}
	if _, err := Generate(in, Config{}); err == nil {
		t.Fatal("expected mismatched erosion geometry to be rejected")
	}
}

func TestErosionSeedSweepStable(t *testing.T) {
	seeds := []int64{5000, 18994, 32988, 46982}

	for _, seed := range seeds {
		world := phase1.Generate(phase1.Config{
			Seed:          seed,
			Width:         160,
			Height:        100,
			PlateCount:    18,
			OceanFraction: 0.68,
		})
		c, err := climate.Generate(climate.InputFromPhase1(world), climate.Config{})
		if err != nil {
			t.Fatalf("seed %d climate: %v", seed, err)
		}
		h, err := hydrology.Generate(hydrology.InputFromWorld(world, c), hydrology.Config{})
		if err != nil {
			t.Fatalf("seed %d hydrology: %v", seed, err)
		}
		m, err := Generate(InputFromWorld(world, c, h), Config{})
		if err != nil {
			t.Fatalf("seed %d erosion: %v", seed, err)
		}

		landN := 0
		valleyN := 0
		sumErosion := 0.0
		maxErosion := 0.0
		strongN := 0

		for i, land := range m.LandMask {
			if !land {
				if m.ErodedElevation[i] != m.OriginalElevation[i] {
					t.Fatalf("seed %d ocean elevation changed at %d", seed, i)
				}
				continue
			}
			landN++
			e := m.TotalErosion[i]
			if math.IsNaN(e) || math.IsInf(e, 0) {
				t.Fatalf("seed %d invalid erosion at %d", seed, i)
			}
			if m.ErodedElevation[i] > m.OriginalElevation[i]+1e-12 {
				t.Fatalf("seed %d erosion raised terrain at %d", seed, i)
			}
			floor := m.Config.SeaLevelNormalized + 0.001
			if m.OriginalElevation[i] > floor {
				if m.ErodedElevation[i] < floor-1e-12 {
					t.Fatalf("seed %d land cut below phase4.1 floor at %d", seed, i)
				}
			} else if math.Abs(m.ErodedElevation[i]-m.OriginalElevation[i]) > 1e-12 {
				t.Fatalf("seed %d low coastal land was modified at %d: original=%.6f eroded=%.6f",
					seed, i, m.OriginalElevation[i], m.ErodedElevation[i])
			}
			sumErosion += e
			if e > maxErosion {
				maxErosion = e
			}
			if m.ValleyIncision[i] > 1e-4 {
				valleyN++
			}
			if e >= 0.015 {
				strongN++
			}
		}

		if landN == 0 {
			t.Fatalf("seed %d has no land", seed)
		}
		avg := sumErosion / float64(landN)
		if avg < 0.0002 || avg > 0.006 {
			t.Fatalf("seed %d average erosion suspicious: %.6f", seed, avg)
		}
		if maxErosion > 0.06 {
			t.Fatalf("seed %d maximum erosion suspicious: %.6f", seed, maxErosion)
		}
		valleyPct := float64(valleyN) / float64(landN)
		if valleyN == 0 || valleyPct > 0.08 {
			t.Fatalf("seed %d valley incision fraction suspicious: %.2f%%", seed, valleyPct*100)
		}
		strongPct := float64(strongN) / float64(landN)
		if strongPct > 0.03 {
			t.Fatalf("seed %d too much strong erosion: %.2f%%", seed, strongPct*100)
		}
	}
}
