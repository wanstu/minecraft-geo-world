package erosion

import (
	"errors"
	"math"

	"geoworld/internal/world"
)

func Generate(input Input, cfg Config) (*Model, error) {
	if input.Width < 2 || input.Height < 2 {
		return nil, errors.New("erosion: grid must be at least 2x2")
	}
	n := input.Width * input.Height
	for _, field := range []struct {
		name string
		data []float64
	}{
		{"elevation", input.Elevation},
		{"crust thickness", input.CrustThickness},
		{"tectonic intensity", input.TectonicIntensity},
		{"tectonic influence", input.TectonicInfluence},
		{"annual precipitation", input.AnnualPrecipMM},
		{"flow accumulation", input.FlowAccumulation},
		{"river strength", input.RiverStrength},
	} {
		if len(field.data) != n {
			return nil, errors.New("erosion: " + field.name + " size does not match dimensions")
		}
	}
	if len(input.LandMask) != n || len(input.LakeID) != n {
		return nil, errors.New("erosion: mask field sizes do not match dimensions")
	}

	cfg = normalizeConfig(cfg, input.Width, input.Height)
	m := &Model{
		Config:            cfg,
		Width:             input.Width,
		Height:            input.Height,
		LandMask:          append([]bool(nil), input.LandMask...),
		OriginalElevation: append([]float64(nil), input.Elevation...),
		Slope:             make([]float64, n),
		Wetness:           make([]float64, n),
		TectonicSupport:   make([]float64, n),
		TerrainMaturity:   make([]float64, n),
		ErosionResistance: make([]float64, n),
		FluvialPower:      make([]float64, n),
		ValleyIncision:    make([]float64, n),
		HillslopeErosion:  make([]float64, n),
		TotalErosion:      make([]float64, n),
		ErodedElevation:   append([]float64(nil), input.Elevation...),
		LocalRelief:       make([]float64, n),
		ReliefIndex:       make([]float64, n),
	}

	m.computeDrivers(input)
	m.erode(input)
	m.computeRelief()

	return m, nil
}

func normalizeConfig(cfg Config, width, height int) Config {
	if cfg.SeaLevelNormalized <= 0 || cfg.SeaLevelNormalized >= 1 {
		cfg.SeaLevelNormalized = world.DefaultSeaLevelNormalized
	}
	if cfg.ReferencePrecipMM <= 0 {
		cfg.ReferencePrecipMM = 1800
	}
	if cfg.SlopeScale <= 0 {
		cfg.SlopeScale = 0.06
	}
	if cfg.MaxValleyIncision <= 0 || cfg.MaxValleyIncision > 0.20 {
		cfg.MaxValleyIncision = 0.055
	}
	if cfg.MaxHillslopeErosion <= 0 || cfg.MaxHillslopeErosion > 0.10 {
		cfg.MaxHillslopeErosion = 0.022
	}
	if cfg.TectonicSupportStrength <= 0 || cfg.TectonicSupportStrength > 1 {
		cfg.TectonicSupportStrength = 0.65
	}
	if cfg.ReliefRadiusCells <= 0 || cfg.ReliefRadiusCells > 32 {
		minDim := width
		if height < minDim {
			minDim = height
		}
		cfg.ReliefRadiusCells = int(math.Round(float64(minDim) * 0.01))
		if cfg.ReliefRadiusCells < 1 {
			cfg.ReliefRadiusCells = 1
		}
		if cfg.ReliefRadiusCells > 32 {
			cfg.ReliefRadiusCells = 32
		}
	}
	if cfg.ReliefScale <= 0 {
		cfg.ReliefScale = 0.16
	}
	return cfg
}

func (m *Model) computeDrivers(input Input) {
	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			if !m.LandMask[i] {
				continue
			}

			slope := localSlope(input.Elevation, input.LandMask, m.Width, m.Height, x, z)
			m.Slope[i] = slope

			wetness := clamp(input.AnnualPrecipMM[i]/m.Config.ReferencePrecipMM, 0, 1.25)
			m.Wetness[i] = wetness

			support := clamp(math.Max(input.TectonicIntensity[i], math.Abs(input.TectonicInfluence[i])), 0, 1)
			m.TectonicSupport[i] = support
			m.TerrainMaturity[i] = clamp(1-support, 0, 1)

			// This is intentionally a coarse erosion-resistance proxy, not a
			// lithology simulator. Thicker crust and active tectonic support make
			// large-scale relief harder to erase.
			resistance := 0.20 +
				0.42*clamp(input.CrustThickness[i], 0, 1) +
				0.30*support
			m.ErosionResistance[i] = clamp(resistance, 0.15, 0.95)

			slopeNorm := clamp(slope/m.Config.SlopeScale, 0, 1.5)
			accumNorm := normalizedAccumulation(input.FlowAccumulation[i], input.RiverThreshold)
			river := clamp(input.RiverStrength[i], 0, 1)

			m.FluvialPower[i] = clamp(
				river*
					(0.30+0.70*accumNorm)*
					(0.35+0.65*clamp(wetness, 0, 1))*
					(0.25+0.75*clamp(slopeNorm, 0, 1)),
				0,
				1,
			)
		}
	}
}

func (m *Model) erode(input Input) {
	for i := range m.ErodedElevation {
		if !m.LandMask[i] {
			continue
		}

		slopeNorm := clamp(m.Slope[i]/m.Config.SlopeScale, 0, 1)
		wet := clamp(m.Wetness[i], 0, 1)
		support := m.TectonicSupport[i]
		resistance := m.ErosionResistance[i]

		valley := m.Config.MaxValleyIncision *
			m.FluvialPower[i] *
			(1 - 0.35*resistance)

		if input.LakeID[i] > 0 {
			// Lakes occupy accommodation space rather than being carved as deep
			// river canyons through every lake cell.
			valley *= 0.12
		}

		hillslope := m.Config.MaxHillslopeErosion *
			slopeNorm *
			(0.25 + 0.75*wet) *
			(1 - m.Config.TectonicSupportStrength*support) *
			(1 - 0.30*resistance)
		hillslope = math.Max(0, hillslope)

		total := clamp(valley+hillslope, 0, m.Config.MaxValleyIncision+m.Config.MaxHillslopeErosion)
		original := m.OriginalElevation[i]

		// Phase 4.1 only carves subaerial terrain. Coast/river-mouth reshaping
		// across sea level belongs to later deposition/coastal refinement.
		floor := m.Config.SeaLevelNormalized + 0.001
		eroded := original
		if original > floor {
			eroded = math.Max(floor, original-total)
		}
		actual := math.Max(0, original-eroded)

		if total > 0 {
			scale := actual / total
			valley *= scale
			hillslope *= scale
		}

		m.ValleyIncision[i] = valley
		m.HillslopeErosion[i] = hillslope
		m.TotalErosion[i] = actual
		m.ErodedElevation[i] = eroded
	}
}

func (m *Model) computeRelief() {
	r := m.Config.ReliefRadiusCells
	for z := 0; z < m.Height; z++ {
		for x := 0; x < m.Width; x++ {
			i := z*m.Width + x
			if !m.LandMask[i] {
				continue
			}

			minV := math.Inf(1)
			maxV := math.Inf(-1)
			for dz := -r; dz <= r; dz++ {
				nz := z + dz
				if nz < 0 || nz >= m.Height {
					continue
				}
				for dx := -r; dx <= r; dx++ {
					nx := x + dx
					if nx < 0 || nx >= m.Width {
						continue
					}
					ni := nz*m.Width + nx
					if !m.LandMask[ni] {
						continue
					}
					v := m.ErodedElevation[ni]
					if v < minV {
						minV = v
					}
					if v > maxV {
						maxV = v
					}
				}
			}

			if math.IsInf(minV, 1) || math.IsInf(maxV, -1) {
				continue
			}
			relief := math.Max(0, maxV-minV)
			m.LocalRelief[i] = relief
			m.ReliefIndex[i] = clamp(relief/m.Config.ReliefScale, 0, 1)
		}
	}
}

func localSlope(elevation []float64, land []bool, width, height, x, z int) float64 {
	i := z*width + x
	center := elevation[i]
	left, right, up, down := center, center, center, center

	if x > 0 {
		ni := i - 1
		if land[ni] {
			left = elevation[ni]
		}
	}
	if x+1 < width {
		ni := i + 1
		if land[ni] {
			right = elevation[ni]
		}
	}
	if z > 0 {
		ni := i - width
		if land[ni] {
			up = elevation[ni]
		}
	}
	if z+1 < height {
		ni := i + width
		if land[ni] {
			down = elevation[ni]
		}
	}

	gx := (right - left) * 0.5
	gz := (down - up) * 0.5
	return math.Hypot(gx, gz)
}

func normalizedAccumulation(accumulation, threshold float64) float64 {
	if accumulation <= 0 {
		return 0
	}
	if threshold <= 0 {
		threshold = 1
	}
	return clamp(math.Log1p(accumulation/threshold)/math.Log(51), 0, 1)
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
