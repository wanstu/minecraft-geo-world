package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;

import java.util.EnumMap;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;

/**
 * Checks all registered Overworld biomes against final voxel material policy,
 * including reserved rare-biome habitats and mixed soil/biome regressions.
 * Registry membership alone is not considered an adequate test.
 */
public final class GeoWorldV26SurfaceBiomeVerifier {
    private GeoWorldV26SurfaceBiomeVerifier() {}

    private static MinecraftVersionProfile PROFILE;
    private static GeoWorldV26BiomeRealizer BIOMES;
    private static GeoWorldShorelinePlanner SHORELINE;
    private static HeightMapping HEIGHTS;

    public static void run(
        MinecraftVersionProfile versionProfile,
        String[] args
    ) throws Exception {
        PROFILE = java.util.Objects.requireNonNull(
            versionProfile,
            "versionProfile"
        );
        BIOMES = new GeoWorldV26BiomeRealizer(PROFILE);
        HEIGHTS = PROFILE.heightMapping();
        SHORELINE = new GeoWorldShorelinePlanner(HEIGHTS);
        long seed = args.length == 0 ? 123456L : Long.parseLong(args[0]);
        var runtime = GeoWorldEngineBridge.generate(
            seed, GeoWorldCreationConfig.defaults()
        ).model();

        Set<String> required = BIOMES.requiredBiomeIds();
        Set<String> catalog = PROFILE.biomeCatalog().candidateBiomeIds();
        if (!required.containsAll(catalog) || required.isEmpty()) {
            throw new AssertionError(
                "World biome requirements do not cover the version catalog: " +
                PROFILE.minecraftVersion()
            );
        }
        Map<GeoWorldV26SurfaceMaterials.Cover, Integer> types =
            new EnumMap<>(GeoWorldV26SurfaceMaterials.Cover.class);
        for (String biome : required) {
            var type = GeoWorldV26SurfaceMaterials.cover(biome);
            types.merge(type, 1, Integer::sum);
        }
        if (types.isEmpty()) throw new AssertionError("empty mappings");

        // User-reported mismatches and a representative from every special
        // surface group: final biome MUST win over intermediary soil metadata.
        assertTop("minecraft:desert", "Alluvial Silt", "minecraft:sand");
        assertTop("minecraft:desert", "Shrub Soil", "minecraft:sand");
        assertTop("minecraft:desert", "Dry Grass Soil", "minecraft:sand");
        assertTop("minecraft:badlands", "Shrub Soil", "minecraft:red_sand");
        assertTop("minecraft:eroded_badlands", "Alluvial Silt", "minecraft:terracotta");
        assertTop("minecraft:badlands", "Desert Gravel", "minecraft:terracotta");
        assertTop("minecraft:wooded_badlands", "Forest Loam", "minecraft:red_sand");
        assertTop("minecraft:plains", "Desert Sand", "minecraft:grass_block");
        assertTop("minecraft:forest", "Desert Sand", "minecraft:grass_block");
        assertTop("minecraft:mushroom_fields", "Shrub Soil", "minecraft:mycelium");
        assertTop("minecraft:pale_garden", "Shrub Soil", "minecraft:pale_moss_block");
        assertTop("minecraft:mangrove_swamp", "Alluvial Silt", "minecraft:mud");
        assertTop("minecraft:snowy_plains", "Shrub Soil", "minecraft:snow_block");
        assertTop("minecraft:snowy_beach", "Desert Sand", "minecraft:snow_block");
        assertTop("minecraft:beach", "Shrub Soil", "minecraft:sand");
        assertTop("minecraft:stony_shore", "Desert Sand", "minecraft:stone");
        assertTop("minecraft:old_growth_spruce_taiga", "Desert Sand", "minecraft:podzol");
        assertTop("minecraft:windswept_gravelly_hills", "Forest Loam", "minecraft:gravel");
        assertTop("minecraft:stony_peaks", "Alpine Snow", "minecraft:stone");
        assertTop("minecraft:jagged_peaks", "Alpine Snow", "minecraft:snow_block");
        assertTop("minecraft:swamp", "Wet Mud", "minecraft:mud");
        assertTop("minecraft:swamp", "Grass Loam", "minecraft:grass_block");

        // v1.1.7 fixes the final-material realization of slope-derived
        // Alpine Rock without changing older locked worlds.
        assertRulesTop(
            "minecraft:plains", "Alpine Rock",
            "geoworld-rules-v1.1.6", "minecraft:grass_block"
        );
        assertRulesTop(
            "minecraft:plains", "Alpine Rock",
            "geoworld-rules-v1.1.7", "minecraft:stone"
        );
        assertRulesTop(
            "minecraft:old_growth_spruce_taiga", "Alpine Rock",
            "geoworld-rules-v1.1.6", "minecraft:podzol"
        );
        assertRulesTop(
            "minecraft:old_growth_spruce_taiga", "Alpine Rock",
            "geoworld-rules-v1.1.7", "minecraft:stone"
        );
        if (!"minecraft:dirt".equals(
            GeoWorldV26SurfaceMaterials.belowForRules(
                "minecraft:plains", "Alpine Rock", null,
                128, 120, -256, 1, "geoworld-rules-v1.1.6"
            )
        )) {
            throw new AssertionError("v1.1.6 alpine-rock subsoil changed");
        }
        if (!"minecraft:stone".equals(
            GeoWorldV26SurfaceMaterials.belowForRules(
                "minecraft:plains", "Alpine Rock", null,
                128, 120, -256, 1, "geoworld-rules-v1.1.7"
            )
        )) {
            throw new AssertionError("v1.1.7 alpine-rock subsoil is not stone");
        }
        if (!"minecraft:gravel".equals(
            GeoWorldV26SurfaceMaterials.topForRules(
                "minecraft:plains", "Alpine Rock", "minecraft:gravel",
                "geoworld-rules-v1.1.7"
            )
        )) {
            throw new AssertionError("shoreline override lost to alpine rock");
        }

        if (!GeoWorldV26SurfaceMaterials.top(
            "minecraft:desert", "Alluvial Silt", "minecraft:sand"
        ).equals("minecraft:sand") ||
            !GeoWorldV26SurfaceMaterials.top(
                "minecraft:badlands", "Shrub Soil", "minecraft:mud"
            ).equals("minecraft:mud")) {
            throw new AssertionError("hydrology-owned shoreline materials were lost");
        }
        if (!"minecraft:sandstone".equals(GeoWorldV26SurfaceMaterials.below(
            "minecraft:desert", "Alluvial Silt", null, 382, 154, -3089, 2
        ))) {
            throw new AssertionError("desert subsoil is not sandstone");
        }
        String terracotta = GeoWorldV26SurfaceMaterials.below(
            "minecraft:badlands", "Desert Gravel", null, 350, 108, -3090, 1
        );
        if (!"minecraft:terracotta".equals(terracotta) &&
            !terracotta.endsWith("_terracotta")) {
            throw new AssertionError("badlands missing terracotta strata: " + terracotta);
        }

        var coverage = BIOMES.coverageForModel(runtime);
        HashSet<String> checked = new HashSet<>();
        int surfacePatches = 0, cavePatches = 0, fringeOverrides = 0;
        int columns = 0, materials = 0;
        for (var habitat : coverage.habitats()) {
            String declared = habitat.biome();
            int x = (int)Math.round(habitat.x());
            int z = (int)Math.round(habitat.z());
            String atCenter = BIOMES.biomeId(
                runtime, x, habitat.y(), z
            );
            if (!declared.equals(atCenter)) {
                throw new AssertionError("habitat biome missing: " + habitat + " got " + atCenter);
            }
            checked.add(declared);
            if (GeoWorldV26SurfaceMaterials.cover(declared) ==
                GeoWorldV26SurfaceMaterials.Cover.CAVE) {
                cavePatches++;
                continue;
            }
            surfacePatches++;
            for (int dx : new int[]{-32, 0, 32}) {
                for (int dz : new int[]{-32, 0, 32}) {
                    int px = x + dx, pz = z + dz;
                    var column = SHORELINE.planModel(runtime, px, pz);
                    int groundY = column.groundBoundary() - 1;
                    String finalBiome = BIOMES.biomeId(
                        runtime, px, groundY, pz
                    );
                    if (!declared.equals(finalBiome)) {
                        throw new AssertionError(
                            "habitat surface biome missing " + declared +
                            " at " + px + "," + groundY + "," + pz +
                            " got " + finalBiome
                        );
                    }
                    String profile = column.sample().surfaceProfile();
                    String top = GeoWorldV26SurfaceMaterials.top(
                        finalBiome, profile, column.surfaceOverride()
                    );
                    String expected = GeoWorldV26SurfaceMaterials.top(
                        declared, profile, null
                    );
                    if (column.surfaceOverride() == null && !top.equals(expected)) {
                        throw new AssertionError(
                            "surface mismatch " + finalBiome + " " + px + "," + pz +
                            " got=" + top + " expected=" + expected + " profile=" + profile
                        );
                    }
                    if (column.surfaceOverride() != null) fringeOverrides++;
                    String underneath = GeoWorldV26SurfaceMaterials.below(
                        finalBiome, profile, column.underOverride(),
                        px, groundY - 2, pz, 2
                    );
                    if (underneath == null) {
                        throw new AssertionError("missing subsoil " + finalBiome);
                    }
                    columns++;
                    materials += 2;
                }
            }
        }
        // v1.1.11 no longer fabricates Mushroom Fields or Pale Garden
        // through the generic biome-coverage reserve. Verify those biomes at
        // their persisted special-island centers instead.
        int specialSurfacePatches = 0;
        for (var landscape : runtime.specialLandscapes()) {
            String expectedBiome = switch (landscape.kind()) {
                case "mushroom_island" -> "minecraft:mushroom_fields";
                case "pale_garden_island" -> "minecraft:pale_garden";
                default -> null;
            };
            if (expectedBiome == null || checked.contains(expectedBiome)) {
                continue;
            }

            int x = (int)Math.round(landscape.centerX());
            int z = (int)Math.round(landscape.centerZ());
            var sample = runtime.sample(x, z);
            int groundY = HEIGHTS.boundaryY(sample.groundHeight()) - 1;
            String actualBiome = BIOMES.biomeId(runtime, x, groundY, z);
            if (!expectedBiome.equals(actualBiome)) {
                throw new AssertionError(
                    "special landscape biome missing " + expectedBiome +
                    " at " + x + "," + groundY + "," + z +
                    " got " + actualBiome +
                    " kind=" + landscape.kind()
                );
            }
            String expectedTop = GeoWorldV26SurfaceMaterials.top(
                expectedBiome,
                sample.surfaceProfile(),
                null
            );
            String actualTop = GeoWorldV26SurfaceMaterials.top(
                actualBiome,
                sample.surfaceProfile(),
                null
            );
            if (!expectedTop.equals(actualTop)) {
                throw new AssertionError(
                    "special landscape surface mismatch " + expectedBiome +
                    " at " + x + "," + z +
                    " got=" + actualTop +
                    " expected=" + expectedTop
                );
            }
            String underneath = GeoWorldV26SurfaceMaterials.below(
                actualBiome,
                sample.surfaceProfile(),
                null,
                x,
                groundY - 2,
                z,
                2
            );
            if (underneath == null) {
                throw new AssertionError(
                    "missing special landscape subsoil " + expectedBiome
                );
            }
            checked.add(expectedBiome);
            specialSurfacePatches++;
            columns++;
            materials += 2;
        }

        if (!checked.equals(required)) {
            var missing = new HashSet<>(required);
            missing.removeAll(checked);
            throw new AssertionError("unverified biome materials: " + missing);
        }
        int expectedCavePatches = types.getOrDefault(
            GeoWorldV26SurfaceMaterials.Cover.CAVE, 0
        );
        int expectedSurfacePatches = required.size() - expectedCavePatches;
        if (surfacePatches + specialSurfacePatches != expectedSurfacePatches ||
            cavePatches != expectedCavePatches) {
            throw new AssertionError(
                "unexpected surface/cave distribution " +
                (surfacePatches + specialSurfacePatches) + "/" + cavePatches +
                " (coverage=" + surfacePatches +
                ", special=" + specialSurfacePatches + ")" +
                " expected " + expectedSurfacePatches + "/" + expectedCavePatches
            );
        }
        // Check the ecological biome itself. A previously unrestrained
        // windswept-gravelly-hills candidate could replace GeoWorld deserts
        // even far from transition zones; fixing material alone is inadequate.
        var geometry = runtime.manifest().geometry();
        var mapper = HEIGHTS;
        int naturalDesertColumns = 0;
        int desertTransitionCells = 0;
        Map<String, Integer> desertConflicts = new HashMap<>();
        for (double x = geometry.originX() + 256;
             x < geometry.originX() + geometry.spanXBlocks() - 256;
             x += 512) {
            for (double z = geometry.originZ() + 256;
                 z < geometry.originZ() + geometry.spanZBlocks() - 256;
                 z += 512) {
                var original = runtime.sample(x, z);
                if (!original.sourceLand() ||
                    !"None".equals(original.waterKind()) ||
                    (!"Hot Desert".equals(original.biomeFamily()) &&
                     !"Temperate Desert".equals(original.biomeFamily()))) {
                    continue;
                }
                int y = mapper.boundaryY(original.groundHeight()) - 1;
                if (coverage.override(runtime, x, y, z) != null) continue;
                // An ecological transition intentionally blends the biome
                // with another model family. Only flag incompatible biomes
                // inside a stable, single-family interior.
                boolean atTransition = false;
                for (int dx : new int[]{-56, 0, 56}) {
                    for (int dz : new int[]{-56, 0, 56}) {
                        if (!original.biomeFamily().equals(
                            runtime.sample(x + dx, z + dz).biomeFamily()
                        )) atTransition = true;
                    }
                }
                if (atTransition) {
                    desertTransitionCells++;
                    continue;
                }
                naturalDesertColumns++;
                String chosen = BIOMES.biomeId(runtime, x, y, z);
                if (!Set.of(
                    "minecraft:desert", "minecraft:badlands",
                    "minecraft:eroded_badlands", "minecraft:wooded_badlands",
                    "minecraft:beach", "minecraft:stony_shore"
                ).contains(chosen)) {
                    desertConflicts.merge(chosen, 1, Integer::sum);
                }
            }
        }
        if (!desertConflicts.isEmpty()) {
            throw new AssertionError(
                "ecological desert biomes displaced by incompatible catalog " +
                desertConflicts + " of " + naturalDesertColumns
            );
        }
        // Inspect real player-reported coordinates, not just synthetic
        // family/profile test cases. Their biome and GeoWorld soil may differ.
        for (int[] position : new int[][]{
            {382, -3089}, {396, -3924}, {-159, -6808},
            {4779, -6910}, {-241, -6871}, {2013, 7289}
        }) {
            int x = position[0], z = position[1];
            var column = SHORELINE.planModel(runtime, x, z);
            String id = BIOMES.biomeId(
                runtime, x, column.groundBoundary() - 1, z
            );
            System.out.printf(
                "reported_coordinate x=%d z=%d biome=%s " +
                "modelFamily=%s profile=%s surface=%s under=%s%n",
                x, z, id, column.sample().biomeFamily(),
                column.sample().surfaceProfile(),
                GeoWorldV26SurfaceMaterials.top(
                    id, column.sample().surfaceProfile(), column.surfaceOverride()
                ),
                GeoWorldV26SurfaceMaterials.below(
                    id, column.sample().surfaceProfile(), column.underOverride(),
                    x, column.groundBoundary() - 2, z, 1
                )
            );
        }
        System.out.printf(
            "GeoWorld surface biome/material PASS seed=%d allBiomes=%d " +
            "surface=%d cave=%d columns=%d materialChecks=%d " +
            "hydrologyPriority=%d naturalDesert=%d desertTransitions=%d " +
            "desertConflicts=%d groupedMaterials=%s%n",
            seed, checked.size(), surfacePatches, cavePatches,
            columns, materials, fringeOverrides,
            naturalDesertColumns, desertTransitionCells,
            desertConflicts.size(), types
        );
    }

    private static void assertTop(String biome, String profile, String expected) {
        String actual = GeoWorldV26SurfaceMaterials.top(biome, profile, null);
        if (!expected.equals(actual)) {
            throw new AssertionError(
                "expected " + expected + " from " + biome +
                " with soil " + profile + ", got " + actual
            );
        }
    }

    private static void assertRulesTop(
        String biome,
        String profile,
        String rules,
        String expected
    ) {
        String actual = GeoWorldV26SurfaceMaterials.topForRules(
            biome, profile, null, rules
        );
        if (!expected.equals(actual)) {
            throw new AssertionError(
                "expected " + expected + " from " + biome +
                " with soil " + profile + " under " + rules +
                ", got " + actual
            );
        }
    }
}
