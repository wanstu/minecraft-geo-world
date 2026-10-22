package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;
import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;

import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.Map;
import java.util.Set;

/** Reports actual final Runtime biome samples, not just registry membership. */
public final class GeoWorldV26BiomeCoverageVerifier {
    private GeoWorldV26BiomeCoverageVerifier() {}

    public static void run(
        MinecraftVersionProfile profile,
        String[] args
    ) throws Exception {
        GeoWorldV26BiomeRealizer realizer =
            new GeoWorldV26BiomeRealizer(profile);
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
                    String id = realizer.biomeId(model, x, y, z);
                    observed.computeIfAbsent(id, _key -> new int[]{0})[0]++;
                    samples++;
                }
            }
        }
        // Rivers are narrow hydrographic features. The coarse ecological
        // grid above can legitimately step over a frozen river even when the
        // final Runtime contains one. Sample actual River/Lake receivers at
        // the model cell scale as well, matching the resolution used by the
        // coverage planner for hydrographic candidates.
        final double riverStep = Math.max(96.0, g.cellSize());
        for (double x = g.originX() + riverStep / 2;
             x <= g.originX() + g.spanXBlocks() - riverStep / 2;
             x += riverStep) {
            for (double z = g.originZ() + riverStep / 2;
                 z <= g.originZ() + g.spanZBlocks() - riverStep / 2;
                 z += riverStep) {
                var sample = model.sample(x, z);
                if (!"River".equals(sample.waterKind()) &&
                    !"Lake".equals(sample.waterKind())) {
                    continue;
                }
                String id = realizer.biomeId(model, x, 319, z);
                observed.computeIfAbsent(id, _key -> new int[]{0})[0]++;
                samples++;
            }
        }

        Set<String> expected = realizer.requiredBiomeIds();
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
        var coverage = realizer.coverageForModel(model);
        if (coverage.habitats().size() != expected.size()) {
            throw new AssertionError("habitat reserve size differs from required catalog");
        }
        for (var habitat : coverage.habitats()) {
            String atCenter = realizer.biomeId(
                model, habitat.x(), habitat.y(), habitat.z()
            );
            String atNearby = realizer.biomeId(
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
