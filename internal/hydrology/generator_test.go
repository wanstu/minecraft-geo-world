package hydrology

import (
	"math"
	"testing"

	"geoworld/internal/climate"
	"geoworld/internal/phase1"
)

func makeBorderOceanInput(width, height int, landElevation float64) Input {
	n := width * height
	in := Input{
		Width:               width,
		Height:              height,
		Elevation:           make([]float64, n),
		LandMask:            make([]bool, n),
		AnnualPrecipMM:      make([]float64, n),
		RainfallSeasonality: make([]float64, n),
	}
	for z := 0; z < height; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			border := x == 0 || z == 0 || x == width-1 || z == height-1
			if border {
				in.Elevation[i] = 0.2
				continue
			}
			in.LandMask[i] = true
			in.Elevation[i] = landElevation
			in.AnnualPrecipMM[i] = 1200
			in.RainfallSeasonality[i] = 0.25
		}
	}
	return in
}

func TestDepressionFillRaisesClosedPit(t *testing.T) {
	in := makeBorderOceanInput(7, 7, 0.72)
	center := 3*in.Width + 3
	in.Elevation[center] = 0.54

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if m.FillDepth[center] <= 0.10 {
		t.Fatalf("closed pit was not meaningfully filled: original=%.3f filled=%.3f depth=%.3f",
			in.Elevation[center], m.FilledElevation[center], m.FillDepth[center])
	}
	if m.FilledElevation[center] <= in.Elevation[center] {
		t.Fatal("filled elevation must exceed depression floor")
	}
	if m.LakePotential[center] <= 0 {
		t.Fatalf("filled depression should have lake potential, got %.3f", m.LakePotential[center])
	}
}

func TestEveryLandCellEventuallyDrainsToOcean(t *testing.T) {
	in := makeBorderOceanInput(18, 12, 0.62)
	for z := 1; z < in.Height-1; z++ {
		for x := 1; x < in.Width-1; x++ {
			i := z*in.Width + x
			in.Elevation[i] += float64((x*17+z*11)%13) * 0.004
		}
	}

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	for start := range m.FlowTo {
		if !m.LandMask[start] {
			continue
		}
		cur := start
		seen := make(map[int]struct{})
		for steps := 0; steps <= len(m.FlowTo); steps++ {
			if cur < 0 || cur >= len(m.FlowTo) {
				break
			}
			if !m.LandMask[cur] {
				cur = -1
				break
			}
			if _, ok := seen[cur]; ok {
				t.Fatalf("flow cycle from cell %d at %d", start, cur)
			}
			seen[cur] = struct{}{}
			next := m.FlowTo[cur]
			if next < 0 || !m.LandMask[next] {
				cur = -1
				break
			}
			cur = next
		}
		if cur != -1 {
			t.Fatalf("cell %d did not drain to ocean", start)
		}
	}
}

func TestFlowAccumulationGrowsDownstream(t *testing.T) {
	width, height := 7, 3
	n := width * height
	in := Input{
		Width:               width,
		Height:              height,
		Elevation:           make([]float64, n),
		LandMask:            make([]bool, n),
		AnnualPrecipMM:      make([]float64, n),
		RainfallSeasonality: make([]float64, n),
	}
	for z := 0; z < height; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			if x == width-1 {
				in.Elevation[i] = 0.2
				continue
			}
			in.LandMask[i] = true
			in.Elevation[i] = 0.90 - float64(x)*0.06
			in.AnnualPrecipMM[i] = 1400
			in.RainfallSeasonality[i] = 0.2
		}
	}

	m, err := Generate(in, Config{
		MinRiverAccumulation: 100,
		RiverQuantile:        0.80,
	})
	if err != nil {
		t.Fatal(err)
	}

	z := 1
	up := m.FlowAccumulation[z*width+0]
	down := m.FlowAccumulation[z*width+5]
	if !(down > up) {
		t.Fatalf("downstream accumulation should grow: upstream=%.1f downstream=%.1f", up, down)
	}
	if m.RiverStrength[z*width+5] <= 0 {
		t.Fatalf("wet downstream cell should become river, strength=%.3f threshold=%.1f accumulation=%.1f",
			m.RiverStrength[z*width+5], m.RiverThreshold, down)
	}
}

func TestDryLandscapeDoesNotInventRiver(t *testing.T) {
	in := makeBorderOceanInput(12, 8, 0.65)
	for i := range in.AnnualPrecipMM {
		in.AnnualPrecipMM[i] = 100
	}

	m, err := Generate(in, Config{
		RunoffLossMM:         180,
		MinRiverAccumulation: 10,
		RiverQuantile:        0.80,
	})
	if err != nil {
		t.Fatal(err)
	}

	for i := range m.RiverStrength {
		if m.RiverStrength[i] != 0 {
			t.Fatalf("dry landscape invented river at %d: %.4f", i, m.RiverStrength[i])
		}
		if m.FlowAccumulation[i] != 0 {
			t.Fatalf("dry landscape has runoff accumulation at %d: %.4f", i, m.FlowAccumulation[i])
		}
	}
}

func TestBasinsAssignedOnlyOnLand(t *testing.T) {
	in := makeBorderOceanInput(14, 10, 0.66)
	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.BasinCount <= 0 {
		t.Fatal("expected at least one basin")
	}
	for i, land := range m.LandMask {
		if land && m.BasinID[i] <= 0 {
			t.Fatalf("land cell %d missing basin", i)
		}
		if !land && m.BasinID[i] != 0 {
			t.Fatalf("ocean cell %d has basin id %d", i, m.BasinID[i])
		}
	}
}

func TestHydrologyDeterministic(t *testing.T) {
	in := makeBorderOceanInput(20, 14, 0.63)
	for z := 1; z < in.Height-1; z++ {
		for x := 1; x < in.Width-1; x++ {
			i := z*in.Width + x
			in.Elevation[i] += math.Sin(float64(x)*0.7+float64(z)*0.4) * 0.04
			in.AnnualPrecipMM[i] = 300 + float64((x*31+z*19)%1500)
			in.RainfallSeasonality[i] = float64((x+z)%10) / 10
		}
	}

	a, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	for i := range a.FlowTo {
		if a.FlowTo[i] != b.FlowTo[i] ||
			a.BasinID[i] != b.BasinID[i] ||
			math.Abs(a.FlowAccumulation[i]-b.FlowAccumulation[i]) > 1e-12 ||
			math.Abs(a.RiverStrength[i]-b.RiverStrength[i]) > 1e-12 {
			t.Fatalf("non-deterministic hydrology at %d", i)
		}
	}
}

func TestHydrologySeedSweepReasonable(t *testing.T) {
	seeds := []int64{4000, 18426, 32852, 54491}
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
		m, err := Generate(InputFromWorld(world, c), Config{})
		if err != nil {
			t.Fatalf("seed %d hydrology: %v", seed, err)
		}

		landN, riverN, fillN, lakeN, wetlandN := 0, 0, 0, 0, 0
		maxFill := 0.0
		for i := range m.LandMask {
			if !m.LandMask[i] {
				continue
			}
			landN++
			if m.RiverStrength[i] > 0 {
				riverN++
			}
			if m.FillDepth[i] > 1e-6 {
				fillN++
				if m.FillDepth[i] > maxFill {
					maxFill = m.FillDepth[i]
				}
			}
			if m.LakePotential[i] >= 0.35 {
				lakeN++
			}
			if m.WetlandPotential[i] >= 0.45 {
				wetlandN++
			}
			if math.IsNaN(m.FlowAccumulation[i]) || math.IsInf(m.FlowAccumulation[i], 0) {
				t.Fatalf("seed %d invalid accumulation at %d", seed, i)
			}
		}
		if landN == 0 {
			t.Fatalf("seed %d has no land", seed)
		}

		riverPct := float64(riverN) / float64(landN)
		fillPct := float64(fillN) / float64(landN)
		lakePct := float64(lakeN) / float64(landN)
		wetlandPct := float64(wetlandN) / float64(landN)

		if riverPct < 0.005 || riverPct > 0.06 {
			t.Fatalf("seed %d river fraction suspicious: %.2f%%", seed, riverPct*100)
		}
		if fillPct > 0.20 {
			t.Fatalf("seed %d too much filled terrain: %.2f%%", seed, fillPct*100)
		}
		if lakePct > 0.12 {
			t.Fatalf("seed %d too many lake candidates: %.2f%%", seed, lakePct*100)
		}
		if wetlandPct > 0.12 {
			t.Fatalf("seed %d too many wetland candidates: %.2f%%", seed, wetlandPct*100)
		}
		if maxFill > 0.25 {
			t.Fatalf("seed %d fill depth suspicious: %.3f", seed, maxFill)
		}
		if m.MajorBasinCount < 2 || m.MajorBasinCount > 120 {
			t.Fatalf("seed %d major basin count suspicious: %d", seed, m.MajorBasinCount)
		}
	}
}

func TestStrahlerRiverOrder(t *testing.T) {
	m := &Model{
		Width:  7,
		Height: 1,
		LandMask: []bool{
			true, true, true, true, true, true, true,
		},
		FilledElevation: []float64{1.0, 1.0, 0.8, 1.0, 1.0, 0.8, 0.5},
		FlowTo:          []int{2, 2, 6, 5, 5, 6, -1},
		RiverStrength:   []float64{1, 1, 1, 1, 1, 1, 1},
		RiverOrder:      make([]uint8, 7),
	}
	m.computeRiverOrder()

	want := []uint8{1, 1, 2, 1, 1, 2, 3}
	for i, w := range want {
		if m.RiverOrder[i] != w {
			t.Fatalf("river order at %d: got %d want %d; all=%v", i, m.RiverOrder[i], w, m.RiverOrder)
		}
	}
}

func TestExplicitLakeHasOutletAndMetadata(t *testing.T) {
	in := makeBorderOceanInput(7, 7, 0.72)
	center := 3*in.Width + 3
	in.Elevation[center] = 0.54

	m, err := Generate(in, Config{
		LakePotentialThreshold: 0.05,
		MinLakeCells:           1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.LakeID[center] <= 0 {
		t.Fatalf("center depression did not become explicit lake: potential=%.3f depth=%.3f",
			m.LakePotential[center], m.FillDepth[center])
	}

	id := m.LakeID[center]
	var lake *Lake
	for i := range m.Lakes {
		if m.Lakes[i].ID == id {
			lake = &m.Lakes[i]
			break
		}
	}
	if lake == nil {
		t.Fatalf("lake metadata missing for id %d", id)
	}
	if lake.CellCount < 1 || lake.MaxDepth <= 0 {
		t.Fatalf("invalid lake metadata: %+v", *lake)
	}
	if lake.SpillOutlet < 0 {
		t.Fatalf("lake has no outlet: %+v", *lake)
	}
	to := m.FlowTo[lake.SpillOutlet]
	if to >= 0 && to < len(m.LakeID) && m.LakeID[to] == lake.ID {
		t.Fatalf("lake outlet still drains inside same lake: outlet=%d to=%d", lake.SpillOutlet, to)
	}
}

func seasonalSlopeInput(warmMM, coldMM float64) Input {
	width, height := 8, 3
	n := width * height
	in := Input{
		Width:               width,
		Height:              height,
		Elevation:           make([]float64, n),
		LandMask:            make([]bool, n),
		AnnualPrecipMM:      make([]float64, n),
		RainfallSeasonality: make([]float64, n),
		WarmSeasonPrecipMM:  make([]float64, n),
		ColdSeasonPrecipMM:  make([]float64, n),
	}
	for z := 0; z < height; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			if x == width-1 {
				in.Elevation[i] = 0.2
				continue
			}
			in.LandMask[i] = true
			in.Elevation[i] = 0.92 - float64(x)*0.055
			in.WarmSeasonPrecipMM[i] = warmMM
			in.ColdSeasonPrecipMM[i] = coldMM
			in.AnnualPrecipMM[i] = warmMM + coldMM
			sum := warmMM + coldMM
			if sum > 0 {
				in.RainfallSeasonality[i] = math.Abs(warmMM-coldMM) / sum
			}
		}
	}
	return in
}

func TestSeasonalRiverPotentialRespondsToDrySeasonFlow(t *testing.T) {
	seasonal, err := Generate(seasonalSlopeInput(1100, 120), Config{
		MinRiverAccumulation: 100,
		RiverQuantile:        0.80,
	})
	if err != nil {
		t.Fatal(err)
	}
	perennial, err := Generate(seasonalSlopeInput(610, 610), Config{
		MinRiverAccumulation: 100,
		RiverQuantile:        0.80,
	})
	if err != nil {
		t.Fatal(err)
	}

	i := 1*seasonal.Width + 6
	if seasonal.RiverStrength[i] <= 0 {
		t.Fatalf("seasonal test downstream cell is not river")
	}
	if !(seasonal.SeasonalRiverPotential[i] > seasonal.PerennialRiverPotential[i]) {
		t.Fatalf("strongly seasonal climate should favor seasonal river: seasonal=%.3f perennial=%.3f",
			seasonal.SeasonalRiverPotential[i], seasonal.PerennialRiverPotential[i])
	}
	if !(perennial.PerennialRiverPotential[i] > perennial.SeasonalRiverPotential[i]) {
		t.Fatalf("balanced climate should favor perennial river: perennial=%.3f seasonal=%.3f",
			perennial.PerennialRiverPotential[i], perennial.SeasonalRiverPotential[i])
	}
}

func TestExplicitLakeIDsAreContiguous(t *testing.T) {
	in := makeBorderOceanInput(11, 9, 0.75)
	for _, p := range [][2]int{{3, 4}, {7, 4}} {
		x, z := p[0], p[1]
		in.Elevation[z*in.Width+x] = 0.55
	}
	m, err := Generate(in, Config{
		LakePotentialThreshold: 0.05,
		MinLakeCells:           1,
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, lake := range m.Lakes {
		want := i + 1
		if lake.ID != want {
			t.Fatalf("lake ids not contiguous: index=%d id=%d", i, lake.ID)
		}
	}
}

func TestPhase34RiverLakeSeedSweep(t *testing.T) {
	seeds := []int64{4000, 18426, 32852, 47278}

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
		m, err := Generate(InputFromWorld(world, c), Config{})
		if err != nil {
			t.Fatalf("seed %d hydrology: %v", seed, err)
		}

		riverN := 0
		seasonalDominant := 0
		maxOrder := uint8(0)
		largestLake := 0

		for i := range m.RiverStrength {
			if m.RiverStrength[i] <= 0 {
				continue
			}
			riverN++
			if m.RiverOrder[i] > maxOrder {
				maxOrder = m.RiverOrder[i]
			}
			if m.SeasonalRiverPotential[i] > m.PerennialRiverPotential[i] {
				seasonalDominant++
			}
			if m.RiverOrder[i] == 0 {
				t.Fatalf("seed %d river cell %d has zero Strahler order", seed, i)
			}
		}

		for _, lake := range m.Lakes {
			if lake.CellCount > largestLake {
				largestLake = lake.CellCount
			}
			if lake.SpillOutlet < 0 || lake.SpillOutlet >= len(m.LakeID) {
				t.Fatalf("seed %d lake %d has invalid spill outlet %d", seed, lake.ID, lake.SpillOutlet)
			}
			to := m.FlowTo[lake.SpillOutlet]
			if to >= 0 && to < len(m.LakeID) && m.LakeID[to] == lake.ID {
				t.Fatalf("seed %d lake %d spill outlet remains inside lake", seed, lake.ID)
			}
		}

		if riverN == 0 {
			t.Fatalf("seed %d has no river cells", seed)
		}
		if maxOrder < 1 || maxOrder > 8 {
			t.Fatalf("seed %d max river order suspicious: %d", seed, maxOrder)
		}

		seasonalPct := float64(seasonalDominant) / float64(riverN)
		if seasonalPct < 0.03 || seasonalPct > 0.75 {
			t.Fatalf("seed %d seasonal-dominant river fraction suspicious: %.1f%%", seed, seasonalPct*100)
		}

		if len(m.Lakes) < 1 || len(m.Lakes) > 30 {
			t.Fatalf("seed %d explicit lake count suspicious: %d", seed, len(m.Lakes))
		}
		if largestLake > 500 {
			t.Fatalf("seed %d largest lake too large for 160x100 audit grid: %d cells", seed, largestLake)
		}
	}
}
