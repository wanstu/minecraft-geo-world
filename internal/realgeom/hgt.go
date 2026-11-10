package realgeom

import (
	"archive/zip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var hgtTilePattern = regexp.MustCompile(`(?i)([ns])(\d{2})([ew])(\d{3})`)

func ReadHGT(path string) (Raster, error) {
	f, err := os.Open(path)
	if err != nil {
		return Raster{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return Raster{}, err
	}
	return readHGTReader(f, info.Size(), path, filepath.Base(path))
}

func ReadHGTZip(path string) (Raster, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return Raster{}, err
	}
	defer zr.Close()
	for _, entry := range zr.File {
		if !strings.EqualFold(filepath.Ext(entry.Name), ".hgt") {
			continue
		}
		rc, err := entry.Open()
		if err != nil {
			return Raster{}, err
		}
		defer rc.Close()
		return readHGTReader(
			rc,
			int64(entry.UncompressedSize64),
			path+"!"+entry.Name,
			filepath.Base(entry.Name),
		)
	}
	return Raster{}, errors.New("realgeom: zip contains no .hgt tile")
}

func readHGTReader(
	reader io.Reader,
	size int64,
	source string,
	tileName string,
) (Raster, error) {
	if size <= 0 || size%2 != 0 {
		return Raster{}, errors.New("realgeom: HGT file size must be a positive even number")
	}
	samples := size / 2
	side := int(math.Round(math.Sqrt(float64(samples))))
	if int64(side*side) != samples || side < 2 {
		return Raster{}, fmt.Errorf("realgeom: HGT sample count %d is not a square grid", samples)
	}

	west, south, err := parseHGTTileOrigin(tileName)
	if err != nil {
		return Raster{}, err
	}
	values := make([]float64, side*side)
	var raw int16
	for i := range values {
		if err := binary.Read(reader, binary.BigEndian, &raw); err != nil {
			if err == io.EOF {
				return Raster{}, errors.New("realgeom: truncated HGT file")
			}
			return Raster{}, err
		}
		values[i] = float64(raw)
	}
	cell := 1.0 / float64(side-1)
	r := Raster{
		Source: source, Width: side, Height: side,
		West: west, South: south, East: west + 1, North: south + 1,
		Values: values, NoData: -32768, CellDegX: cell, CellDegY: cell,
	}
	return r, r.Validate()
}

func parseHGTTileOrigin(name string) (west, south float64, err error) {
	match := hgtTilePattern.FindStringSubmatch(strings.TrimSuffix(name, filepath.Ext(name)))
	if len(match) != 5 {
		return 0, 0, fmt.Errorf("realgeom: cannot infer HGT tile origin from %q", name)
	}
	lat, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, err
	}
	lon, err := strconv.Atoi(match[4])
	if err != nil {
		return 0, 0, err
	}
	south = float64(lat)
	if strings.EqualFold(match[1], "s") {
		south = -south
	}
	west = float64(lon)
	if strings.EqualFold(match[3], "w") {
		west = -west
	}
	return west, south, nil
}
