package landform

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
		{"mountain-core.png", func(i int) color.RGBA {
			return weightColor(m, i, m.MountainCore[i], color.RGBA{218, 205, 190, 255})
		}},
		{"mountain-influence.png", func(i int) color.RGBA {
			return weightColor(m, i, m.MountainInfluence[i], color.RGBA{201, 145, 90, 255})
		}},
		{"rolling-plain-weight.png", func(i int) color.RGBA {
			return weightColor(m, i, m.RollingPlainWeight[i], color.RGBA{104, 166, 88, 255})
		}},
		{"hill-weight.png", func(i int) color.RGBA {
			return weightColor(m, i, m.HillWeight[i], color.RGBA{151, 161, 87, 255})
		}},
		{"foothill-weight.png", func(i int) color.RGBA {
			return weightColor(m, i, m.FoothillWeight[i], color.RGBA{178, 142, 83, 255})
		}},
		{"mountain-weight.png", func(i int) color.RGBA {
			return weightColor(m, i, m.MountainWeight[i], color.RGBA{208, 199, 184, 255})
		}},
		{"plateau-weight.png", func(i int) color.RGBA {
			return weightColor(m, i, m.PlateauWeight[i], color.RGBA{187, 116, 81, 255})
		}},
		{"valley-weight.png", func(i int) color.RGBA {
			return weightColor(m, i, m.ValleyWeight[i], color.RGBA{62, 151, 210, 255})
		}},
	}

	for _, r := range renderers {
		if err := renderScalar(filepath.Join(outDir, r.name), m.Width, m.Height, r.fn); err != nil {
			return err
		}
	}
	return renderDominant(filepath.Join(outDir, "dominant-landform.png"), m)
}

func weightColor(m *Model, i int, v float64, target color.RGBA) color.RGBA {
	if !m.LandMask[i] {
		return color.RGBA{29, 55, 70, 255}
	}
	return lerpColor(color.RGBA{55, 59, 55, 255}, target, clamp(v, 0, 1))
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

func renderDominant(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	colors := []color.RGBA{
		{102, 166, 88, 255},  // rolling plain
		{151, 161, 87, 255},  // hill
		{181, 143, 83, 255},  // foothill
		{210, 201, 186, 255}, // mountain
		{184, 113, 80, 255},  // plateau
	}
	for i := range m.LandMask {
		x, z := i%m.Width, i/m.Width
		if !m.LandMask[i] {
			img.SetRGBA(x, z, color.RGBA{29, 55, 70, 255})
			continue
		}
		weights := []float64{
			m.RollingPlainWeight[i],
			m.HillWeight[i],
			m.FoothillWeight[i],
			m.MountainWeight[i],
			m.PlateauWeight[i],
		}
		best := 0
		for j := 1; j < len(weights); j++ {
			if weights[j] > weights[best] {
				best = j
			}
		}
		c := colors[best]
		if m.ValleyWeight[i] > 0.45 {
			c = lerpColor(c, color.RGBA{61, 151, 210, 255}, clamp((m.ValleyWeight[i]-0.45)/0.55, 0, 0.75))
		}
		img.SetRGBA(x, z, c)
	}
	return writePNG(path, img)
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
	if math.IsNaN(t) {
		t = 0
	}
	t = clamp(t, 0, 1)
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}
