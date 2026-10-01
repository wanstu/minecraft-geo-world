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
	if path == "" {
		return errors.New("worldmodelio: empty save path")
	}
	p, err := snapshotFromWorldModel(model)
	if err != nil {
		return err
	}
	payloadJSON, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("worldmodelio: encode payload checksum: %w", err)
	}
	sum := sha256.Sum256(payloadJSON)

	manifest := Manifest{
		Magic:              FormatMagic,
		SchemaVersion:      CurrentSchemaVersion,
		GeneratorVersion:   CurrentGeneratorVersion,
		FileEncoding:       FileEncoding,
		ChecksumAlgorithm:  ChecksumAlgorithm,
		PayloadSHA256:      hex.EncodeToString(sum[:]),
		PayloadBytes:       int64(len(payloadJSON)),
		Seed:               model.Config.Seed,
		Geometry:           model.Geometry,
		SeaLevelNormalized: model.Config.SeaLevelNormalized,
	}
	env := fileEnvelope{Manifest: manifest, Payload: p}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("worldmodelio: create save directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("worldmodelio: create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	defer cleanup()

	gz := gzip.NewWriter(tmp)
	enc := json.NewEncoder(gz)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(env); err != nil {
		_ = gz.Close()
		return fmt.Errorf("worldmodelio: encode model: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("worldmodelio: finalize compression: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("worldmodelio: sync temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("worldmodelio: close temporary file: %w", err)
	}

	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrTargetExists
		}
		return fmt.Errorf("worldmodelio: publish completed model: %w", err)
	}
	return nil
}

func Load(path string) (*RuntimeModel, error) {
	env, err := readEnvelope(path)
	if err != nil {
		return nil, err
	}
	if err := validateManifest(env.Manifest); err != nil {
		return nil, err
	}

	payloadJSON, err := json.Marshal(env.Payload)
	if err != nil {
		return nil, fmt.Errorf("worldmodelio: re-encode payload: %w", err)
	}
	sum := sha256.Sum256(payloadJSON)
	if env.Manifest.PayloadBytes != int64(len(payloadJSON)) ||
		env.Manifest.PayloadSHA256 != hex.EncodeToString(sum[:]) {
		return nil, ErrChecksumMismatch
	}
	if err := validateEnvelopeContracts(env); err != nil {
		return nil, err
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
	runtimeProfile, err := runtimeprofile.Build(g, eco, profile, runtimeModel)
	if err != nil {
		return nil, fmt.Errorf("worldmodelio: restore runtime profile: %w", err)
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
	return nil
}

func snapshotFromWorldModel(m *generator.WorldModel) (payload, error) {
	if m == nil || m.Surface == nil || m.CoastTerrain == nil ||
		m.Waterbed == nil || m.Runtime == nil || m.Ecology == nil ||
		m.Profile == nil || m.RuntimeProfile == nil {
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
