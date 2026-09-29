package erosion

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
		name string
		fn   func(i int) color.RGBA
	}{
		{"slope.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{30, 54, 68, 255}
			}
			v := clamp(m.Slope[i]/m.Config.SlopeScale, 0, 1)
			return lerpColor(color.RGBA{65, 81, 62, 255}, color.RGBA{224, 201, 126, 255}, v)
		}},
		{"wetness.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{29, 57, 75, 255}
			}
			v := clamp(m.Wetness[i], 0, 1)
			return lerpColor(color.RGBA{154, 119, 77, 255}, color.RGBA{63, 150, 179, 255}, v)
		}},
		{"tectonic-support.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{30, 52, 66, 255}
			}
			v := clamp(m.TectonicSupport[i], 0, 1)
			return lerpColor(color.RGBA{75, 76, 70, 255}, color.RGBA{211, 102, 71, 255}, v)
		}},
		{"terrain-maturity.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{30, 52, 66, 255}
			}
			v := clamp(m.TerrainMaturity[i], 0, 1)
			return lerpColor(color.RGBA{100, 84, 75, 255}, color.RGBA{142, 172, 101, 255}, v)
		}},
		{"fluvial-power.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{29, 54, 70, 255}
			}
			v := clamp(m.FluvialPower[i], 0, 1)
			return lerpColor(color.RGBA{59, 61, 57, 255}, color.RGBA{65, 169, 219, 255}, v)
		}},
		{"valley-incision.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{29, 54, 70, 255}
			}
			v := clamp(m.ValleyIncision[i]/m.Config.MaxValleyIncision, 0, 1)
			return lerpColor(color.RGBA{62, 64, 59, 255}, color.RGBA{57, 148, 207, 255}, v)
		}},
		{"hillslope-erosion.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{29, 54, 70, 255}
			}
			v := clamp(m.HillslopeErosion[i]/m.Config.MaxHillslopeErosion, 0, 1)
			return lerpColor(color.RGBA{68, 66, 59, 255}, color.RGBA{205, 152, 76, 255}, v)
		}},
		{"total-erosion.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{29, 54, 70, 255}
			}
			v := clamp(m.TotalErosion[i]/(m.Config.MaxValleyIncision+m.Config.MaxHillslopeErosion), 0, 1)
			return lerpColor(color.RGBA{60, 62, 58, 255}, color.RGBA{220, 111, 75, 255}, v)
		}},
		{"eroded-elevation.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{30, 62, 82, 255}
			}
			v := clamp((m.ErodedElevation[i]-m.Config.SeaLevelNormalized)/(1-m.Config.SeaLevelNormalized), 0, 1)
			return elevationColor(v)
		}},
		{"relief.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{30, 56, 72, 255}
			}
			v := clamp(m.ReliefIndex[i], 0, 1)
			return lerpColor(color.RGBA{78, 118, 72, 255}, color.RGBA{226, 215, 193, 255}, v)
		}},
	}

	for _, r := range renderers {
		if err := renderScalar(filepath.Join(outDir, r.name), m.Width, m.Height, r.fn); err != nil {
			return err
		}
	}
	return nil
}

func renderScalar(path string, width, height int, fn func(i int) color.RGBA) error {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for z := 0; z < height; z++ {
		for x := 0; x < width; x++ {
			img.SetRGBA(x, z, fn(z*width+x))
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func elevationColor(v float64) color.RGBA {
	switch {
	case v <= 0.20:
		return lerpColor(color.RGBA{73, 121, 68, 255}, color.RGBA{113, 145, 77, 255}, v/0.20)
	case v <= 0.55:
		return lerpColor(color.RGBA{113, 145, 77, 255}, color.RGBA{176, 149, 96, 255}, (v-0.20)/0.35)
	default:
		return lerpColor(color.RGBA{176, 149, 96, 255}, color.RGBA{238, 235, 225, 255}, (v-0.55)/0.45)
	}
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
