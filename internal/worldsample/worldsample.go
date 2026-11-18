package worldsample

import (
	"errors"
	"fmt"
	"math"

	"geoworld/internal/reconcile"
	"geoworld/internal/sampling"
	"geoworld/internal/waterbed"
	"geoworld/internal/world"
)

func Build(surface *world.Surface, ground *waterbed.Model, cfg Config) (*Model, error) {
	if surface == nil {
		return nil, errors.New("worldsample: macro surface is required")
	}
	if ground == nil {
		return nil, errors.New("worldsample: waterbed ground is required")
	}
	if _, err := ground.Geometry.SquareCellSize(); err != nil {
		return nil, fmt.Errorf("worldsample: geometry: %w", err)
	}
	n, err := world.NodeCount(ground.Geometry.Width, ground.Geometry.Height)
	if err != nil {
		return nil, err
	}
	if surface.Width != ground.Geometry.Width ||
		surface.Height != ground.Geometry.Height ||
		len(surface.LandMask) != n ||
		len(surface.EnclosedBelowSea) != n {
		return nil, errors.New("worldsample: surface dimensions do not match ground geometry")
	}
	if !finite(surface.SeaLevelNormalized) ||
		surface.SeaLevelNormalized <= 0 ||
		surface.SeaLevelNormalized >= 1 {
		return nil, errors.New("worldsample: invalid sea level")
	}

	cfg, err = normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}
	if math.Abs(
		ground.Config.RiverCenterDepthNormalized-cfg.RiverCenterDepthNormalized,
	) > 1e-12 {
		return nil, errors.New(
			"worldsample: river center depth conflicts with waterbed profile",
		)
	}

	ocean := make([]float64, n)
	for i, land := range surface.LandMask {
		if !land {
			ocean[i] = 1
		}
	}
	oceanGrid, err := sampling.NewScalarGridWithGeometry(ground.Geometry, ocean)
	if err != nil {
		return nil, err
	}

	return &Model{
		Config:             cfg,
		Geometry:           ground.Geometry,
		SeaLevelNormalized: surface.SeaLevelNormalized,
		Ground:             ground,
		OceanEligibility:   oceanGrid,
	}, nil
}

func (m *Model) Sample(worldX, worldZ float64) Sample {
	ground := m.Ground.Sample(worldX, worldZ)
	oceanEligibility := m.OceanEligibility.SampleSmoothstepBilinear(worldX, worldZ)

	out := Sample{
		GroundHeight:       ground.FinalHeight,
		WaterKind:          reconcile.WaterNone,
		OceanEligibility:   oceanEligibility,
		RiverWeight:        ground.RiverWeight,
		CoastalSeaReceiver: ground.RiverCoastalReceiverDefined && ground.RiverOceanMouthNearby,
	}

	// R8.1c: a formal OceanMouth receiver owns the connected below-sea
	// estuary before local lake ownership is considered. This is deliberately
	// narrower than the ordinary ocean transition: it requires persisted
	// OceanMouth receiver metadata, coastal ocean influence and low terrain.
	// Without this precedence, a lake cell overlapping the mouth can retain a
	// higher lake datum beside sea-level river cells and create a voxel water wall.
	if qualifiesForCoastalBackwater(
		m.Config,
		ground,
		m.SeaLevelNormalized,
		oceanEligibility,
	) {
		out.WaterKind = coastalBackwaterWaterKind(
			m.Config,
			ground,
			oceanEligibility,
		)
		out.WaterSurfaceDefined = true
		out.WaterSurfaceElevation = m.SeaLevelNormalized
		if out.WaterKind == reconcile.WaterRiver {
			out.GroundHeight = m.navigableRiverGroundHeight(
				out.GroundHeight,
				out.WaterSurfaceElevation,
				ground.RiverWeight,
			)
		}
		if oceanEligibility >= m.Config.OceanEligibilityThreshold {
			out.GroundHeight = m.coastalGroundHeight(
				worldX, worldZ, oceanEligibility, ground, out.GroundHeight,
			)
		}
		out.WaterDepth = out.WaterSurfaceElevation - out.GroundHeight
		return out
	}

	// Explicit lakes remain hydrology-owned basins everywhere except the
	// versioned OceanMouth corridor handled above.
	if ground.WaterKind == reconcile.WaterLake &&
		ground.WaterSurfaceDefined &&
		ground.FinalHeight < ground.WaterSurfaceElevation {
		out.WaterKind = reconcile.WaterLake
		out.WaterSurfaceDefined = true
		out.WaterSurfaceElevation = ground.WaterSurfaceElevation
		out.WaterDepth = ground.WaterSurfaceElevation - ground.FinalHeight
		return out
	}

	riverSurface, riverDepth, riverGroundHeight, riverValid :=
		m.riverWaterSurface(ground)

	// Ocean requires both continuous below-sea terrain and continuous macro
	// ocean eligibility. Near a river mouth, however, do not let the ocean
	// classification cut in while the supported river surface is still more
	// than one Minecraft block above sea level. The river profile already
	// limits longitudinal descent to <=1 block per horizontal block, so delaying
	// handoff until this threshold removes multi-block source-water lips without
	// extending the river arbitrarily far offshore.
	if oceanEligibility >= m.Config.OceanEligibilityThreshold &&
		ground.FinalHeight < m.SeaLevelNormalized {
		oneBlock := normalizedBlockStepAbove(
			m.SeaLevelNormalized,
		)
		if !riverValid ||
			riverSurface <= m.SeaLevelNormalized+oneBlock {
			out.WaterKind = reconcile.WaterOcean
			out.WaterSurfaceDefined = true
			out.WaterSurfaceElevation = m.SeaLevelNormalized
			out.GroundHeight = m.coastalGroundHeight(
				worldX, worldZ, oceanEligibility, ground, ground.FinalHeight,
			)
			out.WaterDepth = m.SeaLevelNormalized - out.GroundHeight
			return out
		}
	}

	// River water starts with a constant center depth. If a downstream
	// lake/ocean stands above the natural river surface, Waterbed propagates that
	// receiving datum upstream only as far as needed and Runtime applies it as a
	// flat backwater floor. The channel cross-section still owns where that water
	// can exist, so the floor does not flood the surrounding broad terrain.
	if riverValid {
		out.GroundHeight = riverGroundHeight
		out.WaterKind = reconcile.WaterRiver
		out.WaterSurfaceDefined = true
		out.WaterSurfaceElevation = riverSurface
		out.WaterDepth = riverDepth
	}
	return out
}

const coastalBackwaterMinOceanEligibility = 0.02

// qualifiesForCoastalBackwater never invents a hydrology endpoint. It extends
// the flat sea datum only across the low bank corridor of a formal OceanMouth.
// That corridor is topographically bounded on the inland side and joins the
// ordinary macro ocean on the seaward side. Filling the whole connected
// below-sea corridor avoids a sea-level river column standing beside lower dry
// terrain; requiring the formal mouth flag prevents lake/downstream receiver
// metadata from flooding unrelated lowlands.
func qualifiesForCoastalBackwater(
	cfg Config,
	ground waterbed.Sample,
	seaLevel,
	oceanEligibility float64,
) bool {
	return cfg.CoastalBackwaterVersion >= 1 &&
		ground.RiverCoastalReceiverDefined &&
		ground.RiverOceanMouthNearby &&
		math.Abs(ground.RiverCoastalReceiverSurface-seaLevel) < 1e-9 &&
		oceanEligibility > coastalBackwaterMinOceanEligibility &&
		ground.FinalHeight < seaLevel-1e-12
}

func coastalBackwaterWaterKind(
	cfg Config,
	ground waterbed.Sample,
	oceanEligibility float64,
) reconcile.WaterKind {
	// Slice E v2 separates the receiving water datum from biome/river
	// ownership. A formal river corridor remains River even after the macro
	// ocean field reaches full strength; only the surrounding mouth water turns
	// into Ocean/Warm Ocean.
	if cfg.CoastalBackwaterVersion >= 2 &&
		ground.RiverBedDefined &&
		ground.RiverWeight > 0 {
		return reconcile.WaterRiver
	}
	if oceanEligibility >= cfg.OceanEligibilityThreshold {
		return reconcile.WaterOcean
	}
	return reconcile.WaterRiver
}

func (m *Model) riverWaterSurface(
	ground waterbed.Sample,
) (float64, float64, float64, bool) {
	if ground.WaterKind != reconcile.WaterRiver ||
		ground.RiverWeight <= 0 ||
		!ground.RiverBedDefined {
		return 0, 0, ground.FinalHeight, false
	}
	widthSpan := math.Max(
		1e-9,
		m.Ground.Config.MaxRiverHalfWidthBlocks-
			m.Ground.Config.MinRiverHalfWidthBlocks,
	)
	widthT := math.Max(
		0,
		math.Min(
			1,
			(ground.RiverHalfWidth-m.Ground.Config.MinRiverHalfWidthBlocks)/
				widthSpan,
		),
	)
	centerDepth := m.Config.RiverCenterDepthNormalized *
		(1 + 2*math.Pow(widthT, 1.15))
	surface := ground.RiverBedElevation + centerDepth
	if ground.RiverReceiverSurfaceDefined &&
		ground.RiverReceiverSurface > surface &&
		ground.RiverReceiverWeight > 0 {
		surface = ground.RiverReceiverSurface
	}
	surface = math.Min(1, surface)
	effectiveGround := m.navigableRiverGroundHeight(
		ground.FinalHeight,
		surface,
		ground.RiverWeight,
	)
	localDepth := surface - effectiveGround
	minimumVisibleDepth := m.Config.MinRiverWaterDepthNormalized
	if m.Ground.Config.RiverGeometryVersion >= 8 {
		// Geometry v8 owns the whole curved hydraulic half-width continuously.
		// Fade the visibility threshold toward the lateral edge so a shallow
		// river margin cannot be reclassified as a one-block dry hole after the
		// Waterbed geometry already marked it as river. Keep a quarter-threshold
		// floor to reject numerical puddles and near-zero residual depth.
		minimumVisibleDepth *= math.Max(
			0.25,
			math.Max(0, math.Min(1, ground.RiverWeight)),
		)
	}
	if localDepth < minimumVisibleDepth {
		return 0, 0, ground.FinalHeight, false
	}
	return surface, localDepth, effectiveGround, true
}

func (m *Model) navigableRiverGroundHeight(
	groundHeight,
	waterSurface,
	riverWeight float64,
) float64 {
	if m.Config.RiverNavigationVersion < 1 ||
		m.Config.MinNavigableRiverDepthBlocks == 0 ||
		riverWeight < 0.5 {
		return groundHeight
	}

	waterTopY := int(math.Ceil(normalizedHeightBlockBoundary(waterSurface))) - 1
	groundTopY := int(math.Floor(normalizedHeightBlockBoundary(groundHeight)))
	targetGroundTopY := waterTopY - int(m.Config.MinNavigableRiverDepthBlocks)
	if groundTopY <= targetGroundTopY {
		return groundHeight
	}

	// Aim at the middle of the target ground block. Using a half-block
	// boundary avoids integer-boundary floating-point ambiguity while lowering
	// only as far as needed for the requested number of actual water voxels.
	targetBoundary := float64(targetGroundTopY) + 0.5
	targetHeight := normalizedHeightFromBlockBoundary(targetBoundary)
	return math.Min(groundHeight, targetHeight)
}

const (
	minimumOffshoreDepthBlocks = 5
	shoreDepthTransitionBlocks = 16.0
	shoreDepthDiagonal         = 11.313708498984761
)

var shoreDepthOffsets = [8][2]float64{
	{shoreDepthTransitionBlocks, 0}, {-shoreDepthTransitionBlocks, 0},
	{0, shoreDepthTransitionBlocks}, {0, -shoreDepthTransitionBlocks},
	{shoreDepthDiagonal, shoreDepthDiagonal}, {-shoreDepthDiagonal, shoreDepthDiagonal},
	{shoreDepthDiagonal, -shoreDepthDiagonal}, {-shoreDepthDiagonal, -shoreDepthDiagonal},
}

func (m *Model) coastalGroundHeight(
	x, z, ocean float64,
	ground waterbed.Sample,
	currentHeight float64,
) float64 {
	if m.Config.OceanBathymetryVersion == 0 {
		return currentHeight
	}
	sea := m.SeaLevelNormalized
	if ground.RiverOceanMouthNearby &&
		math.Abs(ground.RiverCoastalReceiverSurface-sea) < 1e-9 {
		currentHeight = math.Min(currentHeight, ground.OceanMouthApronHeight)
	}

	seaBoundary := normalizedHeightBlockBoundary(sea)
	deepFloor := math.Ceil(seaBoundary) - 1 - minimumOffshoreDepthBlocks + 0.5
	if math.Floor(normalizedHeightBlockBoundary(currentHeight)) <= math.Floor(deepFloor) {
		return currentHeight
	}

	// Estimate the nearest dry shore over one chunk, from both continuous
	// terrain elevation and the ocean mask. Each crossing approaches the
	// outer sample smoothly, so the beach blends into navigable water.
	shoreFraction := 1.0
	base := math.Min(ground.BaseHeight, sea)
	threshold := m.Config.OceanEligibilityThreshold
	for _, offset := range shoreDepthOffsets {
		sx, sz := x+offset[0], z+offset[1]
		neighbor := m.Ground.Base.SampleNormalized(sx, sz)
		if neighbor > sea {
			shoreFraction = math.Min(shoreFraction, (sea-base)/(neighbor-base))
		}
		neighborOcean := m.OceanEligibility.SampleSmoothstepBilinear(sx, sz)
		if neighborOcean < threshold {
			shoreFraction = math.Min(shoreFraction, (ocean-threshold)/(ocean-neighborOcean))
		}
	}
	blend := shoreFraction * shoreFraction * (3 - 2*shoreFraction)
	target := normalizedHeightFromBlockBoundary(seaBoundary + (deepFloor-seaBoundary)*blend)
	return math.Min(currentHeight, target)
}

func normalizedHeightBlockBoundary(height float64) float64 {
	height = math.Max(0, math.Min(1, height))
	if height <= 0.5 {
		return -64 + 254*height
	}
	return 63 + 514*(height-0.5)
}

func normalizedHeightFromBlockBoundary(boundary float64) float64 {
	if boundary <= 63 {
		return math.Max(0, math.Min(1, (boundary+64)/254))
	}
	return math.Max(0, math.Min(1, 0.5+(boundary-63)/514))
}

func normalizedBlockStepAbove(height float64) float64 {
	if height < 0.5 {
		return 1.0 / 254.0
	}
	return 1.0 / 514.0
}

func normalizeConfig(cfg Config) (Config, error) {
	if cfg.RiverCenterDepthNormalized == 0 {
		cfg.RiverCenterDepthNormalized = 0.0025
	}
	if !finite(cfg.RiverCenterDepthNormalized) ||
		cfg.RiverCenterDepthNormalized <= 0 ||
		cfg.RiverCenterDepthNormalized >= 0.05 {
		return Config{}, errors.New("worldsample: river center depth must be finite and inside (0,0.05)")
	}
	if cfg.MinRiverWaterDepthNormalized == 0 {
		cfg.MinRiverWaterDepthNormalized = 0.0015
	}
	if !finite(cfg.MinRiverWaterDepthNormalized) ||
		cfg.MinRiverWaterDepthNormalized <= 0 ||
		cfg.MinRiverWaterDepthNormalized >= cfg.RiverCenterDepthNormalized {
		return Config{}, errors.New("worldsample: minimum river water depth must be finite, positive and below center depth")
	}

	if cfg.OceanEligibilityThreshold == 0 {
		cfg.OceanEligibilityThreshold = 0.5
	}
	if !finite(cfg.OceanEligibilityThreshold) ||
		cfg.OceanEligibilityThreshold <= 0 ||
		cfg.OceanEligibilityThreshold >= 1 {
		return Config{}, errors.New("worldsample: ocean eligibility threshold must be finite and inside (0,1)")
	}
	if cfg.CoastalBackwaterVersion > 2 {
		return Config{}, errors.New("worldsample: unsupported coastal backwater version")
	}
	if cfg.OceanBathymetryVersion > 1 {
		return Config{}, errors.New("worldsample: unsupported ocean bathymetry version")
	}
	if cfg.RiverNavigationVersion > 1 {
		return Config{}, errors.New("worldsample: unsupported river navigation version")
	}
	if cfg.RiverNavigationVersion >= 1 {
		if cfg.MinNavigableRiverDepthBlocks == 0 {
			cfg.MinNavigableRiverDepthBlocks = 3
		}
		if cfg.MinNavigableRiverDepthBlocks > 16 {
			return Config{}, errors.New("worldsample: navigable river depth must be inside 1..16 blocks")
		}
	} else if cfg.MinNavigableRiverDepthBlocks != 0 {
		return Config{}, errors.New("worldsample: navigable river depth requires navigation version")
	}
	return cfg, nil
}

func finite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
