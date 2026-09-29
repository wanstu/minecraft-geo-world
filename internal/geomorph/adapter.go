package geomorph

import (
	"geoworld/internal/erosion"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
)

func InputFromModels(h *hydrology.Model, e *erosion.Model, l *landform.Model) Input {
	return Input{
		Width:                   h.Width,
		Height:                  h.Height,
		LandMask:                append([]bool(nil), h.LandMask...),
		Elevation:               append([]float64(nil), e.ErodedElevation...),
		Slope:                   append([]float64(nil), e.Slope...),
		ReliefIndex:             append([]float64(nil), e.ReliefIndex...),
		ValleyIncision:          append([]float64(nil), e.ValleyIncision...),
		FluvialPower:            append([]float64(nil), e.FluvialPower...),
		TectonicSupport:         append([]float64(nil), e.TectonicSupport...),
		FlowTo:                  append([]int(nil), h.FlowTo...),
		FlowAccumulation:        append([]float64(nil), h.FlowAccumulation...),
		RiverStrength:           append([]float64(nil), h.RiverStrength...),
		PerennialRiverPotential: append([]float64(nil), h.PerennialRiverPotential...),
		SeasonalRiverPotential:  append([]float64(nil), h.SeasonalRiverPotential...),
		WetlandPotential:        append([]float64(nil), h.WetlandPotential...),
		LakeID:                  append([]int(nil), h.LakeID...),
		RiverThreshold:          h.RiverThreshold,
		RollingPlainWeight:      append([]float64(nil), l.RollingPlainWeight...),
		HillWeight:              append([]float64(nil), l.HillWeight...),
		FoothillWeight:          append([]float64(nil), l.FoothillWeight...),
		MountainWeight:          append([]float64(nil), l.MountainWeight...),
		PlateauWeight:           append([]float64(nil), l.PlateauWeight...),
		ValleyWeight:            append([]float64(nil), l.ValleyWeight...),
	}
}
