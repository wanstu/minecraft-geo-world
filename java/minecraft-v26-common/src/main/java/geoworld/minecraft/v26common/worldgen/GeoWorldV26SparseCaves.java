package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.BlockPos;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ChunkAccess;

import java.util.SplittableRandom;

/**
 * Sparse deterministic cave bodies owned by GeoWorld.
 *
 * Vanilla noise carvers remain disabled. Cave bodies instead come from global
 * source cells, so chunk generation order cannot create seams. The default is
 * intentionally conservative: small caves exist, large caves are rare, and no
 * body is allowed to punch through a river/lake/ocean column.
 */
public final class GeoWorldV26SparseCaves {
    private static final int CELL_XZ = 96;
    private static final int CELL_Y = 64;
    private static final int SEARCH_MARGIN = 160;
    private static final int RIVER_CELL_XZ = 256;
    private static final int RIVER_SEARCH_MARGIN = 224;
    private static final long CAVE_SALT = 0x47454F4341564531L;
    private static final long UNDERGROUND_RIVER_SALT = 0x554E445249564552L;

    private GeoWorldV26SparseCaves() {}

    public static int populate(
        ChunkAccess chunk,
        GeoWorldV1 model,
        long seed,
        GeoWorldShorelinePlanner.ChunkPlan shoreline,
        HeightMapping heights,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter
    ) {
        if (model.worldProfile().legacy()) {
            return 0;
        }
        double richness = model.worldProfile().parameters().caveRichness();
        double largeRarity = model.worldProfile().parameters().largeCaveRarity();
        if (richness <= 0.0) {
            return 0;
        }

        int minX = chunk.getPos().getMinBlockX();
        int minZ = chunk.getPos().getMinBlockZ();
        int maxX = minX + 15;
        int maxZ = minZ + 15;
        int minY = heights.minY() + 6;
        int maxY = Math.min(112, heights.maxYExclusive() - 10);

        int firstCellX = Math.floorDiv(minX - SEARCH_MARGIN, CELL_XZ);
        int lastCellX = Math.floorDiv(maxX + SEARCH_MARGIN, CELL_XZ);
        int firstCellZ = Math.floorDiv(minZ - SEARCH_MARGIN, CELL_XZ);
        int lastCellZ = Math.floorDiv(maxZ + SEARCH_MARGIN, CELL_XZ);
        int firstCellY = Math.floorDiv(minY, CELL_Y);
        int lastCellY = Math.floorDiv(maxY, CELL_Y);

        final int[] carved = {0};
        for (int cellZ = firstCellZ; cellZ <= lastCellZ; cellZ++) {
            for (int cellX = firstCellX; cellX <= lastCellX; cellX++) {
                for (int cellY = firstCellY; cellY <= lastCellY; cellY++) {
                    SplittableRandom random = new SplittableRandom(
                        cellSeed(seed ^ CAVE_SALT, cellX, cellY, cellZ)
                    );

                    double startChance = Math.min(0.42, 0.11 * richness);
                    if (random.nextDouble() >= startChance) {
                        continue;
                    }

                    boolean large =
                        random.nextDouble() <
                        Math.min(0.09, 0.028 * richness / Math.max(0.25, largeRarity));

                    double startX =
                        (cellX + 0.18 + random.nextDouble() * 0.64) * CELL_XZ;
                    double startY =
                        (cellY + 0.20 + random.nextDouble() * 0.60) * CELL_Y;
                    double startZ =
                        (cellZ + 0.18 + random.nextDouble() * 0.64) * CELL_XZ;

                    double length = large
                        ? 72.0 + random.nextDouble() * 62.0
                        : 22.0 + random.nextDouble() * 36.0;
                    double radius = large
                        ? 5.5 + random.nextDouble() * 4.5
                        : 1.8 + random.nextDouble() * 2.3;
                    double yaw = random.nextDouble() * Math.PI * 2.0;
                    double pitch = (random.nextDouble() - 0.5) * (large ? 0.20 : 0.34);
                    double bend = (random.nextDouble() - 0.5) * 0.55;

                    int steps = Math.max(5, (int)Math.ceil(length / 4.0));
                    for (int step = 0; step <= steps; step++) {
                        double t = step / (double)steps;
                        double curveYaw = yaw + Math.sin(t * Math.PI) * bend;
                        double distance = length * (t - 0.5);
                        double centerX = startX + Math.cos(curveYaw) * distance;
                        double centerZ = startZ + Math.sin(curveYaw) * distance;
                        double centerY = startY + Math.sin(pitch) * distance +
                            Math.sin(t * Math.PI * 2.0 + bend) * radius * 0.35;
                        double localRadius =
                            radius * (0.72 + 0.28 * Math.sin(Math.PI * t));

                        carveEllipsoid(
                            chunk,
                            shoreline,
                            heights,
                            blockSetter,
                            minX,
                            maxX,
                            minZ,
                            maxZ,
                            centerX,
                            centerY,
                            centerZ,
                            localRadius,
                            large ? localRadius * 0.72 : localRadius * 0.58,
                            carved
                        );
                    }
                }
            }
        }

        String rules = model.manifest().generationRulesVersion();
        if (
            "geoworld-rules-v1.1.7".equals(rules) ||
            "geoworld-rules-v1.1.8".equals(rules) ||
            "geoworld-rules-v1.1.9".equals(rules) ||
            "geoworld-rules-v1.1.10".equals(rules)
        ) {
            carved[0] += populateUndergroundRivers(
                chunk,
                model,
                seed,
                shoreline,
                heights,
                blockSetter,
                minX,
                maxX,
                minZ,
                maxZ,
                richness
            );
        }
        return carved[0];
    }

    private static int populateUndergroundRivers(
        ChunkAccess chunk,
        GeoWorldV1 model,
        long seed,
        GeoWorldShorelinePlanner.ChunkPlan shoreline,
        HeightMapping heights,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter,
        int chunkMinX,
        int chunkMaxX,
        int chunkMinZ,
        int chunkMaxZ,
        double caveRichness
    ) {
        int firstCellX = Math.floorDiv(
            chunkMinX - RIVER_SEARCH_MARGIN,
            RIVER_CELL_XZ
        );
        int lastCellX = Math.floorDiv(
            chunkMaxX + RIVER_SEARCH_MARGIN,
            RIVER_CELL_XZ
        );
        int firstCellZ = Math.floorDiv(
            chunkMinZ - RIVER_SEARCH_MARGIN,
            RIVER_CELL_XZ
        );
        int lastCellZ = Math.floorDiv(
            chunkMaxZ + RIVER_SEARCH_MARGIN,
            RIVER_CELL_XZ
        );
        double startChance = Math.min(
            0.18,
            0.075 * Math.max(0.0, caveRichness)
        );
        if (startChance <= 0.0) {
            return 0;
        }

        int placed = 0;
        for (int cellZ = firstCellZ; cellZ <= lastCellZ; cellZ++) {
            for (int cellX = firstCellX; cellX <= lastCellX; cellX++) {
                SplittableRandom random = new SplittableRandom(
                    cellSeed(
                        seed ^ UNDERGROUND_RIVER_SALT,
                        cellX,
                        0,
                        cellZ
                    )
                );
                if (random.nextDouble() >= startChance) {
                    continue;
                }

                double startX =
                    (cellX + 0.16 + random.nextDouble() * 0.68) *
                    RIVER_CELL_XZ;
                double startZ =
                    (cellZ + 0.16 + random.nextDouble() * 0.68) *
                    RIVER_CELL_XZ;
                GeoWorldV1.RuntimeSample source =
                    model.sample(startX, startZ);
                if (
                    !source.sourceLand() ||
                    !"None".equals(source.waterKind())
                ) {
                    continue;
                }

                int surfaceY =
                    heights.boundaryY(source.groundHeight()) - 1;
                int highestWaterY =
                    Math.min(48, surfaceY - 18);
                int lowestWaterY =
                    Math.max(heights.minY() + 10, -42);
                if (highestWaterY <= lowestWaterY) {
                    continue;
                }
                int waterTopY =
                    lowestWaterY +
                    random.nextInt(highestWaterY - lowestWaterY + 1);

                double length =
                    92.0 + random.nextDouble() * 96.0;
                double radiusXZ =
                    2.8 + random.nextDouble() * 2.2;
                double radiusY =
                    2.8 + random.nextDouble() * 1.6;
                double yaw = random.nextDouble() * Math.PI * 2.0;
                double bend =
                    (random.nextDouble() - 0.5) * 0.72;
                double phase = random.nextDouble() * Math.PI * 2.0;
                int steps = Math.max(
                    18,
                    (int)Math.ceil(length / 3.2)
                );

                for (int step = 0; step <= steps; step++) {
                    double t = step / (double)steps;
                    double envelope =
                        Math.pow(Math.sin(Math.PI * t), 0.72);
                    if (envelope <= 1.0e-6) {
                        continue;
                    }
                    double along = length * (t - 0.5);
                    double curveYaw =
                        yaw +
                        Math.sin(
                            (t - 0.5) * Math.PI
                        ) * bend;
                    double centerX =
                        startX + Math.cos(curveYaw) * along;
                    double centerZ =
                        startZ + Math.sin(curveYaw) * along;
                    double centerY =
                        waterTopY +
                        1.65 +
                        Math.sin(
                            t * Math.PI * 2.0 + phase
                        ) * 0.65;
                    double localRadiusXZ =
                        Math.max(1.45, radiusXZ * envelope);
                    double localRadiusY =
                        Math.max(1.65, radiusY * envelope);

                    placed += carveUndergroundRiverSection(
                        chunk,
                        shoreline,
                        heights,
                        blockSetter,
                        chunkMinX,
                        chunkMaxX,
                        chunkMinZ,
                        chunkMaxZ,
                        centerX,
                        centerY,
                        centerZ,
                        localRadiusXZ,
                        localRadiusY,
                        waterTopY
                    );
                }
            }
        }
        return placed;
    }

    private static int carveUndergroundRiverSection(
        ChunkAccess chunk,
        GeoWorldShorelinePlanner.ChunkPlan shoreline,
        HeightMapping heights,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter,
        int chunkMinX,
        int chunkMaxX,
        int chunkMinZ,
        int chunkMaxZ,
        double centerX,
        double centerY,
        double centerZ,
        double radiusXZ,
        double radiusY,
        int waterTopY
    ) {
        int minX = Math.max(
            chunkMinX,
            (int)Math.floor(centerX - radiusXZ)
        );
        int maxX = Math.min(
            chunkMaxX,
            (int)Math.ceil(centerX + radiusXZ)
        );
        int minZ = Math.max(
            chunkMinZ,
            (int)Math.floor(centerZ - radiusXZ)
        );
        int maxZ = Math.min(
            chunkMaxZ,
            (int)Math.ceil(centerZ + radiusXZ)
        );
        int minY = Math.max(
            heights.minY() + 5,
            waterTopY - 1
        );
        int maxY = Math.min(
            heights.maxYExclusive() - 2,
            (int)Math.ceil(centerY + radiusY)
        );
        if (minX > maxX || minZ > maxZ || minY > maxY) {
            return 0;
        }

        BlockPos.MutableBlockPos pos =
            new BlockPos.MutableBlockPos();
        BlockState air = Blocks.AIR.defaultBlockState();
        BlockState water = Blocks.WATER.defaultBlockState();
        int placed = 0;

        for (int x = minX; x <= maxX; x++) {
            int localX = x - chunkMinX;
            double nx = (x + 0.5 - centerX) / radiusXZ;
            for (int z = minZ; z <= maxZ; z++) {
                int localZ = z - chunkMinZ;
                GeoWorldShorelinePlanner.Column column =
                    shoreline.column(localX, localZ);
                if (
                    !column.sample().sourceLand() ||
                    !"None".equals(column.sample().waterKind())
                ) {
                    continue;
                }

                int safeCeiling =
                    column.groundBoundary() - 12;
                int columnMaxY = Math.min(maxY, safeCeiling);
                if (columnMaxY < minY) {
                    continue;
                }

                double nz =
                    (z + 0.5 - centerZ) / radiusXZ;
                double horizontal = nx * nx + nz * nz;
                if (horizontal >= 1.0) {
                    continue;
                }

                for (int y = minY; y <= columnMaxY; y++) {
                    double ny =
                        (y + 0.5 - centerY) / radiusY;
                    if (horizontal + ny * ny > 1.0) {
                        continue;
                    }
                    pos.set(localX, y, localZ);
                    BlockState host = chunk.getBlockState(pos);
                    if (
                        !host.is(Blocks.STONE) &&
                        !host.is(Blocks.DEEPSLATE)
                    ) {
                        continue;
                    }

                    BlockState replacement =
                        y <= waterTopY ? water : air;
                    blockSetter.set(
                        chunk,
                        pos,
                        replacement
                    );
                    placed++;
                }
            }
        }
        return placed;
    }

    private static void carveEllipsoid(
        ChunkAccess chunk,
        GeoWorldShorelinePlanner.ChunkPlan shoreline,
        HeightMapping heights,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter,
        int chunkMinX,
        int chunkMaxX,
        int chunkMinZ,
        int chunkMaxZ,
        double centerX,
        double centerY,
        double centerZ,
        double radiusXZ,
        double radiusY,
        int[] carved
    ) {
        int minX = Math.max(chunkMinX, (int)Math.floor(centerX - radiusXZ));
        int maxX = Math.min(chunkMaxX, (int)Math.ceil(centerX + radiusXZ));
        int minZ = Math.max(chunkMinZ, (int)Math.floor(centerZ - radiusXZ));
        int maxZ = Math.min(chunkMaxZ, (int)Math.ceil(centerZ + radiusXZ));
        int minY = Math.max(heights.minY() + 5, (int)Math.floor(centerY - radiusY));
        int maxY = Math.min(
            heights.maxYExclusive() - 2,
            (int)Math.ceil(centerY + radiusY)
        );

        if (minX > maxX || minZ > maxZ || minY > maxY) {
            return;
        }

        BlockPos.MutableBlockPos pos = new BlockPos.MutableBlockPos();
        BlockState air = Blocks.AIR.defaultBlockState();

        for (int x = minX; x <= maxX; x++) {
            int localX = x - chunkMinX;
            double nx = (x + 0.5 - centerX) / radiusXZ;
            for (int z = minZ; z <= maxZ; z++) {
                int localZ = z - chunkMinZ;
                GeoWorldShorelinePlanner.Column column =
                    shoreline.column(localX, localZ);

                // Never create an unplanned water source or leak path.
                if (
                    !column.sample().sourceLand() ||
                    !"None".equals(column.sample().waterKind())
                ) {
                    continue;
                }

                double nz = (z + 0.5 - centerZ) / radiusXZ;
                double horizontal = nx * nx + nz * nz;
                if (horizontal >= 1.0) {
                    continue;
                }

                int safeCeiling = column.groundBoundary() - 9;
                int columnMaxY = Math.min(maxY, safeCeiling);
                for (int y = minY; y <= columnMaxY; y++) {
                    double ny = (y + 0.5 - centerY) / radiusY;
                    if (horizontal + ny * ny > 1.0) {
                        continue;
                    }

                    pos.set(localX, y, localZ);
                    BlockState host = chunk.getBlockState(pos);
                    if (!host.is(Blocks.STONE) && !host.is(Blocks.DEEPSLATE)) {
                        continue;
                    }
                    blockSetter.set(chunk, pos, air);
                    carved[0]++;
                }
            }
        }
    }

    private static long cellSeed(long seed, int x, int y, int z) {
        long h = seed ^ 0x9E3779B97F4A7C15L;
        h ^= (long)x * 0xBF58476D1CE4E5B9L;
        h ^= (long)y * 0x94D049BB133111EBL;
        h ^= (long)z * 0xD6E8FEB86659FD93L;
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        return h ^ (h >>> 31);
    }
}
