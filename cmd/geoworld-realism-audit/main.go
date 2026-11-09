package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"geoworld/internal/generator"
	"geoworld/internal/geomorphaudit"
	"geoworld/internal/world"
)

func main() {
	seed := flag.Int64("seed", 123456, "world seed")
	width := flag.Int("width", 320, "macro grid width")
	height := flag.Int("height", 200, "macro grid height")
	cell := flag.Float64("cell-size", 128, "macro cell size in blocks")
	step := flag.Float64("route-step", 16, "river route sample spacing in blocks")
	probe := flag.Float64("transition-probe", 16, "terrain transition probe distance in blocks")
	maxDetails := flag.Int("max-details", 64, "maximum detailed records per signal category")
	rulesVersion := flag.String(
		"rules-version",
		"",
		"optional generation rules version override for locked-rule A/B audits",
	)
	flag.Parse()

	geometry := world.GridGeometry{
		Width:       *width,
		Height:      *height,
		OriginX:     -float64(*width-1) * *cell / 2,
		OriginZ:     -float64(*height-1) * *cell / 2,
		SpanXBlocks: float64(*width-1) * *cell,
		SpanZBlocks: float64(*height-1) * *cell,
	}
	generatorConfig := generator.Config{
		Seed:                   *seed,
		Geometry:               geometry,
		PlateCount:             18,
		OceanFraction:          0.68,
		ContinentFragmentation: 0.35,
		ContinentSeparation:    0.65,
	}
	if *rulesVersion != "" {
		profile := generator.StandardWorldProfile()
		profile.GenerationRulesVersion = *rulesVersion
		generatorConfig.WorldProfile = profile
	}
	model, err := generator.GenerateWorld(
		context.Background(),
		generatorConfig,
		nil,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	cfg := geomorphaudit.DefaultConfig()
	cfg.RouteSampleStepBlocks = *step
	cfg.TransitionProbeBlocks = *probe
	cfg.MaxDetailRecords = *maxDetails
	report, err := geomorphaudit.Analyze(model, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	fmt.Println(string(data))
}
