package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"geoworld/internal/goldenvectors"
	"geoworld/internal/worldmodelio"
)

func main() {
	outDir := flag.String("out", filepath.FromSlash("internal/goldenvectors/testdata"), "output directory")
	force := flag.Bool("force", false, "replace existing golden artifacts")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}
	modelPath := filepath.Join(*outDir, "runtime-v1.geoworld")
	jsonPath := filepath.Join(*outDir, "runtime-v1.json")

	if !*force {
		for _, path := range []string{modelPath, jsonPath} {
			if _, err := os.Stat(path); err == nil {
				fatal(fmt.Errorf("%s already exists; pass -force to regenerate", path))
			} else if !os.IsNotExist(err) {
				fatal(err)
			}
		}
	} else {
		_ = os.Remove(modelPath)
		_ = os.Remove(jsonPath)
	}

	model, err := goldenvectors.GenerateFixture(context.Background())
	if err != nil {
		fatal(err)
	}
	if err := worldmodelio.Save(modelPath, model); err != nil {
		fatal(err)
	}

	doc, err := goldenvectors.Build(model)
	if err != nil {
		fatal(err)
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fatal(err)
	}
	data = append(data, byte(10))
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		fatal(err)
	}

	info, err := worldmodelio.Inspect(modelPath)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("golden model: %s (%d bytes)\n", modelPath, info.FileBytes)
	fmt.Printf("golden vectors: %s (%d cases)\n", jsonPath, len(doc.Cases))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
