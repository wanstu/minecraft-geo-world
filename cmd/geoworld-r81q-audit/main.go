// GeoWorld R8.1q: reproducible candidate-only v1.1.27 geomorphology comparison.
// Produces isolated new models and Go samples; does not modify an existing save.
package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"geoworld/internal/generator"
	"geoworld/internal/reconcile"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/worldmodelio"
)

type SiteReport struct {
	ID                          int     `json:"id"`
	X                           float64 `json:"x"`
	Z                           float64 `json:"z"`
	RadiusX                     float64 `json:"radiusX"`
	RadiusZ                     float64 `json:"radiusZ"`
	ChangedPoints               int     `json:"changedPoints"`
	ComparedPoints              int     `json:"comparedPoints"`
	MaxAbsoluteHeightDifference float64 `json:"maxAbsoluteHeightDifference"`
	MaxNormalizedStepYPerBlock  float64 `json:"maxNormalizedStepYPerBlock"`
	WaterChanged                int     `json:"waterChanged"`
	GroundRaised                int     `json:"groundRaised"`
}

type Report struct {
	SchemaVersion            int          `json:"schemaVersion"`
	Seed                     int64        `json:"seed"`
	LegacyRules              string       `json:"legacyRules"`
	CandidateRules           string       `json:"candidateRules"`
	RiversIdentical          bool         `json:"riversIdentical"`
	SpecialFeaturesIdentical bool         `json:"specialFeaturesIdentical"`
	MacroSurfaceIdentical    bool         `json:"macroSurfaceIdentical"`
	OldCoastalVersion        int          `json:"oldCoastalVersion"`
	NewCoastalVersion        int          `json:"newCoastalVersion"`
	OldDurationSeconds       float64      `json:"oldDurationSeconds"`
	NewDurationSeconds       float64      `json:"newDurationSeconds"`
	Sites                    []SiteReport `json:"sites"`
	NotRealGameVisual        bool         `json:"notRealGameVisual"`
	ReleaseEligible          bool         `json:"releaseEligible"`
}

func configuration(src *worldmodelio.RuntimeModel, seed int64, rules string) generator.Config {
	cfg := generator.Config{
		Seed: seed, Geometry: src.Geometry,
		SeaLevelNormalized:      src.Manifest.SeaLevelNormalized,
		WorldProfile:            src.Config.WorldProfile,
		SurfaceOrePile:          src.Config.SurfaceOrePile,
		PlateCount:              src.Config.Phase1.PlateCount,
		OceanFraction:           src.Config.Phase1.OceanFraction,
		EdgeOceanBufferFraction: src.Config.Phase1.EdgeOceanBufferFraction,
		ContinentFragmentation:  src.Config.Phase1.ContinentFragmentation,
		ContinentSeparation:     src.Config.Phase1.ContinentSeparation,
	}
	cfg.WorldProfile.GenerationRulesVersion = rules
	return cfg
}

func value(m *generator.WorldModel, x, z float64) (float64, string, float64, bool) {
	base := m.RuntimeProfile.Sample(x, z)
	final := specialfeatures.Apply(
		m.SpecialFeatures, base, m.Config.SeaLevelNormalized,
		x, z, m.Config.WorldProfile.GenerationRulesVersion,
	)
	water := ""
	switch final.World.WaterKind {
	case reconcile.WaterNone:
		water = "None"
	case reconcile.WaterOcean:
		water = "Ocean"
	case reconcile.WaterLake:
		water = "Lake"
	case reconcile.WaterRiver:
		water = "River"
	default:
		panic("unknown water kind")
	}
	return final.World.GroundHeight, water,
		base.World.GroundHeight, base.World.WaterKind != reconcile.WaterNone
}

// terrain returns the actual fully applied category. The new v1.1.28
// habitat boundary is part of the Go/Java parity contract, not just height.
func terrain(m *generator.WorldModel, x, z float64) string {
	base := m.RuntimeProfile.Sample(x, z)
	final := specialfeatures.Apply(
		m.SpecialFeatures, base, m.Config.SeaLevelNormalized,
		x, z, m.Config.WorldProfile.GenerationRulesVersion,
	)
	return final.TerrainVariant.String()
}

func main() {
	if len(os.Args) != 5 && len(os.Args) != 6 {
		panic("usage: geoworld-r81q-audit <saved-v128.geoworld> <isolated-model-dir> <report.json> <sample.csv> [seed]")
	}
	src, err := worldmodelio.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	if src.Manifest.GenerationRulesVersion != "geoworld-rules-v1.1.29" {
		panic("reference must be real saved v1.1.29 world")
	}
	seed := src.Manifest.Seed
	if len(os.Args) == 6 {
		seed, err = strconv.ParseInt(os.Args[5], 10, 64)
		if err != nil {
			panic(err)
		}
	}
	outDir := os.Args[2]
	oldPath := filepath.Join(outDir, "v129.geoworld")
	newPath := filepath.Join(outDir, "v130.geoworld")
	for _, file := range []string{oldPath, newPath, os.Args[3], os.Args[4]} {
		if _, err := os.Stat(file); err == nil {
			panic("refusing to overwrite existing file " + file)
		}
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		panic(err)
	}
	versions := [2]string{"geoworld-rules-v1.1.29", "geoworld-rules-v1.1.30"}
	models := [2]*generator.WorldModel{}
	durations := [2]float64{}
	for i, ver := range versions {
		start := time.Now()
		generated, err := generator.GenerateWorld(context.Background(), configuration(src, seed, ver), nil)
		if err != nil {
			panic(err)
		}
		models[i] = generated
		durations[i] = time.Since(start).Seconds()
	}
	a, b := models[0], models[1]
	report := Report{
		SchemaVersion: 1, Seed: seed, LegacyRules: versions[0], CandidateRules: versions[1],
		RiversIdentical:          reflect.DeepEqual(a.Waterbed.Rivers, b.Waterbed.Rivers),
		SpecialFeaturesIdentical: reflect.DeepEqual(a.SpecialFeatures, b.SpecialFeatures),
		MacroSurfaceIdentical:    reflect.DeepEqual(a.Surface, b.Surface),
		OldCoastalVersion:        int(a.Config.WorldSample.CoastalBackwaterVersion),
		NewCoastalVersion:        int(b.Config.WorldSample.CoastalBackwaterVersion),
		OldDurationSeconds:       durations[0], NewDurationSeconds: durations[1],
		NotRealGameVisual: true, ReleaseEligible: false,
	}
	if !report.RiversIdentical || !report.SpecialFeaturesIdentical || !report.MacroSurfaceIdentical {
		panic("candidate changed river layout, feature placement or macro terrain")
	}
	if report.OldCoastalVersion != 2 || report.NewCoastalVersion != 2 {
		panic(fmt.Sprintf("coastal hydrology version changed %d vs %d", report.OldCoastalVersion, report.NewCoastalVersion))
	}
	if err := worldmodelio.Save(oldPath, a); err != nil {
		panic(err)
	}
	if err := worldmodelio.Save(newPath, b); err != nil {
		panic(err)
	}
	csvFile, err := os.OpenFile(os.Args[4], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		panic(err)
	}
	defer csvFile.Close()
	writer := csv.NewWriter(csvFile)
	writer.Write([]string{"x", "z", "oldGround", "newGround", "oldWater", "newWater", "oldTerrain", "newTerrain"})
	format := func(v float64) string { return strconv.FormatFloat(v, 'g', 17, 64) }
	for _, f := range b.SpecialFeatures {
		if f.Kind != specialfeatures.Sinkhole {
			continue
		}
		site := SiteReport{ID: f.ID, X: f.CenterX, Z: f.CenterZ, RadiusX: f.RadiusX, RadiusZ: f.RadiusZ}
		for n := 0; n < 48; n++ {
			theta := 2 * math.Pi * float64(n) / 48
			for _, q := range []float64{0, 0.23, 0.43, 0.58, 0.75, 0.89, 0.96, 1.04} {
				x := f.CenterX + q*f.RadiusX*math.Cos(theta)
				z := f.CenterZ + q*f.RadiusZ*math.Sin(theta)
				oy, ow, baseY, wet := value(a, x, z)
				ny, nw, _, _ := value(b, x, z)
				site.ComparedPoints++
				if math.Abs(oy-ny) > 0.002 {
					site.ChangedPoints++
				}
				site.MaxAbsoluteHeightDifference = math.Max(site.MaxAbsoluteHeightDifference, math.Abs(oy-ny))
				if wet && (oy != ny || ow != nw) {
					site.WaterChanged++
				}
				if ny > baseY+1e-10 {
					site.GroundRaised++
				}
				if err := writer.Write([]string{format(x), format(z), format(oy), format(ny), ow, nw, terrain(a, x, z), terrain(b, x, z)}); err != nil {
					panic(err)
				}
			}
		}
		// 4-block interval physical Y/step proxy in every 45 degree sector.
		for n := 0; n < 8; n++ {
			angle := math.Pi * float64(n) / 4
			last, _, _, _ := value(b, f.CenterX, f.CenterZ)
			for dist := 4.0; dist < math.Max(f.RadiusX, f.RadiusZ)*1.05; dist += 4 {
				x := f.CenterX + dist*math.Cos(angle)
				z := f.CenterZ + dist*math.Sin(angle)
				cur, _, _, _ := value(b, x, z)
				site.MaxNormalizedStepYPerBlock = math.Max(site.MaxNormalizedStepYPerBlock, math.Abs(cur-last)*514/4)
				last = cur
			}
		}
		if site.WaterChanged > 0 || site.GroundRaised > 0 {
			panic(fmt.Sprintf("sinkhole %d changed water or raised land", site.ID))
		}
		report.Sites = append(report.Sites, site)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		panic(err)
	}
	bs, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[3], append(bs, '\n'), 0644); err != nil {
		panic(err)
	}
	fmt.Printf("R81q seed=%d old=%s new=%s riverSame=%v featuresSame=%v surfaceSame=%v coast=%d/%d sites=%d samples=%d\n",
		report.Seed, versions[0], versions[1], report.RiversIdentical, report.SpecialFeaturesIdentical, report.MacroSurfaceIdentical,
		report.OldCoastalVersion, report.NewCoastalVersion, len(report.Sites), len(report.Sites)*48*8)
	for _, s := range report.Sites {
		fmt.Printf("  sinkhole=%d changed=%d/%d maxDifference=%.6f maxYPerBlock=%.3f\n",
			s.ID, s.ChangedPoints, s.ComparedPoints, s.MaxAbsoluteHeightDifference, s.MaxNormalizedStepYPerBlock)
	}
}
