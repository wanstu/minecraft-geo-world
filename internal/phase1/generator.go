package phase1

import (
	"math"
	"math/rand/v2"
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

	r := rand.New(rand.NewPCG(uint64(cfg.Seed), uint64(cfg.Seed)^0x9e3779b97f4a7c15))
	plates := make([]Plate, cfg.PlateCount)
	oceanCount := int(math.Round(float64(cfg.PlateCount) * cfg.OceanFraction))
	crustOrder := make([]CrustType, cfg.PlateCount)
	for i := range crustOrder {
		if i < oceanCount {
			crustOrder[i] = Oceanic
		} else {
			crustOrder[i] = Continental
		}
	}
	r.Shuffle(len(crustOrder), func(i, j int) {
		crustOrder[i], crustOrder[j] = crustOrder[j], crustOrder[i]
	})

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
			Crust: crustOrder[i],
		}
	}

	n := cfg.Width * cfg.Height
	w := &World{
		Config:    cfg,
		Plates:    plates,
		PlateID:   make([]int, n),
		Boundary:  make([]BoundaryType, n),
		Elevation: make([]float64, n),
	}

	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			w.PlateID[w.index(x, y)] = nearestPlate(float64(x), float64(y), plates)
		}
	}

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

	// closing > 0: 两个板块沿法线相向运动
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
			p := w.Plates[w.PlateID[i]]

			base := 0.68
			if p.Crust == Oceanic {
				base = 0.28
			}

			switch w.Boundary[i] {
			case Convergent:
				base += 0.24
			case Divergent:
				base -= 0.08
			case Transform:
				base += 0.03
			}

			// 仅用于打散完全平坦的板块内部，不决定宏观结构。
			n := math.Sin(float64(x)*0.071+float64(w.Config.Seed%997)) *
				math.Cos(float64(y)*0.083-float64(w.Config.Seed%313))
			base += n * 0.035

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
