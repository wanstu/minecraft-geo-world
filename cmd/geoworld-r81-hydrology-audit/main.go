// GeoWorld R8.1: read-only water-graph, receiver and sinkhole investigation.
// No generated geometry is changed. Signals require final voxel verification.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"

	"geoworld/internal/reconcile"
	"geoworld/internal/specialfeatures"
	"geoworld/internal/waterbed"
	"geoworld/internal/worldmodelio"
)

const evidenceClass = "investigation_signal"

type Case struct {
	ID                string  `json:"id"`
	Category          string  `json:"category"`
	X                 float64 `json:"x"`
	Z                 float64 `json:"z"`
	Question          string  `json:"question"`
	RequiresVoxelTest bool    `json:"requiresVoxelTest"`
}
type Fixture struct {
	SchemaVersion int    `json:"schemaVersion"`
	Seed          int64  `json:"seed"`
	RulesVersion  string `json:"rulesVersion"`
	Cases         []Case `json:"cases"`
}
type Point struct {
	X              float64 `json:"x"`
	Z              float64 `json:"z"`
	Kind           string  `json:"kind"`
	GroundY        int     `json:"groundY"`
	WaterY         *int    `json:"waterY,omitempty"`
	AboveSeaBlocks *int    `json:"aboveSeaBlocks,omitempty"`
	SpecialFeature string  `json:"specialFeature,omitempty"`
}
type Site struct {
	Case                  Case    `json:"case"`
	Point                 Point   `json:"point"`
	NearestRiver          int     `json:"nearestRiver"`
	NearestDistanceBlocks float64 `json:"nearestDistanceBlocks"`
	NearestOceanMouth     bool    `json:"nearestOceanMouth"`
}
type Mouth struct {
	RiverID            int     `json:"riverId"`
	From               int     `json:"from"`
	To                 int     `json:"to"`
	HalfWidth          float64 `json:"halfWidth"`
	Samples            []Point `json:"samples"`
	DryAt98            bool    `json:"dryAt98"`
	EndpointNotOcean   bool    `json:"endpointNotOcean"`
	ReceiverRiseBlocks int     `json:"receiverRiseBlocks"`
}
type Sinkhole struct {
	ID                       int     `json:"id"`
	X                        float64 `json:"x"`
	Z                        float64 `json:"z"`
	NearestRiver             int     `json:"nearestRiver"`
	NearestDistanceBlocks    float64 `json:"nearestDistanceBlocks"`
	RiverIntersectsFootprint bool    `json:"riverIntersectsFootprint"`
	HasPersistedDrainage     bool    `json:"hasPersistedDrainage"`
}
type Counts struct {
	RiverSegments                  int `json:"riverSegments"`
	TerminalSegments               int `json:"terminalSegments"`
	OceanTerminals                 int `json:"oceanTerminals"`
	LakeTerminals                  int `json:"lakeTerminals"`
	UnclassifiedTerminals          int `json:"unclassifiedTerminals"`
	OceanMouths                    int `json:"oceanMouths"`
	DryApproachSignals             int `json:"dryApproachSignals"`
	EndpointNotOceanSignals        int `json:"endpointNotOceanSignals"`
	SeaAboveApproachSignals        int `json:"seaAboveApproachSignals"`
	LevelJumpSignals               int `json:"levelJumpSignals"`
	Sinkholes                      int `json:"sinkholes,omitempty"`
	SinkholeRiverOverlapSignals    int `json:"sinkholeRiverOverlapSignals"`
	SinkholeOverlapWithoutDrainage int `json:"sinkholeOverlapWithoutDrainage"`
}
type Report struct {
	EvidenceClass string     `json:"evidenceClass"`
	Seed          int64      `json:"seed"`
	RulesVersion  string     `json:"rulesVersion"`
	SeaLevelY     int        `json:"seaLevelY"`
	Counts        Counts     `json:"counts"`
	Sites         []Site     `json:"sites"`
	MouthExamples []Mouth    `json:"mouthExamples,omitempty"`
	Sinkholes     []Sinkhole `json:"sinkholes,omitempty"`
	Limitation    string     `json:"limitation"`
}

func waterKind(k reconcile.WaterKind) string {
	switch k {
	case reconcile.WaterNone:
		return "none"
	case reconcile.WaterRiver:
		return "river"
	case reconcile.WaterOcean:
		return "ocean"
	case reconcile.WaterLake:
		return "lake"
	default:
		return fmt.Sprintf("unknown-%d", k)
	}
}
func projectedY(h float64) int {
	h = math.Max(0, math.Min(1, h))
	if h <= .5 {
		return int(math.Ceil(-64 + 254*h - 1e-9))
	}
	return int(math.Ceil(63 + 514*(h-.5) - 1e-9))
}
func projectedGroundY(h float64) int {
	h = math.Max(0, math.Min(1, h))
	if h <= .5 {
		return int(math.Round(-64 + 254*h))
	}
	return int(math.Round(63 + 514*(h-.5)))
}
func sample(m *worldmodelio.RuntimeModel, x, z float64) Point {
	v := m.Sample(x, z)
	p := Point{X: x, Z: z, Kind: waterKind(v.World.WaterKind), GroundY: projectedGroundY(v.World.GroundHeight), SpecialFeature: v.SpecialFeature}
	if v.World.WaterSurfaceDefined && v.World.WaterKind != reconcile.WaterNone {
		y := projectedY(v.World.WaterSurfaceElevation)
		d := y - projectedY(m.Manifest.SeaLevelNormalized)
		p.WaterY = &y
		p.AboveSeaBlocks = &d
	}
	return p
}
func terminalCategory(s waterbed.RiverSegment) string {
	if s.OceanMouth {
		return "ocean"
	}
	if s.LakeMouth {
		return "lake"
	}
	return "unclassified" // May be legitimate inland/endorheic endpoint.
}
func validateCases(f Fixture, m *worldmodelio.RuntimeModel) error {
	if f.SchemaVersion != 1 {
		return errors.New("cases schema version must be 1")
	}
	if f.Seed != m.Manifest.Seed || f.RulesVersion != m.Manifest.GenerationRulesVersion {
		return fmt.Errorf("fixture/model mismatch: seed %d/%d rules %q/%q", f.Seed, m.Manifest.Seed, f.RulesVersion, m.Manifest.GenerationRulesVersion)
	}
	seen := map[string]bool{}
	for _, c := range f.Cases {
		if c.ID == "" || seen[c.ID] {
			return fmt.Errorf("duplicate or empty case id %q", c.ID)
		}
		seen[c.ID] = true
		if math.IsNaN(c.X) || math.IsInf(c.X, 0) || math.IsNaN(c.Z) || math.IsInf(c.Z, 0) {
			return fmt.Errorf("nonfinite position %s", c.ID)
		}
	}
	return nil
}
func nearestRiver(m *worldmodelio.RuntimeModel, x, z float64) (int, float64) {
	bestID := -1
	best := math.Inf(1)
	for i, s := range m.Waterbed.Rivers {
		// Lower bound for a curved centerline enclosed by the endpoints + bend.
		bend := math.Abs(s.MeanderOffset) + 2
		loX, hiX := math.Min(s.X0, s.X1)-bend, math.Max(s.X0, s.X1)+bend
		loZ, hiZ := math.Min(s.Z0, s.Z1)-bend, math.Max(s.Z0, s.Z1)+bend
		if x < loX-best || x > hiX+best || z < loZ-best || z > hiZ+best {
			continue
		}
		for j := 0; j <= 24; j++ {
			px, pz := waterbed.RiverCenterPoint(s, float64(j)/24)
			if d := math.Hypot(x-px, z-pz); d < best {
				best = d
				bestID = i
			}
		}
	}
	return bestID, best
}
func sinkhole(m *worldmodelio.RuntimeModel, sf specialfeatures.Feature) Sinkhole {
	id, d := nearestRiver(m, sf.CenterX, sf.CenterZ)
	f := Sinkhole{ID: sf.ID, X: sf.CenterX, Z: sf.CenterZ, NearestRiver: id, NearestDistanceBlocks: d, HasPersistedDrainage: len(sf.Drainage) > 0}
	for _, s := range m.Waterbed.Rivers {
		bend := math.Abs(s.MeanderOffset) + s.HalfWidth
		if math.Max(s.X0, s.X1)+bend < sf.CenterX-sf.RadiusX || math.Min(s.X0, s.X1)-bend > sf.CenterX+sf.RadiusX || math.Max(s.Z0, s.Z1)+bend < sf.CenterZ-sf.RadiusZ || math.Min(s.Z0, s.Z1)-bend > sf.CenterZ+sf.RadiusZ {
			continue
		}
		for j := 0; j <= 24; j++ {
			x, z := waterbed.RiverCenterPoint(s, float64(j)/24)
			dx := (x - sf.CenterX) / math.Max(1, sf.RadiusX)
			dz := (z - sf.CenterZ) / math.Max(1, sf.RadiusZ)
			if dx*dx+dz*dz < 1 {
				f.RiverIntersectsFootprint = true
				return f
			}
		}
	}
	return f
}
func audit(m *worldmodelio.RuntimeModel, f Fixture, maxDetails int) (Report, error) {
	if m == nil || m.Waterbed == nil {
		return Report{}, errors.New("missing loaded model or waterbed")
	}
	if maxDetails < 0 {
		return Report{}, errors.New("max-details negative")
	}
	if err := validateCases(f, m); err != nil {
		return Report{}, err
	}
	r := Report{EvidenceClass: evidenceClass, Seed: m.Manifest.Seed, RulesVersion: m.Manifest.GenerationRulesVersion, SeaLevelY: projectedY(m.Manifest.SeaLevelNormalized),
		Sites: []Site{}, MouthExamples: []Mouth{}, Sinkholes: []Sinkhole{},
		Limitation: "All signals use the persisted world model. None prove a Minecraft block/water-wall bug without actual Anvil water-state and bank height inspection."}
	r.Counts.RiverSegments = len(m.Waterbed.Rivers)
	outgoing := map[int]bool{}
	for _, s := range m.Waterbed.Rivers {
		outgoing[s.From] = true
	}
	for i, s := range m.Waterbed.Rivers {
		if !outgoing[s.To] {
			r.Counts.TerminalSegments++
			switch terminalCategory(s) {
			case "ocean":
				r.Counts.OceanTerminals++
			case "lake":
				r.Counts.LakeTerminals++
			default:
				r.Counts.UnclassifiedTerminals++
			}
		}
		if !s.OceanMouth {
			continue
		}
		r.Counts.OceanMouths++
		item := Mouth{RiverID: i, From: s.From, To: s.To, HalfWidth: s.HalfWidth}
		for _, t := range []float64{.75, .90, .98, 1} {
			x, z := waterbed.RiverCenterPoint(s, t)
			item.Samples = append(item.Samples, sample(m, x, z))
		}
		near := item.Samples[2]
		end := item.Samples[3]
		item.DryAt98 = near.Kind == "none"
		item.EndpointNotOcean = end.Kind != "ocean"
		if item.DryAt98 {
			r.Counts.DryApproachSignals++
		}
		if item.EndpointNotOcean {
			r.Counts.EndpointNotOceanSignals++
		}
		if near.WaterY != nil && end.WaterY != nil {
			item.ReceiverRiseBlocks = *end.WaterY - *near.WaterY
			if item.ReceiverRiseBlocks > 1 {
				r.Counts.SeaAboveApproachSignals++
			}
			if math.Abs(float64(item.ReceiverRiseBlocks)) > 1 {
				r.Counts.LevelJumpSignals++
			}
		}
		if len(r.MouthExamples) < maxDetails && (item.DryAt98 || item.EndpointNotOcean || math.Abs(float64(item.ReceiverRiseBlocks)) > 1) {
			r.MouthExamples = append(r.MouthExamples, item)
		}
	}
	for _, c := range f.Cases {
		id, d := nearestRiver(m, c.X, c.Z)
		site := Site{Case: c, Point: sample(m, c.X, c.Z), NearestRiver: id, NearestDistanceBlocks: d}
		if id >= 0 {
			site.NearestOceanMouth = m.Waterbed.Rivers[id].OceanMouth
		}
		r.Sites = append(r.Sites, site)
	}
	for _, f := range m.SpecialFeatures {
		if f.Kind != specialfeatures.Sinkhole {
			continue
		}
		r.Counts.Sinkholes++
		s := sinkhole(m, f)
		if s.RiverIntersectsFootprint {
			r.Counts.SinkholeRiverOverlapSignals++
			if !s.HasPersistedDrainage {
				r.Counts.SinkholeOverlapWithoutDrainage++
			}
		}
		if s.RiverIntersectsFootprint && len(r.Sinkholes) < maxDetails {
			r.Sinkholes = append(r.Sinkholes, s)
		}
	}
	return r, nil
}
func main() {
	modelFile := flag.String("model", "", "path to saved .geoworld")
	fixtureFile := flag.String("cases", "docs/acceptance/r81_hydrology_cases.json", "fixed R8.1 samples")
	output := flag.String("output", "", "output JSON; stdout if empty")
	maxDetails := flag.Int("max-details", 24, "maximum mouth/sinkhole examples")
	flag.Parse()
	if *modelFile == "" {
		fmt.Fprintln(os.Stderr, "--model is required")
		os.Exit(2)
	}
	m, e := worldmodelio.Load(*modelFile)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	b, e := os.ReadFile(*fixtureFile)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	var f Fixture
	if e = json.Unmarshal(b, &f); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	rep, e := audit(m, f, *maxDetails)
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	buf, e := json.MarshalIndent(rep, "", "  ")
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	buf = append(buf, '\n')
	if *output == "" {
		_, _ = os.Stdout.Write(buf)
		return
	}
	if e = os.WriteFile(*output, buf, 0644); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(2)
	}
	fmt.Printf("Saved R8.1 model audit: %s\n", *output)
}
