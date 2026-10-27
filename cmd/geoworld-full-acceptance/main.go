package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"geoworld/acceptance"
	"geoworld/internal/worldmodelio"
)

func main() {
	modelPath := flag.String("model", "", "locked .geoworld model path")
	outputDir := flag.String("output-dir", "build/manual-acceptance", "output directory")
	base := flag.String("base", "geoworld-full-acceptance", "output filename base")
	flag.Parse()
	if *modelPath == "" {
		fatal(fmt.Errorf("-model is required"))
	}
	model, err := worldmodelio.Load(*modelPath)
	if err != nil {
		fatal(err)
	}
	report, err := acceptance.Plan(model)
	if err != nil {
		fatal(err)
	}
	table, err := acceptance.BuildManualTable(report)
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fatal(err)
	}
	jsonData, err := acceptance.ManualTableJSON(table)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*outputDir, *base+".json"), jsonData, 0o644); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(filepath.Join(*outputDir, *base+".md"), []byte(acceptance.ManualTableMarkdown(table)), 0o644); err != nil {
		fatal(err)
	}
	csvFile, err := os.Create(filepath.Join(*outputDir, *base+".csv"))
	if err != nil {
		fatal(err)
	}
	if err := acceptance.WriteManualTableCSV(csvFile, table); err != nil {
		_ = csvFile.Close()
		fatal(err)
	}
	if err := csvFile.Close(); err != nil {
		fatal(err)
	}
	fmt.Printf("full manual acceptance table: %s (%d big categories)\n", *outputDir, len(table.BigCategories))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "geoworld-full-acceptance:", err)
	os.Exit(1)
}
