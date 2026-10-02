package geoworld.neoforge.client;

import geoworld.neoforge.engine.GeoWorldEngineBridge;

import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.creation.GeoWorldCreationConfig.ContinentPreset;

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

        GeoWorldEngineBridge.GeneratedModel generated =
            GeoWorldEngineBridge.generate(seed, config);

        if (generated.model().manifest().seed() != seed) {
            throw new AssertionError("seed mismatch");
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
