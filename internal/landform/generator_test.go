package landform

import (
	"math"
	"testing"

	"geoworld/internal/climate"
	"geoworld/internal/erosion"
	"geoworld/internal/hydrology"
	"geoworld/internal/phase1"
	"geoworld/internal/world"
)

func syntheticInput(width, height int) Input {
	n := width * height
	in := Input{
		Width:           width,
		Height:          height,
		LandMask:        make([]bool, n),
		Elevation:       make([]float64, n),
		ReliefIndex:     make([]float64, n),
		Slope:           make([]float64, n),
		TerrainMaturity: make([]float64, n),
		TectonicSupport: make([]float64, n),
		CrustThickness:  make([]float64, n),
		ValleyIncision:  make([]float64, n),
		FluvialPower:    make([]float64, n),
	}
	for i := 0; i < n; i++ {
		in.LandMask[i] = true
		in.Elevation[i] = 0.62
		in.ReliefIndex[i] = 0.12
		in.Slope[i] = 0.01
		in.TerrainMaturity[i] = 0.8
		in.CrustThickness[i] = 0.6
	}
	return in
}

func baseWeightSum(m *Model, i int) float64 {
	return m.RollingPlainWeight[i] +
		m.HillWeight[i] +
		m.FoothillWeight[i] +
		m.MountainWeight[i] +
		m.PlateauWeight[i]
}

func TestBaseLandformWeightsNormalizeToOne(t *testing.T) {
	in := syntheticInput(8, 6)
	for i := range in.ReliefIndex {
		in.ReliefIndex[i] = float64(i%7) / 7
		in.Elevation[i] = 0.52 + float64(i%5)*0.07
	}
	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	for i, land := range m.LandMask {
		if !land {
			continue
		}
		sum := baseWeightSum(m, i)
		if math.Abs(sum-1) > 1e-12 {
			t.Fatalf("cell %d base weights sum %.12f, want 1", i, sum)
		}
	}
}

func TestLowReliefMatureTerrainPrefersRollingPlain(t *testing.T) {
	in := syntheticInput(3, 3)
	i := 4
	in.Elevation[i] = 0.58
	in.ReliefIndex[i] = 0.08
	in.TerrainMaturity[i] = 0.95

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.RollingPlainWeight[i] <= m.HillWeight[i] ||
		m.RollingPlainWeight[i] <= m.PlateauWeight[i] {
		t.Fatalf("low-relief mature terrain did not prefer rolling plain: plain=%.3f hill=%.3f plateau=%.3f",
			m.RollingPlainWeight[i], m.HillWeight[i], m.PlateauWeight[i])
	}
}

func TestMidReliefMatureTerrainPrefersHillOverPlain(t *testing.T) {
	in := syntheticInput(3, 3)
	i := 4
	in.Elevation[i] = 0.62
	in.ReliefIndex[i] = 0.42
	in.Slope[i] = 0.025
	in.TerrainMaturity[i] = 0.95

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !(m.HillWeight[i] > m.RollingPlainWeight[i]) {
		t.Fatalf("mid-relief mature terrain did not become hill-like: hill=%.3f plain=%.3f",
			m.HillWeight[i], m.RollingPlainWeight[i])
	}
}

func TestHighElevationLowReliefPrefersPlateauNotMountain(t *testing.T) {
	in := syntheticInput(3, 3)
	i := 4
	in.Elevation[i] = 0.86
	in.ReliefIndex[i] = 0.14
	in.Slope[i] = 0.008
	in.CrustThickness[i] = 0.82
	in.TectonicSupport[i] = 0.35

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !(m.PlateauWeight[i] > m.MountainWeight[i]) {
		t.Fatalf("high flat terrain should prefer plateau: plateau=%.3f mountain=%.3f",
			m.PlateauWeight[i], m.MountainWeight[i])
	}
}

func TestHighReliefTerrainPrefersMountain(t *testing.T) {
	in := syntheticInput(3, 3)
	i := 4
	in.Elevation[i] = 0.82
	in.ReliefIndex[i] = 0.92
	in.Slope[i] = 0.08
	in.TectonicSupport[i] = 0.9

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.MountainWeight[i] < 0.60 {
		t.Fatalf("high-relief terrain not mountain-dominant: %.3f", m.MountainWeight[i])
	}
}

func TestMountainCreatesFoothillInfluenceAroundIt(t *testing.T) {
	in := syntheticInput(9, 5)
	center := 2*in.Width + 4
	in.Elevation[center] = 0.88
	in.ReliefIndex[center] = 0.95
	in.Slope[center] = 0.09
	in.TectonicSupport[center] = 0.95

	neighbor := 2*in.Width + 6
	in.Elevation[neighbor] = 0.67
	in.ReliefIndex[neighbor] = 0.43
	in.Slope[neighbor] = 0.035
	in.TerrainMaturity[neighbor] = 0.65

	m, err := Generate(in, Config{MountainInfluenceBlocks: 4 * world.DefaultMacroSpacingBlocks})
	if err != nil {
		t.Fatal(err)
	}
	if m.MountainInfluence[neighbor] <= 0 {
		t.Fatal("mountain influence did not spread away from mountain core")
	}
	if m.FoothillWeight[neighbor] <= 0.05 {
		t.Fatalf("expected meaningful foothill weight near mountain, got %.3f", m.FoothillWeight[neighbor])
	}
}

func TestValleyIsOverlayAndDoesNotBreakBaseNormalization(t *testing.T) {
	in := syntheticInput(3, 3)
	i := 4
	in.ReliefIndex[i] = 0.35
	in.ValleyIncision[i] = 0.01
	in.FluvialPower[i] = 0.8

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.ValleyWeight[i] < 0.5 {
		t.Fatalf("strong incised river did not create valley overlay: %.3f", m.ValleyWeight[i])
	}
	if math.Abs(baseWeightSum(m, i)-1) > 1e-12 {
		t.Fatal("valley overlay changed base landform normalization")
	}
}

func TestOceanHasNoLandformWeights(t *testing.T) {
	in := syntheticInput(3, 3)
	in.LandMask[0] = false

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if baseWeightSum(m, 0) != 0 || m.ValleyWeight[0] != 0 {
		t.Fatalf("ocean received landform weights")
	}
}

func TestWeightResponseIsContinuousAcrossSmallReliefChange(t *testing.T) {
	a := syntheticInput(3, 3)
	b := syntheticInput(3, 3)
	i := 4
	a.ReliefIndex[i] = 0.399
	b.ReliefIndex[i] = 0.401

	ma, err := Generate(a, Config{})
	if err != nil {
		t.Fatal(err)
	}
	mb, err := Generate(b, Config{})
	if err != nil {
		t.Fatal(err)
	}

	diff := math.Abs(ma.HillWeight[i]-mb.HillWeight[i]) +
		math.Abs(ma.RollingPlainWeight[i]-mb.RollingPlainWeight[i]) +
		math.Abs(ma.FoothillWeight[i]-mb.FoothillWeight[i]) +
		math.Abs(ma.MountainWeight[i]-mb.MountainWeight[i]) +
		math.Abs(ma.PlateauWeight[i]-mb.PlateauWeight[i])

	if diff > 0.08 {
		t.Fatalf("small relief change caused abrupt landform weight jump: %.4f", diff)
	}
}

func TestMacroScaleDefaultsFollowPhysicalSpan(t *testing.T) {
	coarseIn := syntheticInput(25, 17)
	coarseIn.Geometry = world.GridGeometry{Width: 25, Height: 17, SpanXBlocks: 3072, SpanZBlocks: 2048}
	fineIn := syntheticInput(49, 33)
	fineIn.Geometry = world.GridGeometry{Width: 49, Height: 33, SpanXBlocks: 3072, SpanZBlocks: 2048}

	coarse, err := Generate(coarseIn, Config{})
	if err != nil {
		t.Fatal(err)
	}
	fine, err := Generate(fineIn, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if math.Abs(coarse.Config.MountainInfluenceBlocks-fine.Config.MountainInfluenceBlocks) > 1e-12 {
		t.Fatalf("mountain influence changed with resolution: coarse=%.2f fine=%.2f blocks",
			coarse.Config.MountainInfluenceBlocks, fine.Config.MountainInfluenceBlocks)
	}
	if math.Abs(coarse.Config.ValleyRadiusBlocks-fine.Config.ValleyRadiusBlocks) > 1e-12 {
		t.Fatalf("valley radius changed with resolution: coarse=%.2f fine=%.2f blocks",
			coarse.Config.ValleyRadiusBlocks, fine.Config.ValleyRadiusBlocks)
	}
	if math.Abs(coarse.Config.ValleyDecayBlocks-fine.Config.ValleyDecayBlocks) > 1e-12 {
		t.Fatalf("valley decay changed with resolution: coarse=%.2f fine=%.2f blocks",
			coarse.Config.ValleyDecayBlocks, fine.Config.ValleyDecayBlocks)
	}
}

func TestSameSeedLandformScaleDoesNotExplodeAtPreviewResolution(t *testing.T) {
	seed := int64(731894205)

	low := generateIntegratedLandform(t, seed, 240, 150)
	high := generateIntegratedLandform(t, seed, 640, 400)

	lowMountain := dominantFraction(low, 3)
	highMountain := dominantFraction(high, 3)
	if math.Abs(lowMountain-highMountain) > 0.12 {
		t.Fatalf("mountain share changes too much across macro resolutions: low=%.2f%% high=%.2f%%",
			lowMountain*100, highMountain*100)
	}
	if lowMountain > 0.35 || highMountain > 0.35 {
		t.Fatalf("mountain share exploded: low=%.2f%% high=%.2f%%",
			lowMountain*100, highMountain*100)
	}

	lowPlain := dominantFraction(low, 0)
	highPlain := dominantFraction(high, 0)
	if lowPlain < 0.30 || highPlain < 0.30 {
		t.Fatalf("plain share collapsed across macro resolutions: low=%.2f%% high=%.2f%%",
			lowPlain*100, highPlain*100)
	}

	if dominantFraction(high, 2) < 0.01 {
		t.Fatalf("formal-resolution world lost foothill transition")
	}
	if mixedTransitionFraction(high) < 0.05 {
		t.Fatalf("formal-resolution world has too little mixed transition: %.2f%%",
			mixedTransitionFraction(high)*100)
	}
}

func generateIntegratedLandform(t *testing.T, seed int64, width, height int) *Model {
	t.Helper()

	world := phase1.Generate(phase1.Config{
		Seed:          seed,
		Width:         width,
		Height:        height,
		PlateCount:    18,
		OceanFraction: 0.68,
	})
	c, err := climate.Generate(climate.InputFromPhase1(world), climate.Config{})
	if err != nil {
		t.Fatalf("climate: %v", err)
	}
	h, err := hydrology.Generate(hydrology.InputFromWorld(world, c), hydrology.Config{})
	if err != nil {
		t.Fatalf("hydrology: %v", err)
	}
	e, err := erosion.Generate(erosion.InputFromWorld(world, c, h), erosion.Config{})
	if err != nil {
		t.Fatalf("erosion: %v", err)
	}
	m, err := Generate(InputFromWorld(world, e), Config{})
	if err != nil {
		t.Fatalf("landform: %v", err)
	}
	return m
}

func dominantFraction(m *Model, category int) float64 {
	landN := 0
	count := 0
	for i, land := range m.LandMask {
		if !land {
			continue
		}
		landN++
		weights := [5]float64{
			m.RollingPlainWeight[i],
			m.HillWeight[i],
			m.FoothillWeight[i],
			m.MountainWeight[i],
			m.PlateauWeight[i],
		}
		best := 0
		for j := 1; j < len(weights); j++ {
			if weights[j] > weights[best] {
				best = j
			}
		}
		if best == category {
			count++
		}
	}
	if landN == 0 {
		return 0
	}
	return float64(count) / float64(landN)
}

func mixedTransitionFraction(m *Model) float64 {
	landN := 0
	mixed := 0
	for i, land := range m.LandMask {
		if !land {
			continue
		}
		landN++
		weights := [5]float64{
			m.RollingPlainWeight[i],
			m.HillWeight[i],
			m.FoothillWeight[i],
			m.MountainWeight[i],
			m.PlateauWeight[i],
		}
		best := 0
		for j := 1; j < len(weights); j++ {
			if weights[j] > weights[best] {
				best = j
			}
		}
		second := 0.0
		for j, w := range weights {
			if j != best && w > second {
				second = w
			}
		}
		if weights[best]-second < 0.20 {
			mixed++
		}
	}
	if landN == 0 {
		return 0
	}
	return float64(mixed) / float64(landN)
}
