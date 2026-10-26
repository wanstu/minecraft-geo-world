package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"geoworld/internal/generator"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
)

type result struct {
	Seed          int64              `json:"seed"`
	Output        string             `json:"output"`
	FileBytes     int64              `json:"fileBytes"`
	DurationMs    int64              `json:"durationMs"`
	Geometry      world.GridGeometry `json:"geometry"`
	SchemaVersion int                `json:"schemaVersion"`
	Generator     string             `json:"generatorVersion"`
}

type progressLine struct {
	Type      string          `json:"type"`
	Stage     generator.Stage `json:"stage"`
	Completed int             `json:"completed"`
	Total     int             `json:"total"`
}

func main() {
	var (
		seed          = flag.Int64("seed", 731894205, "world seed")
		width         = flag.Int("width", 320, "macro grid width")
		height        = flag.Int("height", 200, "macro grid height")
		cellSize      = flag.Float64("cell-size", world.DefaultMacroSpacingBlocks, "blocks per macro node")
		plateCount    = flag.Int("plates", 18, "tectonic plate count")
		ocean         = flag.Float64("ocean", 0.68, "oceanic crust fraction")
		fragmentation = flag.Float64("fragmentation", 0.35, "continent fragmentation")
		separation    = flag.Float64("separation", 0.65, "continent separation")
		profilePreset = flag.String("profile", "standard", "world profile preset id")
		parameters    = flag.String("parameters", "", "25-value world parameter snapshot CSV")
		output        = flag.String("output", "", "output .geoworld path")
		force         = flag.Bool("force", false, "replace an existing output atomically")
		progressJSON  = flag.Bool("progress-json", false, "emit line-delimited progress JSON on stderr")
	)
	flag.Parse()

	if err := validateArgs(*width, *height, *cellSize, *plateCount, *ocean, *fragmentation, *separation, *output); err != nil {
		exitError(err)
	}

	absOutput, err := filepath.Abs(*output)
	if err != nil {
		exitError(err)
	}
	if err := os.MkdirAll(filepath.Dir(absOutput), 0o755); err != nil {
		exitError(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	geometry := world.GridGeometry{
		Width:       *width,
		Height:      *height,
		OriginX:     -float64(*width-1) * *cellSize / 2,
		OriginZ:     -float64(*height-1) * *cellSize / 2,
		SpanXBlocks: float64(*width-1) * *cellSize,
		SpanZBlocks: float64(*height-1) * *cellSize,
	}

	worldProfile := generator.WorldProfileForPreset(*profilePreset)
	if *parameters != "" {
		snapshot, err := generator.ParseParameterSnapshotCSV(*parameters)
		if err != nil {
			exitError(err)
		}
		worldProfile.Parameters = snapshot
	}

	start := time.Now()
	model, err := generator.GenerateWorld(ctx, generator.Config{
		Seed:                   *seed,
		Geometry:               geometry,
		WorldProfile:           worldProfile,
		PlateCount:             *plateCount,
		OceanFraction:          *ocean,
		ContinentFragmentation: *fragmentation,
		ContinentSeparation:    *separation,
	}, func(p generator.Progress) {
		if !*progressJSON {
			return
		}
		_ = json.NewEncoder(os.Stderr).Encode(progressLine{
			Type:      "progress",
			Stage:     p.Stage,
			Completed: p.Completed,
			Total:     p.Total,
		})
	})
	if err != nil {
		exitError(err)
	}

	if _, err := os.Stat(absOutput); err == nil {
		if !*force {
			exitError(fmt.Errorf("output already exists: %s", absOutput))
		}
		if err := worldmodelio.Replace(absOutput, model); err != nil {
			exitError(err)
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := worldmodelio.Save(absOutput, model); err != nil {
			exitError(err)
		}
	} else {
		exitError(err)
	}

	inspection, err := worldmodelio.Validate(absOutput)
	if err != nil {
		exitError(err)
	}
	info, err := os.Stat(absOutput)
	if err != nil {
		exitError(err)
	}

	out := result{
		Seed:          model.Config.Seed,
		Output:        absOutput,
		FileBytes:     info.Size(),
		DurationMs:    time.Since(start).Milliseconds(),
		Geometry:      model.Geometry,
		SchemaVersion: inspection.Manifest.SchemaVersion,
		Generator:     inspection.Manifest.GeneratorVersion,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(out); err != nil {
		exitError(err)
	}
}

func validateArgs(
	width int,
	height int,
	cellSize float64,
	plateCount int,
	ocean float64,
	fragmentation float64,
	separation float64,
	output string,
) error {
	if width < 16 || height < 16 || width > 1024 || height > 1024 {
		return errors.New("width and height must be in 16..1024")
	}
	if cellSize != 64 && cellSize != 128 {
		return errors.New("cell-size must be 64 or 128")
	}
	if plateCount < 2 || plateCount > 128 {
		return errors.New("plates must be in 2..128")
	}
	if ocean < 0.10 || ocean > 0.95 {
		return errors.New("ocean must be in 0.10..0.95")
	}
	if fragmentation < 0.05 || fragmentation > 1.0 {
		return errors.New("fragmentation must be in 0.05..1.0")
	}
	if separation < 0.05 || separation > 1.0 {
		return errors.New("separation must be in 0.05..1.0")
	}
	if output == "" {
		return errors.New("output is required")
	}
	return nil
}

func exitError(err error) {
	fmt.Fprintln(os.Stderr, "geoworld-engine:", err)
	os.Exit(1)
}
