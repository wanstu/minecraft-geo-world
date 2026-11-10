# gocog — local offline-extractor compatibility patch

Upstream: `github.com/tingold/gocog` at `v0.0.0-20251202163215-f108ab4d8e26` (license retained).

Only these files differ from the pinned upstream source:

- `geotiff.go`: load TIFF GeoKeyDirectory, ModelPixelScale, ModelTiepoint, and ModelTransformation tags when their out-of-line values have not been read yet. Needed to recover true georeferencing for HydroSHEDS 15-second Africa source files instead of reporting zero bounds.
- `cog.go`: support TIFF ZSTD compression ID 50000 (observed in the JRC v1.5 occurrence/seasonality export). TIFF LZW requires MSB-first decoding; reject incorrectly decoded *short* output before trying alternative bit order.

These patches are applied only through `go.mod replace` for the Go-based development/offline extractor. They do not introduce a GIS parser or ZSTD dependency into Java Minecraft runtime.

Original module code, copyright notices and license are retained. Do not redistribute the original downloaded HydroSHEDS/JRC/HydroRIVERS rasters with this repo.

Validated against actual `hyd_af_dir_15s.tif`, `hyd_af_acc_15s.tif`, `occurrence.tif`, and `seasonality.tif` on 2026-11-10.
