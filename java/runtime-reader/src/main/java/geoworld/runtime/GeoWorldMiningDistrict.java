package geoworld.runtime;

import java.util.LinkedHashMap;
import java.util.Map;

final class GeoWorldMiningDistrict {
    record Score(
        double miningSuitability,
        double surfaceMiningSuitability,
        double shallowMiningSuitability,
        double deepMiningSuitability,
        double industrialValue,
        double resourceDiversity,
        String dominantResource,
        String miningClass,
        Map<String, Double> resourceProfile
    ) {}

    private static final double[] INDUSTRIAL_WEIGHTS = {
        0.85, 1.0, 0.85, 0.35, 0.20, 0.45, 0.15, 0.15
    };
    private static final double[] DEEP_STRATEGIC_WEIGHTS = {
        0.10, 0.40, 0.15, 0.60, 0.30, 0.75, 1.0, 0.25
    };

    private GeoWorldMiningDistrict() {}

    static Score evaluate(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample sample,
        double worldX,
        double worldZ
    ) {
        if (!sample.sourceLand() || !"None".equals(sample.waterKind())) {
            return new Score(
                0, 0, 0, 0, 0, 0,
                "none", "none", zeroProfile()
            );
        }

        double[] surface = GeoWorldMineralAffinity.normalizedProfile(
            sample.terrainVariant(),
            sample.biomeFamily(),
            80
        );
        double[] shallow = GeoWorldMineralAffinity.normalizedProfile(
            sample.terrainVariant(),
            sample.biomeFamily(),
            28
        );
        double[] deep = GeoWorldMineralAffinity.normalizedProfile(
            sample.terrainVariant(),
            sample.biomeFamily(),
            -40
        );

        double[] composite =
            new double[GeoWorldMineralAffinity.MINERALS.length];
        for (int i = 0; i < composite.length; i++) {
            composite[i] =
                0.30 * surface[i] +
                0.40 * shallow[i] +
                0.30 * deep[i];
        }

        double industrialSurface = industrial(surface);
        double industrialShallow = industrial(shallow);
        double industrialDeep = industrial(deep);
        double industrialValue = industrial(composite);
        double diversity = resourceDiversity(composite);
        double province = provinceSignal(model, worldX, worldZ);
        double exposure = terrainExposure(sample.terrainVariant());
        double access = terrainAccess(sample.terrainVariant());
        double criticalDeep = clamp01(1.55 * deepStrategicValue(deep));

        double surfaceScore = clamp01(
            0.55 * exposure +
            0.30 * industrialSurface +
            0.15 * province
        );
        double shallowScore = clamp01(
            0.40 * industrialShallow +
            0.25 * diversity +
            0.20 * access +
            0.15 * province
        );
        double deepScore = clamp01(
            0.48 * criticalDeep +
            0.22 * industrialDeep +
            0.20 * diversity +
            0.10 * province
        );

        double maxLayer = Math.max(
            surfaceScore,
            Math.max(shallowScore, deepScore)
        );
        double mining = clamp01(
            0.60 * maxLayer +
            0.25 * industrialValue +
            0.15 * diversity
        );

        Map<String, Double> profile = new LinkedHashMap<>();
        String dominant = "none";
        double dominantValue = -1.0;
        for (int i = 0; i < composite.length; i++) {
            String mineral =
                GeoWorldMineralAffinity.MINERALS[i].toLowerCase(
                    java.util.Locale.ROOT
                );
            double value = clamp01(composite[i]);
            profile.put(mineral, value);
            if (value > dominantValue) {
                dominant = mineral;
                dominantValue = value;
            }
        }

        return new Score(
            mining,
            surfaceScore,
            shallowScore,
            deepScore,
            clamp01(industrialValue),
            clamp01(diversity),
            dominant,
            miningClass(
                mining,
                surfaceScore,
                shallowScore,
                deepScore
            ),
            Map.copyOf(profile)
        );
    }

    private static double industrial(double[] profile) {
        double total = 0.0;
        for (int i = 0; i < profile.length; i++) {
            total += profile[i] * INDUSTRIAL_WEIGHTS[i];
        }
        return clamp01(total);
    }

    private static double deepStrategicValue(double[] profile) {
        double total = 0.0;
        for (int i = 0; i < profile.length; i++) {
            total += profile[i] * DEEP_STRATEGIC_WEIGHTS[i];
        }
        return clamp01(total);
    }

    private static double resourceDiversity(double[] profile) {
        double entropy = 0.0;
        for (double p : profile) {
            if (p > 0) {
                entropy -= p * Math.log(p);
            }
        }
        return clamp01(
            entropy / Math.log((double)GeoWorldMineralAffinity.MINERALS.length)
        );
    }

    private static double terrainExposure(String terrain) {
        return switch (terrain) {
            case "Mountain" -> 1.0;
            case "Canyon" -> 0.92;
            case "Foothills" -> 0.85;
            case "Hills" -> 0.75;
            case "Plateau" -> 0.72;
            case "Rolling Plain" -> 0.40;
            case "Plain" -> 0.30;
            case "Alluvial Fan" -> 0.28;
            case "Floodplain", "Alluvial Plain" -> 0.18;
            default -> 0.25;
        };
    }

    private static double terrainAccess(String terrain) {
        return switch (terrain) {
            case "Plain", "Rolling Plain", "Plateau" -> 0.90;
            case "Hills", "Foothills" -> 0.72;
            case "Alluvial Fan" -> 0.68;
            case "Mountain", "Canyon" -> 0.48;
            case "Floodplain", "Alluvial Plain" -> 0.55;
            default -> 0.60;
        };
    }

    private static double provinceSignal(
        GeoWorldV1 model,
        double worldX,
        double worldZ
    ) {
        double regionalization =
            model.worldProfile().parameters().mineralRegionalization();
        if (!Double.isFinite(regionalization) || regionalization <= 0) {
            regionalization = 1.0;
        }
        if (Math.abs(regionalization - 1.0) <= 1.0e-12) {
            return 0.5;
        }
        double strength =
            GeoWorldGeology.provinceStrength(model, worldX, worldZ);
        return clamp01(
            0.5 +
            0.5 * (regionalization - 1.0) * (strength - 0.5)
        );
    }

    private static String miningClass(
        double overall,
        double surface,
        double shallow,
        double deep
    ) {
        if (overall < 0.38) {
            return "low_potential";
        }
        if (
            deep >= 0.74 &&
            deep >= shallow - 0.02 &&
            deep >= surface + 0.02
        ) {
            return "deep_district";
        }
        double[] values = {surface, shallow, deep};
        String[] labels = {
            "surface_district",
            "shallow_district",
            "deep_district"
        };
        int best = 0;
        int second = 1;
        if (values[second] > values[best]) {
            int tmp = best;
            best = second;
            second = tmp;
        }
        for (int i = 2; i < values.length; i++) {
            if (values[i] > values[best]) {
                second = best;
                best = i;
            } else if (values[i] > values[second]) {
                second = i;
            }
        }
        if (values[best] - values[second] <= 0.04) {
            return "mixed_district";
        }
        return labels[best];
    }

    private static Map<String, Double> zeroProfile() {
        Map<String, Double> out = new LinkedHashMap<>();
        for (String mineral : GeoWorldMineralAffinity.MINERALS) {
            out.put(mineral.toLowerCase(java.util.Locale.ROOT), 0.0);
        }
        return Map.copyOf(out);
    }

    private static double clamp01(double value) {
        if (!Double.isFinite(value) || value < 0) {
            return 0;
        }
        return Math.min(1, value);
    }
}
