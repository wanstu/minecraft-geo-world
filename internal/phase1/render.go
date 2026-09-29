package phase1

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
)

func RenderAll(w *World, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	if err := render(filepath.Join(outDir, "plates.png"), w.Config.Width, w.Config.Height, func(x, y int) color.RGBA {
		id := w.PlateID[y*w.Config.Width+x]
		return palette(id)
	}); err != nil {
		return err
	}
	if err := render(filepath.Join(outDir, "crust.png"), w.Config.Width, w.Config.Height, func(x, y int) color.RGBA {
		if w.Crust[y*w.Config.Width+x] == Continental {
			return color.RGBA{92, 130, 78, 255}
		}
		return color.RGBA{31, 76, 99, 255}
	}); err != nil {
		return err
	}
	if err := render(filepath.Join(outDir, "boundaries.png"), w.Config.Width, w.Config.Height, func(x, y int) color.RGBA {
		i := y*w.Config.Width + x
		base := palette(w.PlateID[i])
		switch w.Boundary[i] {
		case Convergent:
			return color.RGBA{232, 92, 77, 255}
		case Divergent:
			return color.RGBA{84, 173, 226, 255}
		case Transform:
			return color.RGBA{235, 193, 82, 255}
		default:
			return dim(base)
		}
	}); err != nil {
		return err
	}
	if err := render(filepath.Join(outDir, "tectonics.png"), w.Config.Width, w.Config.Height, func(x, y int) color.RGBA {
		i := y*w.Config.Width + x
		base := color.RGBA{33, 75, 98, 255}
		if w.Crust[i] == Continental {
			base = color.RGBA{83, 123, 75, 255}
		}
		switch w.Boundary[i] {
		case Convergent:
			return color.RGBA{232, 92, 77, 255}
		case Divergent:
			return color.RGBA{84, 173, 226, 255}
		case Transform:
			return color.RGBA{235, 193, 82, 255}
		default:
			return base
		}
	}); err != nil {
		return err
	}
	return render(filepath.Join(outDir, "elevation.png"), w.Config.Width, w.Config.Height, func(x, y int) color.RGBA {
		v := w.Elevation[y*w.Config.Width+x]
		if v < 0.5 {
			t := uint8(40 + v*170)
			return color.RGBA{18, 58 + t/3, 92 + t/2, 255}
		}
		t := (v - 0.5) * 2
		return color.RGBA{
			uint8(70 + t*155),
			uint8(112 + t*120),
			uint8(67 + t*155),
			255,
		}
	})
}

func render(path string, w, h int, fn func(x, y int) color.RGBA) error {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, fn(x, y))
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func palette(id int) color.RGBA {
	colors := []color.RGBA{
		{90, 137, 167, 255}, {116, 158, 105, 255}, {170, 119, 91, 255},
		{144, 120, 170, 255}, {187, 155, 84, 255}, {86, 156, 148, 255},
		{165, 100, 127, 255}, {110, 127, 179, 255}, {125, 153, 79, 255},
		{181, 127, 72, 255}, {91, 143, 126, 255}, {154, 109, 159, 255},
	}
	return colors[id%len(colors)]
}

func dim(c color.RGBA) color.RGBA {
	return color.RGBA{c.R / 2, c.G / 2, c.B / 2, 255}
}
