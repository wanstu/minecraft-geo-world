package climate

import "geoworld/internal/phase1"

func InputFromPhase1(w *phase1.World) Input {
	land := append([]bool(nil), w.Surface.LandMask...)
	return Input{
		Width:     w.Config.Width,
		Height:    w.Config.Height,
		Geometry:  w.Geometry,
		Elevation: append([]float64(nil), w.Elevation...),
		LandMask:  land,
	}
}
