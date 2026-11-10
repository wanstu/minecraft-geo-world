package realgeom

import (
	"errors"
	"math"
	"sort"
)

type Percentiles struct {
	P10 float64 `json:"p10"`
	P50 float64 `json:"p50"`
	P90 float64 `json:"p90"`
	P99 float64 `json:"p99"`
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type Bounds struct {
	West  float64 `json:"west"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	North float64 `json:"north"`
}

type FileSummary struct {
	Path         string `json:"path"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	ValidSamples int    `json:"validSamples"`
	Bounds       Bounds `json:"bounds"`
}

type Report struct {
	SchemaVersion       int           `json:"schemaVersion"`
	RegionID            string        `json:"regionId,omitempty"`
	DatasetID           string        `json:"datasetId"`
	DatasetVersion      string        `json:"datasetVersion,omitempty"`
	InputFormat         string        `json:"inputFormat"`
	Inputs              []FileSummary `json:"inputs"`
	ValidElevationCells int           `json:"validElevationCells"`
	ValidSlopeCells     int           `json:"validSlopeCells"`
	ReliefWindowKM      float64       `json:"reliefWindowKm"`
	ElevationMeters     Percentiles   `json:"elevationMeters"`
	SlopeDegrees        Percentiles   `json:"slopeDegrees"`
	LocalReliefMeters   Percentiles   `json:"localReliefMeters"`
	HypsometricIntegral float64       `json:"hypsometricIntegral"`
}

type AnalyzeConfig struct {
	RegionID       string
	DatasetID      string
	DatasetVersion string
	InputFormat    string
	ReliefWindowKM float64
	SampleStride   int
}

func Analyze(rasters []Raster, cfg AnalyzeConfig) (Report, error) {
	if len(rasters) == 0 {
		return Report{}, errors.New("realgeom: at least one raster is required")
	}
	if cfg.DatasetID == "" {
		return Report{}, errors.New("realgeom: dataset id is required")
	}
	if cfg.ReliefWindowKM <= 0 {
		cfg.ReliefWindowKM = 5
	}
	if cfg.SampleStride < 1 {
		cfg.SampleStride = 1
	}

	elevations := make([]float64, 0, 1024)
	slopes := make([]float64, 0, 1024)
	reliefs := make([]float64, 0, 256)
	inputs := make([]FileSummary, 0, len(rasters))

	for _, r := range rasters {
		if err := r.Validate(); err != nil {
			return Report{}, err
		}
		valid := 0
		for y := 0; y < r.Height; y += cfg.SampleStride {
			for x := 0; x < r.Width; x += cfg.SampleStride {
				if v, ok := r.At(x, y); ok {
					elevations = append(elevations, v)
					valid++
				}
			}
		}
		slopes = append(slopes, rasterSlopes(r, cfg.SampleStride)...)
		reliefs = append(reliefs, rasterBlockRelief(r, cfg.ReliefWindowKM)...)
		inputs = append(inputs, FileSummary{
			Path: r.Source, Width: r.Width, Height: r.Height, ValidSamples: valid,
			Bounds: Bounds{West: r.West, South: r.South, East: r.East, North: r.North},
		})
	}
	if len(elevations) == 0 {
		return Report{}, errors.New("realgeom: no valid elevation samples")
	}
	elevationStats := summarize(elevations)
	hypsometric := 0.0
	if elevationStats.Max > elevationStats.Min {
		sum := 0.0
		for _, v := range elevations {
			sum += v
		}
		mean := sum / float64(len(elevations))
		hypsometric = (mean - elevationStats.Min) / (elevationStats.Max - elevationStats.Min)
	}

	return Report{
		SchemaVersion: 1, RegionID: cfg.RegionID, DatasetID: cfg.DatasetID,
		DatasetVersion: cfg.DatasetVersion, InputFormat: cfg.InputFormat, Inputs: inputs,
		ValidElevationCells: len(elevations), ValidSlopeCells: len(slopes),
		ReliefWindowKM:  cfg.ReliefWindowKM,
		ElevationMeters: elevationStats, SlopeDegrees: summarize(slopes),
		LocalReliefMeters: summarize(reliefs), HypsometricIntegral: hypsometric,
	}, nil
}

func rasterSlopes(r Raster, stride int) []float64 {
	out := make([]float64, 0, (r.Width*r.Height)/(stride*stride))
	for y := 1; y < r.Height-1; y += stride {
		lat := r.LatitudeAtRow(y)
		dx := r.CellDegX * metersPerDegreeLongitude(lat)
		dy := r.CellDegY * metersPerDegreeLatitude(lat)
		if dx <= 0 || dy <= 0 {
			continue
		}
		for x := 1; x < r.Width-1; x += stride {
			left, okL := r.At(x-1, y)
			right, okR := r.At(x+1, y)
			up, okU := r.At(x, y-1)
			down, okD := r.At(x, y+1)
			if !okL || !okR || !okU || !okD {
				continue
			}
			dzdx := (right - left) / (2 * dx)
			dzdy := (down - up) / (2 * dy)
			slope := math.Atan(math.Hypot(dzdx, dzdy)) * 180 / math.Pi
			if !math.IsNaN(slope) && !math.IsInf(slope, 0) {
				out = append(out, slope)
			}
		}
	}
	return out
}

func rasterBlockRelief(r Raster, windowKM float64) []float64 {
	lat := 0.5 * (r.South + r.North)
	cellM := 0.5 * (r.CellDegX*metersPerDegreeLongitude(lat) +
		r.CellDegY*metersPerDegreeLatitude(lat))
	if cellM <= 0 {
		return nil
	}
	block := int(math.Round(windowKM * 1000 / cellM))
	if block < 2 {
		block = 2
	}
	out := make([]float64, 0, (r.Width/block+1)*(r.Height/block+1))
	for y0 := 0; y0 < r.Height; y0 += block {
		y1 := minInt(y0+block, r.Height)
		for x0 := 0; x0 < r.Width; x0 += block {
			x1 := minInt(x0+block, r.Width)
			minV := math.Inf(1)
			maxV := math.Inf(-1)
			valid := 0
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					v, ok := r.At(x, y)
					if !ok {
						continue
					}
					valid++
					if v < minV {
						minV = v
					}
					if v > maxV {
						maxV = v
					}
				}
			}
			if valid >= 4 && maxV >= minV {
				out = append(out, maxV-minV)
			}
		}
	}
	return out
}

func summarize(values []float64) Percentiles {
	if len(values) == 0 {
		return Percentiles{}
	}
	copyValues := append([]float64(nil), values...)
	sort.Float64s(copyValues)
	return Percentiles{
		P10: percentileSorted(copyValues, 0.10),
		P50: percentileSorted(copyValues, 0.50),
		P90: percentileSorted(copyValues, 0.90),
		P99: percentileSorted(copyValues, 0.99),
		Min: copyValues[0], Max: copyValues[len(copyValues)-1],
	}
}

func percentileSorted(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if q <= 0 {
		return values[0]
	}
	if q >= 1 {
		return values[len(values)-1]
	}
	pos := q * float64(len(values)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return values[lo]
	}
	t := pos - float64(lo)
	return values[lo] + (values[hi]-values[lo])*t
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
