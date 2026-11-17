# GeoWorld v0.1.7 — R8.2 Slice D Snow Siting Acceptance

Date: 2026-11-18  
Rules: `geoworld-rules-v1.1.25`  
Fixed seed: `-1818694639628404530`

## Scope

Slice D only changes **new-world siting** for `volcanic_crater_snow` and `basin_snow`. It does not redesign their carved geometry and it does not reselect SpecialFeatures from an already-persisted `.geoworld` model.

A snow crater now requires a cold, elevated, dry-ring mountain/foothill/plateau background with sufficient local relief. A snow basin additionally requires a measurable local depression. Invalid legacy snow slots are retained as shadow reservations so later legacy features cannot slide into those positions; missing snow slots are then filled from an independent qualified pool.

## Fixed-seed siting result

The old fixed-seed snow sites included warm or flat backgrounds. Slice D removes those placements and selects:

| ID | Kind | Center | Temp | Center Y | 512-block relief | Basin depth | Dry ring | Mountain weight |
| --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 21 | volcanic_crater_snow | 10944,-6464 | -12.36 °C | 302.1 | 97.4 | — | 32/32 | 0.768 |
| 22 | volcanic_crater_snow | 9280,-8896 | -9.01 °C | 231.0 | 121.6 | — | 32/32 | 0.903 |
| 23 | basin_snow | -5184,-3648 | 3.18 °C | 291.3 | 249.3 | 4.50 | 31/32 | 1.000 |
| 24 | basin_snow | -1216,-6464 | -6.25 °C | 274.5 | 111.7 | 4.95 | 32/32 | 1.000 |

Machine-readable evidence: `docs/acceptance/v017_r82_snow_siting.json`.

## Frozen responsibilities

The fixed-seed comparison against the locked v1.1.24 baseline confirms:

- formal Waterbed rivers: **953 → 953, exact equality**;
- sinkholes: **4 → 4, exact feature equality**;
- Cherry crater ID 16: exact feature equality;
- Cherry basin ID 20: exact feature equality.

This is intentional: Slice D is a snow siting change, not a river, sinkhole, island, or Cherry relocation pass.

## Basin river regression

The old basin-river regression was tied to one historical coordinate rectangle, so moving a basin to a valid mountain background could reduce the test to zero checks. The Go and Java verifiers now inspect the currently persisted basin locations/drainage instead.

Confirmed results:

- Go drainage-reach cross-sections: **112 checked, 0 raised**, max normalized excess 0;
- Go longitudinal river samples: **81**, rises greater than 0.25 block: **0**;
- Java basin river verifier: **461 checked, rawRaised=0, voxelRaised=0, dryChannel=0**.

The verifier was made location-independent; its safety requirements were not relaxed.

## Fresh Minecraft 1.21.1 NeoForge Anvil acceptance

A completely fresh world `R817SliceDSnowV125Rect` was generated from the current v1.1.25 engine. Four thin forced-load transects cover one representative snow crater and one representative snow basin:

- crater X transect: 31 chunks;
- crater Z transect: 26 new chunks after the center overlap;
- basin X transect: 99 chunks;
- basin Z transect: 76 new chunks after the center overlap.

Total unique target chunks: **232**.

After `save-all flush` and normal server shutdown:

- target chunks FULL: **232/232**;
- model-profile sample columns readable: **234/234**;
- RuntimeModel-to-Anvil visible geometry max error: **0 blocks**;
- representative crater center: expected Y273, actual Y273;
- representative basin center: expected Y267, actual Y267.

One crater point at `10896,-6464` contains a two-block lava cover over stone. Its visible lava surface is Y271 and matches the model shape Y271 exactly; treating the stone floor as the visible terrain would incorrectly report a -2 block mismatch.

Machine-readable evidence:

- `docs/acceptance/v017_r82_snow_model_profile.json`
- `docs/acceptance/v017_r82_snow_voxel_audit.json`
- audit script: `scripts/v017_r82_snow_voxel_audit.py`

## Regression gates

Passed on the Slice D source:

- `go test ./internal/specialfeatures -count=1`
- `go test ./internal/generator -count=1`
- `go test ./... -count=1`
- NeoForge 1.21.1 `clean check build`: **BUILD SUCCESSFUL**, 45 tasks
- Model Verifier: **34/34**
- biome coverage: **53/53**
- WaterOwnership and existing shoreline/mineral/stronghold verifiers completed in the full build.

## Acceptance

**Slice D passes.**

The fixed seed no longer places snow craters/basins on the previously identified warm or flat backgrounds; accepted snow features have real cold/elevated mountain context, existing non-snow responsibilities remain locked, basin-river safety remains green, and the fresh saved Minecraft Anvil terrain matches the persisted RuntimeModel on all audited profile samples.
