package terrain

import (
	"errors"
	"math"

	"geoworld/internal/sampling"
	"geoworld/internal/world"
)

func Generate(input Input, cfg Config) (*Model, error) {
	if input.Width < 2 || input.Height < 2 {
		return nil, errors.New("terrain: grid must be at least 2x2")
	}
	n := input.Width * input.Height
	for _, field := range []struct {
		name string
		data []float64
	}{
		{"base elevation", input.BaseElevation},
		{"rolling plain weight", input.RollingPlainWeight},
		{"hill weight", input.HillWeight},
		{"foothill weight", input.FoothillWeight},
		{"mountain weight", input.MountainWeight},
		{"plateau weight", input.PlateauWeight},
		{"valley weight", input.ValleyWeight},
		{"floodplain weight", input.FloodplainWeight},
		{"alluvial plain weight", input.AlluvialPlainWeight},
		{"alluvial fan weight", input.AlluvialFanWeight},
		{"canyon weight", input.CanyonWeight},
		{"valley floor width", input.ValleyFloorWidth},
		{"deposition potential", input.DepositionPotential},
	} {
		if len(field.data) != n {
			return nil, errors.New("terrain: " + field.name + " size does not match dimensions")
		}
	}

	geometry, err := resolveInputGeometry(input, cfg)
	if err != nil {
		return nil, err
	}
	cfg = normalizeConfig(cfg, geometry)

	base, err := scalarGrid(geometry, input.BaseElevation)
	if err != nil {
		return nil, err
	}
	plain, err := scalarGrid(geometry, input.RollingPlainWeight)
	if err != nil {
		return nil, err
	}
	hill, err := scalarGrid(geometry, input.HillWeight)
	if err != nil {
		return nil, err
	}
	foothill, err := scalarGrid(geometry, input.FoothillWeight)
	if err != nil {
		return nil, err
	}
	mountain, err := scalarGrid(geometry, input.MountainWeight)
	if err != nil {
		return nil, err
	}
	plateau, err := scalarGrid(geometry, input.PlateauWeight)
	if err != nil {
		return nil, err
	}
	valley, err := scalarGrid(geometry, input.ValleyWeight)
	if err != nil {
		return nil, err
	}
	floodplain, err := scalarGrid(geometry, input.FloodplainWeight)
	if err != nil {
		return nil, err
	}
	alluvialPlain, err := scalarGrid(geometry, input.AlluvialPlainWeight)
	if err != nil {
		return nil, err
	}
	alluvialFan, err := scalarGrid(geometry, input.AlluvialFanWeight)
	if err != nil {
		return nil, err
	}
	canyon, err := scalarGrid(geometry, input.CanyonWeight)
	if err != nil {
		return nil, err
	}
	valleyFloor, err := scalarGrid(geometry, input.ValleyFloorWidth)
	if err != nil {
		return nil, err
	}
	deposition, err := scalarGrid(geometry, input.DepositionPotential)
	if err != nil {
		return nil, err
	}

	return &Model{
		Config:              cfg,
		Geometry:            geometry,
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
	}, nil
}

func resolveInputGeometry(input Input, cfg Config) (world.GridGeometry, error) {
	geometry := input.Geometry
	if geometry.Width == 0 && geometry.Height == 0 &&
		geometry.SpanXBlocks == 0 && geometry.SpanZBlocks == 0 &&
		geometry.OriginX == 0 && geometry.OriginZ == 0 {
		cellSize := cfg.CellSizeBlocks
		if cellSize <= 0 {
			cellSize = world.DefaultMacroSpacingBlocks
		}
		geometry = world.GridGeometry{
			Width: input.Width, Height: input.Height,
			OriginX: cfg.OriginX, OriginZ: cfg.OriginZ,
			SpanXBlocks: float64(input.Width-1) * cellSize,
			SpanZBlocks: float64(input.Height-1) * cellSize,
		}
	}
	if geometry.Width != input.Width || geometry.Height != input.Height {
		return world.GridGeometry{}, errors.New("terrain: geometry dimensions do not match input")
	}
	cellSize, err := geometry.SquareCellSize()
	if err != nil {
		return world.GridGeometry{}, err
	}
	if cfg.CellSizeBlocks > 0 && math.Abs(cfg.CellSizeBlocks-cellSize) > 1e-9 {
		return world.GridGeometry{}, errors.New("terrain: config cell size conflicts with input geometry")
	}
	if cfg.OriginX != 0 && math.Abs(cfg.OriginX-geometry.OriginX) > 1e-9 {
		return world.GridGeometry{}, errors.New("terrain: config origin X conflicts with input geometry")
	}
	if cfg.OriginZ != 0 && math.Abs(cfg.OriginZ-geometry.OriginZ) > 1e-9 {
		return world.GridGeometry{}, errors.New("terrain: config origin Z conflicts with input geometry")
	}
	return geometry, nil
}

func normalizeConfig(cfg Config, geometry world.GridGeometry) Config {
	cfg.CellSizeBlocks = geometry.SpacingX()
	cfg.OriginX = geometry.OriginX
	cfg.OriginZ = geometry.OriginZ

	if cfg.PlainAmplitude <= 0 {
		cfg.PlainAmplitude = 0.0035
	}
	if cfg.HillAmplitude <= 0 {
		cfg.HillAmplitude = 0.018
	}
	if cfg.FoothillAmplitude <= 0 {
		cfg.FoothillAmplitude = 0.028
	}
	if cfg.MountainAmplitude <= 0 {
		cfg.MountainAmplitude = 0.042
	}
	if cfg.PlateauAmplitude <= 0 {
		cfg.PlateauAmplitude = 0.007
	}
	if cfg.ValleyDepth <= 0 {
		cfg.ValleyDepth = 0.007
	}
	if cfg.DepositionLift <= 0 {
		cfg.DepositionLift = 0.0035
	}
	if cfg.AlluvialFanLift <= 0 {
		cfg.AlluvialFanLift = 0.0045
	}
	if cfg.CanyonExtraDepth <= 0 {
		cfg.CanyonExtraDepth = 0.006
	}
	return cfg
}

func scalarGrid(geometry world.GridGeometry, values []float64) (sampling.ScalarGrid, error) {
	return sampling.NewScalarGridWithGeometry(geometry, values)
}

func (m *Model) SampleNormalized(worldX, worldZ float64) float64 {
	return m.Sample(worldX, worldZ).FinalHeight
}

func (m *Model) Sample(worldX, worldZ float64) Sample {
	base := m.BaseElevation.SampleCubicBSpline(worldX, worldZ)
	plain := clamp01(m.RollingPlainWeight.SampleCubicBSpline(worldX, worldZ))
	hill := clamp01(m.HillWeight.SampleCubicBSpline(worldX, worldZ))
	foothill := clamp01(m.FoothillWeight.SampleCubicBSpline(worldX, worldZ))
	mountain := clamp01(m.MountainWeight.SampleCubicBSpline(worldX, worldZ))
	plateau := clamp01(m.PlateauWeight.SampleCubicBSpline(worldX, worldZ))
	valley := clamp01(m.ValleyWeight.SampleCubicBSpline(worldX, worldZ))
	floodplain := clamp01(m.FloodplainWeight.SampleCubicBSpline(worldX, worldZ))
	alluvialPlain := clamp01(m.AlluvialPlainWeight.SampleCubicBSpline(worldX, worldZ))
	alluvialFan := clamp01(m.AlluvialFanWeight.SampleCubicBSpline(worldX, worldZ))
	canyon := clamp01(m.CanyonWeight.SampleCubicBSpline(worldX, worldZ))
	valleyFloor := clamp01(m.ValleyFloorWidth.SampleCubicBSpline(worldX, worldZ))
	deposition := clamp01(m.DepositionPotential.SampleCubicBSpline(worldX, worldZ))

	// These noises are global world-coordinate functions. Chunk coordinates do
	// not enter the calculation, so neighbouring chunks cannot get different
	// answers for the same position.
	plainDetail := fbm(m.Config.Seed+101, worldX, worldZ, 2300, 2) * m.Config.PlainAmplitude
	hillDetail := fbm(m.Config.Seed+211, worldX, worldZ, 1250, 3) * m.Config.HillAmplitude
	foothillDetail := fbm(m.Config.Seed+307, worldX, worldZ, 1500, 3) * m.Config.FoothillAmplitude
	mountainDetail := fbm(m.Config.Seed+401, worldX, worldZ, 1750, 4) * m.Config.MountainAmplitude
	plateauDetail := fbm(m.Config.Seed+503, worldX, worldZ, 2600, 2) * m.Config.PlateauAmplitude

	regionalDetail :=
		plain*plainDetail +
			hill*hillDetail +
			foothill*foothillDetail +
			mountain*mountainDetail +
			plateau*plateauDetail

	// Broad depositional lowlands should be smoother than surrounding terrain.
	// This does not decide where plains exist; it only refines already-derived
	// floodplain / alluvial / valley-floor fields.
	flatness := clamp(
		0.48*floodplain+
			0.36*alluvialPlain+
			0.34*valleyFloor,
		0,
		0.88,
	)
	regionalDetail *= 1 - 0.62*flatness
	regionalDetail *= 1 - 0.28*alluvialFan

	// Valleys are broad overlays, not one-block river cuts. Stronger valley
	// influence lowers the regional surface. Canyon tendency adds a narrower,
	// stronger incision on top of the existing valley geometry.
	valleyMod := 0.82 + 0.18*(0.5+0.5*valueNoise(m.Config.Seed+607, worldX, worldZ, 1800))
	valleyCarve := valley * m.Config.ValleyDepth * valleyMod
	canyonCarve := canyon * m.Config.CanyonExtraDepth

	// Deposition fills a small amount of accommodation space. Alluvial fans add
	// a slightly stronger convex accumulation at mountain exits.
	depositionLift := deposition*m.Config.DepositionLift +
		alluvialFan*m.Config.AlluvialFanLift

	final := boundedAdd(base, regionalDetail+depositionLift-valleyCarve-canyonCarve)
	heightCorrection := 0.0
	if m.HeightCorrection != nil {
		heightCorrection = m.HeightCorrection.SampleSmoothstepBilinear(worldX, worldZ)
		final = clamp(final+heightCorrection, 0, 1)
	}

	return Sample{
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
		RegionalDetail:      regionalDetail,
		DepositionLift:      depositionLift,
		ValleyCarve:         valleyCarve,
		CanyonCarve:         canyonCarve,
		HeightCorrection:    heightCorrection,
		FinalHeight:         final,
	}
}

// boundedAdd applies a height delta without hard clipping at 0 or 1.
func boundedAdd(base, delta float64) float64 {
	base = clamp(base, 0, 1)
	if delta == 0 {
		return base
	}

	if delta > 0 {
		headroom := 1 - base
		if headroom <= 1e-12 {
			return base
		}
		return base + headroom*(1-math.Exp(-delta/headroom))
	}

	footroom := base
	if footroom <= 1e-12 {
		return base
	}
	return base - footroom*(1-math.Exp(delta/footroom))
}

func fbm(seed int64, x, z, wavelength float64, octaves int) float64 {
	if octaves <= 0 {
		return 0
	}
	amplitude := 1.0
	sumAmplitude := 0.0
	sum := 0.0
	freqWavelength := wavelength

	for octave := 0; octave < octaves; octave++ {
		sum += valueNoise(seed+int64(octave)*7919, x, z, freqWavelength) * amplitude
		sumAmplitude += amplitude
		amplitude *= 0.5
		freqWavelength *= 0.5
	}
	if sumAmplitude <= 0 {
		return 0
	}
	return sum / sumAmplitude
}

func valueNoise(seed int64, x, z, wavelength float64) float64 {
	if wavelength <= 0 {
		return 0
	}

	gx := x / wavelength
	gz := z / wavelength
	x0 := int64(math.Floor(gx))
	z0 := int64(math.Floor(gz))
	tx := gx - float64(x0)
	tz := gz - float64(z0)

	fx := quinticFade(tx)
	fz := quinticFade(tz)

	v00 := hashValue(seed, x0, z0)
	v10 := hashValue(seed, x0+1, z0)
	v01 := hashValue(seed, x0, z0+1)
	v11 := hashValue(seed, x0+1, z0+1)

	top := lerp(v00, v10, fx)
	bottom := lerp(v01, v11, fx)
	return lerp(top, bottom, fz)
}

func hashValue(seed, x, z int64) float64 {
	h := uint64(seed) ^
		(uint64(x) * 0x9E3779B185EBCA87) ^
		(uint64(z) * 0xC2B2AE3D27D4EB4F)
	h ^= h >> 30
	h *= 0xBF58476D1CE4E5B9
	h ^= h >> 27
	h *= 0x94D049BB133111EB
	h ^= h >> 31

	// Use the high 53 bits to produce a deterministic value in [-1,1].
	unit := float64(h>>11) / float64(uint64(1)<<53)
	return unit*2 - 1
}

func quinticFade(t float64) float64 {
	t = clamp(t, 0, 1)
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func clamp01(v float64) float64 {
	return clamp(v, 0, 1)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
