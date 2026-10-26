package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"geoworld/acceptance"
	"geoworld/internal/worldmodelio"
)

var fixedSeeds = []int64{
	123456,
	-4852173505788415804,
	-563659577384754347,
	4034658472718554561,
}

var representativePresets = []string{
	"standard",
	"natural",
	"civilization",
	"epic",
}

func main() {
	engine := flag.String("engine", "", "path to the exact geoworld-engine executable under test")
	outputDir := flag.String("output-dir", "", "directory for generated matrix models and batch report")
	randomSeed := flag.Int64("random-seed", 0, "shared random seed; generated when omitted")
	timeout := flag.Duration("timeout", 5*time.Minute, "per-model engine timeout")
	flag.Parse()

	if strings.TrimSpace(*engine) == "" {
		fatal(fmt.Errorf("-engine is required"))
	}
	if strings.TrimSpace(*outputDir) == "" {
		fatal(fmt.Errorf("-output-dir is required"))
	}
	if *randomSeed == 0 {
		*randomSeed = newRandomSeed()
	}

	enginePath, err := filepath.Abs(*engine)
	if err != nil {
		fatal(err)
	}
	if info, err := os.Stat(enginePath); err != nil || info.IsDir() {
		fatal(fmt.Errorf("engine executable not found: %s", enginePath))
	}
	root, err := filepath.Abs(*outputDir)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		fatal(err)
	}

	seeds := append(append([]int64(nil), fixedSeeds...), *randomSeed)
	reports := make([]acceptance.Report, 0, len(seeds)*len(representativePresets))

	for _, preset := range representativePresets {
		dir := filepath.Join(root, preset)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fatal(err)
		}
		for _, seed := range seeds {
			modelPath := filepath.Join(dir, strconv.FormatInt(seed, 10)+".geoworld")
			if err := generate(enginePath, modelPath, preset, seed, *timeout); err != nil {
				fatal(err)
			}
			model, err := worldmodelio.Load(modelPath)
			if err != nil {
				fatal(fmt.Errorf("%s/%d load: %w", preset, seed, err))
			}
			report, err := acceptance.Plan(model)
			if err != nil {
				fatal(fmt.Errorf("%s/%d acceptance: %w", preset, seed, err))
			}
			reports = append(reports, report)
			fmt.Printf(
				"generated preset=%s seed=%d snapshot=%s\n",
				preset,
				seed,
				report.ParameterSnapshotSHA256,
			)
		}
	}

	batch, err := acceptance.BuildBatch(reports)
	if err != nil {
		fatal(err)
	}
	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		fatal(err)
	}
	data = append(data, '\n')
	batchPath := filepath.Join(root, "batch.acceptance.json")
	if err := os.WriteFile(batchPath, data, 0o644); err != nil {
		fatal(err)
	}

	problems := 0
	for _, result := range batch.Results {
		if result.CoverageStatus == acceptance.BatchMissingSample ||
			result.CoverageStatus == acceptance.BatchMissingMatrix {
			problems++
		}
	}
	fmt.Printf(
		"matrix complete=%t models=%d items=%d randomSeed=%d problems=%d report=%s\n",
		batch.MatrixComplete,
		len(batch.Models),
		len(batch.Results),
		*randomSeed,
		problems,
		batchPath,
	)
	if !batch.MatrixComplete || len(batch.Anomalies) != 0 || problems != 0 {
		os.Exit(2)
	}
}

func generate(
	enginePath string,
	outputPath string,
	preset string,
	seed int64,
	timeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		enginePath,
		"-seed", strconv.FormatInt(seed, 10),
		"-profile", preset,
		"-output", outputPath,
		"-force",
	)
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return fmt.Errorf("%s/%d engine timeout: %w", preset, seed, ctx.Err())
	}
	if err != nil {
		return fmt.Errorf(
			"%s/%d engine failed: %w: %s",
			preset,
			seed,
			err,
			strings.TrimSpace(string(output)),
		)
	}
	return nil
}

func newRandomSeed() int64 {
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		fatal(err)
	}
	value := int64(binary.LittleEndian.Uint64(buffer[:]))
	if value == 0 {
		return 1
	}
	return value
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "geoworld-acceptance-matrix:", err)
	os.Exit(1)
}
