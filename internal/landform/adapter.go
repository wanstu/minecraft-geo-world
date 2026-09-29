package landform

import (
	"geoworld/internal/erosion"
	"geoworld/internal/phase1"
)

func InputFromWorld(world *phase1.World, erosionModel *erosion.Model) Input {
	land := append([]bool(nil), erosionModel.LandMask...)

	return Input{
		Width:           world.Config.Width,
		Height:          world.Config.Height,
		LandMask:        land,
		Elevation:       append([]float64(nil), erosionModel.ErodedElevation...),
		ReliefIndex:     append([]float64(nil), erosionModel.ReliefIndex...),
		Slope:           append([]float64(nil), erosionModel.Slope...),
		TerrainMaturity: append([]float64(nil), erosionModel.TerrainMaturity...),
		TectonicSupport: append([]float64(nil), erosionModel.TectonicSupport...),
		CrustThickness:  append([]float64(nil), world.CrustThickness...),
		ValleyIncision:  append([]float64(nil), erosionModel.ValleyIncision...),
		FluvialPower:    append([]float64(nil), erosionModel.FluvialPower...),
	}
}
