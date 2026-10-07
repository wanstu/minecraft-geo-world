// geoworld-river-navigation-audit measures Minecraft integer-water-level
// stair frequency on low-gradient, boat-sized rivers. Steep reaches are
// reported separately as rapids rather than being forced into flat canals.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"

	"geoworld/internal/generator"
	"geoworld/internal/reconcile"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

type bucket struct {
	Name        string  `json:"name"`
	Segments    int     `json:"segments"`
	Length      float64 `json:"lengthBlocks"`
	Changes     int     `json:"waterYChanges"`
	ChangesPerK float64 `json:"changesPerKm"`
	Rises       int     `json:"downstreamRises"`
	DrySamples  int     `json:"dryCenterlineSamples"`
	MaxDrop     int     `json:"maxSingleDropBlocks"`
}

type report struct {
	Seed         int64    `json:"seed"`
	SampleStep   float64  `json:"sampleStepBlocks"`
	LowGradient  []bucket `json:"lowGradient"`
	Rapids       []bucket `json:"rapids"`
	TotalRises   int      `json:"totalDownstreamRises"`
	TotalDry     int      `json:"totalDryCenterlineSamples"`
	StrictPassed bool     `json:"strictPassed"`
}

func main() {
	seed := flag.Int64("seed", 123456, "world seed")
	width := flag.Int("width", 320, "macro grid width")
	height := flag.Int("height", 200, "macro grid height")
	cell := flag.Float64("cell-size", 128, "macro cell size in blocks")
	step := flag.Float64("sample-step", 4, "centerline sample spacing in blocks")
	strict := flag.Bool("strict", false, "exit non-zero when navigable-river limits fail")
	flag.Parse()

	g := world.GridGeometry{
		Width:       *width,
		Height:      *height,
		OriginX:     -float64(*width-1) * *cell / 2,
		OriginZ:     -float64(*height-1) * *cell / 2,
		SpanXBlocks: float64(*width-1) * *cell,
		SpanZBlocks: float64(*height-1) * *cell,
	}
	m, err := generator.GenerateWorld(context.Background(), generator.Config{
		Seed:                   *seed,
		Geometry:               g,
		PlateCount:             18,
		OceanFraction:          0.68,
		ContinentFragmentation: 0.35,
		ContinentSeparation:    0.65,
	}, nil)
	if err != nil {
		panic(err)
	}

	low := map[string]*bucket{
		"small":  {Name: "small 8-12"},
		"medium": {Name: "medium 12-16"},
		"large":  {Name: "large >=16"},
	}
	rapid := map[string]*bucket{
		"small":  {Name: "small 8-12"},
		"medium": {Name: "medium 12-16"},
		"large":  {Name: "large >=16"},
	}

	totalRises, totalDry := 0, 0
	for _, seg := range m.Waterbed.Rivers {
		if seg.HalfWidth < 8 {
			continue
		}
		key := widthBucket(seg.HalfWidth)
		depth := riverCenterDepth(m.Waterbed.Config, seg.HalfWidth)
		startY := minecraftWaterY(seg.Profile[0] + depth)
		endY := minecraftWaterY(seg.Profile[len(seg.Profile)-1] + depth)
		isLow := startY-endY <= 3
		target := rapid[key]
		if isLow {
			target = low[key]
		}
		target.Segments++

		length := math.Hypot(seg.X1-seg.X0, seg.Z1-seg.Z0)
		target.Length += length
		steps := maxInt(1, int(math.Ceil(length/math.Max(1, *step))))
		prevSet := false
		prevY := 0
		for j := 0; j <= steps; j++ {
			t := float64(j) / float64(steps)
			x, z := riverCenter(seg, t)
			s := m.Runtime.Sample(x, z)
			if !s.WaterSurfaceDefined {
				target.DrySamples++
				totalDry++
				prevSet = false
				continue
			}
			// Lake/ocean at the last samples is valid navigable receiving water.
			if s.WaterKind != reconcile.WaterRiver &&
				s.WaterKind != reconcile.WaterLake &&
				s.WaterKind != reconcile.WaterOcean {
				target.DrySamples++
				totalDry++
				prevSet = false
				continue
			}
			y := minecraftWaterY(s.WaterSurfaceElevation)
			if prevSet && y != prevY {
				target.Changes++
				d := prevY - y
				if d < 0 {
					target.Rises++
					totalRises++
				} else if d > target.MaxDrop {
					target.MaxDrop = d
				}
			}
			prevY = y
			prevSet = true
		}
	}

	lowOrder := []bucket{*low["small"], *low["medium"], *low["large"]}
	rapidOrder := []bucket{*rapid["small"], *rapid["medium"], *rapid["large"]}
	for i := range lowOrder {
		if lowOrder[i].Length > 0 {
			lowOrder[i].ChangesPerK = 1000 * float64(lowOrder[i].Changes) / lowOrder[i].Length
		}
	}
	for i := range rapidOrder {
		if rapidOrder[i].Length > 0 {
			rapidOrder[i].ChangesPerK = 1000 * float64(rapidOrder[i].Changes) / rapidOrder[i].Length
		}
	}

	pass := totalRises == 0 &&
		lowOrder[0].ChangesPerK <= 5.0 &&
		lowOrder[1].ChangesPerK <= 3.5 &&
		lowOrder[2].ChangesPerK <= 2.0
	r := report{
		Seed:         *seed,
		SampleStep:   *step,
		LowGradient:  lowOrder,
		Rapids:       rapidOrder,
		TotalRises:   totalRises,
		TotalDry:     totalDry,
		StrictPassed: pass,
	}
	data, _ := json.MarshalIndent(r, "", "  ")
	fmt.Println(string(data))
	if *strict && !pass {
		os.Exit(1)
	}
}

func widthBucket(halfWidth float64) string {
	if halfWidth >= 16 {
		return "large"
	}
	if halfWidth >= 12 {
		return "medium"
	}
	return "small"
}

func riverCenter(seg waterbed.RiverSegment, t float64) (float64, float64) {
	t = clamp(t, 0, 1)
	dx, dz := seg.X1-seg.X0, seg.Z1-seg.Z0
	length := math.Hypot(dx, dz)
	if length <= 1e-12 {
		return seg.X0, seg.Z0
	}
	nx, nz := -dz/length, dx/length
	offset := seg.MeanderOffset * math.Sin(math.Pi*t)
	return seg.X0 + dx*t + nx*offset, seg.Z0 + dz*t + nz*offset
}

func riverCenterDepth(cfg waterbed.Config, halfWidth float64) float64 {
	span := math.Max(1e-9, cfg.MaxRiverHalfWidthBlocks-cfg.MinRiverHalfWidthBlocks)
	t := clamp((halfWidth-cfg.MinRiverHalfWidthBlocks)/span, 0, 1)
	return cfg.RiverCenterDepthNormalized * (1 + 2*math.Pow(t, 1.15))
}

func minecraftWaterY(h float64) int {
	h = clamp(h, 0, 1)
	var mapped float64
	if h <= 0.5 {
		mapped = -64 + 127*(h/0.5)
	} else {
		mapped = 63 + 257*((h-0.5)/0.5)
	}
	return int(math.Ceil(mapped - 1e-9))
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
