package geoworld.runtime;

import java.util.Locale;

/**
 * Pure Natural Model derivation for civilization/settlement planning.
 *
 * This class never observes loaded chunks or gameplay state. The score is a
 * geographic potential only; it does not imply that a settlement exists.
 */
final class GeoWorldSettlementSuitability {
    record Score(
        double buildability,
        double freshWaterAccess,
        double floodRisk,
        double agricultureSuitability,
        double climateComfort,
        double transportAccess,
        double resourceAccess,
        double settlementSuitability
    ) {}

    private GeoWorldSettlementSuitability() {}

    static Score evaluate(
        GeoWorldV1.RuntimeSample sample,
        double slope,
        GeoWorldV1.ClimateSample climate,
        GeoWorldV1.NaturalFeatureSample features
    ) {
        if (!sample.sourceLand() || !isDryLand(sample.waterKind())) {
            return new Score(0, 0, 1, 0, 0, 0, 0, 0);
        }

        double buildability = buildabilityScore(slope, sample.terrainVariant());
        double freshWater = freshWaterScore(features);
        double floodRisk = floodRiskScore(sample, features);
        double agriculture = agricultureScore(sample, slope, climate, features);
        double climateComfort = climateComfortScore(climate);
        double transport = transportScore(
            sample,
            features,
            buildability,
            freshWater
        );
        double resources = resourceScore(sample, features);

        double safety = 1.0 - floodRisk;
        double composite =
            0.22 * buildability +
            0.16 * freshWater +
            0.16 * safety +
            0.16 * agriculture +
            0.10 * climateComfort +
            0.12 * transport +
            0.08 * resources;
        composite *= 0.55 + 0.45 * buildability;
        composite *= 0.62 + 0.38 * safety;

        return new Score(
            clamp01(buildability),
            clamp01(freshWater),
            clamp01(floodRisk),
            clamp01(agriculture),
            clamp01(climateComfort),
            clamp01(transport),
            clamp01(resources),
            clamp01(composite)
        );
    }

    private static double buildabilityScore(double slope, String terrain) {
        slope = Math.max(0, slope);
        double slopeScore = 1.0 - smoothstep(0.0015, 0.014, slope);
        double terrainScore = switch (normalize(terrain)) {
            case "plain" -> 1.0;
            case "rolling plain" -> 0.92;
            case "alluvial plain" -> 0.98;
            case "floodplain" -> 0.90;
            case "alluvial fan" -> 0.84;
            case "plateau" -> 0.78;
            case "hills" -> 0.62;
            case "foothills" -> 0.55;
            case "mountain" -> 0.18;
            case "canyon" -> 0.12;
            case "ocean" -> 0.0;
            default -> 0.58;
        };
        return slopeScore * terrainScore;
    }

    private static double freshWaterScore(
        GeoWorldV1.NaturalFeatureSample features
    ) {
        if (features == null) return 0.12;
        double best = Double.POSITIVE_INFINITY;
        if (features.distanceToRiver() != null) {
            best = Math.min(best, Math.max(0, features.distanceToRiver()));
        }
        if (features.distanceToLake() != null) {
            best = Math.min(best, Math.max(0, features.distanceToLake()));
        }
        if (Double.isFinite(best)) {
            return 1.0 - smoothstep(128, 1600, best);
        }
        return 0.12;
    }

    private static double floodRiskScore(
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.NaturalFeatureSample features
    ) {
        double risk = switch (normalize(sample.terrainVariant())) {
            case "floodplain" -> 0.72;
            case "alluvial plain" -> 0.34;
            case "alluvial fan" -> 0.22;
            case "canyon" -> 0.16;
            default -> 0.04;
        };

        if (features != null && features.distanceToRiver() != null) {
            double d = Math.max(0, features.distanceToRiver());
            double riverRisk = switch (distanceBand(d, 24, 64, 180, 480)) {
                case 0 -> 0.92;
                case 1 -> lerp(0.92, 0.62, (d - 24) / 40);
                case 2 -> lerp(0.62, 0.22, (d - 64) / 116);
                case 3 -> lerp(0.22, 0.04, (d - 180) / 300);
                default -> 0.0;
            };
            risk = Math.max(risk, riverRisk);
        }

        if (features != null && features.distanceToLake() != null) {
            double d = Math.max(0, features.distanceToLake());
            double lakeRisk;
            if (d <= 16) {
                lakeRisk = 0.62;
            } else if (d <= 96) {
                lakeRisk = lerp(0.62, 0.26, (d - 16) / 80);
            } else if (d <= 320) {
                lakeRisk = lerp(0.26, 0.04, (d - 96) / 224);
            } else {
                lakeRisk = 0.0;
            }
            risk = Math.max(risk, lakeRisk);
        }

        String surface = normalize(sample.surfaceProfile());
        if (surface.contains("wet mud") || surface.contains("peat")) {
            risk = Math.max(risk, 0.68);
        }
        if ("wetland".equals(normalize(sample.biomeFamily()))) {
            risk = Math.max(risk, 0.80);
        }
        return clamp01(risk);
    }

    private static double agricultureScore(
        GeoWorldV1.RuntimeSample sample,
        double slope,
        GeoWorldV1.ClimateSample climate,
        GeoWorldV1.NaturalFeatureSample features
    ) {
        if (climate == null || !climate.available()) return 0;
        double temp = bandScore(climate.annualMeanTempC(), -4, 8, 24, 36);
        double precip = bandScore(climate.annualPrecipMM(), 120, 450, 1800, 3400);

        double soil = switch (normalize(sample.surfaceProfile())) {
            case "grass loam" -> 1.0;
            case "alluvial silt" -> 0.98;
            case "forest loam" -> 0.86;
            case "dry grass soil" -> 0.70;
            case "shrub soil" -> 0.56;
            case "podzol" -> 0.48;
            case "wet mud" -> 0.42;
            case "peat" -> 0.38;
            case "tundra soil" -> 0.24;
            case "desert sand", "desert gravel", "alpine rock",
                 "alpine snow", "ice / snow" -> 0.10;
            default -> 0.52;
        };

        double terrain = buildabilityScore(slope, sample.terrainVariant());
        double floodPenalty = 1.0 - 0.45 * floodRiskScore(sample, features);
        return clamp01(
            (0.34 * temp + 0.28 * precip + 0.24 * soil + 0.14 * terrain) *
            floodPenalty
        );
    }

    private static double climateComfortScore(GeoWorldV1.ClimateSample climate) {
        if (climate == null || !climate.available()) return 0;
        double temp = bandScore(climate.annualMeanTempC(), -12, 7, 24, 38);
        double precip = bandScore(climate.annualPrecipMM(), 80, 320, 1800, 3600);
        return clamp01(0.72 * temp + 0.28 * precip);
    }

    private static double transportScore(
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.NaturalFeatureSample features,
        double buildability,
        double freshWater
    ) {
        double waterway = 0.0;
        if (features != null && features.distanceToRiver() != null) {
            double d = Math.max(0, features.distanceToRiver());
            waterway = Math.max(waterway, 1.0 - smoothstep(96, 1100, d));
        }
        if (features != null && features.distanceToLake() != null) {
            double d = Math.max(0, features.distanceToLake());
            waterway = Math.max(
                waterway,
                0.88 * (1.0 - smoothstep(96, 900, d))
            );
        }

        double terrainBonus = switch (normalize(sample.terrainVariant())) {
            case "plain", "alluvial plain" -> 1.0;
            case "rolling plain", "floodplain" -> 0.88;
            case "alluvial fan", "plateau" -> 0.68;
            case "hills", "foothills" -> 0.44;
            case "mountain", "canyon" -> 0.12;
            default -> 0.0;
        };
        return clamp01(
            0.52 * buildability +
            0.30 * waterway +
            0.18 * terrainBonus +
            0.04 * freshWater
        );
    }

    private static double resourceScore(
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.NaturalFeatureSample features
    ) {
        double score = switch (normalize(sample.terrainVariant())) {
            case "foothills" -> 0.88;
            case "hills" -> 0.84;
            case "mountain" -> 0.82;
            case "plateau" -> 0.78;
            case "rolling plain" -> 0.66;
            case "plain", "alluvial plain", "floodplain" -> 0.58;
            case "alluvial fan" -> 0.64;
            case "canyon" -> 0.72;
            default -> 0.50;
        };
        if (normalize(sample.biomeFamily()).contains("forest")) {
            score += 0.08;
        }
        if (freshWaterScore(features) > 0.55) {
            score += 0.05;
        }
        return clamp01(score);
    }

    private static int distanceBand(
        double d,
        double b0,
        double b1,
        double b2,
        double b3
    ) {
        if (d <= b0) return 0;
        if (d <= b1) return 1;
        if (d <= b2) return 2;
        if (d <= b3) return 3;
        return 4;
    }

    private static double bandScore(
        double value,
        double hardLow,
        double goodLow,
        double goodHigh,
        double hardHigh
    ) {
        if (value <= hardLow || value >= hardHigh) return 0;
        if (value < goodLow) return smoothstep(hardLow, goodLow, value);
        if (value <= goodHigh) return 1;
        return 1.0 - smoothstep(goodHigh, hardHigh, value);
    }

    private static boolean isDryLand(String waterKind) {
        String value = normalize(waterKind);
        return value.isEmpty() || "none".equals(value);
    }

    private static String normalize(String value) {
        return value == null
            ? ""
            : value.trim().toLowerCase(Locale.ROOT);
    }

    private static double smoothstep(double edge0, double edge1, double x) {
        if (edge1 <= edge0) return x >= edge1 ? 1 : 0;
        double t = clamp01((x - edge0) / (edge1 - edge0));
        return t * t * (3.0 - 2.0 * t);
    }

    private static double lerp(double a, double b, double t) {
        return a + (b - a) * clamp01(t);
    }

    private static double clamp01(double value) {
        if (!Double.isFinite(value)) return 0;
        return Math.max(0, Math.min(1, value));
    }
}
