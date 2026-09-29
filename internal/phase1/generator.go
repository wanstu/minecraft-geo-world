package phase1

import (
	"math"
	"math/rand/v2"
	"sort"

	"geoworld/internal/world"
)

type continentSeed struct {
	Center Vec2
	Radius Vec2
	Weight float64
}

type cellPoint struct {
	X int
	Y int
}

const (
	crustInteriorScaleBlocks    = 18 * DefaultReferenceCellSizeBlocks
	landBaseInteriorScaleBlocks = 22 * DefaultReferenceCellSizeBlocks
	continentalShelfWidthBlocks = 6 * DefaultReferenceCellSizeBlocks
	continentalSlopeWidthBlocks = 18 * DefaultReferenceCellSizeBlocks
	oceanBasinDepthScaleBlocks  = 28 * DefaultReferenceCellSizeBlocks
	plateauInteriorMinBlocks    = 18 * DefaultReferenceCellSizeBlocks
	continentalCoastWidthBlocks = 3 * DefaultReferenceCellSizeBlocks
	coastRenderLandScaleBlocks  = 36 * DefaultReferenceCellSizeBlocks
	coastRenderOceanScaleBlocks = 42 * DefaultReferenceCellSizeBlocks
)

func Generate(cfg Config) *World {
	if cfg.Width <= 0 {
		cfg.Width = 320
	}
	if cfg.Height <= 0 {
		cfg.Height = 200
	}
	if cfg.PlateCount < 2 {
		cfg.PlateCount = 18
	}
	if cfg.OceanFraction <= 0 || cfg.OceanFraction >= 1 {
		cfg.OceanFraction = 0.68
	}
	if cfg.EdgeOceanBufferFraction <= 0 || cfg.EdgeOceanBufferFraction >= 0.20 {
		cfg.EdgeOceanBufferFraction = 0.04
	}
	if cfg.SpanXBlocks == 0 {
		cfg.SpanXBlocks = float64(cfg.Width-1) * DefaultReferenceCellSizeBlocks
	}
	if cfg.SpanZBlocks == 0 {
		cfg.SpanZBlocks = float64(cfg.Height-1) * DefaultReferenceCellSizeBlocks
	}
	geometry := world.GridGeometry{
		Width: cfg.Width, Height: cfg.Height,
		OriginX: cfg.OriginX, OriginZ: cfg.OriginZ,
		SpanXBlocks: cfg.SpanXBlocks, SpanZBlocks: cfg.SpanZBlocks,
	}
	if _, err := geometry.SquareCellSize(); err != nil {
		panic(err) // Generate keeps the legacy non-error API until Phase 6.2.
	}

	r := rand.New(rand.NewPCG(uint64(cfg.Seed), uint64(cfg.Seed)^0x9e3779b97f4a7c15))
	plates := make([]Plate, cfg.PlateCount)

	for i := range plates {
		angle := r.Float64() * math.Pi * 2
		speed := 0.35 + r.Float64()*0.9
		plates[i] = Plate{
			ID: i,
			Center: Vec2{
				X: r.Float64() * float64(cfg.Width-1),
				Y: r.Float64() * float64(cfg.Height-1),
			},
			Velocity: Vec2{
				X: math.Cos(angle) * speed,
				Y: math.Sin(angle) * speed,
			},
		}
	}

	n := cfg.Width * cfg.Height
	w := &World{
		Geometry:          geometry,
		Config:            cfg,
		Plates:            plates,
		RawPlateID:        make([]int, n),
		PlateID:           make([]int, n),
		Crust:             make([]CrustType, n),
		CrustThickness:    make([]float64, n),
		CoastDistance:     make([]float64, n),
		BaseElevation:     make([]float64, n),
		TerrainZone:       make([]MacroTerrain, n),
		LandPotential:     make([]float64, n),
		Boundary:          make([]BoundaryType, n),
		BoundaryStrength:  make([]float64, n),
		OceanAge:          make([]float64, n),
		Feature:           make([]TectonicFeature, n),
		TectonicIntensity: make([]float64, n),
		TectonicInfluence: make([]float64, n),
		InfluenceFeature:  make([]TectonicFeature, n),
		Elevation:         make([]float64, n),
	}

	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			i := w.index(x, y)
			fx, fy := float64(x), float64(y)
			w.RawPlateID[i] = nearestPlate(fx, fy, plates)
			wx, wy := warpedPlatePoint(fx, fy, cfg)
			w.PlateID[i] = nearestPlate(wx, wy, plates)
		}
	}

	w.generateCrust(r)
	w.computeCoastDistance()
	w.computeCrustThickness()
	w.classifyBoundaries()
	w.computeOceanAge()
	w.classifyFeatures()
	w.computeTectonicInfluence()
	w.computeBaseElevation()
	w.classifyMacroTerrain()
	w.computeElevation()
	// All downstream stages share this macro surface. Final continuous terrain
	// requires a separate coastline/water reconciliation in Phase 6.2.
	surface, err := world.ClassifySurface(cfg.Width, cfg.Height, w.Elevation, world.DefaultSeaLevelNormalized)
	if err != nil {
		panic(err) // Generate's legacy API cannot return errors; generated heights must satisfy this contract.
	}
	w.Surface = surface
	return w
}

func nearestPlate(x, y float64, plates []Plate) int {
	best := 0
	bestD := math.MaxFloat64
	for i, p := range plates {
		dx, dy := x-p.Center.X, y-p.Center.Y
		d := dx*dx + dy*dy
		if d < bestD {
			bestD = d
			best = i
		}
	}
	return best
}

// warpedPlatePoint 对查询坐标做低频连续扭曲，再进行最近板块中心归属。
// 这样保留连续板块，同时降低标准 Voronoi 的直线和规则多边形感。
func warpedPlatePoint(x, y float64, cfg Config) (float64, float64) {
	w := float64(cfg.Width)
	h := float64(cfg.Height)
	minSize := math.Min(w, h)
	amp := minSize * 0.032

	seedPhaseA := float64((cfg.Seed%10007)+10007) / 10007 * math.Pi * 2
	seedPhaseB := float64((cfg.Seed%7919)+7919) / 7919 * math.Pi * 2

	x1 := x / w * math.Pi * 2
	y1 := y / h * math.Pi * 2
	diag := (x + y) / minSize * math.Pi

	dx := math.Sin(y1*2.15+seedPhaseA)*0.58 +
		math.Sin(diag*1.37+seedPhaseB)*0.27 +
		math.Cos(x1*0.83-y1*0.71+seedPhaseA*0.5)*0.15
	dy := math.Cos(x1*1.85+seedPhaseB)*0.56 +
		math.Sin(diag*1.19-seedPhaseA)*0.29 +
		math.Sin(y1*0.91-x1*0.63+seedPhaseB*0.5)*0.15

	return x + dx*amp, y + dy*amp
}

func (w *World) generateCrust(r *rand.Rand) {
	type landCandidate struct {
		index     int
		potential float64
	}

	seedCount := maxInt(4, w.Config.PlateCount/3)
	seeds := make([]continentSeed, seedCount)
	width := float64(w.Config.Width)
	height := float64(w.Config.Height)

	for i := range seeds {
		seeds[i] = continentSeed{
			Center: Vec2{
				X: width * (0.14 + r.Float64()*0.72),
				Y: height * (0.14 + r.Float64()*0.72),
			},
			Radius: Vec2{
				X: width * (0.09 + r.Float64()*0.12),
				Y: height * (0.10 + r.Float64()*0.15),
			},
			Weight: 0.75 + r.Float64()*0.55,
		}
	}

	bufferCells := edgeOceanBufferCells(w.Config)
	candidates := make([]landCandidate, 0, w.Config.Width*w.Config.Height)

	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			fx, fy := float64(x), float64(y)
			potential := 0.0

			for _, s := range seeds {
				dx := (fx - s.Center.X) / s.Radius.X
				dy := (fy - s.Center.Y) / s.Radius.Y
				potential += s.Weight * math.Exp(-(dx*dx+dy*dy)*0.5)
			}

			potential += math.Sin(fx*0.031+float64(w.Config.Seed%401)) * 0.10
			potential += math.Cos(fy*0.037-float64(w.Config.Seed%271)) * 0.08
			potential += math.Sin((fx+fy)*0.019) * 0.05

			// 即使硬缓冲区已经保证外围是海洋，靠近缓冲区的区域仍给予
			// 连续海洋偏置，避免大陆像被矩形刀切断一样贴着 guard band 生长。
			dxEdge := math.Min(fx, width-1-fx) / width
			dyEdge := math.Min(fy, height-1-fy) / height
			edge := math.Min(dxEdge, dyEdge)
			if edge < 0.12 {
				potential -= (0.12 - edge) * 8.0
			}

			i := w.index(x, y)
			w.LandPotential[i] = potential
			w.Crust[i] = Oceanic

			if x >= bufferCells && y >= bufferCells &&
				x < w.Config.Width-bufferCells &&
				y < w.Config.Height-bufferCells {
				candidates = append(candidates, landCandidate{index: i, potential: potential})
			}
		}
	}

	landCount := int(math.Round(float64(len(w.Crust)) * (1 - w.Config.OceanFraction)))
	landCount = clampInt(landCount, 1, len(candidates))

	// 只从 guard band 内侧挑选恰好 landCount 个大陆 Cell。
	// 这样外围海洋缓冲是硬约束，同时全图海洋覆盖率仍能精确命中目标。
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].potential == candidates[j].potential {
			return candidates[i].index < candidates[j].index
		}
		return candidates[i].potential > candidates[j].potential
	})

	for i := 0; i < landCount; i++ {
		w.Crust[candidates[i].index] = Continental
	}
}

func edgeOceanBufferCells(cfg Config) int {
	minSize := minInt(cfg.Width, cfg.Height)
	cells := int(math.Round(float64(minSize) * cfg.EdgeOceanBufferFraction))
	return clampInt(cells, 2, maxInt(2, minSize/5))
}

// computeCoastDistance calculates signed physical distance to the nearest
// crust coastline. Distances are stored in Minecraft blocks rather than cells;
// land is positive, ocean is negative, and coastline nodes on both sides are 0.
func (w *World) computeCoastDistance() {
	const unvisited = -1
	dist := make([]int, len(w.CoastDistance))
	for i := range dist {
		dist[i] = unvisited
	}

	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	queue := make([]cellPoint, 0, len(dist)/6)

	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			for _, d := range dirs {
				nx, ny := x+d[0], y+d[1]
				if !w.inside(nx, ny) {
					continue
				}
				if w.Crust[w.index(nx, ny)] != w.Crust[i] {
					dist[i] = 0
					queue = append(queue, cellPoint{X: x, Y: y})
					break
				}
			}
		}
	}

	head := 0
	for head < len(queue) {
		p := queue[head]
		head++
		pi := w.index(p.X, p.Y)
		for _, d := range dirs {
			nx, ny := p.X+d[0], p.Y+d[1]
			if !w.inside(nx, ny) {
				continue
			}
			ni := w.index(nx, ny)
			if w.Crust[ni] != w.Crust[pi] || dist[ni] != unvisited {
				continue
			}
			dist[ni] = dist[pi] + 1
			queue = append(queue, cellPoint{X: nx, Y: ny})
		}
	}

	cellSizeBlocks := w.Geometry.SpacingX()
	for i := range w.CoastDistance {
		d := dist[i]
		if d < 0 {
			d = maxInt(w.Config.Width, w.Config.Height)
		}
		distanceBlocks := float64(d) * cellSizeBlocks
		if w.Crust[i] == Continental {
			w.CoastDistance[i] = distanceBlocks
		} else {
			w.CoastDistance[i] = -distanceBlocks
		}
	}
}

func (w *World) computeCrustThickness() {
	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			if w.Crust[i] != Continental {
				w.CrustThickness[i] = 0.18
				continue
			}

			d := math.Max(0, w.CoastDistance[i])
			interior := 1 - math.Exp(-d/crustInteriorScaleBlocks)
			lowFreq := math.Sin(float64(x)*0.021+float64(w.Config.Seed%421))*0.025 +
				math.Cos(float64(y)*0.019-float64(w.Config.Seed%353))*0.02
			w.CrustThickness[i] = clamp(0.48+interior*0.34+lowFreq, 0.44, 0.88)
		}
	}
}

func (w *World) classifyBoundaries() {
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			aID := w.PlateID[i]
			best := NoBoundary
			bestStrength := 0.0

			for _, d := range dirs {
				nx, ny := x+d[0], y+d[1]
				if !w.inside(nx, ny) {
					continue
				}
				ni := w.index(nx, ny)
				bID := w.PlateID[ni]
				if bID == aID {
					continue
				}

				boundary, strength := boundaryKinematics(w.Plates[aID], w.Plates[bID])
				if boundaryPriority(boundary) > boundaryPriority(best) ||
					(boundary == best && strength > bestStrength) {
					best = boundary
					bestStrength = strength
				}
			}
			w.Boundary[i] = best
			w.BoundaryStrength[i] = bestStrength
		}
	}
}

func (w *World) computeOceanAge() {
	const unvisited = -1
	dist := make([]int, len(w.OceanAge))
	for i := range dist {
		dist[i] = unvisited
	}

	queue := make([]cellPoint, 0, len(dist)/8)
	addSeed := func(x, y int) {
		i := w.index(x, y)
		if w.Crust[i] != Oceanic || dist[i] == 0 {
			return
		}
		dist[i] = 0
		queue = append(queue, cellPoint{X: x, Y: y})
	}

	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			if w.Crust[i] != Oceanic {
				continue
			}
			for _, d := range [][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
				nx, ny := x+d[0], y+d[1]
				if !w.inside(nx, ny) {
					continue
				}
				ni := w.index(nx, ny)
				if w.Crust[ni] != Oceanic || w.PlateID[ni] == w.PlateID[i] {
					continue
				}
				if classifyBoundary(w.Plates[w.PlateID[i]], w.Plates[w.PlateID[ni]]) == Divergent {
					addSeed(x, y)
					break
				}
			}
		}
	}

	if len(queue) == 0 {
		for y := 0; y < w.Config.Height; y++ {
			for x := 0; x < w.Config.Width; x++ {
				i := w.index(x, y)
				if w.Crust[i] == Oceanic && w.Boundary[i] != NoBoundary {
					addSeed(x, y)
				}
			}
		}
	}

	head := 0
	maxDist := 0
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for head < len(queue) {
		p := queue[head]
		head++
		pi := w.index(p.X, p.Y)

		for _, d := range dirs {
			nx, ny := p.X+d[0], p.Y+d[1]
			if !w.inside(nx, ny) {
				continue
			}
			ni := w.index(nx, ny)
			if w.Crust[ni] != Oceanic || dist[ni] != unvisited {
				continue
			}
			dist[ni] = dist[pi] + 1
			if dist[ni] > maxDist {
				maxDist = dist[ni]
			}
			queue = append(queue, cellPoint{X: nx, Y: ny})
		}
	}

	scale := math.Max(12, float64(maxDist)*0.42)
	for i := range w.OceanAge {
		if w.Crust[i] != Oceanic {
			w.OceanAge[i] = 0
			continue
		}
		if dist[i] < 0 {
			w.OceanAge[i] = 0.65
			continue
		}
		w.OceanAge[i] = clamp(1-math.Exp(-float64(dist[i])/scale), 0, 1)
	}
}

func (w *World) classifyFeatures() {
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			aID := w.PlateID[i]
			best := NoFeature
			bestIntensity := 0.0

			for _, d := range dirs {
				nx, ny := x+d[0], y+d[1]
				if !w.inside(nx, ny) {
					continue
				}
				ni := w.index(nx, ny)
				bID := w.PlateID[ni]
				if bID == aID {
					continue
				}

				boundary, boundaryStrength := boundaryKinematics(w.Plates[aID], w.Plates[bID])
				feature := classifyFeature(
					w.Crust[i], w.Crust[ni],
					w.OceanAge[i], w.OceanAge[ni],
					aID, bID,
					boundary,
				)
				intensity := tectonicFeatureIntensity(
					feature,
					boundaryStrength,
					w.CrustThickness[i], w.CrustThickness[ni],
					w.OceanAge[i], w.OceanAge[ni],
				)

				if featurePriority(feature) > featurePriority(best) ||
					(feature == best && intensity > bestIntensity) {
					best = feature
					bestIntensity = intensity
				}
			}
			w.Feature[i] = best
			w.TectonicIntensity[i] = bestIntensity
		}
	}
}

func classifyBoundary(a, b Plate) BoundaryType {
	boundary, _ := boundaryKinematics(a, b)
	return boundary
}

func boundaryKinematics(a, b Plate) (BoundaryType, float64) {
	nx := b.Center.X - a.Center.X
	ny := b.Center.Y - a.Center.Y
	l := math.Hypot(nx, ny)
	if l == 0 {
		return Transform, 0
	}
	nx /= l
	ny /= l

	rvx := a.Velocity.X - b.Velocity.X
	rvy := a.Velocity.Y - b.Velocity.Y
	closing := rvx*nx + rvy*ny
	relative := math.Hypot(rvx, rvy)
	tangentSq := relative*relative - closing*closing
	if tangentSq < 0 {
		tangentSq = 0
	}
	tangent := math.Sqrt(tangentSq)

	switch {
	case closing > 0.22:
		return Convergent, clamp(0.18+(closing-0.22)/1.55, 0, 1)
	case closing < -0.22:
		return Divergent, clamp(0.18+(-closing-0.22)/1.55, 0, 1)
	default:
		return Transform, clamp(0.12+tangent/1.8, 0, 1)
	}
}

func boundaryPriority(b BoundaryType) int {
	switch b {
	case Convergent:
		return 3
	case Divergent:
		return 2
	case Transform:
		return 1
	default:
		return 0
	}
}

func classifyFeature(local, other CrustType, localAge, otherAge float64, localPlate, otherPlate int, boundary BoundaryType) TectonicFeature {
	switch boundary {
	case Convergent:
		switch {
		case local == Continental && other == Continental:
			return CollisionOrogen
		case local == Continental && other == Oceanic:
			return VolcanicArc
		case local == Oceanic && other == Continental:
			return Trench
		default:
			localSubducts := localAge > otherAge+0.015 ||
				(math.Abs(localAge-otherAge) <= 0.015 && localPlate > otherPlate)
			if localSubducts {
				return Trench
			}
			return IslandArc
		}
	case Divergent:
		if local == Continental || other == Continental {
			return ContinentalRift
		}
		return MidOceanRidge
	case Transform:
		return TransformFault
	default:
		return NoFeature
	}
}

func tectonicFeatureIntensity(feature TectonicFeature, boundaryStrength, localThickness, otherThickness, localAge, otherAge float64) float64 {
	base := clamp(boundaryStrength, 0, 1)
	switch feature {
	case CollisionOrogen:
		thickness := clamp((localThickness+otherThickness)/1.5, 0, 1)
		return clamp(base*(0.72+0.38*thickness), 0, 1)
	case VolcanicArc:
		return clamp(base*(0.72+0.22*otherAge+0.12*localThickness), 0, 1)
	case Trench:
		age := math.Max(localAge, otherAge)
		return clamp(base*(0.68+0.38*age), 0, 1)
	case IslandArc:
		ageContrast := math.Abs(localAge - otherAge)
		return clamp(base*(0.68+0.24*math.Max(localAge, otherAge)+0.12*ageContrast), 0, 1)
	case MidOceanRidge:
		return clamp(base*0.92, 0, 1)
	case ContinentalRift:
		thickness := math.Max(localThickness, otherThickness)
		return clamp(base*(0.68+0.28*thickness), 0, 1)
	case TransformFault:
		return clamp(base*0.72, 0, 1)
	default:
		return 0
	}
}

func (w *World) computeTectonicInfluence() {
	uplift := make([]float64, len(w.TectonicInfluence))
	subsidence := make([]float64, len(w.TectonicInfluence))
	upliftFeature := make([]TectonicFeature, len(w.TectonicInfluence))
	subsidenceFeature := make([]TectonicFeature, len(w.TectonicInfluence))
	cellSizeBlocks := w.Geometry.SpacingX()

	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			feature := w.Feature[i]
			intensity := w.TectonicIntensity[i]
			radiusBlocks, amplitude := featureBand(feature, intensity)
			if radiusBlocks == 0 || amplitude == 0 {
				continue
			}
			radiusCells := maxInt(1, int(math.Ceil(radiusBlocks/cellSizeBlocks)))

			for dy := -radiusCells; dy <= radiusCells; dy++ {
				for dx := -radiusCells; dx <= radiusCells; dx++ {
					tx, ty := x+dx, y+dy
					if !w.inside(tx, ty) {
						continue
					}
					dBlocks := math.Hypot(float64(dx), float64(dy)) * cellSizeBlocks
					if dBlocks > radiusBlocks {
						continue
					}
					ti := w.index(tx, ty)
					if !featureAppliesToCrust(feature, w.Crust[ti]) {
						continue
					}

					falloff := 1 - dBlocks/radiusBlocks
					effect := amplitude * falloff * falloff
					if effect > 0 && effect > uplift[ti] {
						uplift[ti] = effect
						upliftFeature[ti] = feature
					}
					if effect < 0 && effect < subsidence[ti] {
						subsidence[ti] = effect
						subsidenceFeature[ti] = feature
					}
				}
			}
		}
	}

	for i := range w.TectonicInfluence {
		w.TectonicInfluence[i] = clamp(uplift[i]+subsidence[i], -0.38, 0.45)
		if math.Abs(uplift[i]) >= math.Abs(subsidence[i]) {
			w.InfluenceFeature[i] = upliftFeature[i]
		} else {
			w.InfluenceFeature[i] = subsidenceFeature[i]
		}
	}
}

// featureBand resolves legacy cell radii into a physical reference distance.
// The reference keeps the existing 128-block baseline byte-for-byte while
// allowing a denser macro grid to sample the same physical influence radius.
func featureBand(feature TectonicFeature, intensity float64) (radiusBlocks, amplitude float64) {
	intensity = clamp(intensity, 0, 1)
	scaleR := 0.72 + intensity*0.58
	scaleA := 0.52 + intensity*0.78

	baseRadiusCells := 0
	baseAmplitude := 0.0
	switch feature {
	case CollisionOrogen:
		baseRadiusCells, baseAmplitude = 14, 0.34
	case VolcanicArc:
		baseRadiusCells, baseAmplitude = 9, 0.21
	case IslandArc:
		baseRadiusCells, baseAmplitude = 7, 0.18
	case Trench:
		baseRadiusCells, baseAmplitude = 6, -0.24
	case MidOceanRidge:
		baseRadiusCells, baseAmplitude = 8, 0.12
	case ContinentalRift:
		baseRadiusCells, baseAmplitude = 10, -0.13
	case TransformFault:
		baseRadiusCells, baseAmplitude = 4, -0.025
	default:
		return 0, 0
	}

	referenceRadiusCells := maxInt(2, int(math.Round(float64(baseRadiusCells)*scaleR)))
	radiusBlocks = float64(referenceRadiusCells) * DefaultReferenceCellSizeBlocks
	amplitude = baseAmplitude * scaleA
	return radiusBlocks, amplitude
}

func featureAppliesToCrust(feature TectonicFeature, crust CrustType) bool {
	switch feature {
	case CollisionOrogen, VolcanicArc, ContinentalRift:
		return crust == Continental
	case IslandArc, Trench, MidOceanRidge:
		return crust == Oceanic
	default:
		return true
	}
}

func featurePriority(f TectonicFeature) int {
	switch f {
	case CollisionOrogen:
		return 8
	case VolcanicArc:
		return 7
	case Trench:
		return 6
	case IslandArc:
		return 5
	case ContinentalRift:
		return 4
	case MidOceanRidge:
		return 3
	case TransformFault:
		return 2
	default:
		return 0
	}
}

func (w *World) computeBaseElevation() {
	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)

			if w.Crust[i] == Continental {
				d := math.Max(0, w.CoastDistance[i])
				thickness := clamp((w.CrustThickness[i]-0.44)/(0.88-0.44), 0, 1)
				interior := 1 - math.Exp(-d/landBaseInteriorScaleBlocks)
				w.BaseElevation[i] = clamp(0.535+thickness*0.075+interior*0.035, 0.525, 0.66)
				continue
			}

			d := math.Max(0, -w.CoastDistance[i])
			switch {
			case d <= continentalShelfWidthBlocks:
				// 大陆架：浅海缓慢下降。
				t := d / continentalShelfWidthBlocks
				w.BaseElevation[i] = 0.485 - t*0.045
			case d <= continentalSlopeWidthBlocks:
				// 大陆坡：从浅海快速进入深海。
				t := (d - continentalShelfWidthBlocks) / (continentalSlopeWidthBlocks - continentalShelfWidthBlocks)
				w.BaseElevation[i] = 0.44 - t*0.15
			default:
				// 深海盆地：距离海岸越远、地壳越老，平均海底越深。
				deep := 1 - math.Exp(-(d-continentalSlopeWidthBlocks)/oceanBasinDepthScaleBlocks)
				w.BaseElevation[i] = 0.29 - deep*0.045 - w.OceanAge[i]*0.045
			}
			w.BaseElevation[i] = clamp(w.BaseElevation[i], 0.16, 0.495)
		}
	}
}

func (w *World) classifyMacroTerrain() {
	for i := range w.TerrainZone {
		influence := w.TectonicInfluence[i]
		feature := w.InfluenceFeature[i]

		if w.Crust[i] == Continental {
			switch {
			case feature == ContinentalRift && influence < -0.02:
				w.TerrainZone[i] = RiftValleyZone
			case (feature == CollisionOrogen || feature == VolcanicArc) && influence > 0.10:
				w.TerrainZone[i] = MountainBelt
			case w.CoastDistance[i] >= plateauInteriorMinBlocks && w.CrustThickness[i] >= 0.76 && influence > 0.035:
				// 高原不是“普通大陆内部”的同义词；要求厚地壳并存在广域构造抬升。
				w.TerrainZone[i] = Plateau
			case w.CoastDistance[i] <= continentalCoastWidthBlocks:
				w.TerrainZone[i] = ContinentalCoast
			default:
				w.TerrainZone[i] = ContinentalInterior
			}
			continue
		}

		switch {
		case feature == Trench && influence < -0.025:
			w.TerrainZone[i] = TrenchZone
		case feature == IslandArc && influence > 0.025:
			w.TerrainZone[i] = IslandArcZone
		case feature == MidOceanRidge && influence > 0.02:
			w.TerrainZone[i] = MidOceanRidgeZone
		default:
			d := math.Max(0, -w.CoastDistance[i])
			switch {
			case d <= continentalShelfWidthBlocks:
				w.TerrainZone[i] = ContinentalShelf
			case d <= continentalSlopeWidthBlocks:
				w.TerrainZone[i] = ContinentalSlope
			default:
				w.TerrainZone[i] = OceanBasin
			}
		}
	}
}

func (w *World) computeElevation() {
	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			base := w.BaseElevation[i] + w.TectonicInfluence[i]

			// 只提供局部粗糙度，不允许噪声重新决定宏观海陆和构造位置。
			n := math.Sin(float64(x)*0.071+float64(w.Config.Seed%997)) *
				math.Cos(float64(y)*0.083-float64(w.Config.Seed%313))
			roughness := 0.018
			if w.Crust[i] == Oceanic {
				roughness = 0.012
			}
			base += n * roughness

			w.Elevation[i] = clamp(base, 0, 1)
		}
	}
}

func (w *World) index(x, y int) int {
	return y*w.Config.Width + x
}

func (w *World) inside(x, y int) bool {
	return x >= 0 && y >= 0 && x < w.Config.Width && y < w.Config.Height
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

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
