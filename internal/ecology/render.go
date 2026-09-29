package ecology

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

	fields := []struct {
		name   string
		values []float64
		target color.RGBA
	}{
		{"moisture-index.png", m.MoistureIndex, color.RGBA{75, 159, 190, 255}},
		{"aridity-index.png", m.AridityIndex, color.RGBA{210, 156, 82, 255}},
		{"vegetation-density.png", m.VegetationDensity, color.RGBA{86, 163, 92, 255}},
		{"tree-cover.png", m.TreeCoverPotential, color.RGBA{49, 132, 76, 255}},
		{"grass-cover.png", m.GrassCoverPotential, color.RGBA{145, 176, 85, 255}},
		{"shrub-cover.png", m.ShrubCoverPotential, color.RGBA{176, 145, 91, 255}},
		{"bare-ground.png", m.BareGroundPotential, color.RGBA{196, 150, 102, 255}},
		{"snow-cover.png", m.SnowCoverPotential, color.RGBA{229, 236, 239, 255}},
		{"soil-moisture.png", m.SoilMoisture, color.RGBA{72, 139, 181, 255}},
		{"wetland-ecology.png", m.WetlandEcology, color.RGBA{69, 157, 137, 255}},
		{"alpine-ecology.png", m.AlpineEcology, color.RGBA{196, 191, 177, 255}},
		{"region-confidence.png", m.RegionConfidence, color.RGBA{188, 202, 213, 255}},
	}

	for _, f := range fields {
		if err := renderScalar(filepath.Join(outDir, f.name), m, f.values, f.target); err != nil {
			return err
		}
	}
	return renderRegions(filepath.Join(outDir, "eco-regions.png"), m)
}

func renderScalar(path string, m *Model, values []float64, target color.RGBA) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	baseLand := color.RGBA{52, 59, 55, 255}
	baseOcean := color.RGBA{28, 56, 73, 255}

	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			if !m.LandMask[i] {
				img.SetRGBA(x, z, baseOcean)
				continue
			}
			img.SetRGBA(x, z, lerpColor(baseLand, target, clamp(values[i], 0, 1)))
		}
	}
	return writePNG(path, img)
}

func renderRegions(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			img.SetRGBA(x, z, regionColor(m.EcoRegion[i]))
		}
	}
	return writePNG(path, img)
}

func regionColor(r Region) color.RGBA {
	switch r {
	case RegionWarmOcean:
		return color.RGBA{38, 120, 153, 255}
	case RegionTemperateOcean:
		return color.RGBA{39, 100, 139, 255}
	case RegionColdOcean:
		return color.RGBA{54, 89, 126, 255}
	case RegionFrozenOcean:
		return color.RGBA{160, 195, 207, 255}
	case RegionIceSheet:
		return color.RGBA{233, 236, 231, 255}
	case RegionTundra:
		return color.RGBA{160, 164, 128, 255}
	case RegionAlpine:
		return color.RGBA{178, 170, 153, 255}
	case RegionBorealForest:
		return color.RGBA{66, 112, 77, 255}
	case RegionTemperateForest:
		return color.RGBA{72, 135, 78, 255}
	case RegionTemperateGrassland:
		return color.RGBA{151, 162, 82, 255}
	case RegionTemperateDesert:
		return color.RGBA{188, 153, 102, 255}
	case RegionMediterraneanShrubland:
		return color.RGBA{154, 140, 83, 255}
	case RegionSubtropicalForest:
		return color.RGBA{64, 145, 79, 255}
	case RegionHotDesert:
		return color.RGBA{211, 166, 94, 255}
	case RegionSavanna:
		return color.RGBA{177, 172, 72, 255}
	case RegionTropicalMonsoonForest:
		return color.RGBA{51, 151, 75, 255}
	case RegionTropicalRainforest:
		return color.RGBA{39, 124, 70, 255}
	case RegionWetland:
		return color.RGBA{56, 145, 133, 255}
	default:
		return color.RGBA{70, 70, 70, 255}
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
