package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
import geoworld.runtime.GeoWorldV1;

final class GeoWorldShoreline {
    static final int MAX_SHORE_DISTANCE = 4;

    private static final Minecraft26_3HeightMapping HEIGHTS =
        new Minecraft26_3HeightMapping();

    record Column(
        GeoWorldV1.RuntimeSample sample,
        int originalGroundBoundary,
        int groundBoundary,
        int waterBoundary,
        int shoreDistance,
        String shoreWaterKind,
        int shoreWaterBoundary,
        boolean littoralShelf,
        String surfaceOverride,
        String underOverride
    ) {
        boolean hasShore() {
            return shoreDistance > 0 &&
                shoreDistance <= MAX_SHORE_DISTANCE;
        }
    }

    static final class ChunkPlan {
        private final Column[] columns = new Column[16 * 16];

        private ChunkPlan() {}

        Column column(int localX, int localZ) {
            if (localX < 0 || localX >= 16 || localZ < 0 || localZ >= 16) {
                throw new IllegalArgumentException(
                    "local column outside chunk: " + localX + "," + localZ
                );
            }
            return this.columns[localZ * 16 + localX];
        }
    }

    private interface Neighborhood {
        GeoWorldV1.RuntimeSample sample(int dx, int dz);
    }

    private record WaterNeighbor(
        int distance,
        int waterBoundary,
        String waterKind
    ) {}

    private GeoWorldShoreline() {}

    static ChunkPlan planChunk(
        GeoWorldV1 runtime,
        int baseX,
        int baseZ
    ) {
        final int radius = MAX_SHORE_DISTANCE;
        final int stride = 16 + radius * 2;
        GeoWorldV1.RuntimeSample[] samples =
            new GeoWorldV1.RuntimeSample[stride * stride];

        for (int z = 0; z < stride; z++) {
            int worldZ = baseZ + z - radius;
            for (int x = 0; x < stride; x++) {
                int worldX = baseX + x - radius;
                samples[z * stride + x] = runtime.sample(worldX, worldZ);
            }
        }

        ChunkPlan result = new ChunkPlan();
        long seed = runtime.manifest().seed();
        for (int localZ = 0; localZ < 16; localZ++) {
            for (int localX = 0; localX < 16; localX++) {
                final int centerX = localX + radius;
                final int centerZ = localZ + radius;
                int worldX = baseX + localX;
                int worldZ = baseZ + localZ;
                GeoWorldV1.RuntimeSample center =
                    samples[centerZ * stride + centerX];

                Neighborhood neighborhood = (dx, dz) ->
                    samples[
                        (centerZ + dz) * stride +
                        (centerX + dx)
                    ];

                result.columns[localZ * 16 + localX] = planColumn(
                    center,
                    neighborhood,
                    seed,
                    worldX,
                    worldZ
                );
            }
        }
        return result;
    }

    static Column plan(
        GeoWorldV1 runtime,
        int worldX,
        int worldZ
    ) {
        GeoWorldV1.RuntimeSample center = runtime.sample(worldX, worldZ);
        return planColumn(
            center,
            (dx, dz) -> runtime.sample(worldX + dx, worldZ + dz),
            runtime.manifest().seed(),
            worldX,
            worldZ
        );
    }

    static int waterBoundary(
        GeoWorldV1.RuntimeSample sample,
        int groundBoundary
    ) {
        if (
            !sample.waterSurfaceDefined() ||
            sample.waterSurfaceElevation() == null
        ) {
            return groundBoundary;
        }

        int mapped = "River".equals(sample.waterKind())
            ? HEIGHTS.boundaryYCeil(sample.waterSurfaceElevation())
            : HEIGHTS.boundaryY(sample.waterSurfaceElevation());
        if ("River".equals(sample.waterKind())) {
            mapped = Math.max(mapped, groundBoundary + 1);
        }
        return Math.max(groundBoundary, mapped);
    }

    private static Column planColumn(
        GeoWorldV1.RuntimeSample center,
        Neighborhood neighborhood,
        long seed,
        int worldX,
        int worldZ
    ) {
        int originalGround = HEIGHTS.boundaryY(center.groundHeight());
        int ownWater = waterBoundary(center, originalGround);

        if (isWater(center)) {
            return new Column(
                center,
                originalGround,
                originalGround,
                ownWater,
                0,
                center.waterKind(),
                ownWater,
                false,
                null,
                null
            );
        }

        WaterNeighbor shore = findNearestWater(neighborhood);
        if (
            isWetlandPool(
                center,
                neighborhood,
                shore,
                seed,
                worldX,
                worldZ,
                originalGround
            )
        ) {
            // Wetlands are not uniformly mud flats. Carve coherent one-block
            // shallow pools into already-flat wetland ground; macro Hydrology
            // remains responsible for true rivers/lakes.
            return new Column(
                center,
                originalGround,
                originalGround - 1,
                originalGround,
                Integer.MAX_VALUE,
                "WetlandPool",
                originalGround,
                false,
                "minecraft:mud",
                "minecraft:dirt"
            );
        }
        if (shore == null) {
            return new Column(
                center,
                originalGround,
                originalGround,
                ownWater,
                Integer.MAX_VALUE,
                "None",
                originalGround,
                false,
                null,
                null
            );
        }

        int adjustedGround = originalGround;
        boolean shallowFringe = false;
        int deltaToWater = originalGround - shore.waterBoundary();

        // A dry column immediately next to source water may never remain as an
        // empty slot below that source-water boundary. Repair at most two solid
        // blocks; if exactly one block of vertical gap remains, make it a
        // shallow source-water fringe instead of building a 3-block artificial
        // levee. A deeper remainder is left for the verifier to reject as a
        // real model problem.
        if (
            shore.distance() == 1 &&
            adjustedGround < shore.waterBoundary()
        ) {
            adjustedGround = Math.min(
                shore.waterBoundary(),
                adjustedGround + 2
            );
            int remainingGap =
                shore.waterBoundary() - adjustedGround;
            if (remainingGap == 1) {
                shallowFringe = true;
            }
        }

        double shapeNoise = smoothValueNoise(
            seed ^ 0x53484F52455F5348L,
            worldX,
            worldZ,
            28.0
        );
        boolean rocky = isRocky(center);
        boolean steep = deltaToWater >= 2;
        boolean hardBank =
            steep ||
            (deltaToWater >= 1 &&
             shapeNoise < (rocky ? 0.42 : 0.18));

        int shelfWidth = shelfWidth(shore.waterKind(), shapeNoise);
        boolean littoralShelf = false;
        if (
            !hardBank &&
            shore.distance() <= shelfWidth &&
            deltaToWater >= -1 &&
            deltaToWater <= 1
        ) {
            // Only ever lower a natural bank by one block. Columns that were
            // already below the water boundary may be raised farther because
            // that is the stability correction above, not aesthetic shaping.
            int shelfGround = Math.max(
                originalGround - 1,
                shore.waterBoundary()
            );
            adjustedGround = Math.max(
                shore.waterBoundary(),
                Math.min(adjustedGround, shelfGround)
            );
            littoralShelf = adjustedGround == shore.waterBoundary();
        }

        String surface = null;
        String under = null;
        int materialWidth = materialBandWidth(shore.waterKind());
        int verticalOffset = adjustedGround - shore.waterBoundary();
        if (
            shore.distance() <= materialWidth &&
            verticalOffset >= 0 &&
            verticalOffset <= 1 &&
            !steep
        ) {
            String material = chooseMaterial(
                center,
                shore.waterKind(),
                shore.distance(),
                seed,
                worldX,
                worldZ
            );
            if (material != null) {
                surface = material;
                under = underMaterial(material);
            }
        }

        return new Column(
            center,
            originalGround,
            adjustedGround,
            shallowFringe
                ? shore.waterBoundary()
                : adjustedGround,
            shore.distance(),
            shore.waterKind(),
            shore.waterBoundary(),
            littoralShelf,
            surface,
            under
        );
    }

    private static boolean isWetlandPool(
        GeoWorldV1.RuntimeSample center,
        Neighborhood neighborhood,
        WaterNeighbor shore,
        long seed,
        int worldX,
        int worldZ,
        int originalGround
    ) {
        boolean wetland =
            "Wetland".equals(center.biomeFamily()) ||
            "Wet Mud".equals(center.surfaceProfile()) ||
            "Peat".equals(center.surfaceProfile());
        if (!wetland || isRocky(center)) {
            return false;
        }
        if (shore != null && shore.distance() <= 2) {
            return false;
        }

        // Pools belong on flat/very gently undulating marsh ground. Avoid
        // punching water pockets into a two-block local slope.
        int maxDelta = 0;
        for (int[] offset : new int[][]{
            {1, 0}, {-1, 0}, {0, 1}, {0, -1},
            {2, 0}, {-2, 0}, {0, 2}, {0, -2}
        }) {
            int nearby = HEIGHTS.boundaryY(
                neighborhood.sample(
                    offset[0],
                    offset[1]
                ).groundHeight()
            );
            maxDelta = Math.max(
                maxDelta,
                Math.abs(nearby - originalGround)
            );
        }
        if (maxDelta > 1) {
            return false;
        }

        double broad = smoothValueNoise(
            seed ^ 0x5745544C414E4450L,
            worldX,
            worldZ,
            68.0
        );
        double detail = smoothValueNoise(
            seed ^ 0x5348414C4C4F5750L,
            worldX,
            worldZ,
            27.0
        );
        return broad > 0.57 && detail > 0.34;
    }

    private static WaterNeighbor findNearestWater(
        Neighborhood neighborhood
    ) {
        for (int distance = 1; distance <= MAX_SHORE_DISTANCE; distance++) {
            WaterNeighbor best = null;
            for (int dx = -distance; dx <= distance; dx++) {
                int dzAbs = distance - Math.abs(dx);
                best = betterWater(
                    best,
                    waterAt(neighborhood, dx, dzAbs, distance)
                );
                if (dzAbs != 0) {
                    best = betterWater(
                        best,
                        waterAt(neighborhood, dx, -dzAbs, distance)
                    );
                }
            }
            if (best != null) {
                return best;
            }
        }
        return null;
    }

    private static WaterNeighbor waterAt(
        Neighborhood neighborhood,
        int dx,
        int dz,
        int distance
    ) {
        GeoWorldV1.RuntimeSample sample = neighborhood.sample(dx, dz);
        if (!isWater(sample)) {
            return null;
        }
        int ground = HEIGHTS.boundaryY(sample.groundHeight());
        int water = waterBoundary(sample, ground);
        if (water <= ground) {
            return null;
        }
        return new WaterNeighbor(distance, water, sample.waterKind());
    }

    private static WaterNeighbor betterWater(
        WaterNeighbor current,
        WaterNeighbor candidate
    ) {
        if (candidate == null) {
            return current;
        }
        if (current == null) {
            return candidate;
        }
        if (candidate.waterBoundary() != current.waterBoundary()) {
            return candidate.waterBoundary() > current.waterBoundary()
                ? candidate
                : current;
        }
        return waterPriority(candidate.waterKind()) >
            waterPriority(current.waterKind())
            ? candidate
            : current;
    }

    private static int waterPriority(String kind) {
        return switch (kind) {
            case "River" -> 3;
            case "Lake" -> 2;
            case "Ocean" -> 1;
            default -> 0;
        };
    }

    private static boolean isWater(GeoWorldV1.RuntimeSample sample) {
        return !"None".equals(sample.waterKind()) &&
            sample.waterSurfaceDefined() &&
            sample.waterSurfaceElevation() != null;
    }

    private static int shelfWidth(String waterKind, double noise) {
        return switch (waterKind) {
            // Most low-gradient banks get a second dry shelf row toward land.
            // Water itself is never expanded; this only fills the visual
            // saw-tooth backward into the bank.
            case "River" -> noise >= 0.18 ? 2 : 1;
            case "Lake" -> noise >= 0.14 ? 2 : 1;
            case "Ocean" -> noise >= 0.30 ? 2 : 1;
            default -> 1;
        };
    }

    private static int materialBandWidth(String waterKind) {
        return switch (waterKind) {
            case "River" -> 2;
            case "Lake" -> 3;
            case "Ocean" -> 4;
            default -> 1;
        };
    }

    private static boolean isRocky(GeoWorldV1.RuntimeSample sample) {
        return "Mountain".equals(sample.terrainVariant()) ||
            "Canyon".equals(sample.terrainVariant()) ||
            "Alpine Rock".equals(sample.surfaceProfile()) ||
            "Desert Gravel".equals(sample.surfaceProfile());
    }

    private static String chooseMaterial(
        GeoWorldV1.RuntimeSample sample,
        String waterKind,
        int distance,
        long seed,
        int worldX,
        int worldZ
    ) {
        // Shore materials should read as short natural reaches, not
        // one-block salt-and-pepper noise. The broad field chooses the reach
        // material; the smaller field only bends its boundary.
        double patch =
            0.82 * smoothValueNoise(
                seed ^ 0x53484F52455F4D31L,
                worldX,
                worldZ,
                46.0
            ) +
            0.18 * smoothValueNoise(
                seed ^ 0x53484F52455F4D32L,
                worldX,
                worldZ,
                21.0
            );

        String family = sample.biomeFamily();
        String surface = sample.surfaceProfile();

        if (isRocky(sample)) {
            if (distance == 1) {
                return patch < 0.72
                    ? "minecraft:gravel"
                    : "minecraft:coarse_dirt";
            }
            return patch < 0.36
                ? "minecraft:gravel"
                : null;
        }

        if (
            "Wetland".equals(family) ||
            "Wet Mud".equals(surface) ||
            "Peat".equals(surface)
        ) {
            if (distance == 1) {
                return patch < 0.68
                    ? "minecraft:mud"
                    : "minecraft:dirt";
            }
            return patch < 0.52
                ? "minecraft:mud"
                : (patch < 0.78 ? "minecraft:dirt" : null);
        }

        if (
            "Temperate Desert".equals(family) ||
            "Hot Desert".equals(family) ||
            "Desert Sand".equals(surface)
        ) {
            if (distance <= 2) {
                return patch < 0.88
                    ? "minecraft:sand"
                    : "minecraft:gravel";
            }
            return patch < 0.55 ? "minecraft:sand" : null;
        }

        if ("Ocean".equals(waterKind)) {
            if (distance <= 2) {
                if (patch < 0.68) return "minecraft:sand";
                if (patch < 0.83) return "minecraft:gravel";
                if (patch < 0.91) return "minecraft:clay";
                return "minecraft:dirt";
            }
            if (distance == 3) {
                if (patch < 0.46) return "minecraft:sand";
                if (patch < 0.56) return "minecraft:gravel";
                return null;
            }
            return patch < 0.30 ? "minecraft:sand" : null;
        }

        if (distance == 1) {
            // 75%+ of ordinary immediate river/lake shore remains a substrate
            // accepted by vanilla SUPPORTS_SUGAR_CANE (dirt/mud/sand).
            if (patch < 0.10) return "minecraft:clay";
            if (patch < 0.24) return "minecraft:gravel";
            if (patch < 0.48) return "minecraft:mud";
            if (patch < 0.69) return "minecraft:sand";
            if (patch < 0.86) return "minecraft:coarse_dirt";
            return "minecraft:dirt";
        }

        if (distance == 2) {
            if (patch < 0.16) return "minecraft:gravel";
            if (patch < 0.34) return "minecraft:sand";
            if (patch < 0.50) return "minecraft:mud";
            if (patch < 0.70) return "minecraft:coarse_dirt";
            return null;
        }

        if ("Lake".equals(waterKind) && distance == 3) {
            if (patch < 0.18) return "minecraft:mud";
            if (patch < 0.30) return "minecraft:sand";
        }
        return null;
    }

    private static String underMaterial(String surface) {
        return switch (surface) {
            case "minecraft:sand" -> "minecraft:sand";
            case "minecraft:gravel" -> "minecraft:gravel";
            case "minecraft:clay" -> "minecraft:clay";
            case "minecraft:mud",
                 "minecraft:dirt",
                 "minecraft:coarse_dirt" -> "minecraft:dirt";
            default -> null;
        };
    }

    private static double smoothValueNoise(
        long seed,
        double x,
        double z,
        double scale
    ) {
        double gx = x / scale;
        double gz = z / scale;
        int x0 = (int)Math.floor(gx);
        int z0 = (int)Math.floor(gz);
        double tx = smoothstep(gx - x0);
        double tz = smoothstep(gz - z0);

        double v00 = hashUnit(seed, x0, z0);
        double v10 = hashUnit(seed, x0 + 1, z0);
        double v01 = hashUnit(seed, x0, z0 + 1);
        double v11 = hashUnit(seed, x0 + 1, z0 + 1);
        double a = v00 + (v10 - v00) * tx;
        double b = v01 + (v11 - v01) * tx;
        return a + (b - a) * tz;
    }

    private static double hashUnit(long seed, int x, int z) {
        long h = seed;
        h ^= (long)x * 0x9E3779B97F4A7C15L;
        h ^= (long)z * 0xC2B2AE3D27D4EB4FL;
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        h ^= h >>> 31;
        return (h >>> 11) * 0x1.0p-53;
    }

    private static double smoothstep(double t) {
        t = Math.max(0.0, Math.min(1.0, t));
        return t * t * (3.0 - 2.0 * t);
    }
}
