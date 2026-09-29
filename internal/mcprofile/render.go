package mcprofile

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
	if err := renderBiomeFamilies(filepath.Join(outDir, "biome-family.png"), m); err != nil {
		return err
	}
	if err := renderTerrainVariants(filepath.Join(outDir, "terrain-variant.png"), m); err != nil {
		return err
	}
	if err := renderSurfaceProfiles(filepath.Join(outDir, "surface-profile.png"), m); err != nil {
		return err
	}
	if err := renderVegetationProfiles(filepath.Join(outDir, "vegetation-profile.png"), m); err != nil {
		return err
	}
	return renderConfidence(filepath.Join(outDir, "profile-confidence.png"), m)
}

func renderBiomeFamilies(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for i, v := range m.BiomeFamily {
		img.SetRGBA(i%m.Width, i/m.Width, biomeColor(v))
	}
	return writePNG(path, img)
}

func renderTerrainVariants(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for i, v := range m.TerrainVariant {
		img.SetRGBA(i%m.Width, i/m.Width, terrainColor(v))
	}
	return writePNG(path, img)
}

func renderSurfaceProfiles(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for i, v := range m.SurfaceProfile {
		img.SetRGBA(i%m.Width, i/m.Width, surfaceColor(v))
	}
	return writePNG(path, img)
}

func renderVegetationProfiles(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for i, v := range m.VegetationProfile {
		img.SetRGBA(i%m.Width, i/m.Width, vegetationColor(v))
	}
	return writePNG(path, img)
}

func renderConfidence(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	lo := color.RGBA{73, 66, 75, 255}
	hi := color.RGBA{200, 216, 216, 255}
	for i, v := range m.ProfileConfidence {
		img.SetRGBA(i%m.Width, i/m.Width, lerpColor(lo, hi, clamp(v, 0, 1)))
	}
	return writePNG(path, img)
}

func biomeColor(v BiomeFamily) color.RGBA {
	switch v {
	case BiomeWarmOcean:
		return color.RGBA{38, 120, 153, 255}
	case BiomeTemperateOcean:
		return color.RGBA{39, 100, 139, 255}
	case BiomeColdOcean:
		return color.RGBA{54, 89, 126, 255}
	case BiomeFrozenOcean:
		return color.RGBA{160, 195, 207, 255}
	case BiomeIceSheet:
		return color.RGBA{233, 236, 231, 255}
	case BiomeTundra:
		return color.RGBA{160, 164, 128, 255}
	case BiomeAlpine:
		return color.RGBA{178, 170, 153, 255}
	case BiomeBorealForest:
		return color.RGBA{66, 112, 77, 255}
	case BiomeTemperateForest:
		return color.RGBA{72, 135, 78, 255}
	case BiomeTemperateGrassland:
		return color.RGBA{151, 162, 82, 255}
	case BiomeTemperateDesert:
		return color.RGBA{188, 153, 102, 255}
	case BiomeMediterraneanShrubland:
		return color.RGBA{154, 140, 83, 255}
	case BiomeSubtropicalForest:
		return color.RGBA{64, 145, 79, 255}
	case BiomeHotDesert:
		return color.RGBA{211, 166, 94, 255}
	case BiomeSavanna:
		return color.RGBA{177, 172, 72, 255}
	case BiomeTropicalMonsoonForest:
		return color.RGBA{51, 151, 75, 255}
	case BiomeTropicalRainforest:
		return color.RGBA{39, 124, 70, 255}
	case BiomeWetland:
		return color.RGBA{56, 145, 133, 255}
	default:
		return color.RGBA{70, 70, 70, 255}
	}
}

func terrainColor(v TerrainVariant) color.RGBA {
	switch v {
	case TerrainOcean:
		return color.RGBA{31, 73, 96, 255}
	case TerrainPlain:
		return color.RGBA{114, 156, 89, 255}
	case TerrainRollingPlain:
		return color.RGBA{126, 163, 86, 255}
	case TerrainHills:
		return color.RGBA{153, 155, 82, 255}
	case TerrainFoothills:
		return color.RGBA{177, 137, 83, 255}
	case TerrainMountain:
		return color.RGBA{177, 171, 160, 255}
	case TerrainPlateau:
		return color.RGBA{170, 111, 80, 255}
	case TerrainFloodplain:
		return color.RGBA{80, 153, 119, 255}
	case TerrainAlluvialPlain:
		return color.RGBA{104, 169, 112, 255}
	case TerrainAlluvialFan:
		return color.RGBA{193, 147, 88, 255}
	case TerrainCanyon:
		return color.RGBA{158, 91, 73, 255}
	default:
		return color.RGBA{70, 70, 70, 255}
	}
}

func surfaceColor(v SurfaceProfile) color.RGBA {
	switch v {
	case SurfaceOceanSand:
		return color.RGBA{124, 123, 103, 255}
	case SurfaceOceanGravel:
		return color.RGBA{92, 102, 103, 255}
	case SurfaceFrozenOcean:
		return color.RGBA{180, 211, 218, 255}
	case SurfaceIceSnow:
		return color.RGBA{233, 238, 239, 255}
	case SurfaceTundraSoil:
		return color.RGBA{136, 129, 101, 255}
	case SurfaceAlpineRock:
		return color.RGBA{123, 119, 112, 255}
	case SurfaceAlpineSnow:
		return color.RGBA{219, 224, 224, 255}
	case SurfaceForestLoam:
		return color.RGBA{94, 111, 69, 255}
	case SurfacePodzol:
		return color.RGBA{94, 78, 56, 255}
	case SurfaceGrassLoam:
		return color.RGBA{125, 139, 71, 255}
	case SurfaceDryGrass:
		return color.RGBA{151, 132, 73, 255}
	case SurfaceShrubSoil:
		return color.RGBA{145, 116, 78, 255}
	case SurfaceDesertSand:
		return color.RGBA{203, 171, 102, 255}
	case SurfaceDesertGravel:
		return color.RGBA{149, 128, 99, 255}
	case SurfaceAlluvialSilt:
		return color.RGBA{113, 109, 80, 255}
	case SurfaceWetMud:
		return color.RGBA{76, 90, 73, 255}
	case SurfacePeat:
		return color.RGBA{64, 70, 55, 255}
	default:
		return color.RGBA{70, 70, 70, 255}
	}
}

func vegetationColor(v VegetationProfile) color.RGBA {
	switch v {
	case VegetationNone:
		return color.RGBA{99, 102, 99, 255}
	case VegetationTundraSparse:
		return color.RGBA{135, 142, 111, 255}
	case VegetationAlpineSparse:
		return color.RGBA{143, 143, 119, 255}
	case VegetationConiferForest:
		return color.RGBA{54, 103, 69, 255}
	case VegetationMixedForest:
		return color.RGBA{66, 126, 72, 255}
	case VegetationBroadleafForest:
		return color.RGBA{55, 139, 70, 255}
	case VegetationDenseRainforest:
		return color.RGBA{30, 111, 57, 255}
	case VegetationMonsoonForest:
		return color.RGBA{47, 139, 68, 255}
	case VegetationGrassland:
		return color.RGBA{142, 163, 76, 255}
	case VegetationDryGrassland:
		return color.RGBA{168, 150, 70, 255}
	case VegetationShrubland:
		return color.RGBA{150, 131, 78, 255}
	case VegetationSavanna:
		return color.RGBA{170, 163, 70, 255}
	case VegetationDesertSparse:
		return color.RGBA{180, 151, 92, 255}
	case VegetationWetland:
		return color.RGBA{54, 137, 111, 255}
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
