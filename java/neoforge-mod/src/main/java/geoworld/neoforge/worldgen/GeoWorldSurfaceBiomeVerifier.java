package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3BiomeCoverage;
import geoworld.minecraft.v26_3common.Minecraft26_3BiomeRealizer;
import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;

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
public final class GeoWorldSurfaceBiomeVerifier {
    private GeoWorldSurfaceBiomeVerifier() {}

    public static void main(String[] args) throws Exception {
        long seed = args.length == 0 ? 123456L : Long.parseLong(args[0]);
        var runtime = GeoWorldEngineBridge.generate(
            seed, GeoWorldCreationConfig.defaults()
        ).model();

        Set<String> required = Minecraft26_3BiomeRealizer.requiredBiomeIds();
        if (required.size() != 56) {
            throw new AssertionError("Unexpected world biome catalog size " + required.size());
        }
        Map<GeoWorldSurfaceMaterials.Cover, Integer> types =
            new EnumMap<>(GeoWorldSurfaceMaterials.Cover.class);
        for (String biome : required) {
            var type = GeoWorldSurfaceMaterials.cover(biome);
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

        if (!GeoWorldSurfaceMaterials.top(
            "minecraft:desert", "Alluvial Silt", "minecraft:sand"
        ).equals("minecraft:sand") ||
            !GeoWorldSurfaceMaterials.top(
                "minecraft:badlands", "Shrub Soil", "minecraft:mud"
            ).equals("minecraft:mud")) {
            throw new AssertionError("hydrology-owned shoreline materials were lost");
        }
        if (!"minecraft:sandstone".equals(GeoWorldSurfaceMaterials.below(
            "minecraft:desert", "Alluvial Silt", null, 382, 154, -3089, 2
        ))) {
            throw new AssertionError("desert subsoil is not sandstone");
        }
        String terracotta = GeoWorldSurfaceMaterials.below(
            "minecraft:badlands", "Desert Gravel", null, 350, 108, -3090, 1
        );
        if (!"minecraft:terracotta".equals(terracotta) &&
            !terracotta.endsWith("_terracotta")) {
            throw new AssertionError("badlands missing terracotta strata: " + terracotta);
        }

        var coverage = Minecraft26_3BiomeCoverage.forModel(runtime);
        HashSet<String> checked = new HashSet<>();
        int surfacePatches = 0, cavePatches = 0, fringeOverrides = 0;
        int columns = 0, materials = 0;
        for (var habitat : coverage.habitats()) {
            String declared = habitat.biome();
            int x = (int)Math.round(habitat.x());
            int z = (int)Math.round(habitat.z());
            String atCenter = Minecraft26_3BiomeRealizer.biomeId(
                runtime, x, habitat.y(), z
            );
            if (!declared.equals(atCenter)) {
                throw new AssertionError("habitat biome missing: " + habitat + " got " + atCenter);
            }
            checked.add(declared);
            if (GeoWorldSurfaceMaterials.cover(declared) ==
                GeoWorldSurfaceMaterials.Cover.CAVE) {
                cavePatches++;
                continue;
            }
            surfacePatches++;
            for (int dx : new int[]{-32, 0, 32}) {
                for (int dz : new int[]{-32, 0, 32}) {
                    int px = x + dx, pz = z + dz;
                    var column = GeoWorldShoreline.plan(runtime, px, pz);
                    int groundY = column.groundBoundary() - 1;
                    String finalBiome = Minecraft26_3BiomeRealizer.biomeId(
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
                    String top = GeoWorldSurfaceMaterials.top(
                        finalBiome, profile, column.surfaceOverride()
                    );
                    String expected = GeoWorldSurfaceMaterials.top(
                        declared, profile, null
                    );
                    if (column.surfaceOverride() == null && !top.equals(expected)) {
                        throw new AssertionError(
                            "surface mismatch " + finalBiome + " " + px + "," + pz +
                            " got=" + top + " expected=" + expected + " profile=" + profile
                        );
                    }
                    if (column.surfaceOverride() != null) fringeOverrides++;
                    String underneath = GeoWorldSurfaceMaterials.below(
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
        if (!checked.equals(required)) {
            var missing = new HashSet<>(required);
            missing.removeAll(checked);
            throw new AssertionError("unverified biome materials: " + missing);
        }
        if (surfacePatches != 52 || cavePatches != 4) {
            throw new AssertionError(
                "unexpected surface/cave distribution " + surfacePatches + "/" + cavePatches
            );
        }
        // Check the ecological biome itself. A previously unrestrained
        // windswept-gravelly-hills candidate could replace GeoWorld deserts
        // even far from transition zones; fixing material alone is inadequate.
        var geometry = runtime.manifest().geometry();
        var mapper = new Minecraft26_3HeightMapping();
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
                String chosen = Minecraft26_3BiomeRealizer.biomeId(runtime, x, y, z);
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
            var column = GeoWorldShoreline.plan(runtime, x, z);
            String id = Minecraft26_3BiomeRealizer.biomeId(
                runtime, x, column.groundBoundary() - 1, z
            );
            System.out.printf(
                "reported_coordinate x=%d z=%d biome=%s " +
                "modelFamily=%s profile=%s surface=%s under=%s%n",
                x, z, id, column.sample().biomeFamily(),
                column.sample().surfaceProfile(),
                GeoWorldSurfaceMaterials.top(
                    id, column.sample().surfaceProfile(), column.surfaceOverride()
                ),
                GeoWorldSurfaceMaterials.below(
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
        String actual = GeoWorldSurfaceMaterials.top(biome, profile, null);
        if (!expected.equals(actual)) {
            throw new AssertionError(
                "expected " + expected + " from " + biome +
                " with soil " + profile + ", got " + actual
            );
        }
    }
}
