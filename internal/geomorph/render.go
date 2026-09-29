package geomorph

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func RenderAll(m *Model, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	renderers := []struct {
		name   string
		field  []float64
		target color.RGBA
	}{
		{"slope-drop.png", m.SlopeDrop, color.RGBA{220, 157, 76, 255}},
		{"sediment-supply.png", m.SedimentSupply, color.RGBA{187, 127, 76, 255}},
		{"floodplain-weight.png", m.FloodplainWeight, color.RGBA{74, 163, 128, 255}},
		{"alluvial-plain-weight.png", m.AlluvialPlainWeight, color.RGBA{121, 177, 99, 255}},
		{"alluvial-fan-weight.png", m.AlluvialFanWeight, color.RGBA{212, 158, 87, 255}},
		{"canyon-weight.png", m.CanyonWeight, color.RGBA{180, 104, 76, 255}},
		{"valley-floor-width.png", m.ValleyFloorWidth, color.RGBA{78, 160, 208, 255}},
		{"deposition-potential.png", m.DepositionPotential, color.RGBA{190, 168, 100, 255}},
	}

	for _, r := range renderers {
		if err := renderField(filepath.Join(outDir, r.name), m, r.field, r.target); err != nil {
			return err
		}
	}
	return nil
}

func renderField(path string, m *Model, field []float64, target color.RGBA) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	base := color.RGBA{54, 60, 56, 255}
	ocean := color.RGBA{29, 56, 72, 255}

	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			if !m.LandMask[i] {
				img.SetRGBA(x, z, ocean)
				continue
			}
			img.SetRGBA(x, z, lerpColor(base, target, clamp(field[i], 0, 1)))
		}
	}

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
