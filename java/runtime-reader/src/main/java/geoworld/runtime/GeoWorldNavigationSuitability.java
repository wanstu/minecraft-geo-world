package geoworld.runtime;

import java.util.Locale;

final class GeoWorldNavigationSuitability {
    record Score(
        double waterwayNavigability,
        double portSuitability,
        double navigationBottleneckRisk,
        String waterwayClass,
        String portType
    ) {}

    private GeoWorldNavigationSuitability() {}

    static Score evaluate(
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.NaturalFeatureSample features,
        GeoWorldV1.NavigationFeatureSample river,
        GeoWorldSettlementSuitability.Score settlement
    ) {
        String kind = normalize(sample.waterKind());
        double riverNav = 0.0;
        double riverBottleneck = 0.0;
        if (river != null && river.available()) {
            double[] riverScore = riverNavigation(river);
            riverNav = riverScore[0];
            riverBottleneck = riverScore[1];
        }

        double openWaterNav = 0.0;
        if ("ocean".equals(kind)) {
            openWaterNav = openWaterScore(sample.waterDepth(), 0.002, 0.010);
        } else if ("lake".equals(kind)) {
            openWaterNav = openWaterScore(sample.waterDepth(), 0.002, 0.008);
        }

        double navigability = openWaterNav;
        if ("river".equals(kind)) {
            navigability = riverNav;
        }
        if (
            "none".equals(kind) &&
            river != null &&
            river.available() &&
            river.bankDistance() != null &&
            river.bankDistance() <= 160
        ) {
            navigability = riverNav;
        }

        double bottleneck = 0.0;
        if ("river".equals(kind)) {
            bottleneck = riverBottleneck;
        } else if ("lake".equals(kind) || "ocean".equals(kind)) {
            bottleneck = clamp01(1.0 - openWaterNav);
        } else if (
            river != null &&
            river.available() &&
            river.bankDistance() != null &&
            river.bankDistance() <= 160
        ) {
            bottleneck = riverBottleneck;
        }

        PortAccess portAccess = portWaterAccess(features, river, riverNav);
        if (!sample.sourceLand() || !"none".equals(kind)) {
            portAccess = new PortAccess("none", 0.0, 0.0);
        }
        double port = 0.0;
        if (
            sample.sourceLand() &&
            "none".equals(kind) &&
            !"none".equals(portAccess.type())
        ) {
            double build = settlement == null
                ? 0.0
                : clamp01(settlement.buildability());
            double safety = settlement == null
                ? 0.0
                : 1.0 - clamp01(settlement.floodRisk());
            double settlementScore = settlement == null
                ? 0.0
                : clamp01(settlement.settlementSuitability());
            port =
                0.28 * portAccess.waterQuality() +
                0.24 * build +
                0.18 * safety +
                0.16 * settlementScore +
                0.14 * portAccess.waterAccess();
            port *= 0.52 + 0.48 * portAccess.waterAccess();
            port *= 0.58 + 0.42 * build;
            port *= 0.62 + 0.38 * safety;
        }

        return new Score(
            clamp01(navigability),
            clamp01(port),
            clamp01(bottleneck),
            waterwayClass(
                kind,
                navigability,
                river != null &&
                    river.available() &&
                    river.bankDistance() != null &&
                    river.bankDistance() <= 160
            ),
            portAccess.type()
        );
    }

    private static double[] riverNavigation(
        GeoWorldV1.NavigationFeatureSample river
    ) {
        double halfWidth = river.riverHalfWidth() == null
            ? 0.0
            : Math.max(0, river.riverHalfWidth());
        double gradient = river.riverGradient() == null
            ? 0.0
            : Math.max(0, river.riverGradient());

        double width = smoothstep(6.0, 24.0, halfWidth);
        double gradientRisk = smoothstep(0.00008, 0.00036, gradient);
        double gradientScore = 1.0 - gradientRisk;

        double mouthBonus = 0.0;
        if (river.oceanMouth()) {
            mouthBonus = 0.10;
        } else if (river.lakeMouth() || river.lakeOutlet()) {
            mouthBonus = 0.06;
        }

        double navigability = clamp01(
            0.68 * width + 0.32 * gradientScore + mouthBonus
        );
        double widthRisk = 1.0 - smoothstep(7.5, 18.0, halfWidth);
        double bottleneck = Math.max(widthRisk, gradientRisk);
        if (river.oceanMouth() || river.lakeMouth() || river.lakeOutlet()) {
            bottleneck *= 0.82;
        }
        return new double[]{navigability, clamp01(bottleneck)};
    }

    private static double openWaterScore(
        double depth,
        double shallow,
        double good
    ) {
        if (depth <= 0) {
            return 0.92;
        }
        return clamp01(0.72 + 0.28 * smoothstep(shallow, good, depth));
    }

    private record PortAccess(
        String type,
        double waterAccess,
        double waterQuality
    ) {}

    private static PortAccess portWaterAccess(
        GeoWorldV1.NaturalFeatureSample features,
        GeoWorldV1.NavigationFeatureSample river,
        double riverNav
    ) {
        String bestType = "none";
        double bestAccess = 0.0;
        double bestQuality = 0.0;

        if (features != null && features.distanceToCoast() != null) {
            double access = shoreBand(
                Math.max(0, features.distanceToCoast()),
                16,
                220,
                640
            );
            if (access > bestAccess) {
                bestType = "sea";
                bestAccess = access;
                bestQuality = 1.0;
            }
        }

        if (features != null && features.distanceToLake() != null) {
            double access = shoreBand(
                Math.max(0, features.distanceToLake()),
                12,
                180,
                520
            );
            if (access > bestAccess) {
                bestType = "lake";
                bestAccess = access;
                bestQuality = 0.90;
            }
        }

        if (
            river != null &&
            river.available() &&
            river.bankDistance() != null
        ) {
            double access = shoreBand(
                Math.max(0, river.bankDistance()),
                8,
                150,
                420
            );
            double quality = riverNav;
            if (river.oceanMouth()) {
                quality = clamp01(quality + 0.12);
            }
            if (access * quality > bestAccess * bestQuality) {
                bestType = "river";
                bestAccess = access;
                bestQuality = quality;
            }
        }

        return new PortAccess(
            bestType,
            clamp01(bestAccess),
            clamp01(bestQuality)
        );
    }

    private static double shoreBand(
        double distance,
        double tooClose,
        double idealFar,
        double maxDistance
    ) {
        if (distance >= maxDistance) {
            return 0;
        }
        if (distance <= tooClose) {
            return 0.72;
        }
        if (distance <= idealFar) {
            return 1;
        }
        return 1.0 - smoothstep(idealFar, maxDistance, distance);
    }

    private static String waterwayClass(
        String kind,
        double navigability,
        boolean riverAvailable
    ) {
        if ("ocean".equals(kind)) {
            return navigability >= 0.65 ? "open_sea" : "shallow_sea";
        }
        if ("lake".equals(kind)) {
            return navigability >= 0.65 ? "lake" : "shallow_lake";
        }
        if ("river".equals(kind) || riverAvailable) {
            if (navigability >= 0.82) {
                return "major_river";
            }
            if (navigability >= 0.58) {
                return "river";
            }
            if (navigability >= 0.34) {
                return "local_waterway";
            }
            return "non_navigable";
        }
        return "none";
    }

    private static String normalize(String value) {
        return value == null
            ? ""
            : value.trim().toLowerCase(Locale.ROOT);
    }

    private static double smoothstep(double edge0, double edge1, double value) {
        if (edge1 <= edge0) {
            return value >= edge1 ? 1 : 0;
        }
        double t = clamp01((value - edge0) / (edge1 - edge0));
        return t * t * (3.0 - 2.0 * t);
    }

    private static double clamp01(double value) {
        if (!Double.isFinite(value)) {
            return 0;
        }
        return Math.max(0, Math.min(1, value));
    }
}
