package goldenvectors

import (
	"context"
	"errors"
	"fmt"
	"math"

	"geoworld/internal/generator"
	"geoworld/internal/phase1"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/runtimeworld"
	"geoworld/internal/world"
	"geoworld/internal/worldmodelio"
)

const (
	Format         = "geoworld-runtime-golden"
	FormatVersion  = 1
	FloatTolerance = 1e-9
)

type Geometry struct {
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	OriginX     float64 `json:"originX"`
	OriginZ     float64 `json:"originZ"`
	SpanXBlocks float64 `json:"spanXBlocks"`
	SpanZBlocks float64 `json:"spanZBlocks"`
}

func geometryFromWorld(g world.GridGeometry) Geometry {
	return Geometry{
		Width: g.Width, Height: g.Height,
		OriginX: g.OriginX, OriginZ: g.OriginZ,
		SpanXBlocks: g.SpanXBlocks, SpanZBlocks: g.SpanZBlocks,
	}
}

func (g Geometry) World() world.GridGeometry {
	return world.GridGeometry{
		Width: g.Width, Height: g.Height,
		OriginX: g.OriginX, OriginZ: g.OriginZ,
		SpanXBlocks: g.SpanXBlocks, SpanZBlocks: g.SpanZBlocks,
	}
}

type Source struct {
	Seed                    int64    `json:"seed"`
	Geometry                Geometry `json:"geometry"`
	SeaLevelNormalized      float64  `json:"seaLevelNormalized"`
	PlateCount              int      `json:"plateCount"`
	OceanFraction           float64  `json:"oceanFraction"`
	EdgeOceanBufferFraction float64  `json:"edgeOceanBufferFraction"`
	ContinentFragmentation  float64  `json:"continentFragmentation"`
	ContinentSeparation     float64  `json:"continentSeparation"`
}

type Document struct {
	Format             string  `json:"format"`
	FormatVersion      int     `json:"formatVersion"`
	ModelSchemaVersion int     `json:"modelSchemaVersion"`
	GeneratorVersion   string  `json:"generatorVersion"`
	FloatTolerance     float64 `json:"floatTolerance"`
	Source             Source  `json:"source"`
	Cases              []Case  `json:"cases"`
}

type Case struct {
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Description string  `json:"description"`
	X           float64 `json:"x"`
	Z           float64 `json:"z"`
	Expected    Sample  `json:"expected"`
}

type Sample struct {
	GroundHeight          float64  `json:"groundHeight"`
	WaterKind             string   `json:"waterKind"`
	WaterSurfaceDefined   bool     `json:"waterSurfaceDefined"`
	WaterSurfaceElevation *float64 `json:"waterSurfaceElevation,omitempty"`
	WaterDepth            float64  `json:"waterDepth"`
	OceanEligibility      float64  `json:"oceanEligibility"`
	RiverWeight           float64  `json:"riverWeight"`

	SourceNode   int     `json:"sourceNode"`
	SourceWeight float64 `json:"sourceWeight"`
	SourceLand   bool    `json:"sourceLand"`

	EcoRegion        string  `json:"ecoRegion"`
	RegionConfidence float64 `json:"regionConfidence"`

	BiomeFamily       string  `json:"biomeFamily"`
	TerrainVariant    string  `json:"terrainVariant"`
	SurfaceProfile    string  `json:"surfaceProfile"`
	VegetationProfile string  `json:"vegetationProfile"`
	ProfileConfidence float64 `json:"profileConfidence"`

	SemanticFallback bool `json:"semanticFallback"`
	SemanticMismatch bool `json:"semanticMismatch"`
}

type pointSpec struct {
	name        string
	category    string
	description string
	x           float64
	z           float64
}

func FixtureConfig() generator.Config {
	const cell = world.DefaultMacroSpacingBlocks
	return generator.Config{
		Seed: 22,
		Geometry: world.GridGeometry{
			Width:       48,
			Height:      30,
			OriginX:     -3072,
			OriginZ:     -2048,
			SpanXBlocks: 47 * cell,
			SpanZBlocks: 29 * cell,
		},
		SeaLevelNormalized:      world.DefaultSeaLevelNormalized,
		PlateCount:              12,
		OceanFraction:           0.68,
		EdgeOceanBufferFraction: 0.04,
		ContinentFragmentation:  phase1.DefaultContinentFragmentation,
		ContinentSeparation:     phase1.DefaultContinentSeparation,
	}
}

func GenerateFixture(ctx context.Context) (*generator.WorldModel, error) {
	return generator.GenerateWorld(ctx, FixtureConfig(), nil)
}

func Build(model *generator.WorldModel) (Document, error) {
	if model == nil || model.Surface == nil || model.Waterbed == nil || model.RuntimeProfile == nil {
		return Document{}, errors.New("goldenvectors: complete runtime world model is required")
	}
	runtimeWorld, err := runtimeworld.NewGenerated(model)
	if err != nil {
		return Document{}, err
	}
	specs, err := selectPoints(model, runtimeWorld)
	if err != nil {
		return Document{}, err
	}
	doc := Document{
		Format:             Format,
		FormatVersion:      FormatVersion,
		ModelSchemaVersion: worldmodelio.CurrentSchemaVersion,
		GeneratorVersion:   worldmodelio.CurrentGeneratorVersion,
		FloatTolerance:     FloatTolerance,
		Source: Source{
			Seed:                    model.Config.Seed,
			Geometry:                geometryFromWorld(model.Geometry),
			SeaLevelNormalized:      model.Config.SeaLevelNormalized,
			PlateCount:              model.Config.Phase1.PlateCount,
			OceanFraction:           model.Config.Phase1.OceanFraction,
			EdgeOceanBufferFraction: model.Config.Phase1.EdgeOceanBufferFraction,
			ContinentFragmentation:  model.Config.Phase1.ContinentFragmentation,
			ContinentSeparation:     model.Config.Phase1.ContinentSeparation,
		},
		Cases: make([]Case, 0, len(specs)),
	}
	for _, p := range specs {
		doc.Cases = append(doc.Cases, Case{
			Name:        p.name,
			Category:    p.category,
			Description: p.description,
			X:           p.x,
			Z:           p.z,
			Expected:    FromRuntimeSample(runtimeWorld.Sample(p.x, p.z)),
		})
	}
	return doc, nil
}

func FromRuntimeSample(s runtimeprofile.Sample) Sample {
	out := Sample{
		GroundHeight:        s.World.GroundHeight,
		WaterKind:           waterKindName(s.World.WaterKind),
		WaterSurfaceDefined: s.World.WaterSurfaceDefined,
		WaterDepth:          s.World.WaterDepth,
		OceanEligibility:    s.World.OceanEligibility,
		RiverWeight:         s.World.RiverWeight,
		SourceNode:          s.SourceNode,
		SourceWeight:        s.SourceWeight,
		SourceLand:          s.SourceLand,
		EcoRegion:           s.EcoRegion.String(),
		RegionConfidence:    s.RegionConfidence,
		BiomeFamily:         s.BiomeFamily.String(),
		TerrainVariant:      s.TerrainVariant.String(),
		SurfaceProfile:      s.SurfaceProfile.String(),
		VegetationProfile:   s.VegetationProfile.String(),
		ProfileConfidence:   s.ProfileConfidence,
		SemanticFallback:    s.SemanticFallback,
		SemanticMismatch:    s.SemanticMismatch,
	}
	if s.World.WaterSurfaceDefined {
		value := s.World.WaterSurfaceElevation
		out.WaterSurfaceElevation = &value
	}
	return out
}

func CompareSample(got, want Sample, tolerance float64) error {
	if got.WaterKind != want.WaterKind ||
		got.WaterSurfaceDefined != want.WaterSurfaceDefined ||
		got.SourceNode != want.SourceNode ||
		got.SourceLand != want.SourceLand ||
		got.EcoRegion != want.EcoRegion ||
		got.BiomeFamily != want.BiomeFamily ||
		got.TerrainVariant != want.TerrainVariant ||
		got.SurfaceProfile != want.SurfaceProfile ||
		got.VegetationProfile != want.VegetationProfile ||
		got.SemanticFallback != want.SemanticFallback ||
		got.SemanticMismatch != want.SemanticMismatch {
		return fmt.Errorf("discrete runtime fields differ: got=%+v want=%+v", got, want)
	}
	if (got.WaterSurfaceElevation == nil) != (want.WaterSurfaceElevation == nil) {
		return fmt.Errorf("water surface presence differs")
	}
	if got.WaterSurfaceElevation != nil && !near(*got.WaterSurfaceElevation, *want.WaterSurfaceElevation, tolerance) {
		return fmt.Errorf("water surface differs: got=%.17g want=%.17g", *got.WaterSurfaceElevation, *want.WaterSurfaceElevation)
	}
	values := []struct {
		name string
		got  float64
		want float64
	}{
		{"groundHeight", got.GroundHeight, want.GroundHeight},
		{"waterDepth", got.WaterDepth, want.WaterDepth},
		{"oceanEligibility", got.OceanEligibility, want.OceanEligibility},
		{"riverWeight", got.RiverWeight, want.RiverWeight},
		{"sourceWeight", got.SourceWeight, want.SourceWeight},
		{"regionConfidence", got.RegionConfidence, want.RegionConfidence},
		{"profileConfidence", got.ProfileConfidence, want.ProfileConfidence},
	}
	for _, value := range values {
		if !near(value.got, value.want, tolerance) {
			return fmt.Errorf("%s differs: got=%.17g want=%.17g tolerance=%g", value.name, value.got, value.want, tolerance)
		}
	}
	return nil
}

func selectPoints(model *generator.WorldModel, runtimeWorld *runtimeworld.Model) ([]pointSpec, error) {
	g := model.Geometry
	if err := g.Validate(); err != nil {
		return nil, err
	}
	endX := g.OriginX + g.SpanXBlocks
	endZ := g.OriginZ + g.SpanZBlocks
	midX := g.OriginX + g.SpanXBlocks/2
	midZ := g.OriginZ + g.SpanZBlocks/2

	points := []pointSpec{
		{"corner_nw", "outer-corner", "minimum X/Z model corner", g.OriginX, g.OriginZ},
		{"corner_ne", "outer-corner", "maximum X, minimum Z model corner", endX, g.OriginZ},
		{"corner_sw", "outer-corner", "minimum X, maximum Z model corner", g.OriginX, endZ},
		{"corner_se", "outer-corner", "maximum X/Z model corner", endX, endZ},
		{"edge_north", "outer-edge", "midpoint of minimum-Z model edge", midX, g.OriginZ},
		{"edge_south", "outer-edge", "midpoint of maximum-Z model edge", midX, endZ},
		{"edge_west", "outer-edge", "midpoint of minimum-X model edge", g.OriginX, midZ},
		{"edge_east", "outer-edge", "midpoint of maximum-X model edge", endX, midZ},
		{"negative_chunk_boundary", "chunk-boundary", "negative X/Z coordinate aligned to Minecraft 16-block chunk boundaries but not macro nodes", -1008, -496},
		{"zero_crossing", "coordinate-origin", "world coordinate origin inside the finite GeoWorld region", 0, 0},
	}

	macroX, macroZ := g.NodePosition(17, 11)
	points = append(points,
		pointSpec{"macro_node", "macro-grid", "exact macro-grid node", macroX, macroZ},
		pointSpec{"macro_cell_interior", "macro-grid", "non-node point inside the adjacent macro cell", macroX + 37.25, macroZ + 79.5},
	)

	if p, ok := firstMixedCoastPoint(model); ok {
		points = append(points, pointSpec{
			"mixed_coast",
			"coast",
			"center of the first macro cell whose corners cross the formal land/ocean mask",
			p[0], p[1],
		})
	} else {
		return nil, errors.New("goldenvectors: fixture contains no mixed coast cell")
	}

	foundRiver := false
	for _, segment := range model.Waterbed.Rivers {
		if !segment.ReceiverSurfaceDefined {
			continue
		}
		// A receiver-bound segment may now extend all the way into the ocean or
		// lake. Pick the furthest downstream point that is still Runtime River,
		// so the golden case keeps exercising river backwater semantics rather
		// than accidentally turning into an Ocean sample.
		for t := 0.95; t >= 0.05; t -= 0.05 {
			x := segment.X0 + (segment.X1-segment.X0)*t
			z := segment.Z0 + (segment.Z1-segment.Z0)*t
			if model.Runtime.Sample(x, z).WaterKind != reconcile.WaterRiver {
				continue
			}
			points = append(points, pointSpec{
				"river_backwater",
				"river-mouth",
				"furthest downstream Runtime River point on a segment with a propagated receiver surface",
				x,
				z,
			})
			foundRiver = true
			break
		}
		if foundRiver {
			break
		}
	}
	if !foundRiver {
		return nil, errors.New("goldenvectors: fixture contains no river with receiver surface")
	}

	if len(model.Waterbed.Lakes) == 0 {
		return nil, errors.New("goldenvectors: fixture contains no lake nodes")
	}
	foundLake := false
	for _, lake := range model.Waterbed.Lakes {
		sample := model.Runtime.Sample(lake.X, lake.Z)
		if sample.WaterKind != reconcile.WaterLake {
			continue
		}
		points = append(points, pointSpec{
			"lake_center",
			"lake",
			"center of a persisted lake node with an actual Runtime lake surface",
			lake.X, lake.Z,
		})
		foundLake = true
		break
	}
	if !foundLake {
		return nil, errors.New("goldenvectors: fixture contains no rendered lake center")
	}

	// All cases above are finite-model invariants. Keep validating them before
	// deliberately appending infinite-exterior cases.
	for _, point := range points {
		if point.x < g.OriginX-1e-9 || point.x > endX+1e-9 ||
			point.z < g.OriginZ-1e-9 || point.z > endZ+1e-9 {
			return nil, fmt.Errorf("goldenvectors: point %s lies outside fixture geometry: %.3f,%.3f", point.name, point.x, point.z)
		}
	}

	transition := math.Max(
		2048,
		0.5*(g.SpacingX()+g.SpacingZ())*24*model.Config.WorldProfile.Parameters.TransitionWidth,
	)
	points = append(points,
		pointSpec{
			"outer_far_ocean",
			"outer-extension",
			"far-ocean sample beyond the finite east edge",
			endX + 3*transition,
			midZ,
		},
		pointSpec{
			"outer_polar_ocean",
			"outer-extension",
			"high-latitude sample beyond the north edge that must resolve to frozen ocean",
			midX,
			g.OriginZ - 2*transition,
		},
	)
	if island, ok := firstOuterIsland(runtimeWorld, g, transition); ok {
		points = append(points, pointSpec{
			"outer_sparse_island",
			"outer-extension",
			"deterministically located sparse offshore island outside the finite model",
			island[0],
			island[1],
		})
	} else {
		return nil, errors.New("goldenvectors: fixture contains no sparse outer island")
	}
	return points, nil
}

func firstOuterIsland(
	runtimeWorld *runtimeworld.Model,
	g world.GridGeometry,
	transition float64,
) ([2]float64, bool) {
	minZ := g.OriginZ + 0.20*g.SpanZBlocks
	maxZ := g.OriginZ + 0.80*g.SpanZBlocks
	minDistance := 0.20 * transition
	maxDistance := 2.40 * transition
	step := math.Max(48, math.Min(g.SpacingX(), g.SpacingZ())*0.5)
	for distance := minDistance; distance <= maxDistance; distance += step {
		for z := minZ; z <= maxZ; z += step {
			for _, x := range []float64{
				g.OriginX - distance,
				g.OriginX + g.SpanXBlocks + distance,
			} {
				sample := runtimeWorld.Sample(x, z)
				if sample.SourceLand && sample.World.WaterKind == reconcile.WaterNone {
					return [2]float64{x, z}, true
				}
			}
		}
	}
	return [2]float64{}, false
}

func firstMixedCoastPoint(model *generator.WorldModel) ([2]float64, bool) {
	g := model.Geometry
	for z := 0; z < g.Height-1; z++ {
		for x := 0; x < g.Width-1; x++ {
			ids := [4]int{
				z*g.Width + x,
				z*g.Width + x + 1,
				(z+1)*g.Width + x,
				(z+1)*g.Width + x + 1,
			}
			land := 0
			for _, i := range ids {
				if model.Surface.LandMask[i] {
					land++
				}
			}
			if land == 0 || land == 4 {
				continue
			}
			return [2]float64{
				g.OriginX + (float64(x)+0.5)*g.SpacingX(),
				g.OriginZ + (float64(z)+0.5)*g.SpacingZ(),
			}, true
		}
	}
	return [2]float64{}, false
}

func waterKindName(kind reconcile.WaterKind) string {
	switch kind {
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

func near(a, b, tolerance float64) bool {
	if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) {
		return false
	}
	return math.Abs(a-b) <= tolerance
}
