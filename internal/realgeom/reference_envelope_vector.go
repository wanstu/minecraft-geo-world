package realgeom

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

// makeVectorReferenceEnvelope derives a *spatial distribution*, not a single
// reference target. Every cell receives the true clipped segment length.
func makeVectorReferenceEnvelope(path string, cfg ReferenceExtractConfig, r ReferenceEnvelope) (ReferenceEnvelope, error) {
	if strings.ToLower(filepath.Ext(path)) != ".shp" {
		return ReferenceEnvelope{}, fmt.Errorf("realgeom: expected .shp for hydrorivers")
	}
	dbf := strings.TrimSuffix(path, filepath.Ext(path)) + ".dbf"
	sidecar, err := fingerprint(dbf)
	if err != nil {
		return ReferenceEnvelope{}, err
	}
	r.Source.SidecarSHA256 = sidecar
	r.Source.SampleStride = 0 // not applicable to complete vector geometry
	r.MinValidPerBlock = 0    // vector: empty geographic cells are observed zeros
	v, err := ExtractHydroRIVERSGrid(path, cfg.RegionID, cfg.DatasetVersion, cfg.Bounds, r.Grid)
	if err != nil {
		return ReferenceEnvelope{}, err
	}
	if v.IntersectingReaches == 0 {
		return r, fmt.Errorf("realgeom: no HydroRIVERS features intersect reference bbox")
	}
	lengths := 0.0
	for i, c := range v.GridCells {
		lengths += c.ClippedLengthKM
		r.Cells[i].ValidSamples = c.IntersectingReaches // zero-feature cells are also observed
		r.Cells[i].Metrics = map[string]float64{
			"hydrorivers_vector_density_km_per_km2": c.DrainageDensityKMPerKM2,
		}
	}
	if math.Abs(lengths-v.ClippedLengthKM) > math.Max(1e-6, v.ClippedLengthKM*1e-7) {
		return r, fmt.Errorf("realgeom: vector grid length not conserved: %g != %g", lengths, v.ClippedLengthKM)
	}
	band := CreateEnvelopeBand("hydrorivers_vector_density_km_per_km2", "km_per_km2", r.Cells, 0)
	band.Status = "diagnostic_only_not_for_geoworld_comparison"
	r.Metrics = []EnvelopeBand{band}
	r.ValidSamples = v.IntersectingReaches
	r.QualityFlags = append(r.QualityFlags,
		"vector_clipped_length_conserved_across_grid",
		"zero_feature_grid_cells_are_valid",
		"vector_density_not_channel_width_or_floodplain_width",
		"geodesic_line_length_approximation",
		"vector_cell_area_is_spherical_approximation",
		"dataset_stratum_not_worldwide_model_target",
	)
	return r, nil
}
