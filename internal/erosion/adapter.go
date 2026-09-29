package erosion

import (
	"geoworld/internal/climate"
	"geoworld/internal/hydrology"
	"geoworld/internal/phase1"
)

func InputFromWorld(world *phase1.World, climateModel *climate.Model, hydro *hydrology.Model) Input {
	n := world.Config.Width * world.Config.Height
	land := append([]bool(nil), hydro.LandMask...)

	precip := climateModel.SeasonalAnnualPrecipMM
	if len(precip) != n {
		precip = climateModel.AnnualPrecipMM
	}

	return Input{
		Width:             world.Config.Width,
		Height:            world.Config.Height,
		LandMask:          land,
		Elevation:         append([]float64(nil), world.Elevation...),
		CrustThickness:    append([]float64(nil), world.CrustThickness...),
		TectonicIntensity: append([]float64(nil), world.TectonicIntensity...),
		TectonicInfluence: append([]float64(nil), world.TectonicInfluence...),
		AnnualPrecipMM:    append([]float64(nil), precip...),
		FlowAccumulation:  append([]float64(nil), hydro.FlowAccumulation...),
		RiverStrength:     append([]float64(nil), hydro.RiverStrength...),
		LakeID:            append([]int(nil), hydro.LakeID...),
		RiverThreshold:    hydro.RiverThreshold,
	}
}
