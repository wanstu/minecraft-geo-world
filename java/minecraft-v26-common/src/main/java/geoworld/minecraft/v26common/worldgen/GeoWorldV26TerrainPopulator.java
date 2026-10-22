package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.performance.GeoWorldChunkPerformance;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.BlockPos;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ChunkAccess;
import net.minecraft.world.level.levelgen.Heightmap;

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
            int worldZ
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
        OverlayPopulator mineralPopulator,
        OverlayPopulator amethystPopulator
    ) {
        final boolean measure = GeoWorldChunkPerformance.enabled();
        final long began = measure ? System.nanoTime() : 0L;
        GeoWorldV1 runtime = modelProvider.model(seed);
        final long modelReady = measure ? System.nanoTime() : 0L;

        int baseX = chunk.getPos().getMinBlockX();
        int baseZ = chunk.getPos().getMinBlockZ();

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

                for (int y = heights.minY(); y < maxBoundary; y++) {
                    BlockState state = blockResolver.blockAt(
                        column,
                        y,
                        surfaceBiome,
                        terracottaStrata,
                        worldX,
                        worldZ
                    );
                    if (state.isAir()) continue;

                    blockSetter.set(
                        chunk,
                        pos.set(localX, y, localZ),
                        state
                    );
                    oceanFloor.update(localX, y, localZ, state);
                    worldSurface.update(localX, y, localZ, state);
                }
            }
        }

        final long voxelReady = measure ? System.nanoTime() : 0L;
        mineralPopulator.populate(chunk, runtime, seed, shoreline);
        final long mineralsReady = measure ? System.nanoTime() : 0L;
        amethystPopulator.populate(chunk, runtime, seed, shoreline);

        if (measure) {
            GeoWorldChunkPerformance.record(
                modelReady - began,
                shoreReady - modelReady,
                biomeNanos,
                Math.max(0L, voxelReady - shoreReady - biomeNanos),
                mineralsReady - voxelReady,
                System.nanoTime() - mineralsReady
            );
        }

        return CompletableFuture.completedFuture(chunk);
    }
}
