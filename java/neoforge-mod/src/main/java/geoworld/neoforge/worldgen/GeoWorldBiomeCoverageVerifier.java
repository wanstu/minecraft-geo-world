package geoworld.neoforge.worldgen;

import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;
import geoworld.minecraft.v26_3common.Minecraft26_3BiomeRealizer;
import geoworld.minecraft.v26_3common.Minecraft26_3BiomeCoverage;

import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.Map;
import java.util.Set;

/** Reports actual final Runtime biome samples, not just registry membership. */
public final class GeoWorldBiomeCoverageVerifier {
    private GeoWorldBiomeCoverageVerifier() {}

    public static void main(String[] args) throws Exception {
        final long seed = args.length == 0 ? 123456L : Long.parseLong(args[0]);
        GeoWorldV1 model = GeoWorldEngineBridge.generate(
            seed, GeoWorldCreationConfig.defaults()
        ).model();
        var g = model.manifest().geometry();
        Map<String, int[]> observed = new LinkedHashMap<>();
        int samples = 0;
        final int step = 320;
        for (double x = g.originX() + 160; x <= g.originX() + g.spanXBlocks() - 160; x += step) {
            for (double z = g.originZ() + 160; z <= g.originZ() + g.spanZBlocks() - 160; z += step) {
                for (int y : new int[]{-48, -24, 0, 24, 64, 120, 240, 319}) {
                    String id = Minecraft26_3BiomeRealizer.biomeId(model, x, y, z);
                    observed.computeIfAbsent(id, _key -> new int[]{0})[0]++;
                    samples++;
                }
            }
        }
        Set<String> expected = Minecraft26_3BiomeRealizer.requiredBiomeIds();
        Set<String> missing = new LinkedHashSet<>(expected);
        missing.removeAll(observed.keySet());
        System.out.printf("GeoWorld realized biome samples seed=%d samples=%d observed=%d/required=%d missing=%d%n", seed, samples, observed.size(), expected.size(), missing.size());
        System.out.println("Missing: " + missing);
        if (!missing.isEmpty()) {
            throw new AssertionError(
                "registered overworld biomes absent from sampled real-world generation: " +
                missing
            );
        }
        var coverage = Minecraft26_3BiomeCoverage.forModel(model);
        if (coverage.habitats().size() != expected.size()) {
            throw new AssertionError("habitat reserve size differs from required catalog");
        }
        for (var habitat : coverage.habitats()) {
            String atCenter = Minecraft26_3BiomeRealizer.biomeId(
                model, habitat.x(), habitat.y(), habitat.z()
            );
            String atNearby = Minecraft26_3BiomeRealizer.biomeId(
                model, habitat.x() + 32, habitat.y(), habitat.z()
            );
            if (!habitat.biome().equals(atCenter) ||
                !habitat.biome().equals(atNearby)) {
                throw new AssertionError(
                    "habitat was not continuous: " + habitat +
                    " center=" + atCenter + " nearby=" + atNearby
                );
            }
        }
        System.out.println(
            "All overworld biomes verified in actual source samples and habitat interiors."
        );
    }
}
