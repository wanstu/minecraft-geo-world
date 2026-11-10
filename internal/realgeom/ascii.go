package realgeom

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

func ReadESRIASCII(path string) (Raster, error) {
	f, err := os.Open(path)
	if err != nil {
		return Raster{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 64*1024*1024)
	header := make(map[string]float64)
	values := make([]float64, 0)
	dataRows := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if dataRows == 0 && len(fields) == 2 {
			key := strings.ToLower(fields[0])
			switch key {
			case "ncols", "nrows", "xllcorner", "yllcorner", "xllcenter", "yllcenter", "cellsize", "nodata_value":
				value, err := strconv.ParseFloat(fields[1], 64)
				if err != nil {
					return Raster{}, fmt.Errorf("realgeom: invalid ASCII header %s: %w", key, err)
				}
				header[key] = value
				continue
			}
		}

		ncols := int(header["ncols"])
		nrows := int(header["nrows"])
		if ncols < 2 || nrows < 2 || header["cellsize"] <= 0 {
			return Raster{}, errors.New("realgeom: ASCII data started before required header was complete")
		}
		if len(fields) != ncols {
			return Raster{}, fmt.Errorf(
				"realgeom: ASCII row %d has %d values, want %d",
				dataRows, len(fields), ncols,
			)
		}
		for _, field := range fields {
			v, err := strconv.ParseFloat(field, 64)
			if err != nil {
				return Raster{}, fmt.Errorf("realgeom: invalid ASCII value %q: %w", field, err)
			}
			values = append(values, v)
		}
		dataRows++
	}
	if err := scanner.Err(); err != nil {
		return Raster{}, err
	}

	ncols := int(header["ncols"])
	nrows := int(header["nrows"])
	cell := header["cellsize"]
	if ncols < 2 || nrows < 2 || cell <= 0 {
		return Raster{}, errors.New("realgeom: ASCII grid requires ncols, nrows and positive cellsize")
	}
	if dataRows != nrows {
		return Raster{}, fmt.Errorf("realgeom: ASCII grid has %d data rows, want %d", dataRows, nrows)
	}

	xll, xCorner := header["xllcorner"]
	if !xCorner {
		var ok bool
		xll, ok = header["xllcenter"]
		if !ok {
			return Raster{}, errors.New("realgeom: ASCII grid missing xllcorner/xllcenter")
		}
		xll -= cell / 2
	}
	yll, yCorner := header["yllcorner"]
	if !yCorner {
		var ok bool
		yll, ok = header["yllcenter"]
		if !ok {
			return Raster{}, errors.New("realgeom: ASCII grid missing yllcorner/yllcenter")
		}
		yll -= cell / 2
	}
	noData := math.NaN()
	if v, ok := header["nodata_value"]; ok {
		noData = v
	}
	r := Raster{
		Source: path, Width: ncols, Height: nrows,
		West: xll, South: yll,
		East:   xll + float64(ncols)*cell,
		North:  yll + float64(nrows)*cell,
		Values: values, NoData: noData,
		CellDegX: cell, CellDegY: cell,
	}
	return r, r.Validate()
}
