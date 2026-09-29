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
		Config:        cfg,
		Plates:        plates,
		PlateID:       make([]int, n),
		Crust:         make([]CrustType, n),
		LandPotential: make([]float64, n),
		Boundary:      make([]BoundaryType, n),
		Feature:       make([]TectonicFeature, n),
		Elevation:     make([]float64, n),
	}

	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			w.PlateID[w.index(x, y)] = nearestPlate(float64(x), float64(y), plates)
		}
	}

	w.generateCrust(r)
	w.classifyTectonics()
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

func (w *World) classifyTectonics() {
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			aID := w.PlateID[i]
			bestBoundary := NoBoundary
			bestFeature := NoFeature

			for _, d := range dirs {
				nx, ny := x+d[0], y+d[1]
				if nx < 0 || ny < 0 || nx >= w.Config.Width || ny >= w.Config.Height {
					continue
				}
				ni := w.index(nx, ny)
				bID := w.PlateID[ni]
				if bID == aID {
					continue
				}

				boundary := classifyBoundary(w.Plates[aID], w.Plates[bID])
				feature := classifyFeature(w.Crust[i], w.Crust[ni], boundary)

				bestBoundary = strongerBoundary(bestBoundary, boundary)
				bestFeature = strongerFeature(bestFeature, feature)
			}

			w.Boundary[i] = bestBoundary
			w.Feature[i] = bestFeature
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

func classifyFeature(local, other CrustType, boundary BoundaryType) TectonicFeature {
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
			return OceanicSubduction
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
	case OceanicSubduction:
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
				base = 0.25
			}

			switch w.Feature[i] {
			case CollisionOrogen:
				base += 0.30
			case VolcanicArc:
				base += 0.20
			case Trench:
				base -= 0.18
			case OceanicSubduction:
				base += 0.06
			case MidOceanRidge:
				base += 0.10
			case ContinentalRift:
				base -= 0.11
			case TransformFault:
				base += 0.02
			}

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
