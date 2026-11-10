package realgeom

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
)

type ReferenceGrid struct {
	Columns int    `json:"columns"`
	Rows    int    `json:"rows"`
	Bounds  Bounds `json:"bounds"`
}

type ReferenceSource struct {
	DatasetID         string           `json:"datasetId"`
	DatasetVersion    string           `json:"datasetVersion"`
	Layer             string           `json:"layer"`
	ObservationPeriod string           `json:"observationPeriod"`
	FileName          string           `json:"fileName"`
	SHA256            string           `json:"sha256"`
	SidecarSHA256     string           `json:"sidecarSha256,omitempty"`
	SampleStride      int              `json:"sampleStride,omitempty"`
	Raster            *GeoTIFFScanInfo `json:"raster,omitempty"`
}

type EnvelopeBand struct {
	Name           string      `json:"name"`
	Unit           string      `json:"unit"`
	Status         string      `json:"status"`
	Statistic      string      `json:"statistic"`
	SampleBlocks   int         `json:"sampleBlocks"`
	ExcludedBlocks int         `json:"excludedBlocks"`
	Percentiles    Percentiles `json:"percentiles"`
}

type ReferenceCell struct {
	Row           int                `json:"row"`
	Column        int                `json:"column"`
	Bounds        Bounds             `json:"bounds"`
	ValidSamples  int                `json:"validSamples"`
	NoDataSamples int                `json:"noDataSamples"`
	Metrics       map[string]float64 `json:"metrics,omitempty"`
}

type ReferenceEnvelope struct {
	SchemaVersion    int             `json:"schemaVersion"`
	RegionID         string          `json:"regionId"`
	Status           string          `json:"status"`
	Grid             ReferenceGrid   `json:"grid"`
	Source           ReferenceSource `json:"source"`
	MinValidPerBlock int             `json:"minValidPerBlock"`
	ValidSamples     int             `json:"validSamples"`
	NoDataSamples    int             `json:"noDataSamples"`
	QualityFlags     []string        `json:"qualityFlags"`
	Metrics          []EnvelopeBand  `json:"metrics"`
	Cells            []ReferenceCell `json:"cells"`
}

type ReferenceExtractConfig struct {
	RegionID         string
	Bounds           Bounds
	Columns          int
	Rows             int
	DatasetVersion   string
	Layer            string
	SampleStride     int
	StripeRows       int
	MinValidPerBlock int
}

type referenceAccumCell struct {
	valid, nodata                          int
	positive, frequent, partYear, fullYear int
	directions                             [8]int
	accumulator                            []float64
}

func (g ReferenceGrid) Validate() error {
	if !validBounds(g.Bounds) || g.Columns < 1 || g.Rows < 1 || g.Columns*g.Rows > 10000 {
		return fmt.Errorf("realgeom: invalid reference grid")
	}
	return nil
}
func (g ReferenceGrid) CellBounds(row, col int) Bounds {
	dx := (g.Bounds.East - g.Bounds.West) / float64(g.Columns)
	dy := (g.Bounds.North - g.Bounds.South) / float64(g.Rows)
	return Bounds{
		West:  g.Bounds.West + float64(col)*dx,
		East:  g.Bounds.West + float64(col+1)*dx,
		North: g.Bounds.North - float64(row)*dy,
		South: g.Bounds.North - float64(row+1)*dy,
	}
}
func (g ReferenceGrid) CellAt(lon, lat float64) (int, bool) {
	if lon < g.Bounds.West-1e-8 || lon > g.Bounds.East+1e-8 ||
		lat < g.Bounds.South-1e-8 || lat > g.Bounds.North+1e-8 {
		return 0, false
	}
	col := int(math.Floor((lon - g.Bounds.West) / (g.Bounds.East - g.Bounds.West) * float64(g.Columns)))
	row := int(math.Floor((g.Bounds.North - lat) / (g.Bounds.North - g.Bounds.South) * float64(g.Rows)))
	if col < 0 {
		col = 0
	}
	if col >= g.Columns {
		col = g.Columns - 1
	}
	if row < 0 {
		row = 0
	}
	if row >= g.Rows {
		row = g.Rows - 1
	}
	return row*g.Columns + col, true
}

func fingerprint(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func SourceDescription(layer, version string) (string, string, error) {
	switch layer {
	case LayerJRCOccurrence:
		if version != "" && version != "1.5" {
			return "", "", fmt.Errorf("realgeom: JRC occurrence locked at v1.5")
		}
		return "JRC_Global_Surface_Water", "1984-2024", nil
	case LayerJRCSeasonality:
		if version != "" && version != "1.5" {
			return "", "", fmt.Errorf("realgeom: JRC seasonality locked at v1.5")
		}
		return "JRC_Global_Surface_Water", "2024", nil
	case LayerHydroDirection, LayerHydroAccumulation:
		if version != "" && version != "1.1" {
			return "", "", fmt.Errorf("realgeom: HydroSHEDS locked at v1.1")
		}
		return "HydroSHEDS_core", "static_v1.1", nil
	case "hydrorivers":
		if version != "" && version != "1" {
			return "", "", fmt.Errorf("realgeom: HydroRIVERS locked at v1")
		}
		return "HydroRIVERS", "static_v1", nil
	default:
		return "", "", fmt.Errorf("realgeom: unsupported layer %q", layer)
	}
}

func NewReferenceEnvelope(path string, cfg ReferenceExtractConfig) (ReferenceEnvelope, error) {
	grid := ReferenceGrid{Columns: cfg.Columns, Rows: cfg.Rows, Bounds: cfg.Bounds}
	if err := grid.Validate(); err != nil {
		return ReferenceEnvelope{}, err
	}
	dataset, period, err := SourceDescription(cfg.Layer, cfg.DatasetVersion)
	if err != nil {
		return ReferenceEnvelope{}, err
	}
	if cfg.SampleStride < 1 {
		cfg.SampleStride = 1
	}
	if cfg.MinValidPerBlock < 1 {
		cfg.MinValidPerBlock = 32
	}
	sha, err := fingerprint(path)
	if err != nil {
		return ReferenceEnvelope{}, err
	}
	report := ReferenceEnvelope{
		SchemaVersion: 1, RegionID: cfg.RegionID,
		Status: "candidate_spatial_distribution_not_generation_target",
		Grid:   grid, MinValidPerBlock: cfg.MinValidPerBlock,
		Source: ReferenceSource{DatasetID: dataset, DatasetVersion: cfg.DatasetVersion,
			Layer: cfg.Layer, ObservationPeriod: period, FileName: filepath.Base(path),
			SHA256: sha, SampleStride: cfg.SampleStride},
		QualityFlags: []string{"geographic_bbox_grid", "not_global_ground_truth", "not_world_generation_target"},
		Cells:        make([]ReferenceCell, grid.Columns*grid.Rows),
	}
	for i := range report.Cells {
		report.Cells[i].Row = i / grid.Columns
		report.Cells[i].Column = i % grid.Columns
		report.Cells[i].Bounds = grid.CellBounds(report.Cells[i].Row, report.Cells[i].Column)
	}
	if cfg.Layer == "hydrorivers" {
		return makeVectorReferenceEnvelope(path, cfg, report)
	}
	if !SupportedHydroRasterLayer(cfg.Layer) {
		return ReferenceEnvelope{}, fmt.Errorf("realgeom: layer %q unsupported", cfg.Layer)
	}
	report.QualityFlags = append(report.QualityFlags, "raster_source_pixel_center_sampling")
	acc := make([]referenceAccumCell, len(report.Cells))
	var source GeoTIFFScanInfo
	var dx, dy float64
	info, err := ScanGeoTIFF(path, GeoTIFFScanOptions{Crop: &cfg.Bounds, SampleStride: cfg.SampleStride, StripeRows: cfg.StripeRows,
		OnInfo: func(i GeoTIFFScanInfo) error {
			source = i
			dx = (i.SourceBounds.East - i.SourceBounds.West) / float64(i.SourceWidth)
			dy = (i.SourceBounds.North - i.SourceBounds.South) / float64(i.SourceHeight)
			return nil
		}},
		func(x, y int, v float64) error {
			lon := source.SourceBounds.West + (float64(x)+.5)*dx
			lat := source.SourceBounds.North - (float64(y)+.5)*dy
			at, ok := grid.CellAt(lon, lat)
			if !ok {
				return fmt.Errorf("realgeom: raster sample outside locked grid: %.9f, %.9f", lon, lat)
			}
			c := &acc[at]
			nodata, _ := DefaultLayerNoData(cfg.Layer)
			if v == nodata {
				c.nodata++
				return nil
			}
			if math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v {
				return fmt.Errorf("realgeom: invalid raster code %v", v)
			}
			switch cfg.Layer {
			case LayerJRCOccurrence:
				if v < 0 || v > 100 {
					return fmt.Errorf("realgeom: occurrence %v outside 0-100", v)
				}
				if v > 0 {
					c.positive++
				}
				if v >= 90 {
					c.frequent++
				}
			case LayerJRCSeasonality:
				if v < 0 || v > 12 {
					return fmt.Errorf("realgeom: seasonality %v outside 0-12", v)
				}
				if v > 0 {
					c.positive++
				}
				if v > 0 && v < 12 {
					c.partYear++
				}
				if v == 12 {
					c.fullYear++
				}
			case LayerHydroDirection:
				if v != 0 {
					k := 0
					switch int(v) {
					case 1:
						k = 0
					case 2:
						k = 1
					case 4:
						k = 2
					case 8:
						k = 3
					case 16:
						k = 4
					case 32:
						k = 5
					case 64:
						k = 6
					case 128:
						k = 7
					default:
						return fmt.Errorf("realgeom: unsupported D8 code %v", v)
					}
					c.directions[k]++
				}
			case LayerHydroAccumulation:
				if v < 0 || v >= 4294967295 {
					return fmt.Errorf("realgeom: invalid accumulation %v", v)
				}
				c.accumulator = append(c.accumulator, v)
			}
			c.valid++
			return nil
		})
	if err != nil {
		return ReferenceEnvelope{}, err
	}
	report.Source.Raster = &info
	for i, a := range acc {
		c := &report.Cells[i]
		c.ValidSamples = a.valid
		c.NoDataSamples = a.nodata
		report.ValidSamples += a.valid
		report.NoDataSamples += a.nodata
		if a.valid < cfg.MinValidPerBlock {
			continue
		}
		c.Metrics = make(map[string]float64)
		switch cfg.Layer {
		case LayerJRCOccurrence:
			c.Metrics["jrc_occurrence_positive_fraction"] = float64(a.positive) / float64(a.valid)
			c.Metrics["jrc_occurrence_ge90_fraction"] = float64(a.frequent) / float64(a.valid)
		case LayerJRCSeasonality:
			c.Metrics["jrc_2024_water_any_month_fraction"] = float64(a.positive) / float64(a.valid)
			c.Metrics["jrc_2024_water_1to11_months_fraction"] = float64(a.partYear) / float64(a.valid)
			c.Metrics["jrc_2024_water_12_months_fraction"] = float64(a.fullYear) / float64(a.valid)
		case LayerHydroDirection:
			n := 0
			for _, count := range a.directions {
				n += count
			}
			if n > 0 {
				entropy := 0.0
				for _, count := range a.directions {
					if count > 0 {
						p := float64(count) / float64(n)
						entropy -= p * math.Log(p)
					}
				}
				c.Metrics["hydrosheds_d8_direction_entropy"] = entropy / math.Log(8)
			}
		case LayerHydroAccumulation:
			if len(a.accumulator) > 0 {
				c.Metrics["hydrosheds_accumulation_p90_cells"] = summarize(a.accumulator).P90
			}
		}
	}
	specs := map[string]string{}
	switch cfg.Layer {
	case LayerJRCOccurrence:
		specs["jrc_occurrence_positive_fraction"] = "fraction"
		specs["jrc_occurrence_ge90_fraction"] = "fraction"
		report.QualityFlags = append(report.QualityFlags, "occurrence_is_multi_year_1984_2024", "ge90_is_proxy_not_permanent_class", "sampled_pixel_fractions_not_geodesic_area")
	case LayerJRCSeasonality:
		specs["jrc_2024_water_any_month_fraction"] = "fraction"
		specs["jrc_2024_water_1to11_months_fraction"] = "fraction"
		specs["jrc_2024_water_12_months_fraction"] = "fraction"
		report.QualityFlags = append(report.QualityFlags, "seasonality_is_single_year_2024_not_1984_2024", "sampled_pixel_fractions_not_geodesic_area")
	case LayerHydroDirection:
		specs["hydrosheds_d8_direction_entropy"] = "normalized_entropy"
		report.QualityFlags = append(report.QualityFlags, "direction_entropy_diagnostic_only")
	case LayerHydroAccumulation:
		specs["hydrosheds_accumulation_p90_cells"] = "upstream_source_cell_count"
		report.QualityFlags = append(report.QualityFlags, "acc_cell_count_not_upstream_km2", "acc_not_scale_safe_for_geoworld")
	}
	for name, unit := range specs {
		band := CreateEnvelopeBand(name, unit, report.Cells, cfg.MinValidPerBlock)
		if cfg.Layer == LayerHydroAccumulation || cfg.Layer == LayerHydroDirection {
			band.Status = "diagnostic_only_not_for_geoworld_comparison"
		}
		report.Metrics = append(report.Metrics, band)
	}
	sortEnvelopeBands(report.Metrics)
	return report, nil
}

func CreateEnvelopeBand(name, unit string, cells []ReferenceCell, minValid int) EnvelopeBand {
	out := EnvelopeBand{Name: name, Unit: unit, Statistic: "p10_p50_p90_across_equal_geographic_bbox_cells", Status: "candidate_not_calibrated"}
	vals := []float64{}
	for _, c := range cells {
		if c.ValidSamples < minValid {
			out.ExcludedBlocks++
			continue
		}
		v, ok := c.Metrics[name]
		if !ok || math.IsNaN(v) || math.IsInf(v, 0) {
			out.ExcludedBlocks++
			continue
		}
		vals = append(vals, v)
	}
	out.SampleBlocks = len(vals)
	if len(vals) > 0 {
		out.Percentiles = summarize(vals)
	}
	return out
}

func sortEnvelopeBands(x []EnvelopeBand) {
	// Small metric set: stable on-disk order for reproducible JSON.
	for i := 1; i < len(x); i++ {
		for j := i; j > 0 && x[j].Name < x[j-1].Name; j-- {
			x[j], x[j-1] = x[j-1], x[j]
		}
	}
}
