// GeoWorld R8.1g hydrologic regression fixture: regenerate an independent
// v1.1.26 world from persisted seed, geometry and world profile. Boolean
// creation defaults such as SurfaceOrePile may differ; acceptance must compare
// saved payloads and never assume they are entirely identical.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"geoworld/internal/generator"
	"geoworld/internal/worldmodelio"
)

func main() {
	if len(os.Args) != 3 {
		panic("usage: geoworld-r81g-generate <saved-v125.geoworld> <new-v126.geoworld>")
	}
	src, err := worldmodelio.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	if src.Manifest.GenerationRulesVersion != "geoworld-rules-v1.1.25" {
		panic("source must be saved v125 world")
	}
	if _, err := os.Stat(os.Args[2]); err == nil {
		panic("refusing to overwrite existing world fixture")
	}
	cfg := generator.Config{
		Seed: src.Manifest.Seed, Geometry: src.Geometry, SeaLevelNormalized: src.Manifest.SeaLevelNormalized,
		WorldProfile: src.Config.WorldProfile, SurfaceOrePile: src.Config.SurfaceOrePile,
		PlateCount: src.Config.Phase1.PlateCount, OceanFraction: src.Config.Phase1.OceanFraction,
		EdgeOceanBufferFraction: src.Config.Phase1.EdgeOceanBufferFraction,
		ContinentFragmentation:  src.Config.Phase1.ContinentFragmentation,
		ContinentSeparation:     src.Config.Phase1.ContinentSeparation,
	}
	cfg.WorldProfile.GenerationRulesVersion = "geoworld-rules-v1.1.26"
	start := time.Now()
	out, err := generator.GenerateWorld(context.Background(), cfg, nil)
	if err != nil {
		panic(err)
	}
	fmt.Printf("generated seed=%d rules=%s rivers=%d special=%d duration=%s\n",
		cfg.Seed, cfg.WorldProfile.GenerationRulesVersion, len(out.Waterbed.Rivers), len(out.SpecialFeatures), time.Since(start))
	if err := worldmodelio.Save(os.Args[2], out); err != nil {
		panic(err)
	}
	saved, err := worldmodelio.Load(os.Args[2])
	if err != nil {
		panic(err)
	}
	fmt.Printf("saved/verified=%s rules=%s coastVersion=%d\n", os.Args[2],
		saved.Manifest.GenerationRulesVersion, saved.Config.WorldSample.CoastalBackwaterVersion)
}
