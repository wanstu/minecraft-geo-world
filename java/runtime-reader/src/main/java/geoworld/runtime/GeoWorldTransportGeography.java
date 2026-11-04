package geoworld.runtime;

final class GeoWorldTransportGeography {
    record Score(
        double corridorSuitability,
        double riverValleyCorridor,
        double hillPassCorridor,
        double mountainPassCorridor,
        double plateauAccessCorridor,
        double basinGatewayCorridor,
        double lowDivideCorridor,
        double straitIsthmusPotential,
        double chokepointStrength,
        String corridorClass
    ) {}

    private GeoWorldTransportGeography() {}

    static Score evaluate(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.HydrologySample hydrology,
        GeoWorldSettlementSuitability.Score settlement,
        GeoWorldNavigationSuitability.Score navigation,
        double x,
        double z
    ) {
        boolean land = sample.sourceLand();
        double straitIsthmus = straitIsthmusScore(model, x, z, land);
        if (!land) {
            return new Score(
                0, 0, 0, 0, 0, 0, 0,
                straitIsthmus,
                straitIsthmus,
                "none"
            );
        }

        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        double stepX = Math.max(1.0, geometry.spacingX());
        double stepZ = Math.max(1.0, geometry.spacingZ());
        double slope = slopeAt(model, x, z, stepX, stepZ);
        double lowSlope = 1.0 - smoothstep(0.0015, 0.014, Math.max(0, slope));
        double build = clamp01(settlement.buildability());
        double transport = clamp01(settlement.transportAccess());
        double fresh = clamp01(settlement.freshWaterAccess());
        double floodSafety = 1.0 - clamp01(settlement.floodRisk());

        double saddle = saddleScore(model, x, z, stepX, stepZ);
        double basinBoundary = basinBoundaryScore(model, hydrology, x, z, stepX, stepZ);
        double plateauEdge = plateauEdgeScore(model, sample, x, z, stepX, stepZ);
        double riverStrength = riverStrength(model, hydrology);

        double riverValley = clamp01(
            0.28 * transport +
            0.22 * fresh +
            0.18 * lowSlope +
            0.14 * build +
            0.12 * riverStrength +
            0.06 * floodSafety
        );

        double hillContext = terrainWeight(
            sample.terrainVariant(),
            "Hills", "Foothills"
        );
        double hillPass = clamp01(
            hillContext * (
                0.34 * transport +
                0.24 * build +
                0.22 * saddle +
                0.20 * lowSlope
            )
        );

        double mountainContext = terrainWeight(
            sample.terrainVariant(),
            "Mountain", "Canyon", "Foothills"
        );
        double mountainPass = clamp01(
            mountainContext * (
                0.44 * saddle +
                0.24 * transport +
                0.18 * lowSlope +
                0.14 * build
            )
        );

        double plateauContext = terrainWeight(
            sample.terrainVariant(),
            "Plateau", "Foothills", "Hills"
        );
        double plateauAccess = clamp01(
            plateauContext * (
                0.36 * plateauEdge +
                0.28 * transport +
                0.20 * build +
                0.16 * lowSlope
            )
        );

        double basinGateway = clamp01(
            basinBoundary * (
                0.32 * transport +
                0.22 * build +
                0.20 * lowSlope +
                0.14 * fresh +
                0.12 * (1.0 - saddle)
            )
        );

        double lowDivide = clamp01(
            basinBoundary * (
                0.34 * lowSlope +
                0.28 * transport +
                0.18 * build +
                0.12 * (1.0 - saddle) +
                0.08 * floodSafety
            )
        );

        double[] values = {
            riverValley,
            hillPass,
            mountainPass,
            plateauAccess,
            basinGateway,
            lowDivide
        };
        String[] classes = {
            "river_valley",
            "hill_pass",
            "mountain_pass",
            "plateau_access",
            "basin_gateway",
            "low_divide"
        };
        int best = 0;
        for (int i = 1; i < values.length; i++) {
            if (values[i] > values[best]) {
                best = i;
            }
        }
        String corridorClass = values[best] < 0.34
            ? "low_potential"
            : classes[best];
        double chokepoint = Math.max(
            Math.max(mountainPass, hillPass),
            Math.max(
                Math.max(basinGateway, lowDivide),
                straitIsthmus
            )
        );

        return new Score(
            clamp01(values[best]),
            riverValley,
            hillPass,
            mountainPass,
            plateauAccess,
            basinGateway,
            lowDivide,
            straitIsthmus,
            clamp01(chokepoint),
            corridorClass
        );
    }

    private static double slopeAt(
        GeoWorldV1 model,
        double x,
        double z,
        double stepX,
        double stepZ
    ) {
        double left = model.sample(x - stepX, z).groundHeight();
        double right = model.sample(x + stepX, z).groundHeight();
        double down = model.sample(x, z - stepZ).groundHeight();
        double up = model.sample(x, z + stepZ).groundHeight();
        double dx = (right - left) / (2.0 * stepX);
        double dz = (up - down) / (2.0 * stepZ);
        return Math.hypot(dx, dz);
    }

    private static double saddleScore(
        GeoWorldV1 model,
        double x,
        double z,
        double stepX,
        double stepZ
    ) {
        double c = model.sample(x, z).groundHeight();
        double n = model.sample(x, z - stepZ).groundHeight();
        double s = model.sample(x, z + stepZ).groundHeight();
        double w = model.sample(x - stepX, z).groundHeight();
        double e = model.sample(x + stepX, z).groundHeight();

        double ns = Math.min(Math.max(0, n - c), Math.max(0, s - c));
        double ew = Math.min(Math.max(0, e - c), Math.max(0, w - c));
        double barrier = Math.max(ns, ew);
        double open = Math.min(ns, ew);
        double barrierScore = smoothstep(0.006, 0.055, barrier);
        double anisotropy = clamp01((barrier - open) / 0.035);
        return clamp01(barrierScore * (0.45 + 0.55 * anisotropy));
    }

    private static double basinBoundaryScore(
        GeoWorldV1 model,
        GeoWorldV1.HydrologySample center,
        double x,
        double z,
        double stepX,
        double stepZ
    ) {
        if (center == null || !center.available() || center.watershedId() <= 0) {
            return 0;
        }
        int[] seen = new int[5];
        seen[0] = center.watershedId();
        int distinct = 1;
        double[][] offsets = {
            {stepX, 0},
            {-stepX, 0},
            {0, stepZ},
            {0, -stepZ}
        };
        for (double[] offset : offsets) {
            GeoWorldV1.HydrologySample neighbor =
                model.sampleHydrology(x + offset[0], z + offset[1]);
            if (!neighbor.available() || neighbor.watershedId() <= 0) {
                continue;
            }
            int id = neighbor.watershedId();
            boolean known = false;
            for (int i = 0; i < distinct; i++) {
                if (seen[i] == id) {
                    known = true;
                    break;
                }
            }
            if (!known) {
                seen[distinct++] = id;
            }
        }
        if (distinct < 2) {
            return 0;
        }
        return clamp01(0.55 + 0.15 * (distinct - 2));
    }

    private static double plateauEdgeScore(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample center,
        double x,
        double z,
        double stepX,
        double stepZ
    ) {
        boolean plateau = "Plateau".equals(center.terrainVariant());
        int transitions = 0;
        GeoWorldV1.RuntimeSample[] neighbors = {
            model.sample(x + stepX, z),
            model.sample(x - stepX, z),
            model.sample(x, z + stepZ),
            model.sample(x, z - stepZ)
        };
        for (GeoWorldV1.RuntimeSample neighbor : neighbors) {
            if ("Plateau".equals(neighbor.terrainVariant()) != plateau) {
                transitions++;
            }
        }
        return transitions == 0 ? 0 : clamp01(transitions / 4.0);
    }

    private static double riverStrength(
        GeoWorldV1 model,
        GeoWorldV1.HydrologySample hydrology
    ) {
        if (hydrology == null || !hydrology.available()) {
            return 0;
        }
        double order = clamp01(hydrology.riverOrder() / 5.0);
        double accumulation = 0;
        if (hydrology.flowAccumulation() > 0) {
            accumulation = Math.sqrt(
                clamp01(
                    hydrology.flowAccumulation() /
                    model.maxFlowAccumulation()
                )
            );
        }
        return Math.max(order, accumulation);
    }

    private static double straitIsthmusScore(
        GeoWorldV1 model,
        double x,
        double z,
        boolean land
    ) {
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        double stepX = Math.max(1.0, geometry.spacingX());
        double stepZ = Math.max(1.0, geometry.spacingZ());
        final int maxRadius = 4;
        double best = 0;

        boolean west = model.sample(x - stepX, z).sourceLand();
        boolean east = model.sample(x + stepX, z).sourceLand();
        boolean north = model.sample(x, z - stepZ).sourceLand();
        boolean south = model.sample(x, z + stepZ).sourceLand();

        if (land) {
            if (west && east) {
                int a = firstLandStateDistance(
                    model, x, z, 0, -stepZ, false, maxRadius
                );
                int b = firstLandStateDistance(
                    model, x, z, 0, stepZ, false, maxRadius
                );
                best = Math.max(best, crossingNarrowness(a, b, maxRadius));
            }
            if (north && south) {
                int a = firstLandStateDistance(
                    model, x, z, -stepX, 0, false, maxRadius
                );
                int b = firstLandStateDistance(
                    model, x, z, stepX, 0, false, maxRadius
                );
                best = Math.max(best, crossingNarrowness(a, b, maxRadius));
            }
            return best;
        }

        if (!west && !east) {
            int a = firstLandStateDistance(
                model, x, z, 0, -stepZ, true, maxRadius
            );
            int b = firstLandStateDistance(
                model, x, z, 0, stepZ, true, maxRadius
            );
            best = Math.max(best, crossingNarrowness(a, b, maxRadius));
        }
        if (!north && !south) {
            int a = firstLandStateDistance(
                model, x, z, -stepX, 0, true, maxRadius
            );
            int b = firstLandStateDistance(
                model, x, z, stepX, 0, true, maxRadius
            );
            best = Math.max(best, crossingNarrowness(a, b, maxRadius));
        }
        return best;
    }

    private static int firstLandStateDistance(
        GeoWorldV1 model,
        double x,
        double z,
        double dx,
        double dz,
        boolean wantLand,
        int maxRadius
    ) {
        for (int distance = 1; distance <= maxRadius; distance++) {
            GeoWorldV1.RuntimeSample sample =
                model.sample(x + dx * distance, z + dz * distance);
            if (sample.sourceLand() == wantLand) {
                return distance;
            }
        }
        return 0;
    }

    private static double crossingNarrowness(
        int a,
        int b,
        int maxRadius
    ) {
        if (a <= 0 || b <= 0 || maxRadius <= 1) {
            return 0;
        }
        double excess =
            (a + b - 2.0) / (2.0 * (maxRadius - 1.0));
        return clamp01(1.0 - 0.30 * excess);
    }

    private static double terrainWeight(String terrain, String... values) {
        for (int i = 0; i < values.length; i++) {
            if (values[i].equals(terrain)) {
                return switch (i) {
                    case 0 -> 1.0;
                    case 1 -> 0.78;
                    default -> 0.58;
                };
            }
        }
        return 0;
    }

    private static double smoothstep(double edge0, double edge1, double x) {
        if (edge1 <= edge0) {
            return x >= edge1 ? 1 : 0;
        }
        double t = clamp01((x - edge0) / (edge1 - edge0));
        return t * t * (3 - 2 * t);
    }

    private static double clamp01(double value) {
        if (!Double.isFinite(value) || value <= 0) {
            return 0;
        }
        return Math.min(1, value);
    }
}
