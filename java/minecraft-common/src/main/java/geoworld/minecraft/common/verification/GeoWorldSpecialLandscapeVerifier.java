package geoworld.minecraft.common.verification;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig.ContinentPreset;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.util.HashMap;
import java.util.Map;

public final class GeoWorldSpecialLandscapeVerifier {
    private GeoWorldSpecialLandscapeVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldCreationConfig config = new GeoWorldCreationConfig(
            96,
            60,
            128,
            18,
            0.68,
            0.35,
            0.65,
            ContinentPreset.DEFAULT
        );
        long seed = 123456L;

        GeoWorldEngineBridge.GeneratedModel generated =
            GeoWorldEngineBridge.generate(seed, config);
        GeoWorldV1 model = generated.model();

        if (!"geoworld-rules-v1.1.11".equals(
            model.manifest().generationRulesVersion()
        )) {
            throw new AssertionError(
                "special landscape verifier requires v1.1.11, got " +
                model.manifest().generationRulesVersion()
            );
        }

        var landscapes = model.specialLandscapes();
        if (landscapes.isEmpty()) {
            throw new AssertionError("generated model has no special landscapes");
        }

        Map<String, Integer> counts = new HashMap<>();
        double sea = model.manifest().seaLevelNormalized();

        for (GeoWorldV1.SpecialLandscape landscape : landscapes) {
            counts.merge(landscape.kind(), 1, Integer::sum);

            if (
                "pale_garden_island".equals(landscape.kind()) &&
                !"ocean".equals(landscape.waterContext())
            ) {
                throw new AssertionError(
                    "pale garden island is not ocean-only: " + landscape
                );
            }
            if (
                "cherry_island".equals(landscape.kind()) &&
                !("ocean".equals(landscape.waterContext()) ||
                  "lake".equals(landscape.waterContext()))
            ) {
                throw new AssertionError(
                    "cherry island has invalid water context: " + landscape
                );
            }
            if (
                "mushroom_island".equals(landscape.kind()) &&
                !("ocean".equals(landscape.waterContext()) ||
                  "lake".equals(landscape.waterContext()))
            ) {
                throw new AssertionError(
                    "mushroom island has invalid water context: " + landscape
                );
            }

            GeoWorldV1.RuntimeSample sample = model.sample(
                landscape.centerX(),
                landscape.centerZ()
            );
            if (!landscape.kind().equals(sample.specialFeature())) {
                throw new AssertionError(
                    "runtime sample lost special landscape " + landscape +
                    ", sampleFeature=" + sample.specialFeature()
                );
            }

            if (landscape.kind().endsWith("_island")) {
                if (!"None".equals(sample.waterKind()) || !sample.sourceLand()) {
                    throw new AssertionError(
                        "special island center is not land: " + landscape +
                        ", water=" + sample.waterKind()
                    );
                }
                if (sample.groundHeight() <= sea) {
                    throw new AssertionError(
                        "special island center is not above sea datum: " +
                        landscape + ", height=" + sample.groundHeight()
                    );
                }
            }
        }

        requireKind(counts, "mushroom_island");
        requireKind(counts, "pale_garden_island");
        requireKind(counts, "cherry_island");
        requireKind(counts, "sinkhole");

        System.out.printf(
            "GeoWorld special landscape verification passed: seed=%d, landscapes=%d, kinds=%s, cacheKey=%s%n",
            seed,
            landscapes.size(),
            counts,
            generated.cacheKey()
        );
    }

    private static void requireKind(Map<String, Integer> counts, String kind) {
        if (counts.getOrDefault(kind, 0) <= 0) {
            throw new AssertionError(
                "missing required special landscape kind " + kind +
                ", got=" + counts
            );
        }
    }
}
