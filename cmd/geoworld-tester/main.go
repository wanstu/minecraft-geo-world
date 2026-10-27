package main

import (
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"io/fs"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"geoworld/acceptance"
	"geoworld/internal/generator"
	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/runtimeworld"
	"geoworld/internal/terrain"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
)

//go:embed web/*
var webFiles embed.FS

type activeWorld struct {
	seed                    int64
	geometry                world.GridGeometry
	sample                  func(float64, float64) runtimeprofile.Sample
	previewGrid             []runtimeprofile.Sample
	mineralPreviewModelPath string
	riverCount              int
	lakeCount               int
	plateCount              int
	oceanFraction           float64
	continentFragmentation  float64
	continentSeparation     float64
	worldPreset             string
	worldProfileVersion     string
	generationRulesVersion  string
	worldModelVersion       string
	parameterSnapshotSHA256 string
	parameterSnapshot       generator.ParameterSnapshot
	landComponents          int
	majorLandComponents     int
	largestLandShare        float64
	finalRiverUpslope       int
	finalLakeBedViolations  int
	finalCoastMismatch      int
	runtimeFallback         int
	runtimeMismatch         int
	anomalies               anomalyReport
	source                  string
	model                   *generator.WorldModel
	reopened                *worldmodelio.RuntimeModel
	modelPath               string
	generatedAt             time.Time
	duration                time.Duration
}

const anomalyPointLimit = 250

const (
	anomalyCoastMismatch   = "coast_mismatch"
	anomalyRiverUpslope    = "river_upslope"
	anomalyLakeBed         = "lake_bed"
	anomalyRuntimeFallback = "runtime_fallback"
	anomalyRuntimeMismatch = "runtime_mismatch"
)

type anomalyPoint struct {
	Kind   string  `json:"kind"`
	Label  string  `json:"label"`
	X      float64 `json:"x"`
	Z      float64 `json:"z"`
	Detail string  `json:"detail,omitempty"`
}

type anomalyReport struct {
	Total     int            `json:"total"`
	Counts    map[string]int `json:"counts"`
	Points    []anomalyPoint `json:"points"`
	Truncated bool           `json:"truncated"`
}

type appState struct {
	mu         sync.RWMutex
	active     *activeWorld
	generating bool
	exporting  bool
	stage      generator.Stage
	completed  int
	total      int
	startedAt  time.Time
	lastError  string
}

type generateRequest struct {
	Seed                   int64   `json:"seed"`
	Width                  int     `json:"width"`
	Height                 int     `json:"height"`
	CellSize               float64 `json:"cellSize"`
	PlateCount             int     `json:"plateCount"`
	OceanFraction          float64 `json:"oceanFraction"`
	ContinentFragmentation float64 `json:"continentFragmentation"`
	ContinentSeparation    float64 `json:"continentSeparation"`
	WorldPreset            string  `json:"worldPreset"`
	Parameters             string  `json:"parameters"`
}

type saveRequest struct {
	Name string `json:"name"`
}

type loadRequest struct {
	Name string `json:"name"`
}

type exportMinecraftRequest struct {
	Name      string `json:"name"`
	Directory string `json:"directory"`
	Radius    *int   `json:"radius"`
	QAMode    *bool  `json:"qaMode"`
	Overwrite bool   `json:"overwrite"`
}

type minecraftExportResult struct {
	MinecraftVersion string `json:"minecraftVersion"`
	Seed             int64  `json:"seed"`
	Output           string `json:"output"`
	SpawnX           int    `json:"spawnX"`
	SpawnY           int    `json:"spawnY"`
	SpawnZ           int    `json:"spawnZ"`
	CenterChunkX     int    `json:"centerChunkX"`
	CenterChunkZ     int    `json:"centerChunkZ"`
	RadiusChunks     int    `json:"radiusChunks"`
	Chunks           int    `json:"chunks"`
	Regions          int    `json:"regions"`
	QAMode           bool   `json:"qaMode"`
}

type openFolderRequest struct {
	Path string `json:"path"`
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
	addr := flag.String("addr", "0.0.0.0:8788", "listen address")
	openBrowser := flag.Bool("open", true, "open browser automatically")
	modelDir := flag.String("models", filepath.FromSlash("output/models"), "saved model directory")
	flag.Parse()

	repoRoot, err := findRepoRoot()
	if err != nil {
		log.Printf("repository integration unavailable: %v", err)
		repoRoot = ""
	}

	state := &appState{}
	if err := os.MkdirAll(*modelDir, 0o755); err != nil {
		log.Fatal(err)
	}

	staticFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	staticHandler := http.FileServer(http.FS(staticFS))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		staticHandler.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/api/status", state.handleStatus)
	mux.HandleFunc("/api/generate", state.handleGenerate)
	mux.HandleFunc("/api/sample", state.handleSample)
	mux.HandleFunc("/api/anomalies", state.handleAnomalies)
	mux.HandleFunc("/api/acceptance", state.handleAcceptance)
	mux.HandleFunc("/api/features", state.handlePreviewFeatures)
	mux.HandleFunc("/api/minerals", state.handleMineralBodies)
	mux.HandleFunc("/api/layer", state.handleLayer)
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) { state.handleSave(w, r, *modelDir) })
	mux.HandleFunc("/api/load", func(w http.ResponseWriter, r *http.Request) { state.handleLoad(w, r, *modelDir) })
	mux.HandleFunc("/api/models", func(w http.ResponseWriter, r *http.Request) { handleModels(w, r, *modelDir) })
	mux.HandleFunc("/api/export-minecraft", func(w http.ResponseWriter, r *http.Request) {
		state.handleExportMinecraft(w, r, repoRoot)
	})
	mux.HandleFunc("/api/open-folder", handleOpenFolder)

	url := browserURL(*addr)
	fmt.Printf("GeoWorld Tester listening: http://%s\n", *addr)
	fmt.Printf("GeoWorld Tester local: %s\n", url)
	fmt.Printf("models: %s\n", *modelDir)
	fmt.Printf("repo: %s\n", repoRoot)
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
		"exporting":  s.exporting,
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
	if strings.TrimSpace(req.WorldPreset) == "" {
		req.WorldPreset = "standard"
	}
	req.WorldPreset = strings.TrimSpace(req.WorldPreset)
	switch req.WorldPreset {
	case "standard", "natural", "civilization", "expansive", "epic", "near_vanilla", "custom":
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unknown world preset: %s", req.WorldPreset))
		return
	}
	if req.WorldPreset == "custom" && strings.TrimSpace(req.Parameters) == "" {
		writeError(w, http.StatusBadRequest, errors.New("custom world preset requires a complete 25-value parameter snapshot"))
		return
	}
	worldProfile := generator.WorldProfileForPreset(req.WorldPreset)
	if strings.TrimSpace(req.Parameters) != "" {
		snapshot, err := generator.ParseParameterSnapshotCSV(req.Parameters)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		worldProfile.Preset = strings.TrimSpace(req.WorldPreset)
		worldProfile.Parameters = snapshot
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
	if s.exporting {
		s.mu.Unlock()
		writeError(w, http.StatusConflict, errors.New("minecraft export already running"))
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
	var model *generator.WorldModel
	var err error
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("generation panic: %v\n%s", recovered, debug.Stack())
				err = fmt.Errorf("generation panic: %v", recovered)
			}
		}()
		model, err = generator.GenerateWorld(r.Context(), generator.Config{
			Seed:                   req.Seed,
			Geometry:               g,
			WorldProfile:           worldProfile,
			PlateCount:             req.PlateCount,
			OceanFraction:          req.OceanFraction,
			ContinentFragmentation: req.ContinentFragmentation,
			ContinentSeparation:    req.ContinentSeparation,
		}, func(p generator.Progress) {
			s.mu.Lock()
			s.stage = p.Stage
			s.completed = p.Completed
			s.total = p.Total
			s.mu.Unlock()
		})
	}()
	duration := time.Since(start)

	if err != nil {
		s.mu.Lock()
		s.generating = false
		s.lastError = err.Error()
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	runtimeWorld, err := runtimeworld.NewGenerated(model)
	if err != nil {
		s.mu.Lock()
		s.generating = false
		s.lastError = err.Error()
		s.mu.Unlock()
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	landComponents, majorLandComponents, largestLandShare := summarizeLand(model.Surface)
	active := &activeWorld{
		seed:                    model.Config.Seed,
		geometry:                model.Geometry,
		sample:                  runtimeWorld.Sample,
		riverCount:              len(model.Waterbed.Rivers),
		lakeCount:               len(model.Waterbed.Lakes),
		plateCount:              model.Config.Phase1.PlateCount,
		oceanFraction:           model.Config.Phase1.OceanFraction,
		continentFragmentation:  model.Config.Phase1.ContinentFragmentation,
		continentSeparation:     model.Config.Phase1.ContinentSeparation,
		worldPreset:             model.Config.WorldProfile.Preset,
		worldProfileVersion:     model.Config.WorldProfile.WorldProfileVersion,
		generationRulesVersion:  model.Config.WorldProfile.GenerationRulesVersion,
		worldModelVersion:       model.Config.WorldProfile.WorldModelVersion,
		parameterSnapshotSHA256: parameterSnapshotSHA256(model.Config.WorldProfile.Parameters),
		parameterSnapshot:       model.Config.WorldProfile.Parameters,
		landComponents:          landComponents,
		majorLandComponents:     majorLandComponents,
		largestLandShare:        largestLandShare,
		source:                  "generated",
		model:                   model,
		generatedAt:             time.Now(),
		duration:                duration,
	}
	active.previewGrid = buildPreviewGrid(active)
	applyAnomalyReport(active, buildAnomalyReport(active))
	s.mu.Lock()
	s.generating = false
	s.lastError = ""
	s.active = active
	s.mu.Unlock()
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

func (s *appState) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active == nil {
		writeError(w, http.StatusConflict, errors.New("generate or load a world first"))
		return
	}
	writeJSON(w, http.StatusOK, active.anomalies)
}

func (s *appState) currentAcceptanceReport() (acceptance.Report, error) {
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active == nil {
		return acceptance.Report{}, errors.New("generate or load a world first")
	}
	if active.reopened != nil {
		return acceptance.Plan(active.reopened)
	}
	if active.model == nil {
		return acceptance.Report{}, errors.New("active world has no acceptance model")
	}

	tempDir, err := os.MkdirTemp("", "geoworld-acceptance-*")
	if err != nil {
		return acceptance.Report{}, err
	}
	defer os.RemoveAll(tempDir)
	path := filepath.Join(tempDir, "active.geoworld")
	if err := worldmodelio.Save(path, active.model); err != nil {
		return acceptance.Report{}, err
	}
	reopened, err := worldmodelio.Load(path)
	if err != nil {
		return acceptance.Report{}, err
	}
	return acceptance.Plan(reopened)
}

func (s *appState) handleAcceptance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	report, err := s.currentAcceptanceReport()
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	full, err := acceptance.BuildManualTable(report)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	tableType := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("type")))
	if tableType == "" {
		tableType = "spot"
	}
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "json"
	}
	download := r.URL.Query().Get("download") == "1"

	if tableType == "full" {
		if download {
			w.Header().Set(
				"Content-Disposition",
				fmt.Sprintf("attachment; filename=geoworld-full-acceptance-seed-%d.%s", report.Seed, acceptanceExtension(format)),
			)
		}
		switch format {
		case "json":
			data, marshalErr := acceptance.ManualTableJSON(full)
			if marshalErr != nil {
				writeError(w, http.StatusInternalServerError, marshalErr)
				return
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(data)
		case "markdown", "md":
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			_, _ = io.WriteString(w, acceptance.ManualTableMarkdown(full))
		case "csv":
			w.Header().Set("Content-Type", "text/csv; charset=utf-8")
			_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
			if err := acceptance.WriteManualTableCSV(w, full); err != nil {
				log.Printf("write full acceptance csv: %v", err)
			}
		default:
			writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported acceptance format %q", format))
		}
		return
	}

	if tableType != "spot" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported acceptance type %q", tableType))
		return
	}
	perBig := parseBoundedInt(r.URL.Query().Get("perBig"), 2, 1, 12)
	spot, err := acceptance.BuildSpotTable(full, perBig)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if download {
		w.Header().Set(
			"Content-Disposition",
			fmt.Sprintf("attachment; filename=geoworld-spot-acceptance-seed-%d.%s", report.Seed, acceptanceExtension(format)),
		)
	}
	switch format {
	case "json":
		data, marshalErr := acceptance.SpotTableJSON(spot)
		if marshalErr != nil {
			writeError(w, http.StatusInternalServerError, marshalErr)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(data)
	case "markdown", "md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_, _ = io.WriteString(w, acceptance.SpotTableMarkdown(spot))
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})
		if err := acceptance.WriteSpotTableCSV(w, spot); err != nil {
			log.Printf("write spot acceptance csv: %v", err)
		}
	default:
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported acceptance format %q", format))
	}
}

func acceptanceExtension(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "markdown", "md":
		return "md"
	case "csv":
		return "csv"
	default:
		return "json"
	}
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
	width := parseBoundedInt(r.URL.Query().Get("width"), 1200, 160, 2400)
	height := parseBoundedInt(r.URL.Query().Get("height"), 750, 100, 1600)

	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active == nil {
		writeError(w, http.StatusConflict, errors.New("generate or load a world first"))
		return
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if layer == "tectonic" {
		renderTectonicPreviewLayer(img, active)
	} else if layer == "skeleton" || layer == "rivers" || layer == "lakes" || layer == "anomalies" {
		if err := renderDiagnosticLayer(img, active, layer); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
	} else if layer == "raw" || layer == "coast" {
		if err := renderIntermediateLayer(img, active, layer); err != nil {
			writeError(w, http.StatusConflict, err)
			return
		}
	} else {
		g := active.geometry
		for py := 0; py < height; py++ {
			z := g.OriginZ + float64(py)/float64(maxInt(height-1, 1))*g.SpanZBlocks
			for px := 0; px < width; px++ {
				x := g.OriginX + float64(px)/float64(maxInt(width-1, 1))*g.SpanXBlocks
				sm := previewSampleAt(active, x, z)
				img.SetRGBA(px, py, layerColor(layer, sm))
			}
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
	s.mu.Lock()
	if s.active == active {
		active.modelPath = path
	}
	s.mu.Unlock()
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
	landComponents, majorLandComponents, largestLandShare := summarizeLand(model.Surface)
	fragmentation := model.Config.Phase1.ContinentFragmentation
	if fragmentation == 0 {
		fragmentation = phase1.DefaultContinentFragmentation
	}
	separation := model.Config.Phase1.ContinentSeparation
	if separation == 0 {
		separation = phase1.DefaultContinentSeparation
	}
	loadedPreset := model.Manifest.Preset
	if loadedPreset == "" {
		loadedPreset = "legacy"
	}
	loadedProfileVersion := model.Manifest.WorldProfileVersion
	if loadedProfileVersion == "" {
		loadedProfileVersion = "legacy-rc1"
	}
	loadedRulesVersion := model.Manifest.GenerationRulesVersion
	if loadedRulesVersion == "" {
		loadedRulesVersion = "legacy-rc1"
	}
	loadedModelVersion := model.Manifest.WorldModelVersion
	if loadedModelVersion == "" {
		loadedModelVersion = "legacy-model-v1"
	}
	runtimeWorld, err := runtimeworld.NewReopened(model)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	active := &activeWorld{
		seed:                    model.Manifest.Seed,
		geometry:                model.Geometry,
		sample:                  runtimeWorld.Sample,
		riverCount:              len(model.Waterbed.Rivers),
		lakeCount:               len(model.Waterbed.Lakes),
		plateCount:              model.Config.Phase1.PlateCount,
		oceanFraction:           model.Config.Phase1.OceanFraction,
		continentFragmentation:  fragmentation,
		continentSeparation:     separation,
		worldPreset:             loadedPreset,
		worldProfileVersion:     loadedProfileVersion,
		generationRulesVersion:  loadedRulesVersion,
		worldModelVersion:       loadedModelVersion,
		parameterSnapshotSHA256: model.Manifest.ParameterSnapshotSHA256,
		parameterSnapshot:       model.Config.WorldProfile.Parameters,
		landComponents:          landComponents,
		majorLandComponents:     majorLandComponents,
		largestLandShare:        largestLandShare,
		source:                  "reopened",
		reopened:                model,
		modelPath:               path,
		generatedAt:             time.Now(),
		duration:                time.Since(start),
	}
	active.previewGrid = buildPreviewGrid(active)
	applyAnomalyReport(active, buildAnomalyReport(active))
	active.duration = time.Since(start)
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

func (s *appState) handleExportMinecraft(w http.ResponseWriter, r *http.Request, repoRoot string) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	if strings.TrimSpace(repoRoot) == "" {
		writeError(w, http.StatusNotImplemented, errors.New(
			"Minecraft QA export is only available when the GeoWorld source repository is present; preview and acceptance remain fully available in standalone mode",
		))
		return
	}

	var req exportMinecraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	radius := 8
	if req.Radius != nil {
		radius = *req.Radius
	}
	if radius < 0 || radius > 31 {
		writeError(w, http.StatusBadRequest, errors.New("radius must be in 0..31 chunks"))
		return
	}
	qaMode := true
	if req.QAMode != nil {
		qaMode = *req.QAMode
	}

	s.mu.Lock()
	if s.generating {
		s.mu.Unlock()
		writeError(w, http.StatusConflict, errors.New("generation already running"))
		return
	}
	if s.exporting {
		s.mu.Unlock()
		writeError(w, http.StatusConflict, errors.New("minecraft export already running"))
		return
	}
	active := s.active
	if active == nil {
		s.mu.Unlock()
		writeError(w, http.StatusConflict, errors.New("generate or load a world first"))
		return
	}
	s.exporting = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.exporting = false
		s.mu.Unlock()
	}()

	worldName := strings.TrimSpace(req.Name)
	if worldName == "" {
		worldName = fmt.Sprintf("GeoWorld %d QA", active.seed)
	}
	folderName := sanitizeFolderName(worldName)
	if folderName == "" {
		folderName = fmt.Sprintf("GeoWorld-%d", active.seed)
	}

	parent := strings.TrimSpace(req.Directory)
	if parent == "" {
		parent = filepath.Join("output", "minecraft-exports")
	}
	if !filepath.IsAbs(parent) {
		parent = filepath.Join(repoRoot, parent)
	}
	parent, err := filepath.Abs(parent)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	target := filepath.Join(parent, folderName)
	if !req.Overwrite {
		if _, err := os.Stat(target); err == nil {
			writeError(w, http.StatusConflict, fmt.Errorf("export target already exists: %s", target))
			return
		} else if !os.IsNotExist(err) {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}

	modelPath := active.modelPath
	var tempModelDir string
	if modelPath != "" {
		if abs, err := filepath.Abs(modelPath); err == nil {
			modelPath = abs
		}
		if _, err := os.Stat(modelPath); err != nil {
			modelPath = ""
		}
	}
	if modelPath == "" {
		if active.model == nil {
			writeError(w, http.StatusConflict, errors.New("active reopened model has no source file"))
			return
		}
		tempModelDir, err = os.MkdirTemp("", "geoworld-tester-export-model-*")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		defer os.RemoveAll(tempModelDir)
		modelPath = filepath.Join(tempModelDir, "active.geoworld")
		if err := worldmodelio.Save(modelPath, active.model); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("prepare export model: %w", err))
			return
		}
	}

	resultDir, err := os.MkdirTemp("", "geoworld-tester-export-result-*")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer os.RemoveAll(resultDir)
	resultPath := filepath.Join(resultDir, "result.json")

	script := filepath.Join(repoRoot, "java", "minecraft-save-writer", "generate-preview-world.ps1")
	if _, err := os.Stat(script); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("minecraft writer not found: %w", err))
		return
	}

	args := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", script,
		"--model", modelPath,
		"--output", target,
		"--name", worldName,
		"--radius", strconv.Itoa(radius),
		"--result-json", resultPath,
	}
	if qaMode {
		args = append(args, "--qa")
	} else {
		args = append(args, "--survival")
	}
	if req.Overwrite {
		args = append(args, "--force")
	}

	cmd := exec.CommandContext(r.Context(), powershellCommand(), args...)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		writeError(w, http.StatusInternalServerError, fmt.Errorf("minecraft export failed: %s", message))
		return
	}

	resultBytes, err := os.ReadFile(resultPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("read export result: %w", err))
		return
	}
	var result minecraftExportResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("decode export result: %w", err))
		return
	}
	if result.Output == "" {
		writeError(w, http.StatusInternalServerError, errors.New("minecraft export returned empty output path"))
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func handleOpenFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var req openFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	path := strings.TrimSpace(req.Path)
	if path == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !info.IsDir() {
		path = filepath.Dir(path)
	}
	if err := openFolder(path); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"path": path})
}

func activeSummary(a *activeWorld) map[string]any {
	summary := map[string]any{
		"seed":                    a.seed,
		"geometry":                a.geometry,
		"riverSegments":           a.riverCount,
		"lakeNodes":               a.lakeCount,
		"plateCount":              a.plateCount,
		"oceanFraction":           a.oceanFraction,
		"continentFragmentation":  a.continentFragmentation,
		"continentSeparation":     a.continentSeparation,
		"worldPreset":             a.worldPreset,
		"worldProfileVersion":     a.worldProfileVersion,
		"generationRulesVersion":  a.generationRulesVersion,
		"worldModelVersion":       a.worldModelVersion,
		"parameterSnapshotSha256": a.parameterSnapshotSHA256,
		"parameterSnapshot":       a.parameterSnapshot,
		"parameterCsv":            parameterSnapshotCSV(a.parameterSnapshot),
		"landComponents":          a.landComponents,
		"majorLandComponents":     a.majorLandComponents,
		"largestLandShare":        a.largestLandShare,
		"finalCoastMismatch":      a.finalCoastMismatch,
		"finalRiverUpslope":       a.finalRiverUpslope,
		"finalLakeBedViolations":  a.finalLakeBedViolations,
		"runtimeFallback":         a.runtimeFallback,
		"runtimeMismatch":         a.runtimeMismatch,
		"anomalyTotal":            a.anomalies.Total,
		"anomalyTruncated":        a.anomalies.Truncated,
		"source":                  a.source,
		"durationMs":              a.duration.Milliseconds(),
		"intermediateDiagnostics": false,
	}
	if a.model != nil && a.model.Terrain != nil && a.model.Reconciliation != nil {
		r := a.model.Reconciliation
		summary["intermediateDiagnostics"] = true
		summary["coastAdjustedNodes"] = r.CoastAdjustedNodes
		summary["coastRaisedNodes"] = r.CoastRaisedNodes
		summary["coastCarvedNodes"] = r.CoastCarvedNodes
		summary["surfaceMismatchBefore"] = r.SurfaceMismatchBefore
		summary["surfaceMismatchAfter"] = r.SurfaceMismatchAfter
		summary["appliedCoastSurfaceMismatch"] = r.AppliedCoastSurfaceMismatch
		summary["rawUpslopeRiverEdges"] = r.RawUpslopeRiverEdges
		summary["targetUpslopeRiverEdges"] = r.TargetUpslopeRiverEdges
		summary["lakeBedViolations"] = r.LakeBedViolations
	}
	return summary
}

func summarizeLand(surface *world.Surface) (components, major int, largestShare float64) {
	if surface == nil || surface.Width <= 0 || surface.Height <= 0 || len(surface.LandMask) == 0 {
		return 0, 0, 0
	}
	seen := make([]bool, len(surface.LandMask))
	queue := make([]int, 0, 256)
	totalLand := 0
	for _, land := range surface.LandMask {
		if land {
			totalLand++
		}
	}
	if totalLand == 0 {
		return 0, 0, 0
	}
	majorThreshold := totalLand / 50
	if majorThreshold < 16 {
		majorThreshold = 16
	}
	largest := 0
	for start, land := range surface.LandMask {
		if !land || seen[start] {
			continue
		}
		components++
		seen[start] = true
		queue = queue[:0]
		queue = append(queue, start)
		size := 0
		for head := 0; head < len(queue); head++ {
			i := queue[head]
			size++
			x, z := i%surface.Width, i/surface.Width
			if x > 0 {
				n := i - 1
				if surface.LandMask[n] && !seen[n] {
					seen[n] = true
					queue = append(queue, n)
				}
			}
			if x+1 < surface.Width {
				n := i + 1
				if surface.LandMask[n] && !seen[n] {
					seen[n] = true
					queue = append(queue, n)
				}
			}
			if z > 0 {
				n := i - surface.Width
				if surface.LandMask[n] && !seen[n] {
					seen[n] = true
					queue = append(queue, n)
				}
			}
			if z+1 < surface.Height {
				n := i + surface.Width
				if surface.LandMask[n] && !seen[n] {
					seen[n] = true
					queue = append(queue, n)
				}
			}
		}
		if size >= majorThreshold {
			major++
		}
		if size > largest {
			largest = size
		}
	}
	return components, major, float64(largest) / float64(totalLand)
}

func summarizeRuntimeSemantics(g world.GridGeometry, sample func(float64, float64) runtimeprofile.Sample) (fallback, mismatch int) {
	if sample == nil {
		return 0, 0
	}
	for z := 0; z < g.Height; z++ {
		for x := 0; x < g.Width; x++ {
			wx, wz := g.NodePosition(x, z)
			s := sample(wx, wz)
			if s.SemanticFallback {
				fallback++
			}
			if s.SemanticMismatch {
				mismatch++
			}
		}
	}
	return fallback, mismatch
}

func applyAnomalyReport(active *activeWorld, report anomalyReport) {
	active.anomalies = report
	active.finalCoastMismatch = report.Counts[anomalyCoastMismatch]
	active.finalRiverUpslope = report.Counts[anomalyRiverUpslope]
	active.finalLakeBedViolations = report.Counts[anomalyLakeBed]
	active.runtimeFallback = report.Counts[anomalyRuntimeFallback]
	active.runtimeMismatch = report.Counts[anomalyRuntimeMismatch]
}

func buildAnomalyReport(active *activeWorld) anomalyReport {
	report := anomalyReport{Counts: map[string]int{
		anomalyCoastMismatch:   0,
		anomalyRiverUpslope:    0,
		anomalyLakeBed:         0,
		anomalyRuntimeFallback: 0,
		anomalyRuntimeMismatch: 0,
	}}
	if active == nil {
		return report
	}
	add := func(point anomalyPoint) {
		report.Counts[point.Kind]++
		report.Total++
		if len(report.Points) < anomalyPointLimit {
			report.Points = append(report.Points, point)
		} else {
			report.Truncated = true
		}
	}

	surface := activeSurface(active)
	coast := activeCoastTerrain(active)
	if surface != nil && coast != nil && surface.Width == active.geometry.Width && surface.Height == active.geometry.Height {
		heights := make([]float64, surface.Width*surface.Height)
		for z := 0; z < surface.Height; z++ {
			for x := 0; x < surface.Width; x++ {
				wx, wz := active.geometry.NodePosition(x, z)
				heights[z*surface.Width+x] = coast.SampleNormalized(wx, wz)
			}
		}
		if classified, err := world.ClassifySurface(surface.Width, surface.Height, heights, surface.SeaLevelNormalized); err == nil {
			for i, land := range classified.LandMask {
				if land == surface.LandMask[i] {
					continue
				}
				x, z := i%surface.Width, i/surface.Width
				wx, wz := active.geometry.NodePosition(x, z)
				add(anomalyPoint{
					Kind: anomalyCoastMismatch, Label: "最终海岸冲突",
					X: wx, Z: wz,
					Detail: fmt.Sprintf("Surface=%t CoastTerrain=%t", surface.LandMask[i], land),
				})
			}
		}
	}

	wb := activeWaterbed(active)
	if wb != nil {
		epsilon := activeFinalUpslopeEpsilon(active)
		for segID, seg := range wb.Rivers {
			steps := 16
			if profileSteps := 2 * (len(seg.Profile) - 1); profileSteps > steps {
				steps = profileSteps
			}
			prevOwn := diagnosticRiverProfile(seg, 0)
			for j := 1; j <= steps; j++ {
				t := float64(j) / float64(steps)
				x, z := diagnosticRiverCenter(seg, t)
				own := diagnosticRiverProfile(seg, t)
				rise := own - prevOwn
				if rise > epsilon {
					add(anomalyPoint{
						Kind: anomalyRiverUpslope, Label: "最终河床逆坡",
						X: x, Z: z,
						Detail: fmt.Sprintf("segment=%d rise=%.8f", segID, rise),
					})
				}
				prevOwn = own
			}
		}
		for _, lake := range wb.Lakes {
			// Hydrology lake nodes are macro depression candidates. Detailed
			// terrain owns the final shoreline, so a candidate that Waterbed
			// intentionally leaves dry must not be reported as a lake-bed error.
			ground := wb.Sample(lake.X, lake.Z)
			if ground.WaterKind != reconcile.WaterLake ||
				!ground.WaterSurfaceDefined {
				continue
			}
			excess := ground.FinalHeight - lake.BedCeiling
			if excess > epsilon {
				add(anomalyPoint{
					Kind: anomalyLakeBed, Label: "最终湖床违规",
					X: lake.X, Z: lake.Z,
					Detail: fmt.Sprintf("lake=%d excess=%.8f", lake.LakeID, excess),
				})
			}
		}
	}

	if active.sample != nil {
		for z := 0; z < active.geometry.Height; z++ {
			for x := 0; x < active.geometry.Width; x++ {
				wx, wz := active.geometry.NodePosition(x, z)
				s := active.sample(wx, wz)
				if s.SemanticFallback {
					add(anomalyPoint{
						Kind: anomalyRuntimeFallback, Label: "Runtime 节点回退",
						X: wx, Z: wz,
						Detail: fmt.Sprintf("sourceNode=%d", s.SourceNode),
					})
				}
				if s.SemanticMismatch {
					add(anomalyPoint{
						Kind: anomalyRuntimeMismatch, Label: "Runtime 节点冲突",
						X: wx, Z: wz,
						Detail: fmt.Sprintf("sourceNode=%d", s.SourceNode),
					})
				}
			}
		}
	}
	return report
}

func activeFinalUpslopeEpsilon(active *activeWorld) float64 {
	if active == nil {
		return 0
	}
	if active.model != nil && active.model.Reconciliation != nil {
		return math.Max(0, active.model.Reconciliation.Config.FinalUpslopeEpsilon)
	}
	if active.reopened != nil {
		return math.Max(0, active.reopened.Config.Reconcile.FinalUpslopeEpsilon)
	}
	return 0
}

func activeCoastTerrain(active *activeWorld) *terrain.Model {
	if active == nil {
		return nil
	}
	if active.model != nil {
		return active.model.CoastTerrain
	}
	if active.reopened != nil {
		return active.reopened.CoastTerrain
	}
	return nil
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

func renderIntermediateLayer(img *image.RGBA, active *activeWorld, layer string) error {
	if active == nil || active.model == nil || active.model.Terrain == nil || active.model.Reconciliation == nil {
		return errors.New("this diagnostic requires a freshly generated world")
	}
	switch layer {
	case "raw":
		g := active.geometry
		b := img.Bounds()
		for py := 0; py < b.Dy(); py++ {
			z := g.OriginZ + float64(py)/float64(maxInt(b.Dy()-1, 1))*g.SpanZBlocks
			for px := 0; px < b.Dx(); px++ {
				x := g.OriginX + float64(px)/float64(maxInt(b.Dx()-1, 1))*g.SpanXBlocks
				img.SetRGBA(px, py, terrainHeightColor(active.model.Terrain.SampleNormalized(x, z)))
			}
		}
		return nil
	case "coast":
		return renderCoastConstraintLayer(img, active)
	default:
		return errors.New("unknown intermediate diagnostic layer")
	}
}

func renderCoastConstraintLayer(img *image.RGBA, active *activeWorld) error {
	r := active.model.Reconciliation
	target := active.model.Surface
	if target == nil || r.RawTerrainSurface == nil || r.AppliedCoastSurface == nil {
		return errors.New("coast diagnostics are unavailable for this world")
	}
	n := target.Width * target.Height
	if len(r.CoastCorrection) != n ||
		len(r.RawTerrainSurface.LandMask) != n ||
		len(r.AppliedCoastSurface.LandMask) != n {
		return errors.New("coast diagnostic fields do not match the world geometry")
	}
	b := img.Bounds()
	for py := 0; py < b.Dy(); py++ {
		gz := int(math.Round(float64(py) / float64(maxInt(b.Dy()-1, 1)) * float64(target.Height-1)))
		for px := 0; px < b.Dx(); px++ {
			gx := int(math.Round(float64(px) / float64(maxInt(b.Dx()-1, 1)) * float64(target.Width-1)))
			i := gz*target.Width + gx
			rawMismatch := r.RawTerrainSurface.LandMask[i] != target.LandMask[i]
			finalMismatch := r.AppliedCoastSurface.LandMask[i] != target.LandMask[i]
			correction := r.CoastCorrection[i]
			var c color.RGBA
			switch {
			case finalMismatch:
				c = color.RGBA{229, 72, 168, 255}
			case rawMismatch:
				c = color.RGBA{238, 142, 67, 255}
			case correction > 1e-15:
				c = color.RGBA{221, 195, 84, 255}
			case correction < -1e-15:
				c = color.RGBA{89, 178, 211, 255}
			default:
				c = diagnosticSurfaceColor(target, gx, gz, false)
			}
			img.SetRGBA(px, py, c)
		}
	}
	return nil
}

func renderDiagnosticLayer(img *image.RGBA, active *activeWorld, layer string) error {
	surface := activeSurface(active)
	if surface == nil || surface.Width <= 0 || surface.Height <= 0 || len(surface.LandMask) != surface.Width*surface.Height {
		return errors.New("surface diagnostics are unavailable for this world")
	}
	b := img.Bounds()
	for py := 0; py < b.Dy(); py++ {
		gz := int(math.Round(float64(py) / float64(maxInt(b.Dy()-1, 1)) * float64(surface.Height-1)))
		for px := 0; px < b.Dx(); px++ {
			gx := int(math.Round(float64(px) / float64(maxInt(b.Dx()-1, 1)) * float64(surface.Width-1)))
			if layer == "anomalies" {
				img.SetRGBA(px, py, anomalyBaseColor(surface, gx, gz))
			} else {
				img.SetRGBA(px, py, diagnosticSurfaceColor(surface, gx, gz, layer == "skeleton"))
			}
		}
	}

	wb := activeWaterbed(active)
	switch layer {
	case "rivers":
		if wb == nil {
			return errors.New("river diagnostics are unavailable for this world")
		}
		for _, segment := range wb.Rivers {
			radius := diagnosticPhysicalRadius(
				active.geometry,
				b,
				segment.HalfWidth,
				1,
				3,
			)
			const curveSteps = 12
			prevX, prevZ := diagnosticRiverCenter(segment, 0)
			px0, py0 := diagnosticWorldToPixel(
				active.geometry,
				b,
				prevX,
				prevZ,
			)
			for j := 1; j <= curveSteps; j++ {
				t := float64(j) / curveSteps
				x, z := diagnosticRiverCenter(segment, t)
				px1, py1 := diagnosticWorldToPixel(
					active.geometry,
					b,
					x,
					z,
				)
				drawDiagnosticLine(
					img,
					px0,
					py0,
					px1,
					py1,
					radius,
					color.RGBA{70, 188, 235, 255},
				)
				px0, py0 = px1, py1
			}
		}
	case "lakes":
		if wb == nil {
			return errors.New("lake diagnostics are unavailable for this world")
		}
		for _, lake := range wb.Lakes {
			x, y := diagnosticWorldToPixel(active.geometry, b, lake.X, lake.Z)
			radius := diagnosticPhysicalRadius(active.geometry, b, lake.OuterRadius, 2, 8)
			drawDiagnosticDisc(img, x, y, radius, color.RGBA{73, 169, 205, 255})
			drawDiagnosticDisc(img, x, y, maxInt(1, radius/2), color.RGBA{115, 211, 238, 255})
		}
	case "anomalies":
		for _, point := range active.anomalies.Points {
			x, y := diagnosticWorldToPixel(active.geometry, b, point.X, point.Z)
			c := anomalyColor(point.Kind)
			drawDiagnosticDisc(img, x, y, 5, color.RGBA{20, 22, 24, 255})
			drawDiagnosticDisc(img, x, y, 3, c)
		}
	}
	return nil
}

func anomalyBaseColor(surface *world.Surface, x, z int) color.RGBA {
	if surface.LandMask[z*surface.Width+x] {
		return color.RGBA{66, 84, 73, 255}
	}
	return color.RGBA{27, 58, 76, 255}
}

func anomalyColor(kind string) color.RGBA {
	switch kind {
	case anomalyCoastMismatch:
		return color.RGBA{239, 76, 94, 255}
	case anomalyRiverUpslope:
		return color.RGBA{242, 151, 61, 255}
	case anomalyLakeBed:
		return color.RGBA{66, 197, 226, 255}
	case anomalyRuntimeFallback:
		return color.RGBA{235, 205, 75, 255}
	case anomalyRuntimeMismatch:
		return color.RGBA{219, 82, 190, 255}
	default:
		return color.RGBA{240, 240, 240, 255}
	}
}

func activeSurface(active *activeWorld) *world.Surface {
	if active == nil {
		return nil
	}
	if active.model != nil {
		return active.model.Surface
	}
	if active.reopened != nil {
		return active.reopened.Surface
	}
	return nil
}

func activeWaterbed(active *activeWorld) *waterbed.Model {
	if active == nil {
		return nil
	}
	if active.model != nil {
		return active.model.Waterbed
	}
	if active.reopened != nil {
		return active.reopened.Waterbed
	}
	return nil
}

func diagnosticSurfaceColor(surface *world.Surface, x, z int, emphasizeCoast bool) color.RGBA {
	i := z*surface.Width + x
	land := surface.LandMask[i]
	if emphasizeCoast && diagnosticCoastNode(surface, x, z) {
		if land {
			return color.RGBA{232, 201, 132, 255}
		}
		return color.RGBA{110, 164, 186, 255}
	}
	if surface.EnclosedBelowSea != nil && i < len(surface.EnclosedBelowSea) && surface.EnclosedBelowSea[i] {
		return color.RGBA{151, 112, 76, 255}
	}
	if land {
		return color.RGBA{79, 117, 75, 255}
	}
	return color.RGBA{27, 69, 96, 255}
}

func diagnosticCoastNode(surface *world.Surface, x, z int) bool {
	i := z*surface.Width + x
	land := surface.LandMask[i]
	if x > 0 && surface.LandMask[i-1] != land {
		return true
	}
	if x+1 < surface.Width && surface.LandMask[i+1] != land {
		return true
	}
	if z > 0 && surface.LandMask[i-surface.Width] != land {
		return true
	}
	if z+1 < surface.Height && surface.LandMask[i+surface.Width] != land {
		return true
	}
	return false
}

func diagnosticRiverProfile(
	segment waterbed.RiverSegment,
	t float64,
) float64 {
	t = math.Max(0, math.Min(1, t))
	if len(segment.Profile) == 0 {
		return segment.Bed0 + (segment.Bed1-segment.Bed0)*t
	}
	if len(segment.Profile) == 1 {
		return segment.Profile[0]
	}
	pos := t * float64(len(segment.Profile)-1)
	i0 := int(math.Floor(pos))
	i1 := i0 + 1
	if i1 >= len(segment.Profile) {
		i1 = len(segment.Profile) - 1
	}
	f := pos - float64(i0)
	return segment.Profile[i0] +
		(segment.Profile[i1]-segment.Profile[i0])*f
}

func diagnosticRiverCenter(
	segment waterbed.RiverSegment,
	t float64,
) (float64, float64) {
	return waterbed.RiverCenterPoint(segment, t)
}

func diagnosticWorldToPixel(g world.GridGeometry, b image.Rectangle, x, z float64) (int, int) {
	fx, fz := 0.5, 0.5
	if g.SpanXBlocks > 0 {
		fx = (x - g.OriginX) / g.SpanXBlocks
	}
	if g.SpanZBlocks > 0 {
		fz = (z - g.OriginZ) / g.SpanZBlocks
	}
	fx = clamp(fx, 0, 1)
	fz = clamp(fz, 0, 1)
	return int(math.Round(fx * float64(maxInt(b.Dx()-1, 0)))), int(math.Round(fz * float64(maxInt(b.Dy()-1, 0))))
}

func diagnosticPhysicalRadius(g world.GridGeometry, b image.Rectangle, blocks float64, lo, hi int) int {
	scale := 0.0
	if g.SpanXBlocks > 0 {
		scale = float64(maxInt(b.Dx()-1, 1)) / g.SpanXBlocks
	}
	if g.SpanZBlocks > 0 {
		zScale := float64(maxInt(b.Dy()-1, 1)) / g.SpanZBlocks
		if scale == 0 || zScale < scale {
			scale = zScale
		}
	}
	r := int(math.Round(blocks * scale))
	if r < lo {
		r = lo
	}
	if r > hi {
		r = hi
	}
	return r
}

func drawDiagnosticLine(img *image.RGBA, x0, y0, x1, y1, radius int, c color.RGBA) {
	dx := absInt(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -absInt(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		drawDiagnosticDisc(img, x0, y0, radius, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func drawDiagnosticDisc(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	b := img.Bounds()
	r2 := radius * radius
	for y := cy - radius; y <= cy+radius; y++ {
		if y < b.Min.Y || y >= b.Max.Y {
			continue
		}
		for x := cx - radius; x <= cx+radius; x++ {
			if x < b.Min.X || x >= b.Max.X {
				continue
			}
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
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
	case "landform":
		return landformColor(s)
	case "surface":
		return surfaceColor(s)
	case "minerals":
		return mineralProvinceColor(s)
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

func landformColor(s runtimeprofile.Sample) color.RGBA {
	if s.World.WaterKind == reconcile.WaterOcean {
		return color.RGBA{34, 86, 123, 255}
	}
	switch s.TerrainVariant {
	case mcprofile.TerrainPlain:
		return color.RGBA{127, 166, 91, 255}
	case mcprofile.TerrainRollingPlain:
		return color.RGBA{150, 169, 88, 255}
	case mcprofile.TerrainHills:
		return color.RGBA{167, 143, 79, 255}
	case mcprofile.TerrainFoothills:
		return color.RGBA{157, 120, 83, 255}
	case mcprofile.TerrainMountain:
		return color.RGBA{122, 111, 107, 255}
	case mcprofile.TerrainPlateau:
		return color.RGBA{177, 132, 87, 255}
	case mcprofile.TerrainFloodplain:
		return color.RGBA{93, 151, 105, 255}
	case mcprofile.TerrainAlluvialPlain:
		return color.RGBA{113, 159, 112, 255}
	case mcprofile.TerrainAlluvialFan:
		return color.RGBA{160, 149, 101, 255}
	case mcprofile.TerrainCanyon:
		return color.RGBA{143, 91, 73, 255}
	default:
		return color.RGBA{104, 119, 98, 255}
	}
}

func surfaceColor(s runtimeprofile.Sample) color.RGBA {
	switch s.SurfaceProfile {
	case mcprofile.SurfaceOceanSand:
		return color.RGBA{62, 126, 156, 255}
	case mcprofile.SurfaceOceanGravel:
		return color.RGBA{67, 101, 127, 255}
	case mcprofile.SurfaceFrozenOcean:
		return color.RGBA{176, 207, 218, 255}
	case mcprofile.SurfaceIceSnow:
		return color.RGBA{235, 240, 239, 255}
	case mcprofile.SurfaceTundraSoil:
		return color.RGBA{159, 159, 132, 255}
	case mcprofile.SurfaceAlpineRock:
		return color.RGBA{144, 143, 139, 255}
	case mcprofile.SurfaceAlpineSnow:
		return color.RGBA{244, 245, 242, 255}
	case mcprofile.SurfaceForestLoam:
		return color.RGBA{92, 119, 74, 255}
	case mcprofile.SurfacePodzol:
		return color.RGBA{89, 92, 65, 255}
	case mcprofile.SurfaceGrassLoam:
		return color.RGBA{124, 153, 78, 255}
	case mcprofile.SurfaceDryGrass:
		return color.RGBA{166, 151, 77, 255}
	case mcprofile.SurfaceShrubSoil:
		return color.RGBA{150, 126, 84, 255}
	case mcprofile.SurfaceDesertSand:
		return color.RGBA{218, 190, 116, 255}
	case mcprofile.SurfaceDesertGravel:
		// Keep legacy gravel visually distinct so old models are easy to spot.
		return color.RGBA{174, 157, 132, 255}
	case mcprofile.SurfaceAlluvialSilt:
		return color.RGBA{139, 132, 91, 255}
	case mcprofile.SurfaceWetMud:
		return color.RGBA{92, 105, 76, 255}
	case mcprofile.SurfacePeat:
		return color.RGBA{72, 82, 64, 255}
	default:
		return color.RGBA{110, 119, 101, 255}
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

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		goMod := filepath.Join(dir, "go.mod")
		writer := filepath.Join(dir, "java", "minecraft-save-writer", "generate-preview-world.ps1")
		if _, err := os.Stat(goMod); err == nil {
			if _, err := os.Stat(writer); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("cannot locate GeoWorld repository root")
}

func powershellCommand() string {
	if runtime.GOOS == "windows" {
		return "powershell.exe"
	}
	return "pwsh"
}

func openFolder(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func sanitizeFolderName(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range v {
		switch {
		case r < 32:
			b.WriteRune('_')
		case strings.ContainsRune(`<>:"/\|?*`, r):
			b.WriteRune('_')
		default:
			b.WriteRune(r)
		}
	}
	name := strings.TrimRight(strings.TrimSpace(b.String()), ". ")
	if name == "" {
		return ""
	}
	upper := strings.ToUpper(name)
	switch upper {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return "_" + name
	}
	return name
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

func parameterSnapshotCSV(snapshot generator.ParameterSnapshot) string {
	values := []float64{
		snapshot.MainlandScale, snapshot.RegionScale, snapshot.TransitionWidth,
		snapshot.MountainStrength, snapshot.PlainContinuity, snapshot.PlateauScale,
		snapshot.CliffRarity, snapshot.RareMacroTerrain, snapshot.RiverDensity,
		snapshot.MainRiverScale, snapshot.LakeAbundance, snapshot.WetlandAbundance,
		snapshot.ClimateRegionScale, snapshot.ClimateTransitionWidth,
		snapshot.RainShadowStrength, snapshot.BiomeContinuity, snapshot.CaveRichness,
		snapshot.LargeCaveRarity, snapshot.MineralRegionalization,
		snapshot.SpawnPreference, snapshot.StructureInfluence,
		snapshot.OuterIslandAbundance, snapshot.PolarTransitionWidth,
		snapshot.OceanClimateStrength, snapshot.PrevailingWindStrength,
	}
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.FormatFloat(value, 'f', 6, 64)
	}
	return strings.Join(parts, ",")
}

func parameterSnapshotSHA256(snapshot generator.ParameterSnapshot) string {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return fmt.Sprintf("%x", sum[:])
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

func browserURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://" + addr
	}
	switch host {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port)
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
