package worldmodelio

import (
	"errors"

	"geoworld/internal/ecology"
	"geoworld/internal/generator"
	"geoworld/internal/mcprofile"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/terrain"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
	"geoworld/internal/worldsample"
)

const (
	CurrentSchemaVersion    = 1
	CurrentGeneratorVersion = "phase6.3-v1"
	FormatMagic             = "geoworld-model"
	FileEncoding            = "gzip+json"
	ChecksumAlgorithm       = "sha256"
)

var (
	ErrTargetExists                = errors.New("worldmodelio: target already exists")
	ErrUnsupportedSchema           = errors.New("worldmodelio: unsupported schema version")
	ErrUnsupportedGeneratorVersion = errors.New("worldmodelio: unsupported generator version")
	ErrChecksumMismatch            = errors.New("worldmodelio: payload checksum mismatch")
)

type Manifest struct {
	Magic              string             `json:"magic"`
	SchemaVersion      int                `json:"schemaVersion"`
	GeneratorVersion   string             `json:"generatorVersion"`
	FileEncoding       string             `json:"fileEncoding"`
	ChecksumAlgorithm  string             `json:"checksumAlgorithm"`
	PayloadSHA256      string             `json:"payloadSha256"`
	PayloadBytes       int64              `json:"payloadBytes"`
	Seed               int64              `json:"seed"`
	Geometry           world.GridGeometry `json:"geometry"`
	SeaLevelNormalized float64            `json:"seaLevelNormalized"`
}

type Inspection struct {
	Manifest  Manifest `json:"manifest"`
	FileBytes int64    `json:"fileBytes"`
}

// RuntimeModel is the reopened, Runtime-ready subset of generator.WorldModel.
// Expensive Phase1..Geomorph debug/intermediate models are intentionally absent.
type RuntimeModel struct {
	Manifest Manifest
	Config   generator.ResolvedConfig

	Geometry world.GridGeometry
	Surface  *world.Surface

	CoastTerrain   *terrain.Model
	Waterbed       *waterbed.Model
	Runtime        *worldsample.Model
	Ecology        *ecology.Model
	Profile        *mcprofile.Model
	RuntimeProfile *runtimeprofile.Model
	ClimatePreview *ClimatePreview
}

type ClimatePreview struct {
	AnnualMeanTempC []float64
	AnnualPrecipMM  []float64
	MonsoonIndex    []float64
	WindX           []float64
	WindZ           []float64
}

func (m *RuntimeModel) Sample(worldX, worldZ float64) runtimeprofile.Sample {
	return m.RuntimeProfile.Sample(worldX, worldZ)
}

type fileEnvelope struct {
	Manifest Manifest `json:"manifest"`
	Payload  payload  `json:"payload"`
}

type payload struct {
	Config         generator.ResolvedConfig `json:"config"`
	Surface        surfaceSnapshot          `json:"surface"`
	Terrain        terrainSnapshot          `json:"terrain"`
	Waterbed       waterbedSnapshot         `json:"waterbed"`
	ClimatePreview *climatePreviewSnapshot  `json:"climatePreview,omitempty"`
	Ecology        ecologySnapshot          `json:"ecology"`
	Profile        profileSnapshot          `json:"profile"`
}

type surfaceSnapshot struct {
	LandMask         []bool `json:"landMask"`
	EnclosedBelowSea []bool `json:"enclosedBelowSea"`
}

type terrainSnapshot struct {
	Config terrain.Config `json:"config"`

	BaseElevation       []float64 `json:"baseElevation"`
	RollingPlainWeight  []float64 `json:"rollingPlainWeight"`
	HillWeight          []float64 `json:"hillWeight"`
	FoothillWeight      []float64 `json:"foothillWeight"`
	MountainWeight      []float64 `json:"mountainWeight"`
	PlateauWeight       []float64 `json:"plateauWeight"`
	ValleyWeight        []float64 `json:"valleyWeight"`
	FloodplainWeight    []float64 `json:"floodplainWeight"`
	AlluvialPlainWeight []float64 `json:"alluvialPlainWeight"`
	AlluvialFanWeight   []float64 `json:"alluvialFanWeight"`
	CanyonWeight        []float64 `json:"canyonWeight"`
	ValleyFloorWidth    []float64 `json:"valleyFloorWidth"`
	DepositionPotential []float64 `json:"depositionPotential"`
	HeightCorrection    []float64 `json:"heightCorrection,omitempty"`
}

type waterbedSnapshot struct {
	Config waterbed.Config         `json:"config"`
	Rivers []waterbed.RiverSegment `json:"rivers"`
	Lakes  []waterbed.LakeNode     `json:"lakes"`
}

type climatePreviewSnapshot struct {
	AnnualMeanTempC []float64 `json:"annualMeanTempC,omitempty"`
	AnnualPrecipMM  []float64 `json:"annualPrecipMM,omitempty"`
	MonsoonIndex    []float64 `json:"monsoonIndex,omitempty"`
	WindX           []float64 `json:"windX,omitempty"`
	WindZ           []float64 `json:"windZ,omitempty"`
}

type ecologySnapshot struct {
	LandMask         []bool           `json:"landMask"`
	EcoRegion        []ecology.Region `json:"ecoRegion"`
	RegionConfidence []float64        `json:"regionConfidence"`
}

type profileSnapshot struct {
	LandMask          []bool                        `json:"landMask"`
	BiomeFamily       []mcprofile.BiomeFamily       `json:"biomeFamily"`
	TerrainVariant    []mcprofile.TerrainVariant    `json:"terrainVariant"`
	SurfaceProfile    []mcprofile.SurfaceProfile    `json:"surfaceProfile"`
	VegetationProfile []mcprofile.VegetationProfile `json:"vegetationProfile"`
	ProfileConfidence []float64                     `json:"profileConfidence"`
}
