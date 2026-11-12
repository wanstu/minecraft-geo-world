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

        if (!("geoworld-rules-v1.1.22".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.23".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.24".equals(
            model.manifest().generationRulesVersion()
        ) || "geoworld-rules-v1.1.25".equals(
            model.manifest().generationRulesVersion()
        ))))) {
            throw new AssertionError(
                "special landscape verifier requires v1.1.22-v1.1.25, got " +
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
                verifyIslandMoat(model, landscape);
            }

            if (
                "sinkhole".equals(landscape.kind()) ||
                "basin_cherry".equals(landscape.kind()) ||
                "basin_snow".equals(landscape.kind()) ||
                "volcanic_crater_cherry".equals(landscape.kind()) ||
                "volcanic_crater_snow".equals(landscape.kind())
            ) {
                verifyDepressionSemantics(model, landscape);
            }

            System.out.printf(
                "SPECIAL_POINT kind=%s id=%d x=%.0f z=%.0f radius=%.0fx%.0f water=%s feature=%s%n",
                landscape.kind(),
                landscape.id(),
                landscape.centerX(),
                landscape.centerZ(),
                landscape.radiusX(),
                landscape.radiusZ(),
                landscape.waterContext(),
                sample.specialFeature()
            );
        }

        requireKind(counts, "mushroom_island");
        requireKind(counts, "pale_garden_island");
        requireKind(counts, "cherry_island");
        requireKind(counts, "sinkhole");
        requireAnyKind(
            counts,
            "basin_cherry",
            "basin_snow",
            "volcanic_crater_cherry",
            "volcanic_crater_snow"
        );

        System.out.printf(
            "GeoWorld special landscape verification passed: seed=%d, landscapes=%d, kinds=%s, cacheKey=%s%n",
            seed,
            landscapes.size(),
            counts,
            generated.cacheKey()
        );
    }

    private static void verifyIslandMoat(
        GeoWorldV1 model,
        GeoWorldV1.SpecialLandscape landscape
    ) {
        String expectedWater = switch (landscape.waterContext()) {
            case "ocean" -> "Ocean";
            case "lake" -> "Lake";
            default -> throw new AssertionError(
                "special island has no water context: " + landscape
            );
        };
        double searchLimit =
            Math.max(landscape.radiusX(), landscape.radiusZ()) * 3.0 + 256.0;
        for (int i = 0; i < 48; i++) {
            double angle = 2.0 * Math.PI * i / 48.0;
            double dx = Math.cos(angle);
            double dz = Math.sin(angle);
            double lastFeatureRadius = -1.0;
            for (double radius = 0.0; radius <= searchLimit; radius += 4.0) {
                GeoWorldV1.RuntimeSample sample = model.sample(
                    landscape.centerX() + dx * radius,
                    landscape.centerZ() + dz * radius
                );
                if (landscape.kind().equals(sample.specialFeature())) {
                    lastFeatureRadius = radius;
                }
            }
            if (lastFeatureRadius < 0.0) {
                throw new AssertionError(
                    "special island vanished along radial sample: kind=" +
                    landscape.kind() + " angle=" + angle
                );
            }
            for (double moat : new double[] {16.0, 32.0}) {
                double radius = lastFeatureRadius + moat;
                GeoWorldV1.RuntimeSample sample = model.sample(
                    landscape.centerX() + dx * radius,
                    landscape.centerZ() + dz * radius
                );
                if (
                    !expectedWater.equals(sample.waterKind()) ||
                    landscape.kind().equals(sample.specialFeature())
                ) {
                    throw new AssertionError(
                        "special island lost water moat: kind=" +
                        landscape.kind() + " context=" +
                        landscape.waterContext() + " angle=" + angle +
                        " edge=" + lastFeatureRadius + " moat=" + moat +
                        " water=" + sample.waterKind() + " feature=" +
                        sample.specialFeature()
                    );
                }
            }
        }
    }

    private static void verifyDepressionSemantics(
        GeoWorldV1 model,
        GeoWorldV1.SpecialLandscape landscape
    ) {
        String kind = landscape.kind();
        if (
            landscape.baseHeight() <= 0.0 ||
            landscape.floorHeight() <= 0.0 ||
            landscape.floorHeight() >= landscape.baseHeight()
        ) {
            throw new AssertionError(
                "invalid carved-depression height contract: " + landscape
            );
        }
        if (landscape.rimHeight() > landscape.baseHeight() + 1.0e-9) {
            throw new AssertionError(
                "non-island special terrain invented a raised rim: " +
                landscape
            );
        }

        double maxRadius = Math.max(
            landscape.radiusX(),
            landscape.radiusZ()
        );
        if ("sinkhole".equals(kind) && maxRadius > 280.0) {
            throw new AssertionError(
                "sinkhole is too large and reads as a basin: " + landscape
            );
        }
        if (kind.startsWith("volcanic_crater_") && maxRadius > 300.0) {
            throw new AssertionError(
                "volcanic crater is too large for a single peak: " + landscape
            );
        }
        if (kind.startsWith("basin_") && maxRadius < 500.0) {
            throw new AssertionError(
                "basin is too small to remain distinct from a sinkhole: " +
                landscape
            );
        }

        GeoWorldV1.RuntimeSample center = model.sample(
            landscape.centerX(),
            landscape.centerZ()
        );
        GeoWorldV1.RuntimeSample inner = model.sample(
            landscape.centerX() + landscape.radiusX() * 0.30,
            landscape.centerZ()
        );
        GeoWorldV1.RuntimeSample outer = model.sample(
            landscape.centerX() + landscape.radiusX() * 0.92,
            landscape.centerZ()
        );

        if (
            !"Plain".equals(center.terrainVariant()) ||
            !"Plain".equals(inner.terrainVariant())
        ) {
            throw new AssertionError(
                "depression floor is not plain: kind=" + kind +
                " center=" + center.terrainVariant() +
                " inner=" + inner.terrainVariant()
            );
        }
        if (!habitableFloorBiome(center.biomeFamily()) ||
            !habitableFloorBiome(inner.biomeFamily())) {
            throw new AssertionError(
                "depression floor has forbidden biome: kind=" + kind +
                " center=" + center.biomeFamily() +
                " inner=" + inner.biomeFamily()
            );
        }
        boolean continuousSinkhole = "sinkhole".equals(kind) &&
            "geoworld-rules-v1.1.25".equals(
                model.manifest().generationRulesVersion()
            );
        if (continuousSinkhole) {
            // v1.1.25 intentionally replaced the large flat inner floor.
            // Verify a continuous descent toward the bottom rather than
            // silently relaxing the older flat-floor contract.
            GeoWorldV1.RuntimeSample middle = model.sample(
                landscape.centerX() + landscape.radiusX() * 0.55,
                landscape.centerZ()
            );
            double centerToInner =
                inner.groundHeight() - center.groundHeight();
            double innerToMiddle =
                middle.groundHeight() - inner.groundHeight();
            double middleToOuter =
                outer.groundHeight() - middle.groundHeight();
            if (
                centerToInner < 0.008 || centerToInner > 0.05 ||
                innerToMiddle < 0.010 || innerToMiddle > 0.08 ||
                middleToOuter < 0.005 || middleToOuter > 0.08
            ) {
                throw new AssertionError(
                    "new sinkhole slope is not gradual: kind=" + kind +
                    " center=" + center.groundHeight() +
                    " inner=" + inner.groundHeight() +
                    " middle=" + middle.groundHeight() +
                    " outer=" + outer.groundHeight()
                );
            }
        } else if (
            Math.abs(center.groundHeight() - inner.groundHeight()) > 0.015
        ) {
            // Preserve pre-v1.1.25 semantics for old sinkholes, basins,
            // and volcanic crater floors.
            throw new AssertionError(
                "depression center is not flat: kind=" + kind +
                " center=" + center.groundHeight() +
                " inner=" + inner.groundHeight()
            );
        }
        if (outer.groundHeight() - center.groundHeight() < 0.018) {
            throw new AssertionError(
                "depression did not carve below surrounding terrain: kind=" +
                kind + " center=" + center.groundHeight() +
                " outer=" + outer.groundHeight()
            );
        }

        if ("sinkhole".equals(kind)) {
            boolean canyonWall = false;
            double minWall = Double.POSITIVE_INFINITY;
            double maxWall = Double.NEGATIVE_INFINITY;
            for (int i = 0; i < 16; i++) {
                double angle = 2.0 * Math.PI * i / 16.0;
                double x = landscape.centerX() +
                    Math.cos(angle) * landscape.radiusX() * 0.68;
                double z = landscape.centerZ() +
                    Math.sin(angle) * landscape.radiusZ() * 0.68;
                GeoWorldV1.RuntimeSample wall = model.sample(x, z);
                minWall = Math.min(minWall, wall.groundHeight());
                maxWall = Math.max(maxWall, wall.groundHeight());
                canyonWall |= "Canyon".equals(wall.terrainVariant());
            }
            if (!canyonWall) {
                throw new AssertionError(
                    "sinkhole has no canyon-like carved wall: " + landscape
                );
            }
            if (maxWall - minWall < 0.004) {
                throw new AssertionError(
                    "sinkhole wall is still an unnaturally uniform ring: " +
                    landscape
                );
            }
        }
    }

    private static boolean habitableFloorBiome(String biome) {
        return "Temperate Grassland".equals(biome) ||
            "Mediterranean Shrubland".equals(biome) ||
            "Savanna".equals(biome);
    }

    private static void requireAnyKind(
        Map<String, Integer> counts,
        String... kinds
    ) {
        for (String kind : kinds) {
            if (counts.getOrDefault(kind, 0) > 0) return;
        }
        throw new AssertionError(
            "missing required special landscape family " +
            String.join("/", kinds) + ", got=" + counts
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
