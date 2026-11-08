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
import net.minecraft.world.level.levelgen.structure.StructurePiece;
import net.minecraft.world.level.levelgen.structure.PoolElementStructurePiece;
import net.minecraft.world.level.levelgen.structure.StructureStart;
import net.minecraft.world.level.levelgen.structure.TerrainAdjustment;
import net.minecraft.world.level.levelgen.structure.pools.StructureTemplatePool;

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

        boolean slopeAwareTransition =
            ("geoworld-rules-v1.1.15".equals(
                model.manifest().generationRulesVersion()
            ) || "geoworld-rules-v1.1.16".equals(
                model.manifest().generationRulesVersion()
            ) || "geoworld-rules-v1.1.18".equals(
                model.manifest().generationRulesVersion()
            ));
        AdaptationPolicy policy = policy(
            model.worldProfile().parameters().structureInfluence(),
            slopeAwareTransition
        );
        if (!policy.enabled()) return;

        int chunkMinX = chunk.getPos().getMinBlockX();
        int chunkMinZ = chunk.getPos().getMinBlockZ();
        int chunkMaxX = chunkMinX + 15;
        int chunkMaxZ = chunkMinZ + 15;

        var autoHeightmaps = chunk.getPersistedStatus().heightmapsAfter();
        Heightmap oceanFloor = autoHeightmaps.contains(
            Heightmap.Types.OCEAN_FLOOR_WG
        )
            ? null
            : chunk.getOrCreateHeightmapUnprimed(
                Heightmap.Types.OCEAN_FLOOR_WG
            );
        Heightmap worldSurface = autoHeightmaps.contains(
            Heightmap.Types.WORLD_SURFACE_WG
        )
            ? null
            : chunk.getOrCreateHeightmapUnprimed(
                Heightmap.Types.WORLD_SURFACE_WG
            );
        BlockPos.MutableBlockPos pos = new BlockPos.MutableBlockPos();

        for (StructureStart start : starts) {
            if (start == null || !start.isValid()) {
                continue;
            }

            TerrainAdjustment adjustment =
                start.getStructure().terrainAdaptation();
            if (adjustment == TerrainAdjustment.NONE) {
                continue;
            }
            if (adjustment == TerrainAdjustment.BEARD_THIN) {
                populateBeardThin(
                    chunk,
                    model,
                    start,
                    shoreline,
                    heights,
                    blockSetter,
                    policy,
                    oceanFloor,
                    worldSurface,
                    pos
                );
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

            int margin = policy.margin();
            int minX = Math.max(chunkMinX, box.minX() - margin);
            int maxX = Math.min(chunkMaxX, box.maxX() + margin);
            int minZ = Math.max(chunkMinZ, box.minZ() - margin);
            int maxZ = Math.min(chunkMaxZ, box.maxZ() + margin);
            if (minX > maxX || minZ > maxZ) continue;

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
                    int targetBoundary = slopeAwareTransition
                        ? adaptedBoundarySlopeAware(
                            naturalBoundary,
                            centerBoundary,
                            edgeDistance,
                            policy
                        )
                        : adaptedBoundary(
                            naturalBoundary,
                            centerBoundary,
                            localWeight,
                            policy
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

    /**
     * Vanilla BEARD_THIN is piece-local: Beardifier evaluates each rigid
     * structure piece (and its jigsaw junctions) instead of flattening the
     * complete StructureStart bounding box toward one global center height.
     *
     * GeoWorld keeps that locality while applying the existing bounded
     * structure-influence policy to its block terrain. This is especially
     * important for large jigsaw structures such as abandoned camps: a camp
     * may span a wide slope, but no distant piece is allowed to turn the whole
     * start into one artificial terrace.
     */
    private static void populateBeardThin(
        ChunkAccess chunk,
        GeoWorldV1 model,
        StructureStart start,
        GeoWorldShorelinePlanner.ChunkPlan shoreline,
        HeightMapping heights,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter,
        AdaptationPolicy policy,
        Heightmap oceanFloor,
        Heightmap worldSurface,
        BlockPos.MutableBlockPos pos
    ) {
        int chunkMinX = chunk.getPos().getMinBlockX();
        int chunkMinZ = chunk.getPos().getMinBlockZ();
        int chunkMaxX = chunkMinX + 15;
        int chunkMaxZ = chunkMinZ + 15;
        int margin = beardThinMargin(policy);

        double[] bestWeight = new double[256];
        int[] naturalBoundaries = new int[256];
        int[] targetBoundaries = new int[256];
        boolean[] planned = new boolean[256];

        for (StructurePiece piece : start.getPieces()) {
            if (
                piece == null ||
                !piece.isCloseToChunk(chunk.getPos(), margin)
            ) {
                continue;
            }

            int pieceGroundDelta = 0;
            if (piece instanceof PoolElementStructurePiece poolPiece) {
                if (
                    poolPiece.getElement().getProjection() !=
                        StructureTemplatePool.Projection.RIGID
                ) {
                    // TERRAIN_MATCHING pieces already carry the Vanilla
                    // gravity processor and must follow the natural surface.
                    continue;
                }
                pieceGroundDelta = poolPiece.getGroundLevelDelta();
            }

            BoundingBox box = piece.getBoundingBox();
            int centerX = (box.minX() + box.maxX()) / 2;
            int centerZ = (box.minZ() + box.maxZ()) / 2;
            GeoWorldV1.RuntimeSample center = model.sample(
                centerX,
                centerZ
            );
            if (!eligible(center)) {
                continue;
            }

            int centerNaturalBoundary = heights.boundaryY(
                center.groundHeight()
            );
            if (box.maxY() < centerNaturalBoundary - 10) {
                // Preserve underground pieces exactly as before.
                continue;
            }

            int pieceBoundary = clamp(
                box.minY() + pieceGroundDelta,
                heights.minY() + 2,
                heights.maxYExclusive() - 1
            );
            int minX = Math.max(chunkMinX, box.minX() - margin);
            int maxX = Math.min(chunkMaxX, box.maxX() + margin);
            int minZ = Math.max(chunkMinZ, box.minZ() - margin);
            int maxZ = Math.min(chunkMaxZ, box.maxZ() + margin);
            if (minX > maxX || minZ > maxZ) {
                continue;
            }

            for (int worldZ = minZ; worldZ <= maxZ; worldZ++) {
                for (int worldX = minX; worldX <= maxX; worldX++) {
                    int localX = worldX - chunkMinX;
                    int localZ = worldZ - chunkMinZ;
                    int index = localZ * 16 + localX;
                    GeoWorldShorelinePlanner.Column column =
                        shoreline.column(localX, localZ);
                    if (!eligible(column.sample())) {
                        continue;
                    }

                    double edgeDistance = distanceOutside(
                        worldX,
                        worldZ,
                        box
                    );
                    double localWeight = beardThinWeight(
                        edgeDistance,
                        margin
                    );
                    if (
                        localWeight <= 0.0 ||
                        localWeight <= bestWeight[index]
                    ) {
                        continue;
                    }

                    int naturalBoundary = column.groundBoundary();
                    int targetBoundary = beardThinBoundary(
                        naturalBoundary,
                        pieceBoundary,
                        edgeDistance,
                        policy
                    );
                    targetBoundary = clamp(
                        targetBoundary,
                        heights.minY() + 2,
                        heights.maxYExclusive() - 1
                    );

                    bestWeight[index] = localWeight;
                    naturalBoundaries[index] = naturalBoundary;
                    targetBoundaries[index] = targetBoundary;
                    planned[index] = targetBoundary != naturalBoundary;
                }
            }
        }

        for (int localZ = 0; localZ < 16; localZ++) {
            for (int localX = 0; localX < 16; localX++) {
                int index = localZ * 16 + localX;
                if (!planned[index]) {
                    continue;
                }
                adaptColumn(
                    chunk,
                    blockSetter,
                    oceanFloor,
                    worldSurface,
                    pos,
                    chunkMinX + localX,
                    chunkMinZ + localZ,
                    localX,
                    localZ,
                    naturalBoundaries[index],
                    targetBoundaries[index]
                );
            }
        }
    }

    static int beardThinMargin(AdaptationPolicy policy) {
        // Vanilla's beard kernel has radius 12. Keep at least that much room
        // for a smooth local fade, while still honoring a stronger user
        // structure-influence margin when it exceeds Vanilla's radius.
        return Math.max(12, policy.margin());
    }

    static double beardThinWeight(
        double edgeDistance,
        int margin
    ) {
        if (margin <= 0 || edgeDistance >= margin) {
            return 0.0;
        }
        return 1.0 - smoothstep(
            edgeDistance / Math.max(1.0, margin)
        );
    }

    static int beardThinBoundary(
        int naturalBoundary,
        int pieceBoundary,
        double edgeDistance,
        AdaptationPolicy policy
    ) {
        int margin = beardThinMargin(policy);
        double localWeight = beardThinWeight(
            edgeDistance,
            margin
        );
        return adaptedBoundary(
            naturalBoundary,
            pieceBoundary,
            localWeight,
            policy
        );
    }
    static AdaptationPolicy policy(double influence) {
        return policy(influence, false);
    }

    static AdaptationPolicy policy(
        double influence,
        boolean slopeAwareTransition
    ) {
        if (influence <= 1.0e-9) {
            return new AdaptationPolicy(false, 0, 0.0, 0);
        }
        double bounded = Math.max(0.0, Math.min(2.5, influence));
        double strength = Math.min(0.78, 0.18 + 0.24 * bounded);
        int maxShift = Math.max(1, (int)Math.round(2 + 2 * bounded));
        int baseMargin = Math.max(
            2,
            (int)Math.round(2 + bounded * 2)
        );
        int margin = slopeAwareTransition
            ? Math.max(baseMargin, maxShift * 2 + 2)
            : baseMargin;
        return new AdaptationPolicy(
            true,
            margin,
            strength,
            maxShift
        );
    }

    static int adaptedBoundarySlopeAware(
        int naturalBoundary,
        int centerBoundary,
        double edgeDistance,
        AdaptationPolicy policy
    ) {
        if (!policy.enabled()) {
            return naturalBoundary;
        }
        double normalized = Math.max(
            0.0,
            Math.min(
                1.0,
                edgeDistance / Math.max(1.0, policy.margin())
            )
        );
        double localWeight = 1.0 - smoothstep(normalized);

        int fullShift = clamp(
            (int)Math.round(
                (centerBoundary - naturalBoundary) * policy.strength()
            ),
            -policy.maxShift(),
            policy.maxShift()
        );
        int taperedShift = (int)Math.round(fullShift * localWeight);
        return naturalBoundary + taperedShift;
    }

    static int adaptedBoundary(
        int naturalBoundary,
        int centerBoundary,
        double localWeight,
        AdaptationPolicy policy
    ) {
        if (!policy.enabled() || localWeight <= 0.0) {
            return naturalBoundary;
        }
        double weight = Math.max(0.0, Math.min(1.0, localWeight));
        int requested = (int)Math.round(
            naturalBoundary +
            (centerBoundary - naturalBoundary) *
                policy.strength() *
                weight
        );
        return clamp(
            requested,
            naturalBoundary - policy.maxShift(),
            naturalBoundary + policy.maxShift()
        );
    }

    record AdaptationPolicy(
        boolean enabled,
        int margin,
        double strength,
        int maxShift
    ) {}

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
        if (oceanFloor != null) {
            oceanFloor.update(localX, y, localZ, state);
        }
        if (worldSurface != null) {
            worldSurface.update(localX, y, localZ, state);
        }
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
