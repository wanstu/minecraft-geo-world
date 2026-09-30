package terrain

import (
	"geoworld/internal/erosion"
	"geoworld/internal/geomorph"
	"geoworld/internal/landform"
)

func InputFromModels(erosionModel *erosion.Model, landformModel *landform.Model, geomorphModel *geomorph.Model) Input {
	return Input{
		Width:               erosionModel.Width,
		Height:              erosionModel.Height,
		Geometry:            erosionModel.Geometry,
		BaseElevation:       append([]float64(nil), erosionModel.ErodedElevation...),
		RollingPlainWeight:  append([]float64(nil), landformModel.RollingPlainWeight...),
		HillWeight:          append([]float64(nil), landformModel.HillWeight...),
		FoothillWeight:      append([]float64(nil), landformModel.FoothillWeight...),
		MountainWeight:      append([]float64(nil), landformModel.MountainWeight...),
		PlateauWeight:       append([]float64(nil), landformModel.PlateauWeight...),
		ValleyWeight:        append([]float64(nil), landformModel.ValleyWeight...),
		FloodplainWeight:    append([]float64(nil), geomorphModel.FloodplainWeight...),
		AlluvialPlainWeight: append([]float64(nil), geomorphModel.AlluvialPlainWeight...),
		AlluvialFanWeight:   append([]float64(nil), geomorphModel.AlluvialFanWeight...),
		CanyonWeight:        append([]float64(nil), geomorphModel.CanyonWeight...),
		ValleyFloorWidth:    append([]float64(nil), geomorphModel.ValleyFloorWidth...),
		DepositionPotential: append([]float64(nil), geomorphModel.DepositionPotential...),
	}
}
