// geoworld-reference-envelope converts locked real-source pixels/segments
// into per-archetype spatial reference distributions. It does NOT tune terrain.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"geoworld/internal/realgeom"
)

func main() {
	layer := flag.String("layer", "", "jrc-occurrence|jrc-seasonality|hydrosheds-direction|hydrosheds-accumulation|hydrorivers")
	input := flag.String("input", "", "path to source GeoTIFF or HydroRIVERS .shp")
	region := flag.String("region", "", "locked reference region ID")
	registry := flag.String("registry", "docs/realistic_geomorphology_reference_registry.json", "R6 region registry")
	version := flag.String("version", "", "locked dataset version (default per layer)")
	cols := flag.Int("grid-cols", 8, "spatial grid columns")
	rows := flag.Int("grid-rows", 10, "spatial grid rows")
	stride := flag.Int("sample-stride", 16, "source-grid sampling stride for raster data")
	stripe := flag.Int("geotiff-stripe-rows", 128, "GeoTIFF reading stripe height")
	minValid := flag.Int("min-valid-per-block", 32, "minimum valid source pixels per spatial grid block")
	output := flag.String("output", "-", "JSON output path; - for stdout")
	flag.Parse()
	if *layer == "" || *input == "" || *region == "" {
		fmt.Fprintln(os.Stderr, "--layer, --input and --region are required")
		os.Exit(2)
	}
	b, err := realgeom.LoadRegionBounds(*registry, *region)
	if err != nil {
		fatal(err)
	}
	v := *version
	if v == "" {
		switch *layer {
		case realgeom.LayerJRCOccurrence, realgeom.LayerJRCSeasonality:
			v = "1.5"
		case realgeom.LayerHydroDirection, realgeom.LayerHydroAccumulation:
			v = "1.1"
		case "hydrorivers":
			v = "1"
		default:
			fatal(fmt.Errorf("invalid --layer %s", *layer))
		}
	}
	report, err := realgeom.NewReferenceEnvelope(*input, realgeom.ReferenceExtractConfig{
		RegionID: *region, Bounds: b, Columns: *cols, Rows: *rows, DatasetVersion: v, Layer: *layer,
		SampleStride: *stride, StripeRows: *stripe, MinValidPerBlock: *minValid,
	})
	if err != nil {
		fatal(err)
	}
	for _, m := range report.Metrics {
		if m.SampleBlocks == 0 {
			fatal(fmt.Errorf("no valid sampled spatial blocks for %s", m.Name))
		}
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fatal(err)
	}
	data = append(data, '\n')
	if *output == "-" {
		if _, err := os.Stdout.Write(data); err != nil {
			fatal(err)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fatal(err)
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
