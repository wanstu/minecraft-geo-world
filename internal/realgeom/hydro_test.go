package realgeom

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestLockedRegistryAndPixelCenterCrop(t *testing.T) {
	b, err := LoadRegionBounds("../../docs/realistic_geomorphology_reference_registry.json",
		"REF-04-zambezi-barotse-floodplain")
	if err != nil {
		t.Fatal(err)
	}
	if b.West != 22 || b.South != -17.5 || b.East != 24.8 || b.North != -14 {
		t.Fatalf("unexpected REF-04 bbox: %+v", b)
	}
	source := Bounds{West: 0, South: 0, East: 4, North: 4}
	crop := Bounds{West: 1, South: 1, East: 3, North: 3}
	w, err := WindowForBounds(source, 4, 4, &crop)
	if err != nil || w != (PixelWindow{X: 1, Y: 1, Width: 2, Height: 2}) {
		t.Fatalf("crop=%+v err=%v", w, err)
	}
	all, err := WindowForBounds(source, 4, 4, nil)
	if err != nil || all.Width != 4 || all.Height != 4 {
		t.Fatalf("full=%+v err=%v", all, err)
	}
	outside := Bounds{West: 5, South: 5, East: 6, North: 6}
	if _, err := WindowForBounds(source, 4, 4, &outside); err == nil {
		t.Fatal("expected no-overlap failure")
	}
	if _, err := LoadRegionBounds("../../docs/realistic_geomorphology_reference_registry.json", "MISSING"); err == nil {
		t.Fatal("unknown region should fail closed")
	}
}

func TestHistogramAndHydroSemantics(t *testing.T) {
	p := histogramPercentiles(map[int]int{0: 4, 50: 1, 100: 5}, 10)
	if p.P10 != 0 || p.P50 != 75 || p.P90 != 100 || p.Min != 0 || p.Max != 100 {
		t.Fatalf("histogram quantiles: %+v", p)
	}
	for _, name := range []string{LayerJRCOccurrence, LayerJRCSeasonality, LayerHydroDirection, LayerHydroAccumulation} {
		if !SupportedHydroRasterLayer(name) {
			t.Fatalf("not supported: %s", name)
		}
		if _, err := DefaultLayerNoData(name); err != nil {
			t.Fatal(err)
		}
	}
	if SupportedHydroRasterLayer("unknown") {
		t.Fatal("unsupported layer accepted")
	}
	if v, _ := DefaultLayerNoData(LayerHydroAccumulation); v != 4294967295 {
		t.Fatalf("acc nodata=%g", v)
	}
}

func TestSegmentClippingAndDistance(t *testing.T) {
	b := Bounds{West: 0, South: 0, East: 1, North: 1}
	x0, y0, x1, y1, ok := clipLine(-1, .5, 2, .5, b)
	if !ok || x0 != 0 || y0 != .5 || x1 != 1 || y1 != .5 {
		t.Fatalf("clip=%g %g %g %g %v", x0, y0, x1, y1, ok)
	}
	if _, _, _, _, ok := clipLine(-2, -2, -1, -1, b); ok {
		t.Fatal("outside segment was retained")
	}
	dist := haversineKm(x0, y0, x1, y1)
	if math.Abs(dist-111.2) > 1 {
		t.Fatalf("distance=%g", dist)
	}
}

func TestHydroRIVERSClipsPolylineNotFeatureLength(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "line.shp")
	dbf := filepath.Join(dir, "line.dbf")
	shp := make([]byte, 100+8+80)
	binary.BigEndian.PutUint32(shp[0:4], 9994)
	binary.BigEndian.PutUint32(shp[24:28], uint32(len(shp)/2))
	binary.LittleEndian.PutUint32(shp[28:32], 1000)
	binary.LittleEndian.PutUint32(shp[32:36], 3)
	for i, v := range []float64{-1, .5, 2, .5} {
		binary.LittleEndian.PutUint64(shp[36+i*8:], math.Float64bits(v))
	}
	record := shp[100:]
	binary.BigEndian.PutUint32(record[:4], 1)
	binary.BigEndian.PutUint32(record[4:8], 40)
	content := record[8:]
	binary.LittleEndian.PutUint32(content[:4], 3)
	for i, v := range []float64{-1, .5, 2, .5} {
		binary.LittleEndian.PutUint64(content[4+i*8:], math.Float64bits(v))
	}
	binary.LittleEndian.PutUint32(content[36:40], 1) // parts
	binary.LittleEndian.PutUint32(content[40:44], 2) // points
	binary.LittleEndian.PutUint32(content[44:48], 0)
	for i, v := range []float64{-1, .5, 2, .5} {
		binary.LittleEndian.PutUint64(content[48+i*8:], math.Float64bits(v))
	}
	if err := os.WriteFile(path, shp, 0o644); err != nil {
		t.Fatal(err)
	}
	const recordLen = 28
	const headerLen = 32 + 3*32 + 1
	d := make([]byte, headerLen+recordLen+1)
	d[0] = 3
	binary.LittleEndian.PutUint32(d[4:8], 1)
	binary.LittleEndian.PutUint16(d[8:10], headerLen)
	binary.LittleEndian.PutUint16(d[10:12], recordLen)
	for i, name := range []string{"HYRIV_ID", "NEXT_DOWN", "ORD_STRA"} {
		start := 32 + i*32
		copy(d[start:start+11], name)
		d[start+11] = 'N'
		d[start+16] = 9
	}
	d[headerLen-1] = 13
	d[headerLen] = ' '
	for i, v := range []int{10, 0, 2} {
		copy(d[headerLen+1+i*9:headerLen+1+(i+1)*9], fmt.Sprintf("%9d", v))
	}
	d[len(d)-1] = 26
	if err := os.WriteFile(dbf, d, 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := ExtractHydroRIVERS(path, "ref", "1", Bounds{West: 0, South: 0, East: 1, North: 1})
	if err != nil {
		t.Fatal(err)
	}
	if report.IntersectingReaches != 1 || report.ClippedSegments != 1 || report.LocalConfluences != 0 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if math.Abs(report.ClippedLengthKM-111.2) > 1 {
		t.Fatalf("clipped length %g should reflect only bbox width, not 3-degree original", report.ClippedLengthKM)
	}
	grid := ReferenceGrid{Columns: 2, Rows: 2, Bounds: Bounds{West: 0, South: 0, East: 1, North: 1}}
	gridded, err := ExtractHydroRIVERSGrid(path, "ref", "1", grid.Bounds, grid)
	if err != nil {
		t.Fatal(err)
	}
	sum := 0.0
	for _, cell := range gridded.GridCells {
		sum += cell.ClippedLengthKM
	}
	if math.Abs(sum-gridded.ClippedLengthKM) > 1e-6 {
		t.Fatalf("grid length conservation: gridded %f original %f", sum, gridded.ClippedLengthKM)
	}
	if len(gridded.GridCells) != 4 {
		t.Fatalf("wrong grid cell count %d", len(gridded.GridCells))
	}
}
