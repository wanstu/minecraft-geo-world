package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func dimensionControlRequest(state *appState, path, payload string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	req.RemoteAddr = "127.0.0.1:53001"
	req.Header.Set("X-GeoWorld-Game-Token", "test-token")
	rec := httptest.NewRecorder()
	switch path {
	case "/api/game/session":
		state.handleGameSession(rec, req)
	case "/api/game/teleport":
		state.handleGameTeleport(rec, req)
	case "/api/game/chunks":
		state.handleGameChunks(rec, req)
	}
	return rec
}

func TestGameDimensionGuardsAndReturnToOverworld(t *testing.T) {
	for _, dimension := range []string{"minecraft:the_nether", "minecraft:the_end", "mod:other", ""} {
		t.Run(dimension, func(t *testing.T) {
			state := &appState{
				gameToken: "test-token",
				gameSession: gameSessionState{
					Managed: true, State: "bound",
					Player:            &gamePlayerPosition{Dimension: "minecraft:overworld"},
					ChunkGeneration:   &gameChunkGenerationState{Available: true, State: "pending"},
					TeleportCommandID: 1, TeleportState: "pending",
				},
				pendingTeleport: &gameTeleportCommand{ID: 1, X: 32, Y: 90, Z: 32},
				pendingChunky:   &gameChunkCommand{ID: 1, Action: "start"},
			}
			payload, _ := json.Marshal(gameSessionRequest{
				Action: "player", PlayerX: 8, PlayerY: 65, PlayerZ: 8, Dimension: dimension,
			})
			rec := dimensionControlRequest(state, "/api/game/session", string(payload))
			if rec.Code != http.StatusOK {
				t.Fatalf("dimension update: %d %s", rec.Code, rec.Body.String())
			}
			if state.gameSession.inOverworld() || state.pendingTeleport != nil || state.pendingChunky != nil {
				t.Fatal("leaving overworld must disable controls and clear queued coordinate commands")
			}
			if state.gameSession.TeleportState != "failed" || state.gameSession.ChunkGeneration.State != "failed" {
				t.Fatal("cancelled commands must have an explicit failure state")
			}
			rec = dimensionControlRequest(state, "/api/game/teleport", `{"x":0,"y":90,"z":0}`)
			if rec.Code != http.StatusConflict {
				t.Fatalf("cross-dimension teleport: %d", rec.Code)
			}
			for _, action := range []string{"start", "continue"} {
				rec = dimensionControlRequest(state, "/api/game/chunks", `{"action":"`+action+`","x1":0,"z1":0,"x2":31,"z2":31}`)
				if rec.Code != http.StatusConflict {
					t.Fatalf("cross-dimension %s: %d", action, rec.Code)
				}
			}

			rec = dimensionControlRequest(state, "/api/game/session", `{"action":"player","playerX":64,"playerY":90,"playerZ":64,"dimension":"minecraft:overworld"}`)
			if rec.Code != http.StatusOK || !state.gameSession.inOverworld() {
				t.Fatal("return must restore overworld status")
			}
			rec = dimensionControlRequest(state, "/api/game/teleport", `{"x":32,"y":90,"z":32}`)
			if rec.Code != http.StatusAccepted || state.pendingTeleport == nil {
				t.Fatalf("restored teleport: %d %s", rec.Code, rec.Body.String())
			}
			rec = dimensionControlRequest(state, "/api/game/chunks", `{"action":"start","x1":0,"z1":0,"x2":31,"z2":31}`)
			if rec.Code != http.StatusAccepted || state.pendingChunky == nil {
				t.Fatalf("restored pregen: %d %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestGameWithoutPlayerCannotTeleport(t *testing.T) {
	state := &appState{gameToken: "test-token", gameSession: gameSessionState{Managed: true, State: "bound"}}
	rec := dimensionControlRequest(state, "/api/game/teleport", `{"x":0,"y":90,"z":0}`)
	if rec.Code != http.StatusConflict || state.pendingTeleport != nil {
		t.Fatalf("missing dimension: %d", rec.Code)
	}
}

func TestGameCanStopOverworldPregenFromOtherDimensions(t *testing.T) {
	for _, action := range []string{"pause", "cancel"} {
		state := &appState{gameToken: "test-token", gameSession: gameSessionState{
			Managed: true, State: "bound",
			Player:          &gamePlayerPosition{Dimension: "minecraft:the_end"},
			ChunkGeneration: &gameChunkGenerationState{Available: true, State: "running"},
		}}
		rec := dimensionControlRequest(state, "/api/game/chunks", `{"action":"`+action+`"}`)
		if rec.Code != http.StatusAccepted || state.pendingChunky == nil || state.pendingChunky.Action != action {
			t.Fatalf("%s of existing overworld task must remain available: %d %s", action, rec.Code, rec.Body.String())
		}
	}
}

func TestStrongholdUsesBoundMinecraftVersion(t *testing.T) {
	for _, tc := range []struct{ bound, launcher, want string }{
		{"1.21.1", "26.3", "1.21.1"},
		{"26.3", "", "26.3"},
		{"", "1.21.1", "1.21.1"},
		{"", "", "26.3"},
	} {
		state := &appState{gameMinecraftVersion: tc.launcher, gameSession: gameSessionState{MinecraftVersion: tc.bound}}
		if got := state.strongholdMinecraftVersion(); got != tc.want {
			t.Fatalf("bound=%q launcher=%q: got %q want %q", tc.bound, tc.launcher, got, tc.want)
		}
	}
}
