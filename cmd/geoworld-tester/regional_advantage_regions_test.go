package main

import (
	"encoding/json"
	"image"
	"net/http"
	"net/http/httptest"
	"testing"

	"geoworld/internal/advantage"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/world"
)

func TestRegionalAdvantageRegionLookupAndBoundary(t *testing.T) {
	geometry := world.GridGeometry{
		Width:       2,
		Height:      2,
		OriginX:     0,
		OriginZ:     0,
		SpanXBlocks: 1,
		SpanZBlocks: 1,
	}
	active := &activeWorld{
		geometry: geometry,
		regionalAdvantageGrid: []regionalAdvantagePreviewCell{
			{RegionID: 1},
			{RegionID: 2},
			{RegionID: 1},
			{RegionID: 0},
		},
		regionalAdvantageRegions: advantage.RegionModel{
			Width:         2,
			Height:        2,
			CellRegionIDs: []int{1, 2, 1, 0},
			Regions: []advantage.Region{
				{ID: 1, Key: "building-0001", DominantAdvantage: "building"},
				{ID: 2, Key: "port-0002", DominantAdvantage: "port"},
			},
		},
	}

	region, ok := regionalAdvantageRegionAt(active, 0, 0)
	if !ok || region.ID != 1 || region.DominantAdvantage != "building" {
		t.Fatalf("unexpected region lookup: region=%+v ok=%t", region, ok)
	}
	region, ok = regionalAdvantageRegionAt(active, 1, 0)
	if !ok || region.ID != 2 || region.DominantAdvantage != "port" {
		t.Fatalf("unexpected second region lookup: region=%+v ok=%t", region, ok)
	}
	if _, ok := regionalAdvantageRegionAt(active, 0.5, 0.5); ok {
		t.Fatal("region lookup should not invent a region for an unassigned macro cell")
	}
	if !regionalAdvantageBoundaryAt(active, 1, 0, 0.5, 0, image.Rect(0, 0, 101, 101)) {
		t.Fatal("region boundary should be visible at the midpoint between different region IDs")
	}
}

func TestSampleAPIIncludesRegionalRegionDetails(t *testing.T) {
	active := &activeWorld{
		geometry: world.GridGeometry{
			Width:       2,
			Height:      2,
			OriginX:     0,
			OriginZ:     0,
			SpanXBlocks: 1,
			SpanZBlocks: 1,
		},
		sample: func(float64, float64) runtimeprofile.Sample {
			return runtimeprofile.Sample{}
		},
		regionalAdvantageGrid: []regionalAdvantagePreviewCell{
			{RegionID: 1},
			{RegionID: 2},
			{RegionID: 1},
			{RegionID: 0},
		},
		regionalAdvantageRegions: advantage.RegionModel{
			Width:         2,
			Height:        2,
			CellRegionIDs: []int{1, 2, 1, 0},
			Regions: []advantage.Region{
				{ID: 1, Key: "building-0001", DominantAdvantage: "building", CellCount: 3, AreaBlocks2: 3},
				{ID: 2, Key: "port-0002", DominantAdvantage: "port", CellCount: 1, AreaBlocks2: 1},
			},
		},
	}
	state := &appState{active: active}
	req := httptest.NewRequest(http.MethodGet, "/api/sample?x=0&z=0", nil)
	rec := httptest.NewRecorder()
	state.handleSample(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("sample status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode sample response: %v", err)
	}
	region, ok := payload["regionalRegion"].(map[string]any)
	if !ok {
		t.Fatalf("regionalRegion missing from sample response: %s", rec.Body.String())
	}
	if region["id"] != float64(1) || region["dominantAdvantage"] != "building" {
		t.Fatalf("unexpected regionalRegion payload: %+v", region)
	}
}
