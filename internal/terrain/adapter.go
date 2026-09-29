package terrain

import (
	"geoworld/internal/erosion"
	"geoworld/internal/landform"
)

func InputFromModels(erosionModel *erosion.Model, landformModel *landform.Model) Input {
	return Input{
		Width:              erosionModel.Width,
		Height:             erosionModel.Height,
		BaseElevation:      append([]float64(nil), erosionModel.ErodedElevation...),
		RollingPlainWeight: append([]float64(nil), landformModel.RollingPlainWeight...),
		HillWeight:         append([]float64(nil), landformModel.HillWeight...),
		FoothillWeight:     append([]float64(nil), landformModel.FoothillWeight...),
		MountainWeight:     append([]float64(nil), landformModel.MountainWeight...),
		PlateauWeight:      append([]float64(nil), landformModel.PlateauWeight...),
		ValleyWeight:       append([]float64(nil), landformModel.ValleyWeight...),
	}
}
