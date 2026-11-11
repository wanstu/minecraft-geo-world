package geoworld.minecraft.common.verification;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig.ContinentPreset;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.common.engine.GeoWorldMineralContext;

import java.util.List;

public final class GeoWorldEngineBridgeVerifier {
    private GeoWorldEngineBridgeVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldCreationConfig config = new GeoWorldCreationConfig(
            48,
            30,
            128,
            12,
            0.68,
            0.35,
            0.65,
            ContinentPreset.DEFAULT
        );
        long seed = 3873414348276034653L;

        GeoWorldMineralContext.publish(
            List.of("copper", "iron", "tin", "zinc"),
            List.of("copper", "iron", "zinc")
        );

        GeoWorldEngineBridge.GeneratedModel generated =
            GeoWorldEngineBridge.generate(seed, config);

        if (generated.model().manifest().seed() != seed) {
            throw new AssertionError("seed mismatch");
        }
        if (generated.model().mineralDistrictAssignments().isEmpty()) {
            throw new AssertionError("mineral assignments were not frozen into the World Model");
        }
        boolean resolvedMissingPrimary = false;
        for (var assignment : generated.model().mineralDistrictAssignments()) {
            if (!List.of("copper", "iron", "zinc").contains(assignment.primary()) ||
                !List.of("copper", "iron", "zinc").contains(assignment.companion())) {
                throw new AssertionError("assignment used a mineral absent from the generation-time world: " + assignment);
            }
            if (assignment.primary().equals(assignment.companion())) {
                throw new AssertionError("primary and companion must differ: " + assignment);
            }
            if (assignment.plannedPrimary().equals("tin") && !assignment.primary().equals("tin")) {
                resolvedMissingPrimary = true;
            }
        }
        if (!resolvedMissingPrimary) {
            throw new AssertionError("test context did not exercise missing-primary replacement");
        }
        if (!"geoworld-profile-v1.1".equals(
            generated.model().manifest().worldProfileVersion()
        )) {
            throw new AssertionError(
                "world profile mismatch: " +
                generated.model().manifest().worldProfileVersion()
            );
        }
        if (!("geoworld-rules-v1.1.22".equals(
            generated.model().manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.23".equals(
            generated.model().manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.24".equals(
            generated.model().manifest().generationRulesVersion()
        ) || "geoworld-rules-v1.1.25".equals(
            generated.model().manifest().generationRulesVersion()
        ))))) {
            throw new AssertionError(
                "generation rules mismatch: " +
                generated.model().manifest().generationRulesVersion()
            );
        }
        if (!"geoworld-model-v1".equals(
            generated.model().manifest().worldModelVersion()
        )) {
            throw new AssertionError(
                "world model mismatch: " +
                generated.model().manifest().worldModelVersion()
            );
        }
        var geometry = generated.model().manifest().geometry();
        if (geometry.width() != 48 || geometry.height() != 30) {
            throw new AssertionError(
                "geometry mismatch: " +
                geometry.width() +
                "x" +
                geometry.height()
            );
        }
        if (
            Math.abs(geometry.spanXBlocks() - 6016.0) > 1.0e-9 ||
            Math.abs(geometry.spanZBlocks() - 3712.0) > 1.0e-9
        ) {
            throw new AssertionError(
                "geometry span mismatch: " +
                geometry.spanXBlocks() +
                "x" +
                geometry.spanZBlocks()
            );
        }

        var sample = generated.model().sample(0.0, 0.0);
        System.out.printf(
            "GeoWorld engine bridge verification passed: seed=%d, grid=%dx%d, " +
            "cacheKey=%s, cached=%s, duration=%dms, centerBiome=%s, water=%s%n",
            seed,
            geometry.width(),
            geometry.height(),
            generated.cacheKey(),
            generated.cached(),
            generated.durationMs(),
            sample.biomeFamily(),
            sample.waterKind()
        );
    }
}
