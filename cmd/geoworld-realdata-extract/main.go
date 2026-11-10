package main

import (
	"encoding/json"
	"flag"
	"fmt"
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
	format := flag.String("format", "auto", "auto|hgt|hgt-zip|esri-ascii")
	dataset := flag.String("dataset", "", "dataset id, e.g. NASADEM_HGT or GEBCO_2026")
	version := flag.String("version", "", "dataset version")
	region := flag.String("region", "", "reference region id")
	reliefWindow := flag.Float64("relief-window-km", 5, "local relief block size in km")
	stride := flag.Int("sample-stride", 4, "sample every Nth raster cell for elevation/slope statistics")
	output := flag.String("output", "-", "output JSON path or - for stdout")
	flag.Parse()

	if len(inputs) == 0 || *dataset == "" {
		fmt.Fprintln(os.Stderr, "--input and --dataset are required")
		os.Exit(2)
	}

	rasters := make([]realgeom.Raster, 0, len(inputs))
	resolvedFormats := make(map[string]struct{})
	for _, path := range inputs {
		f := resolveFormat(*format, path)
		var (
			r   realgeom.Raster
			err error
		)
		switch f {
		case "hgt":
			r, err = realgeom.ReadHGT(path)
		case "hgt-zip":
			r, err = realgeom.ReadHGTZip(path)
		case "esri-ascii":
			r, err = realgeom.ReadESRIASCII(path)
		default:
			err = fmt.Errorf("unsupported input format %q", f)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			os.Exit(2)
		}
		resolvedFormats[f] = struct{}{}
		rasters = append(rasters, r)
	}
	resolvedFormat := *format
	if *format == "auto" {
		if len(resolvedFormats) == 1 {
			for f := range resolvedFormats {
				resolvedFormat = f
			}
		} else {
			resolvedFormat = "mixed"
		}
	}

	report, err := realgeom.Analyze(rasters, realgeom.AnalyzeConfig{
		RegionID: *region, DatasetID: *dataset, DatasetVersion: *version,
		InputFormat: resolvedFormat, ReliefWindowKM: *reliefWindow,
		SampleStride: *stride,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
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
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil && filepath.Dir(*output) != "." {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func resolveFormat(requested, path string) string {
	if requested != "auto" {
		return requested
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".hgt":
		return "hgt"
	case ".zip":
		return "hgt-zip"
	case ".asc", ".ascii", ".txt":
		return "esri-ascii"
	default:
		return "unknown"
	}
}
