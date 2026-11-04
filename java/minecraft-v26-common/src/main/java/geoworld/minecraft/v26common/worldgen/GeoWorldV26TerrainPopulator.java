package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.performance.GeoWorldChunkPerformance;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.BlockPos;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ChunkAccess;
import net.minecraft.world.level.chunk.LevelChunkSection;
import net.minecraft.world.level.chunk.status.ChunkStatus;
import net.minecraft.world.level.levelgen.Heightmap;

import java.util.EnumSet;
import java.util.concurrent.CompletableFuture;

/**
 * Shared Minecraft 26.x terrain population pipeline.
 *
 * Minecraft 26.2 and 26.3 expose different ChunkGenerator lifecycle methods,
 * but the actual GeoWorld voxel realization is identical. Version shells call
 * this class from fillFromNoise/buildTerrain respectively.
 */
public final class GeoWorldV26TerrainPopulator {
    private GeoWorldV26TerrainPopulator() {}

    @FunctionalInterface
    public interface ModelProvider {
        GeoWorldV1 model(long seed);
    }

    @FunctionalInterface
    public interface SurfaceBiomeResolver {
        String biomeId(
            GeoWorldV1 runtime,
            int x,
            int y,
            int z,
            GeoWorldV1.RuntimeSample sample
        );
    }

    @FunctionalInterface
    public interface BadlandsClassifier {
        boolean isBadlands(String biomeId);
    }

    @FunctionalInterface
    public interface ColumnBlockResolver {
        BlockState blockAt(
            GeoWorldShorelinePlanner.Column column,
            int y,
            String surfaceBiome,
            boolean terracottaStrata,
            int worldX,
            int worldZ,
            String generationRulesVersion
        );
    }

    @FunctionalInterface
    public interface BlockSetter {
        void set(ChunkAccess chunk, BlockPos pos, BlockState state);
    }

    @FunctionalInterface
    public interface OverlayPopulator {
        void populate(
            ChunkAccess chunk,
            GeoWorldV1 runtime,
            long seed,
            GeoWorldShorelinePlanner.ChunkPlan shoreline
        );
    }

    public static CompletableFuture<ChunkAccess> populate(
        ChunkAccess chunk,
        long seed,
        ModelProvider modelProvider,
        HeightMapping heights,
        GeoWorldShorelinePlanner shorelinePlanner,
        SurfaceBiomeResolver biomeResolver,
        BadlandsClassifier badlandsClassifier,
        ColumnBlockResolver blockResolver,
        BlockSetter blockSetter,
        OverlayPopulator structurePopulator,
        OverlayPopulator cavePopulator,
        OverlayPopulator mineralPopulator,
        OverlayPopulator amethystPopulator
    ) {
        final boolean measure = GeoWorldChunkPerformance.enabled();
        final long began = measure ? System.nanoTime() : 0L;
        GeoWorldV1 runtime = modelProvider.model(seed);
        final long modelReady = measure ? System.nanoTime() : 0L;

        int baseX = chunk.getPos().getMinBlockX();
        int baseZ = chunk.getPos().getMinBlockZ();
        String generationRulesVersion =
            runtime.manifest().generationRulesVersion();
        BlockState bedrockState = Blocks.BEDROCK.defaultBlockState();
        BlockState deepslateState = Blocks.DEEPSLATE.defaultBlockState();
        BlockState stoneState = Blocks.STONE.defaultBlockState();

        GeoWorldShorelinePlanner.ChunkPlan shoreline =
            shorelinePlanner.planChunkModel(runtime, baseX, baseZ);
        final long shoreReady = measure ? System.nanoTime() : 0L;
        long biomeNanos = 0L;

        BlockPos.MutableBlockPos pos = new BlockPos.MutableBlockPos();
        Heightmap oceanFloor = chunk.getOrCreateHeightmapUnprimed(
            Heightmap.Types.OCEAN_FLOOR_WG
        );
        Heightmap worldSurface = chunk.getOrCreateHeightmapUnprimed(
            Heightmap.Types.WORLD_SURFACE_WG
        );
        var autoHeightmaps = chunk.getPersistedStatus().heightmapsAfter();
        boolean autoOceanFloor = autoHeightmaps.contains(
            Heightmap.Types.OCEAN_FLOOR_WG
        );
        boolean autoWorldSurface = autoHeightmaps.contains(
            Heightmap.Types.WORLD_SURFACE_WG
        );
        boolean fastTerrainWrite =
            chunk.getPersistedStatus() == ChunkStatus.BIOMES &&
            autoOceanFloor &&
            autoWorldSurface;
        LevelChunkSection[] sections =
            fastTerrainWrite ? chunk.getSections() : null;
        int acquiredSectionCount = 0;
        boolean[] touchedSections =
            fastTerrainWrite ? new boolean[sections.length] : null;

        try {
            if (fastTerrainWrite) {
                for (
                    ;
                    acquiredSectionCount < sections.length;
                    acquiredSectionCount++
                ) {
                    sections[acquiredSectionCount].acquire();
                }
            }

        for (int localZ = 0; localZ < 16; localZ++) {
            for (int localX = 0; localX < 16; localX++) {
                int worldX = baseX + localX;
                int worldZ = baseZ + localZ;
                GeoWorldShorelinePlanner.Column column =
                    shoreline.column(localX, localZ);
                GeoWorldV1.RuntimeSample sample = column.sample();

                int groundBoundary = column.groundBoundary();
                int waterBoundary = column.waterBoundary();
                final long biomeStarted =
                    measure ? System.nanoTime() : 0L;
                String surfaceBiome = biomeResolver.biomeId(
                    runtime,
                    worldX,
                    groundBoundary - 1,
                    worldZ,
                    sample
                );
                if (measure) {
                    biomeNanos += System.nanoTime() - biomeStarted;
                }
                boolean terracottaStrata =
                    badlandsClassifier.isBadlands(surfaceBiome);

                int maxBoundary = Math.min(
                    heights.maxYExclusive(),
                    Math.max(groundBoundary, waterBoundary)
                );

                int minY = heights.minY();
                int deepEndExclusive = Math.max(
                    minY,
                    Math.min(
                        maxBoundary,
                        groundBoundary - (terracottaStrata ? 51 : 4)
                    )
                );

                // Below the surface-material corridor the resolver is exactly
                // bedrock/deepslate/stone. Write those states directly and
                // reserve blockAt() for the shallow material and water layers.
                if (fastTerrainWrite) {
                    // Hold every chunk section once for the terrain fill
                    // so each column can use the unchecked write overload
                    // without repeating threading-detector acquire/release.
                    int y = minY;
                    while (y < maxBoundary) {
                        int sectionIndex = chunk.getSectionIndex(y);
                        LevelChunkSection section = sections[sectionIndex];
                        touchedSections[sectionIndex] = true;
                        int sectionEnd = Math.min(
                            maxBoundary,
                            (y & ~15) + 16
                        );
                        for (; y < sectionEnd; y++) {
                            BlockState state;
                            if (y < deepEndExclusive) {
                                state = y == minY
                                    ? bedrockState
                                    : (y < 0
                                        ? deepslateState
                                        : stoneState);
                            } else {
                                state = blockResolver.blockAt(
                                    column,
                                    y,
                                    surfaceBiome,
                                    terracottaStrata,
                                    worldX,
                                    worldZ,
                                    generationRulesVersion
                                );
                                if (state.isAir()) {
                                    continue;
                                }
                            }
                            section.getStates()
                                .getAndSetUnchecked(
                                    localX,
                                    y & 15,
                                    localZ,
                                    state
                                );
                        }
                    }
                } else {
                    for (int y = minY; y < deepEndExclusive; y++) {
                    BlockState state = y == minY
                        ? bedrockState
                        : (y < 0 ? deepslateState : stoneState);

                    blockSetter.set(
                        chunk,
                        pos.set(localX, y, localZ),
                        state
                    );
                    if (!autoOceanFloor) {
                        oceanFloor.update(localX, y, localZ, state);
                    }
                    if (!autoWorldSurface) {
                        worldSurface.update(localX, y, localZ, state);
                    }
                }

                for (int y = deepEndExclusive; y < maxBoundary; y++) {
                    BlockState state = blockResolver.blockAt(
                        column,
                        y,
                        surfaceBiome,
                        terracottaStrata,
                        worldX,
                        worldZ,
                        generationRulesVersion
                    );
                    if (state.isAir()) continue;

                    blockSetter.set(
                        chunk,
                        pos.set(localX, y, localZ),
                        state
                    );
                    if (!autoOceanFloor) {
                        oceanFloor.update(localX, y, localZ, state);
                    }
                    if (!autoWorldSurface) {
                        worldSurface.update(localX, y, localZ, state);
                    }
                }
            }
        }

        }
        } finally {
            if (fastTerrainWrite) {
                for (int i = 0; i < sections.length; i++) {
                    if (touchedSections[i]) {
                        sections[i].recalcBlockCounts();
                    }
                }
                for (int i = acquiredSectionCount - 1; i >= 0; i--) {
                    sections[i].release();
                }
            }
        }

        if (fastTerrainWrite) {
            Heightmap.primeHeightmaps(
                chunk,
                EnumSet.of(
                    Heightmap.Types.OCEAN_FLOOR_WG,
                    Heightmap.Types.WORLD_SURFACE_WG
                )
            );
        }

        final long voxelReady = measure ? System.nanoTime() : 0L;
        structurePopulator.populate(chunk, runtime, seed, shoreline);
        final long structuresReady = measure ? System.nanoTime() : 0L;
        cavePopulator.populate(chunk, runtime, seed, shoreline);
        final long cavesReady = measure ? System.nanoTime() : 0L;
        mineralPopulator.populate(chunk, runtime, seed, shoreline);
        final long mineralsReady = measure ? System.nanoTime() : 0L;
        amethystPopulator.populate(chunk, runtime, seed, shoreline);

        if (measure) {
            GeoWorldChunkPerformance.record(
                modelReady - began,
                shoreReady - modelReady,
                biomeNanos,
                Math.max(0L, voxelReady - shoreReady - biomeNanos),
                structuresReady - voxelReady,
                cavesReady - structuresReady,
                mineralsReady - cavesReady,
                System.nanoTime() - mineralsReady,
                fastTerrainWrite
            );
        }

        return CompletableFuture.completedFuture(chunk);
    }
}
