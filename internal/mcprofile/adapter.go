package mcprofile

import (
	"geoworld/internal/ecology"
	"geoworld/internal/geomorph"
	"geoworld/internal/landform"
)

func InputFromModels(e *ecology.Model, l *landform.Model, g *geomorph.Model) Input {
	regions := make([]uint8, len(e.EcoRegion))
	for i, r := range e.EcoRegion {
		regions[i] = uint8(r)
	}
	return Input{
		Width:               e.Width,
		Height:              e.Height,
		LandMask:            append([]bool(nil), e.LandMask...),
		EcoRegion:           regions,
		RegionConfidence:    append([]float64(nil), e.RegionConfidence...),
		TreeCoverPotential:  append([]float64(nil), e.TreeCoverPotential...),
		GrassCoverPotential: append([]float64(nil), e.GrassCoverPotential...),
		ShrubCoverPotential: append([]float64(nil), e.ShrubCoverPotential...),
		BareGroundPotential: append([]float64(nil), e.BareGroundPotential...),
		SnowCoverPotential:  append([]float64(nil), e.SnowCoverPotential...),
		SoilMoisture:        append([]float64(nil), e.SoilMoisture...),
		WetlandEcology:      append([]float64(nil), e.WetlandEcology...),
		AlpineEcology:       append([]float64(nil), e.AlpineEcology...),
		RollingPlainWeight:  append([]float64(nil), l.RollingPlainWeight...),
		HillWeight:          append([]float64(nil), l.HillWeight...),
		FoothillWeight:      append([]float64(nil), l.FoothillWeight...),
		MountainWeight:      append([]float64(nil), l.MountainWeight...),
		PlateauWeight:       append([]float64(nil), l.PlateauWeight...),
		FloodplainWeight:    append([]float64(nil), g.FloodplainWeight...),
		AlluvialPlainWeight: append([]float64(nil), g.AlluvialPlainWeight...),
		AlluvialFanWeight:   append([]float64(nil), g.AlluvialFanWeight...),
		CanyonWeight:        append([]float64(nil), g.CanyonWeight...),
	}
}
