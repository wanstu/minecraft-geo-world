package terrain

import (
	"math"
	"testing"
)

func uniformInput(width, height int, base float64) Input {
	n := width * height
	in := Input{
		Width:              width,
		Height:             height,
		BaseElevation:      make([]float64, n),
		RollingPlainWeight: make([]float64, n),
		HillWeight:         make([]float64, n),
		FoothillWeight:     make([]float64, n),
		MountainWeight:     make([]float64, n),
		PlateauWeight:      make([]float64, n),
		ValleyWeight:       make([]float64, n),
	}
	for i := range in.BaseElevation {
		in.BaseElevation[i] = base
		in.RollingPlainWeight[i] = 1
	}
	return in
}

func TestTerrainDeterministicAtSameCoordinate(t *testing.T) {
	in := uniformInput(8, 6, 0.64)
	m, err := Generate(in, Config{Seed: 12345})
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range [][2]float64{
		{17.25, 31.5},
		{128, 256},
		{511.75, 321.125},
	} {
		a := m.SampleNormalized(p[0], p[1])
		b := m.SampleNormalized(p[0], p[1])
		if a != b {
			t.Fatalf("same world coordinate changed: %.12f vs %.12f", a, b)
		}
	}
}

func TestDifferentSeedsChangeRegionalDetail(t *testing.T) {
	in := uniformInput(8, 6, 0.64)
	a, err := Generate(in, Config{Seed: 111})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(in, Config{Seed: 222})
	if err != nil {
		t.Fatal(err)
	}

	different := false
	for _, p := range [][2]float64{
		{93.0, 71.0},
		{241.5, 199.25},
		{529.75, 401.5},
	} {
		if math.Abs(a.SampleNormalized(p[0], p[1])-b.SampleNormalized(p[0], p[1])) > 1e-8 {
			different = true
			break
		}
	}
	if !different {
		t.Fatal("different seeds did not change regional detail")
	}
}

func TestRegionalTerrainMacroBoundaryHeightAndSlopeAreContinuous(t *testing.T) {
	in := uniformInput(8, 7, 0.64)
	for z := 0; z < in.Height; z++ {
		for x := 0; x < in.Width; x++ {
			i := z*in.Width + x
			in.BaseElevation[i] = 0.57 + float64(x)*0.018 + float64(z)*0.009
			in.RollingPlainWeight[i] = 0.25
			in.HillWeight[i] = 0.45
			in.FoothillWeight[i] = 0.20
			in.MountainWeight[i] = 0.10
		}
	}

	m, err := Generate(in, Config{Seed: 731894205, CellSizeBlocks: 128})
	if err != nil {
		t.Fatal(err)
	}

	x := 3.0 * 128
	z := 2.41 * 128
	eps := 1e-4

	left := m.SampleNormalized(x-eps, z)
	at := m.SampleNormalized(x, z)
	right := m.SampleNormalized(x+eps, z)
	if math.Abs(left-at) > 1e-7 || math.Abs(right-at) > 1e-7 {
		t.Fatalf("height discontinuity at macro boundary: left=%.12f at=%.12f right=%.12f",
			left, at, right)
	}

	h := 0.05
	leftSlope := (m.SampleNormalized(x, z) - m.SampleNormalized(x-h, z)) / h
	rightSlope := (m.SampleNormalized(x+h, z) - m.SampleNormalized(x, z)) / h
	if math.Abs(leftSlope-rightSlope) > 2e-5 {
		t.Fatalf("slope discontinuity at macro boundary: left=%.9f right=%.9f",
			leftSlope, rightSlope)
	}
}

func TestRegionalTerrainChunkBoundariesHaveNoSeam(t *testing.T) {
	in := uniformInput(8, 6, 0.66)
	for i := range in.BaseElevation {
		in.RollingPlainWeight[i] = 0.25
		in.HillWeight[i] = 0.35
		in.FoothillWeight[i] = 0.20
		in.MountainWeight[i] = 0.20
	}

	m, err := Generate(in, Config{Seed: 77, CellSizeBlocks: 128})
	if err != nil {
		t.Fatal(err)
	}

	z := 271.375
	for x := 16.0; x <= 512; x += 16 {
		eps := 1e-5
		before := m.SampleNormalized(x-eps, z)
		at := m.SampleNormalized(x, z)
		after := m.SampleNormalized(x+eps, z)
		if math.Abs(before-at) > 1e-7 || math.Abs(after-at) > 1e-7 {
			t.Fatalf("chunk seam at x=%.0f: %.12f %.12f %.12f", x, before, at, after)
		}
	}
}

func TestTerrainDetailAmplitudeFollowsLandformWeight(t *testing.T) {
	plain := uniformInput(8, 8, 0.65)
	hill := uniformInput(8, 8, 0.65)
	mountain := uniformInput(8, 8, 0.65)
	plateau := uniformInput(8, 8, 0.65)

	for i := range plain.BaseElevation {
		hill.RollingPlainWeight[i] = 0
		hill.HillWeight[i] = 1

		mountain.RollingPlainWeight[i] = 0
		mountain.MountainWeight[i] = 1

		plateau.RollingPlainWeight[i] = 0
		plateau.PlateauWeight[i] = 1
	}

	cfg := Config{Seed: 2026, CellSizeBlocks: 128}
	mp, _ := Generate(plain, cfg)
	mh, _ := Generate(hill, cfg)
	mm, _ := Generate(mountain, cfg)
	mt, _ := Generate(plateau, cfg)

	plainStd := detailStdDev(mp)
	hillStd := detailStdDev(mh)
	mountainStd := detailStdDev(mm)
	plateauStd := detailStdDev(mt)

	if !(mountainStd > hillStd && hillStd > plainStd) {
		t.Fatalf("terrain detail amplitude order wrong: plain=%.6f hill=%.6f mountain=%.6f",
			plainStd, hillStd, mountainStd)
	}
	if !(plateauStd < hillStd) {
		t.Fatalf("plateau should remain flatter than hills: plateau=%.6f hill=%.6f",
			plateauStd, hillStd)
	}
}

func TestValleyOverlayLowersAndSmoothsTerrain(t *testing.T) {
	base := uniformInput(7, 7, 0.68)
	valley := uniformInput(7, 7, 0.68)
	for i := range valley.ValleyWeight {
		valley.ValleyWeight[i] = 1
	}

	cfg := Config{Seed: 991, CellSizeBlocks: 128}
	a, err := Generate(base, cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(valley, cfg)
	if err != nil {
		t.Fatal(err)
	}

	x, z := 333.25, 281.75
	sa := a.Sample(x, z)
	sb := b.Sample(x, z)
	if !(sb.FinalHeight < sa.FinalHeight) {
		t.Fatalf("valley did not lower terrain: base=%.6f valley=%.6f", sa.FinalHeight, sb.FinalHeight)
	}
	if sb.ValleyCarve <= 0 {
		t.Fatalf("valley carve not recorded")
	}
	if !(math.Abs(sb.RegionalDetail) <= math.Abs(sa.RegionalDetail)+0.01) {
		t.Fatalf("valley unexpectedly amplified regional detail: base=%.6f valley=%.6f",
			sa.RegionalDetail, sb.RegionalDetail)
	}
}

func TestSampledBaseWeightsRemainNormalizedInInteriorLand(t *testing.T) {
	in := uniformInput(7, 7, 0.64)
	for i := range in.BaseElevation {
		in.RollingPlainWeight[i] = 0.18
		in.HillWeight[i] = 0.22
		in.FoothillWeight[i] = 0.16
		in.MountainWeight[i] = 0.27
		in.PlateauWeight[i] = 0.17
	}

	m, err := Generate(in, Config{Seed: 42})
	if err != nil {
		t.Fatal(err)
	}

	for z := 160.0; z <= 560; z += 37 {
		for x := 160.0; x <= 560; x += 41 {
			s := m.Sample(x, z)
			sum := s.RollingPlainWeight + s.HillWeight + s.FoothillWeight + s.MountainWeight + s.PlateauWeight
			if math.Abs(sum-1) > 1e-10 {
				t.Fatalf("sampled base weights not normalized at %.1f,%.1f: %.12f", x, z, sum)
			}
		}
	}
}

func TestRegionalTerrainOutputFiniteAndBounded(t *testing.T) {
	in := uniformInput(8, 6, 0.64)
	for i := range in.BaseElevation {
		in.RollingPlainWeight[i] = 0.15
		in.HillWeight[i] = 0.20
		in.FoothillWeight[i] = 0.20
		in.MountainWeight[i] = 0.35
		in.PlateauWeight[i] = 0.10
		in.ValleyWeight[i] = float64(i%5) / 5
	}

	m, err := Generate(in, Config{Seed: -1234})
	if err != nil {
		t.Fatal(err)
	}

	for z := 0.0; z <= 5*128; z += 13.25 {
		for x := 0.0; x <= 7*128; x += 11.75 {
			v := m.SampleNormalized(x, z)
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
				t.Fatalf("invalid terrain height at %.2f,%.2f: %v", x, z, v)
			}
		}
	}
}

func detailStdDev(m *Model) float64 {
	values := make([]float64, 0, 300)
	for z := 96.0; z <= 700; z += 41 {
		for x := 112.0; x <= 700; x += 37 {
			values = append(values, m.Sample(x, z).RegionalDetail)
		}
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		d := v - mean
		variance += d * d
	}
	variance /= float64(len(values))
	return math.Sqrt(variance)
}

func TestBoundedAddAvoidsNewHardSaturation(t *testing.T) {
	for _, tc := range []struct {
		base  float64
		delta float64
	}{
		{0.98, 0.05},
		{0.995, 0.03},
		{0.02, -0.05},
		{0.005, -0.03},
	} {
		got := boundedAdd(tc.base, tc.delta)
		if got < 0 || got > 1 {
			t.Fatalf("boundedAdd escaped range: base=%.3f delta=%.3f got=%.9f", tc.base, tc.delta, got)
		}
		if tc.base < 1 && tc.delta > 0 && got >= 1 {
			t.Fatalf("positive detail hard-saturated: base=%.3f delta=%.3f got=%.12f", tc.base, tc.delta, got)
		}
		if tc.base > 0 && tc.delta < 0 && got <= 0 {
			t.Fatalf("negative detail hard-saturated: base=%.3f delta=%.3f got=%.12f", tc.base, tc.delta, got)
		}
	}
}
