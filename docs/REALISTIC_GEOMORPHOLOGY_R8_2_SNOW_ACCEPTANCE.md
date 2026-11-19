# GeoWorld v0.1.7 — R8.2 Slice D Snow Siting Acceptance

Date: 2026-11-18  
Rules: `geoworld-rules-v1.1.25`  
Fixed seed: `-1818694639628404530`

## Scope

Slice D only changes **new-world siting** for `volcanic_crater_snow` and `basin_snow`. It does not redesign their carved geometry and it does not reselect SpecialFeatures from an already-persisted `.geoworld` model.

A snow crater now requires a cold, elevated mountain/foothill/plateau background with sufficient local relief and a continuous inland ring. A snow basin additionally requires a measurable local depression and **no Ocean/Lake samples on either the 512-block or 768-block 32-point ring**. Rivers are allowed through those rings because a real mountain basin may have an inlet/outlet drainage corridor.

Invalid legacy snow slots are retained as shadow reservations so later legacy features cannot slide into those positions; missing snow slots are then filled from an independent qualified pool.

## Fixed-seed siting result

The old fixed-seed snow sites included warm, flat, or too-coastal backgrounds. The final Slice D siting selects:

| ID | Kind | Center | Temp | Background Y | Terrain | 512 relief | Basin depth | Land 512 | Land 768 | Mtn + Foothill |
| --- | --- | --- | ---: | ---: | --- | ---: | ---: | ---: | ---: | ---: |
| 21 | volcanic_crater_snow | 10944,-6464 | -12.36 °C | 302.1 | Mountain | 97.4 | — | 32/32 | 32/32 | 0.948 |
| 22 | volcanic_crater_snow | 9280,-8896 | -9.01 °C | 231.0 | Mountain | 121.6 | — | 32/32 | 32/32 | 0.988 |
| 23 | basin_snow | -1216,-9280 | -4.20 °C | 130.0 | Foothills | 111.5 | 5.72 | 32/32 | 32/32 | 0.547 |
| 24 | basin_snow | 3008,-9280 | -4.52 °C | 159.6 | Foothills | 130.9 | 4.37 | 32/32 | 32/32 | 0.674 |

Machine-readable evidence: `docs/acceptance/v017_r82_snow_siting.json`.

## Frozen responsibilities

The fixed-seed comparison against the locked v1.1.24 baseline confirms:

- formal Waterbed rivers: **953 → 953, exact equality**;
- sinkholes: **4 → 4, exact feature equality**;
- Cherry crater ID 16: exact feature equality;
- Cherry basin ID 20: exact feature equality.

This is intentional: Slice D is a snow siting change, not a river, sinkhole, island, or Cherry relocation pass.

## Basin river regression

The basin-river verifier is location-independent and follows currently persisted basin locations/drainage rather than one historical coordinate rectangle.

The previously completed full Slice D regression established:

- Go drainage-reach cross-sections: **112 checked, 0 raised**, max normalized excess 0;
- Go longitudinal river samples: **81**, rises greater than 0.25 block: **0**;
- Java basin river verifier: **461 checked, rawRaised=0, voxelRaised=0, dryChannel=0**.

The final basin #23 still carries **22 persisted drainage reaches**, so the coastal-continuity tightening did not avoid the hydrologically interesting basin.

## Full geometry acceptance

The earlier fresh world `R817SliceDSnowV125Rect` exercised complete orthogonal profiles across one snow crater and one snow basin:

- target chunks FULL: **232/232**;
- model-profile sample columns readable: **234/234**;
- RuntimeModel-to-Anvil visible geometry max error: **0 blocks**;
- representative crater center: expected Y273, actual Y273;
- representative basin center in that geometry run: expected Y267, actual Y267.

That run validates the unchanged snow crater/basin realization geometry. Machine-readable evidence remains:

- `docs/acceptance/v017_r82_snow_model_profile.json`
- `docs/acceptance/v017_r82_snow_voxel_audit.json`

## Final siting spot acceptance

The final coastal-continuity tightening moved the snow basins again, without changing their realization geometry. A second completely fresh 1.21.1 NeoForge world, `R817SliceDSnowV125Spot`, therefore validates the **new final coordinates** rather than rerunning hundreds of unchanged geometry chunks.

The spot set contains:

- snow crater #21 center;
- snow basin #23 center;
- basin #23 four cardinal samples at radius 512;
- basin #23 four cardinal samples at radius 768.

Results:

- target chunks FULL: **10/10**;
- model/Anvil sample rows: **12/12**;
- RuntimeModel-to-Anvil visible geometry max error: **0 blocks**;
- snow crater #21 center: expected Y273, actual Y273;
- snow basin #23 center: expected Y106, actual Y106;
- basin #23 audited geometry range: actual Y98–Y218.

Machine-readable evidence:

- `docs/acceptance/v017_r82_snow_siting_spot_voxel.json`
- audit script: `scripts/v017_r82_snow_voxel_audit.py`

The audit script now derives feature centers from the crossing fixture transects instead of hard-coding historical siting coordinates.

## Regression gates

Current coastal-tightening source has passed:

- `go test ./internal/specialfeatures ./internal/generator`;
- `go test ./internal/...`;
- `go test ./acceptance`;
- all already-run command/audit packages through the model/realism group;
- fresh NeoForge model identity check: current SpecialFeatures exact, formal rivers exact;
- final fresh-world spot Anvil acceptance: **10/10 FULL, 12/12 exact, 0-block error**.

The previous Slice D full Java gate also passed NeoForge 1.21.1 `clean check build`, Model Verifier **34/34**, biome coverage **53/53**, and the existing WaterOwnership / shoreline / mineral / stronghold verifiers.

## Acceptance

**Slice D passes with the final inland-continuity tightening.**

The fixed seed no longer places snow craters/basins on warm, flat, or immediately coastal backgrounds. Both final snow basins have 32/32 non-Ocean/non-Lake samples at 512 and 768 blocks, existing non-snow responsibilities remain locked, the hydrologically active basin is retained, and a fresh saved Minecraft world matches the persisted RuntimeModel exactly at the final representative siting coordinates.
