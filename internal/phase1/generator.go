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
		Elevation:     make([]float64, n),
	}

	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			w.PlateID[w.index(x, y)] = nearestPlate(float64(x), float64(y), plates)
		}
	}

	w.generateCrust(r)
	w.classifyBoundaries()
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

			// 宏观扰动只改变大陆轮廓，不直接生成最终地形细节。
			potential += math.Sin(fx*0.031+float64(w.Config.Seed%401)) * 0.10
			potential += math.Cos(fy*0.037-float64(w.Config.Seed%271)) * 0.08
			potential += math.Sin((fx+fy)*0.019) * 0.05

			// 给主地理区外围留海洋缓冲，避免大陆直接顶到无尽海洋边界。
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
	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)
			a := w.PlateID[i]
			best := NoBoundary

			if x+1 < w.Config.Width {
				if b := w.PlateID[w.index(x+1, y)]; b != a {
					best = stronger(best, classify(w.Plates[a], w.Plates[b]))
				}
			}
			if y+1 < w.Config.Height {
				if b := w.PlateID[w.index(x, y+1)]; b != a {
					best = stronger(best, classify(w.Plates[a], w.Plates[b]))
				}
			}
			if x > 0 {
				if b := w.PlateID[w.index(x-1, y)]; b != a {
					best = stronger(best, classify(w.Plates[a], w.Plates[b]))
				}
			}
			if y > 0 {
				if b := w.PlateID[w.index(x, y-1)]; b != a {
					best = stronger(best, classify(w.Plates[a], w.Plates[b]))
				}
			}
			w.Boundary[i] = best
		}
	}
}

func classify(a, b Plate) BoundaryType {
	nx := b.Center.X - a.Center.X
	ny := b.Center.Y - a.Center.Y
	l := math.Hypot(nx, ny)
	if l == 0 {
		return Transform
	}
	nx /= l
	ny /= l

	// closing > 0: 两个板块沿边界法线相向运动。
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

func stronger(a, b BoundaryType) BoundaryType {
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

func (w *World) computeElevation() {
	for y := 0; y < w.Config.Height; y++ {
		for x := 0; x < w.Config.Width; x++ {
			i := w.index(x, y)

			base := 0.66
			if w.Crust[i] == Oceanic {
				base = 0.25
			}

			switch w.Boundary[i] {
			case Convergent:
				if w.Crust[i] == Continental {
					base += 0.27
				} else {
					base += 0.10
				}
			case Divergent:
				base -= 0.07
			case Transform:
				base += 0.025
			}

			// 仅用于打散完全平坦区域，不负责宏观大陆结构。
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
