package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Set;

public final class GeoWorldV26ShorelineVerifier {
    private static final long SEED = 123456L;
    private static final int[][] REGRESSION_CENTERS = {
        {5594, -1216},
        {9769, -474},
        {7245, -4708},
        {2023, -2985},
        {2100, -3010},
        {10883, -1293},
        {10325, 2673},
        {10824, 2515},
        // User-reported island coast/high-rim regression.
        {-4096, -11072},
        // User-reported long 1x1 riverbank zipper regression.
        {1590, -3096},
        // User-reported high-water riverbank containment regression.
        {1637, -3015},
        // User-reported repaired-bank material mismatch regression.
        {1535, -1245}
    };
    private static final Set<String> SUGAR_CANE_SUBSTRATES = Set.of(
        "minecraft:grass_block",
        "minecraft:dirt",
        "minecraft:coarse_dirt",
        "minecraft:podzol",
        "minecraft:mud",
        "minecraft:sand"
    );

    private GeoWorldV26ShorelineVerifier() {}

    private static MinecraftVersionProfile PROFILE;
    private static HeightMapping HEIGHTS;
    private static GeoWorldShorelinePlanner SHORELINE;
    private static GeoWorldV26BiomeRealizer BIOMES;

    public static void run(
        MinecraftVersionProfile profile,
        String[] args
    ) throws Exception {
        PROFILE = java.util.Objects.requireNonNull(profile, "profile");
        HEIGHTS = PROFILE.heightMapping();
        SHORELINE = new GeoWorldShorelinePlanner(HEIGHTS);
        BIOMES = new GeoWorldV26BiomeRealizer(PROFILE);
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
        verifyLowlandOceanBackshore();
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
                    GeoWorldShorelinePlanner.ChunkPlan plan =
                        SHORELINE.planChunkModel(runtime, baseX, baseZ);

                    for (int checkZ : new int[]{2, 8, 13}) {
                        for (int checkX : new int[]{2, 8, 13}) {
                            GeoWorldShorelinePlanner.Column chunkColumn =
                                plan.column(checkX, checkZ);
                            GeoWorldShorelinePlanner.Column pointColumn =
                                SHORELINE.planModel(
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
                            GeoWorldShorelinePlanner.Column centerColumn =
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

                    HeightMapping heights = HEIGHTS;
                    for (int localZ = 0; localZ < 16; localZ++) {
                        for (int localX = 0; localX < 16; localX++) {
                            GeoWorldShorelinePlanner.Column riverColumn =
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
                            GeoWorldShorelinePlanner.Column a =
                                plan.column(localX, localZ);
                            if (
                                "None".equals(a.sample().waterKind()) &&
                                a.shoreDistance() == 1 &&
                                a.surfaceOverride() != null
                            ) {
                                if (localX < 15) {
                                    GeoWorldShorelinePlanner.Column b =
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
                                    GeoWorldShorelinePlanner.Column b =
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
                            GeoWorldShorelinePlanner.Column column =
                                plan.column(localX, localZ);
                            if ("None".equals(column.sample().waterKind()) &&
                                ("Lake".equals(column.shoreWaterKind()) ||
                                 "Ocean".equals(column.shoreWaterKind())) &&
                                column.shoreDistance() <=
                                    GeoWorldShorelinePlanner.MAX_GENTLE_BANK_DISTANCE &&
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
                                : PROFILE.surfaceProfileMapping().surfaceBlock(
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

        int reportedRiverThirdRowShelves =
            countReportedRiverThirdRowShelves(
                runtime,
                1590,
                -3096,
                16
            );

        int localRiverMaterialGaps =
            countImmediateRiverMaterialGaps(
                runtime,
                10824,
                2515,
                16
            );

        System.out.printf(
            "GeoWorld shoreline scan: seed=%d, immediate=%d, stable=%d, " +
            "shallowFringes=%d, shelves=%d, materials=%d, caneReady=%d/%d, " +
            "maxLowering=%d, gentleBanks=%d, inlandGentleBanks=%d, " +
            "maxRaising=%d at %d,%d %s (%d->%d), " +
            "raiseBuckets=[1:%d,2-3:%d,4+:%d], materialChanges=%d/%d, " +
            "narrowBackshoreGrooves=%d, smallBackshorePits=%d, " +
            "reportedRiverThirdRowShelves=%d, " +
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
            reportedRiverThirdRowShelves,
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
        if (reportedRiverThirdRowShelves != 0) {
            throw new AssertionError(
                "reported riverbank still lowers a third dry shelf row: " +
                reportedRiverThirdRowShelves
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
        String generationRulesVersion =
            runtime.manifest().generationRulesVersion();
        int maxAllowedRiverRaise =
            (
                "geoworld-rules-v1.1.6".equals(generationRulesVersion) ||
                "geoworld-rules-v1.1.7".equals(generationRulesVersion) ||
                "geoworld-rules-v1.1.8".equals(generationRulesVersion) ||
                "geoworld-rules-v1.1.9".equals(generationRulesVersion)
            ) ? 4 : 2;
        if (
            maxRaiseByKind.getOrDefault("River", 0) >
                maxAllowedRiverRaise ||
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
        verifyReportedIslandCoast(runtime);
        verifyReportedRiverbankRaster(runtime);
        verifyReportedRepairMaterialBlend(runtime);

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

    private static void verifyReportedIslandCoast(
        GeoWorldV1 runtime
    ) {
        final int centerX = -4096;
        final int centerZ = -11072;
        final int radius = 1536;
        final int sampleStep = 16;
        final int coastProbe = 16;
        final int interiorProbe = 64;

        ArrayList<Integer> coastHeights = new ArrayList<>();
        ArrayList<Integer> interiorHeights = new ArrayList<>();

        for (int z = centerZ - radius; z <= centerZ + radius; z += sampleStep) {
            for (int x = centerX - radius; x <= centerX + radius; x += sampleStep) {
                GeoWorldV1.RuntimeSample sample = runtime.sample(x, z);
                if (
                    !sample.sourceLand() ||
                    !"None".equals(sample.waterKind())
                ) {
                    continue;
                }

                int groundY = HEIGHTS.boundaryY(sample.groundHeight());
                if (hasWaterNeighbor(runtime, x, z, coastProbe, "Ocean")) {
                    coastHeights.add(groundY);
                } else if (
                    !hasWaterNeighbor(runtime, x, z, interiorProbe, null)
                ) {
                    interiorHeights.add(groundY);
                }
            }
        }

        if (coastHeights.size() < 100 || interiorHeights.size() < 1000) {
            throw new AssertionError(
                "reported island coast regression window lost coverage: coast=" +
                coastHeights.size() + " interior=" + interiorHeights.size()
            );
        }

        coastHeights.sort(Integer::compareTo);
        interiorHeights.sort(Integer::compareTo);

        int coastP10 = percentileSorted(coastHeights, 0.10);
        int coastP50 = percentileSorted(coastHeights, 0.50);
        int coastP90 = percentileSorted(coastHeights, 0.90);
        int interiorP50 = percentileSorted(interiorHeights, 0.50);
        int maxNormalCoastP90 = HEIGHTS.seaLevel() + 9;

        if (coastP90 > maxNormalCoastP90) {
            throw new AssertionError(
                "reported island coast regressed to a high rim: coastP90=" +
                coastP90 + " allowed=" + maxNormalCoastP90
            );
        }
        if (interiorP50 - coastP50 < 12) {
            throw new AssertionError(
                "reported island coast no longer rises naturally inland: " +
                "coastP50=" + coastP50 +
                " interiorP50=" + interiorP50
            );
        }

        System.out.printf(
            "GeoWorld reported island coast PASS: center=%d,%d, " +
            "coast=%d p10/p50/p90=%d/%d/%d, interior=%d p50=%d%n",
            centerX,
            centerZ,
            coastHeights.size(),
            coastP10,
            coastP50,
            coastP90,
            interiorHeights.size(),
            interiorP50
        );
    }

    private static boolean hasWaterNeighbor(
        GeoWorldV1 runtime,
        int x,
        int z,
        int distance,
        String requiredKind
    ) {
        int[][] directions = {
            {-1, 0}, {1, 0}, {0, -1}, {0, 1},
            {-1, -1}, {-1, 1}, {1, -1}, {1, 1}
        };
        for (int[] direction : directions) {
            GeoWorldV1.RuntimeSample neighbor = runtime.sample(
                x + direction[0] * distance,
                z + direction[1] * distance
            );
            if (
                !"None".equals(neighbor.waterKind()) &&
                (
                    requiredKind == null ||
                    requiredKind.equals(neighbor.waterKind())
                )
            ) {
                return true;
            }
        }
        return false;
    }

    private static int percentileSorted(
        ArrayList<Integer> values,
        double percentile
    ) {
        int index = (int)Math.round(
            percentile * (values.size() - 1)
        );
        return values.get(
            Math.max(0, Math.min(values.size() - 1, index))
        );
    }

    private static void verifyReportedRiverbankRaster(
        GeoWorldV1 runtime
    ) {
        // Search beyond the visible regression crop before declaring the
        // first River cell to be the bank edge. The reported screenshot crop
        // starts near x=1540, but some rows already contain water at x=1539.
        final int minX = 1500;
        final int maxX = 1760;
        final int minZ = -3094;
        final int maxZ = -3064;

        int previousRight = Integer.MIN_VALUE;
        int repeatedRows = 0;
        int currentUnitRun = 0;
        int longestUnitRun = 0;
        int maxEdgeStep = 0;
        int rows = 0;

        for (int z = minZ; z <= maxZ; z++) {
            int right = Integer.MIN_VALUE;
            for (int x = minX; x <= maxX; x++) {
                GeoWorldV1.RuntimeSample sample = runtime.sample(x, z);
                if ("River".equals(sample.waterKind())) {
                    right = x;
                }
            }
            if (right == Integer.MIN_VALUE) {
                throw new AssertionError(
                    "reported riverbank regression lost river row at z=" + z
                );
            }

            if (previousRight != Integer.MIN_VALUE) {
                int delta = right - previousRight;
                if (delta < 0 || delta > 2) {
                    throw new AssertionError(
                        "reported riverbank edge jumped at z=" + z +
                        ": previous=" + previousRight +
                        " now=" + right
                    );
                }
                maxEdgeStep = Math.max(maxEdgeStep, delta);
                if (delta == 0) {
                    repeatedRows++;
                    currentUnitRun = 0;
                } else if (delta == 1) {
                    currentUnitRun++;
                    longestUnitRun = Math.max(
                        longestUnitRun,
                        currentUnitRun
                    );
                } else {
                    currentUnitRun = 0;
                }
            }
            previousRight = right;
            rows++;

            for (int offset = 1; offset <= 2; offset++) {
                GeoWorldShorelinePlanner.Column bank =
                    SHORELINE.planModel(runtime, right + offset, z);
                if (
                    !"None".equals(bank.sample().waterKind()) ||
                    !"River".equals(bank.shoreWaterKind()) ||
                    bank.shoreDistance() > 2 ||
                    bank.surfaceOverride() == null
                ) {
                    throw new AssertionError(
                        "reported riverbank lost continuous two-block material " +
                        "band at " + (right + offset) + "," + z +
                        ": water=" + bank.sample().waterKind() +
                        " shore=" + bank.shoreWaterKind() +
                        " distance=" + bank.shoreDistance() +
                        " material=" + bank.surfaceOverride()
                    );
                }
            }
        }

        if (repeatedRows < 4) {
            throw new AssertionError(
                "reported riverbank regressed to a 1x1 zipper: repeatedRows=" +
                repeatedRows + "/" + rows
            );
        }
        if (longestUnitRun > 5) {
            throw new AssertionError(
                "reported riverbank has too long a unit staircase: run=" +
                longestUnitRun
            );
        }
        System.out.printf(
            "GeoWorld reported riverbank raster PASS: rows=%d, " +
            "repeatedRows=%d, longestUnitRun=%d, maxEdgeStep=%d%n",
            rows,
            repeatedRows,
            longestUnitRun,
            maxEdgeStep
        );
    }

    private static void verifyReportedRepairMaterialBlend(
        GeoWorldV1 runtime
    ) {
        final int radius = 24;
        final int[][] directions = {
            {-1, 0},
            {1, 0},
            {0, -1},
            {0, 1}
        };
        int checked = 0;

        for (int[] regressionCenter : REGRESSION_CENTERS) {
            int centerX = regressionCenter[0];
            int centerZ = regressionCenter[1];
            int minX = centerX - radius;
            int maxX = centerX + radius;
            int minZ = centerZ - radius;
            int maxZ = centerZ + radius;

            Map<Long, GeoWorldShorelinePlanner.Column> columns =
                new LinkedHashMap<>();
            int minChunkX = Math.floorDiv(minX - 1, 16);
            int maxChunkX = Math.floorDiv(maxX + 1, 16);
            int minChunkZ = Math.floorDiv(minZ - 1, 16);
            int maxChunkZ = Math.floorDiv(maxZ + 1, 16);
            for (int chunkZ = minChunkZ; chunkZ <= maxChunkZ; chunkZ++) {
                for (int chunkX = minChunkX; chunkX <= maxChunkX; chunkX++) {
                    int baseX = chunkX * 16;
                    int baseZ = chunkZ * 16;
                    GeoWorldShorelinePlanner.ChunkPlan plan =
                        SHORELINE.planChunkModel(runtime, baseX, baseZ);
                    for (int localZ = 0; localZ < 16; localZ++) {
                        for (int localX = 0; localX < 16; localX++) {
                            int worldX = baseX + localX;
                            int worldZ = baseZ + localZ;
                            long key =
                                (((long)worldX) << 32) ^
                                (worldZ & 0xffffffffL);
                            columns.put(
                                key,
                                plan.column(localX, localZ)
                            );
                        }
                    }
                }
            }

            for (int z = minZ; z <= maxZ; z++) {
                for (int x = minX; x <= maxX; x++) {
                    GeoWorldShorelinePlanner.Column column =
                        columns.get(
                            (((long)x) << 32) ^ (z & 0xffffffffL)
                        );
                    if (
                        column == null ||
                        !"None".equals(column.sample().waterKind()) ||
                        !"River".equals(column.shoreWaterKind()) ||
                        column.shoreDistance() < 2 ||
                        column.shoreDistance() > 3
                    ) {
                        continue;
                    }

                    boolean hasShorewardMaterial = false;
                    for (int[] direction : directions) {
                        int nx = x + direction[0];
                        int nz = z + direction[1];
                        GeoWorldShorelinePlanner.Column neighbor =
                            columns.get(
                                (((long)nx) << 32) ^
                                (nz & 0xffffffffL)
                            );
                        if (
                            neighbor != null &&
                            "None".equals(neighbor.sample().waterKind()) &&
                            "River".equals(neighbor.shoreWaterKind()) &&
                            neighbor.shoreDistance() > 0 &&
                            neighbor.shoreDistance() <
                                column.shoreDistance() &&
                            neighbor.surfaceOverride() != null
                        ) {
                            hasShorewardMaterial = true;
                            break;
                        }
                    }
                    if (!hasShorewardMaterial) {
                        continue;
                    }

                    checked++;
                    if (column.surfaceOverride() == null) {
                        throw new AssertionError(
                            "riverbank material gap at " +
                            x + "," + z +
                            ": distance=" + column.shoreDistance() +
                            " ground=" + column.groundBoundary() +
                            " original=" +
                            column.originalGroundBoundary()
                        );
                    }
                }
            }
        }

        if (checked == 0) {
            throw new AssertionError(
                "riverbank material-gap verifier did not cover any " +
                "distance-2/3 river shore columns"
            );
        }
        System.out.printf(
            "GeoWorld riverbank material-gap PASS: checked=%d%n",
            checked
        );
    }

    private static GeoWorldShorelinePlanner.Column cachedPlan(
        GeoWorldV1 runtime,
        Map<Long, GeoWorldShorelinePlanner.Column> cache,
        int x,
        int z
    ) {
        long key = (((long)x) << 32) ^ (z & 0xffffffffL);
        GeoWorldShorelinePlanner.Column existing = cache.get(key);
        if (existing != null) {
            return existing;
        }
        GeoWorldShorelinePlanner.Column planned =
            SHORELINE.planModel(runtime, x, z);
        cache.put(key, planned);
        return planned;
    }

    private static void verifyLowlandOceanBackshore() {
        // Reported real shoreline: dry original Y60, sea surface Y63.
        // The inland side must ramp back down, not form a 3-block-high
        // narrow berm. No increase in water height or dry-side flooding.
        int sea = 63;
        for (int depression = 1; depression <= 4; depression++) {
            int original = sea - depression;
            int previous = sea;
            for (int distance = 1; distance <=
                    GeoWorldShorelinePlanner.MAX_GENTLE_BANK_DISTANCE + 1;
                    distance++) {
                int minimum = GeoWorldShorelinePlanner.oceanBackshoreMinimum(
                    distance, sea, original
                );
                int ground = Math.max(original, minimum);
                if (ground > previous || previous - ground > 1 ||
                    ground < original) {
                    throw new AssertionError(
                        "coastal lowland shoreline berm failed: depth=" +
                        depression + " d=" + distance +
                        " previous=" + previous + " now=" + ground
                    );
                }
                if (distance == 1 && ground < sea) {
                    throw new AssertionError(
                        "ocean rim lower than the water boundary"
                    );
                }
                previous = ground;
            }
            if (previous != original) {
                throw new AssertionError(
                    "backshore taper failed to meet the original lowland"
                );
            }
        }
        System.out.println("GeoWorld coastal lowland berm taper PASS");
    }

    private static void verifyGentleBankProfile() {
        for (int relief : new int[]{8, 16, 24, 32}) {
            int water = 63;
            int previous = water;
            for (int distance = 1;
                 distance <= GeoWorldShorelinePlanner.MAX_GENTLE_BANK_DISTANCE;
                 distance++) {
                int capped = GeoWorldShorelinePlanner.gentleBankCap(
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
        String regressionBiome = BIOMES.biomeId(
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
                    String biome = BIOMES.biomeId(
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
        int poolRims = 0;
        for (int dz = -4; dz <= 4; dz++) {
            for (int dx = -4; dx <= 4; dx++) {
                int baseX = (centerChunkX + dx) * 16;
                int baseZ = (centerChunkZ + dz) * 16;
                GeoWorldShorelinePlanner.ChunkPlan plan =
                    SHORELINE.planChunkModel(
                        runtime,
                        baseX,
                        baseZ
                    );
                for (int localZ = 0; localZ < 16; localZ++) {
                    for (int localX = 0; localX < 16; localX++) {
                        GeoWorldShorelinePlanner.Column column =
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
                        int depth =
                            column.waterBoundary() -
                            column.groundBoundary();
                        if (depth == 1) {
                            pools++;
                            if (!"minecraft:mud".equals(
                                column.surfaceOverride()
                            )) {
                                throw new AssertionError(
                                    "wetland pool bed is not mud at " +
                                    (baseX + localX) + "," +
                                    (baseZ + localZ)
                                );
                            }
                            continue;
                        }
                        if (depth == 0 && column.shoreDistance() == 1) {
                            poolRims++;
                            if (!"minecraft:mud".equals(
                                column.surfaceOverride()
                            )) {
                                throw new AssertionError(
                                    "wetland pool rim is not mud at " +
                                    (baseX + localX) + "," +
                                    (baseZ + localZ)
                                );
                            }
                            continue;
                        }
                        throw new AssertionError(
                            "wetland pool/rim has invalid depth at " +
                            (baseX + localX) + "," +
                            (baseZ + localZ) +
                            ": depth=" + depth +
                            " shoreDistance=" + column.shoreDistance()
                        );
                    }
                }
            }
        }

        if (eligible == 0 || pools == 0) {
            throw new AssertionError(
                "wetland planner produced no shallow pools: eligible=" +
                eligible +
                " pools=" +
                pools +
                " rims=" +
                poolRims
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
            "pools=%d rims=%d eligible=%d ratio=%.3f%n",
            center[0],
            center[1],
            pools,
            poolRims,
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
        GeoWorldShorelinePlanner.Column a,
        GeoWorldShorelinePlanner.Column b
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
        GeoWorldShorelinePlanner.Column center,
        GeoWorldShorelinePlanner.Column west,
        GeoWorldShorelinePlanner.Column east,
        GeoWorldShorelinePlanner.Column north,
        GeoWorldShorelinePlanner.Column south
    ) {
        if (
            !"None".equals(center.sample().waterKind()) ||
            center.shoreDistance() < 2 ||
            center.shoreDistance() > 4
        ) {
            return false;
        }
        int higherDry = 0;
        for (GeoWorldShorelinePlanner.Column neighbor :
            new GeoWorldShorelinePlanner.Column[]{west, east, north, south}) {
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
        GeoWorldShorelinePlanner.Column center,
        GeoWorldShorelinePlanner.Column west,
        GeoWorldShorelinePlanner.Column east,
        GeoWorldShorelinePlanner.Column north,
        GeoWorldShorelinePlanner.Column south
    ) {
        int violations = 0;
        for (GeoWorldShorelinePlanner.Column inland :
            new GeoWorldShorelinePlanner.Column[]{west, east, north, south}) {
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
        GeoWorldShorelinePlanner.Column center,
        GeoWorldShorelinePlanner.Column a,
        GeoWorldShorelinePlanner.Column b
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

    private static int countImmediateRiverMaterialGaps(
        GeoWorldV1 runtime,
        int centerX,
        int centerZ,
        int radius
    ) {
        int minX = centerX - radius;
        int maxX = centerX + radius;
        int minZ = centerZ - radius;
        int maxZ = centerZ + radius;
        int minChunkX = Math.floorDiv(minX, 16);
        int maxChunkX = Math.floorDiv(maxX, 16);
        int minChunkZ = Math.floorDiv(minZ, 16);
        int maxChunkZ = Math.floorDiv(maxZ, 16);
        int count = 0;
        for (int chunkZ = minChunkZ; chunkZ <= maxChunkZ; chunkZ++) {
            for (int chunkX = minChunkX; chunkX <= maxChunkX; chunkX++) {
                int baseX = chunkX * 16;
                int baseZ = chunkZ * 16;
                GeoWorldShorelinePlanner.ChunkPlan plan =
                    SHORELINE.planChunkModel(runtime, baseX, baseZ);
                for (int localZ = 0; localZ < 16; localZ++) {
                    int worldZ = baseZ + localZ;
                    if (worldZ < minZ || worldZ > maxZ) continue;
                    for (int localX = 0; localX < 16; localX++) {
                        int worldX = baseX + localX;
                        if (worldX < minX || worldX > maxX) continue;
                        GeoWorldShorelinePlanner.Column column =
                            plan.column(localX, localZ);
                        if (
                            "None".equals(column.sample().waterKind()) &&
                            "River".equals(column.shoreWaterKind()) &&
                            column.shoreDistance() == 1 &&
                            column.surfaceOverride() == null
                        ) {
                            count++;
                        }
                    }
                }
            }
        }
        return count;
    }

    private static int countReportedRiverThirdRowShelves(
        GeoWorldV1 runtime,
        int centerX,
        int centerZ,
        int radius
    ) {
        int minX = centerX - radius;
        int maxX = centerX + radius;
        int minZ = centerZ - radius;
        int maxZ = centerZ + radius;
        int minChunkX = Math.floorDiv(minX, 16);
        int maxChunkX = Math.floorDiv(maxX, 16);
        int minChunkZ = Math.floorDiv(minZ, 16);
        int maxChunkZ = Math.floorDiv(maxZ, 16);
        int count = 0;
        for (int chunkZ = minChunkZ; chunkZ <= maxChunkZ; chunkZ++) {
            for (int chunkX = minChunkX; chunkX <= maxChunkX; chunkX++) {
                int baseX = chunkX * 16;
                int baseZ = chunkZ * 16;
                GeoWorldShorelinePlanner.ChunkPlan plan =
                    SHORELINE.planChunkModel(runtime, baseX, baseZ);
                for (int localZ = 0; localZ < 16; localZ++) {
                    int worldZ = baseZ + localZ;
                    if (worldZ < minZ || worldZ > maxZ) continue;
                    for (int localX = 0; localX < 16; localX++) {
                        int worldX = baseX + localX;
                        if (worldX < minX || worldX > maxX) continue;
                        GeoWorldShorelinePlanner.Column column =
                            plan.column(localX, localZ);
                        if (
                            "None".equals(column.sample().waterKind()) &&
                            "River".equals(column.shoreWaterKind()) &&
                            column.shoreDistance() >= 3 &&
                            column.littoralShelf()
                        ) {
                            count++;
                        }
                    }
                }
            }
        }
        return count;
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
