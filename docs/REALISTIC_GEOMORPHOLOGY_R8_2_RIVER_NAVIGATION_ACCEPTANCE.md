# GeoWorld v0.1.7 — R8.2 Slice E River Navigation Acceptance

Date: 2026-11-18  
Rules: `geoworld-rules-v1.1.26`  
Fixed seed: `-1818694639628404530`

## Scope

Slice E addresses two gameplay-visible river navigation failures without changing the persisted v1.1.25 world model:

1. formal rivers could contain 1–2 block deep channel sections that were difficult to navigate by boat;
2. tropical OceanMouth corridors could be realized as `minecraft:warm_ocean`, allowing vanilla coral reefs to occupy the river-to-sea navigation path.

The fix is intentionally split by responsibility:

- Waterbed river topology / centerline geometry stays unchanged;
- WorldSample v1.1.26 guarantees a three-block Minecraft-quantized navigation core by lowering only the inner river bed, never by raising the water surface;
- Coastal backwater v2 keeps the formal OceanMouth channel semantically River while continuing to share the receiving sea datum;
- OceanMouth influence metadata is exposed to Minecraft biome realization;
- warm-water Ocean inside the existing mouth influence corridor is realized as `minecraft:lukewarm_ocean` rather than coral-bearing `minecraft:warm_ocean`;
- outside the OceanMouth influence corridor, warm-ocean climate remains unchanged for Slice F to calibrate globally.

## Version isolation

New worlds use `geoworld-rules-v1.1.26`.

Persisted v1.1.25 worlds retain:

- CoastalBackwaterVersion 1;
- RiverNavigationVersion 0;
- no three-block navigation carve;
- their already-persisted SpecialFeatures and river model.

The v1.1.26 fixed-seed regeneration keeps:

- formal Waterbed rivers: **953 → 953, exact equality**;
- persisted SpecialFeatures: **18 → 18, exact equality after reload**;
- RiverGeometryVersion: **8**, unchanged.

Only Runtime river navigation / mouth semantics are versioned forward.

## Three-block navigable core

The navigation rule is applied only to the inner river corridor at `RiverWeight >= 0.5`.

Minecraft voxel quantization is used directly:

- water top: `ceil(blockBoundary(waterSurface)) - 1`;
- ground top: `floor(blockBoundary(ground))`;
- required navigable depth: **3 actual water blocks**.

The water surface is not raised. If necessary, the river bed is lowered only enough to satisfy the voxel contract. Banks outside the navigation core remain naturally shallow.

Fixed-seed Runtime audit:

- formal rivers: **953**;
- OceanMouth segments: **88**;
- sampled formal river center columns: **9,386**;
- minimum actual water depth: **3 blocks**;
- p10: **3 blocks**;
- p50: **3 blocks**;
- columns shallower than 3 blocks: **0**;
- OceanMouth center samples: **828 River / 0 Ocean**.

Java/Minecraft height-mapping verifier independently confirms:

- navigation core checks: **1,705**;
- shallow (<3 blocks): **0**.

## River mouth warm-ocean / coral protection

The investigation found that the observed coast problem is real but is not caused by a special coastal warming bug.

Fixed-seed macro-ocean baseline:

- nearest coastal macro ring: Warm Ocean family **49.15%**;
- offshore at 6+ macro cells: **34.19%**;
- at absolute latitude 0–15°: coast **100%**, offshore **100%** Warm Ocean family;
- at 15–30°: coast **100%**, offshore **100%**;
- at 30–45°: coast **23.60%**, offshore **27.44%**.

Therefore the underlying semantic issue is:

**warm water climate is currently being mapped too directly to a coral-reef Minecraft biome.**

Slice E does not globally cool tropical oceans. It protects navigation only where hydrology says a formal OceanMouth exists.

Cross-section audit across all OceanMouth terminal reaches found:

- mouth-influence samples: **12,572**;
- River samples inside influence: **7,167**;
- Ocean samples inside influence: **2,791**;
- influenced Ocean samples whose climate family is still Warm Ocean: **2,201**.

Those warm-water samples remain climatically warm but are realized as `minecraft:lukewarm_ocean` while inside the mouth influence corridor.

Java SurfaceBiomeVerifier confirms:

- mouth influence checks: **920**;
- Warm Ocean family Ocean samples: **59**;
- final `minecraft:warm_ocean` violations: **0**;
- formal River biome violations: **0**.

## Biome coverage hydrology fix

Slice E also exposed an older bug in guaranteed biome coverage habitats:

- a land/ocean habitat reservation could overlap a nearby River or Ocean at its circular edge;
- the coverage override could then replace the Runtime hydrology biome merely to preserve the synthetic habitat circle;
- candidate selection also treated River/Lake columns as ordinary land.

This is now constrained by hydrology:

- ordinary land habitats require `WaterKind=None`;
- river habitats require River;
- ocean habitats require Ocean;
- habitat circles are clipped naturally at real river / ocean boundaries;
- a v1.1.26 OceanMouth influence corridor cannot be reintroduced as `minecraft:warm_ocean` by a coverage override.

All **53/53** required overworld biomes still remain observable after this correction.

## Fresh Minecraft Anvil acceptance

A completely fresh 1.21.1 NeoForge world was generated:

- world: `R817SliceERiverV126Spot`;
- representative tropical OceanMouth: river segment #43;
- audited rectangle: 97×97 columns around the river mouth;
- target chunks: **49/49 FULL**;
- audited saved columns: **9,409**.

Direct saved-Anvil block-state results:

- navigation-core columns: **1,558**;
- minimum actual water depth: **3 blocks**;
- core columns shallower than 3: **0**;
- wet columns inside OceanMouth influence: **5,231**;
- Warm Ocean family columns inside influence: **2,128**;
- actual saved `*coral*` blocks inside OceanMouth influence: **0**;
- actual saved coral blocks in the same rectangle but outside the influence corridor: **84**.

The 84 outside-corridor coral blocks are an important negative control: coral generation is active in this warm-water area, but the formal river-mouth navigation corridor remains clear.

Machine-readable evidence:

- `docs/acceptance/v017_r82_river_navigation_v126_voxel.json`
- audit script: `scripts/v017_slicee_river_mouth_voxel_audit.py`

## Regression gates

Passed on the final Slice E source:

- `go test ./internal/...`;
- `go test ./internal/generator`;
- `go test ./internal/worldsample`;
- `go test ./internal/runtimeprofile`;
- `go test ./internal/specialfeatures`;
- Java biome coverage: **53/53**;
- Java Model Verifier: **34/34**;
- Basin River: **461 checks, 0 raised, 0 dry**;
- SurfaceBiome Slice E navigation: **1705 core / 0 shallow / 0 warm-ocean violations / 0 river-biome violations**;
- WaterOwnership: **0 unowned sources, 0 unsafe exposed sources, 0 free-fall faces, 0 runtime water lost**;
- NeoForge 1.21.1 `clean check build`: **BUILD SUCCESSFUL**;
- fresh saved-Anvil acceptance: **49/49 FULL, min depth 3, 0 shallow, 0 mouth coral, 84 outside-mouth coral**.

## Acceptance

**Slice E passes.**

The formal river-to-sea route is now navigable through a three-block-deep inner channel, retains River hydrology semantics at the mouth, and no longer becomes a coral-bearing Warm Ocean corridor. Tropical warm water remains present outside the hydrologically defined navigation influence.

The broader question of how much tropical coast should be realized as actual coral-bearing `minecraft:warm_ocean` is deliberately carried into Slice F together with the forest/jungle area-distribution work.
