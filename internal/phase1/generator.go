package phase1

import (
	"math"
	"math/rand/v2"
	"sort"
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
		Config:            cfg,
		Plates:            plates,
		RawPlateID:        make([]int, n),
		PlateID:           make([]int, n),
		Crust:             make([]CrustType, n),
		LandPotential:     make([]float64, n),
		Boundary:          make([]BoundaryType, n),
		OceanAge:          make([]float64, n),
		Feature:           make([]TectonicFeature, n),
		TectonicInfluence: make([]float64, n),
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
	w.classifyBoundaries()
	w.computeOceanAge()
	w.classifyFeatures()
	w.computeTectonicInfluence()
	w.computeElevation()
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

// warpedPlatePoint 对“查询坐标”做平滑低频扭曲，再执行最近板块中心查询。
// 它保留 Voronoi 的连续大区块优势，同时降低笔直边界和规则多边形感。
// 这里故意不使用逐 Cell 随机抖动，避免板块边缘出现盐胡椒碎片。
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

			dxEdge := math.Min(fx, width-1-fx) / width
			dyEdge := math.Min(fy, height-1-fy) / height
			edge := math.Min(dxEdge, dyEdge)
			if edge < 0.12 {
				potential -= (0.12 - edge) * 8.0
			}

			w.LandPotential[w.index(x, y)] = potential
		}
	}

	values := append([]float64(nil), w.LandPotential...)
	sort.Float64s(values)
	landCount := int(math.Round(float64(len(values)) * (1 - w.Config.OceanFraction)))
	landCount = clampInt(landCount, 1, len(values)-1)
	threshold := values[len(values)-landCount]

	for i, p := range w.LandPotential {
		if p >= threshold {
			w.Crust[i] = Continental
		} else {
			w.Crust[i] = Oceanic
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
				best = strongerBoundary(best, classifyBoundary(w.Plates[aID], w.Plates[bID]))
			}
			w.Boundary[i] = best
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

	// 真正的年龄源首先来自海洋-海洋张裂边界（洋中脊）。
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

	// 极端 Seed 可能没有海洋-海洋张裂边界；退化时用海洋板块边缘作为年轻地壳源，
	// 保证年龄场仍然连续可用，而不是整张图失效。
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
			// 与年龄源不连通的小海盆，给一个中等偏老的稳定值。
			w.OceanAge[i] = 0.65
			continue
		}
		// 距洋中脊越远越老；指数曲线避免最远处全部挤在 1.0。
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

				boundary := classifyBoundary(w.Plates[aID], w.Plates[bID])
				feature := classifyFeature(
					w.Crust[i], w.Crust[ni],
					w.OceanAge[i], w.OceanAge[ni],
					aID, bID,
					boundary,
				)
				best = strongerFeature(best, feature)
			}
			w.Feature[i] = best
		}
	}
}

func classifyBoundary(a, b Plate) BoundaryType {
	nx := b.Center.X - a.Center.X
	ny := b.Center.Y - a.Center.Y
	l := math.Hypot(nx, ny)
	if l == 0 {
		return Transform
	}
	nx /= l
	ny /= l

	closing := (a.Velocity.X-b.Velocity.X)*nx + (a.Velocity.Y-b.Velocity.Y)*ny
	switch {
	case closing > 0.22:
		return Convergent
	case closing < -0.22:
		return Divergent
	default:
		return Transform
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
			// 海洋-海洋汇聚：较老、较冷的一侧优先俯冲形成海沟，
			// 较年轻的上覆板块一侧形成岛弧。年龄接近时用 plate ID 稳定打破平局。
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

func (w *World) computeTectonicInfluence() {
	uplift := make([]float64, len(w.TectonicInfluence))
	subsidence := make([]float64, len(w.TectonicInfluence))

	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			feature := w.Feature[i]
			radius, amplitude := featureBand(feature)
			if radius == 0 || amplitude == 0 {
				continue
			}

			for dy := -radius; dy <= radius; dy++ {
				for dx := -radius; dx <= radius; dx++ {
					tx, ty := x+dx, y+dy
					if !w.inside(tx, ty) {
						continue
					}
					d := math.Hypot(float64(dx), float64(dy))
					if d > float64(radius) {
						continue
					}
					ti := w.index(tx, ty)
					if !featureAppliesToCrust(feature, w.Crust[ti]) {
						continue
					}
					falloff := 1 - d/float64(radius)
					effect := amplitude * falloff * falloff
					if effect > 0 && effect > uplift[ti] {
						uplift[ti] = effect
					}
					if effect < 0 && effect < subsidence[ti] {
						subsidence[ti] = effect
					}
				}
			}
		}
	}

	for i := range w.TectonicInfluence {
		w.TectonicInfluence[i] = clamp(uplift[i]+subsidence[i], -0.38, 0.45)
	}
}

func featureBand(feature TectonicFeature) (radius int, amplitude float64) {
	switch feature {
	case CollisionOrogen:
		return 14, 0.34
	case VolcanicArc:
		return 9, 0.21
	case IslandArc:
		return 7, 0.18
	case Trench:
		return 6, -0.24
	case MidOceanRidge:
		return 8, 0.12
	case ContinentalRift:
		return 10, -0.13
	case TransformFault:
		return 4, -0.025
	default:
		return 0, 0
	}
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

func strongerBoundary(a, b BoundaryType) BoundaryType {
	if a == Convergent || b == Convergent {
		return Convergent
	}
	if a == Divergent || b == Divergent {
		return Divergent
	}
	if a == Transform || b == Transform {
		return Transform
	}
	return NoBoundary
}

func strongerFeature(a, b TectonicFeature) TectonicFeature {
	if featurePriority(b) > featurePriority(a) {
		return b
	}
	return a
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

func (w *World) computeElevation() {
	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)

			base := 0.66
			if w.Crust[i] == Oceanic {
				// 年轻海洋地壳较热、海底较高；老地壳冷却收缩后海盆更深。
				base = 0.31 - w.OceanAge[i]*0.10
			}

			base += w.TectonicInfluence[i]

			// 只提供局部粗糙度，不允许噪声重新决定大陆 / 山系位置。
			n := math.Sin(float64(x)*0.071+float64(w.Config.Seed%997)) *
				math.Cos(float64(y)*0.083-float64(w.Config.Seed%313))
			base += n * 0.03

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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
