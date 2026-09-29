package geomorph

import (
	"math"
	"testing"
)

func baseInput(width, height int) Input {
	n := width * height
	in := Input{
		Width:                   width,
		Height:                  height,
		LandMask:                make([]bool, n),
		Elevation:               make([]float64, n),
		Slope:                   make([]float64, n),
		ReliefIndex:             make([]float64, n),
		ValleyIncision:          make([]float64, n),
		FluvialPower:            make([]float64, n),
		TectonicSupport:         make([]float64, n),
		FlowTo:                  make([]int, n),
		FlowAccumulation:        make([]float64, n),
		RiverStrength:           make([]float64, n),
		PerennialRiverPotential: make([]float64, n),
		SeasonalRiverPotential:  make([]float64, n),
		WetlandPotential:        make([]float64, n),
		LakeID:                  make([]int, n),
		RiverThreshold:          1000,
		RollingPlainWeight:      make([]float64, n),
		HillWeight:              make([]float64, n),
		FoothillWeight:          make([]float64, n),
		MountainWeight:          make([]float64, n),
		PlateauWeight:           make([]float64, n),
		ValleyWeight:            make([]float64, n),
	}
	for i := 0; i < n; i++ {
		in.LandMask[i] = true
		in.Elevation[i] = 0.62
		in.Slope[i] = 0.01
		in.ReliefIndex[i] = 0.15
		in.FlowTo[i] = -1
		in.RollingPlainWeight[i] = 1
	}
	return in
}

func TestLargeLowGradientRiverCreatesWideFloodplain(t *testing.T) {
	in := baseInput(5, 3)
	i := 1*in.Width + 2
	in.RiverStrength[i] = 0.95
	in.FlowAccumulation[i] = 12000
	in.PerennialRiverPotential[i] = 0.85
	in.ValleyWeight[i] = 0.9
	in.WetlandPotential[i] = 0.35
	in.Slope[i] = 0.006
	in.ReliefIndex[i] = 0.12

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.FloodplainWeight[i] < 0.45 {
		t.Fatalf("large low-gradient river did not create floodplain: %.3f", m.FloodplainWeight[i])
	}
	if m.ValleyFloorWidth[i] < 0.35 {
		t.Fatalf("large low-gradient river did not create broad valley floor: %.3f", m.ValleyFloorWidth[i])
	}
	if m.CanyonWeight[i] > 0.15 {
		t.Fatalf("flat low-relief river became canyon-like: %.3f", m.CanyonWeight[i])
	}
}

func TestSteepHighReliefRiverCreatesCanyonAndNarrowsFloor(t *testing.T) {
	flat := baseInput(5, 3)
	steep := baseInput(5, 3)
	i := 1*flat.Width + 2

	for _, in := range []*Input{&flat, &steep} {
		in.RiverStrength[i] = 0.95
		in.FlowAccumulation[i] = 15000
		in.ValleyWeight[i] = 0.95
		in.ValleyIncision[i] = 0.012
		in.FluvialPower[i] = 0.85
	}

	flat.Slope[i] = 0.006
	flat.ReliefIndex[i] = 0.16

	steep.Slope[i] = 0.048
	steep.ReliefIndex[i] = 0.88
	steep.RollingPlainWeight[i] = 0
	steep.MountainWeight[i] = 0.85
	steep.FoothillWeight[i] = 0.15

	a, err := Generate(flat, Config{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(steep, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if !(b.CanyonWeight[i] > a.CanyonWeight[i]+0.35) {
		t.Fatalf("steep high-relief river did not become canyon-like: flat=%.3f steep=%.3f",
			a.CanyonWeight[i], b.CanyonWeight[i])
	}
	if !(b.ValleyFloorWidth[i] < a.ValleyFloorWidth[i]) {
		t.Fatalf("canyon did not narrow valley floor: flat=%.3f steep=%.3f",
			a.ValleyFloorWidth[i], b.ValleyFloorWidth[i])
	}
}

func TestSlopeBreakAtFoothillCreatesAlluvialFan(t *testing.T) {
	withDrop := baseInput(5, 3)
	noDrop := baseInput(5, 3)

	target := 1*withDrop.Width + 2
	upstream := 1*withDrop.Width + 1

	for _, in := range []*Input{&withDrop, &noDrop} {
		in.FlowTo[upstream] = target
		in.RiverStrength[target] = 0.82
		in.FlowAccumulation[target] = 7000
		in.ValleyWeight[target] = 0.65
		in.FoothillWeight[target] = 0.75
		in.RollingPlainWeight[target] = 0.25

		in.ReliefIndex[upstream] = 0.82
		in.FluvialPower[upstream] = 0.78
		in.TectonicSupport[upstream] = 0.55
	}

	withDrop.Slope[upstream] = 0.055
	withDrop.Slope[target] = 0.008

	noDrop.Slope[upstream] = 0.012
	noDrop.Slope[target] = 0.010

	a, err := Generate(withDrop, Config{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(noDrop, Config{})
	if err != nil {
		t.Fatal(err)
	}

	if a.SlopeDrop[target] < 0.5 {
		t.Fatalf("expected strong slope break, got %.3f", a.SlopeDrop[target])
	}
	if !(a.AlluvialFanWeight[target] > b.AlluvialFanWeight[target]+0.10) {
		t.Fatalf("slope break did not strengthen alluvial fan: drop=%.3f no-drop=%.3f",
			a.AlluvialFanWeight[target], b.AlluvialFanWeight[target])
	}
}

func TestNoRiverDoesNotInventFluvialLandforms(t *testing.T) {
	in := baseInput(8, 6)
	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	for i := range m.LandMask {
		if m.FloodplainWeight[i] != 0 ||
			m.AlluvialFanWeight[i] != 0 ||
			m.CanyonWeight[i] != 0 ||
			m.ValleyFloorWidth[i] != 0 {
			t.Fatalf("cell %d invented fluvial geomorphology: flood=%.3f fan=%.3f canyon=%.3f width=%.3f",
				i, m.FloodplainWeight[i], m.AlluvialFanWeight[i], m.CanyonWeight[i], m.ValleyFloorWidth[i])
		}
	}
}

func TestLakeSuppressesCanyonAndFan(t *testing.T) {
	in := baseInput(5, 3)
	i := 1*in.Width + 2
	in.RiverStrength[i] = 1
	in.FlowAccumulation[i] = 20000
	in.ValleyWeight[i] = 1
	in.ValleyIncision[i] = 0.015
	in.FluvialPower[i] = 1
	in.ReliefIndex[i] = 0.95
	in.Slope[i] = 0.05
	in.FoothillWeight[i] = 0.8
	in.RollingPlainWeight[i] = 0.2
	in.LakeID[i] = 1

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.CanyonWeight[i] != 0 {
		t.Fatalf("lake became canyon: %.3f", m.CanyonWeight[i])
	}
	if m.AlluvialFanWeight[i] > 0.12 {
		t.Fatalf("lake retained too much alluvial fan weight: %.3f", m.AlluvialFanWeight[i])
	}
}

func TestAlluvialPlainNeedsLowReliefLowSlopeDrainage(t *testing.T) {
	plain := baseInput(5, 3)
	rough := baseInput(5, 3)
	i := 1*plain.Width + 2

	for _, in := range []*Input{&plain, &rough} {
		in.FlowAccumulation[i] = 10000
		in.RiverStrength[i] = 0.7
		in.ValleyWeight[i] = 0.75
		in.WetlandPotential[i] = 0.5
		in.PerennialRiverPotential[i] = 0.7
	}

	plain.Slope[i] = 0.004
	plain.ReliefIndex[i] = 0.08

	rough.Slope[i] = 0.045
	rough.ReliefIndex[i] = 0.75
	rough.RollingPlainWeight[i] = 0.1
	rough.HillWeight[i] = 0.9

	a, err := Generate(plain, Config{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(rough, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !(a.AlluvialPlainWeight[i] > b.AlluvialPlainWeight[i]+0.25) {
		t.Fatalf("lowland drainage did not favor alluvial plain: lowland=%.3f rough=%.3f",
			a.AlluvialPlainWeight[i], b.AlluvialPlainWeight[i])
	}
}

func TestDeterministicAndBounded(t *testing.T) {
	in := baseInput(9, 7)
	for i := range in.LandMask {
		in.Slope[i] = float64((i*13)%17) / 400
		in.ReliefIndex[i] = float64((i*7)%19) / 19
		in.FlowAccumulation[i] = float64((i * 101) % 20000)
		in.RiverStrength[i] = float64((i*3)%11) / 10
		in.ValleyWeight[i] = float64((i*5)%13) / 12
		in.FluvialPower[i] = float64((i*17)%23) / 22
	}
	for i := 0; i+1 < len(in.FlowTo); i++ {
		in.FlowTo[i] = i + 1
	}

	a, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	fieldsA := [][]float64{
		a.FloodplainWeight,
		a.AlluvialPlainWeight,
		a.AlluvialFanWeight,
		a.CanyonWeight,
		a.ValleyFloorWidth,
		a.DepositionPotential,
	}
	fieldsB := [][]float64{
		b.FloodplainWeight,
		b.AlluvialPlainWeight,
		b.AlluvialFanWeight,
		b.CanyonWeight,
		b.ValleyFloorWidth,
		b.DepositionPotential,
	}

	for f := range fieldsA {
		for i := range fieldsA[f] {
			if fieldsA[f][i] != fieldsB[f][i] {
				t.Fatalf("non-deterministic field %d cell %d", f, i)
			}
			v := fieldsA[f][i]
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
				t.Fatalf("field %d cell %d out of range: %v", f, i, v)
			}
		}
	}
}
