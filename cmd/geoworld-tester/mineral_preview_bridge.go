package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"geoworld/internal/worldmodelio"
)

const mineralPreviewClass = "geoworld.minecraft.v26common.worldgen.GeoWorldV26MineralPreviewCli"

var sharedMineralPlanner persistentMineralPlanner

type persistentMineralPlanner struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	java   string
	cp     string
}

func (b *persistentMineralPlanner) run(
	javaExecutable string,
	classpath string,
	fields ...string,
) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.ensure(javaExecutable, classpath); err != nil {
		return nil, err
	}
	output, err := b.request(fields...)
	if err == nil {
		return output, nil
	}

	b.stop()
	if err := b.ensure(javaExecutable, classpath); err != nil {
		return nil, err
	}
	return b.request(fields...)
}

func (b *persistentMineralPlanner) ensure(
	javaExecutable string,
	classpath string,
) error {
	if b.cmd != nil && b.java == javaExecutable && b.cp == classpath {
		return nil
	}
	b.stop()

	cmd := exec.Command(
		javaExecutable,
		"-cp",
		classpath,
		mineralPreviewClass,
		"--server",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}

	b.cmd = cmd
	b.stdin = stdin
	b.stdout = bufio.NewReaderSize(stdoutPipe, 256*1024)
	b.java = javaExecutable
	b.cp = classpath
	return nil
}

func (b *persistentMineralPlanner) request(fields ...string) ([]byte, error) {
	if b.stdin == nil || b.stdout == nil {
		return nil, errors.New("mineral planner process is not running")
	}
	for _, field := range fields {
		if strings.ContainsAny(field, "\r\n\t") {
			return nil, errors.New("mineral planner request contains invalid control characters")
		}
	}
	if _, err := io.WriteString(b.stdin, strings.Join(fields, "\t")+"\n"); err != nil {
		return nil, err
	}
	output, err := b.stdout.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	output = bytes.TrimSpace(output)
	if !json.Valid(output) {
		return nil, errors.New("mineral planner returned invalid JSON")
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(output, &payload); err == nil {
		if raw, ok := payload["error"]; ok {
			var message string
			if json.Unmarshal(raw, &message) == nil && message != "" {
				return nil, errors.New(message)
			}
		}
	}
	return output, nil
}

func (b *persistentMineralPlanner) stop() {
	if b.stdin != nil {
		_, _ = io.WriteString(b.stdin, "QUIT\n")
		_ = b.stdin.Close()
	}
	if b.cmd != nil && b.cmd.Process != nil {
		_ = b.cmd.Process.Kill()
		_ = b.cmd.Wait()
	}
	b.cmd = nil
	b.stdin = nil
	b.stdout = nil
	b.java = ""
	b.cp = ""
}

func (s *appState) handleMineralBodies(w http.ResponseWriter, r *http.Request) {
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

	mode := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mode")))
	if mode == "" {
		mode = "major"
	}
	if mode != "major" && mode != "all" {
		writeError(w, http.StatusBadRequest, errors.New("mineral mode must be major or all"))
		return
	}

	g := active.geometry
	modelMinX := int(g.OriginX)
	modelMaxX := int(g.OriginX + g.SpanXBlocks)
	modelMinZ := int(g.OriginZ)
	modelMaxZ := int(g.OriginZ + g.SpanZBlocks)
	minX, err := queryInt(r, "minX", modelMinX)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	maxX, err := queryInt(r, "maxX", modelMaxX)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	minZ, err := queryInt(r, "minZ", modelMinZ)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	maxZ, err := queryInt(r, "maxZ", modelMaxZ)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	minX = maxInt(minX, modelMinX)
	maxX = minInt(maxX, modelMaxX)
	minZ = maxInt(minZ, modelMinZ)
	maxZ = minInt(maxZ, modelMaxZ)
	if minX > maxX || minZ > maxZ {
		writeError(w, http.StatusBadRequest, errors.New("mineral viewport is outside the model"))
		return
	}

	maxSpan := 9000
	if mode == "all" {
		maxSpan = 4200
	}
	if maxX-minX > maxSpan || maxZ-minZ > maxSpan {
		writeError(
			w,
			http.StatusRequestEntityTooLarge,
			fmt.Errorf("%s mineral viewport exceeds %d blocks; zoom in", mode, maxSpan),
		)
		return
	}

	limit := parseBoundedInt(r.URL.Query().Get("limit"), 1600, 100, 5000)
	javaExecutable := strings.TrimSpace(os.Getenv("GEOWORLD_JAVA"))
	jarPath := strings.TrimSpace(os.Getenv("GEOWORLD_TOOL_JAR"))
	if javaExecutable == "" || jarPath == "" {
		writeError(
			w,
			http.StatusServiceUnavailable,
			errors.New("exact mineral preview is available from the executable GeoWorld JAR"),
		)
		return
	}
	if _, err := os.Stat(javaExecutable); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("java runtime unavailable: %w", err))
		return
	}
	if _, err := os.Stat(jarPath); err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("GeoWorld JAR unavailable: %w", err))
		return
	}

	modelPath, err := s.ensureMineralPreviewModel(active)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	output, err := sharedMineralPlanner.run(
		javaExecutable,
		jarPath,
		modelPath,
		strconv.Itoa(minX),
		strconv.Itoa(maxX),
		strconv.Itoa(minZ),
		strconv.Itoa(maxZ),
		mode,
		strconv.Itoa(limit),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("mineral planner bridge: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(output)
}

func (s *appState) ensureMineralPreviewModel(active *activeWorld) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != active {
		return "", errors.New("active world changed while preparing mineral preview")
	}
	if active.mineralPreviewModelPath != "" {
		return active.mineralPreviewModelPath, nil
	}
	if active.modelPath != "" {
		path, err := filepath.Abs(active.modelPath)
		if err != nil {
			return "", err
		}
		active.mineralPreviewModelPath = path
		return path, nil
	}
	if active.model == nil {
		return "", errors.New("active world has no materializable model")
	}

	cacheDir := filepath.Join(os.TempDir(), "geoworld-web-tool", "mineral-models")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}
	hash := active.parameterSnapshotSHA256
	if len(hash) > 12 {
		hash = hash[:12]
	}
	rules := sanitizeName(active.generationRulesVersion)
	if rules == "" {
		rules = "legacy"
	}
	name := fmt.Sprintf(
		"seed-%d-%dx%d-%s-%s.geoworld",
		active.seed,
		active.geometry.Width,
		active.geometry.Height,
		rules,
		hash,
	)
	path := filepath.Join(cacheDir, name)
	if _, err := os.Stat(path); err == nil {
		active.mineralPreviewModelPath = path
		return path, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := worldmodelio.Save(path, active.model); err != nil {
		return "", err
	}
	active.mineralPreviewModelPath = path
	return path, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func queryInt(r *http.Request, name string, fallback int) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", name)
	}
	return parsed, nil
}
