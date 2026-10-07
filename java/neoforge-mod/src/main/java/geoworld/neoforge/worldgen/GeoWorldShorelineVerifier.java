package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3ProfileMapping;
import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;

public final class GeoWorldShorelineVerifier {
    private static final long SEED = 123456L;
    private static final int[][] REGRESSION_CENTERS = {
        {5594, -1216},
        {9769, -474},
        {7245, -4708}
    };
    private static final Set<String> SUGAR_CANE_SUBSTRATES = Set.of(
        "minecraft:grass_block",
        "minecraft:dirt",
        "minecraft:coarse_dirt",
        "minecraft:podzol",
        "minecraft:mud",
        "minecraft:sand"
    );

    private GeoWorldShorelineVerifier() {}

    public static void main(String[] args) throws Exception {
        Path isolatedCache = Files.createTempDirectory(
            "geoworld-shoreline-verifier-"
        );
        System.setProperty(
            "geoworld.cache.dir",
            isolatedCache.toAbsolutePath().toString()
        );
        System.setProperty("geoworld.engine.embeddedOnly", "true");

        GeoWorldEngineBridge.GeneratedModel generated =
            GeoWorldEngineBridge.generate(
                SEED,
                GeoWorldCreationConfig.defaults()
            );
        if (generated.cached()) {
            throw new AssertionError(
                "shoreline verifier must use a freshly generated model"
            );
        }
        GeoWorldV1 runtime = generated.model();

        ArrayList<int[]> scanCenters = new ArrayList<>();
        for (int[] center : REGRESSION_CENTERS) {
            scanCenters.add(center);
        }
        for (String waterKind : new String[]{"Lake", "Ocean"}) {
            int[] boundary = findBoundaryCenter(runtime, waterKind);
            if (boundary == null) {
                throw new AssertionError(
                    "could not find " + waterKind + " shoreline in seed " + SEED
                );
            }
            scanCenters.add(boundary);
        }

        int immediate = 0;
        int stable = 0;
        int shelves = 0;
        int materials = 0;
        int plantableEligible = 0;
        int plantable = 0;
        int maxLowering = 0;
        int maxRaising = 0;
        int maxRaiseX = 0;
        int maxRaiseZ = 0;
        String maxRaiseKind = "None";
        int maxRaiseOriginal = 0;
        int maxRaiseWater = 0;
        int raised1 = 0;
        int raised2to3 = 0;
        int raised4plus = 0;
        int materialEdges = 0;
        int materialChanges = 0;
        Map<String, Integer> waterKinds = new LinkedHashMap<>();
        Map<String, Integer> maxRaiseByKind = new LinkedHashMap<>();

        for (int[] center : scanCenters) {
            int centerChunkX = Math.floorDiv(center[0], 16);
            int centerChunkZ = Math.floorDiv(center[1], 16);
            for (int dz = -4; dz <= 4; dz++) {
                for (int dx = -4; dx <= 4; dx++) {
                    int baseX = (centerChunkX + dx) * 16;
                    int baseZ = (centerChunkZ + dz) * 16;
                    GeoWorldShoreline.ChunkPlan plan =
                        GeoWorldShoreline.planChunk(runtime, baseX, baseZ);

                    for (int localZ = 0; localZ < 16; localZ++) {
                        for (int localX = 0; localX < 16; localX++) {
                            GeoWorldShoreline.Column a =
                                plan.column(localX, localZ);
                            if (
                                "None".equals(a.sample().waterKind()) &&
                                a.shoreDistance() == 1 &&
                                a.surfaceOverride() != null
                            ) {
                                if (localX < 15) {
                                    GeoWorldShoreline.Column b =
                                        plan.column(localX + 1, localZ);
                                    if (
                                        "None".equals(b.sample().waterKind()) &&
                                        b.shoreDistance() == 1 &&
                                        b.surfaceOverride() != null &&
                                        a.shoreWaterKind().equals(
                                            b.shoreWaterKind()
                                        )
                                    ) {
                                        materialEdges++;
                                        if (!a.surfaceOverride().equals(
                                            b.surfaceOverride()
                                        )) {
                                            materialChanges++;
                                        }
                                    }
                                }
                                if (localZ < 15) {
                                    GeoWorldShoreline.Column b =
                                        plan.column(localX, localZ + 1);
                                    if (
                                        "None".equals(b.sample().waterKind()) &&
                                        b.shoreDistance() == 1 &&
                                        b.surfaceOverride() != null &&
                                        a.shoreWaterKind().equals(
                                            b.shoreWaterKind()
                                        )
                                    ) {
                                        materialEdges++;
                                        if (!a.surfaceOverride().equals(
                                            b.surfaceOverride()
                                        )) {
                                            materialChanges++;
                                        }
                                    }
                                }
                            }
                        }
                    }

                    for (int localZ = 0; localZ < 16; localZ++) {
                        for (int localX = 0; localX < 16; localX++) {
                            GeoWorldShoreline.Column column =
                                plan.column(localX, localZ);
                            if (
                                !"None".equals(column.sample().waterKind()) ||
                                !column.hasShore()
                            ) {
                                continue;
                            }

                            int lowering =
                                column.originalGroundBoundary() -
                                column.groundBoundary();
                            int raising =
                                column.groundBoundary() -
                                column.originalGroundBoundary();
                            maxLowering = Math.max(maxLowering, lowering);
                            if (raising > maxRaising) {
                                maxRaising = raising;
                                maxRaiseX = baseX + localX;
                                maxRaiseZ = baseZ + localZ;
                                maxRaiseKind = column.shoreWaterKind();
                                maxRaiseOriginal = column.originalGroundBoundary();
                                maxRaiseWater = column.shoreWaterBoundary();
                            }
                            if (raising == 1) {
                                raised1++;
                            } else if (raising >= 2 && raising <= 3) {
                                raised2to3++;
                            } else if (raising >= 4) {
                                raised4plus++;
                            }

                            if (lowering > 1) {
                                throw new AssertionError(
                                    "shore aesthetic lowering exceeded one block at " +
                                    (baseX + localX) + "," +
                                    (baseZ + localZ) +
                                    ": original=" +
                                    column.originalGroundBoundary() +
                                    " adjusted=" +
                                    column.groundBoundary()
                                );
                            }

                            if (column.littoralShelf()) {
                                shelves++;
                            }
                            if (column.surfaceOverride() != null) {
                                materials++;
                            }

                            waterKinds.merge(
                                column.shoreWaterKind(),
                                1,
                                Integer::sum
                            );
                            maxRaiseByKind.merge(
                                column.shoreWaterKind(),
                                raising,
                                Math::max
                            );

                            if (column.shoreDistance() != 1) {
                                continue;
                            }
                            immediate++;

                            if (
                                column.groundBoundary() <
                                column.shoreWaterBoundary()
                            ) {
                                throw new AssertionError(
                                    "dry shore below adjacent source water at " +
                                    (baseX + localX) + "," +
                                    (baseZ + localZ) +
                                    ": ground=" +
                                    column.groundBoundary() +
                                    " water=" +
                                    column.shoreWaterBoundary()
                                );
                            }
                            stable++;

                            if (!isPlantingClimate(column.sample())) {
                                continue;
                            }
                            plantableEligible++;
                            String top = column.surfaceOverride() != null
                                ? column.surfaceOverride()
                                : Minecraft26_3ProfileMapping.surfaceBlock(
                                    column.sample().surfaceProfile()
                                );
                            if (SUGAR_CANE_SUBSTRATES.contains(top)) {
                                plantable++;
                            }
                        }
                    }
                }
            }
        }

        System.out.printf(
            "GeoWorld shoreline scan: seed=%d, immediate=%d, stable=%d, " +
            "shelves=%d, materials=%d, caneReady=%d/%d, " +
            "maxLowering=%d, maxRaising=%d at %d,%d %s (%d->%d), " +
            "raiseBuckets=[1:%d,2-3:%d,4+:%d], materialChanges=%d/%d, " +
            "kinds=%s, maxByKind=%s%n",
            SEED,
            immediate,
            stable,
            shelves,
            materials,
            plantable,
            plantableEligible,
            maxLowering,
            maxRaising,
            maxRaiseX,
            maxRaiseZ,
            maxRaiseKind,
            maxRaiseOriginal,
            maxRaiseWater,
            raised1,
            raised2to3,
            raised4plus,
            materialChanges,
            materialEdges,
            waterKinds,
            maxRaiseByKind
        );

        if (immediate == 0) {
            throw new AssertionError(
                "regression windows no longer contain an immediate shoreline"
            );
        }
        if (shelves == 0) {
            throw new AssertionError(
                "shore planner produced no littoral shelves"
            );
        }
        if (materials == 0) {
            throw new AssertionError(
                "shore planner produced no material transitions"
            );
        }
        if (maxRaising > 2) {
            throw new AssertionError(
                "shore stability would create an artificial levee: max raise=" +
                maxRaising + " at " + maxRaiseX + "," + maxRaiseZ +
                " kind=" + maxRaiseKind
            );
        }
        for (String requiredKind : new String[]{"River", "Lake", "Ocean"}) {
            if (!waterKinds.containsKey(requiredKind)) {
                throw new AssertionError(
                    "shoreline verifier did not cover " + requiredKind
                );
            }
        }

        if (materialEdges > 0) {
            double changeRatio =
                (double)materialChanges / materialEdges;
            if (changeRatio > 0.24) {
                throw new AssertionError(
                    "shore material changes are too fragmented: " +
                    materialChanges + "/" + materialEdges +
                    " ratio=" + changeRatio
                );
            }
        }

        if (plantableEligible > 0) {
            double ratio = (double)plantable / plantableEligible;
            if (ratio < 0.60) {
                throw new AssertionError(
                    "too little planting-compatible immediate shoreline: " +
                    plantable + "/" + plantableEligible +
                    " ratio=" + ratio
                );
            }
        }

        System.out.printf(
            "GeoWorld shoreline verification passed: seed=%d, immediate=%d, " +
            "stable=%d, shelves=%d, materials=%d, caneReady=%d/%d, " +
            "maxLowering=%d, maxRaising=%d, kinds=%s%n",
            SEED,
            immediate,
            stable,
            shelves,
            materials,
            plantable,
            plantableEligible,
            maxLowering,
            maxRaising,
            waterKinds
        );
        cleanupCache(isolatedCache);
    }

    private static int[] findBoundaryCenter(
        GeoWorldV1 runtime,
        String targetKind
    ) {
        GeoWorldV1.Geometry geometry = runtime.manifest().geometry();
        int step = (int)Math.round(
            geometry.spanXBlocks() /
            Math.max(1.0, geometry.width() - 1.0)
        );
        int minX = (int)Math.round(geometry.originX()) + step;
        int minZ = (int)Math.round(geometry.originZ()) + step;
        int maxX = (int)Math.round(
            geometry.originX() + geometry.spanXBlocks()
        ) - step;
        int maxZ = (int)Math.round(
            geometry.originZ() + geometry.spanZBlocks()
        ) - step;

        for (int z = minZ; z <= maxZ; z += step) {
            for (int x = minX; x <= maxX; x += step) {
                String here = runtime.sample(x, z).waterKind();
                String east = runtime.sample(x + step, z).waterKind();
                if (
                    targetKind.equals(here) != targetKind.equals(east)
                ) {
                    return new int[]{x + step / 2, z};
                }
                String south = runtime.sample(x, z + step).waterKind();
                if (
                    targetKind.equals(here) != targetKind.equals(south)
                ) {
                    return new int[]{x, z + step / 2};
                }
            }
        }
        return null;
    }

    private static void cleanupCache(Path root) {
        try (var paths = Files.walk(root)) {
            paths.sorted(
                (a, b) -> Integer.compare(
                    b.getNameCount(),
                    a.getNameCount()
                )
            ).forEach(path -> {
                try {
                    Files.deleteIfExists(path);
                } catch (Exception ignored) {
                }
            });
        } catch (Exception ignored) {
        }
    }

    private static boolean isPlantingClimate(
        GeoWorldV1.RuntimeSample sample
    ) {
        return !"Alpine".equals(sample.biomeFamily()) &&
            !"Tundra".equals(sample.biomeFamily()) &&
            !"Ice Sheet".equals(sample.biomeFamily()) &&
            !"Alpine Rock".equals(sample.surfaceProfile()) &&
            !"Alpine Snow".equals(sample.surfaceProfile()) &&
            !"Ice / Snow".equals(sample.surfaceProfile());
    }
}
