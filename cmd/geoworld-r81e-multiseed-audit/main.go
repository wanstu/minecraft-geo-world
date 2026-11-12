package main

import (
	"context"
	"encoding/json"
	"fmt"
	"geoworld/internal/generator"
	"geoworld/internal/reconcile"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/worldmodelio"
	"math"
	"os"
	"reflect"
	"strconv"
	"time"
)

type Section struct {
	Axis         string  `json:"axis"`
	OldGrade     float64 `json:"oldMaxSlopeProxy"`
	NewGrade     float64 `json:"newMaxSlopeProxy"`
	OldFlat      int     `json:"oldNearFloorCount"`
	NewFlat      int     `json:"newNearFloorCount"`
	Steps        int     `json:"samples"`
	WetChanged   int     `json:"waterChanged"`
	GroundRaised int     `json:"groundRaised"`
}
type Site struct {
	ID               int `json:"id"`
	X, Z             float64
	RadiusX, RadiusZ float64
	Profiles         []Section `json:"profiles"`
}
type Case struct {
	Seed             int64      `json:"seed"`
	Seconds          [2]float64 `json:"generationSeconds"`
	RiverCount       int        `json:"riverCount"`
	RiversSame       bool       `json:"riversSame"`
	FeatureCount     int        `json:"featureCount"`
	FeaturesSame     bool       `json:"featuresSame"`
	MacroSurfaceSame bool       `json:"macroSurfaceSame"`
	Sites            []Site     `json:"sinkholes"`
	Warnings         []string   `json:"warnings,omitempty"`
}
type Report struct {
	SchemaVersion   int       `json:"schemaVersion"`
	EvidenceClass   string    `json:"evidenceClass"`
	Rules           [2]string `json:"rules"`
	Cases           []Case    `json:"cases"`
	ReleaseEligible bool      `json:"releaseEligible"`
}

func cfg(ref *worldmodelio.RuntimeModel, seed int64, rules string) generator.Config {
	c := generator.Config{Seed: seed, Geometry: ref.Geometry, SeaLevelNormalized: ref.Manifest.SeaLevelNormalized,
		WorldProfile: ref.Config.WorldProfile, SurfaceOrePile: ref.Config.SurfaceOrePile,
		PlateCount: ref.Config.Phase1.PlateCount, OceanFraction: ref.Config.Phase1.OceanFraction,
		EdgeOceanBufferFraction: ref.Config.Phase1.EdgeOceanBufferFraction,
		ContinentFragmentation:  ref.Config.Phase1.ContinentFragmentation, ContinentSeparation: ref.Config.Phase1.ContinentSeparation}
	c.WorldProfile.GenerationRulesVersion = rules
	return c
}
func sample(m *generator.WorldModel, f specialfeatures.Feature, axis string) ([]float64, int, int) {
	radius := f.RadiusX
	if axis == "z" {
		radius = f.RadiusZ
	}
	count := int(math.Ceil(1.06 * radius / 4))
	out := make([]float64, 0, count*2+1)
	wet, raised := 0, 0
	for i := -count; i <= count; i++ {
		x, z := f.CenterX, f.CenterZ
		if axis == "x" {
			x += float64(i) * 4
		} else {
			z += float64(i) * 4
		}
		b := m.RuntimeProfile.Sample(x, z)
		a := specialfeatures.Apply(m.SpecialFeatures, b, m.Config.SeaLevelNormalized, x, z, m.Config.WorldProfile.GenerationRulesVersion)
		if b.World.WaterKind != reconcile.WaterNone && a.World.GroundHeight != b.World.GroundHeight {
			wet++
		}
		if a.World.GroundHeight > b.World.GroundHeight+1e-10 {
			raised++
		}
		out = append(out, a.World.GroundHeight)
	}
	return out, wet, raised
}
func grade(ys []float64) (float64, int) {
	minY := ys[0]
	for _, y := range ys {
		minY = math.Min(minY, y)
	}
	maxGrade, near := 0., 0
	for i, y := range ys {
		if math.Abs(y-minY) < 1./514 {
			near++
		}
		if i > 0 {
			maxGrade = math.Max(maxGrade, math.Abs(y-ys[i-1])*514./4.)
		}
	}
	return maxGrade, near
}
func audit(seed int64, ref *worldmodelio.RuntimeModel) (Case, error) {
	rules := [2]string{"geoworld-rules-v1.1.24", "geoworld-rules-v1.1.25"}
	models := [2]*generator.WorldModel{}
	times := [2]float64{}
	for i, r := range rules {
		t := time.Now()
		m, err := generator.GenerateWorld(context.Background(), cfg(ref, seed, r), nil)
		if err != nil {
			return Case{}, err
		}
		models[i] = m
		times[i] = time.Since(t).Seconds()
	}
	a, b := models[0], models[1]
	c := Case{Seed: seed, Seconds: times, RiverCount: len(b.Waterbed.Rivers), RiversSame: reflect.DeepEqual(a.Waterbed.Rivers, b.Waterbed.Rivers),
		FeatureCount: len(b.SpecialFeatures), FeaturesSame: reflect.DeepEqual(a.SpecialFeatures, b.SpecialFeatures),
		MacroSurfaceSame: reflect.DeepEqual(a.Surface, b.Surface)}
	if !c.RiversSame {
		c.Warnings = append(c.Warnings, "river network changed")
	}
	if !c.FeaturesSame {
		c.Warnings = append(c.Warnings, "feature placement changed")
	}
	if !c.MacroSurfaceSame {
		c.Warnings = append(c.Warnings, "surface changed")
	}
	for _, f := range b.SpecialFeatures {
		if f.Kind != specialfeatures.Sinkhole {
			continue
		}
		site := Site{ID: f.ID, X: f.CenterX, Z: f.CenterZ, RadiusX: f.RadiusX, RadiusZ: f.RadiusZ}
		for _, axis := range []string{"x", "z"} {
			old, w1, r1 := sample(a, f, axis)
			newer, w2, r2 := sample(b, f, axis)
			og, on := grade(old)
			ng, nn := grade(newer)
			s := Section{Axis: axis, OldGrade: og, NewGrade: ng, OldFlat: on, NewFlat: nn, Steps: len(newer), WetChanged: w1 + w2, GroundRaised: r1 + r2}
			site.Profiles = append(site.Profiles, s)
			if s.WetChanged > 0 {
				c.Warnings = append(c.Warnings, fmt.Sprintf("site %d changed water %s", f.ID, axis))
			}
			if s.GroundRaised > 0 {
				c.Warnings = append(c.Warnings, fmt.Sprintf("site %d raised ground %s", f.ID, axis))
			}
			if s.NewGrade > s.OldGrade+0.25 {
				c.Warnings = append(c.Warnings, fmt.Sprintf("site %d slope increased %s", f.ID, axis))
			}
		}
		c.Sites = append(c.Sites, site)
	}
	if len(c.Sites) == 0 {
		c.Warnings = append(c.Warnings, "no sinkholes")
	}
	return c, nil
}
func main() {
	if len(os.Args) < 4 {
		panic("usage: multi-seed <saved-v125.geoworld> <output.json> <seed> ...")
	}
	ref, err := worldmodelio.Load(os.Args[1])
	if err != nil {
		panic(err)
	}
	if ref.Manifest.GenerationRulesVersion != "geoworld-rules-v1.1.25" {
		panic("reference rules not v125")
	}
	out := Report{SchemaVersion: 1, EvidenceClass: "GO_MODEL_ONLY_not_real_Minecraft_chunks", Rules: [2]string{"geoworld-rules-v1.1.24", "geoworld-rules-v1.1.25"}, ReleaseEligible: false}
	for _, v := range os.Args[3:] {
		seed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			panic(err)
		}
		c, err := audit(seed, ref)
		if err != nil {
			panic(err)
		}
		out.Cases = append(out.Cases, c)
		fmt.Printf("seed=%d rivers=%d features=%d sinkholes=%d sameRivers=%t sameFeatures=%t warnings=%v duration=[%.1fs %.1fs]\n", c.Seed, c.RiverCount, c.FeatureCount, len(c.Sites), c.RiversSame, c.FeaturesSame, c.Warnings, c.Seconds[0], c.Seconds[1])
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(os.Args[2], append(b, '\n'), 0644); err != nil {
		panic(err)
	}
}
