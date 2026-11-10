package realgeom

import (
	"encoding/json"
	"fmt"
	"os"
)

// LoadRegionBounds resolves one locked sampling window from the R6 registry.
func LoadRegionBounds(path, regionID string) (Bounds, error) {
	if regionID == "" {
		return Bounds{}, fmt.Errorf("realgeom: --region is required for registry cropping")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Bounds{}, err
	}
	var registry struct {
		CRS     string `json:"crs"`
		Regions []struct {
			ID   string    `json:"id"`
			BBox []float64 `json:"bbox"`
		} `json:"regions"`
	}
	if err := json.Unmarshal(data, &registry); err != nil {
		return Bounds{}, fmt.Errorf("realgeom: parse registry: %w", err)
	}
	if registry.CRS != "EPSG:4326" {
		return Bounds{}, fmt.Errorf("realgeom: expected EPSG:4326 registry, got %q", registry.CRS)
	}
	for _, r := range registry.Regions {
		if r.ID != regionID {
			continue
		}
		if len(r.BBox) != 4 {
			return Bounds{}, fmt.Errorf("realgeom: invalid bbox for %s", regionID)
		}
		b := Bounds{West: r.BBox[0], South: r.BBox[1], East: r.BBox[2], North: r.BBox[3]}
		if !validBounds(b) {
			return Bounds{}, fmt.Errorf("realgeom: invalid geographic bbox for %s", regionID)
		}
		return b, nil
	}
	return Bounds{}, fmt.Errorf("realgeom: region %q not found in registry", regionID)
}

func validBounds(b Bounds) bool {
	return b.West >= -180 && b.East <= 180 && b.South >= -90 && b.North <= 90 &&
		b.West < b.East && b.South < b.North
}
