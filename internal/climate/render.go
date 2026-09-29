package climate

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
		{"latitude.png", func(i int) color.RGBA {
			lat := m.LatitudeDeg[i]
			t := (lat + m.Config.MaxLatitudeDeg) / (2 * m.Config.MaxLatitudeDeg)
			return lerpColor(color.RGBA{70, 114, 205, 255}, color.RGBA{224, 111, 76, 255}, t)
		}},
		{"temperature-latitude.png", func(i int) color.RGBA {
			return temperatureColor(m.LatitudeBaseTempC[i])
		}},
		{"land-elevation.png", func(i int) color.RGBA {
			if m.LandElevationM[i] <= 0 {
				return color.RGBA{22, 49, 72, 255}
			}
			t := clamp(m.LandElevationM[i]/m.Config.MaxLandElevationM, 0, 1)
			return lerpColor(color.RGBA{78, 130, 73, 255}, color.RGBA{238, 237, 228, 255}, t)
		}},
		{"temperature-annual.png", func(i int) color.RGBA {
			return temperatureColor(m.AnnualMeanTempC[i])
		}},
		{"pressure.png", func(i int) color.RGBA {
			return pressureColor(m.PressureIndex[i])
		}},
		{"sst.png", func(i int) color.RGBA {
			if m.LandMask[i] {
				return color.RGBA{58, 64, 59, 255}
			}
			return temperatureColor(m.SeaSurfaceTempC[i])
		}},
		{"ocean-temp-correction.png", func(i int) color.RGBA {
			v := m.OceanTempCorrectionC[i]
			if math.Abs(v) < 0.02 {
				return color.RGBA{65, 68, 67, 255}
			}
			if v > 0 {
				return lerpColor(color.RGBA{65, 68, 67, 255}, color.RGBA{219, 91, 66, 255}, clamp(v/4, 0, 1))
			}
			return lerpColor(color.RGBA{65, 68, 67, 255}, color.RGBA{72, 126, 211, 255}, clamp(-v/4, 0, 1))
		}},
		{"temperature-adjusted.png", func(i int) color.RGBA {
			return temperatureColor(m.AdjustedAnnualTempC[i])
		}},
		{"moisture.png", func(i int) color.RGBA {
			v := clamp(m.Moisture[i]/1.2, 0, 1)
			return lerpColor(color.RGBA{126, 104, 76, 255}, color.RGBA{71, 168, 196, 255}, v)
		}},
		{"orographic-lift.png", func(i int) color.RGBA {
			v := clamp(m.OrographicLift[i], 0, 1)
			return lerpColor(color.RGBA{48, 51, 52, 255}, color.RGBA{229, 202, 109, 255}, v)
		}},
		{"precipitation.png", func(i int) color.RGBA {
			return precipitationColor(m.AnnualPrecipMM[i])
		}},
		{"seasonal-amplitude.png", func(i int) color.RGBA {
			v := clamp(m.SeasonalAmplitudeC[i]/24, 0, 1)
			return lerpColor(color.RGBA{75, 111, 158, 255}, color.RGBA{219, 133, 72, 255}, v)
		}},
		{"summer-temperature.png", func(i int) color.RGBA {
			return temperatureColor(m.SummerTempC[i])
		}},
		{"winter-temperature.png", func(i int) color.RGBA {
			return temperatureColor(m.WinterTempC[i])
		}},
		{"monsoon-index.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{35, 59, 72, 255}
			}
			v := clamp(m.MonsoonIndex[i], 0, 1)
			return lerpColor(color.RGBA{75, 73, 65, 255}, color.RGBA{79, 181, 204, 255}, v)
		}},
		{"warm-season-precip.png", func(i int) color.RGBA {
			return precipitationColor(m.WarmSeasonPrecipMM[i] * 2)
		}},
		{"cold-season-precip.png", func(i int) color.RGBA {
			return precipitationColor(m.ColdSeasonPrecipMM[i] * 2)
		}},
		{"rainfall-seasonality.png", func(i int) color.RGBA {
			v := clamp(m.RainfallSeasonality[i], 0, 1)
			return lerpColor(color.RGBA{83, 88, 82, 255}, color.RGBA{210, 151, 67, 255}, v)
		}},
		{"monsoon-rain-ratio.png", func(i int) color.RGBA {
			if !m.LandMask[i] {
				return color.RGBA{34, 54, 65, 255}
			}
			v := clamp(m.MonsoonRainRatio[i], 0, 1)
			return lerpColor(color.RGBA{65, 66, 60, 255}, color.RGBA{55, 155, 191, 255}, v)
		}},
	}

	for _, r := range renderers {
		if err := render(filepath.Join(outDir, r.name), m.Width, m.Height, r.fn); err != nil {
			return err
		}
	}
	if err := renderWind(filepath.Join(outDir, "wind.png"), m); err != nil {
		return err
	}
	if err := renderOceanCurrent(filepath.Join(outDir, "ocean-current.png"), m); err != nil {
		return err
	}
	if err := renderMonsoonWind(filepath.Join(outDir, "summer-monsoon-wind.png"), m, true); err != nil {
		return err
	}
	return renderMonsoonWind(filepath.Join(outDir, "winter-monsoon-wind.png"), m, false)
}

func temperatureColor(tempC float64) color.RGBA {
	switch {
	case tempC <= -20:
		return lerpColor(color.RGBA{238, 249, 255, 255}, color.RGBA{104, 154, 214, 255}, clamp((tempC+40)/20, 0, 1))
	case tempC <= 0:
		return lerpColor(color.RGBA{104, 154, 214, 255}, color.RGBA{91, 197, 200, 255}, (tempC+20)/20)
	case tempC <= 15:
		return lerpColor(color.RGBA{91, 197, 200, 255}, color.RGBA{126, 190, 91, 255}, tempC/15)
	case tempC <= 25:
		return lerpColor(color.RGBA{126, 190, 91, 255}, color.RGBA{236, 203, 79, 255}, (tempC-15)/10)
	default:
		return lerpColor(color.RGBA{236, 203, 79, 255}, color.RGBA{213, 73, 61, 255}, clamp((tempC-25)/15, 0, 1))
	}
}

func pressureColor(v float64) color.RGBA {
	v = clamp(v, 0, 1)
	if v < 0.5 {
		return lerpColor(color.RGBA{65, 111, 190, 255}, color.RGBA{116, 152, 177, 255}, v/0.5)
	}
	return lerpColor(color.RGBA{116, 152, 177, 255}, color.RGBA{199, 95, 74, 255}, (v-0.5)/0.5)
}

func render(path string, w, h int, fn func(i int) color.RGBA) error {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			img.SetRGBA(x, y, fn(i))
		}
	}
	return writePNG(path, img)
}

func renderWind(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			i := y*m.Width + x
			bg := pressureColor(m.PressureIndex[i])
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(float64(bg.R) * 0.48),
				G: uint8(float64(bg.G) * 0.48),
				B: uint8(float64(bg.B) * 0.48),
				A: 255,
			})
		}
	}

	step := maxInt(18, minInt(m.Width, m.Height)/16)
	arrow := color.RGBA{232, 243, 247, 235}
	for y := step / 2; y < m.Height; y += step {
		for x := step / 2; x < m.Width; x += step {
			i := y*m.Width + x
			wx, wz := m.WindX[i], m.WindZ[i]
			mag := math.Hypot(wx, wz)
			if mag < 1e-6 {
				continue
			}
			dx, dz := wx/mag, wz/mag
			length := float64(step) * 0.36 * clamp(mag, 0.35, 1)
			x1 := int(math.Round(float64(x) + dx*length))
			y1 := int(math.Round(float64(y) + dz*length))

			drawLine(img, x, y, x1, y1, arrow)

			angle := math.Atan2(dz, dx)
			headLen := math.Max(3, float64(step)*0.12)
			for _, offset := range []float64{0.55, -0.55} {
				hx := int(math.Round(float64(x1) - math.Cos(angle+offset)*headLen))
				hy := int(math.Round(float64(y1) - math.Sin(angle+offset)*headLen))
				drawLine(img, x1, y1, hx, hy, arrow)
			}
		}
	}

	return writePNG(path, img)
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

func renderOceanCurrent(path string, m *Model) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			i := y*m.Width + x
			if m.LandMask[i] {
				img.SetRGBA(x, y, color.RGBA{45, 49, 44, 255})
				continue
			}
			bg := temperatureColor(m.SeaSurfaceTempC[i])
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(float64(bg.R) * 0.58),
				G: uint8(float64(bg.G) * 0.58),
				B: uint8(float64(bg.B) * 0.58),
				A: 255,
			})
		}
	}

	step := maxInt(18, minInt(m.Width, m.Height)/16)
	arrow := color.RGBA{238, 246, 248, 238}
	for y := step / 2; y < m.Height; y += step {
		for x := step / 2; x < m.Width; x += step {
			i := y*m.Width + x
			if m.LandMask[i] {
				continue
			}
			cx, cz := m.CurrentX[i], m.CurrentZ[i]
			mag := math.Hypot(cx, cz)
			if mag < 1e-5 {
				continue
			}
			dx, dz := cx/mag, cz/mag
			length := float64(step) * 0.38 * clamp(mag, 0.25, 1)
			x1 := int(math.Round(float64(x) + dx*length))
			y1 := int(math.Round(float64(y) + dz*length))
			drawLine(img, x, y, x1, y1, arrow)

			angle := math.Atan2(dz, dx)
			headLen := math.Max(3, float64(step)*0.12)
			for _, offset := range []float64{0.55, -0.55} {
				hx := int(math.Round(float64(x1) - math.Cos(angle+offset)*headLen))
				hy := int(math.Round(float64(y1) - math.Sin(angle+offset)*headLen))
				drawLine(img, x1, y1, hx, hy, arrow)
			}
		}
	}
	return writePNG(path, img)
}

func precipitationColor(mm float64) color.RGBA {
	switch {
	case mm <= 250:
		return lerpColor(color.RGBA{170, 132, 82, 255}, color.RGBA{201, 179, 101, 255}, clamp(mm/250, 0, 1))
	case mm <= 750:
		return lerpColor(color.RGBA{201, 179, 101, 255}, color.RGBA{121, 171, 93, 255}, (mm-250)/500)
	case mm <= 1500:
		return lerpColor(color.RGBA{121, 171, 93, 255}, color.RGBA{65, 154, 113, 255}, (mm-750)/750)
	case mm <= 3000:
		return lerpColor(color.RGBA{65, 154, 113, 255}, color.RGBA{60, 133, 181, 255}, (mm-1500)/1500)
	default:
		return lerpColor(color.RGBA{60, 133, 181, 255}, color.RGBA{75, 77, 157, 255}, clamp((mm-3000)/3000, 0, 1))
	}
}

func renderMonsoonWind(path string, m *Model, summer bool) error {
	img := image.NewRGBA(image.Rect(0, 0, m.Width, m.Height))
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			i := y*m.Width + x
			if !m.LandMask[i] {
				img.SetRGBA(x, y, color.RGBA{30, 49, 61, 255})
				continue
			}
			v := clamp(m.MonsoonIndex[i], 0, 1)
			img.SetRGBA(x, y, lerpColor(
				color.RGBA{54, 55, 52, 255},
				color.RGBA{64, 137, 158, 255},
				v,
			))
		}
	}

	step := maxInt(18, minInt(m.Width, m.Height)/16)
	arrow := color.RGBA{240, 246, 247, 240}
	for y := step / 2; y < m.Height; y += step {
		for x := step / 2; x < m.Width; x += step {
			i := y*m.Width + x
			if !m.LandMask[i] || m.MonsoonIndex[i] < 0.08 {
				continue
			}

			wx, wz := m.SummerWindX[i], m.SummerWindZ[i]
			if !summer {
				wx, wz = m.WinterWindX[i], m.WinterWindZ[i]
			}
			mag := math.Hypot(wx, wz)
			if mag < 1e-6 {
				continue
			}
			dx, dz := wx/mag, wz/mag
			length := float64(step) * 0.36 * clamp(mag, 0.3, 1.4)
			x1 := int(math.Round(float64(x) + dx*length))
			y1 := int(math.Round(float64(y) + dz*length))
			drawLine(img, x, y, x1, y1, arrow)

			angle := math.Atan2(dz, dx)
			headLen := math.Max(3, float64(step)*0.12)
			for _, offset := range []float64{0.55, -0.55} {
				hx := int(math.Round(float64(x1) - math.Cos(angle+offset)*headLen))
				hy := int(math.Round(float64(y1) - math.Sin(angle+offset)*headLen))
				drawLine(img, x1, y1, hx, hy, arrow)
			}
		}
	}

	return writePNG(path, img)
}
