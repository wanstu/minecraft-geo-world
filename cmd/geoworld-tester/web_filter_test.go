package main

import (
	"os"
	"strings"
	"testing"
)

func TestMineralFilterUsesFastOverlayPath(t *testing.T) {
	data, err := os.ReadFile("web/app.js")
	if err != nil {
		t.Fatalf("read web/app.js: %v", err)
	}
	source := string(data)

	required := []string{
		"function applyMineralFilterToOverlay()",
		"item.dataset.minerals=primaryKey+','+companionKey",
		"marker.dataset.mineral=mineralKey",
		"$('mineralFilter').addEventListener('change',()=>{",
		"applyMineralFilterToOverlay();",
		"renderDensityCanvas();",
	}
	for _, fragment := range required {
		if !strings.Contains(source, fragment) {
			t.Fatalf("fast mineral filter contract missing %q", fragment)
		}
	}

	if strings.Contains(source, "['riverFilter','mineralFilter','specialFilter']") {
		t.Fatal("mineral filter regressed to the shared full-overlay render handler")
	}

	start := strings.Index(source, "$('mineralFilter').addEventListener('change',()=>{")
	if start < 0 {
		t.Fatal("mineral filter change handler missing")
	}
	end := strings.Index(source[start:], "});")
	if end < 0 {
		t.Fatal("mineral filter change handler is malformed")
	}
	handler := source[start : start+end]
	if strings.Contains(handler, "renderFeatureOverlay(") {
		t.Fatal("mineral filter change must not rebuild the full SVG overlay")
	}
}
