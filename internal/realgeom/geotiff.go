package realgeom

import (
	"fmt"
	"math"
	"strings"

	"github.com/tingold/gocog"
)

type GeoTIFFOptions struct {
	NoData     float64
	HasNoData  bool
	StripeRows int
}

func AddGeoTIFF(
	analyzer *Analyzer,
	path string,
	opts GeoTIFFOptions,
) error {
	if analyzer == nil {
		return fmt.Errorf("realgeom: analyzer is required")
	}
	cog, err := gocog.Open(path, nil)
	if err != nil {
		return fmt.Errorf("realgeom: open GeoTIFF: %w", err)
	}
	if !strings.EqualFold(cog.CRS(), "EPSG:4326") {
		return fmt.Errorf(
			"realgeom: GeoTIFF CRS %q is unsupported; expected EPSG:4326",
			cog.CRS(),
		)
	}
	if cog.BandCount() != 1 {
		return fmt.Errorf(
			"realgeom: expected one-band GeoTIFF, got %d bands",
			cog.BandCount(),
		)
	}
	width := cog.Width()
	height := cog.Height()
	if width < 2 || height < 2 {
		return fmt.Errorf("realgeom: GeoTIFF must be at least 2x2")
	}
	bounds := cog.Bounds()
	if bounds.Min[0] < -180.0001 || bounds.Max[0] > 180.0001 ||
		bounds.Min[1] < -90.0001 || bounds.Max[1] > 90.0001 ||
		bounds.Min[0] >= bounds.Max[0] || bounds.Min[1] >= bounds.Max[1] {
		return fmt.Errorf(
			"realgeom: invalid EPSG:4326 bounds: %+v",
			bounds,
		)
	}
	if opts.StripeRows <= 0 {
		opts.StripeRows = 512
	}
	if opts.StripeRows > height {
		opts.StripeRows = height
	}
	if cog.OverviewCount() > 0 {
		maxMainRows := (height - 1) / 100
		if maxMainRows < 2 {
			return fmt.Errorf(
				"realgeom: GeoTIFF height %d is too small to force main-resolution window reads while overviews are present",
				height,
			)
		}
		if opts.StripeRows > maxMainRows {
			opts.StripeRows = maxMainRows
		}
	}

	validSamples := 0
	for y0 := 0; y0 < height; {
		stripeHeight := opts.StripeRows
		remaining := height - y0
		if stripeHeight > remaining {
			stripeHeight = remaining
		}
		if remaining-stripeHeight == 1 {
			if stripeHeight > 2 {
				stripeHeight--
			} else if cog.OverviewCount() == 0 {
				stripeHeight++
			} else {
				return fmt.Errorf(
					"realgeom: cannot split GeoTIFF height %d into main-resolution windows of at least two rows",
					height,
				)
			}
		}
		if stripeHeight < 2 {
			return fmt.Errorf(
				"realgeom: GeoTIFF final stripe has only %d row",
				stripeHeight,
			)
		}
		data, err := cog.ReadWindow(gocog.Rectangle{
			X: 0, Y: y0, Width: width, Height: stripeHeight,
		})
		if err != nil {
			return fmt.Errorf(
				"realgeom: read GeoTIFF rows %d:%d: %w",
				y0, y0+stripeHeight, err,
			)
		}
		if data.Bands != 1 ||
			data.Width != width ||
			data.Height != stripeHeight {
			return fmt.Errorf(
				"realgeom: GeoTIFF window changed resolution: requested=%dx%d got=%dx%dx%d",
				width, stripeHeight,
				data.Width, data.Height, data.Bands,
			)
		}
		values := make([]float64, data.Width*data.Height)
		for y := 0; y < data.Height; y++ {
			for x := 0; x < data.Width; x++ {
				v, err := geoTIFFValue(cog.DataType(), data.AtUnchecked(0, x, y))
				if err != nil {
					return err
				}
				values[y*data.Width+x] = v
			}
		}
		noData := math.NaN()
		if opts.HasNoData {
			noData = opts.NoData
		}
		r := Raster{
			Source: path,
			Width:  data.Width, Height: data.Height,
			West: data.Bounds.Min[0], South: data.Bounds.Min[1],
			East: data.Bounds.Max[0], North: data.Bounds.Max[1],
			Values: values, NoData: noData,
			CellDegX: (data.Bounds.Max[0] - data.Bounds.Min[0]) /
				float64(data.Width),
			CellDegY: (data.Bounds.Max[1] - data.Bounds.Min[1]) /
				float64(data.Height),
		}
		valid, err := analyzer.AddRasterChunk(r)
		if err != nil {
			return err
		}
		validSamples += valid
		y0 += stripeHeight
	}
	analyzer.AddInputSummary(FileSummary{
		Path: path, Width: width, Height: height,
		ValidSamples: validSamples,
		Bounds: Bounds{
			West: bounds.Min[0], South: bounds.Min[1],
			East: bounds.Max[0], North: bounds.Max[1],
		},
	})
	return nil
}

func geoTIFFValue(
	dataType gocog.DataType,
	raw uint64,
) (float64, error) {
	switch dataType {
	case gocog.DTByte:
		return float64(raw), nil
	case gocog.DTSByte:
		return float64(int64(raw)), nil
	case gocog.DTSShort:
		return float64(raw), nil
	case gocog.DTSShortS:
		return float64(int64(raw)), nil
	case gocog.DTSLong:
		return float64(raw), nil
	case gocog.DTSLongS:
		return float64(int64(raw)), nil
	case gocog.DTFloat:
		return float64(math.Float32frombits(uint32(raw))), nil
	case gocog.DTDouble:
		return math.Float64frombits(raw), nil
	default:
		return 0, fmt.Errorf(
			"realgeom: unsupported GeoTIFF sample type %d",
			dataType,
		)
	}
}
