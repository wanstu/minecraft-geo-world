package realgeom

import (
	"errors"
	"math"
)

type Raster struct {
	Source   string
	Width    int
	Height   int
	West     float64
	South    float64
	East     float64
	North    float64
	Values   []float64
	NoData   float64
	CellDegX float64
	CellDegY float64
}

func (r Raster) Validate() error {
	if r.Width < 2 || r.Height < 2 {
		return errors.New("realgeom: raster must be at least 2x2")
	}
	if len(r.Values) != r.Width*r.Height {
		return errors.New("realgeom: raster value count does not match dimensions")
	}
	if !(r.West < r.East) || !(r.South < r.North) {
		return errors.New("realgeom: invalid geographic bounds")
	}
	if r.CellDegX <= 0 || r.CellDegY <= 0 {
		return errors.New("realgeom: cell size must be positive")
	}
	return nil
}

func (r Raster) At(x, y int) (float64, bool) {
	if x < 0 || x >= r.Width || y < 0 || y >= r.Height {
		return 0, false
	}
	v := r.Values[y*r.Width+x]
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	if !math.IsNaN(r.NoData) && v == r.NoData {
		return 0, false
	}
	return v, true
}

func (r Raster) LatitudeAtRow(y int) float64 {
	if r.Height <= 1 {
		return 0.5 * (r.South + r.North)
	}
	return r.North - float64(y)*r.CellDegY
}

func metersPerDegreeLatitude(latDeg float64) float64 {
	lat := latDeg * math.Pi / 180
	return 111132.92 - 559.82*math.Cos(2*lat) + 1.175*math.Cos(4*lat)
}

func metersPerDegreeLongitude(latDeg float64) float64 {
	lat := latDeg * math.Pi / 180
	return 111412.84*math.Cos(lat) - 93.5*math.Cos(3*lat) + 0.118*math.Cos(5*lat)
}
