package realgeom

import (
	"archive/zip"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/tingold/gocog"
)

func TestReadHGTAndZip(t *testing.T) {
	dir := t.TempDir()
	rawPath := filepath.Join(dir, "n04w075.hgt")
	values := []int16{
		9, 8, 7,
		6, -32768, 4,
		3, 2, 1,
	}
	f, err := os.Create(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range values {
		if err := binary.Write(f, binary.BigEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	r, err := ReadHGT(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if r.Width != 3 || r.Height != 3 || r.West != -75 || r.South != 4 {
		t.Fatalf("unexpected HGT metadata: %+v", r)
	}
	if _, ok := r.At(1, 1); ok {
		t.Fatal("HGT void value should be nodata")
	}

	zipPath := filepath.Join(dir, "NASADEM_HGT_n04w075.zip")
	zf, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(zf)
	entry, err := zw.Create("n04w075.hgt")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zf.Close(); err != nil {
		t.Fatal(err)
	}

	zr, err := ReadHGTZip(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	if zr.Width != r.Width || zr.West != r.West || zr.South != r.South {
		t.Fatalf("zip HGT metadata differs: raw=%+v zip=%+v", r, zr)
	}
}

func TestReadESRIASCII(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.asc")
	content := "ncols 3\nnrows 3\nxllcorner 10\nyllcorner 20\ncellsize 0.01\nNODATA_value -9999\n" +
		"9 8 7\n6 -9999 4\n3 2 1\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := ReadESRIASCII(path)
	if err != nil {
		t.Fatal(err)
	}
	if r.Width != 3 || r.Height != 3 || r.West != 10 || r.South != 20 {
		t.Fatalf("unexpected ASCII metadata: %+v", r)
	}
	if _, ok := r.At(1, 1); ok {
		t.Fatal("ASCII nodata should be invalid")
	}
}

func TestGeoTIFFValueDecoding(t *testing.T) {
	signed16 := int16(-1234)
	got, err := geoTIFFValue(gocog.DTSShortS, uint64(signed16))
	if err != nil || got != -1234 {
		t.Fatalf("signed16 got=%g err=%v", got, err)
	}

	signed32 := int32(-1234567)
	got, err = geoTIFFValue(gocog.DTSLongS, uint64(signed32))
	if err != nil || got != -1234567 {
		t.Fatalf("signed32 got=%g err=%v", got, err)
	}

	f32 := float32(-67.680824)
	got, err = geoTIFFValue(gocog.DTFloat, uint64(math.Float32bits(f32)))
	if err != nil || math.Abs(got-float64(f32)) > 1e-6 {
		t.Fatalf("float32 got=%g want=%g err=%v", got, f32, err)
	}

	f64 := 2131.558838
	got, err = geoTIFFValue(gocog.DTDouble, math.Float64bits(f64))
	if err != nil || math.Abs(got-f64) > 1e-12 {
		t.Fatalf("float64 got=%g want=%g err=%v", got, f64, err)
	}
}

func TestAnalyzePlaneProducesExpectedSlope(t *testing.T) {
	const cell = 0.001
	values := make([]float64, 25)
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			values[y*5+x] = float64(x * 10)
		}
	}
	r := Raster{
		Source: "plane", Width: 5, Height: 5,
		West: 0, South: 0, East: 5 * cell, North: 5 * cell,
		Values: values, NoData: math.NaN(),
		CellDegX: cell, CellDegY: cell,
	}
	report, err := Analyze([]Raster{r}, AnalyzeConfig{
		DatasetID: "synthetic", InputFormat: "test",
		ReliefWindowKM: 0.2, SampleStride: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.ValidElevationCells != 25 || report.ValidSlopeCells != 9 {
		t.Fatalf("unexpected sample counts: %+v", report)
	}
	if report.SlopeDegrees.P50 < 5.0 || report.SlopeDegrees.P50 > 5.3 {
		t.Fatalf("slope p50=%g want about 5.14 deg", report.SlopeDegrees.P50)
	}
	if math.Abs(report.HypsometricIntegral-0.5) > 1e-12 {
		t.Fatalf("hypsometric integral=%g", report.HypsometricIntegral)
	}
	if report.LocalReliefMeters.Max <= 0 {
		t.Fatal("expected positive local relief")
	}
}
