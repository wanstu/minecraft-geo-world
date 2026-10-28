package main

import (
	"net/http"

	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
	"geoworld/internal/reconcile"
)

type previewFeatureResponse struct {
	Seed              int64                     `json:"seed"`
	Rivers            []previewRiver            `json:"rivers"`
	Lakes             []previewLake             `json:"lakes"`
	Peaks             []previewPointFeature     `json:"peaks"`
	SpecialLandscapes []previewSpecialLandscape `json:"specialLandscapes"`
	Counts            map[string]int            `json:"counts"`
	Note              string                    `json:"note,omitempty"`
}

type previewSpecialLandscape struct {
	Kind         string  `json:"kind"`
	Name         string  `json:"name"`
	X            float64 `json:"x"`
	Z            float64 `json:"z"`
	RadiusX      float64 `json:"radiusX"`
	RadiusZ      float64 `json:"radiusZ"`
	WaterContext string  `json:"waterContext,omitempty"`
}

type previewRiver struct {
	X0            float64 `json:"x0"`
	Z0            float64 `json:"z0"`
	X1            float64 `json:"x1"`
	Z1            float64 `json:"z1"`
	HalfWidth     float64 `json:"halfWidth"`
	OceanMouth    bool    `json:"oceanMouth,omitempty"`
	LakeMouth     bool    `json:"lakeMouth,omitempty"`
	LakeOutlet    bool    `json:"lakeOutlet,omitempty"`
	MeanderOffset float64 `json:"meanderOffset,omitempty"`
}

type previewLake struct {
	LakeID       int     `json:"lakeId"`
	X            float64 `json:"x"`
	Z            float64 `json:"z"`
	InnerRadius  float64 `json:"innerRadius"`
	OuterRadius  float64 `json:"outerRadius"`
	WaterSurface float64 `json:"waterSurface"`
	CoreCarve    bool    `json:"coreCarve,omitempty"`
}

type previewPointFeature struct {
	Kind     string  `json:"kind"`
	Subtype  string  `json:"subtype"`
	Name     string  `json:"name"`
	X        float64 `json:"x"`
	Z        float64 `json:"z"`
	Value    float64 `json:"value,omitempty"`
	Strength float64 `json:"strength,omitempty"`
	Detail   string  `json:"detail,omitempty"`
}

func (s *appState) handlePreviewFeatures(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	s.mu.RLock()
	active := s.active
	s.mu.RUnlock()
	if active == nil {
		writeError(w, http.StatusConflict, errNoActiveWorld)
		return
	}

	water := activeWaterbed(active)
	rivers := make([]previewRiver, 0)
	lakes := make([]previewLake, 0)
	if water != nil {
		rivers = make([]previewRiver, 0, len(water.Rivers))
		for _, seg := range water.Rivers {
			rivers = append(rivers, previewRiver{
				X0: seg.X0, Z0: seg.Z0, X1: seg.X1, Z1: seg.Z1,
				HalfWidth: seg.HalfWidth, OceanMouth: seg.OceanMouth,
				LakeMouth: seg.LakeMouth, LakeOutlet: seg.LakeOutlet,
				MeanderOffset: seg.MeanderOffset,
			})
		}
		lakes = make([]previewLake, 0, len(water.Lakes))
		for _, node := range water.Lakes {
			lakes = append(lakes, previewLake{
				LakeID: node.LakeID, X: node.X, Z: node.Z,
				InnerRadius: node.InnerRadius, OuterRadius: node.OuterRadius,
				WaterSurface: node.WaterSurface, CoreCarve: node.CoreCarve,
			})
		}
	}

	peaks := collectPreviewPeaks(active)
	special := collectPreviewSpecialLandscapes(active)
	tectonicCount := countPreviewTectonics(active)
	resp := previewFeatureResponse{
		Seed:              active.seed,
		Rivers:            rivers,
		Lakes:             lakes,
		Peaks:             peaks,
		SpecialLandscapes: special,
		Counts: map[string]int{
			"rivers":    len(rivers),
			"lakes":     len(lakes),
			"peaks":     len(peaks),
			"special":   len(special),
			"tectonics": tectonicCount,
		},
		Note: "世界预览要素直接来自当前 World Model；与验收表和抽检表完全独立。",
	}
	writeJSON(w, http.StatusOK, resp)
}

var errNoActiveWorld = &previewStateError{"generate or load a world first"}

type previewStateError struct{ text string }

func (e *previewStateError) Error() string { return e.text }

func collectPreviewSpecialLandscapes(active *activeWorld) []previewSpecialLandscape {
	features := activeSpecialFeatures(active)
	if len(features) == 0 {
		return nil
	}
	out := make([]previewSpecialLandscape, 0, len(features))
	for _, feature := range features {
		out = append(out, previewSpecialLandscape{
			Kind:         string(feature.Kind),
			Name:         specialFeatureLabel(string(feature.Kind)),
			X:            feature.CenterX,
			Z:            feature.CenterZ,
			RadiusX:      feature.RadiusX,
			RadiusZ:      feature.RadiusZ,
			WaterContext: feature.WaterContext,
		})
	}
	return out
}

func collectPreviewPeaks(active *activeWorld) []previewPointFeature {
	if active == nil || active.sample == nil {
		return nil
	}
	g := active.geometry
	peaks := make([]previewPointFeature, 0, 128)
	for z := 1; z < g.Height-1; z++ {
		for x := 1; x < g.Width-1; x++ {
			wx, wz := g.NodePosition(x, z)
			sample := active.sample(wx, wz)
			if sample.World.WaterKind != reconcile.WaterNone {
				continue
			}
			if sample.TerrainVariant != mcprofile.TerrainMountain &&
				sample.TerrainVariant != mcprofile.TerrainPlateau {
				continue
			}
			height := sample.World.GroundHeight
			isPeak := true
			maxNeighbor := -1.0
			for dz := -1; dz <= 1 && isPeak; dz++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dz == 0 {
						continue
					}
					nx, nz := g.NodePosition(x+dx, z+dz)
					neighbor := active.sample(nx, nz).World.GroundHeight
					if neighbor > maxNeighbor {
						maxNeighbor = neighbor
					}
					if neighbor >= height {
						isPeak = false
						break
					}
				}
			}
			if !isPeak {
				continue
			}
			prominence := height - maxNeighbor
			peaks = append(peaks, previewPointFeature{
				Kind: "peak", Subtype: sample.TerrainVariant.String(),
				Name: "山峰", X: wx, Z: wz, Value: height,
				Strength: prominence,
				Detail:   sample.BiomeFamily.String(),
			})
		}
	}
	return peaks
}

func countPreviewTectonics(active *activeWorld) int {
	if active == nil {
		return 0
	}
	var features []phase1.TectonicFeature
	if active.model != nil && active.model.Phase1 != nil {
		features = active.model.Phase1.Feature
	} else if active.reopened != nil && active.reopened.TectonicQuery != nil {
		features = active.reopened.TectonicQuery.Feature
	}
	count := 0
	for _, feature := range features {
		if feature != phase1.NoFeature {
			count++
		}
	}
	return count
}

func collectPreviewTectonics(active *activeWorld) []previewPointFeature {
	if active == nil {
		return nil
	}
	var features []phase1.TectonicFeature
	var intensity []float64
	if active.model != nil && active.model.Phase1 != nil {
		features = active.model.Phase1.Feature
		intensity = active.model.Phase1.TectonicIntensity
	} else if active.reopened != nil && active.reopened.TectonicQuery != nil {
		features = active.reopened.TectonicQuery.Feature
		intensity = active.reopened.TectonicQuery.Intensity
	}
	if len(features) == 0 {
		return nil
	}
	g := active.geometry
	out := make([]previewPointFeature, 0, len(features)/8)
	for index, feature := range features {
		if feature == phase1.NoFeature {
			continue
		}
		x := index % g.Width
		z := index / g.Width
		if z >= g.Height {
			break
		}
		wx, wz := g.NodePosition(x, z)
		strength := 0.0
		if index < len(intensity) {
			strength = intensity[index]
		}
		out = append(out, previewPointFeature{
			Kind: "tectonic", Subtype: tectonicFeatureName(feature),
			Name: tectonicFeatureLabel(feature),
			X:    wx, Z: wz, Strength: strength,
		})
	}
	return out
}

func tectonicFeatureName(feature phase1.TectonicFeature) string {
	switch feature {
	case phase1.CollisionOrogen:
		return "collision_orogen"
	case phase1.VolcanicArc:
		return "volcanic_arc"
	case phase1.Trench:
		return "trench"
	case phase1.IslandArc:
		return "island_arc"
	case phase1.MidOceanRidge:
		return "mid_ocean_ridge"
	case phase1.ContinentalRift:
		return "continental_rift"
	case phase1.TransformFault:
		return "transform_fault"
	default:
		return "unknown"
	}
}

func tectonicFeatureLabel(feature phase1.TectonicFeature) string {
	switch feature {
	case phase1.CollisionOrogen:
		return "碰撞造山带"
	case phase1.VolcanicArc:
		return "火山弧"
	case phase1.Trench:
		return "海沟"
	case phase1.IslandArc:
		return "岛弧"
	case phase1.MidOceanRidge:
		return "洋中脊"
	case phase1.ContinentalRift:
		return "大陆裂谷"
	case phase1.TransformFault:
		return "转换断层"
	default:
		return "地质构造"
	}
}
