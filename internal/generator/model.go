package generator

import (
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

// Config is the single formal input for the world-generation pipeline.
// Geometry, seed and sea level are shared contracts and must not diverge by stage.
type Config struct {
	Seed               int64
	Geometry           world.GridGeometry
	SeaLevelNormalized float64

	PlateCount              int
	OceanFraction           float64
	EdgeOceanBufferFraction float64
	ContinentFragmentation  float64
	ContinentSeparation     float64

	Climate     climate.Config
	Hydrology   hydrology.Config
	Erosion     erosion.Config
	Landform    landform.Config
	Geomorph    geomorph.Config
	Ecology     ecology.Config
	Terrain     terrain.Config
	Reconcile   reconcile.Config
	Waterbed    waterbed.Config
	WorldSample worldsample.Config
}

// ResolvedConfig records the values actually used after every stage has applied
// its compatibility defaults. This is the configuration that later persistence
// must store, rather than the possibly sparse Config supplied by a caller.
type ResolvedConfig struct {
	Seed               int64
	Geometry           world.GridGeometry
	SeaLevelNormalized float64

	Phase1      phase1.Config
	Climate     climate.Config
	Hydrology   hydrology.Config
	Erosion     erosion.Config
	Landform    landform.Config
	Geomorph    geomorph.Config
	Ecology     ecology.Config
	Terrain     terrain.Config
	Reconcile   reconcile.Config
	Waterbed    waterbed.Config
	WorldSample worldsample.Config
}

// WorldModel is the unified in-memory result. Phase 6.3 will define which
// runtime fields are persisted; for Phase 6.2 the stage models remain attached
// so consistency correction can reason across the whole causal chain.
type WorldModel struct {
	Config   ResolvedConfig
	Geometry world.GridGeometry
	Surface  *world.Surface

	Phase1         *phase1.World
	Climate        *climate.Model
	Hydrology      *hydrology.Model
	Erosion        *erosion.Model
	Landform       *landform.Model
	Geomorph       *geomorph.Model
	Ecology        *ecology.Model
	Profile        *mcprofile.Model
	Terrain        *terrain.Model
	CoastTerrain   *terrain.Model
	Reconciliation *reconcile.Model
	Waterbed       *waterbed.Model
	Runtime        *worldsample.Model
	RuntimeProfile *runtimeprofile.Model
}

// Stage identifies one deterministic pipeline boundary for progress reporting.
type Stage string

const (
	StagePhase1         Stage = "phase1"
	StageClimate        Stage = "climate"
	StageHydrology      Stage = "hydrology"
	StageErosion        Stage = "erosion"
	StageLandform       Stage = "landform"
	StageGeomorph       Stage = "geomorph"
	StageEcology        Stage = "ecology"
	StageProfile        Stage = "mcprofile"
	StageTerrain        Stage = "terrain"
	StageReconcile      Stage = "reconcile"
	StageWaterbed       Stage = "waterbed"
	StageWorldSample    Stage = "worldsample"
	StageRuntimeProfile Stage = "runtimeprofile"
)

var orderedStages = [...]Stage{
	StagePhase1,
	StageClimate,
	StageHydrology,
	StageErosion,
	StageLandform,
	StageGeomorph,
	StageEcology,
	StageProfile,
	StageTerrain,
	StageReconcile,
	StageWaterbed,
	StageWorldSample,
	StageRuntimeProfile,
}

type Progress struct {
	Stage     Stage
	Completed int
	Total     int
}

type ProgressFunc func(Progress)
