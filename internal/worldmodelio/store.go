package worldmodelio

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"

	"geoworld/internal/ecology"
	"geoworld/internal/generator"
	"geoworld/internal/mcprofile"
	"geoworld/internal/phase1"
	"geoworld/internal/runtimeprofile"
	"geoworld/internal/sampling"
	"geoworld/internal/terrain"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
	"geoworld/internal/worldsample"
)

// Save writes a complete new world-model file. V1 deliberately refuses to
// replace an existing target. Publication uses a hard link from a completely
// written same-directory temporary file, so a failed save cannot leave a
// partially-written target or destroy an existing model.
func Save(path string, model *generator.WorldModel) error {
	env, err := envelopeFromWorldModel(path, model)
	if err != nil {
		return err
	}
	tmpPath, cleanup, err := writeCompletedTemp(path, env)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrTargetExists
		}
		return fmt.Errorf("worldmodelio: publish completed model: %w", err)
	}
	return nil
}

// Replace atomically publishes a complete model over an existing target. The
// old target remains untouched until the fully-written same-directory
// temporary file has been flushed and closed. Platform-specific replacement is
// used for the final filesystem operation; callers never delete the old model
// first.
func Replace(path string, model *generator.WorldModel) error {
	if path == "" {
		return errors.New("worldmodelio: empty save path")
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("worldmodelio: replace target does not exist: %w", err)
		}
		return fmt.Errorf("worldmodelio: inspect replace target: %w", err)
	}

	env, err := envelopeFromWorldModel(path, model)
	if err != nil {
		return err
	}
	tmpPath, cleanup, err := writeCompletedTemp(path, env)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := replaceCompletedFile(tmpPath, path); err != nil {
		return fmt.Errorf("worldmodelio: replace completed model: %w", err)
	}
	return nil
}

func envelopeFromWorldModel(path string, model *generator.WorldModel) (fileEnvelope, error) {
	if path == "" {
		return fileEnvelope{}, errors.New("worldmodelio: empty save path")
	}
	p, err := snapshotFromWorldModel(model)
	if err != nil {
		return fileEnvelope{}, err
	}
	payloadJSON, err := json.Marshal(p)
	if err != nil {
		return fileEnvelope{}, fmt.Errorf("worldmodelio: encode payload checksum: %w", err)
	}
	sum := sha256.Sum256(payloadJSON)
	parameterJSON, err := json.Marshal(model.Config.WorldProfile.Parameters)
	if err != nil {
		return fileEnvelope{}, fmt.Errorf("worldmodelio: encode parameter snapshot checksum: %w", err)
	}
	parameterSum := sha256.Sum256(parameterJSON)

	manifest := Manifest{
		Magic:                   FormatMagic,
		SchemaVersion:           CurrentSchemaVersion,
		GeneratorVersion:        CurrentGeneratorVersion,
		WorldProfileVersion:     model.Config.WorldProfile.WorldProfileVersion,
		GenerationRulesVersion:  model.Config.WorldProfile.GenerationRulesVersion,
		WorldModelVersion:       model.Config.WorldProfile.WorldModelVersion,
		Preset:                  model.Config.WorldProfile.Preset,
		ParameterSnapshotSHA256: hex.EncodeToString(parameterSum[:]),
		FileEncoding:            FileEncoding,
		ChecksumAlgorithm:       ChecksumAlgorithm,
		PayloadSHA256:           hex.EncodeToString(sum[:]),
		PayloadBytes:            int64(len(payloadJSON)),
		Seed:                    model.Config.Seed,
		Geometry:                model.Geometry,
		SeaLevelNormalized:      model.Config.SeaLevelNormalized,
	}
	return fileEnvelope{Manifest: manifest, Payload: p}, nil
}

func writeCompletedTemp(path string, env fileEnvelope) (string, func(), error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, fmt.Errorf("worldmodelio: create save directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return "", nil, fmt.Errorf("worldmodelio: create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}

	gz := gzip.NewWriter(tmp)
	enc := json.NewEncoder(gz)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(env); err != nil {
		_ = gz.Close()
		cleanup()
		return "", nil, fmt.Errorf("worldmodelio: encode model: %w", err)
	}
	if err := gz.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("worldmodelio: finalize compression: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("worldmodelio: sync temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("worldmodelio: close temporary file: %w", err)
	}
	return tmpPath, cleanup, nil
}

func Inspect(path string) (Inspection, error) {
	if path == "" {
		return Inspection{}, errors.New("worldmodelio: empty inspect path")
	}
	info, err := os.Stat(path)
	if err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: inspect model file: %w", err)
	}
	f, err := os.Open(path)
	if err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: open model: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: open compressed model: %w", err)
	}
	defer gz.Close()

	dec := json.NewDecoder(gz)
	start, err := dec.Token()
	if err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: read envelope start: %w", err)
	}
	if d, ok := start.(json.Delim); !ok || d != '{' {
		return Inspection{}, errors.New("worldmodelio: invalid envelope")
	}
	if !dec.More() {
		return Inspection{}, errors.New("worldmodelio: empty envelope")
	}
	key, err := dec.Token()
	if err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: read manifest key: %w", err)
	}
	if key != "manifest" {
		return Inspection{}, errors.New("worldmodelio: manifest must be the first envelope field")
	}
	var manifest Manifest
	if err := dec.Decode(&manifest); err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: decode manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		return Inspection{}, err
	}
	return Inspection{Manifest: manifest, FileBytes: info.Size()}, nil
}

// Validate verifies file-level version/header/checksum and payload/config
// contracts without rebuilding Terrain/Waterbed/Runtime indexes.
func Validate(path string) (Inspection, error) {
	inspection, err := Inspect(path)
	if err != nil {
		return Inspection{}, err
	}
	f, err := os.Open(path)
	if err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: open model: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: open compressed model: %w", err)
	}
	defer gz.Close()

	var raw struct {
		Manifest Manifest        `json:"manifest"`
		Payload  json.RawMessage `json:"payload"`
	}
	dec := json.NewDecoder(gz)
	if err := dec.Decode(&raw); err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: decode validation envelope: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return Inspection{}, errors.New("worldmodelio: multiple JSON values in model file")
		}
		return Inspection{}, fmt.Errorf("worldmodelio: trailing model data: %w", err)
	}
	if raw.Manifest != inspection.Manifest {
		return Inspection{}, errors.New("worldmodelio: manifest changed during validation")
	}
	sum := sha256.Sum256(raw.Payload)
	if inspection.Manifest.PayloadBytes != int64(len(raw.Payload)) ||
		inspection.Manifest.PayloadSHA256 != hex.EncodeToString(sum[:]) {
		return Inspection{}, ErrChecksumMismatch
	}
	var p payload
	if err := json.Unmarshal(raw.Payload, &p); err != nil {
		return Inspection{}, fmt.Errorf("worldmodelio: decode validation payload: %w", err)
	}
	if err := validateEnvelopeContracts(fileEnvelope{Manifest: inspection.Manifest, Payload: p}); err != nil {
		return Inspection{}, err
	}
	return inspection, nil
}

func Load(path string) (*RuntimeModel, error) {
	// Validate hashes against the exact persisted payload bytes before decoding
	// into the current struct shape. Re-marshalling a legacy payload after a
	// schema-compatible struct extension would otherwise introduce zero-value
	// fields and create a false checksum mismatch.
	inspection, err := Validate(path)
	if err != nil {
		return nil, err
	}
	env, err := readEnvelope(path)
	if err != nil {
		return nil, err
	}
	if env.Manifest != inspection.Manifest {
		return nil, errors.New("worldmodelio: manifest changed between validation and load")
	}

	g := env.Manifest.Geometry
	n, _ := world.NodeCount(g.Width, g.Height)
	surface := &world.Surface{
		Width:              g.Width,
		Height:             g.Height,
		SeaLevelNormalized: env.Manifest.SeaLevelNormalized,
		LandMask:           append([]bool(nil), env.Payload.Surface.LandMask...),
		EnclosedBelowSea:   append([]bool(nil), env.Payload.Surface.EnclosedBelowSea...),
	}
	if len(surface.LandMask) != n || len(surface.EnclosedBelowSea) != n {
		return nil, errors.New("worldmodelio: persisted surface size does not match geometry")
	}
	for i := range surface.LandMask {
		if surface.EnclosedBelowSea[i] && !surface.LandMask[i] {
			return nil, errors.New("worldmodelio: enclosed-below-sea node cannot be ocean")
		}
	}

	coastTerrain, err := restoreTerrain(g, env.Payload.Terrain)
	if err != nil {
		return nil, err
	}
	wb, err := waterbed.Restore(
		coastTerrain,
		env.Payload.Waterbed.Config,
		env.Payload.Waterbed.Rivers,
		env.Payload.Waterbed.Lakes,
	)
	if err != nil {
		return nil, fmt.Errorf("worldmodelio: restore waterbed: %w", err)
	}
	runtimeModel, err := worldsample.Build(surface, wb, env.Payload.Config.WorldSample)
	if err != nil {
		return nil, fmt.Errorf("worldmodelio: restore runtime water: %w", err)
	}

	eco, err := restoreEcology(g, surface, env.Payload.Config.Ecology, env.Payload.Ecology)
	if err != nil {
		return nil, err
	}
	profile, err := restoreProfile(g, surface, env.Payload.Profile)
	if err != nil {
		return nil, err
	}
	runtimeProfile, err := runtimeprofile.BuildWithContinuity(g, eco, profile, runtimeModel, env.Payload.Config.WorldProfile.Parameters.BiomeContinuity)
	if err != nil {
		return nil, fmt.Errorf("worldmodelio: restore runtime profile: %w", err)
	}
	hydrologyQuery, err := restoreHydrologyQuery(n, env.Payload.HydrologyQuery)
	if err != nil {
		return nil, err
	}
	tectonicQuery, err := restoreTectonicQuery(n, env.Payload.TectonicQuery)
	if err != nil {
		return nil, err
	}

	return &RuntimeModel{
		Manifest:       env.Manifest,
		Config:         env.Payload.Config,
		Geometry:       g,
		Surface:        surface,
		CoastTerrain:   coastTerrain,
		Waterbed:       wb,
		Runtime:        runtimeModel,
		Ecology:        eco,
		Profile:        profile,
		RuntimeProfile: runtimeProfile,
		ClimatePreview: restoreClimatePreview(env.Payload.ClimatePreview),
		HydrologyQuery: hydrologyQuery,
		TectonicQuery:  tectonicQuery,
	}, nil
}

func readEnvelope(path string) (fileEnvelope, error) {
	f, err := os.Open(path)
	if err != nil {
		return fileEnvelope{}, fmt.Errorf("worldmodelio: open model: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fileEnvelope{}, fmt.Errorf("worldmodelio: open compressed model: %w", err)
	}
	defer gz.Close()

	var env fileEnvelope
	dec := json.NewDecoder(gz)
	if err := dec.Decode(&env); err != nil {
		return fileEnvelope{}, fmt.Errorf("worldmodelio: decode model: %w", err)
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fileEnvelope{}, errors.New("worldmodelio: multiple JSON values in model file")
		}
		return fileEnvelope{}, fmt.Errorf("worldmodelio: trailing model data: %w", err)
	}
	return env, nil
}

func validateManifest(m Manifest) error {
	if m.Magic != FormatMagic || m.FileEncoding != FileEncoding ||
		m.ChecksumAlgorithm != ChecksumAlgorithm {
		return errors.New("worldmodelio: invalid model format header")
	}
	if m.SchemaVersion != CurrentSchemaVersion {
		return ErrUnsupportedSchema
	}
	if m.GeneratorVersion != CurrentGeneratorVersion {
		return ErrUnsupportedGeneratorVersion
	}
	identityPresent := m.WorldProfileVersion != "" || m.GenerationRulesVersion != "" ||
		m.WorldModelVersion != "" || m.Preset != "" || m.ParameterSnapshotSHA256 != ""
	if identityPresent {
		if m.WorldProfileVersion == "" || m.GenerationRulesVersion == "" ||
			m.WorldModelVersion == "" || m.Preset == "" ||
			len(m.ParameterSnapshotSHA256) != sha256.Size*2 {
			return errors.New("worldmodelio: incomplete world profile identity")
		}
	}
	if err := m.Geometry.Validate(); err != nil {
		return fmt.Errorf("worldmodelio: invalid manifest geometry: %w", err)
	}
	if !finite(m.SeaLevelNormalized) || m.SeaLevelNormalized <= 0 || m.SeaLevelNormalized >= 1 {
		return errors.New("worldmodelio: invalid manifest sea level")
	}
	if len(m.PayloadSHA256) != sha256.Size*2 || m.PayloadBytes <= 0 {
		return errors.New("worldmodelio: invalid payload metadata")
	}
	return nil
}

func validateEnvelopeContracts(env fileEnvelope) error {
	m := env.Manifest
	c := env.Payload.Config
	if c.Seed != m.Seed ||
		!sameGeometry(c.Geometry, m.Geometry) ||
		c.SeaLevelNormalized != m.SeaLevelNormalized {
		return errors.New("worldmodelio: manifest and resolved config disagree")
	}
	if env.Payload.Terrain.Config != c.Terrain {
		return errors.New("worldmodelio: terrain config differs from resolved config")
	}
	if env.Payload.Waterbed.Config != c.Waterbed {
		return errors.New("worldmodelio: waterbed config differs from resolved config")
	}
	if m.WorldProfileVersion != "" {
		p := c.WorldProfile
		if p.WorldProfileVersion != m.WorldProfileVersion ||
			p.GenerationRulesVersion != m.GenerationRulesVersion ||
			p.WorldModelVersion != m.WorldModelVersion || p.Preset != m.Preset {
			return errors.New("worldmodelio: manifest and world profile identity disagree")
		}
		parameterJSON, err := json.Marshal(p.Parameters)
		if err != nil {
			return fmt.Errorf("worldmodelio: encode persisted parameter snapshot checksum: %w", err)
		}
		parameterSum := sha256.Sum256(parameterJSON)
		if hex.EncodeToString(parameterSum[:]) != m.ParameterSnapshotSHA256 {
			return errors.New("worldmodelio: parameter snapshot checksum mismatch")
		}
	}
	return nil
}

func snapshotFromWorldModel(m *generator.WorldModel) (payload, error) {
	if m == nil || m.Surface == nil || m.CoastTerrain == nil || m.Phase1 == nil ||
		m.Waterbed == nil || m.Runtime == nil || m.Climate == nil || m.Hydrology == nil ||
		m.Ecology == nil || m.Profile == nil || m.RuntimeProfile == nil {
		return payload{}, errors.New("worldmodelio: world model is not Runtime-complete")
	}
	if err := m.Geometry.Validate(); err != nil {
		return payload{}, err
	}
	if !sameGeometry(m.Geometry, m.Config.Geometry) ||
		m.Config.Seed != m.CoastTerrain.Config.Seed ||
		m.Config.SeaLevelNormalized != m.Surface.SeaLevelNormalized {
		return payload{}, errors.New("worldmodelio: world model shared contracts disagree")
	}
	if !sameGeometry(m.CoastTerrain.Geometry, m.Geometry) ||
		!sameGeometry(m.Waterbed.Geometry, m.Geometry) ||
		!sameGeometry(m.Runtime.Geometry, m.Geometry) ||
		!sameGeometry(m.RuntimeProfile.Geometry, m.Geometry) {
		return payload{}, errors.New("worldmodelio: Runtime model geometry disagrees")
	}

	return payload{
		Config: m.Config,
		Surface: surfaceSnapshot{
			LandMask:         append([]bool(nil), m.Surface.LandMask...),
			EnclosedBelowSea: append([]bool(nil), m.Surface.EnclosedBelowSea...),
		},
		Terrain: snapshotTerrain(m.CoastTerrain),
		Waterbed: waterbedSnapshot{
			Config: m.Waterbed.Config,
			Rivers: cloneRivers(m.Waterbed.Rivers),
			Lakes:  append([]waterbed.LakeNode(nil), m.Waterbed.Lakes...),
		},
		ClimatePreview: &climatePreviewSnapshot{
			AnnualMeanTempC: cloneFloat64(m.Climate.AdjustedAnnualTempC),
			AnnualPrecipMM:  cloneFloat64(m.Climate.AnnualPrecipMM),
			MonsoonIndex:    cloneFloat64(m.Climate.MonsoonIndex),
			WindX:           cloneFloat64(m.Climate.WindX),
			WindZ:           cloneFloat64(m.Climate.WindZ),
		},
		HydrologyQuery: &hydrologyQuerySnapshot{
			BasinID:          append([]int(nil), m.Hydrology.BasinID...),
			FlowTo:           append([]int(nil), m.Hydrology.FlowTo...),
			FlowAccumulation: cloneFloat64(m.Hydrology.FlowAccumulation),
			RiverOrder:       append([]uint8(nil), m.Hydrology.RiverOrder...),
		},
		TectonicQuery: &tectonicQuerySnapshot{
			Feature:          append([]phase1.TectonicFeature(nil), m.Phase1.Feature...),
			Intensity:        cloneFloat64(m.Phase1.TectonicIntensity),
			InfluenceFeature: append([]phase1.TectonicFeature(nil), m.Phase1.InfluenceFeature...),
			Influence:        cloneFloat64(m.Phase1.TectonicInfluence),
		},
		Ecology: ecologySnapshot{
			LandMask:         append([]bool(nil), m.Ecology.LandMask...),
			EcoRegion:        append([]ecology.Region(nil), m.Ecology.EcoRegion...),
			RegionConfidence: append([]float64(nil), m.Ecology.RegionConfidence...),
		},
		Profile: profileSnapshot{
			LandMask:          append([]bool(nil), m.Profile.LandMask...),
			BiomeFamily:       append([]mcprofile.BiomeFamily(nil), m.Profile.BiomeFamily...),
			TerrainVariant:    append([]mcprofile.TerrainVariant(nil), m.Profile.TerrainVariant...),
			SurfaceProfile:    append([]mcprofile.SurfaceProfile(nil), m.Profile.SurfaceProfile...),
			VegetationProfile: append([]mcprofile.VegetationProfile(nil), m.Profile.VegetationProfile...),
			ProfileConfidence: append([]float64(nil), m.Profile.ProfileConfidence...),
		},
	}, nil
}

func snapshotTerrain(m *terrain.Model) terrainSnapshot {
	out := terrainSnapshot{
		Config:              m.Config,
		BaseElevation:       cloneFloat64(m.BaseElevation.Values),
		RollingPlainWeight:  cloneFloat64(m.RollingPlainWeight.Values),
		HillWeight:          cloneFloat64(m.HillWeight.Values),
		FoothillWeight:      cloneFloat64(m.FoothillWeight.Values),
		MountainWeight:      cloneFloat64(m.MountainWeight.Values),
		PlateauWeight:       cloneFloat64(m.PlateauWeight.Values),
		ValleyWeight:        cloneFloat64(m.ValleyWeight.Values),
		FloodplainWeight:    cloneFloat64(m.FloodplainWeight.Values),
		AlluvialPlainWeight: cloneFloat64(m.AlluvialPlainWeight.Values),
		AlluvialFanWeight:   cloneFloat64(m.AlluvialFanWeight.Values),
		CanyonWeight:        cloneFloat64(m.CanyonWeight.Values),
		ValleyFloorWidth:    cloneFloat64(m.ValleyFloorWidth.Values),
		DepositionPotential: cloneFloat64(m.DepositionPotential.Values),
	}
	if m.HeightCorrection != nil {
		out.HeightCorrection = cloneFloat64(m.HeightCorrection.Values)
	}
	return out
}

func restoreTerrain(g world.GridGeometry, s terrainSnapshot) (*terrain.Model, error) {
	if s.Config.CliffFrequency == 0 {
		// Legacy v1 payloads predate the explicit cliff-frequency parameter.
		s.Config.CliffFrequency = 1.0
	}
	makeGrid := func(name string, values []float64) (sampling.ScalarGrid, error) {
		grid, err := sampling.NewScalarGridWithGeometry(g, values)
		if err != nil {
			return sampling.ScalarGrid{}, fmt.Errorf("worldmodelio: restore terrain %s: %w", name, err)
		}
		return grid, nil
	}

	base, err := makeGrid("base elevation", s.BaseElevation)
	if err != nil {
		return nil, err
	}
	plain, err := makeGrid("rolling plain", s.RollingPlainWeight)
	if err != nil {
		return nil, err
	}
	hill, err := makeGrid("hill", s.HillWeight)
	if err != nil {
		return nil, err
	}
	foothill, err := makeGrid("foothill", s.FoothillWeight)
	if err != nil {
		return nil, err
	}
	mountain, err := makeGrid("mountain", s.MountainWeight)
	if err != nil {
		return nil, err
	}
	plateau, err := makeGrid("plateau", s.PlateauWeight)
	if err != nil {
		return nil, err
	}
	valley, err := makeGrid("valley", s.ValleyWeight)
	if err != nil {
		return nil, err
	}
	floodplain, err := makeGrid("floodplain", s.FloodplainWeight)
	if err != nil {
		return nil, err
	}
	alluvialPlain, err := makeGrid("alluvial plain", s.AlluvialPlainWeight)
	if err != nil {
		return nil, err
	}
	alluvialFan, err := makeGrid("alluvial fan", s.AlluvialFanWeight)
	if err != nil {
		return nil, err
	}
	canyon, err := makeGrid("canyon", s.CanyonWeight)
	if err != nil {
		return nil, err
	}
	valleyFloor, err := makeGrid("valley floor", s.ValleyFloorWidth)
	if err != nil {
		return nil, err
	}
	deposition, err := makeGrid("deposition", s.DepositionPotential)
	if err != nil {
		return nil, err
	}

	m := &terrain.Model{
		Config:              s.Config,
		Geometry:            g,
		BaseElevation:       base,
		RollingPlainWeight:  plain,
		HillWeight:          hill,
		FoothillWeight:      foothill,
		MountainWeight:      mountain,
		PlateauWeight:       plateau,
		ValleyWeight:        valley,
		FloodplainWeight:    floodplain,
		AlluvialPlainWeight: alluvialPlain,
		AlluvialFanWeight:   alluvialFan,
		CanyonWeight:        canyon,
		ValleyFloorWidth:    valleyFloor,
		DepositionPotential: deposition,
	}
	if len(s.HeightCorrection) > 0 {
		correction, err := makeGrid("height correction", s.HeightCorrection)
		if err != nil {
			return nil, err
		}
		m.HeightCorrection = &correction
	}
	return m, nil
}

func restoreEcology(g world.GridGeometry, surface *world.Surface, cfg ecology.Config, s ecologySnapshot) (*ecology.Model, error) {
	n, _ := world.NodeCount(g.Width, g.Height)
	if len(s.LandMask) != n || len(s.EcoRegion) != n || len(s.RegionConfidence) != n {
		return nil, errors.New("worldmodelio: persisted ecology size does not match geometry")
	}
	for i := 0; i < n; i++ {
		if s.LandMask[i] != surface.LandMask[i] {
			return nil, errors.New("worldmodelio: persisted ecology land mask disagrees with surface")
		}
		if s.EcoRegion[i] == ecology.RegionUnknown ||
			!finite01(s.RegionConfidence[i]) {
			return nil, errors.New("worldmodelio: persisted ecology contains invalid Runtime field")
		}
	}
	return &ecology.Model{
		Config:           cfg,
		Width:            g.Width,
		Height:           g.Height,
		LandMask:         append([]bool(nil), s.LandMask...),
		EcoRegion:        append([]ecology.Region(nil), s.EcoRegion...),
		RegionConfidence: append([]float64(nil), s.RegionConfidence...),
	}, nil
}

func restoreProfile(g world.GridGeometry, surface *world.Surface, s profileSnapshot) (*mcprofile.Model, error) {
	n, _ := world.NodeCount(g.Width, g.Height)
	if len(s.LandMask) != n || len(s.BiomeFamily) != n ||
		len(s.TerrainVariant) != n || len(s.SurfaceProfile) != n ||
		len(s.VegetationProfile) != n || len(s.ProfileConfidence) != n {
		return nil, errors.New("worldmodelio: persisted profile size does not match geometry")
	}
	for i := 0; i < n; i++ {
		if s.LandMask[i] != surface.LandMask[i] {
			return nil, errors.New("worldmodelio: persisted profile land mask disagrees with surface")
		}
		if s.BiomeFamily[i] == mcprofile.BiomeUnknown ||
			s.TerrainVariant[i] == mcprofile.TerrainUnknown ||
			s.SurfaceProfile[i] == mcprofile.SurfaceUnknown ||
			s.VegetationProfile[i] == mcprofile.VegetationUnknown ||
			!finite01(s.ProfileConfidence[i]) {
			return nil, errors.New("worldmodelio: persisted profile contains invalid Runtime field")
		}
	}
	return &mcprofile.Model{
		Width:             g.Width,
		Height:            g.Height,
		LandMask:          append([]bool(nil), s.LandMask...),
		BiomeFamily:       append([]mcprofile.BiomeFamily(nil), s.BiomeFamily...),
		TerrainVariant:    append([]mcprofile.TerrainVariant(nil), s.TerrainVariant...),
		SurfaceProfile:    append([]mcprofile.SurfaceProfile(nil), s.SurfaceProfile...),
		VegetationProfile: append([]mcprofile.VegetationProfile(nil), s.VegetationProfile...),
		ProfileConfidence: append([]float64(nil), s.ProfileConfidence...),
	}, nil
}

func cloneRivers(in []waterbed.RiverSegment) []waterbed.RiverSegment {
	out := make([]waterbed.RiverSegment, len(in))
	for i, seg := range in {
		seg.Profile = cloneFloat64(seg.Profile)
		out[i] = seg
	}
	return out
}

func restoreHydrologyQuery(n int, s *hydrologyQuerySnapshot) (*HydrologyQuery, error) {
	if s == nil {
		return nil, nil
	}
	if len(s.BasinID) != n || len(s.FlowTo) != n ||
		len(s.FlowAccumulation) != n || len(s.RiverOrder) != n {
		return nil, errors.New("worldmodelio: persisted hydrology query size does not match geometry")
	}
	for i := 0; i < n; i++ {
		if s.BasinID[i] < 0 ||
			s.FlowTo[i] < -1 || s.FlowTo[i] >= n ||
			!finite(s.FlowAccumulation[i]) || s.FlowAccumulation[i] < 0 {
			return nil, errors.New("worldmodelio: persisted hydrology query contains invalid value")
		}
	}
	return &HydrologyQuery{
		BasinID:          append([]int(nil), s.BasinID...),
		FlowTo:           append([]int(nil), s.FlowTo...),
		FlowAccumulation: cloneFloat64(s.FlowAccumulation),
		RiverOrder:       append([]uint8(nil), s.RiverOrder...),
	}, nil
}

func restoreTectonicQuery(n int, s *tectonicQuerySnapshot) (*TectonicQuery, error) {
	if s == nil {
		return nil, nil
	}
	if len(s.Feature) != n || len(s.Intensity) != n ||
		len(s.InfluenceFeature) != n || len(s.Influence) != n {
		return nil, errors.New("worldmodelio: persisted tectonic query size does not match geometry")
	}
	for i := 0; i < n; i++ {
		if s.Feature[i] > phase1.TransformFault ||
			s.InfluenceFeature[i] > phase1.TransformFault ||
			!finite(s.Intensity[i]) || s.Intensity[i] < 0 || s.Intensity[i] > 1 ||
			!finite(s.Influence[i]) {
			return nil, errors.New("worldmodelio: persisted tectonic query contains invalid value")
		}
	}
	return &TectonicQuery{
		Feature:          append([]phase1.TectonicFeature(nil), s.Feature...),
		Intensity:        cloneFloat64(s.Intensity),
		InfluenceFeature: append([]phase1.TectonicFeature(nil), s.InfluenceFeature...),
		Influence:        cloneFloat64(s.Influence),
	}, nil
}

func restoreClimatePreview(s *climatePreviewSnapshot) *ClimatePreview {
	if s == nil {
		return nil
	}
	if len(s.AnnualMeanTempC) == 0 &&
		len(s.AnnualPrecipMM) == 0 &&
		len(s.MonsoonIndex) == 0 &&
		len(s.WindX) == 0 &&
		len(s.WindZ) == 0 {
		return nil
	}
	return &ClimatePreview{
		AnnualMeanTempC: cloneFloat64(s.AnnualMeanTempC),
		AnnualPrecipMM:  cloneFloat64(s.AnnualPrecipMM),
		MonsoonIndex:    cloneFloat64(s.MonsoonIndex),
		WindX:           cloneFloat64(s.WindX),
		WindZ:           cloneFloat64(s.WindZ),
	}
}

func cloneFloat64(in []float64) []float64 {
	return append([]float64(nil), in...)
}

func finite01(v float64) bool {
	return finite(v) && v >= 0 && v <= 1
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

func sameGeometry(a, b world.GridGeometry) bool {
	return a.Width == b.Width &&
		a.Height == b.Height &&
		math.Abs(a.OriginX-b.OriginX) <= 1e-12 &&
		math.Abs(a.OriginZ-b.OriginZ) <= 1e-12 &&
		math.Abs(a.SpanXBlocks-b.SpanXBlocks) <= 1e-9 &&
		math.Abs(a.SpanZBlocks-b.SpanZBlocks) <= 1e-9
}
