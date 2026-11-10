package realgeom

import (
	"fmt"
	"math"
	"strings"

	"github.com/tingold/gocog"
)

type PixelWindow struct{ X, Y, Width, Height int }

type GeoTIFFScanOptions struct {
	Crop         *Bounds
	StripeRows   int
	NoData       float64
	HasNoData    bool
	SampleStride int
}

type GeoTIFFScanInfo struct {
	Source       string      `json:"source"`
	SourceWidth  int         `json:"sourceWidth"`
	SourceHeight int         `json:"sourceHeight"`
	SourceBounds Bounds      `json:"sourceBounds"`
	Window       PixelWindow `json:"pixelWindow"`
	Bounds       Bounds      `json:"bounds"`
}

// WindowForBounds includes pixel centers in the requested W/S/E/N rectangle.
// Pixel windows are half-open and keep the source grid alignment.
func WindowForBounds(source Bounds, width, height int, crop *Bounds) (PixelWindow, error) {
	if !validBounds(source) || width < 1 || height < 1 {
		return PixelWindow{}, fmt.Errorf("realgeom: invalid GeoTIFF geographic geometry")
	}
	if crop == nil {
		return PixelWindow{0, 0, width, height}, nil
	}
	if !validBounds(*crop) {
		return PixelWindow{}, fmt.Errorf("realgeom: invalid crop bbox")
	}
	dx := (source.East - source.West) / float64(width)
	dy := (source.North - source.South) / float64(height)
	clamp := func(v, limit int) int {
		if v < 0 {
			return 0
		}
		if v > limit {
			return limit
		}
		return v
	}
	x0 := clamp(int(math.Ceil((crop.West-source.West)/dx-0.5)), width)
	x1 := clamp(int(math.Ceil((crop.East-source.West)/dx-0.5)), width)
	y0 := clamp(int(math.Ceil((source.North-crop.North)/dy-0.5)), height)
	y1 := clamp(int(math.Ceil((source.North-crop.South)/dy-0.5)), height)
	if x1 <= x0 || y1 <= y0 {
		return PixelWindow{}, fmt.Errorf("realgeom: crop does not intersect any source pixel centers")
	}
	return PixelWindow{X: x0, Y: y0, Width: x1 - x0, Height: y1 - y0}, nil
}

// ScanGeoTIFF reads only requested pixels, by full-resolution stripes.
// Callback coordinates are absolute to the source image and not stripe-local.
func ScanGeoTIFF(path string, opts GeoTIFFScanOptions, visit func(x, y int, v float64) error) (GeoTIFFScanInfo, error) {
	cog, err := gocog.Open(path, nil)
	if err != nil {
		return GeoTIFFScanInfo{}, fmt.Errorf("realgeom: open GeoTIFF: %w", err)
	}
	if !strings.EqualFold(cog.CRS(), "EPSG:4326") || cog.BandCount() != 1 {
		return GeoTIFFScanInfo{}, fmt.Errorf("realgeom: expected one-band EPSG:4326 GeoTIFF, got %s / %d bands", cog.CRS(), cog.BandCount())
	}
	b := cog.Bounds()
	source := Bounds{West: b.Min[0], South: b.Min[1], East: b.Max[0], North: b.Max[1]}
	w, h := cog.Width(), cog.Height()
	window, err := WindowForBounds(source, w, h, opts.Crop)
	if err != nil {
		return GeoTIFFScanInfo{}, err
	}
	dx := (source.East - source.West) / float64(w)
	dy := (source.North - source.South) / float64(h)
	actual := Bounds{
		West:  source.West + float64(window.X)*dx,
		East:  source.West + float64(window.X+window.Width)*dx,
		North: source.North - float64(window.Y)*dy,
		South: source.North - float64(window.Y+window.Height)*dy,
	}
	info := GeoTIFFScanInfo{Source: path, SourceWidth: w, SourceHeight: h, SourceBounds: source, Window: window, Bounds: actual}
	rows := opts.StripeRows
	if rows <= 0 {
		rows = 128
	}
	if rows > window.Height {
		rows = window.Height
	}
	// This reader can select an overview for a large requested window. Keep
	// the read window below 1% of main-image height and assert its dimensions.
	if cog.OverviewCount() > 0 {
		maxRows := (h - 1) / 100
		if maxRows < 2 {
			return info, fmt.Errorf("realgeom: cannot guarantee main-resolution reads")
		}
		if rows > maxRows {
			rows = maxRows
		}
	}
	stride := opts.SampleStride
	if stride < 1 {
		stride = 1
	}
	for y := window.Y; y < window.Y+window.Height; {
		height := rows
		if height > window.Y+window.Height-y {
			height = window.Y + window.Height - y
		}
		if height < 1 {
			return info, fmt.Errorf("realgeom: empty stripe")
		}
		data, err := cog.ReadWindow(gocog.Rectangle{X: window.X, Y: y, Width: window.Width, Height: height})
		if err != nil {
			return info, fmt.Errorf("realgeom: read %s source rows %d:%d: %w", path, y, y+height, err)
		}
		if data.Bands != 1 || data.Width != window.Width || data.Height != height {
			return info, fmt.Errorf("realgeom: GeoTIFF unexpectedly switched overview/resolution: wanted %dx%d, got %dx%dx%d", window.Width, height, data.Width, data.Height, data.Bands)
		}
		for yy := 0; yy < height; yy++ {
			sy := y + yy
			if sy%stride != 0 {
				continue
			}
			for xx := 0; xx < window.Width; xx++ {
				sx := window.X + xx
				if sx%stride != 0 {
					continue
				}
				v, err := geoTIFFValue(cog.DataType(), data.AtUnchecked(0, xx, yy))
				if err != nil {
					return info, err
				}
				if opts.HasNoData && v == opts.NoData {
					continue
				}
				if math.IsNaN(v) || math.IsInf(v, 0) {
					continue
				}
				if err := visit(sx, sy, v); err != nil {
					return info, err
				}
			}
		}
		y += height
	}
	return info, nil
}
