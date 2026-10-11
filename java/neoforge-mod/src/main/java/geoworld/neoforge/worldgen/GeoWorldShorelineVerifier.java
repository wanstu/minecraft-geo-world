package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3BiomeRealizer;
import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
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
        {7245, -4708},
        {2023, -2985},
        {2100, -3010},
        {10883, -1293},
        {10325, 2673},
        {10824, 2515}
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
        Path isolatedCache = Path.of(
            "build",
            "geoworld-shoreline-verifier-cache"
        ).toAbsolutePath();
        cleanupCache(isolatedCache);
        Files.createDirectories(isolatedCache);
        System.setProperty(
            "geoworld.cache.dir",
            isolatedCache.toAbsolutePath().toString()
        );
        System.setProperty("geoworld.engine.embeddedOnly", "true");

        verifyGentleBankProfile();
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
        int shallowFringes = 0;
        int shelves = 0;
        int materials = 0;
        int plantableEligible = 0;
        int plantable = 0;
        int maxLowering = 0;
        int gentleBankColumns = 0;
        int inlandGentleBankColumns = 0;
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
        int narrowBackshoreGrooves = 0;
        int smallBackshorePits = 0;
        int oceanBackshoreStepViolations = 0;
        int riverWaterBoundaryModelMismatches = 0;
        int riverDryVoxelColumns = 0;
        int riverBedQuantizationLowers = 0;
        int pointPlanMismatches = 0;
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

                    for (int checkZ : new int[]{2, 8, 13}) {
                        for (int checkX : new int[]{2, 8, 13}) {
                            GeoWorldShoreline.Column chunkColumn =
                                plan.column(checkX, checkZ);
                            GeoWorldShoreline.Column pointColumn =
                                GeoWorldShoreline.plan(
                                    runtime,
                                    baseX + checkX,
                                    baseZ + checkZ
                                );
                            if (!samePlannedColumn(
                                chunkColumn,
                                pointColumn
                            )) {
                                pointPlanMismatches++;
                            }
                        }
                    }

                    for (int localZ = 1; localZ < 15; localZ++) {
                        for (int localX = 1; localX < 15; localX++) {
                            GeoWorldShoreline.Column centerColumn =
                                plan.column(localX, localZ);
                            if (
                                "None".equals(
                                    centerColumn.sample().waterKind()
                                ) &&
                                centerColumn.shoreDistance() >= 2 &&
                                centerColumn.shoreDistance() <= 3
                            ) {
                                boolean grooveX = isBackshoreGroove(
                                    centerColumn,
                                    plan.column(localX - 1, localZ),
                                    plan.column(localX + 1, localZ)
                                );
                                boolean grooveZ = isBackshoreGroove(
                                    centerColumn,
                                    plan.column(localX, localZ - 1),
                                    plan.column(localX, localZ + 1)
                                );
                                if (grooveX || grooveZ) {
                                    narrowBackshoreGrooves++;
                                }
                            }
                            if (
                                isSmallBackshorePit(
                                    centerColumn,
                                    plan.column(localX - 1, localZ),
                                    plan.column(localX + 1, localZ),
                                    plan.column(localX, localZ - 1),
                                    plan.column(localX, localZ + 1)
                                )
                            ) {
                                smallBackshorePits++;
                            }
                            if (
                                "None".equals(
                                    centerColumn.sample().waterKind()
                                ) &&
                                "Ocean".equals(
                                    centerColumn.shoreWaterKind()
                                ) &&
                                centerColumn.shoreDistance() >= 1 &&
                                centerColumn.shoreDistance() <= 3
                            ) {
                                oceanBackshoreStepViolations +=
                                    oceanBackshoreViolations(
                                        centerColumn,
                                        plan.column(localX - 1, localZ),
                                        plan.column(localX + 1, localZ),
                                        plan.column(localX, localZ - 1),
                                        plan.column(localX, localZ + 1)
                                    );
                            }
                        }
                    }

                    Minecraft26_3HeightMapping heights =
                        new Minecraft26_3HeightMapping();
                    for (int localZ = 0; localZ < 16; localZ++) {
                        for (int localX = 0; localX < 16; localX++) {
                            GeoWorldShoreline.Column riverColumn =
                                plan.column(localX, localZ);
                            if (
                                "River".equals(
                                    riverColumn.sample().waterKind()
                                )
                            ) {
                                int expectedWater =
                                    heights.boundaryYCeil(
                                        riverColumn.sample()
                                            .waterSurfaceElevation()
                                    );
                                if (
                                    riverColumn.waterBoundary() !=
                                    expectedWater
                                ) {
                                    riverWaterBoundaryModelMismatches++;
                                }
                                if (
                                    riverColumn.waterBoundary() <=
                                    riverColumn.groundBoundary()
                                ) {
                                    riverDryVoxelColumns++;
                                }
                                if (
                                    riverColumn.groundBoundary() <
                                    riverColumn.originalGroundBoundary()
                                ) {
                                    riverBedQuantizationLowers++;
                                }
                            }
                        }
                    }

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
                            if ("None".equals(column.sample().waterKind()) &&
                                ("Lake".equals(column.shoreWaterKind()) ||
                                 "Ocean".equals(column.shoreWaterKind())) &&
                                column.shoreDistance() <=
                                    GeoWorldShoreline.MAX_GENTLE_BANK_DISTANCE &&
                                column.groundBoundary() <
                                    column.originalGroundBoundary()) {
                                gentleBankColumns++;
                                if (!column.hasShore()) inlandGentleBankColumns++;
                            }
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

                            if (lowering > (
                                    ("Lake".equals(column.shoreWaterKind()) ||
                                     "Ocean".equals(column.shoreWaterKind()))
                                        ? 31 : 1)) {
                                throw new AssertionError(
                                    "shore taper lowered terrain beyond safety bound at " +
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
                                column.waterBoundary() >
                                column.groundBoundary()
                            ) {
                                if (
                                    column.waterBoundary() !=
                                    column.shoreWaterBoundary() ||
                                    column.waterBoundary() -
                                    column.groundBoundary() != 1
                                ) {
                                    throw new AssertionError(
                                        "shore fringe water is not a one-block " +
                                        "source shelf at " +
                                        (baseX + localX) + "," +
                                        (baseZ + localZ) +
                                        ": ground=" +
                                        column.groundBoundary() +
                                        " water=" +
                                        column.waterBoundary() +
                                        " source=" +
                                        column.shoreWaterBoundary()
                                    );
                                }
                                shallowFringes++;
                                stable++;
                                continue;
                            }

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

        int localRiverMaterialGaps = 0;
        for (int z = 2515 - 48; z <= 2515 + 48; z++) {
            for (int x = 10824 - 48; x <= 10824 + 48; x++) {
                GeoWorldShoreline.Column column =
                    GeoWorldShoreline.plan(runtime, x, z);
                if (
                    "None".equals(column.sample().waterKind()) &&
                    "River".equals(column.shoreWaterKind()) &&
                    column.shoreDistance() == 1 &&
                    column.surfaceOverride() == null
                ) {
                    localRiverMaterialGaps++;
                }
            }
        }

        System.out.printf(
            "GeoWorld shoreline scan: seed=%d, immediate=%d, stable=%d, " +
            "shallowFringes=%d, shelves=%d, materials=%d, caneReady=%d/%d, " +
            "maxLowering=%d, gentleBanks=%d, inlandGentleBanks=%d, " +
            "maxRaising=%d at %d,%d %s (%d->%d), " +
            "raiseBuckets=[1:%d,2-3:%d,4+:%d], materialChanges=%d/%d, " +
            "narrowBackshoreGrooves=%d, smallBackshorePits=%d, " +
            "oceanBackshoreStepViolations=%d, " +
            "riverWaterBoundaryModelMismatches=%d, " +
            "riverDryVoxelColumns=%d, riverBedQuantizationLowers=%d, " +
            "pointPlanMismatches=%d, kinds=%s, maxByKind=%s%n",
            SEED,
            immediate,
            stable,
            shallowFringes,
            shelves,
            materials,
            plantable,
            plantableEligible,
            maxLowering,
            gentleBankColumns,
            inlandGentleBankColumns,
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
            narrowBackshoreGrooves,
            smallBackshorePits,
            oceanBackshoreStepViolations,
            riverWaterBoundaryModelMismatches,
            riverDryVoxelColumns,
            riverBedQuantizationLowers,
            pointPlanMismatches,
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
        if (narrowBackshoreGrooves != 0) {
            throw new AssertionError(
                "shore planner left narrow backshore grooves: " +
                narrowBackshoreGrooves
            );
        }
        if (smallBackshorePits != 0) {
            throw new AssertionError(
                "shore planner left small backshore pits: " +
                smallBackshorePits
            );
        }
        if (oceanBackshoreStepViolations != 0) {
            throw new AssertionError(
                "ocean backshore did not smooth inward: " +
                oceanBackshoreStepViolations
            );
        }
        if (localRiverMaterialGaps != 0) {
            throw new AssertionError(
                "riverbank regression window contains material gaps: " +
                localRiverMaterialGaps
            );
        }
        if (riverWaterBoundaryModelMismatches != 0) {
            throw new AssertionError(
                "river voxel water surface diverged from Runtime model: " +
                riverWaterBoundaryModelMismatches
            );
        }
        if (riverDryVoxelColumns != 0) {
            throw new AssertionError(
                "Runtime river columns lost their voxel water layer: " +
                riverDryVoxelColumns
            );
        }
        if (pointPlanMismatches != 0) {
            throw new AssertionError(
                "chunk and point shoreline plans disagree: " +
                pointPlanMismatches
            );
        }
        if (
            maxRaiseByKind.getOrDefault("River", 0) > 2 ||
            maxRaiseByKind.getOrDefault("Lake", 0) > 2 ||
            maxRaiseByKind.getOrDefault("Ocean", 0) > 3
        ) {
            throw new AssertionError(
                "shore stability exceeded per-water-body raise cap: max raise=" +
                maxRaising + " at " + maxRaiseX + "," + maxRaiseZ +
                " kind=" + maxRaiseKind +
                " maxByKind=" + maxRaiseByKind
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

        verifyWetlandPools(runtime);
        verifyRiverbankBiomes(runtime);

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

    private static void verifyGentleBankProfile() {
        for (int relief : new int[]{8, 16, 24, 32}) {
            int water = 63;
            int previous = water;
            for (int distance = 1;
                 distance <= GeoWorldShoreline.MAX_GENTLE_BANK_DISTANCE;
                 distance++) {
                int capped = GeoWorldShoreline.gentleBankCap(
                    water + relief, water, distance
                );
                if (capped < previous || capped > water + relief ||
                    capped - previous > 6) {
                    throw new AssertionError(
                        "lake/ocean gentle shore profile is discontinuous: " +
                        "relief=" + relief + " d=" + distance +
                        " prev=" + previous + " cap=" + capped
                    );
                }
                previous = capped;
            }
            if (previous != water + relief) {
                throw new AssertionError(
                    "shore transition does not rejoin natural ground"
                );
            }
        }
        System.out.println("GeoWorld moderate lake/ocean bank taper PASS");
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

    private static void verifyRiverbankBiomes(
        GeoWorldV1 runtime
    ) {
        int regressionX = 4732;
        int regressionZ = -1470;
        String regressionBiome = Minecraft26_3BiomeRealizer.biomeId(
            runtime,
            regressionX,
            regressionZ
        );
        if (isSwampBiome(regressionBiome)) {
            throw new AssertionError(
                "riverbank regression is still swamp at " +
                regressionX + "," + regressionZ +
                ": " + regressionBiome
            );
        }

        int bankCells = 0;
        int swampCells = 0;
        int unsupportedSwamps = 0;
        int[][] centers = {
            {4732, -1470},
            {1724, -3024},
            {1855, -2889},
            {1862, -2631}
        };
        for (int[] center : centers) {
            for (int dz = -96; dz <= 96; dz += 4) {
                for (int dx = -96; dx <= 96; dx += 4) {
                    int x = center[0] + dx;
                    int z = center[1] + dz;
                    GeoWorldV1.RuntimeSample sample =
                        runtime.sample(x, z);
                    if (
                        !"None".equals(sample.waterKind()) ||
                        !nearRiver(runtime, x, z, 10)
                    ) {
                        continue;
                    }
                    bankCells++;
                    String biome = Minecraft26_3BiomeRealizer.biomeId(
                        runtime,
                        x,
                        z
                    );
                    if (!isSwampBiome(biome)) {
                        continue;
                    }
                    swampCells++;
                    boolean trueWetland =
                        "Wetland".equals(sample.biomeFamily()) ||
                        "Peat".equals(sample.surfaceProfile()) ||
                        "Wetland".equals(sample.vegetationProfile());
                    if (!trueWetland) {
                        unsupportedSwamps++;
                    }
                }
            }
        }
        if (unsupportedSwamps > 0) {
            throw new AssertionError(
                "riverbank biome scan found " + unsupportedSwamps +
                " swamp cells without wetland ecology"
            );
        }
        if (bankCells == 0) {
            throw new AssertionError(
                "riverbank biome scan found no dry riverbank cells"
            );
        }
        System.out.printf(
            "GeoWorld riverbank biome verification passed: " +
            "bankCells=%d swampCells=%d unsupported=0 regression=%s%n",
            bankCells,
            swampCells,
            regressionBiome
        );
    }

    private static boolean nearRiver(
        GeoWorldV1 runtime,
        int x,
        int z,
        int radius
    ) {
        for (int distance = 2; distance <= radius; distance += 2) {
            int diagonal = (int)Math.round(
                distance * 0.7071067811865476
            );
            int[][] offsets = {
                {distance, 0},
                {-distance, 0},
                {0, distance},
                {0, -distance},
                {diagonal, diagonal},
                {diagonal, -diagonal},
                {-diagonal, diagonal},
                {-diagonal, -diagonal}
            };
            for (int[] offset : offsets) {
                if (
                    "River".equals(
                        runtime.sample(
                            x + offset[0],
                            z + offset[1]
                        ).waterKind()
                    )
                ) {
                    return true;
                }
            }
        }
        return false;
    }

    private static boolean isSwampBiome(String biome) {
        return "minecraft:swamp".equals(biome) ||
            "minecraft:mangrove_swamp".equals(biome);
    }

    private static void verifyWetlandPools(
        GeoWorldV1 runtime
    ) {
        int[] center = findWetlandCenter(runtime);
        if (center == null) {
            throw new AssertionError(
                "could not find a dry Wetland region in seed " + SEED
            );
        }

        int centerChunkX = Math.floorDiv(center[0], 16);
        int centerChunkZ = Math.floorDiv(center[1], 16);
        int eligible = 0;
        int pools = 0;
        for (int dz = -4; dz <= 4; dz++) {
            for (int dx = -4; dx <= 4; dx++) {
                int baseX = (centerChunkX + dx) * 16;
                int baseZ = (centerChunkZ + dz) * 16;
                GeoWorldShoreline.ChunkPlan plan =
                    GeoWorldShoreline.planChunk(
                        runtime,
                        baseX,
                        baseZ
                    );
                for (int localZ = 0; localZ < 16; localZ++) {
                    for (int localX = 0; localX < 16; localX++) {
                        GeoWorldShoreline.Column column =
                            plan.column(localX, localZ);
                        if (
                            !isWetlandSemantics(column.sample()) ||
                            !"None".equals(
                                column.sample().waterKind()
                            )
                        ) {
                            continue;
                        }
                        eligible++;
                        if (!"WetlandPool".equals(
                            column.shoreWaterKind()
                        )) {
                            continue;
                        }
                        pools++;
                        if (
                            column.waterBoundary() -
                            column.groundBoundary() != 1
                        ) {
                            throw new AssertionError(
                                "wetland pool is not one block deep at " +
                                (baseX + localX) + "," +
                                (baseZ + localZ)
                            );
                        }
                        if (!"minecraft:mud".equals(
                            column.surfaceOverride()
                        )) {
                            throw new AssertionError(
                                "wetland pool bed is not mud at " +
                                (baseX + localX) + "," +
                                (baseZ + localZ)
                            );
                        }
                    }
                }
            }
        }

        if (eligible == 0 || pools == 0) {
            throw new AssertionError(
                "wetland planner produced no shallow pools: eligible=" +
                eligible +
                " pools=" +
                pools
            );
        }
        double ratio = (double)pools / eligible;
        if (ratio < 0.03 || ratio > 0.55) {
            throw new AssertionError(
                "wetland pool coverage is implausible: " +
                pools + "/" + eligible +
                " ratio=" + ratio
            );
        }

        System.out.printf(
            "GeoWorld wetland pool verification passed: center=%d,%d " +
            "pools=%d/%d ratio=%.3f%n",
            center[0],
            center[1],
            pools,
            eligible,
            ratio
        );
    }

    private static int[] findWetlandCenter(
        GeoWorldV1 runtime
    ) {
        GeoWorldV1.Geometry geometry = runtime.manifest().geometry();
        int macroStep = (int)Math.round(
            geometry.spanXBlocks() /
            Math.max(1.0, geometry.width() - 1.0)
        );
        int step = Math.max(16, macroStep / 4);
        int minX = (int)Math.round(geometry.originX()) + macroStep;
        int minZ = (int)Math.round(geometry.originZ()) + macroStep;
        int maxX = (int)Math.round(
            geometry.originX() + geometry.spanXBlocks()
        ) - macroStep;
        int maxZ = (int)Math.round(
            geometry.originZ() + geometry.spanZBlocks()
        ) - macroStep;

        for (int z = minZ; z <= maxZ; z += step) {
            for (int x = minX; x <= maxX; x += step) {
                GeoWorldV1.RuntimeSample sample =
                    runtime.sample(x, z);
                if (
                    isWetlandSemantics(sample) &&
                    "None".equals(sample.waterKind())
                ) {
                    return new int[]{x, z};
                }
            }
        }
        return null;
    }

    private static boolean samePlannedColumn(
        GeoWorldShoreline.Column a,
        GeoWorldShoreline.Column b
    ) {
        return a.originalGroundBoundary() == b.originalGroundBoundary() &&
            a.groundBoundary() == b.groundBoundary() &&
            a.waterBoundary() == b.waterBoundary() &&
            a.shoreDistance() == b.shoreDistance() &&
            java.util.Objects.equals(
                a.shoreWaterKind(),
                b.shoreWaterKind()
            ) &&
            a.shoreWaterBoundary() == b.shoreWaterBoundary() &&
            a.littoralShelf() == b.littoralShelf() &&
            java.util.Objects.equals(
                a.surfaceOverride(),
                b.surfaceOverride()
            ) &&
            java.util.Objects.equals(
                a.underOverride(),
                b.underOverride()
            );
    }

    private static boolean isSmallBackshorePit(
        GeoWorldShoreline.Column center,
        GeoWorldShoreline.Column west,
        GeoWorldShoreline.Column east,
        GeoWorldShoreline.Column north,
        GeoWorldShoreline.Column south
    ) {
        if (
            !"None".equals(center.sample().waterKind()) ||
            center.shoreDistance() < 2 ||
            center.shoreDistance() > 4
        ) {
            return false;
        }
        int higherDry = 0;
        for (GeoWorldShoreline.Column neighbor :
            new GeoWorldShoreline.Column[]{west, east, north, south}) {
            if (
                "None".equals(neighbor.sample().waterKind()) &&
                center.shoreWaterKind().equals(
                    neighbor.shoreWaterKind()
                ) &&
                neighbor.groundBoundary() >
                center.groundBoundary()
            ) {
                higherDry++;
            }
        }
        return higherDry >= 3;
    }

    private static int oceanBackshoreViolations(
        GeoWorldShoreline.Column center,
        GeoWorldShoreline.Column west,
        GeoWorldShoreline.Column east,
        GeoWorldShoreline.Column north,
        GeoWorldShoreline.Column south
    ) {
        int violations = 0;
        for (GeoWorldShoreline.Column inland :
            new GeoWorldShoreline.Column[]{west, east, north, south}) {
            if (
                !"None".equals(inland.sample().waterKind()) ||
                !"Ocean".equals(inland.shoreWaterKind()) ||
                inland.shoreDistance() != center.shoreDistance() + 1
            ) {
                continue;
            }
            if (
                inland.groundBoundary() <
                center.groundBoundary() - 1
            ) {
                violations++;
            }
        }
        return violations;
    }

    private static boolean isBackshoreGroove(
        GeoWorldShoreline.Column center,
        GeoWorldShoreline.Column a,
        GeoWorldShoreline.Column b
    ) {
        if (
            !"None".equals(a.sample().waterKind()) ||
            !"None".equals(b.sample().waterKind()) ||
            !center.shoreWaterKind().equals(a.shoreWaterKind()) ||
            !center.shoreWaterKind().equals(b.shoreWaterKind())
        ) {
            return false;
        }
        int distance = center.shoreDistance();
        boolean bracketsShore =
            (a.shoreDistance() == distance - 1 &&
             b.shoreDistance() == distance + 1) ||
            (b.shoreDistance() == distance - 1 &&
             a.shoreDistance() == distance + 1);
        return bracketsShore &&
            a.groundBoundary() > center.groundBoundary() &&
            b.groundBoundary() > center.groundBoundary();
    }

    private static boolean isWetlandSemantics(
        GeoWorldV1.RuntimeSample sample
    ) {
        return "Wetland".equals(sample.biomeFamily()) ||
            "Wet Mud".equals(sample.surfaceProfile()) ||
            "Peat".equals(sample.surfaceProfile());
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
