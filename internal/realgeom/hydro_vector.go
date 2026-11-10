package realgeom

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type HydroVectorReport struct {
	SchemaVersion           int          `json:"schemaVersion"`
	RegionID                string       `json:"regionId"`
	DatasetID               string       `json:"datasetId"`
	DatasetVersion          string       `json:"datasetVersion"`
	Layer                   string       `json:"layer"`
	Input                   string       `json:"input"`
	Crop                    Bounds       `json:"crop"`
	ReadReaches             int          `json:"readReaches"`
	IntersectingReaches     int          `json:"intersectingReaches"`
	ClippedSegments         int          `json:"clippedSegments"`
	ClippedLengthKM         float64      `json:"clippedLengthKm"`
	RegionAreaKM2           float64      `json:"regionAreaKm2"`
	DrainageDensityKMPerKM2 float64      `json:"drainageDensityKmPerKm2"`
	LocalConfluences        int          `json:"localConfluences"`
	StrahlerOrder           *Percentiles `json:"strahlerOrder,omitempty"`
	QualityFlags            []string     `json:"qualityFlags"`
}

type dbfField struct{ offset, length int }
type dbfTable struct {
	file                       *os.File
	recordLength, headerLength int
	fields                     map[string]dbfField
}

func openDBF(path string) (*dbfTable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var hdr [32]byte
	if _, err = io.ReadFull(f, hdr[:]); err != nil {
		f.Close()
		return nil, err
	}
	headerLen := int(binary.LittleEndian.Uint16(hdr[8:10]))
	recordLen := int(binary.LittleEndian.Uint16(hdr[10:12]))
	if headerLen < 33 || recordLen < 2 {
		f.Close()
		return nil, fmt.Errorf("realgeom: invalid DBF header")
	}
	fields := make(map[string]dbfField)
	off := 1
	for pos := 32; pos+32 <= headerLen; pos += 32 {
		var desc [32]byte
		if _, err = f.ReadAt(desc[:], int64(pos)); err != nil {
			f.Close()
			return nil, err
		}
		if desc[0] == 13 {
			break
		}
		name := strings.TrimRight(string(desc[:11]), "\x00 ")
		size := int(desc[16])
		if size <= 0 || off+size > recordLen {
			f.Close()
			return nil, fmt.Errorf("realgeom: malformed DBF field %q", name)
		}
		fields[name] = dbfField{off, size}
		off += size
	}
	for _, k := range []string{"HYRIV_ID", "NEXT_DOWN", "ORD_STRA"} {
		if _, ok := fields[k]; !ok {
			f.Close()
			return nil, fmt.Errorf("realgeom: missing HydroRIVERS DBF field %s", k)
		}
	}
	return &dbfTable{file: f, recordLength: recordLen, headerLength: headerLen, fields: fields}, nil
}

func (d *dbfTable) numbers(index int) (int64, int64, int64, error) {
	row := make([]byte, d.recordLength)
	if _, err := d.file.ReadAt(row, int64(d.headerLength)+int64(index)*int64(d.recordLength)); err != nil {
		return 0, 0, 0, err
	}
	if row[0] == 0x2a {
		return 0, 0, 0, fmt.Errorf("realgeom: DBF record %d is deleted", index)
	}
	number := func(field string) (int64, error) {
		f := d.fields[field]
		return strconv.ParseInt(strings.TrimSpace(string(row[f.offset:f.offset+f.length])), 10, 64)
	}
	id, err := number("HYRIV_ID")
	if err != nil {
		return 0, 0, 0, err
	}
	next, err := number("NEXT_DOWN")
	if err != nil {
		return 0, 0, 0, err
	}
	order, err := number("ORD_STRA")
	return id, next, order, err
}

// ExtractHydroRIVERS reads a source .shp and its matching .dbf, crops each
// polyline segment to the registry bbox, and reports length inside the bbox.
// It never substitutes a whole-feature length for a clipped length.
func ExtractHydroRIVERS(path string, regionID, version string, crop Bounds) (HydroVectorReport, error) {
	r := HydroVectorReport{SchemaVersion: 1, RegionID: regionID, DatasetID: "HydroRIVERS", DatasetVersion: version,
		Layer: "hydrorivers", Input: path, Crop: crop, QualityFlags: []string{
			"source_polyline_segments_clipped_to_registry_bbox",
			"density_is_vector_length_per_bbox_area",
			"local_confluences_exclude_upstream_outside_bbox",
		},
	}
	if !validBounds(crop) {
		return r, fmt.Errorf("realgeom: invalid bbox")
	}
	if strings.ToLower(filepath.Ext(path)) != ".shp" {
		return r, fmt.Errorf("realgeom: HydroRIVERS expects .shp")
	}
	file, err := os.Open(path)
	if err != nil {
		return r, err
	}
	defer file.Close()
	table, err := openDBF(strings.TrimSuffix(path, filepath.Ext(path)) + ".dbf")
	if err != nil {
		return r, err
	}
	defer table.file.Close()
	var header [100]byte
	if _, err = io.ReadFull(file, header[:]); err != nil {
		return r, err
	}
	if binary.BigEndian.Uint32(header[:4]) != 9994 {
		return r, fmt.Errorf("realgeom: invalid shapefile magic")
	}
	shapeType := binary.LittleEndian.Uint32(header[32:36])
	if shapeType != 3 && shapeType != 13 {
		return r, fmt.Errorf("realgeom: unsupported polyline type %d", shapeType)
	}
	downstream := map[int64]int64{}
	orders := []float64{}
	for idx := 0; ; idx++ {
		var rh [8]byte
		_, err := io.ReadFull(file, rh[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return r, fmt.Errorf("realgeom: record header %d: %w", idx, err)
		}
		contentBytes := int(binary.BigEndian.Uint32(rh[4:8])) * 2
		if contentBytes < 4 || contentBytes > 64<<20 {
			return r, fmt.Errorf("realgeom: invalid feature length %d", contentBytes)
		}
		r.ReadReaches++
		if contentBytes < 44 {
			if _, err = file.Seek(int64(contentBytes), io.SeekCurrent); err != nil {
				return r, err
			}
			continue
		}
		var head [44]byte
		if _, err = io.ReadFull(file, head[:]); err != nil {
			return r, err
		}
		typ := binary.LittleEndian.Uint32(head[0:4])
		if typ == 0 {
			if _, err = file.Seek(int64(contentBytes-44), io.SeekCurrent); err != nil {
				return r, err
			}
			continue
		}
		if typ != 3 && typ != 13 {
			return r, fmt.Errorf("realgeom: unexpected geometry type %d", typ)
		}
		box := Bounds{West: math.Float64frombits(binary.LittleEndian.Uint64(head[4:12])),
			South: math.Float64frombits(binary.LittleEndian.Uint64(head[12:20])),
			East:  math.Float64frombits(binary.LittleEndian.Uint64(head[20:28])),
			North: math.Float64frombits(binary.LittleEndian.Uint64(head[28:36]))}
		parts := int(binary.LittleEndian.Uint32(head[36:40]))
		points := int(binary.LittleEndian.Uint32(head[40:44]))
		remaining := contentBytes - 44
		if box.East < crop.West || box.West > crop.East || box.North < crop.South || box.South > crop.North {
			if _, err = file.Seek(int64(remaining), io.SeekCurrent); err != nil {
				return r, err
			}
			continue
		}
		needed := int64(parts)*4 + int64(points)*16
		if parts < 1 || points < 2 || needed > int64(remaining) {
			return r, fmt.Errorf("realgeom: malformed record %d", idx)
		}
		body := make([]byte, remaining)
		if _, err = io.ReadFull(file, body); err != nil {
			return r, err
		}
		starts := make([]int, parts+1)
		for p := 0; p < parts; p++ {
			starts[p] = int(binary.LittleEndian.Uint32(body[p*4:]))
			if starts[p] < 0 || starts[p] >= points || (p > 0 && starts[p] <= starts[p-1]) {
				return r, fmt.Errorf("realgeom: invalid part index at record %d", idx)
			}
		}
		starts[parts] = points
		xypoint := func(i int) (float64, float64) {
			off := parts*4 + i*16
			return math.Float64frombits(binary.LittleEndian.Uint64(body[off:])),
				math.Float64frombits(binary.LittleEndian.Uint64(body[off+8:]))
		}
		clippedLength := 0.0
		for p := 0; p < parts; p++ {
			for i := starts[p] + 1; i < starts[p+1]; i++ {
				x1, y1 := xypoint(i - 1)
				x2, y2 := xypoint(i)
				a, b, c, d, ok := clipLine(x1, y1, x2, y2, crop)
				if !ok {
					continue
				}
				length := haversineKm(a, b, c, d)
				if length > 0 {
					clippedLength += length
					r.ClippedSegments++
				}
			}
		}
		if clippedLength <= 0 {
			continue
		}
		id, next, order, err := table.numbers(idx)
		if err != nil {
			return r, fmt.Errorf("realgeom: feature %d DBF: %w", idx, err)
		}
		downstream[id] = next
		orders = append(orders, float64(order))
		r.IntersectingReaches++
		r.ClippedLengthKM += clippedLength
	}
	incoming := map[int64]int{}
	for _, next := range downstream {
		if _, ok := downstream[next]; ok {
			incoming[next]++
		}
	}
	for _, n := range incoming {
		if n >= 2 {
			r.LocalConfluences++
		}
	}
	if len(orders) > 0 {
		p := summarize(orders)
		r.StrahlerOrder = &p
	}
	const earthRadius = 6371.0088
	r.RegionAreaKM2 = earthRadius * earthRadius * (crop.East - crop.West) * math.Pi / 180 *
		(math.Sin(crop.North*math.Pi/180) - math.Sin(crop.South*math.Pi/180))
	if r.RegionAreaKM2 <= 0 {
		return r, fmt.Errorf("realgeom: invalid crop area")
	}
	r.DrainageDensityKMPerKM2 = r.ClippedLengthKM / r.RegionAreaKM2
	return r, nil
}

// Liang-Barsky segment clipping in geographic lon/lat, then distance is
// measured along the clipped polyline with a great-circle approximation.
func clipLine(x0, y0, x1, y1 float64, b Bounds) (float64, float64, float64, float64, bool) {
	dx, dy := x1-x0, y1-y0
	p := [4]float64{-dx, dx, -dy, dy}
	q := [4]float64{x0 - b.West, b.East - x0, y0 - b.South, b.North - y0}
	t0, t1 := 0.0, 1.0
	for i := 0; i < 4; i++ {
		if p[i] == 0 {
			if q[i] < 0 {
				return 0, 0, 0, 0, false
			}
			continue
		}
		t := q[i] / p[i]
		if p[i] < 0 {
			if t > t1 {
				return 0, 0, 0, 0, false
			}
			if t > t0 {
				t0 = t
			}
		}
		if p[i] > 0 {
			if t < t0 {
				return 0, 0, 0, 0, false
			}
			if t < t1 {
				t1 = t
			}
		}
	}
	if t0 >= t1 {
		return 0, 0, 0, 0, false
	}
	return x0 + t0*dx, y0 + t0*dy, x0 + t1*dx, y0 + t1*dy, true
}

func haversineKm(lon1, lat1, lon2, lat2 float64) float64 {
	const rad = math.Pi / 180
	dlat := (lat2 - lat1) * rad
	dlon := (lon2 - lon1) * rad
	a := math.Pow(math.Sin(dlat/2), 2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Pow(math.Sin(dlon/2), 2)
	return 2 * 6371.0088 * math.Asin(math.Min(1, math.Sqrt(a)))
}
