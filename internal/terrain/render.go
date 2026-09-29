package terrain

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type RenderConfig struct {
	WidthPx  int
	HeightPx int
}

func RenderAll(m *Model, outDir string, cfg RenderConfig) error {
	if cfg.WidthPx <= 0 {
		cfg.WidthPx = 960
	}
	if cfg.HeightPx <= 0 {
		cfg.HeightPx = 600
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	widthWorld := float64(m.BaseElevation.Width-1) * m.Config.CellSizeBlocks
	heightWorld := float64(m.BaseElevation.Height-1) * m.Config.CellSizeBlocks

	finalImg := image.NewRGBA(image.Rect(0, 0, cfg.WidthPx, cfg.HeightPx))
	detailImg := image.NewRGBA(image.Rect(0, 0, cfg.WidthPx, cfg.HeightPx))
	valleyImg := image.NewRGBA(image.Rect(0, 0, cfg.WidthPx, cfg.HeightPx))

	maxDetail := math.Max(
		m.Config.MountainAmplitude,
		math.Max(m.Config.FoothillAmplitude, m.Config.HillAmplitude),
	)

	for py := 0; py < cfg.HeightPx; py++ {
		z := float64(py) / float64(maxInt(cfg.HeightPx-1, 1)) * heightWorld
		for px := 0; px < cfg.WidthPx; px++ {
			x := float64(px) / float64(maxInt(cfg.WidthPx-1, 1)) * widthWorld
			s := m.Sample(x, z)

			finalImg.SetRGBA(px, py, elevationColor(s.FinalHeight))

			detailNorm := 0.5
			if maxDetail > 0 {
				detailNorm = clamp(0.5+s.RegionalDetail/(2*maxDetail), 0, 1)
			}
			detailImg.SetRGBA(px, py,
				lerpColor(color.RGBA{52, 91, 148, 255}, color.RGBA{218, 171, 85, 255}, detailNorm))

			valleyNorm := 0.0
			if m.Config.ValleyDepth > 0 {
				valleyNorm = clamp(s.ValleyCarve/m.Config.ValleyDepth, 0, 1)
			}
			valleyImg.SetRGBA(px, py,
				lerpColor(color.RGBA{55, 60, 57, 255}, color.RGBA{54, 153, 210, 255}, valleyNorm))
		}
	}

	if err := writePNG(filepath.Join(outDir, "regional-height.png"), finalImg); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(outDir, "regional-detail.png"), detailImg); err != nil {
		return err
	}
	return writePNG(filepath.Join(outDir, "valley-carve.png"), valleyImg)
}

func elevationColor(v float64) color.RGBA {
	switch {
	case v < 0.35:
		return lerpColor(color.RGBA{25, 57, 83, 255}, color.RGBA{45, 96, 126, 255}, v/0.35)
	case v < 0.50:
		return lerpColor(color.RGBA{45, 96, 126, 255}, color.RGBA{93, 145, 101, 255}, (v-0.35)/0.15)
	case v < 0.68:
		return lerpColor(color.RGBA{93, 145, 101, 255}, color.RGBA{171, 153, 98, 255}, (v-0.50)/0.18)
	case v < 0.84:
		return lerpColor(color.RGBA{171, 153, 98, 255}, color.RGBA{193, 179, 151, 255}, (v-0.68)/0.16)
	default:
		return lerpColor(color.RGBA{193, 179, 151, 255}, color.RGBA{239, 238, 232, 255}, (v-0.84)/0.16)
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
