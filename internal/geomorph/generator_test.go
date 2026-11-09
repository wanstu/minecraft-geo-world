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
		TotalErosion:            make([]float64, n),
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
		in.TotalErosion[i] = float64((i*11)%17) / 850
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
		a.TransportCapacity,
		a.SedimentAccommodation,
		a.SedimentCompetenceLoss,
	}
	fieldsB := [][]float64{
		b.FloodplainWeight,
		b.AlluvialPlainWeight,
		b.AlluvialFanWeight,
		b.CanyonWeight,
		b.ValleyFloorWidth,
		b.DepositionPotential,
		b.TransportCapacity,
		b.SedimentAccommodation,
		b.SedimentCompetenceLoss,
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

	rawA := [][]float64{
		a.SedimentProduction,
		a.SedimentFlux,
		a.SedimentDeposition,
		a.SedimentExport,
	}
	rawB := [][]float64{
		b.SedimentProduction,
		b.SedimentFlux,
		b.SedimentDeposition,
		b.SedimentExport,
	}
	for f := range rawA {
		for i := range rawA[f] {
			if rawA[f][i] != rawB[f][i] {
				t.Fatalf("non-deterministic sediment field %d cell %d", f, i)
			}
			v := rawA[f][i]
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
				t.Fatalf("sediment field %d cell %d invalid: %v", f, i, v)
			}
		}
	}
}

func TestSedimentBudgetDoesNotInventProductionWithoutActualErosion(t *testing.T) {
	in := baseInput(5, 3)
	i := 1*in.Width + 2
	in.ReliefIndex[i] = 0.95
	in.FluvialPower[i] = 1
	in.TectonicSupport[i] = 0.8
	in.RiverStrength[i] = 1
	in.FlowAccumulation[i] = 20000
	in.Slope[i] = 0.05

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	for i := range m.LandMask {
		if m.SedimentProduction[i] != 0 ||
			m.SedimentFlux[i] != 0 ||
			m.SedimentDeposition[i] != 0 ||
			m.SedimentExport[i] != 0 {
			t.Fatalf(
				"cell %d invented sediment without actual erosion: production=%g flux=%g deposition=%g export=%g",
				i,
				m.SedimentProduction[i],
				m.SedimentFlux[i],
				m.SedimentDeposition[i],
				m.SedimentExport[i],
			)
		}
	}
}

func TestSedimentBudgetRoutesAndConservesProxyMass(t *testing.T) {
	in := baseInput(9, 3)
	row := 1
	for n := 0; n < 7; n++ {
		i := row*in.Width + n + 1
		in.TotalErosion[i] = 0.004 + 0.004*float64(6-n)
		in.Slope[i] = 0.040 - 0.005*float64(n)
		in.ReliefIndex[i] = 0.65 - 0.07*float64(n)
		in.FluvialPower[i] = 0.90 - 0.08*float64(n)
		in.RiverStrength[i] = 0.90
		in.FlowAccumulation[i] = 2000 + 2500*float64(n)
		in.ValleyWeight[i] = 0.80
		in.RollingPlainWeight[i] = 0.25 + 0.10*float64(n)
		if n < 6 {
			in.FlowTo[i] = row*in.Width + n + 2
		}
	}

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	production := 0.0
	deposition := 0.0
	exported := 0.0
	for i := range m.LandMask {
		production += m.SedimentProduction[i]
		deposition += m.SedimentDeposition[i]
		exported += m.SedimentExport[i]
	}
	if production <= 0 || deposition <= 0 || exported <= 0 {
		t.Fatalf(
			"expected active routed budget: production=%g deposition=%g export=%g",
			production,
			deposition,
			exported,
		)
	}
	if delta := math.Abs(production - deposition - exported); delta > 1e-10 {
		t.Fatalf(
			"sediment proxy mass is not conserved: production=%g deposition=%g export=%g delta=%g",
			production,
			deposition,
			exported,
			delta,
		)
	}

	last := row*in.Width + 7
	if m.SedimentFlux[last] <= m.SedimentProduction[last] {
		t.Fatalf(
			"downstream cell did not receive upstream sediment: flux=%g local=%g",
			m.SedimentFlux[last],
			m.SedimentProduction[last],
		)
	}
}

func TestAlluvialFanCoreDepositsRoutedLoadAcrossMountainFrontSlopeBreak(t *testing.T) {
	in := baseInput(7, 3)
	row := 1
	upstream := row*in.Width + 2
	target := row*in.Width + 3
	outlet := row*in.Width + 4

	in.FlowTo[upstream] = target
	in.FlowTo[target] = outlet

	in.RollingPlainWeight[upstream] = 0
	in.MountainWeight[upstream] = 0.85
	in.FoothillWeight[upstream] = 0.15
	in.ReliefIndex[upstream] = 0.82
	in.Slope[upstream] = 0.055
	in.FluvialPower[upstream] = 0.20
	in.TectonicSupport[upstream] = 0.55
	in.RiverStrength[upstream] = 0.90
	in.FlowAccumulation[upstream] = 6000
	in.ValleyWeight[upstream] = 0.70
	in.TotalErosion[upstream] = 0.10

	in.FoothillWeight[target] = 0.75
	in.RollingPlainWeight[target] = 0.25
	in.ReliefIndex[target] = 0.16
	in.Slope[target] = 0.008
	in.FluvialPower[target] = 0.60
	in.RiverStrength[target] = 0.95
	in.FlowAccumulation[target] = 20000
	in.ValleyWeight[target] = 0.75
	in.PerennialRiverPotential[target] = 0.80

	in.RollingPlainWeight[outlet] = 0.80
	in.ReliefIndex[outlet] = 0.12
	in.Slope[outlet] = 0.006
	in.RiverStrength[outlet] = 0.95
	in.FlowAccumulation[outlet] = 24000
	in.ValleyWeight[outlet] = 0.80

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if m.AlluvialFanCore[target] < 0.10 {
		t.Fatalf(
			"expected a real mountain-front fan core, got %.3f",
			m.AlluvialFanCore[target],
		)
	}

	incoming := m.SedimentFlux[upstream]
	if incoming <= 0 {
		t.Fatal("expected routed sediment to reach fan core")
	}
	if m.SedimentDeposition[target] < incoming*0.04 {
		t.Fatalf(
			"fan-forming slope break failed to deposit routed load: fanCore=%.3f incoming=%g deposition=%g fraction=%.3f",
			m.AlluvialFanCore[target],
			incoming,
			m.SedimentDeposition[target],
			m.SedimentDeposition[target]/incoming,
		)
	}
}

func TestStableChannelDoesNotTaxIncomingLoadAtEveryCell(t *testing.T) {
	in := baseInput(45, 3)
	row := 1
	const start = 2
	const count = 40

	for n := 0; n < count; n++ {
		i := row*in.Width + start + n
		in.RollingPlainWeight[i] = 0
		in.ValleyWeight[i] = 0.10
		in.Slope[i] = 0.04
		in.ReliefIndex[i] = 0.40
		in.FluvialPower[i] = 0.80
		in.RiverStrength[i] = 0.90
		in.FlowAccumulation[i] = 6000 + 250*float64(n)
		if n+1 < count {
			in.FlowTo[i] = row*in.Width + start + n + 1
		}
	}
	source := row*in.Width + start
	outlet := row*in.Width + start + count - 1
	in.TotalErosion[source] = 0.10

	m, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}

	production := m.SedimentProduction[source]
	exported := m.SedimentExport[outlet]
	if production <= 0 {
		t.Fatal("expected source production")
	}
	if exported < production*0.70 {
		t.Fatalf(
			"stable channel repeatedly removed carried load without a depositional sink: production=%g outletExport=%g retained=%.3f",
			production,
			exported,
			exported/production,
		)
	}
}

func TestCapacityDropWithAccommodationDepositsIncomingLoad(t *testing.T) {
	build := func(drop bool) Input {
		in := baseInput(45, 3)
		row := 1
		const start = 2
		const count = 40
		const breakAt = 20
		for n := 0; n < count; n++ {
			i := row*in.Width + start + n
			in.RollingPlainWeight[i] = 0
			in.ValleyWeight[i] = 0.10
			in.Slope[i] = 0.04
			in.ReliefIndex[i] = 0.40
			in.FluvialPower[i] = 0.80
			in.RiverStrength[i] = 0.90
			in.FlowAccumulation[i] = 6000 + 250*float64(n)
			if drop && n >= breakAt {
				in.RollingPlainWeight[i] = 1
				in.ValleyWeight[i] = 0.90
				in.WetlandPotential[i] = 0.35
				in.PerennialRiverPotential[i] = 0.85
				in.Slope[i] = 0.005
				in.ReliefIndex[i] = 0.10
				in.FluvialPower[i] = 0.12
			}
			if n+1 < count {
				in.FlowTo[i] = row*in.Width + start + n + 1
			}
		}
		in.TotalErosion[row*in.Width+start] = 0.10
		return in
	}

	stable, err := Generate(build(false), Config{})
	if err != nil {
		t.Fatal(err)
	}
	drop, err := Generate(build(true), Config{})
	if err != nil {
		t.Fatal(err)
	}

	const row = 1
	const start = 2
	const breakAt = 20
	breakCell := row*45 + start + breakAt
	outlet := row*45 + start + 39
	source := row*45 + start

	if stable.SedimentDeposition[breakCell] > 1e-9 {
		t.Fatalf(
			"stable channel unexpectedly deposited incoming load at comparison cell: %g",
			stable.SedimentDeposition[breakCell],
		)
	}
	if drop.SedimentDeposition[breakCell] <
		drop.SedimentProduction[source]*0.10 {
		t.Fatalf(
			"capacity drop with accommodation did not deposit routed load: deposition=%g production=%g",
			drop.SedimentDeposition[breakCell],
			drop.SedimentProduction[source],
		)
	}
	if drop.SedimentExport[outlet] >= stable.SedimentExport[outlet] {
		t.Fatalf(
			"capacity drop did not reduce outlet load: stable=%g drop=%g",
			stable.SedimentExport[outlet],
			drop.SedimentExport[outlet],
		)
	}
}

func TestLakeTrapsRoutedSediment(t *testing.T) {
	build := func(lake bool) Input {
		in := baseInput(9, 3)
		row := 1
		for n := 0; n < 7; n++ {
			i := row*in.Width + n + 1
			in.TotalErosion[i] = 0.02
			in.Slope[i] = 0.025
			in.ReliefIndex[i] = 0.30
			in.FluvialPower[i] = 0.70
			in.RiverStrength[i] = 0.90
			in.FlowAccumulation[i] = 2500 + 3000*float64(n)
			in.ValleyWeight[i] = 0.85
			if n < 6 {
				in.FlowTo[i] = row*in.Width + n + 2
			}
		}
		if lake {
			in.LakeID[row*in.Width+6] = 1
		}
		return in
	}

	openModel, err := Generate(build(false), Config{})
	if err != nil {
		t.Fatal(err)
	}
	lakeModel, err := Generate(build(true), Config{})
	if err != nil {
		t.Fatal(err)
	}

	lakeCell := 1*9 + 6
	if lakeModel.SedimentDeposition[lakeCell] <=
		openModel.SedimentDeposition[lakeCell] {
		t.Fatalf(
			"lake did not increase sediment trapping: open=%g lake=%g",
			openModel.SedimentDeposition[lakeCell],
			lakeModel.SedimentDeposition[lakeCell],
		)
	}

	sumExport := func(m *Model) float64 {
		total := 0.0
		for _, v := range m.SedimentExport {
			total += v
		}
		return total
	}
	if sumExport(lakeModel) >= sumExport(openModel) {
		t.Fatalf(
			"lake did not reduce downstream sediment export: open=%g lake=%g",
			sumExport(openModel),
			sumExport(lakeModel),
		)
	}
}

func TestLongitudinalGradeSmoothsMatureAlluvialRiver(t *testing.T) {
	in := baseInput(11, 3)
	row := 1
	elevations := []float64{
		0.640, 0.635, 0.630, 0.625, 0.620,
		0.610, 0.595, 0.575, 0.550,
	}
	for n, elevation := range elevations {
		x := n + 1
		i := row*in.Width + x
		in.Elevation[i] = elevation
		in.Slope[i] = 0.006
		in.ReliefIndex[i] = 0.10
		in.RiverStrength[i] = 0.95
		in.FlowAccumulation[i] = 2000 + float64(n)*3000
		in.PerennialRiverPotential[i] = 0.9
		in.WetlandPotential[i] = 0.25
		in.ValleyWeight[i] = 0.95
		in.RollingPlainWeight[i] = 1
		if n+1 < len(elevations) {
			in.FlowTo[i] = row*in.Width + x + 1
		}
	}

	legacy, err := Generate(in, Config{})
	if err != nil {
		t.Fatal(err)
	}
	for i, correction := range legacy.LongitudinalGradeCorrection {
		if correction != 0 {
			t.Fatalf("legacy correction[%d]=%.12f want 0", i, correction)
		}
	}

	graded, err := Generate(in, Config{LongitudinalGradeVersion: 1})
	if err != nil {
		t.Fatal(err)
	}

	corrected := make([]float64, len(elevations))
	for n, elevation := range elevations {
		i := row*in.Width + n + 1
		corrected[n] = elevation + graded.LongitudinalGradeCorrection[i]
	}
	upperDrop := corrected[1] - corrected[3]
	lowerDrop := corrected[5] - corrected[7]
	originalUpperDrop := elevations[1] - elevations[3]
	originalLowerDrop := elevations[5] - elevations[7]

	if !(originalLowerDrop > originalUpperDrop*3) {
		t.Fatalf("test fixture is not downstream-steepening: upper=%.6f lower=%.6f",
			originalUpperDrop, originalLowerDrop)
	}
	if !(lowerDrop < originalLowerDrop-0.004) {
		t.Fatalf("grade correction did not reduce lower-reach steepening: before=%.6f after=%.6f profile=%v",
			originalLowerDrop, lowerDrop, corrected)
	}
	if !(upperDrop > originalUpperDrop) {
		t.Fatalf("grade correction did not redistribute drop upstream: before=%.6f after=%.6f profile=%v",
			originalUpperDrop, upperDrop, corrected)
	}
	for n := 1; n < len(corrected); n++ {
		if corrected[n] > corrected[n-1]+1e-9 {
			t.Fatalf("graded river rose downstream at %d: %.6f -> %.6f",
				n, corrected[n-1], corrected[n])
		}
	}
}

func TestLongitudinalGradeDoesNotFlattenMountainCanyon(t *testing.T) {
	in := baseInput(9, 3)
	row := 1
	for n := 0; n < 7; n++ {
		i := row*in.Width + n + 1
		in.Elevation[i] = 0.72 - float64(n)*0.018
		in.Slope[i] = 0.055
		in.ReliefIndex[i] = 0.9
		in.RiverStrength[i] = 1
		in.FlowAccumulation[i] = 3000 + float64(n)*4000
		in.FluvialPower[i] = 0.9
		in.ValleyIncision[i] = 0.02
		in.ValleyWeight[i] = 0.9
		in.RollingPlainWeight[i] = 0
		in.MountainWeight[i] = 0.85
		in.FoothillWeight[i] = 0.2
		if n < 6 {
			in.FlowTo[i] = row*in.Width + n + 2
		}
	}

	m, err := Generate(in, Config{LongitudinalGradeVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 7; n++ {
		i := row*in.Width + n + 1
		if math.Abs(m.LongitudinalGradeCorrection[i]) > 1e-12 {
			t.Fatalf("mountain/canyon river received grade correction at %d: %.12f",
				i, m.LongitudinalGradeCorrection[i])
		}
	}
}
