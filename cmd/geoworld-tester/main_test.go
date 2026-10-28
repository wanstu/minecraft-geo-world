package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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
	remoteReq.RemoteAddr = "192.168.1.88:53000"
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
	if got, want := controlRec.Body.String(), "1\t120.500000\t91.000000\t-340.500000"; got != want {
		t.Fatalf("control payload=%q want=%q", got, want)
	}
	if state.pendingTeleport != nil {
		t.Fatal("teleport command was not consumed after game poll")
	}

	emptyReq := httptest.NewRequest(http.MethodGet, "/api/game/control", nil)
	emptyReq.Header.Set("X-GeoWorld-Game-Token", "secret-token")
	emptyRec := httptest.NewRecorder()
	state.handleGameControl(emptyRec, emptyReq)
	if emptyRec.Code != http.StatusNoContent {
		t.Fatalf("empty control poll status=%d want=%d", emptyRec.Code, http.StatusNoContent)
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
