package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.BlockPos;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ChunkAccess;
import net.minecraft.world.level.levelgen.Heightmap;
import net.minecraft.world.level.levelgen.structure.BoundingBox;
import net.minecraft.world.level.levelgen.structure.StructureStart;
import net.minecraft.world.level.levelgen.structure.TerrainAdjustment;

import java.util.List;

/**
 * Small, local terrain concessions for Vanilla structures that explicitly
 * request terrain adaptation.
 *
 * GeoWorld never chooses structure positions here. StructureManager remains
 * authoritative. Major rivers, water bodies, canyons and mountain cores are
 * protected even at the strongest setting.
 */
public final class GeoWorldV26StructureAdaptation {
    private GeoWorldV26StructureAdaptation() {}

    public static void populate(
        ChunkAccess chunk,
        GeoWorldV1 model,
        List<StructureStart> starts,
        GeoWorldShorelinePlanner.ChunkPlan shoreline,
        HeightMapping heights,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter
    ) {
        if (model.worldProfile().legacy() || starts == null || starts.isEmpty()) return;

        double influence = model
            .worldProfile()
            .parameters()
            .structureInfluence();
        if (influence <= 1.0e-9) return;
        influence = Math.max(0.0, Math.min(2.5, influence));

        int chunkMinX = chunk.getPos().getMinBlockX();
        int chunkMinZ = chunk.getPos().getMinBlockZ();
        int chunkMaxX = chunkMinX + 15;
        int chunkMaxZ = chunkMinZ + 15;

        Heightmap oceanFloor = chunk.getOrCreateHeightmapUnprimed(
            Heightmap.Types.OCEAN_FLOOR_WG
        );
        Heightmap worldSurface = chunk.getOrCreateHeightmapUnprimed(
            Heightmap.Types.WORLD_SURFACE_WG
        );
        BlockPos.MutableBlockPos pos = new BlockPos.MutableBlockPos();

        for (StructureStart start : starts) {
            if (
                start == null ||
                !start.isValid() ||
                start.getStructure().terrainAdaptation() ==
                    TerrainAdjustment.NONE
            ) {
                continue;
            }

            BoundingBox box = start
                .getStructure()
                .adjustBoundingBox(start.getBoundingBox());

            int centerX = (box.minX() + box.maxX()) / 2;
            int centerZ = (box.minZ() + box.maxZ()) / 2;
            GeoWorldV1.RuntimeSample center = model.sample(centerX, centerZ);
            if (!eligible(center)) continue;

            int centerBoundary = heights.boundaryY(center.groundHeight());
            if (box.maxY() < centerBoundary - 10) {
                // Underground structures keep their Vanilla placement and do
                // not flatten the natural surface above them.
                continue;
            }

            int margin = Math.max(2, (int)Math.round(2 + influence * 2));
            int minX = Math.max(chunkMinX, box.minX() - margin);
            int maxX = Math.min(chunkMaxX, box.maxX() + margin);
            int minZ = Math.max(chunkMinZ, box.minZ() - margin);
            int maxZ = Math.min(chunkMaxZ, box.maxZ() + margin);
            if (minX > maxX || minZ > maxZ) continue;

            double strength = Math.min(0.78, 0.18 + 0.24 * influence);
            int maxShift = Math.max(1, (int)Math.round(2 + 2 * influence));

            for (int worldZ = minZ; worldZ <= maxZ; worldZ++) {
                for (int worldX = minX; worldX <= maxX; worldX++) {
                    int localX = worldX - chunkMinX;
                    int localZ = worldZ - chunkMinZ;
                    GeoWorldShorelinePlanner.Column column =
                        shoreline.column(localX, localZ);
                    GeoWorldV1.RuntimeSample sample = column.sample();
                    if (!eligible(sample)) continue;

                    double edgeDistance = distanceOutside(
                        worldX,
                        worldZ,
                        box
                    );
                    double localWeight = 1.0 -
                        smoothstep(edgeDistance / Math.max(1.0, margin));
                    if (localWeight <= 0) continue;

                    int naturalBoundary = column.groundBoundary();
                    int requested = (int)Math.round(
                        naturalBoundary +
                        (centerBoundary - naturalBoundary) *
                            strength *
                            localWeight
                    );
                    int targetBoundary = clamp(
                        requested,
                        naturalBoundary - maxShift,
                        naturalBoundary + maxShift
                    );
                    targetBoundary = clamp(
                        targetBoundary,
                        heights.minY() + 2,
                        heights.maxYExclusive() - 1
                    );
                    if (targetBoundary == naturalBoundary) continue;

                    adaptColumn(
                        chunk,
                        blockSetter,
                        oceanFloor,
                        worldSurface,
                        pos,
                        worldX,
                        worldZ,
                        localX,
                        localZ,
                        naturalBoundary,
                        targetBoundary
                    );
                }
            }
        }
    }

    private static boolean eligible(GeoWorldV1.RuntimeSample sample) {
        if (
            !sample.sourceLand() ||
            !"None".equals(sample.waterKind()) ||
            sample.riverWeight() > 0.12
        ) {
            return false;
        }
        return switch (sample.terrainVariant()) {
            case "Mountain", "Canyon", "Floodplain", "Alluvial Plain",
                "Alluvial Fan" -> false;
            default -> true;
        };
    }

    private static void adaptColumn(
        ChunkAccess chunk,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter,
        Heightmap oceanFloor,
        Heightmap worldSurface,
        BlockPos.MutableBlockPos pos,
        int worldX,
        int worldZ,
        int localX,
        int localZ,
        int naturalBoundary,
        int targetBoundary
    ) {
        int naturalTop = naturalBoundary - 1;
        BlockState top = chunk.getBlockState(
            pos.set(worldX, naturalTop, worldZ)
        );
        BlockState filler = chunk.getBlockState(
            pos.set(worldX, Math.max(naturalTop - 1, targetBoundary - 1), worldZ)
        );
        if (filler.isAir()) {
            filler = naturalTop < 0
                ? Blocks.DEEPSLATE.defaultBlockState()
                : Blocks.STONE.defaultBlockState();
        }

        if (targetBoundary > naturalBoundary) {
            for (int y = naturalBoundary; y < targetBoundary; y++) {
                BlockState state =
                    y == targetBoundary - 1 ? top : filler;
                set(
                    chunk,
                    blockSetter,
                    oceanFloor,
                    worldSurface,
                    pos,
                    worldX,
                    y,
                    worldZ,
                    localX,
                    localZ,
                    state
                );
            }
            return;
        }

        for (int y = naturalTop; y >= targetBoundary; y--) {
            set(
                chunk,
                blockSetter,
                oceanFloor,
                worldSurface,
                pos,
                worldX,
                y,
                worldZ,
                localX,
                localZ,
                Blocks.AIR.defaultBlockState()
            );
        }
        int newTop = targetBoundary - 1;
        set(
            chunk,
            blockSetter,
            oceanFloor,
            worldSurface,
            pos,
            worldX,
            newTop,
            worldZ,
            localX,
            localZ,
            top
        );
    }

    private static void set(
        ChunkAccess chunk,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter,
        Heightmap oceanFloor,
        Heightmap worldSurface,
        BlockPos.MutableBlockPos pos,
        int worldX,
        int y,
        int worldZ,
        int localX,
        int localZ,
        BlockState state
    ) {
        blockSetter.set(
            chunk,
            pos.set(worldX, y, worldZ),
            state
        );
        oceanFloor.update(localX, y, localZ, state);
        worldSurface.update(localX, y, localZ, state);
    }

    private static double distanceOutside(
        int x,
        int z,
        BoundingBox box
    ) {
        double dx = x < box.minX()
            ? box.minX() - x
            : x > box.maxX()
                ? x - box.maxX()
                : 0;
        double dz = z < box.minZ()
            ? box.minZ() - z
            : z > box.maxZ()
                ? z - box.maxZ()
                : 0;
        return Math.hypot(dx, dz);
    }

    private static double smoothstep(double t) {
        t = Math.max(0.0, Math.min(1.0, t));
        return t * t * (3.0 - 2.0 * t);
    }

    private static int clamp(int value, int lo, int hi) {
        return Math.max(lo, Math.min(hi, value));
    }
}
