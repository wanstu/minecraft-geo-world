package climate

import "geoworld/internal/phase1"

func InputFromPhase1(w *phase1.World) Input {
	land := make([]bool, len(w.Crust))
	for i, crust := range w.Crust {
		land[i] = crust == phase1.Continental
	}
	return Input{
		Width:     w.Config.Width,
		Height:    w.Config.Height,
		Elevation: append([]float64(nil), w.Elevation...),
		LandMask:  land,
	}
}
