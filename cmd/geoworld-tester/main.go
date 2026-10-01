package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"geoworld/internal/generator"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
)

//go:embed web/*
var webFiles embed.FS

type activeWorld struct {
	seed        int64
	geometry    world.GridGeometry
	sample      func(float64, float64) runtimeprofile.Sample
	riverCount  int
	lakeCount   int
	source      string
	model       *generator.WorldModel
	reopened    *worldmodelio.RuntimeModel
	generatedAt time.Time
	duration    time.Duration
}

type appState struct {
	mu         sync.RWMutex
	active     *activeWorld
	generating bool
	stage      generator.Stage
	completed  int
	total      int
	startedAt  time.Time
	lastError  string
}

type generateRequest struct {
	Seed          int64   `json:"seed"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	CellSize      float64 `json:"cellSize"`
	PlateCount    int     `json:"plateCount"`
	OceanFraction float64 `json:"oceanFraction"`
}

type saveRequest struct {
	Name string `json:"name"`
}

type loadRequest struct {
	Name string `json:"name"`
}

type modelInfo struct {
	Name             string             `json:"name"`
	FileBytes        int64              `json:"fileBytes"`
	Seed             int64              `json:"seed"`
	SchemaVersion    int                `json:"schemaVersion"`
	GeneratorVersion string             `json:"generatorVersion"`
	Geometry         world.GridGeometry `json:"geometry"`
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8788", "listen address")
	openBrowser := flag.Bool("open", true, "open browser automatically")
	modelDir := flag.String("models", filepath.FromSlash("output/models"), "saved model directory")
	flag.Parse()

	state := &appState{}
	if err := os.MkdirAll(*modelDir, 0o755); err != nil {
		log.Fatal(err)
	}

	staticFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
	mux.HandleFunc("/api/status", state.handleStatus)
	mux.HandleFunc("/api/generate", state.handleGenerate)
	mux.HandleFunc("/api/sample", state.handleSample)
	mux.HandleFunc("/api/layer", state.handleLayer)
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) { state.handleSave(w, r, *modelDir) })
	mux.HandleFunc("/api/load", func(w http.ResponseWriter, r *http.Request) { state.handleLoad(w, r, *modelDir) })
	mux.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) { handleModels(w, r, *modelDir) })

	url := "http://" + *addr
	fmt.Printf("GeoWorld Tester: %s\n", url)
	fmt.Printf("models: %s\n", *modelDir)
	if *openBrowser {
		go func() {
			time.Sleep(350 * time.Millisecond)
			_ = openURL(url)
		}()
	}
	log.Fatal(http.ListenAndServe(*addr, logRequests(mux)))
}

func (s *appState) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	resp := map[string]any{
		"generating": s.generating,
		"stage":      s.stage,
		"completed":  s.completed,
		"total":      s.total,
		"error":      s.lastError,
	}
	if s.generating {
		resp["elapsedMs"] = time.Since(s.startedAt).Milliseconds()
	}
	if s.active != nil {
		resp["active"] = activeSummary(s.active)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *appState) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Width == 0 {
		req.Width = 320
	}
	if req.Height == 0 {
		req.Height = 200
	}
	if req.CellSize == 0 {
		req.CellSize = world.DefaultMacroSpacingBlocks
	}
	if req.PlateCount == 0 {
		req.PlateCount = 18
	}
	if req.OceanFraction == 0 {
		req.OceanFraction = 0.68
	}
	if req.Width < 16 || req.Height < 16 || req.Width > 1024 || req.Height > 1024 ||
		req.CellSize <= 0 || req.PlateCount < 2 ||
		req.OceanFraction <= 0 || req.OceanFraction >= 1 {
		writeError(w, http.StatusBadRequest, errors.New("invalid generation parameters"))
		return
	}

	s.mu.Lock()
	if s.generating {
		s.mu.Unlock()
		writeError(w, http.StatusConflict, errors.New("generation already running"))
		return
	}
	s.generating = true
	s.stage = ""
	s.completed = 0
	s.total = 13
	s.startedAt = time.Now()
	s.lastError = ""
	s.mu.Unlock()

	start := time.Now()
	g := world.GridGeometry{
		Width: req.Width, Height: req.Height,
		OriginX:     -float64(req.Width-1) * req.CellSize / 2,
		OriginZ:     -float64(req.Height-1) * req.CellSize / 2,
		SpanXBlocks: float64(req.Width-1) * req.CellSize,
		SpanZBlocks: float64(req.Height-1) * req.CellSize,
	}
	model, err := generator.GenerateWorld(context.Background(), generator.Config{
		Seed:          req.Seed,
		Geometry:      g,
		PlateCount:    req.PlateCount,
		OceanFraction: req.OceanFraction,
	}, func(p generator.Progress) {
		s.mu.Lock()
		s.stage = p.Stage
		s.completed = p.Completed
		s.total = p.Total
		s.mu.Unlock()
	})
	duration := time.Since(start)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.generating = false
	if err != nil {
		s.lastError = err.Error()
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	active := &activeWorld{
		seed:        model.Config.Seed,
		geometry:    model.Geometry,
		sample:      model.RuntimeProfile.Sample,
		riverCount:  len(model.Waterbed.Rivers),
		lakeCount:   len(model.Waterbed.Lakes),
		source:      "generated",
		model:       model,
		generatedAt: time.Now(),
		duration:    duration,
	}
	s.active = active
	writeJSON(w, http.StatusOK, activeSummary(active))
}

func (s *appState) handleSample(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	x, err := strconv.ParseFloat(r.URL.Query().Get("x"), 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid x"))
		return
	}
	z, err := strconv.ParseFloat(r.URL.Query().Get("z"), 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("invalid z"))
		return
	}
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active == nil {
		writeError(w, http.StatusConflict, errors.New("generate or load a world first"))
		return
	}
	sample := active.sample(x, z)
	writeJSON(w, http.StatusOK, sampleJSON(x, z, sample))
}

func (s *appState) handleLayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	layer := r.URL.Query().Get("name")
	if layer == "" {
		layer = "final"
	}
	width := parseBoundedInt(r.URL.Query().Get("width"), 720, 160, 1200)
	height := parseBoundedInt(r.URL.Query().Get("height"), 450, 100, 800)

	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active == nil {
		writeError(w, http.StatusConflict, errors.New("generate or load a world first"))
		return
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	g := active.geometry
	for py := 0; py < height; py++ {
		z := g.OriginZ + float64(py)/float64(maxInt(height-1, 1))*g.SpanZBlocks
		for px := 0; px < width; px++ {
			x := g.OriginX + float64(px)/float64(maxInt(width-1, 1))*g.SpanXBlocks
			sm := active.sample(x, z)
			img.SetRGBA(px, py, layerColor(layer, sm))
		}
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	if err := png.Encode(w, img); err != nil {
		log.Printf("encode layer: %v", err)
	}
}

func (s *appState) handleSave(w http.ResponseWriter, r *http.Request, modelDir string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req saveRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active == nil || active.model == nil {
		writeError(w, http.StatusConflict, errors.New("only a freshly generated world can be saved"))
		return
	}
	name := sanitizeName(req.Name)
	if name == "" {
		name = fmt.Sprintf("seed-%d-%dx%d", active.seed, active.geometry.Width, active.geometry.Height)
	}
	path := filepath.Join(modelDir, name+".geoworld")
	if _, err := os.Stat(path); err == nil {
		if err := worldmodelio.Replace(path, active.model); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	} else if os.IsNotExist(err) {
		if err := worldmodelio.Save(path, active.model); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	} else {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	info, err := worldmodelio.Inspect(path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "info": info})
}

func (s *appState) handleLoad(w http.ResponseWriter, r *http.Request, modelDir string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req loadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	name := sanitizeName(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, errors.New("model name is required"))
		return
	}
	path := filepath.Join(modelDir, name+".geoworld")
	start := time.Now()
	model, err := worldmodelio.Load(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	active := &activeWorld{
		seed:        model.Manifest.Seed,
		geometry:    model.Geometry,
		sample:      model.RuntimeProfile.Sample,
		riverCount:  len(model.Waterbed.Rivers),
		lakeCount:   len(model.Waterbed.Lakes),
		source:      "reopened",
		reopened:    model,
		generatedAt: time.Now(),
		duration:    time.Since(start),
	}
	s.mu.Lock()
	s.active = active
	s.lastError = ""
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, activeSummary(active))
}

func handleModels(w http.ResponseWriter, r *http.Request, modelDir string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	entries, err := os.ReadDir(modelDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	var models []modelInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".geoworld") {
			continue
		}
		path := filepath.Join(modelDir, entry.Name())
		inspection, err := worldmodelio.Inspect(path)
		if err != nil {
			continue
		}
		models = append(models, modelInfo{
			Name:             strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			FileBytes:        inspection.FileBytes,
			Seed:             inspection.Manifest.Seed,
			SchemaVersion:    inspection.Manifest.SchemaVersion,
			GeneratorVersion: inspection.Manifest.GeneratorVersion,
			Geometry:         inspection.Manifest.Geometry,
		})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].Name < models[j].Name })
	writeJSON(w, http.StatusOK, models)
}

func activeSummary(a *activeWorld) map[string]any {
	return map[string]any{
		"seed":          a.seed,
		"geometry":      a.geometry,
		"riverSegments": a.riverCount,
		"lakeNodes":     a.lakeCount,
		"source":        a.source,
		"durationMs":    a.duration.Milliseconds(),
	}
}

func sampleJSON(x, z float64, s runtimeprofile.Sample) map[string]any {
	return map[string]any{
		"x": x, "z": z,
		"groundHeight":          s.World.GroundHeight,
		"waterKind":             waterKindName(s.World.WaterKind),
		"waterSurfaceDefined":   s.World.WaterSurfaceDefined,
		"waterSurfaceElevation": s.World.WaterSurfaceElevation,
		"waterDepth":            s.World.WaterDepth,
		"oceanEligibility":      s.World.OceanEligibility,
		"riverWeight":           s.World.RiverWeight,
		"ecoRegion":             s.EcoRegion.String(),
		"regionConfidence":      s.RegionConfidence,
		"biomeFamily":           s.BiomeFamily.String(),
		"terrainVariant":        s.TerrainVariant.String(),
		"surfaceProfile":        s.SurfaceProfile.String(),
		"vegetationProfile":     s.VegetationProfile.String(),
		"profileConfidence":     s.ProfileConfidence,
		"sourceNode":            s.SourceNode,
		"sourceWeight":          s.SourceWeight,
		"sourceLand":            s.SourceLand,
		"semanticFallback":      s.SemanticFallback,
		"semanticMismatch":      s.SemanticMismatch,
	}
}

func waterKindName(k reconcile.WaterKind) string {
	switch k {
	case reconcile.WaterOcean:
		return "Ocean"
	case reconcile.WaterLake:
		return "Lake"
	case reconcile.WaterRiver:
		return "River"
	default:
		return "None"
	}
}

func layerColor(layer string, s runtimeprofile.Sample) color.RGBA {
	switch layer {
	case "water":
		switch s.World.WaterKind {
		case reconcile.WaterOcean:
			return color.RGBA{28, 88, 132, 255}
		case reconcile.WaterLake:
			return color.RGBA{44, 138, 174, 255}
		case reconcile.WaterRiver:
			return color.RGBA{66, 158, 205, 255}
		default:
			return terrainHeightColor(s.World.GroundHeight)
		}
	case "biome":
		return biomeColor(s)
	case "profile":
		return profileColor(s)
	default:
		if s.World.WaterKind != reconcile.WaterNone {
			switch s.World.WaterKind {
			case reconcile.WaterOcean:
				return color.RGBA{29, 83, 123, 255}
			case reconcile.WaterLake:
				return color.RGBA{48, 132, 164, 255}
			case reconcile.WaterRiver:
				return color.RGBA{67, 145, 190, 255}
			}
		}
		return terrainHeightColor(s.World.GroundHeight)
	}
}

func terrainHeightColor(v float64) color.RGBA {
	v = clamp(v, 0, 1)
	switch {
	case v < 0.50:
		return lerp(color.RGBA{45, 85, 70, 255}, color.RGBA{92, 142, 83, 255}, v/0.5)
	case v < 0.72:
		return lerp(color.RGBA{92, 142, 83, 255}, color.RGBA{171, 150, 92, 255}, (v-0.5)/0.22)
	case v < 0.88:
		return lerp(color.RGBA{171, 150, 92, 255}, color.RGBA{178, 174, 164, 255}, (v-0.72)/0.16)
	default:
		return lerp(color.RGBA{178, 174, 164, 255}, color.RGBA{239, 240, 237, 255}, (v-0.88)/0.12)
	}
}

func biomeColor(s runtimeprofile.Sample) color.RGBA {
	name := s.BiomeFamily.String()
	switch {
	case strings.Contains(name, "Ocean"):
		return color.RGBA{35, 101, 143, 255}
	case strings.Contains(name, "Rainforest"):
		return color.RGBA{32, 112, 59, 255}
	case strings.Contains(name, "Forest"):
		return color.RGBA{61, 126, 69, 255}
	case strings.Contains(name, "Desert"):
		return color.RGBA{205, 164, 96, 255}
	case strings.Contains(name, "Savanna"):
		return color.RGBA{177, 169, 72, 255}
	case strings.Contains(name, "Grassland"):
		return color.RGBA{145, 159, 79, 255}
	case strings.Contains(name, "Tundra"):
		return color.RGBA{155, 158, 128, 255}
	case strings.Contains(name, "Alpine") || strings.Contains(name, "Ice"):
		return color.RGBA{204, 207, 200, 255}
	case strings.Contains(name, "Wetland"):
		return color.RGBA{55, 139, 126, 255}
	default:
		return color.RGBA{104, 132, 91, 255}
	}
}

func profileColor(s runtimeprofile.Sample) color.RGBA {
	if s.SemanticMismatch || s.SemanticFallback {
		return color.RGBA{220, 67, 67, 255}
	}
	conf := clamp(s.ProfileConfidence, 0, 1)
	return lerp(color.RGBA{86, 77, 91, 255}, color.RGBA{208, 220, 214, 255}, conf)
}

func lerp(a, b color.RGBA, t float64) color.RGBA {
	t = clamp(t, 0, 1)
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func sanitizeName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func parseBoundedInt(raw string, fallback, lo, hi int) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		return fallback
	}
	return n
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func openURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}
