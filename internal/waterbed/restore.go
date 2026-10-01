package waterbed

import (
	"errors"
	"math"

	"geoworld/internal/terrain"
	"geoworld/internal/world"
)

// Restore rebuilds a Runtime-ready Waterbed from persisted local river/lake
// geometry without rerunning Hydrology/Reconciliation. Spatial buckets are
// intentionally derived at load time and therefore are not part of the file
// format.
func Restore(base *terrain.Model, cfg Config, rivers []RiverSegment, lakes []LakeNode) (*Model, error) {
	if base == nil {
		return nil, errors.New("waterbed: restore requires base terrain")
	}
	if _, err := base.Geometry.SquareCellSize(); err != nil {
		return nil, err
	}
	cfg, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}
	n, err := world.NodeCount(base.Geometry.Width, base.Geometry.Height)
	if err != nil {
		return nil, err
	}

	m := &Model{
		Config:       cfg,
		Geometry:     base.Geometry,
		Base:         base,
		Rivers:       make([]RiverSegment, len(rivers)),
		Lakes:        append([]LakeNode(nil), lakes...),
		riverBuckets: make([][]int, n),
		lakeBuckets:  make([][]int, n),
	}

	for i, in := range rivers {
		if in.From < 0 || in.From >= n || in.To < 0 || in.To >= n || in.From == in.To {
			return nil, errors.New("waterbed: persisted river edge index is invalid")
		}
		if !finite(in.X0) || !finite(in.Z0) || !finite(in.X1) || !finite(in.Z1) ||
			!finite(in.Bed0) || !finite(in.Bed1) ||
			!finite(in.HalfWidth) || in.HalfWidth <= 0 {
			return nil, errors.New("waterbed: persisted river geometry is invalid")
		}
		if in.ReceiverSurfaceDefined &&
			(!finite(in.ReceiverSurface) || in.ReceiverSurface < 0 || in.ReceiverSurface > 1) {
			return nil, errors.New("waterbed: persisted river receiver surface is invalid")
		}
		if len(in.Profile) == 0 {
			return nil, errors.New("waterbed: persisted river profile is empty")
		}
		profile := append([]float64(nil), in.Profile...)
		for j, v := range profile {
			if !finite(v) || v < 0 || v > 1 {
				return nil, errors.New("waterbed: persisted river profile contains invalid height")
			}
			if j > 0 && v > profile[j-1]+1e-12 {
				return nil, errors.New("waterbed: persisted river profile is not monotonic")
			}
		}

		x0, z0 := base.Geometry.NodePosition(in.From%base.Geometry.Width, in.From/base.Geometry.Width)
		x1, z1 := base.Geometry.NodePosition(in.To%base.Geometry.Width, in.To/base.Geometry.Width)
		if math.Abs(in.X0-x0) > 1e-9 || math.Abs(in.Z0-z0) > 1e-9 ||
			math.Abs(in.X1-x1) > 1e-9 || math.Abs(in.Z1-z1) > 1e-9 {
			return nil, errors.New("waterbed: persisted river endpoints do not match geometry")
		}
		in.Profile = profile
		m.Rivers[i] = in
		m.indexRiver(i, in)
	}

	for i, lake := range m.Lakes {
		if lake.Index < 0 || lake.Index >= n || lake.LakeID <= 0 ||
			!finite(lake.X) || !finite(lake.Z) ||
			!finite(lake.BedCeiling) || !finite(lake.InnerRadius) ||
			!finite(lake.OuterRadius) || !finite(lake.WaterSurface) ||
			lake.InnerRadius < 0 || lake.OuterRadius <= lake.InnerRadius ||
			lake.BedCeiling < 0 || lake.BedCeiling > 1 ||
			lake.WaterSurface <= 0 || lake.WaterSurface > 1 ||
			lake.BedCeiling >= lake.WaterSurface {
			return nil, errors.New("waterbed: persisted lake geometry is invalid")
		}
		x, z := base.Geometry.NodePosition(lake.Index%base.Geometry.Width, lake.Index/base.Geometry.Width)
		if math.Abs(lake.X-x) > 1e-9 || math.Abs(lake.Z-z) > 1e-9 {
			return nil, errors.New("waterbed: persisted lake center does not match geometry")
		}
		m.indexLake(i, lake)
	}
	return m, nil
}
