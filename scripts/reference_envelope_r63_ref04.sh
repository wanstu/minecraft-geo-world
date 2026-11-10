#!/usr/bin/env bash
# R6.3a REF-04: offline-only reproduction of real-data spatial envelopes.
# Requires the original source files under .tmp/realdata-r62c; no GIS runtime dependency.
set -euo pipefail
cd "$(dirname "$0")/.."

go_bin="${GO_BIN:-}"
if [[ -z "$go_bin" ]]; then
  if command -v go >/dev/null 2>&1; then
    go_bin="$(command -v go)"
  else
    go_bin=".tmp/realdata-r62c/tools/go/bin/go"
  fi
fi
if [[ ! -x "$go_bin" ]]; then
  printf 'Go executable unavailable: %s (set GO_BIN)\n' "$go_bin" >&2
  exit 2
fi

root=".tmp/realdata-r62c"
out="docs/reference_envelopes/REF-04-zambezi-barotse-floodplain"
region="REF-04-zambezi-barotse-floodplain"
required=(
 "$root/occurrence.tif"
 "$root/seasonality.tif"
 "$root/dir/hyd_af_dir_15s.tif"
 "$root/acc/hyd_af_acc_15s.tif"
 "$root/rivers/HydroRIVERS_v10_af_shp/HydroRIVERS_v10_af.shp"
 "$root/rivers/HydroRIVERS_v10_af_shp/HydroRIVERS_v10_af.dbf"
)
for file in "${required[@]}"; do
 if [[ ! -s "$file" ]]; then
  printf 'Missing reference data: %s\n' "$file" >&2
  exit 2
 fi
done

extract(){
 local layer="$1" input="$2" stride="$3"
 echo "Reference extraction: $layer"
 "$go_bin" run ./cmd/geoworld-reference-envelope \
   --region "$region" --layer "$layer" --input "$input" \
   --grid-cols 8 --grid-rows 10 --min-valid-per-block 32 \
   --sample-stride "$stride" \
   --output "$out/$layer.json"
}
extract jrc-occurrence "$root/occurrence.tif" 32
extract jrc-seasonality "$root/seasonality.tif" 32
extract hydrosheds-direction "$root/dir/hyd_af_dir_15s.tif" 8
extract hydrosheds-accumulation "$root/acc/hyd_af_acc_15s.tif" 8
extract hydrorivers "$root/rivers/HydroRIVERS_v10_af_shp/HydroRIVERS_v10_af.shp" 1

echo "R6.3a candidate envelopes written to $out"
