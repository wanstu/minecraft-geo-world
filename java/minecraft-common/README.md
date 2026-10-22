# minecraft-common

Shared Minecraft-facing GeoWorld logic that must stay independent of any
specific Minecraft version and any mod loader.

Allowed dependencies:
- GeoWorld runtime/model classes
- Java standard library
- stable third-party libraries already used by shared code

Current reusable contracts/rules include:
- `MinecraftVersionProfile`
  - version identity
  - `HeightMapping`
  - `SurfaceProfileMapping`
  - `BiomeFamilyMapping`
  - `BiomeCatalog`
- `MinecraftPlatformAdapter<S, ChunkCodec, BiomeCodec>`
- `ServerLifecycleAdapter<S>`
- `WorldgenRegistrationAdapter<ChunkCodec, BiomeCodec>`
- `HeightMapping`
- `GeoWorldShorelinePlanner` / `ShorelineRules`
- `GeoWorldMineralPlanner` / `GeoWorldMineralRules`
- `GeoWorldChunkPerformance`
- GeoWorld creation config, native engine bridge and spawn planning

Platform features are intentionally split into small capability interfaces.
GeoWorld composes lifecycle + worldgen today; another mod can depend on only
the capabilities it actually needs.

Forbidden dependencies:
- `net.minecraft.*`
- `net.neoforged.*`
- `net.fabricmc.*`
- version packages such as `geoworld.minecraft.v26_3common.*`

Version-specific codecs, registries, biome catalogs, block mappings, screens,
and ChunkGenerator/BiomeSource subclasses belong in `minecraft-v<version>-common`.

Loader-specific registration and lifecycle hooks belong in the corresponding
NeoForge/Fabric/Forge adapter.
