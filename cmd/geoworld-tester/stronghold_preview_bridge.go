package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const strongholdPreviewClass = "geoworld.minecraft.v26_3common.worldgen.GeoWorldV26_3StrongholdPreviewCli"

type strongholdPreviewPayload struct {
	Seed      int64  `json:"seed"`
	SetID     string `json:"setId"`
	Distance  int    `json:"distance"`
	Spread    int    `json:"spread"`
	Count     int    `json:"count"`
	Positions []struct {
		Index  int `json:"index"`
		ChunkX int `json:"chunkX"`
		ChunkZ int `json:"chunkZ"`
		X      int `json:"x"`
		Z      int `json:"z"`
	} `json:"positions"`
}

func (s *appState) strongholdPreviewCandidates(
	active *activeWorld,
) ([]structureCandidate, error) {
	if active == nil {
		return nil, errNoActiveWorld
	}

	active.strongholdMu.Lock()
	defer active.strongholdMu.Unlock()

	if active.strongholdLoaded {
		if active.strongholdErr != "" {
			return nil, errors.New(active.strongholdErr)
		}
		return append([]structureCandidate(nil), active.strongholdCandidates...), nil
	}

	candidates, err := s.loadStrongholdPreviewCandidates(active)
	active.strongholdLoaded = true
	if err != nil {
		active.strongholdErr = err.Error()
		return nil, err
	}
	active.strongholdCandidates = append(
		[]structureCandidate(nil),
		candidates...,
	)
	return append([]structureCandidate(nil), candidates...), nil
}

func (s *appState) loadStrongholdPreviewCandidates(
	active *activeWorld,
) ([]structureCandidate, error) {
	javaExecutable := strings.TrimSpace(os.Getenv("GEOWORLD_JAVA"))
	jarPath := strings.TrimSpace(os.Getenv("GEOWORLD_TOOL_JAR"))
	if javaExecutable == "" || jarPath == "" {
		return nil, errors.New(
			"exact stronghold preview is available from the executable GeoWorld JAR",
		)
	}
	if _, err := os.Stat(javaExecutable); err != nil {
		return nil, fmt.Errorf("java runtime unavailable: %w", err)
	}
	if _, err := os.Stat(jarPath); err != nil {
		return nil, fmt.Errorf("GeoWorld JAR unavailable: %w", err)
	}

	modelPath, err := s.ensureMineralPreviewModel(active)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		javaExecutable,
		"-cp",
		jarPath,
		strongholdPreviewClass,
		modelPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, errors.New("stronghold planner timed out")
	}
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("stronghold planner bridge: %s", message)
	}

	var payload strongholdPreviewPayload
	if err := json.Unmarshal(bytes.TrimSpace(output), &payload); err != nil {
		return nil, fmt.Errorf("stronghold planner returned invalid JSON: %w", err)
	}
	if payload.Seed != active.seed {
		return nil, fmt.Errorf(
			"stronghold planner seed mismatch: got %d want %d",
			payload.Seed,
			active.seed,
		)
	}
	if payload.SetID != "minecraft:strongholds" {
		return nil, fmt.Errorf(
			"stronghold planner set mismatch: %q",
			payload.SetID,
		)
	}
	if payload.Count != 128 || len(payload.Positions) != payload.Count {
		return nil, fmt.Errorf(
			"stronghold planner count mismatch: declared=%d returned=%d",
			payload.Count,
			len(payload.Positions),
		)
	}

	candidates := make([]structureCandidate, 0, len(payload.Positions))
	for i, position := range payload.Positions {
		if position.Index != i {
			return nil, fmt.Errorf(
				"stronghold planner index mismatch: got %d want %d",
				position.Index,
				i,
			)
		}
		candidates = append(candidates, structureCandidate{
			SetID:  payload.SetID,
			ChunkX: position.ChunkX,
			ChunkZ: position.ChunkZ,
			X:      position.X,
			Z:      position.Z,
		})
	}
	return candidates, nil
}
