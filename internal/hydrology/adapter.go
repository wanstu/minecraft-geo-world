package hydrology

import (
	"geoworld/internal/climate"
	"geoworld/internal/phase1"
)

func InputFromWorld(world *phase1.World, climateModel *climate.Model) Input {
	n := world.Config.Width * world.Config.Height
	land := append([]bool(nil), climateModel.LandMask...)

	precip := climateModel.SeasonalAnnualPrecipMM
	if len(precip) != n {
		precip = climateModel.AnnualPrecipMM
	}

	return Input{
		Width:               world.Config.Width,
		Height:              world.Config.Height,
		Elevation:           append([]float64(nil), world.Elevation...),
		LandMask:            land,
		AnnualPrecipMM:      append([]float64(nil), precip...),
		RainfallSeasonality: append([]float64(nil), climateModel.RainfallSeasonality...),
		WarmSeasonPrecipMM:  append([]float64(nil), climateModel.WarmSeasonPrecipMM...),
		ColdSeasonPrecipMM:  append([]float64(nil), climateModel.ColdSeasonPrecipMM...),
	}
}
