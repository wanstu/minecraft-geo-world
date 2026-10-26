package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.minecraft.v26common.GeoWorldMinecraftIds;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.BlockPos;
import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.world.level.ChunkPos;
import net.minecraft.world.level.levelgen.structure.StructureStart;
import net.minecraft.world.level.biome.BiomeGenerationSettings;
import net.minecraft.world.level.LevelHeightAccessor;
import net.minecraft.world.level.NoiseColumn;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ChunkAccess;
import net.minecraft.world.level.levelgen.Heightmap;

import java.util.concurrent.CompletableFuture;

/**
 * Shared Minecraft 26.x generator implementation behind version-specific
 * ChunkGenerator lifecycle adapters.
 */
public final class GeoWorldV26GeneratorSupport {
    @FunctionalInterface
    public interface HeightGetter {
        int get(LevelHeightAccessor heightAccessor);
    }

    @FunctionalInterface
    public interface CarverCopier {
        void copy(
            BiomeGenerationSettings source,
            BiomeGenerationSettings.PlainBuilder target
        );
    }

    public record MinecraftApi(
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter,
        HeightGetter minY,
        HeightGetter maxY,
        CarverCopier carverCopier
    ) {
        public MinecraftApi {
            java.util.Objects.requireNonNull(blockSetter, "blockSetter");
            java.util.Objects.requireNonNull(minY, "minY");
            java.util.Objects.requireNonNull(maxY, "maxY");
            java.util.Objects.requireNonNull(carverCopier, "carverCopier");
        }

        public int minY(LevelHeightAccessor heightAccessor) {
            return minY.get(heightAccessor);
        }

        public int maxY(LevelHeightAccessor heightAccessor) {
            return maxY.get(heightAccessor);
        }

        public void copyCarvers(
            BiomeGenerationSettings source,
            BiomeGenerationSettings.PlainBuilder target
        ) {
            carverCopier.copy(source, target);
        }
    }

    private final MinecraftApi api;
    private final HeightMapping heights;
    private final GeoWorldV26BiomeRealizer biomes;
    private final GeoWorldShorelinePlanner shoreline;

    public GeoWorldV26GeneratorSupport(
        MinecraftVersionProfile profile,
        MinecraftApi api
    ) {
        java.util.Objects.requireNonNull(profile, "profile");
        this.api = java.util.Objects.requireNonNull(api, "api");
        this.heights = profile.heightMapping();
        this.biomes = new GeoWorldV26BiomeRealizer(profile);
        this.shoreline = new GeoWorldShorelinePlanner(this.heights);
    }

    public HeightMapping heights() {
        return this.heights;
    }

    public GeoWorldShorelinePlanner shoreline() {
        return this.shoreline;
    }

    public CompletableFuture<ChunkAccess> populateTerrain(
        ChunkAccess chunk,
        long seed,
        GeoWorldV26TerrainPopulator.ModelProvider modelProvider
    ) {
        return populateTerrain(chunk, seed, modelProvider, null);
    }

    public CompletableFuture<ChunkAccess> populateTerrain(
        ChunkAccess chunk,
        long seed,
        GeoWorldV26TerrainPopulator.ModelProvider modelProvider,
        java.util.List<StructureStart> structureStarts
    ) {
        return GeoWorldV26TerrainPopulator.populate(
            chunk,
            seed,
            modelProvider,
            this.heights,
            this.shoreline,
            this.biomes::biomeIdFromSample,
            biome -> GeoWorldV26SurfaceMaterials.cover(biome) ==
                GeoWorldV26SurfaceMaterials.Cover.BADLANDS,
            this::blockAt,
            this.api.blockSetter(),
            (target, model, worldSeed, shore) ->
                GeoWorldV26StructureAdaptation.populate(
                    target,
                    model,
                    structureStarts,
                    shore,
                    this.heights,
                    this.api.blockSetter()
                ),
            (target, model, worldSeed, shore) ->
                GeoWorldV26SparseCaves.populate(
                    target,
                    model,
                    worldSeed,
                    shore,
                    this.heights,
                    this.api.blockSetter()
                ),
            (target, model, worldSeed, shore) ->
                GeoWorldV26MineralVeins.populate(
                    target,
                    model,
                    worldSeed,
                    shore,
                    this.api.blockSetter()
                ),
            (target, model, worldSeed, shore) ->
                GeoWorldV26AmethystGeodes.populate(
                    target,
                    worldSeed,
                    shore,
                    this.api.blockSetter()
                )
        );
    }

    public int baseHeight(
        GeoWorldV1 model,
        int x,
        int z,
        Heightmap.Types type,
        LevelHeightAccessor heightAccessor
    ) {
        GeoWorldShorelinePlanner.Column column =
            this.shoreline.planModel(model, x, z);
        int ground = column.groundBoundary();
        int water = column.waterBoundary();

        if (
            type == Heightmap.Types.OCEAN_FLOOR ||
            type == Heightmap.Types.OCEAN_FLOOR_WG
        ) {
            return clampToHeightAccessor(ground, heightAccessor);
        }
        return clampToHeightAccessor(
            Math.max(ground, water),
            heightAccessor
        );
    }

    public NoiseColumn baseColumn(
        GeoWorldV1 model,
        int x,
        int z,
        LevelHeightAccessor heightAccessor
    ) {
        GeoWorldShorelinePlanner.Column column =
            this.shoreline.planModel(model, x, z);
        int ground = column.groundBoundary();
        String surfaceBiome = this.biomes.biomeIdFromSample(
            model,
            x,
            ground - 1,
            z,
            column.sample()
        );
        boolean terracottaStrata =
            GeoWorldV26SurfaceMaterials.cover(surfaceBiome) ==
            GeoWorldV26SurfaceMaterials.Cover.BADLANDS;

        BlockState[] states = new BlockState[heightAccessor.getHeight()];
        int minY = this.api.minY(heightAccessor);
        for (int index = 0; index < states.length; index++) {
            int y = minY + index;
            states[index] = blockAt(
                column,
                y,
                surfaceBiome,
                terracottaStrata,
                x,
                z,
                model.manifest().generationRulesVersion()
            );
        }
        return new NoiseColumn(minY, states);
    }

    public int spawnHeight(
        GeoWorldV1 model,
        long modelSeed,
        long spawnOriginSeed,
        ChunkPos spawnOrigin,
        LevelHeightAccessor heightAccessor
    ) {
        if (model != null && spawnOriginSeed == modelSeed) {
            int x = spawnOrigin.getMinBlockX() + 8;
            int z = spawnOrigin.getMinBlockZ() + 8;
            GeoWorldV1.RuntimeSample sample = model.sample(x, z);
            int ground = this.heights.boundaryY(sample.groundHeight());
            return clampToHeightAccessor(ground, heightAccessor);
        }
        return Math.min(
            this.api.maxY(heightAccessor),
            this.heights.seaLevel() + 1
        );
    }

    public String debugLine(GeoWorldV1 model, BlockPos feetPos) {
        GeoWorldV1.RuntimeSample sample =
            model.sample(feetPos.getX(), feetPos.getZ());
        return "GeoWorld ground=" +
            String.format(
                java.util.Locale.ROOT,
                "%.3f",
                sample.groundHeight()
            ) +
            " water=" +
            sample.waterKind() +
            " biome=" +
            sample.biomeFamily();
    }

    private BlockState blockAt(
        GeoWorldShorelinePlanner.Column column,
        int y,
        String surfaceBiome,
        boolean terracottaStrata,
        int worldX,
        int worldZ,
        String generationRulesVersion
    ) {
        GeoWorldV1.RuntimeSample sample = column.sample();
        int groundBoundary = column.groundBoundary();
        int waterBoundary = column.waterBoundary();

        if (
            y < this.heights.minY() ||
            y >= this.heights.maxYExclusive()
        ) {
            return Blocks.AIR.defaultBlockState();
        }
        if (y == this.heights.minY()) {
            return Blocks.BEDROCK.defaultBlockState();
        }

        if (y < groundBoundary) {
            int depth = groundBoundary - 1 - y;
            if (depth == 0) {
                return blockState(
                    GeoWorldV26SurfaceMaterials.topForRules(
                        surfaceBiome,
                        sample.surfaceProfile(),
                        column.surfaceOverride(),
                        generationRulesVersion
                    )
                );
            }
            if (depth > 3 && (!terracottaStrata || depth > 50)) {
                return y < 0
                    ? Blocks.DEEPSLATE.defaultBlockState()
                    : Blocks.STONE.defaultBlockState();
            }
            String shallow = GeoWorldV26SurfaceMaterials.belowForRules(
                surfaceBiome,
                sample.surfaceProfile(),
                column.underOverride(),
                worldX,
                y,
                worldZ,
                depth,
                generationRulesVersion
            );
            if (shallow != null) {
                return blockState(shallow);
            }
            return y < 0
                ? Blocks.DEEPSLATE.defaultBlockState()
                : Blocks.STONE.defaultBlockState();
        }

        if (y < waterBoundary) {
            if (
                y == waterBoundary - 1 &&
                "Frozen Ocean".equals(sample.biomeFamily()) &&
                "Ocean".equals(sample.waterKind())
            ) {
                return Blocks.ICE.defaultBlockState();
            }
            return Blocks.WATER.defaultBlockState();
        }

        return Blocks.AIR.defaultBlockState();
    }

    private int clampToHeightAccessor(
        int boundary,
        LevelHeightAccessor heightAccessor
    ) {
        int min = this.api.minY(heightAccessor);
        int maxExclusive = min + heightAccessor.getHeight();
        return Math.max(min, Math.min(maxExclusive, boundary));
    }

    private static BlockState blockState(String id) {
        var block = (net.minecraft.world.level.block.Block)
            GeoWorldMinecraftIds.registryValue(BuiltInRegistries.BLOCK, id);
        if (block == null) {
            throw new IllegalArgumentException(
                "Unknown Minecraft block id: " + id
            );
        }
        return block.defaultBlockState();
    }
}
