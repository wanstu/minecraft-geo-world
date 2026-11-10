package realgeom

import (
	"fmt"
	"math"
	"sort"
)

const (
	LayerJRCOccurrence     = "jrc-occurrence"
	LayerJRCSeasonality    = "jrc-seasonality"
	LayerHydroDirection    = "hydrosheds-direction"
	LayerHydroAccumulation = "hydrosheds-accumulation"
)

type HydroRasterConfig struct {
	Layer          string
	RegionID       string
	DatasetID      string
	DatasetVersion string
	Crop           Bounds
	SampleStride   int
	StripeRows     int
	NoData         float64
	HasNoData      bool
}

type HydroRasterReport struct {
	SchemaVersion       int             `json:"schemaVersion"`
	RegionID            string          `json:"regionId"`
	DatasetID           string          `json:"datasetId"`
	DatasetVersion      string          `json:"datasetVersion"`
	Layer               string          `json:"layer"`
	Input               GeoTIFFScanInfo `json:"input"`
	SampleStride        int             `json:"sampleStride"`
	SampledCells        int             `json:"sampledCells"`
	NoDataCells         int             `json:"noDataCells"`
	ValidCells          int             `json:"validCells"`
	InvalidCells        int             `json:"invalidCells"`
	NoDataValue         float64         `json:"noDataValue"`
	ValueDistribution   *Percentiles    `json:"valueDistribution,omitempty"`
	FlowDirectionCounts map[string]int  `json:"flowDirectionCounts,omitempty"`
	Water               *WaterSummary   `json:"water,omitempty"`
	QualityFlags        []string        `json:"qualityFlags"`
}

type WaterSummary struct {
	DryCells          int     `json:"dryCells"`
	ActiveCells       int     `json:"activeCells"`
	SeasonalCells     int     `json:"seasonalCells"`
	PermanentCells    int     `json:"permanentCells"`
	ActiveFraction    float64 `json:"activeFraction"`
	SeasonalFraction  float64 `json:"seasonalFraction"`
	PermanentFraction float64 `json:"permanentFraction"`
}

func SupportedHydroRasterLayer(s string) bool {
	switch s {
	case LayerJRCOccurrence, LayerJRCSeasonality, LayerHydroDirection, LayerHydroAccumulation:
		return true
	default:
		return false
	}
}

func DefaultLayerNoData(layer string) (float64, error) {
	switch layer {
	case LayerJRCOccurrence, LayerJRCSeasonality, LayerHydroDirection:
		return 255, nil
	case LayerHydroAccumulation:
		return 4294967295, nil
	}
	return 0, fmt.Errorf("realgeom: unsupported raster layer %q", layer)
}

func ExtractHydroRaster(path string, cfg HydroRasterConfig) (HydroRasterReport, error) {
	if !SupportedHydroRasterLayer(cfg.Layer) {
		return HydroRasterReport{}, fmt.Errorf("realgeom: invalid layer %q", cfg.Layer)
	}
	if !validBounds(cfg.Crop) {
		return HydroRasterReport{}, fmt.Errorf("realgeom: hydro layer needs a valid registry crop")
	}
	if cfg.SampleStride < 1 {
		cfg.SampleStride = 1
	}
	if !cfg.HasNoData {
		v, err := DefaultLayerNoData(cfg.Layer)
		if err != nil {
			return HydroRasterReport{}, err
		}
		cfg.NoData = v
	}
	r := HydroRasterReport{
		SchemaVersion: 1, RegionID: cfg.RegionID, DatasetID: cfg.DatasetID,
		DatasetVersion: cfg.DatasetVersion, Layer: cfg.Layer,
		SampleStride: cfg.SampleStride, NoDataValue: cfg.NoData,
		QualityFlags: []string{"source_pixels_only", "registry_bbox_pixel_center_crop", "no_overview_resampling"},
	}
	histogram := map[int]int{}
	vals := []float64{}
	flow := map[string]int{}
	water := &WaterSummary{}
	// The scan is deliberately allowed to see NoData: the denominator must
	// include sampled unavailable pixels rather than silently dropping them.
	info, err := ScanGeoTIFF(path, GeoTIFFScanOptions{Crop: &cfg.Crop, SampleStride: cfg.SampleStride, StripeRows: cfg.StripeRows},
		func(_, _ int, v float64) error {
			r.SampledCells++
			if v == cfg.NoData {
				r.NoDataCells++
				return nil
			}
			valid := false
			switch cfg.Layer {
			case LayerJRCOccurrence:
				valid = math.Trunc(v) == v && v >= 0 && v <= 100
				if valid {
					histogram[int(v)]++
					if v == 0 {
						water.DryCells++
					} else {
						water.ActiveCells++
						if v >= 90 {
							water.PermanentCells++
						} else {
							water.SeasonalCells++
						}
					}
				}
			case LayerJRCSeasonality:
				valid = math.Trunc(v) == v && v >= 0 && v <= 12
				if valid {
					histogram[int(v)]++
					if v == 0 {
						water.DryCells++
					} else {
						water.ActiveCells++
						if v == 12 {
							water.PermanentCells++
						} else {
							water.SeasonalCells++
						}
					}
				}
			case LayerHydroDirection:
				iv := int(v)
				valid = v == 0 || iv == 1 || iv == 2 || iv == 4 || iv == 8 || iv == 16 || iv == 32 || iv == 64 || iv == 128
				if valid {
					flow[fmt.Sprintf("%d", iv)]++
				}
			case LayerHydroAccumulation:
				valid = v >= 0 && math.Trunc(v) == v && v < 4294967295
				if valid {
					vals = append(vals, v)
				}
			}
			if valid {
				r.ValidCells++
			} else {
				r.InvalidCells++
			}
			return nil
		})
	if err != nil {
		return HydroRasterReport{}, err
	}
	r.Input = info
	if r.SampledCells == 0 {
		return r, fmt.Errorf("realgeom: no sampled pixels in crop")
	}
	if r.InvalidCells > 0 {
		return r, fmt.Errorf("realgeom: %d values violate %s semantics", r.InvalidCells, cfg.Layer)
	}
	if r.ValidCells == 0 {
		return r, fmt.Errorf("realgeom: no valid values for %s", cfg.Layer)
	}
	switch cfg.Layer {
	case LayerJRCOccurrence, LayerJRCSeasonality:
		q := histogramPercentiles(histogram, r.ValidCells)
		r.ValueDistribution = &q
		denom := float64(r.ValidCells)
		water.ActiveFraction = float64(water.ActiveCells) / denom
		water.SeasonalFraction = float64(water.SeasonalCells) / denom
		water.PermanentFraction = float64(water.PermanentCells) / denom
		r.Water = water
		r.QualityFlags = append(r.QualityFlags, "water_fractions_use_valid_pixels_only")
		if cfg.Layer == LayerJRCOccurrence {
			r.QualityFlags = append(r.QualityFlags, "permanent_is_occurrence_ge_90_percent_proxy")
		} else {
			r.QualityFlags = append(r.QualityFlags, "zero_months_represents_nonwater_or_unobserved")
		}
	case LayerHydroDirection:
		r.FlowDirectionCounts = flow
		r.QualityFlags = append(r.QualityFlags, "zero_direction_is_outlet_or_sink_without_msk")
	case LayerHydroAccumulation:
		q := summarize(vals)
		r.ValueDistribution = &q
		r.QualityFlags = append(r.QualityFlags, "upstream_count_in_source_cells")
	}
	return r, nil
}

// histogramPercentiles is exact for the integer-coded JRC products.
func histogramPercentiles(hist map[int]int, n int) Percentiles {
	if n == 0 {
		return Percentiles{}
	}
	keys := make([]int, 0, len(hist))
	for k := range hist {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	at := func(rank int) float64 {
		count := 0
		for _, k := range keys {
			count += hist[k]
			if rank < count {
				return float64(k)
			}
		}
		return float64(keys[len(keys)-1])
	}
	quant := func(p float64) float64 {
		rank := p * float64(n-1)
		lo, hi := int(math.Floor(rank)), int(math.Ceil(rank))
		frac := rank - float64(lo)
		return at(lo)*(1-frac) + at(hi)*frac
	}
	return Percentiles{Min: float64(keys[0]), Max: float64(keys[len(keys)-1]), P10: quant(.1), P50: quant(.5), P90: quant(.9), P99: quant(.99)}
}
