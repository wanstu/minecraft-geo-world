package ecology

import (
	"geoworld/internal/climate"
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/hydrology"
	"geoworld/internal/landform"
)

func InputFromModels(c *climate.Model, h *hydrology.Model, e *erosion.Model, l *landform.Model, g *geomorph.Model) Input {
	return Input{
		Width:                   c.Width,
		Height:                  c.Height,
		LandMask:                append([]bool(nil), c.LandMask...),
		AnnualTempC:             append([]float64(nil), c.AdjustedAnnualTempC...),
		SummerTempC:             append([]float64(nil), c.SummerTempC...),
		WinterTempC:             append([]float64(nil), c.WinterTempC...),
		AnnualPrecipMM:          append([]float64(nil), c.SeasonalAnnualPrecipMM...),
		WarmSeasonPrecipMM:      append([]float64(nil), c.WarmSeasonPrecipMM...),
		ColdSeasonPrecipMM:      append([]float64(nil), c.ColdSeasonPrecipMM...),
		RainfallSeasonality:     append([]float64(nil), c.RainfallSeasonality...),
		MonsoonIndex:            append([]float64(nil), c.MonsoonIndex...),
		SeaSurfaceTempC:         append([]float64(nil), c.SeaSurfaceTempC...),
		Elevation:               append([]float64(nil), e.ErodedElevation...),
		ReliefIndex:             append([]float64(nil), e.ReliefIndex...),
		RollingPlainWeight:      append([]float64(nil), l.RollingPlainWeight...),
		HillWeight:              append([]float64(nil), l.HillWeight...),
		FoothillWeight:          append([]float64(nil), l.FoothillWeight...),
		MountainWeight:          append([]float64(nil), l.MountainWeight...),
		PlateauWeight:           append([]float64(nil), l.PlateauWeight...),
		WetlandPotential:        append([]float64(nil), h.WetlandPotential...),
		PerennialRiverPotential: append([]float64(nil), h.PerennialRiverPotential...),
		SeasonalRiverPotential:  append([]float64(nil), h.SeasonalRiverPotential...),
		FloodplainWeight:        append([]float64(nil), g.FloodplainWeight...),
		AlluvialPlainWeight:     append([]float64(nil), g.AlluvialPlainWeight...),
		ValleyFloorWidth:        append([]float64(nil), g.ValleyFloorWidth...),
	}
}
