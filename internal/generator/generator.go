package generator

import (
	"context"
	"errors"
	"fmt"
	"math"

	"geoworld/internal/climate"
	"geoworld/internal/ecology"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
	"geoworld/internal/reconcile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/terrain"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
	"geoworld/internal/worldsample"
)

// GenerateWorld executes the formal world-generation pipeline.
//
// Shared world contracts are resolved once here:
//   - seed
//   - GridGeometry
//   - normalized sea level
//
// Stage-specific compatibility defaults remain inside each stage for now, but
// their resolved configs are collected into WorldModel.Config so Phase 6.3 can
// persist the exact configuration that produced a model.
func GenerateWorld(ctx context.Context, cfg Config, progress ProgressFunc) (*WorldModel, error) {
	resolved, err := resolveTopLevelConfig(cfg)
	if err != nil {
		return nil, err
	}

	reporter := progressReporter{fn: progress, total: len(orderedStages)}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}

	phase1Cfg := phase1.Config{
		Seed:                    resolved.Seed,
		Width:                   resolved.Geometry.Width,
		Height:                  resolved.Geometry.Height,
		OriginX:                 resolved.Geometry.OriginX,
		OriginZ:                 resolved.Geometry.OriginZ,
		SpanXBlocks:             resolved.Geometry.SpanXBlocks,
		SpanZBlocks:             resolved.Geometry.SpanZBlocks,
		SeaLevelNormalized:      resolved.SeaLevelNormalized,
		PlateCount:              resolved.PlateCount,
		OceanFraction:           resolved.OceanFraction,
		EdgeOceanBufferFraction: resolved.EdgeOceanBufferFraction,
	}
	w, err := phase1.GenerateChecked(phase1Cfg)
	if err != nil {
		return nil, fmt.Errorf("generator: phase1: %w", err)
	}
	reporter.done(StagePhase1)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	climateCfg := resolved.Climate
	climateCfg.SeaLevelNormalized = resolved.SeaLevelNormalized
	c, err := climate.Generate(climate.InputFromPhase1(w), climateCfg)
	if err != nil {
		return nil, fmt.Errorf("generator: climate: %w", err)
	}
	reporter.done(StageClimate)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	hydrologyCfg := resolved.Hydrology
	hydrologyCfg.SeaLevelNormalized = resolved.SeaLevelNormalized
	h, err := hydrology.Generate(hydrology.InputFromWorld(w, c), hydrologyCfg)
	if err != nil {
		return nil, fmt.Errorf("generator: hydrology: %w", err)
	}
	reporter.done(StageHydrology)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	erosionCfg := resolved.Erosion
	erosionCfg.SeaLevelNormalized = resolved.SeaLevelNormalized
	e, err := erosion.Generate(erosion.InputFromWorld(w, c, h), erosionCfg)
	if err != nil {
		return nil, fmt.Errorf("generator: erosion: %w", err)
	}
	reporter.done(StageErosion)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	landformCfg := resolved.Landform
	landformCfg.SeaLevelNormalized = resolved.SeaLevelNormalized
	l, err := landform.Generate(landform.InputFromWorld(w, e), landformCfg)
	if err != nil {
		return nil, fmt.Errorf("generator: landform: %w", err)
	}
	reporter.done(StageLandform)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	geomorphCfg := resolved.Geomorph
	geomorphCfg.SeaLevelNormalized = resolved.SeaLevelNormalized
	g, err := geomorph.Generate(geomorph.InputFromModels(h, e, l), geomorphCfg)
	if err != nil {
		return nil, fmt.Errorf("generator: geomorph: %w", err)
	}
	reporter.done(StageGeomorph)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	eco, err := ecology.Generate(ecology.InputFromModels(c, h, e, l, g), resolved.Ecology)
	if err != nil {
		return nil, fmt.Errorf("generator: ecology: %w", err)
	}
	reporter.done(StageEcology)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	p, err := mcprofile.Generate(mcprofile.InputFromModels(eco, l, g))
	if err != nil {
		return nil, fmt.Errorf("generator: mcprofile: %w", err)
	}
	reporter.done(StageProfile)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	terrainCfg := resolved.Terrain
	terrainCfg.Seed = resolved.Seed
	// Formal adapters already carry Geometry. Legacy geometry fields are cleared
	// so the unified generator has exactly one geometry source of truth.
	terrainCfg.CellSizeBlocks = 0
	terrainCfg.OriginX = 0
	terrainCfg.OriginZ = 0
	t, err := terrain.Generate(terrain.InputFromModels(e, l, g), terrainCfg)
	if err != nil {
		return nil, fmt.Errorf("generator: terrain: %w", err)
	}
	reporter.done(StageTerrain)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	rec, err := reconcile.Build(reconcile.Input{
		Geometry:           w.Geometry,
		SeaLevelNormalized: w.Surface.SeaLevelNormalized,
		MacroSurface:       w.Surface,
		MacroElevation:     w.Elevation,
		Hydrology:          h,
		Terrain:            t,
	}, resolved.Reconcile)
	if err != nil {
		return nil, fmt.Errorf("generator: reconcile: %w", err)
	}
	coastTerrain, err := terrain.WithHeightCorrection(t, rec.CoastCorrection)
	if err != nil {
		return nil, fmt.Errorf("generator: coast terrain: %w", err)
	}
	if err := rec.EvaluateCoastApplication(coastTerrain, w.Surface); err != nil {
		return nil, fmt.Errorf("generator: evaluate coast terrain: %w", err)
	}
	if err := rec.RebuildRiverConstraintsFromTerrain(coastTerrain, w.Elevation, h); err != nil {
		return nil, fmt.Errorf("generator: rebuild river constraints: %w", err)
	}
	reporter.done(StageReconcile)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	wb, err := waterbed.Build(coastTerrain, rec, h, resolved.Waterbed)
	if err != nil {
		return nil, fmt.Errorf("generator: waterbed: %w", err)
	}
	reporter.done(StageWaterbed)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	runtimeModel, err := worldsample.Build(w.Surface, wb, resolved.WorldSample)
	if err != nil {
		return nil, fmt.Errorf("generator: worldsample: %w", err)
	}
	reporter.done(StageWorldSample)

	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	runtimeProfile, err := runtimeprofile.Build(w.Geometry, eco, p, runtimeModel)
	if err != nil {
		return nil, fmt.Errorf("generator: runtimeprofile: %w", err)
	}
	reporter.done(StageRuntimeProfile)

	model := &WorldModel{
		Geometry:       w.Geometry,
		Surface:        w.Surface,
		Phase1:         w,
		Climate:        c,
		Hydrology:      h,
		Erosion:        e,
		Landform:       l,
		Geomorph:       g,
		Ecology:        eco,
		Profile:        p,
		Terrain:        t,
		CoastTerrain:   coastTerrain,
		Reconciliation: rec,
		Waterbed:       wb,
		Runtime:        runtimeModel,
		RuntimeProfile: runtimeProfile,
	}
	model.Config = ResolvedConfig{
		Seed:               resolved.Seed,
		Geometry:           w.Geometry,
		SeaLevelNormalized: w.Surface.SeaLevelNormalized,
		Phase1:             w.Config,
		Climate:            c.Config,
		Hydrology:          h.Config,
		Erosion:            e.Config,
		Landform:           l.Config,
		Geomorph:           g.Config,
		Ecology:            eco.Config,
		Terrain:            t.Config,
		Reconcile:          rec.Config,
		Waterbed:           wb.Config,
		WorldSample:        runtimeModel.Config,
	}
	return model, nil
}

type resolvedTopLevel struct {
	Config
}

func resolveTopLevelConfig(cfg Config) (resolvedTopLevel, error) {
	if cfg.Geometry.Width == 0 && cfg.Geometry.Height == 0 &&
		cfg.Geometry.OriginX == 0 && cfg.Geometry.OriginZ == 0 &&
		cfg.Geometry.SpanXBlocks == 0 && cfg.Geometry.SpanZBlocks == 0 {
		return resolvedTopLevel{}, errors.New("generator: geometry is required")
	}
	if _, err := cfg.Geometry.SquareCellSize(); err != nil {
		return resolvedTopLevel{}, fmt.Errorf("generator: geometry: %w", err)
	}

	if cfg.SeaLevelNormalized == 0 {
		cfg.SeaLevelNormalized = world.DefaultSeaLevelNormalized
	}
	if math.IsNaN(cfg.SeaLevelNormalized) || math.IsInf(cfg.SeaLevelNormalized, 0) ||
		cfg.SeaLevelNormalized <= 0 || cfg.SeaLevelNormalized >= 1 {
		return resolvedTopLevel{}, errors.New("generator: sea level must be finite and between 0 and 1")
	}

	if cfg.PlateCount == 0 {
		cfg.PlateCount = 18
	}
	if cfg.PlateCount < 2 {
		return resolvedTopLevel{}, errors.New("generator: plate count must be at least 2")
	}
	if cfg.OceanFraction == 0 {
		cfg.OceanFraction = 0.68
	}
	if math.IsNaN(cfg.OceanFraction) || math.IsInf(cfg.OceanFraction, 0) ||
		cfg.OceanFraction <= 0 || cfg.OceanFraction >= 1 {
		return resolvedTopLevel{}, errors.New("generator: ocean fraction must be finite and between 0 and 1")
	}
	if cfg.EdgeOceanBufferFraction == 0 {
		cfg.EdgeOceanBufferFraction = 0.04
	}
	if math.IsNaN(cfg.EdgeOceanBufferFraction) || math.IsInf(cfg.EdgeOceanBufferFraction, 0) ||
		cfg.EdgeOceanBufferFraction <= 0 || cfg.EdgeOceanBufferFraction >= 0.20 {
		return resolvedTopLevel{}, errors.New("generator: edge ocean buffer fraction must be finite and between 0 and 0.20")
	}

	if err := rejectSeaLevelConflict("climate", cfg.Climate.SeaLevelNormalized, cfg.SeaLevelNormalized); err != nil {
		return resolvedTopLevel{}, err
	}
	if err := rejectSeaLevelConflict("hydrology", cfg.Hydrology.SeaLevelNormalized, cfg.SeaLevelNormalized); err != nil {
		return resolvedTopLevel{}, err
	}
	if err := rejectSeaLevelConflict("erosion", cfg.Erosion.SeaLevelNormalized, cfg.SeaLevelNormalized); err != nil {
		return resolvedTopLevel{}, err
	}
	if err := rejectSeaLevelConflict("landform", cfg.Landform.SeaLevelNormalized, cfg.SeaLevelNormalized); err != nil {
		return resolvedTopLevel{}, err
	}
	if err := rejectSeaLevelConflict("geomorph", cfg.Geomorph.SeaLevelNormalized, cfg.SeaLevelNormalized); err != nil {
		return resolvedTopLevel{}, err
	}

	if cfg.Terrain.Seed != 0 && cfg.Terrain.Seed != cfg.Seed {
		return resolvedTopLevel{}, errors.New("generator: terrain seed conflicts with top-level seed")
	}
	if cfg.Terrain.CellSizeBlocks != 0 || cfg.Terrain.OriginX != 0 || cfg.Terrain.OriginZ != 0 {
		return resolvedTopLevel{}, errors.New("generator: terrain legacy geometry fields are not allowed; use top-level geometry")
	}

	return resolvedTopLevel{Config: cfg}, nil
}

func rejectSeaLevelConflict(stage string, stageValue, sharedValue float64) error {
	if stageValue == 0 {
		return nil
	}
	if math.IsNaN(stageValue) || math.IsInf(stageValue, 0) || math.Abs(stageValue-sharedValue) > 1e-12 {
		return fmt.Errorf("generator: %s sea level conflicts with top-level sea level", stage)
	}
	return nil
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return fmt.Errorf("generator: %w", ctx.Err())
	default:
		return nil
	}
}

type progressReporter struct {
	fn        ProgressFunc
	completed int
	total     int
}

func (p *progressReporter) done(stage Stage) {
	p.completed++
	if p.fn != nil {
		p.fn(Progress{Stage: stage, Completed: p.completed, Total: p.total})
	}
}
