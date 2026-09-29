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

	renderers := []struct {
		name string
		fn   func(x, y int) color.RGBA
	}{
		{"plates-raw.png", func(x, y int) color.RGBA {
			return palette(w.RawPlateID[w.index(x, y)])
		}},
		{"plates.png", func(x, y int) color.RGBA {
			return palette(w.PlateID[w.index(x, y)])
		}},
		{"crust.png", func(x, y int) color.RGBA {
			if w.Crust[w.index(x, y)] == Continental {
				return color.RGBA{92, 130, 78, 255}
			}
			return color.RGBA{31, 76, 99, 255}
		}},
		{"coast-distance.png", func(x, y int) color.RGBA {
			i := w.index(x, y)
			d := w.CoastDistance[i]
			if d >= 0 {
				t := clamp(d/36, 0, 1)
				return lerpColor(color.RGBA{204, 219, 166, 255}, color.RGBA{69, 104, 58, 255}, t)
			}
			t := clamp((-d)/42, 0, 1)
			return lerpColor(color.RGBA{105, 196, 210, 255}, color.RGBA{15, 42, 86, 255}, t)
		}},
		{"crust-thickness.png", func(x, y int) color.RGBA {
			i := w.index(x, y)
			if w.Crust[i] == Oceanic {
				return color.RGBA{24, 52, 69, 255}
			}
			t := clamp((w.CrustThickness[i]-0.44)/(0.88-0.44), 0, 1)
			return lerpColor(color.RGBA{110, 118, 77, 255}, color.RGBA{231, 204, 128, 255}, t)
		}},
		{"base-elevation.png", func(x, y int) color.RGBA {
			return elevationColor(w.BaseElevation[w.index(x, y)])
		}},
		{"terrain-zones.png", func(x, y int) color.RGBA {
			return terrainColor(w.TerrainZone[w.index(x, y)])
		}},
		{"boundaries.png", func(x, y int) color.RGBA {
			i := w.index(x, y)
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
		}},
		{"tectonics.png", func(x, y int) color.RGBA {
			i := w.index(x, y)
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
		}},
		{"features.png", func(x, y int) color.RGBA {
			i := w.index(x, y)
			return featureColor(w.Feature[i], w.Crust[i])
		}},
		{"ocean-age.png", func(x, y int) color.RGBA {
			i := w.index(x, y)
			if w.Crust[i] == Continental {
				return color.RGBA{58, 73, 55, 255}
			}
			age := clamp(w.OceanAge[i], 0, 1)
			return lerpColor(
				color.RGBA{78, 211, 226, 255},
				color.RGBA{16, 31, 83, 255},
				age,
			)
		}},
		{"subduction.png", func(x, y int) color.RGBA {
			i := w.index(x, y)
			switch w.Feature[i] {
			case Trench:
				return color.RGBA{32, 76, 173, 255}
			case VolcanicArc:
				return color.RGBA{245, 135, 54, 255}
			case IslandArc:
				return color.RGBA{226, 87, 147, 255}
			default:
				if w.Crust[i] == Continental {
					return color.RGBA{38, 52, 39, 255}
				}
				return color.RGBA{13, 31, 48, 255}
			}
		}},
		{"tectonic-intensity.png", func(x, y int) color.RGBA {
			i := w.index(x, y)
			v := clamp(w.TectonicIntensity[i], 0, 1)
			if w.Feature[i] == NoFeature {
				if w.Crust[i] == Continental {
					return color.RGBA{39, 48, 39, 255}
				}
				return color.RGBA{18, 34, 47, 255}
			}
			return lerpColor(color.RGBA{74, 78, 84, 255}, color.RGBA{244, 93, 61, 255}, v)
		}},
		{"tectonic-influence.png", func(x, y int) color.RGBA {
			v := w.TectonicInfluence[w.index(x, y)]
			if v >= 0 {
				t := clamp(v/0.45, 0, 1)
				return lerpColor(color.RGBA{40, 45, 51, 255}, color.RGBA{237, 111, 74, 255}, t)
			}
			t := clamp(-v/0.38, 0, 1)
			return lerpColor(color.RGBA{40, 45, 51, 255}, color.RGBA{55, 114, 214, 255}, t)
		}},
		{"elevation.png", func(x, y int) color.RGBA {
			return elevationColor(w.Elevation[w.index(x, y)])
		}},
	}

	for _, r := range renderers {
		if err := render(filepath.Join(outDir, r.name), w.Config.Width, w.Config.Height, r.fn); err != nil {
			return err
		}
	}
	return nil
}

func featureColor(feature TectonicFeature, crust CrustType) color.RGBA {
	switch feature {
	case CollisionOrogen:
		return color.RGBA{239, 83, 80, 255}
	case VolcanicArc:
		return color.RGBA{245, 142, 66, 255}
	case Trench:
		return color.RGBA{46, 72, 128, 255}
	case IslandArc:
		return color.RGBA{220, 88, 149, 255}
	case MidOceanRidge:
		return color.RGBA{67, 185, 214, 255}
	case ContinentalRift:
		return color.RGBA{224, 188, 70, 255}
	case TransformFault:
		return color.RGBA{202, 202, 202, 255}
	default:
		if crust == Continental {
			return color.RGBA{47, 67, 48, 255}
		}
		return color.RGBA{18, 42, 56, 255}
	}
}

func terrainColor(t MacroTerrain) color.RGBA {
	switch t {
	case ContinentalInterior:
		return color.RGBA{95, 139, 76, 255}
	case ContinentalCoast:
		return color.RGBA{151, 166, 91, 255}
	case ContinentalShelf:
		return color.RGBA{72, 158, 176, 255}
	case ContinentalSlope:
		return color.RGBA{38, 102, 145, 255}
	case OceanBasin:
		return color.RGBA{20, 55, 105, 255}
	case MountainBelt:
		return color.RGBA{163, 128, 82, 255}
	case Plateau:
		return color.RGBA{139, 146, 82, 255}
	case TrenchZone:
		return color.RGBA{32, 36, 92, 255}
	case IslandArcZone:
		return color.RGBA{204, 92, 139, 255}
	case MidOceanRidgeZone:
		return color.RGBA{75, 193, 213, 255}
	case RiftValleyZone:
		return color.RGBA{206, 171, 70, 255}
	default:
		return color.RGBA{60, 60, 60, 255}
	}
}

func elevationColor(v float64) color.RGBA {
	v = clamp(v, 0, 1)
	if v < 0.5 {
		t := clamp(v/0.5, 0, 1)
		return lerpColor(color.RGBA{9, 29, 71, 255}, color.RGBA{74, 167, 191, 255}, t)
	}
	t := clamp((v-0.5)/0.5, 0, 1)
	if t < 0.65 {
		return lerpColor(color.RGBA{77, 128, 70, 255}, color.RGBA{177, 159, 99, 255}, t/0.65)
	}
	return lerpColor(color.RGBA{177, 159, 99, 255}, color.RGBA{239, 239, 231, 255}, (t-0.65)/0.35)
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

func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	t = clamp(t, 0, 1)
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}
