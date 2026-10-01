package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"geoworld/internal/worldmodelio"
)

type report struct {
	Validated bool                    `json:"validated"`
	Info      worldmodelio.Inspection `json:"info"`
}

func main() {
	validate := flag.Bool("validate", false, "verify payload checksum and config contracts without rebuilding Runtime")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: geoworld-model-inspect [-validate] <model.geoworld>")
		os.Exit(2)
	}

	path := flag.Arg(0)
	var (
		info worldmodelio.Inspection
		err  error
	)
	if *validate {
		info, err = worldmodelio.Validate(path)
	} else {
		info, err = worldmodelio.Inspect(path)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(report{Validated: *validate, Info: info}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
