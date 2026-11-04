package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"geoworld/internal/generator"
	"geoworld/internal/mcprofile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
)

func TestGamePlayerSyncAndTeleportControl(t *testing.T) {
	state := &appState{
		gameToken:  "secret-token",
		gameClient: "Minecraft 26.3 Fabric",
		gameSession: gameSessionState{
			Managed:   true,
			State:     "bound",
			Client:    "Minecraft 26.3 Fabric",
			WorldName: "Smoke World",
			Revision:  1,
		},
	}

	bad := httptest.NewRequest(
		http.MethodPost,
		"/api/game/session",
		strings.NewReader(`{"action":"player","playerX":1,"playerY":2,"playerZ":3}`),
	)
	bad.Header.Set("X-GeoWorld-Game-Token", "wrong-token")
	badRec := httptest.NewRecorder()
	state.handleGameSession(badRec, bad)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token status=%d want=%d", badRec.Code, http.StatusUnauthorized)
	}

	playerReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/session",
		strings.NewReader(`{"action":"player","client":"Minecraft 26.3 Fabric","playerX":4878.5,"playerY":134.238865,"playerZ":1542.5,"dimension":"minecraft:overworld"}`),
	)
	playerReq.Header.Set("X-GeoWorld-Game-Token", "secret-token")
	playerRec := httptest.NewRecorder()
	state.handleGameSession(playerRec, playerReq)
	if playerRec.Code != http.StatusOK {
		t.Fatalf("player sync status=%d body=%s", playerRec.Code, playerRec.Body.String())
	}
	if state.gameSession.Player == nil {
		t.Fatal("player position was not stored")
	}
	if got := *state.gameSession.Player; got.X != 4878.5 || got.Y != 134.238865 || got.Z != 1542.5 ||
		got.Dimension != "minecraft:overworld" {
		t.Fatalf("unexpected player position: %+v", got)
	}

	remoteReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/teleport",
		strings.NewReader(`{"x":120.5,"y":91,"z":-340.5}`),
	)
	remoteReq.RemoteAddr = "203.0.113.88:53000"
	remoteRec := httptest.NewRecorder()
	state.handleGameTeleport(remoteRec, remoteReq)
	if remoteRec.Code != http.StatusForbidden {
		t.Fatalf("remote teleport status=%d want=%d", remoteRec.Code, http.StatusForbidden)
	}

	tpReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/teleport",
		strings.NewReader(`{"x":120.5,"y":91,"z":-340.5}`),
	)
	tpReq.RemoteAddr = "127.0.0.1:53001"
	tpRec := httptest.NewRecorder()
	state.handleGameTeleport(tpRec, tpReq)
	if tpRec.Code != http.StatusAccepted {
		t.Fatalf("local teleport status=%d body=%s", tpRec.Code, tpRec.Body.String())
	}
	if state.pendingTeleport == nil {
		t.Fatal("teleport command was not queued")
	}
	if state.pendingTeleport.X != 120.5 || state.pendingTeleport.Y != 91 || state.pendingTeleport.Z != -340.5 {
		t.Fatalf("unexpected queued teleport: %+v", state.pendingTeleport)
	}

	controlReq := httptest.NewRequest(http.MethodGet, "/api/game/control", nil)
	controlReq.Header.Set("X-GeoWorld-Game-Token", "secret-token")
	controlRec := httptest.NewRecorder()
	state.handleGameControl(controlRec, controlReq)
	if controlRec.Code != http.StatusOK {
		t.Fatalf("control poll status=%d body=%s", controlRec.Code, controlRec.Body.String())
	}
	if got, want := controlRec.Body.String(), "teleport\t1\t120.500000\t91.000000\t-340.500000"; got != want {
		t.Fatalf("control payload=%q want=%q", got, want)
	}
	if state.pendingTeleport == nil {
		t.Fatal("teleport command must remain queued until Minecraft acknowledges execution")
	}

	ackReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/session",
		strings.NewReader(`{"action":"teleport_result","commandId":1,"success":true,"message":"executed"}`),
	)
	ackReq.Header.Set("X-GeoWorld-Game-Token", "secret-token")
	ackRec := httptest.NewRecorder()
	state.handleGameSession(ackRec, ackReq)
	if ackRec.Code != http.StatusOK {
		t.Fatalf("teleport ack status=%d body=%s", ackRec.Code, ackRec.Body.String())
	}
	if state.pendingTeleport != nil {
		t.Fatal("teleport command was not cleared after Minecraft acknowledgement")
	}
	if state.gameSession.TeleportCommandID != 1 || state.gameSession.TeleportState != "done" {
		t.Fatalf("unexpected teleport acknowledgement state: %+v", state.gameSession)
	}

	emptyReq := httptest.NewRequest(http.MethodGet, "/api/game/control", nil)
	emptyReq.Header.Set("X-GeoWorld-Game-Token", "secret-token")
	emptyRec := httptest.NewRecorder()
	state.handleGameControl(emptyRec, emptyReq)
	if emptyRec.Code != http.StatusNoContent {
		t.Fatalf("empty control poll status=%d want=%d", emptyRec.Code, http.StatusNoContent)
	}
}

func TestLocalInterfaceAddressCountsAsLocalGameControl(t *testing.T) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Skipf("cannot enumerate local interfaces: %v", err)
	}
	for _, addr := range addrs {
		var ip net.IP
		switch value := addr.(type) {
		case *net.IPNet:
			ip = value.IP
		case *net.IPAddr:
			ip = value.IP
		}
		if ip == nil || ip.IsLoopback() || ip.To4() == nil {
			continue
		}
		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		req.RemoteAddr = net.JoinHostPort(ip.String(), "53002")
		if !isLoopbackRequest(req) {
			t.Fatalf("local interface %s should be allowed for local game control", ip)
		}
		return
	}
	t.Skip("no non-loopback IPv4 interface available")
}

func TestGameChunkRegionControl(t *testing.T) {
	state := &appState{
		gameToken: "secret-token",
		gameSession: gameSessionState{
			Managed: true,
			State:   "bound",
			ChunkGeneration: &gameChunkGenerationState{
				Available: true,
				State:     "idle",
			},
		},
	}

	remoteReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/chunks",
		strings.NewReader(`{"action":"start","x1":0,"z1":0,"x2":31,"z2":31}`),
	)
	remoteReq.RemoteAddr = "203.0.113.88:53000"
	remoteRec := httptest.NewRecorder()
	state.handleGameChunks(remoteRec, remoteReq)
	if remoteRec.Code != http.StatusForbidden {
		t.Fatalf("remote chunks status=%d want=%d", remoteRec.Code, http.StatusForbidden)
	}

	startReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/chunks",
		strings.NewReader(`{"action":"start","x1":-1,"z1":1,"x2":17,"z2":33}`),
	)
	startReq.RemoteAddr = "127.0.0.1:53001"
	startRec := httptest.NewRecorder()
	state.handleGameChunks(startRec, startReq)
	if startRec.Code != http.StatusAccepted {
		t.Fatalf("chunk start status=%d body=%s", startRec.Code, startRec.Body.String())
	}
	if state.pendingChunky == nil {
		t.Fatal("chunk command was not queued")
	}
	if got := *state.pendingChunky; got.X1 != -16 || got.Z1 != 0 || got.X2 != 31 || got.Z2 != 47 {
		t.Fatalf("unexpected aligned chunk command: %+v", got)
	}
	if state.gameSession.ChunkGeneration.TotalChunks != 9 {
		t.Fatalf("total chunks=%d want=9", state.gameSession.ChunkGeneration.TotalChunks)
	}

	controlReq := httptest.NewRequest(http.MethodGet, "/api/game/control", nil)
	controlReq.Header.Set("X-GeoWorld-Game-Token", "secret-token")
	controlRec := httptest.NewRecorder()
	state.handleGameControl(controlRec, controlReq)
	if got, want := controlRec.Body.String(), "chunky\t1\tstart\t-16\t0\t31\t47"; got != want {
		t.Fatalf("chunk control payload=%q want=%q", got, want)
	}

	ackReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/session",
		strings.NewReader(`{"action":"chunky_result","commandId":1,"success":true,"message":"executed"}`),
	)
	ackReq.Header.Set("X-GeoWorld-Game-Token", "secret-token")
	ackRec := httptest.NewRecorder()
	state.handleGameSession(ackRec, ackReq)
	if ackRec.Code != http.StatusOK {
		t.Fatalf("chunk ack status=%d body=%s", ackRec.Code, ackRec.Body.String())
	}
	if state.pendingChunky != nil {
		t.Fatal("chunk command was not cleared after acknowledgement")
	}
	if state.gameSession.ChunkGeneration.State != "running" {
		t.Fatalf("chunk state=%q want=running", state.gameSession.ChunkGeneration.State)
	}

	statusReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/session",
		strings.NewReader(`{"action":"chunky_status","chunkyAvailable":true,"chunkyTaskExists":true,"chunkyProcessedChunks":5,"chunkyElapsedMillis":1200}`),
	)
	statusReq.Header.Set("X-GeoWorld-Game-Token", "secret-token")
	statusRec := httptest.NewRecorder()
	state.handleGameSession(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("chunk status sync=%d body=%s", statusRec.Code, statusRec.Body.String())
	}
	if got := state.gameSession.ChunkGeneration.ProcessedChunks; got != 5 {
		t.Fatalf("processed chunks=%d want=5", got)
	}
}

func TestStatusReportsLocalGameControlByRequester(t *testing.T) {
	state := &appState{}

	tests := []struct {
		name       string
		remoteAddr string
		want       bool
	}{
		{name: "loopback ipv4", remoteAddr: "127.0.0.1:53001", want: true},
		{name: "loopback ipv6", remoteAddr: "[::1]:53002", want: true},
		{name: "lan", remoteAddr: "192.168.1.88:53003", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
			req.RemoteAddr = tt.remoteAddr
			rec := httptest.NewRecorder()
			state.handleStatus(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			got, ok := payload["localGameControl"].(bool)
			if !ok {
				t.Fatalf("localGameControl missing or not bool: %#v", payload["localGameControl"])
			}
			if got != tt.want {
				t.Fatalf("localGameControl=%v want=%v for %s", got, tt.want, tt.remoteAddr)
			}
		})
	}
}

func TestBrowserURL(t *testing.T) {
	tests := []struct {
		addr string
		want string
	}{
		{"0.0.0.0:8788", "http://127.0.0.1:8788"},
		{":8788", "http://127.0.0.1:8788"},
		{"127.0.0.1:8788", "http://127.0.0.1:8788"},
		{"192.168.1.41:8788", "http://192.168.1.41:8788"},
		{"[::]:8788", "http://127.0.0.1:8788"},
	}
	for _, tt := range tests {
		if got := browserURL(tt.addr); got != tt.want {
			t.Fatalf("browserURL(%q) = %q, want %q", tt.addr, got, tt.want)
		}
	}
}

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"GeoWorld 世界 QA", "GeoWorld 世界 QA"},
		{"GeoWorld 世界:QA?", "GeoWorld 世界_QA_"},
		{"  preview.  ", "preview"},
		{"CON", "_CON"},
		{"a/b\\c|d*e", "a_b_c_d_e"},
	}
	for _, tt := range tests {
		if got := sanitizeFolderName(tt.in); got != tt.want {
			t.Fatalf("sanitizeFolderName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestQualityLayerPalettesDistinguishTerrainAndSurface(t *testing.T) {
	plain := runtimeprofile.Sample{TerrainVariant: mcprofile.TerrainPlain}
	mountain := runtimeprofile.Sample{TerrainVariant: mcprofile.TerrainMountain}
	if landformColor(plain) == landformColor(mountain) {
		t.Fatal("plain and mountain must use different landform colors")
	}

	sand := runtimeprofile.Sample{SurfaceProfile: mcprofile.SurfaceDesertSand}
	gravel := runtimeprofile.Sample{SurfaceProfile: mcprofile.SurfaceDesertGravel}
	snow := runtimeprofile.Sample{SurfaceProfile: mcprofile.SurfaceAlpineSnow}
	if surfaceColor(sand) == surfaceColor(gravel) {
		t.Fatal("desert sand and legacy desert gravel must be visually distinguishable")
	}
	if surfaceColor(snow) == surfaceColor(sand) {
		t.Fatal("alpine snow and desert sand must be visually distinguishable")
	}
}

func TestAnomalyReportMatchesAfterModelReopen(t *testing.T) {
	const (
		width    = 80
		height   = 50
		cellSize = 128.0
	)
	geometry := world.GridGeometry{
		Width:       width,
		Height:      height,
		OriginX:     -float64(width-1) * cellSize / 2,
		OriginZ:     -float64(height-1) * cellSize / 2,
		SpanXBlocks: float64(width-1) * cellSize,
		SpanZBlocks: float64(height-1) * cellSize,
	}
	model, err := generator.GenerateWorld(context.Background(), generator.Config{
		Seed:     731894205,
		Geometry: geometry,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	generated := &activeWorld{
		geometry: model.Geometry,
		sample:   model.RuntimeProfile.Sample,
		model:    model,
	}
	generatedReport := buildAnomalyReport(generated)

	path := filepath.Join(t.TempDir(), "world.geoworld")
	if err := worldmodelio.Save(path, model); err != nil {
		t.Fatal(err)
	}
	reopenedModel, err := worldmodelio.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	reopened := &activeWorld{
		geometry: reopenedModel.Geometry,
		sample:   reopenedModel.RuntimeProfile.Sample,
		reopened: reopenedModel,
	}
	reopenedReport := buildAnomalyReport(reopened)

	if !reflect.DeepEqual(generatedReport, reopenedReport) {
		t.Fatalf("anomaly report changed after reopen\ngenerated: %#v\nreopened: %#v", generatedReport, reopenedReport)
	}

	waterbedAudit := model.Waterbed.Audit(16, model.Config.Reconcile.FinalUpslopeEpsilon)
	if got := generatedReport.Counts[anomalyRiverUpslope]; got != waterbedAudit.RiverUpslopeSteps {
		t.Fatalf("river upslope count = %d, audit = %d", got, waterbedAudit.RiverUpslopeSteps)
	}
	if got := generatedReport.Counts[anomalyLakeBed]; got != waterbedAudit.LakeBedViolations {
		t.Fatalf("lake bed count = %d, audit = %d", got, waterbedAudit.LakeBedViolations)
	}
	if got := generatedReport.Counts[anomalyCoastMismatch]; got != model.Reconciliation.AppliedCoastSurfaceMismatch {
		t.Fatalf("coast mismatch count = %d, reconciliation = %d", got, model.Reconciliation.AppliedCoastSurfaceMismatch)
	}
	fallbacks, mismatches := summarizeRuntimeSemantics(model.Geometry, model.RuntimeProfile.Sample)
	if got := generatedReport.Counts[anomalyRuntimeFallback]; got != fallbacks {
		t.Fatalf("runtime fallback count = %d, macro audit = %d", got, fallbacks)
	}
	if got := generatedReport.Counts[anomalyRuntimeMismatch]; got != mismatches {
		t.Fatalf("runtime mismatch count = %d, macro audit = %d", got, mismatches)
	}
}

func TestGameBindStaysLazyAndDoesNotWarmFullWorldCaches(t *testing.T) {
	const (
		width    = 32
		height   = 20
		cellSize = 128.0
	)
	model, err := generator.GenerateWorld(
		context.Background(),
		generator.Config{
			Seed: 123456,
			Geometry: world.GridGeometry{
				Width:       width,
				Height:      height,
				OriginX:     -float64(width-1) * cellSize / 2,
				OriginZ:     -float64(height-1) * cellSize / 2,
				SpanXBlocks: float64(width-1) * cellSize,
				SpanZBlocks: float64(height-1) * cellSize,
			},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "world.geoworld")
	if err := worldmodelio.Save(path, model); err != nil {
		t.Fatal(err)
	}

	reopened, err := worldmodelio.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	manifest := reopened.Manifest
	state := &appState{
		gameToken:            "test-session-key",
		gameClient:           "Minecraft 1.21.1 NeoForge",
		gameMinecraftVersion: "1.21.1",
		gameSession: gameSessionState{
			Managed:  true,
			State:    "menu",
			Client:   "Minecraft 1.21.1 NeoForge",
			Revision: 1,
		},
	}
	req := gameSessionRequest{
		Action:                  "bind",
		Client:                  "Minecraft 1.21.1 NeoForge",
		WorldName:               "Lazy Bind Test",
		ModelPath:               path,
		BindingVersion:          2,
		MinecraftVersion:        "1.21.1",
		ModelSchemaVersion:      manifest.SchemaVersion,
		GeneratorVersion:        manifest.GeneratorVersion,
		Seed:                    manifest.Seed,
		WorldProfileVersion:     manifest.WorldProfileVersion,
		GenerationRulesVersion:  manifest.GenerationRulesVersion,
		WorldModelVersion:       manifest.WorldModelVersion,
		Preset:                  manifest.Preset,
		ParameterSnapshotSHA256: manifest.ParameterSnapshotSHA256,
		DynamicWorldgen:         true,
		OreFamilies:             []string{"copper", "zinc"},
		OreIcons: map[string]string{
			"copper": "data:image/png;base64,Y29wcGVy",
			"zinc":   "data:image/png;base64,emluYw==",
		},
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	httpReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/session",
		strings.NewReader(string(body)),
	)
	httpReq.Header.Set("X-GeoWorld-Game-Token", "test-session-key")
	rec := httptest.NewRecorder()
	state.handleGameSession(rec, httpReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("bind status=%d body=%s", rec.Code, rec.Body.String())
	}
	if got := state.gameSession.OreIcons["zinc"]; got != "data:image/png;base64,emluYw==" {
		t.Fatalf("modded ore icon was not preserved in game session: %q", got)
	}
	if len(state.gameSession.OreIcons) != 2 {
		t.Fatalf("ore icon count=%d want=2", len(state.gameSession.OreIcons))
	}

	assertCold := func(stage string) {
		t.Helper()
		state.mu.RLock()
		active := state.active
		state.mu.RUnlock()
		if active == nil {
			t.Fatalf("%s: active world missing", stage)
		}
		if len(active.previewGrid) != 0 ||
			len(active.settlementGrid) != 0 ||
			len(active.navigationGrid) != 0 ||
			len(active.miningGrid) != 0 ||
			len(active.regionalAdvantageGrid) != 0 ||
			len(active.transportGeoModel.Scores) != 0 ||
			len(active.civilizationGrid) != 0 {
			t.Fatalf(
				"%s: managed bind eagerly warmed full-world caches: "+
					"preview=%d settlement=%d navigation=%d mining=%d "+
					"advantage=%d transport=%d civilization=%d",
				stage,
				len(active.previewGrid),
				len(active.settlementGrid),
				len(active.navigationGrid),
				len(active.miningGrid),
				len(active.regionalAdvantageGrid),
				len(active.transportGeoModel.Scores),
				len(active.civilizationGrid),
			)
		}
	}
	assertCold("immediate")
	time.Sleep(200 * time.Millisecond)
	assertCold("after-200ms")
}

func TestGameBindMinecraftVersionFollowsLauncher(t *testing.T) {
	state := &appState{
		gameToken:            "secret-token",
		gameClient:           "Minecraft 1.21.1 NeoForge",
		gameMinecraftVersion: "1.21.1",
		gameSession: gameSessionState{
			Managed:  true,
			State:    "menu",
			Client:   "Minecraft 1.21.1 NeoForge",
			Revision: 1,
		},
	}

	req := gameSessionRequest{
		Action:           "bind",
		BindingVersion:   2,
		MinecraftVersion: "26.3",
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	httpReq := httptest.NewRequest(
		http.MethodPost,
		"/api/game/session",
		strings.NewReader(string(body)),
	)
	httpReq.Header.Set("X-GeoWorld-Game-Token", "secret-token")
	rec := httptest.NewRecorder()
	state.handleGameSession(rec, httpReq)
	if rec.Code != http.StatusConflict {
		t.Fatalf("wrong-version status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "1.21.1") {
		t.Fatalf("version mismatch must use launcher version: %s", rec.Body.String())
	}
}
