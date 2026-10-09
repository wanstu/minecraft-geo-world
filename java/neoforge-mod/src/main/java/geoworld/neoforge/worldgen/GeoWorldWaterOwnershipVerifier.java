package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.HashSet;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;

public final class GeoWorldWaterOwnershipVerifier {
    private static final long SEED = 123456L;
    private static final int GLOBAL_STRIDE_BLOCKS = 2048;

    private static final int[][] REGRESSION_CENTERS = {
        {5594, -1216},
        {9769, -474},
        {7245, -4708},
        {2023, -2985},
        {2100, -3010},
        {10883, -1293},
        {10325, 2673},
        {10824, 2515},
        {10429, -1125},
        {8000, -576},
        {9152, -832},
        {8128, -704}
    };

    private static final Minecraft26_3HeightMapping HEIGHTS =
        new Minecraft26_3HeightMapping();

    private GeoWorldWaterOwnershipVerifier() {}

    public static void main(String[] args) throws Exception {
        Path isolatedCache = Path.of(
            "build",
            "geoworld-water-ownership-verifier-cache"
        ).toAbsolutePath();
        cleanupCache(isolatedCache);
        Files.createDirectories(isolatedCache);
        System.setProperty(
            "geoworld.cache.dir",
            isolatedCache.toString()
        );
        System.setProperty("geoworld.engine.embeddedOnly", "true");

        GeoWorldEngineBridge.GeneratedModel generated =
            GeoWorldEngineBridge.generate(
                SEED,
                GeoWorldCreationConfig.defaults()
            );
        if (generated.cached()) {
            throw new AssertionError(
                "water ownership verifier must use a fresh model"
            );
        }

        GeoWorldV1 runtime = generated.model();
        Stats stats = new Stats();
        Set<Long> scannedChunks = new HashSet<>();

        for (int[] center : REGRESSION_CENTERS) {
            int centerChunkX = Math.floorDiv(center[0], 16);
            int centerChunkZ = Math.floorDiv(center[1], 16);
            for (int dz = -4; dz <= 4; dz++) {
                for (int dx = -4; dx <= 4; dx++) {
                    scanChunkOnce(
                        runtime,
                        centerChunkX + dx,
                        centerChunkZ + dz,
                        scannedChunks,
                        stats
                    );
                }
            }
        }

        GeoWorldV1.Geometry geometry = runtime.manifest().geometry();
        int minX = (int) Math.floor(geometry.originX());
        int minZ = (int) Math.floor(geometry.originZ());
        int maxX = (int) Math.ceil(
            geometry.originX() + geometry.spanXBlocks()
        );
        int maxZ = (int) Math.ceil(
            geometry.originZ() + geometry.spanZBlocks()
        );

        for (
            int z = minZ + GLOBAL_STRIDE_BLOCKS / 2;
            z <= maxZ;
            z += GLOBAL_STRIDE_BLOCKS
        ) {
            for (
                int x = minX + GLOBAL_STRIDE_BLOCKS / 2;
                x <= maxX;
                x += GLOBAL_STRIDE_BLOCKS
            ) {
                scanChunkOnce(
                    runtime,
                    Math.floorDiv(x, 16),
                    Math.floorDiv(z, 16),
                    scannedChunks,
                    stats
                );
            }
        }

        System.out.printf(
            "GeoWorld final water ownership: seed=%d, chunks=%d, columns=%d, " +
            "waterColumns=%d, owners=%s, " +
            "unownedSurfaceWaterSources=%d, unsafeExposedWaterSources=%d, " +
            "freeFallWaterFaces=%d, runtimeWaterLost=%d, " +
            "riverWaterBoundaryModelMismatches=%d, riverLocalWaterPeaks=%d, " +
            "wetlandPoolLeaks=%d%n",
            SEED,
            scannedChunks.size(),
            stats.columns,
            stats.waterColumns,
            stats.owners,
            stats.unownedSurfaceWaterSources,
            stats.unsafeExposedWaterSources,
            stats.freeFallWaterFaces,
            stats.runtimeWaterLost,
            stats.riverWaterBoundaryModelMismatches,
            stats.riverLocalWaterPeaks,
            stats.wetlandPoolLeaks
        );

        if (stats.unownedSurfaceWaterSources != 0) {
            throw new AssertionError(
                "final terrain contains unowned surface water: " +
                stats.unownedSurfaceWaterSources +
                exampleSuffix(stats.firstUnowned)
            );
        }
        if (stats.unsafeExposedWaterSources != 0) {
            throw new AssertionError(
                "final terrain contains water with an open dry escape: " +
                stats.unsafeExposedWaterSources +
                exampleSuffix(stats.firstUnsafe)
            );
        }
        if (stats.freeFallWaterFaces != 0) {
            throw new AssertionError(
                "final terrain contains unsupported multi-block water faces: " +
                stats.freeFallWaterFaces +
                exampleSuffix(stats.firstFreeFall)
            );
        }
        if (stats.runtimeWaterLost != 0) {
            throw new AssertionError(
                "Runtime-owned water disappeared during voxel planning: " +
                stats.runtimeWaterLost +
                exampleSuffix(stats.firstLost)
            );
        }
        if (stats.riverWaterBoundaryModelMismatches != 0) {
            throw new AssertionError(
                "river voxel surface diverges from Runtime water surface: " +
                stats.riverWaterBoundaryModelMismatches +
                exampleSuffix(stats.firstRiverMismatch)
            );
        }
        if (stats.riverLocalWaterPeaks != 0) {
            throw new AssertionError(
                "river contains isolated one-block-high surface peaks: " +
                stats.riverLocalWaterPeaks +
                exampleSuffix(stats.firstRiverPeak)
            );
        }
        if (stats.wetlandPoolLeaks != 0) {
            throw new AssertionError(
                "wetland pool has an uncontained dry-side escape: " +
                stats.wetlandPoolLeaks +
                exampleSuffix(stats.firstWetlandLeak)
            );
        }
    }

    private static void scanChunkOnce(
        GeoWorldV1 runtime,
        int chunkX,
        int chunkZ,
        Set<Long> scannedChunks,
        Stats stats
    ) {
        long key =
            ((long) chunkX << 32) ^ (chunkZ & 0xffffffffL);
        if (!scannedChunks.add(key)) {
            return;
        }

        int baseX = chunkX * 16;
        int baseZ = chunkZ * 16;
        GeoWorldShoreline.ChunkPlan plan =
            GeoWorldShoreline.planChunk(runtime, baseX, baseZ);

        stats.columns += 16 * 16;

        for (int localZ = 0; localZ < 16; localZ++) {
            for (int localX = 0; localX < 16; localX++) {
                int worldX = baseX + localX;
                int worldZ = baseZ + localZ;
                GeoWorldShoreline.Column center =
                    plan.column(localX, localZ);
                auditColumn(center, worldX, worldZ, stats);

                if (localX < 15) {
                    auditFace(
                        center,
                        plan.column(localX + 1, localZ),
                        worldX,
                        worldZ,
                        worldX + 1,
                        worldZ,
                        stats
                    );
                }
                if (localZ < 15) {
                    auditFace(
                        center,
                        plan.column(localX, localZ + 1),
                        worldX,
                        worldZ,
                        worldX,
                        worldZ + 1,
                        stats
                    );
                }

                if (
                    localX > 0 &&
                    localX < 15 &&
                    localZ > 0 &&
                    localZ < 15 &&
                    "River".equals(owner(center))
                ) {
                    GeoWorldShoreline.Column west =
                        plan.column(localX - 1, localZ);
                    GeoWorldShoreline.Column east =
                        plan.column(localX + 1, localZ);
                    GeoWorldShoreline.Column north =
                        plan.column(localX, localZ - 1);
                    GeoWorldShoreline.Column south =
                        plan.column(localX, localZ + 1);
                    if (
                        isRiverOneLower(west, center) &&
                        isRiverOneLower(east, center) ||
                        isRiverOneLower(north, center) &&
                        isRiverOneLower(south, center)
                    ) {
                        stats.riverLocalWaterPeaks++;
                        if (stats.firstRiverPeak == null) {
                            stats.firstRiverPeak =
                                worldX + "," + worldZ;
                        }
                    }
                }
            }
        }
    }

    private static void auditColumn(
        GeoWorldShoreline.Column column,
        int worldX,
        int worldZ,
        Stats stats
    ) {
        boolean generatedWater =
            column.waterBoundary() > column.groundBoundary();
        String owner = owner(column);

        if (generatedWater) {
            stats.waterColumns++;
            stats.owners.merge(owner, 1L, Long::sum);

            if ("Unowned".equals(owner)) {
                stats.unownedSurfaceWaterSources++;
                if (stats.firstUnowned == null) {
                    stats.firstUnowned = worldX + "," + worldZ;
                }
            }
        }

        boolean runtimeOwnsWater =
            runtimeOwnsVoxelWater(column);
        if (runtimeOwnsWater && !generatedWater) {
            stats.runtimeWaterLost++;
            if (stats.firstLost == null) {
                stats.firstLost = worldX + "," + worldZ +
                    " kind=" + column.sample().waterKind();
            }
        }

        if (
            generatedWater &&
            "River".equals(owner)
        ) {
            int expected = HEIGHTS.boundaryYCeil(
                column.sample().waterSurfaceElevation()
            );
            if (column.waterBoundary() != expected) {
                stats.riverWaterBoundaryModelMismatches++;
                if (stats.firstRiverMismatch == null) {
                    stats.firstRiverMismatch =
                        worldX + "," + worldZ +
                        " actual=" + column.waterBoundary() +
                        " expected=" + expected;
                }
            }
        }
    }

    private static void auditFace(
        GeoWorldShoreline.Column a,
        GeoWorldShoreline.Column b,
        int ax,
        int az,
        int bx,
        int bz,
        Stats stats
    ) {
        auditDirectedFace(a, b, ax, az, bx, bz, stats);
        auditDirectedFace(b, a, bx, bz, ax, az, stats);
    }

    private static void auditDirectedFace(
        GeoWorldShoreline.Column water,
        GeoWorldShoreline.Column neighbor,
        int waterX,
        int waterZ,
        int neighborX,
        int neighborZ,
        Stats stats
    ) {
        if (water.waterBoundary() <= water.groundBoundary()) {
            return;
        }

        boolean neighborWater =
            neighbor.waterBoundary() > neighbor.groundBoundary();
        int neighborTop = Math.max(
            neighbor.groundBoundary(),
            neighborWater
                ? neighbor.waterBoundary()
                : neighbor.groundBoundary()
        );
        int exposedDrop =
            water.waterBoundary() - neighborTop;

        if (!neighborWater && exposedDrop > 0) {
            stats.unsafeExposedWaterSources++;
            if (stats.firstUnsafe == null) {
                stats.firstUnsafe =
                    waterX + "," + waterZ +
                    " -> " + neighborX + "," + neighborZ +
                    " owner=" + owner(water) +
                    " water=" + water.waterBoundary() +
                    " neighborGround=" + neighbor.groundBoundary();
            }

            if ("WetlandPool".equals(owner(water))) {
                stats.wetlandPoolLeaks++;
                if (stats.firstWetlandLeak == null) {
                    stats.firstWetlandLeak =
                        waterX + "," + waterZ +
                        " -> " + neighborX + "," + neighborZ;
                }
            }
        }

        if (exposedDrop > 1) {
            stats.freeFallWaterFaces++;
            if (stats.firstFreeFall == null) {
                stats.firstFreeFall =
                    waterX + "," + waterZ +
                    " -> " + neighborX + "," + neighborZ +
                    " drop=" + exposedDrop +
                    " owner=" + owner(water);
            }
        }
    }

    private static boolean runtimeOwnsVoxelWater(
        GeoWorldShoreline.Column column
    ) {
        GeoWorldV1.RuntimeSample sample = column.sample();
        if (
            "None".equals(sample.waterKind()) ||
            !sample.waterSurfaceDefined() ||
            sample.waterSurfaceElevation() == null
        ) {
            return false;
        }
        if ("River".equals(sample.waterKind())) {
            return true;
        }
        return HEIGHTS.boundaryY(sample.waterSurfaceElevation()) >
            column.originalGroundBoundary();
    }

    private static boolean isRiverOneLower(
        GeoWorldShoreline.Column candidate,
        GeoWorldShoreline.Column center
    ) {
        return "River".equals(owner(candidate)) &&
            candidate.waterBoundary() ==
                center.waterBoundary() - 1;
    }

    private static String owner(
        GeoWorldShoreline.Column column
    ) {
        if (
            column.sample().waterSurfaceDefined() &&
            switch (column.sample().waterKind()) {
                case "River", "Lake", "Ocean" -> true;
                default -> false;
            }
        ) {
            return column.sample().waterKind();
        }
        if (
            "None".equals(column.sample().waterKind()) &&
            "WetlandPool".equals(column.shoreWaterKind()) &&
            column.waterBoundary() > column.groundBoundary()
        ) {
            return "WetlandPool";
        }
        return column.waterBoundary() > column.groundBoundary()
            ? "Unowned"
            : "Dry";
    }

    private static String exampleSuffix(String example) {
        return example == null ? "" : " example=" + example;
    }

    private static void cleanupCache(Path root) throws Exception {
        if (!Files.exists(root)) {
            return;
        }
        try (var paths = Files.walk(root)) {
            paths.sorted((a, b) -> b.compareTo(a))
                .forEach(path -> {
                    try {
                        Files.deleteIfExists(path);
                    } catch (Exception e) {
                        throw new RuntimeException(e);
                    }
                });
        }
    }

    private static final class Stats {
        long columns;
        long waterColumns;
        long unownedSurfaceWaterSources;
        long unsafeExposedWaterSources;
        long freeFallWaterFaces;
        long runtimeWaterLost;
        long riverWaterBoundaryModelMismatches;
        long riverLocalWaterPeaks;
        long wetlandPoolLeaks;

        final Map<String, Long> owners = new LinkedHashMap<>();

        String firstUnowned;
        String firstUnsafe;
        String firstFreeFall;
        String firstLost;
        String firstRiverMismatch;
        String firstRiverPeak;
        String firstWetlandLeak;
    }
}
