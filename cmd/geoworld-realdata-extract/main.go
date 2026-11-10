package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"geoworld/internal/realgeom"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	var inputs stringList
	flag.Var(&inputs, "input", "input raster path; repeat for multiple tiles")
	format := flag.String("format", "auto", "auto|hgt|hgt-zip|esri-ascii|geotiff")
	dataset := flag.String("dataset", "", "dataset id, e.g. NASADEM_HGT or GEBCO_2026")
	version := flag.String("version", "", "dataset version")
	region := flag.String("region", "", "reference region id")
	reliefWindow := flag.Float64("relief-window-km", 5, "local relief block size in km")
	stride := flag.Int("sample-stride", 4, "sample every Nth raster cell for elevation/slope statistics")
	stripeRows := flag.Int("geotiff-stripe-rows", 512, "GeoTIFF rows processed per chunk")
	noData := flag.Float64("nodata", math.NaN(), "optional explicit NoData value for GeoTIFF")
	output := flag.String("output", "-", "output JSON path or - for stdout")
	flag.Parse()

	if len(inputs) == 0 || *dataset == "" {
		fmt.Fprintln(os.Stderr, "--input and --dataset are required")
		os.Exit(2)
	}

	resolvedFormats := make(map[string]struct{})
	analyzer, err := realgeom.NewAnalyzer(realgeom.AnalyzeConfig{
		RegionID: *region, DatasetID: *dataset, DatasetVersion: *version,
		InputFormat: *format, ReliefWindowKM: *reliefWindow,
		SampleStride: *stride,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	for _, path := range inputs {
		f := resolveFormat(*format, path)
		resolvedFormats[f] = struct{}{}
		switch f {
		case "hgt":
			r, err := realgeom.ReadHGT(path)
			if err != nil {
				failInput(path, err)
			}
			if err := analyzer.AddRaster(r); err != nil {
				failInput(path, err)
			}
		case "hgt-zip":
			r, err := realgeom.ReadHGTZip(path)
			if err != nil {
				failInput(path, err)
			}
			if err := analyzer.AddRaster(r); err != nil {
				failInput(path, err)
			}
		case "esri-ascii":
			r, err := realgeom.ReadESRIASCII(path)
			if err != nil {
				failInput(path, err)
			}
			if err := analyzer.AddRaster(r); err != nil {
				failInput(path, err)
			}
		case "geotiff":
			err := realgeom.AddGeoTIFF(analyzer, path, realgeom.GeoTIFFOptions{
				NoData: *noData, HasNoData: !math.IsNaN(*noData),
				StripeRows: *stripeRows,
			})
			if err != nil {
				failInput(path, err)
			}
		default:
			failInput(path, fmt.Errorf("unsupported input format %q", f))
		}
	}

	report, err := analyzer.Report()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if *format == "auto" {
		if len(resolvedFormats) == 1 {
			for f := range resolvedFormats {
				report.InputFormat = f
			}
		} else {
			report.InputFormat = "mixed"
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	data = append(data, '\n')
	if *output == "-" {
		_, _ = os.Stdout.Write(data)
		return
	}
	dir := filepath.Dir(*output)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func failInput(path string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
	os.Exit(2)
}

func resolveFormat(requested, path string) string {
	if requested != "auto" {
		return requested
	}
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".hgt"):
		return "hgt"
	case strings.HasSuffix(lower, ".zip"):
		return "hgt-zip"
	case strings.HasSuffix(lower, ".asc"),
		strings.HasSuffix(lower, ".ascii"),
		strings.HasSuffix(lower, ".txt"):
		return "esri-ascii"
	case strings.HasSuffix(lower, ".tif"),
		strings.HasSuffix(lower, ".tiff"):
		return "geotiff"
	default:
		return "unknown"
	}
}
