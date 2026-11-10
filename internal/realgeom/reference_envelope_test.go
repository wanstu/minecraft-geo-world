package realgeom

import (
	"math"
	"testing"
)

func TestReferenceGridLocationAndBoundaries(t *testing.T) {
	g := ReferenceGrid{Columns: 4, Rows: 2, Bounds: Bounds{West: 0, South: 0, East: 4, North: 2}}
	if err := g.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		lon, lat float64
		index    int
		ok       bool
	}{
		{0.5, 1.5, 0, true}, {3.5, 1.5, 3, true}, {0.5, .5, 4, true},
		{3.5, .5, 7, true}, {4, 0, 7, true}, {-1, 1, 0, false}, {2, 3, 0, false},
	} {
		idx, ok := g.CellAt(tc.lon, tc.lat)
		if ok != tc.ok || (ok && idx != tc.index) {
			t.Errorf("CellAt(%g,%g)=%d,%v want %d,%v", tc.lon, tc.lat, idx, ok, tc.index, tc.ok)
		}
	}
	for row := 0; row < g.Rows; row++ {
		for col := 0; col < g.Columns; col++ {
			b := g.CellBounds(row, col)
			if math.Abs((b.East-b.West)-1) > 1e-12 || math.Abs((b.North-b.South)-1) > 1e-12 {
				t.Fatal(b)
			}
		}
	}
	if err := (&ReferenceGrid{Columns: 0, Rows: 10, Bounds: g.Bounds}).Validate(); err == nil {
		t.Fatal("invalid grid accepted")
	}
}

func TestEnvelopeBandsHandleZeroValuesAndMissingBlocks(t *testing.T) {
	cells := []ReferenceCell{
		{ValidSamples: 100, Metrics: map[string]float64{"water": 0}},
		{ValidSamples: 100, Metrics: map[string]float64{"water": 0.1}},
		{ValidSamples: 100, Metrics: map[string]float64{"water": 0.2}},
		{ValidSamples: 5, Metrics: map[string]float64{"water": 0.9}},
		{ValidSamples: 100, Metrics: map[string]float64{"wrong": 3}},
	}
	band := CreateEnvelopeBand("water", "fraction", cells, 32)
	if band.SampleBlocks != 3 || band.ExcludedBlocks != 2 {
		t.Fatalf("counts = %+v", band)
	}
	if math.Abs(band.Percentiles.P50-.1) > 1e-12 || math.Abs(band.Percentiles.P90-.18) > 1e-12 {
		t.Fatalf("unexpected percentiles: %+v", band.Percentiles)
	}
	if band.Status != "candidate_not_calibrated" {
		t.Fatal(band.Status)
	}
}

func TestEnvelopeSourceVersionAndTemporalSemantics(t *testing.T) {
	tests := []struct{ layer, version, dataset, period string }{
		{LayerJRCOccurrence, "1.5", "JRC_Global_Surface_Water", "1984-2024"},
		{LayerJRCSeasonality, "1.5", "JRC_Global_Surface_Water", "2024"},
		{LayerHydroDirection, "1.1", "HydroSHEDS_core", "static_v1.1"},
		{"hydrorivers", "1", "HydroRIVERS", "static_v1"},
	}
	for _, tt := range tests {
		ds, p, err := SourceDescription(tt.layer, tt.version)
		if err != nil || ds != tt.dataset || p != tt.period {
			t.Errorf("%s got %q/%q err=%v", tt.layer, ds, p, err)
		}
	}
	if _, _, err := SourceDescription(LayerJRCSeasonality, "1.4"); err == nil {
		t.Fatal("unlocked dataset version accepted")
	}
	if _, _, err := SourceDescription("not-a-real-layer", "1"); err == nil {
		t.Fatal("invalid layer accepted")
	}
}
