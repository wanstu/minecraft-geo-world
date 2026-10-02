package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	goruntime "runtime"
	"time"

	"geoworld/internal/generator"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
)

type sampleCheck struct {
	X        float64
	Z        float64
	Expected runtimeprofile.Sample
}

type report struct {
	Seed           int64
	Width          int
	Height         int
	CellSizeBlocks float64

	SchemaVersion    int
	GeneratorVersion string

	FileBytes    int64
	PayloadBytes int64

	RiverSegments int
	LakeNodes     int

	Samples    int
	Mismatches int

	GenerateMillis int64
	SaveMillis     int64
	LoadMillis     int64

	GeneratePeakHeapBytes uint64
	GenerateAllocBytes    uint64

	SaveBaselineHeapBytes    uint64
	SavePeakHeapBytes        uint64
	SaveIncrementalPeakBytes uint64
	SaveAllocBytes           uint64

	LoadBaselineHeapBytes uint64
	LoadPeakHeapBytes     uint64
	LoadRetainedHeapBytes uint64
	LoadAllocBytes        uint64

	FileToPayloadRatio float64
}

type memoryMeasure struct {
	BaselineHeapBytes uint64
	PeakHeapBytes     uint64
	RetainedHeapBytes uint64
	AllocBytes        uint64
}

func main() {
	seed := flag.Int64("seed", 731894205, "world seed")
	width := flag.Int("width", 640, "macro grid nodes X")
	height := flag.Int("height", 400, "macro grid nodes Z")
	cellSize := flag.Float64("cell-size", 128, "blocks per macro-node interval")
	output := flag.String("output", "", "optional new .geoworld path; must not already exist")
	flag.Parse()

	if *width < 16 || *height < 16 || *cellSize <= 0 {
		fail(fmt.Errorf("invalid geometry"))
	}
	g := world.GridGeometry{
		Width: *width, Height: *height,
		SpanXBlocks: float64(*width-1) * *cellSize,
		SpanZBlocks: float64(*height-1) * *cellSize,
	}
	if err := g.Validate(); err != nil {
		fail(err)
	}

	var model *generator.WorldModel
	generateDuration, generateMemory, err := measureMemory(func() error {
		var generateErr error
		model, generateErr = generator.GenerateWorld(context.Background(), generator.Config{
			Seed:          *seed,
			Geometry:      g,
			PlateCount:    18,
			OceanFraction: 0.68,
		}, nil)
		return generateErr
	})
	if err != nil {
		fail(err)
	}

	checks := collectSamples(model)
	riverCount := len(model.Waterbed.Rivers)
	lakeCount := len(model.Waterbed.Lakes)

	path := *output
	var cleanup func()
	if path == "" {
		dir, err := os.MkdirTemp("", "geoworld-model-audit-*")
		if err != nil {
			fail(err)
		}
		path = filepath.Join(dir, "audit.geoworld")
		cleanup = func() { _ = os.RemoveAll(dir) }
	} else {
		cleanup = func() {}
	}
	defer cleanup()

	saveDuration, saveMemory, err := measureMemory(func() error {
		return worldmodelio.Save(path, model)
	})
	if err != nil {
		fail(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		fail(err)
	}

	// Reopen independently from the expensive generation graph.
	model = nil
	goruntime.GC()

	var reopened *worldmodelio.RuntimeModel
	loadDuration, loadMemory, err := measureMemory(func() error {
		var loadErr error
		reopened, loadErr = worldmodelio.Load(path)
		return loadErr
	})
	if err != nil {
		fail(err)
	}

	mismatches := 0
	for _, c := range checks {
		got := reopened.Sample(c.X, c.Z)
		if !reflect.DeepEqual(got, c.Expected) {
			mismatches++
		}
	}

	r := report{
		Seed:             *seed,
		Width:            *width,
		Height:           *height,
		CellSizeBlocks:   *cellSize,
		SchemaVersion:    reopened.Manifest.SchemaVersion,
		GeneratorVersion: reopened.Manifest.GeneratorVersion,
		FileBytes:        info.Size(),
		PayloadBytes:     reopened.Manifest.PayloadBytes,
		RiverSegments:    riverCount,
		LakeNodes:        lakeCount,
		Samples:          len(checks),
		Mismatches:       mismatches,
		GenerateMillis:   generateDuration.Milliseconds(),
		SaveMillis:       saveDuration.Milliseconds(),
		LoadMillis:       loadDuration.Milliseconds(),

		GeneratePeakHeapBytes: generateMemory.PeakHeapBytes,
		GenerateAllocBytes:    generateMemory.AllocBytes,

		SaveBaselineHeapBytes:    saveMemory.BaselineHeapBytes,
		SavePeakHeapBytes:        saveMemory.PeakHeapBytes,
		SaveIncrementalPeakBytes: saturatingSub(saveMemory.PeakHeapBytes, saveMemory.BaselineHeapBytes),
		SaveAllocBytes:           saveMemory.AllocBytes,

		LoadBaselineHeapBytes: loadMemory.BaselineHeapBytes,
		LoadPeakHeapBytes:     loadMemory.PeakHeapBytes,
		LoadRetainedHeapBytes: loadMemory.RetainedHeapBytes,
		LoadAllocBytes:        loadMemory.AllocBytes,

		FileToPayloadRatio: float64(info.Size()) / float64(reopened.Manifest.PayloadBytes),
	}
	if err := json.NewEncoder(os.Stdout).Encode(r); err != nil {
		fail(err)
	}
	if mismatches != 0 {
		os.Exit(1)
	}
}

func measureMemory(run func() error) (time.Duration, memoryMeasure, error) {
	goruntime.GC()

	var before goruntime.MemStats
	goruntime.ReadMemStats(&before)
	measure := memoryMeasure{
		BaselineHeapBytes: before.HeapAlloc,
		PeakHeapBytes:     before.HeapAlloc,
	}

	done := make(chan struct{})
	sampled := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		defer close(sampled)
		for {
			select {
			case <-ticker.C:
				var current goruntime.MemStats
				goruntime.ReadMemStats(&current)
				if current.HeapAlloc > measure.PeakHeapBytes {
					measure.PeakHeapBytes = current.HeapAlloc
				}
			case <-done:
				return
			}
		}
	}()

	start := time.Now()
	err := run()
	duration := time.Since(start)
	close(done)
	<-sampled

	var afterRun goruntime.MemStats
	goruntime.ReadMemStats(&afterRun)
	if afterRun.HeapAlloc > measure.PeakHeapBytes {
		measure.PeakHeapBytes = afterRun.HeapAlloc
	}
	measure.AllocBytes = afterRun.TotalAlloc - before.TotalAlloc

	goruntime.GC()
	var retained goruntime.MemStats
	goruntime.ReadMemStats(&retained)
	measure.RetainedHeapBytes = retained.HeapAlloc
	return duration, measure, err
}

func saturatingSub(a, b uint64) uint64 {
	if a <= b {
		return 0
	}
	return a - b
}

func collectSamples(m *generator.WorldModel) []sampleCheck {
	var points [][2]float64
	g := m.Geometry

	xStep := maxInt(1, g.Width/15)
	zStep := maxInt(1, g.Height/10)
	for z := 0; z < g.Height; z += zStep {
		for x := 0; x < g.Width; x += xStep {
			wx, wz := g.NodePosition(x, z)
			points = append(points, [2]float64{wx, wz})
		}
	}
	points = append(points,
		[2]float64{g.OriginX, g.OriginZ},
		[2]float64{g.OriginX + g.SpanXBlocks, g.OriginZ},
		[2]float64{g.OriginX, g.OriginZ + g.SpanZBlocks},
		[2]float64{g.OriginX + g.SpanXBlocks, g.OriginZ + g.SpanZBlocks},
	)

	for i, seg := range m.Waterbed.Rivers {
		if i >= 32 {
			break
		}
		for _, t := range []float64{0.2, 0.5, 0.8} {
			points = append(points, [2]float64{
				seg.X0 + (seg.X1-seg.X0)*t,
				seg.Z0 + (seg.Z1-seg.Z0)*t,
			})
		}
	}
	for i, lake := range m.Waterbed.Lakes {
		if i >= 32 {
			break
		}
		points = append(points, [2]float64{lake.X, lake.Z})
	}

	mixed := 0
	spacingX, spacingZ := g.SpacingX(), g.SpacingZ()
	for z := 0; z < g.Height-1 && mixed < 64; z++ {
		for x := 0; x < g.Width-1 && mixed < 64; x++ {
			ids := [4]int{
				z*g.Width + x,
				z*g.Width + x + 1,
				(z+1)*g.Width + x,
				(z+1)*g.Width + x + 1,
			}
			land := 0
			for _, i := range ids {
				if m.Surface.LandMask[i] {
					land++
				}
			}
			if land == 0 || land == 4 {
				continue
			}
			mixed++
			for _, off := range [][2]float64{{0.25, 0.25}, {0.5, 0.5}, {0.75, 0.75}} {
				points = append(points, [2]float64{
					g.OriginX + (float64(x)+off[0])*spacingX,
					g.OriginZ + (float64(z)+off[1])*spacingZ,
				})
			}
		}
	}

	checks := make([]sampleCheck, 0, len(points))
	for _, p := range points {
		checks = append(checks, sampleCheck{
			X: p[0], Z: p[1],
			Expected: m.RuntimeProfile.Sample(p[0], p[1]),
		})
	}
	return checks
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
