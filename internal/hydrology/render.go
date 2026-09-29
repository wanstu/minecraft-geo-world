package hydrology

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func RenderAll(m *Model, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	renderers := []struct {
		name string
		fn   func(i int) color.RGBA
	}{
		{"filled-elevation.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{31, 62, 80, 255}
			}
			v := clamp((m.FilledElevation[i]-m.Config.SeaLevelNormalized)/(1-m.Config.SeaLevelNormalized), 0, 1)
			return lerpColor(color.RGBA{70, 114, 68, 255}, color.RGBA{229, 226, 211, 255}, v)
		}},
		{"fill-depth.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{28, 52, 68, 255}
			}
			v := clamp(m.FillDepth[i]/0.08, 0, 1)
			return lerpColor(color.RGBA{55, 59, 54, 255}, color.RGBA{68, 158, 211, 255}, v)
		}},
		{"runoff.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{30, 55, 69, 255}
			}
			v := clamp(m.LocalRunoffMM[i]/1200, 0, 1)
			return lerpColor(color.RGBA{146, 117, 77, 255}, color.RGBA{54, 148, 187, 255}, v)
		}},
		{"flow-accumulation.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{28, 50, 65, 255}
			}
			v := 0.0
			if m.FlowAccumulation[i] > 0 {
				v = clamp(math.Log1p(m.FlowAccumulation[i])/math.Log1p(math.Max(m.RiverThreshold*50, 1)), 0, 1)
			}
			return lerpColor(color.RGBA{67, 65, 59, 255}, color.RGBA{62, 166, 212, 255}, v)
		}},
		{"basins.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{27, 52, 68, 255}
			}
			return basinColor(m.BasinID[i])
		}},
		{"lake-potential.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{27, 51, 67, 255}
			}
			v := clamp(m.LakePotential[i], 0, 1)
			return lerpColor(color.RGBA{60, 63, 58, 255}, color.RGBA{72, 184, 219, 255}, v)
		}},
		{"wetland-potential.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{27, 51, 67, 255}
			}
			v := clamp(m.WetlandPotential[i], 0, 1)
			return lerpColor(color.RGBA{75, 71, 57, 255}, color.RGBA{79, 159, 113, 255}, v)
		}},
	}

	for _, r := range renderers {
		if err := renderScalar(filepath.Join(outDir, r.name), m.Width, m.Height, r.fn); err != nil {
			return err
		}
	}
	if err := renderFlow(filepath.Join(outDir, "flow-direction.png"), m); err != nil {
		return err
	}
	return renderRivers(filepath.Join(outDir, "rivers.png"), m)
}

func renderScalar(path string, width, height int, fn func(i int) color.RGBA) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for z := 0; z < height; z++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, z, fn(z*width+x))
		}
	}
	return writePNG(path, img)
}

func renderFlow(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			if !m.LandMask[i] {
				img.SetRGBA(x, z, color.RGBA{28, 52, 68, 255})
			} else {
				img.SetRGBA(x, z, color.RGBA{53, 60, 55, 255})
			}
		}
	}

	step := maxInt(12, minInt(m.Width, m.Height)/22)
	arrow := color.RGBA{221, 232, 225, 220}
	for z := step / 2; z < m.Height; z += step {
		for x := step / 2; x < m.Width; x += step {
			i := z*m.Width + x
			if !m.LandMask[i] {
				continue
			}
			to := m.FlowTo[i]
			if to < 0 || to >= len(m.FlowTo) {
				continue
			}
			tx, tz := to%m.Width, to/m.Width
			dx := tx - x
			dz := tz - z
			if dx == 0 && dz == 0 {
				continue
			}
			length := float64(step) * 0.32
			mag := math.Hypot(float64(dx), float64(dz))
			ex := int(math.Round(float64(x) + float64(dx)/mag*length))
			ez := int(math.Round(float64(z) + float64(dz)/mag*length))
			drawLine(img, x, z, ex, ez, arrow)
		}
	}
	return writePNG(path, img)
}

func renderRivers(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			if !m.LandMask[i] {
				img.SetRGBA(x, z, color.RGBA{30, 62, 82, 255})
				continue
			}
			e := clamp((m.OriginalElevation[i]-m.Config.SeaLevelNormalized)/(1-m.Config.SeaLevelNormalized), 0, 1)
			base := lerpColor(color.RGBA{62, 91, 60, 255}, color.RGBA{176, 165, 129, 255}, e)
			if m.WetlandPotential[i] > 0.45 {
				base = lerpColor(base, color.RGBA{67, 132, 92, 255}, m.WetlandPotential[i]*0.55)
			}
			if m.LakePotential[i] > 0.35 {
				base = lerpColor(base, color.RGBA{67, 155, 204, 255}, m.LakePotential[i])
			}
			img.SetRGBA(x, z, base)
		}
	}

	river := color.RGBA{75, 173, 225, 255}
	for i, strength := range m.RiverStrength {
		if strength <= 0 || !m.LandMask[i] {
			continue
		}
		x, z := i%m.Width, i/m.Width
		to := m.FlowTo[i]
		if to < 0 || to >= len(m.FlowTo) {
			continue
		}
		tx, tz := to%m.Width, to/m.Width
		drawLine(img, x, z, tx, tz, river)
		if strength > 0.55 {
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, nz := x+d[0], z+d[1]
				if nx >= 0 && nx < m.Width && nz >= 0 && nz < m.Height {
					img.SetRGBA(nx, nz, river)
				}
			}
		}
	}

	return writePNG(path, img)
}

func basinColor(id int) color.RGBA {
	if id <= 0 {
		return color.RGBA{55, 58, 55, 255}
	}
	x := uint32(id) * 2654435761
	return color.RGBA{
		R: uint8(72 + (x & 0x7f)),
		G: uint8(72 + ((x >> 8) & 0x7f)),
		B: uint8(72 + ((x >> 16) & 0x7f)),
		A: 255,
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := absInt(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -absInt(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy

	for {
		if image.Pt(x0, y0).In(img.Bounds()) {
			img.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	t = clamp(t, 0, 1)
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
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

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
