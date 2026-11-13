package geoworld.minecraft.common.worldgen;

import geoworld.runtime.GeoWorldV1;

import java.util.SplittableRandom;
import java.util.function.Consumer;

/**
 * Sparse, civilization-scale mineral districts for v1.1.15+ worlds.
 *
 * Districts are intentionally much larger and rarer than ordinary ore veins:
 * one successful 4096-block cell contains several long, related lodes with one
 * primary mineral family and a smaller companion family. The planner is pure
 * world-space geometry, so chunk request order cannot change the result.
 */
public final class GeoWorldMiningDistrictPlanner {
    private static final int CELL_XZ = 4096;
    private static final int SEARCH_MARGIN = 1700;
    private static final long DISTRICT_SALT = 0x4D494E4544495354L;

    private final HeightMapping heights;

    public GeoWorldMiningDistrictPlanner(HeightMapping heights) {
        this.heights = java.util.Objects.requireNonNull(heights, "heights");
    }

    public record District(
        long id,
        double centerX,
        double centerZ,
        double radius,
        GeoWorldMineralRules.Kind primary,
        GeoWorldMineralRules.Kind companion,
        int primaryLodes,
        int companionLodes
    ) {}

    public record Placement(
        int x,
        int y,
        int z,
        GeoWorldMineralRules.Kind mineral,
        long districtId,
        boolean companion
    ) {}

    public record Lode(
        long districtId,
        GeoWorldMineralRules.Kind mineral,
        boolean companion,
        double centerX,
        double centerY,
        double centerZ,
        double length,
        double radius,
        double angle,
        double tilt,
        double bend
    ) {}

    public void plan(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> consumer
    ) {
        planInternal(
            model, seed, minX, maxX, minZ, maxZ,
            consumer, null, null
        );
    }

    public void planLodes(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Lode> consumer
    ) {
        planInternal(
            model, seed, minX, maxX, minZ, maxZ,
            null, consumer, null
        );
    }

    public void planDistricts(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<District> consumer
    ) {
        planInternal(
            model, seed, minX, maxX, minZ, maxZ,
            null, null, consumer
        );
    }

    private void planInternal(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> placementConsumer,
        Consumer<Lode> lodeConsumer,
        Consumer<District> districtConsumer
    ) {
        if (!usesMiningDistricts(model)) return;

        int firstX = Math.floorDiv(minX - SEARCH_MARGIN, CELL_XZ);
        int lastX = Math.floorDiv(maxX + SEARCH_MARGIN, CELL_XZ);
        int firstZ = Math.floorDiv(minZ - SEARCH_MARGIN, CELL_XZ);
        int lastZ = Math.floorDiv(maxZ + SEARCH_MARGIN, CELL_XZ);

        for (int cellZ = firstZ; cellZ <= lastZ; cellZ++) {
            for (int cellX = firstX; cellX <= lastX; cellX++) {
                long id = cellSeed(seed ^ DISTRICT_SALT, cellX, 0, cellZ);
                SplittableRandom random = new SplittableRandom(id);

                // Roughly one district per two 4096x4096 candidate cells.
                if (random.nextDouble() > 0.54) continue;

                double centerX = (
                    cellX + 0.18 + random.nextDouble() * 0.64
                ) * CELL_XZ;
                double centerZ = (
                    cellZ + 0.18 + random.nextDouble() * 0.64
                ) * CELL_XZ;
                double radius = 720.0 + random.nextDouble() * 320.0;

                if (
                    centerX + radius < minX ||
                    centerX - radius > maxX ||
                    centerZ + radius < minZ ||
                    centerZ - radius > maxZ
                ) {
                    continue;
                }

                GeoWorldV1.RuntimeSample geology =
                    model.sample(centerX, centerZ);
                int groundY = heights.boundaryY(geology.groundHeight());
                int lowY = heights.minY() + 12;
                int highY = Math.min(
                    groundY - 16,
                    heights.maxYExclusive() - 12
                );
                if (highY <= lowY + 8) continue;

                int probeY =
                    lowY + random.nextInt(highY - lowY + 1);
                GeoWorldMineralRules.Kind primary =
                    chooseDistrictMineral(geology, probeY, random);
                GeoWorldMineralRules.Kind companion =
                    companionFor(primary);
                // A district is a broad mining region, not one giant ore blob.
                // Use many thinner related lodes spread across the district so
                // it can support long-term mining without exposing solid walls
                // of iron/coal.
                int primaryLodes = 36 + random.nextInt(17);
                int companionLodes = 10 + random.nextInt(7);

                if (districtConsumer != null) {
                    districtConsumer.accept(new District(
                        id,
                        centerX,
                        centerZ,
                        radius,
                        primary,
                        companion,
                        primaryLodes,
                        companionLodes
                    ));
                }

                // The district itself is the mineable body. Do not model it
                // as dozens of isolated ore veins embedded in mostly stone.
                // A broad connected mineralized mass is generated first; the
                // lodes below remain geological/detail metadata only.
                if (placementConsumer != null) {
                    // The district occupies the whole underground chunk
                    // column. Vertical ore placement is no longer centered
                    // around one preferred Y band.
                    emitDistrictMass(
                        id,
                        primary,
                        companion,
                        centerX,
                        centerZ,
                        radius,
                        minX,
                        maxX,
                        minZ,
                        maxZ,
                        placementConsumer
                    );
                }

                double strike = random.nextDouble() * Math.PI * 2.0;
                int total = primaryLodes + companionLodes;
                for (int i = 0; i < total; i++) {
                    boolean isCompanion = i >= primaryLodes;
                    GeoWorldMineralRules.Kind mineral =
                        isCompanion ? companion : primary;

                    double radial =
                        Math.sqrt(random.nextDouble()) * radius * 0.78;
                    double theta =
                        random.nextDouble() * Math.PI * 2.0;
                    double x =
                        centerX + Math.cos(theta) * radial;
                    double z =
                        centerZ + Math.sin(theta) * radial;

                    int preferredY = GeoWorldMineralRules.guaranteeY(
                        mineral,
                        random.nextBoolean()
                    );
                    double y = Math.max(
                        lowY,
                        Math.min(
                            highY,
                            preferredY + random.nextInt(-28, 29)
                        )
                    );

                    double baseLength = districtLength(
                        mineral,
                        isCompanion
                    );
                    double length =
                        baseLength *
                        (0.78 + random.nextDouble() * 0.52);
                    double lodeRadius =
                        districtRadius(mineral, isCompanion) *
                        (0.82 + random.nextDouble() * 0.42);
                    double angle =
                        strike +
                        (random.nextDouble() - 0.5) *
                        (isCompanion ? 1.20 : 0.62);
                    double tilt =
                        (random.nextDouble() - 0.5) * 0.16;
                    double bend =
                        (random.nextDouble() - 0.5) *
                        Math.min(14.0, length * 0.18);

                    emitLode(
                        mineral,
                        id,
                        isCompanion,
                        x,
                        y,
                        z,
                        length,
                        lodeRadius,
                        angle,
                        tilt,
                        bend,
                        minX,
                        maxX,
                        minZ,
                        maxZ,
                        lodeConsumer
                    );
                }
            }
        }
    }

    private static boolean usesMiningDistricts(GeoWorldV1 model) {
        return ("geoworld-rules-v1.1.15".equals(
            model.manifest().generationRulesVersion()
        ) || "geoworld-rules-v1.1.16".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.18".equals(
            model.manifest().generationRulesVersion()
        ) || "geoworld-rules-v1.1.19".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.20".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.21".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.22".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.23".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.24".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.25".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.26".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.27".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.28".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.29".equals(
            model.manifest().generationRulesVersion()
        ) || "geoworld-rules-v1.1.30".equals(
            model.manifest().generationRulesVersion()
        )))))))))))));
    }

    private static GeoWorldMineralRules.Kind chooseDistrictMineral(
        GeoWorldV1.RuntimeSample geology,
        double y,
        SplittableRandom random
    ) {
        for (int i = 0; i < 16; i++) {
            GeoWorldMineralRules.Kind mineral =
                GeoWorldMineralRules.chooseMineral(
                    geology.terrainVariant(),
                    geology.biomeFamily(),
                    y,
                    random.nextDouble()
                );
            if (mineral != null) return mineral;
        }
        return GeoWorldMineralRules.Kind.IRON;
    }

    private static GeoWorldMineralRules.Kind companionFor(
        GeoWorldMineralRules.Kind primary
    ) {
        return switch (primary) {
            case COAL -> GeoWorldMineralRules.Kind.IRON;
            case IRON -> GeoWorldMineralRules.Kind.COPPER;
            case COPPER -> GeoWorldMineralRules.Kind.GOLD;
            case GOLD -> GeoWorldMineralRules.Kind.COPPER;
            case LAPIS -> GeoWorldMineralRules.Kind.GOLD;
            case REDSTONE -> GeoWorldMineralRules.Kind.DIAMOND;
            case DIAMOND -> GeoWorldMineralRules.Kind.REDSTONE;
            case EMERALD -> GeoWorldMineralRules.Kind.GOLD;
        };
    }

    private static double districtLength(
        GeoWorldMineralRules.Kind mineral,
        boolean companion
    ) {
        double length = switch (mineral) {
            case COAL -> 46.0;
            case IRON -> 42.0;
            case COPPER -> 44.0;
            case GOLD, LAPIS, REDSTONE -> 34.0;
            case DIAMOND, EMERALD -> 28.0;
        };
        return companion ? length * 0.68 : length;
    }

    private static double districtRadius(
        GeoWorldMineralRules.Kind mineral,
        boolean companion
    ) {
        double radius = switch (mineral) {
            case COAL -> 4.4;
            case IRON -> 4.0;
            case COPPER -> 4.2;
            case GOLD, LAPIS, REDSTONE -> 3.2;
            case DIAMOND, EMERALD -> 2.5;
        };
        return companion ? radius * 0.68 : radius;
    }

    private void emitDistrictMass(
        long districtId,
        GeoWorldMineralRules.Kind primary,
        GeoWorldMineralRules.Kind companion,
        double centerX,
        double centerZ,
        double districtRadius,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> placementConsumer
    ) {
        SplittableRandom shape =
            new SplittableRandom(districtId ^ 0x4D415353424F4459L);
        double angle = shape.nextDouble() * Math.PI * 2.0;
        double cos = Math.cos(angle);
        double sin = Math.sin(angle);

        // A district spans many chunk columns horizontally. Its outline is
        // broad but intentionally irregular, not one clean ellipse.
        double majorRadius =
            districtRadius * (0.78 + shape.nextDouble() * 0.18);
        double minorRadius =
            districtRadius * (0.60 + shape.nextDouble() * 0.20);

        int left = Math.max(
            minX,
            (int)Math.floor(centerX - majorRadius - 96.0)
        );
        int right = Math.min(
            maxX,
            (int)Math.ceil(centerX + majorRadius + 96.0)
        );
        int north = Math.max(
            minZ,
            (int)Math.floor(centerZ - majorRadius - 96.0)
        );
        int south = Math.min(
            maxZ,
            (int)Math.ceil(centerZ + majorRadius + 96.0)
        );
        if (left > right || north > south) return;

        int minY = heights.minY() + 1;
        int maxY = heights.maxYExclusive() - 1;
        long oreSalt = districtId ^ 0x4F52454D41545258L;
        long companionSalt =
            districtId ^ 0x434F4D50414E494FL;
        long shapeSalt =
            districtId ^ 0x5348415045574152L;

        for (int bx = left; bx <= right; bx++) {
            for (int bz = north; bz <= south; bz++) {
                double localX = bx - centerX;
                double localZ = bz - centerZ;
                double rx = localX * cos + localZ * sin;
                double rz = -localX * sin + localZ * cos;
                double horizontal =
                    (rx * rx) / (majorRadius * majorRadius) +
                    (rz * rz) / (minorRadius * minorRadius);

                // Smooth low-frequency edge warping removes the former
                // 32-block stair-step / rectangular outline.
                double coarse =
                    smoothNoise2D(shapeSalt, bx, bz, 112);
                double detail =
                    smoothNoise2D(
                        shapeSalt ^ 0x4544474544455441L,
                        bx,
                        bz,
                        48
                    );
                double edgeScale =
                    0.80 + coarse * 0.30 + detail * 0.12;
                double edgeLimit = edgeScale * edgeScale;
                if (horizontal > edgeLimit) continue;

                double normalizedHorizontal =
                    Math.min(1.0, horizontal / edgeLimit);
                double baseOreFraction =
                    districtOreFractionTarget(normalizedHorizontal);

                // Once an X/Z column belongs to the mining district, the
                // mineralized body spans the complete vertical world column.
                // The Minecraft adapter will still preserve air, water,
                // soil and surface blocks by replacing only stone/deepslate
                // below the safe ground boundary.
                for (int by = minY; by <= maxY; by++) {
                    // Continuous multi-scale 3D fields replace the old
                    // axis-aligned 3x3x3 / 9x7x9 cells. Waste rock therefore
                    // forms curved lenses, seams and pockets instead of cubes.
                    double coarseOre = smoothNoise3D(
                        oreSalt ^ 0x434F415253454F52L,
                        bx,
                        by * 0.72,
                        bz,
                        26.0
                    );
                    double mediumOre = smoothNoise3D(
                        oreSalt ^ 0x4D454449554D4F52L,
                        bx,
                        by * 0.84,
                        bz,
                        11.0
                    );
                    double fineOre = smoothNoise3D(
                        oreSalt ^ 0x46494E454F524546L,
                        bx,
                        by,
                        bz,
                        5.0
                    );
                    double oreField =
                        coarseOre * 0.50 +
                        mediumOre * 0.34 +
                        fineOre * 0.12 +
                        unitHash(cellSeed(
                            oreSalt ^ 0x475241494E4A4954L,
                            bx,
                            by,
                            bz
                        )) * 0.04;

                    double verticalVariation =
                        (
                            smoothNoise3D(
                                oreSalt ^ 0x5645525447524144L,
                                bx,
                                by,
                                bz,
                                37.0
                            ) - 0.5
                        ) * 0.08;
                    double oreFraction = Math.max(
                        0.40,
                        Math.min(
                            0.70,
                            baseOreFraction + verticalVariation
                        )
                    );
                    if (oreField > oreFieldThreshold(oreFraction)) {
                        // Host rock remains as irregular waste-rock lenses
                        // inside the ore-dominant district column.
                        continue;
                    }

                    // Companion ore uses an independent warped field. Its
                    // pockets are coherent and curved, not grid-aligned.
                    double companionCoarse = smoothNoise3D(
                        companionSalt ^ 0x434F4D50434F4152L,
                        bx + smoothNoise3D(
                            companionSalt ^ 0x5741525058L,
                            bx, by, bz, 43.0
                        ) * 17.0,
                        by * 0.78,
                        bz + smoothNoise3D(
                            companionSalt ^ 0x574152505AL,
                            bx, by, bz, 47.0
                        ) * 17.0,
                        19.0
                    );
                    double companionFine = smoothNoise3D(
                        companionSalt ^ 0x434F4D5046494E45L,
                        bx,
                        by,
                        bz,
                        8.0
                    );
                    double companionField =
                        companionCoarse * 0.72 +
                        companionFine * 0.28;
                    double companionShare =
                        0.12 +
                        smoothNoise3D(
                            companionSalt ^ 0x5348415245L,
                            bx,
                            by,
                            bz,
                            61.0
                        ) * 0.10;
                    boolean isCompanion =
                        companionField <
                        companionFieldThreshold(companionShare);

                    placementConsumer.accept(
                        new Placement(
                            bx,
                            by,
                            bz,
                            isCompanion ? companion : primary,
                            districtId,
                            isCompanion
                        )
                    );
                }
            }
        }
    }

    private static double smoothNoise3D(
        long seed,
        double x,
        double y,
        double z,
        double scale
    ) {
        double gx = x / scale;
        double gy = y / scale;
        double gz = z / scale;
        int ix = (int)Math.floor(gx);
        int iy = (int)Math.floor(gy);
        int iz = (int)Math.floor(gz);
        double tx = smoothStep(gx - ix);
        double ty = smoothStep(gy - iy);
        double tz = smoothStep(gz - iz);

        double n000 = unitHash(cellSeed(seed, ix, iy, iz));
        double n100 = unitHash(cellSeed(seed, ix + 1, iy, iz));
        double n010 = unitHash(cellSeed(seed, ix, iy + 1, iz));
        double n110 = unitHash(cellSeed(seed, ix + 1, iy + 1, iz));
        double n001 = unitHash(cellSeed(seed, ix, iy, iz + 1));
        double n101 = unitHash(cellSeed(seed, ix + 1, iy, iz + 1));
        double n011 = unitHash(cellSeed(seed, ix, iy + 1, iz + 1));
        double n111 = unitHash(cellSeed(seed, ix + 1, iy + 1, iz + 1));

        double nx00 = n000 + (n100 - n000) * tx;
        double nx10 = n010 + (n110 - n010) * tx;
        double nx01 = n001 + (n101 - n001) * tx;
        double nx11 = n011 + (n111 - n011) * tx;
        double nxy0 = nx00 + (nx10 - nx00) * ty;
        double nxy1 = nx01 + (nx11 - nx01) * ty;
        return nxy0 + (nxy1 - nxy0) * tz;
    }

    private static double oreFieldThreshold(double oreFraction) {
        double fraction = Math.max(0.40, Math.min(0.70, oreFraction));
        // Multi-octave interpolated noise is concentrated around 0.5, so
        // its threshold cannot equal the desired volume fraction directly.
        // This calibrated range maps district targets back to ~40-70%
        // realized ore volume instead of the former ~87% center density.
        return 0.46 + (fraction - 0.40) * (0.07 / 0.30);
    }

    private static double companionFieldThreshold(double share) {
        double normalized = Math.max(0.12, Math.min(0.22, share));
        return 0.34 + (normalized - 0.12) * 1.00;
    }

    private static double smoothNoise2D(
        long seed,
        double x,
        double z,
        double scale
    ) {
        double gx = x / scale;
        double gz = z / scale;
        int ix = (int)Math.floor(gx);
        int iz = (int)Math.floor(gz);
        double tx = smoothStep(gx - ix);
        double tz = smoothStep(gz - iz);

        double n00 = unitHash(cellSeed(seed, ix, 0, iz));
        double n10 = unitHash(cellSeed(seed, ix + 1, 0, iz));
        double n01 = unitHash(cellSeed(seed, ix, 0, iz + 1));
        double n11 = unitHash(cellSeed(seed, ix + 1, 0, iz + 1));
        double nx0 = n00 + (n10 - n00) * tx;
        double nx1 = n01 + (n11 - n01) * tx;
        return nx0 + (nx1 - nx0) * tz;
    }

    private static double smoothStep(double value) {
        return value * value * (3.0 - 2.0 * value);
    }

    public static double districtOreFractionTarget(
        double normalizedDistanceSquared
    ) {
        double distance = Math.max(
            0.0,
            Math.min(1.0, normalizedDistanceSquared)
        );
        double coreFactor =
            Math.max(0.0, 1.0 - Math.sqrt(distance));
        return 0.40 + coreFactor * 0.30;
    }

    private void emitLode(
        GeoWorldMineralRules.Kind mineral,
        long districtId,
        boolean companion,
        double x,
        double y,
        double z,
        double length,
        double radius,
        double angle,
        double tilt,
        double bend,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Lode> lodeConsumer
    ) {
        double reach =
            length * 0.5 + Math.abs(bend) + radius * 1.5;
        if (
            x + reach < minX ||
            x - reach > maxX ||
            z + reach < minZ ||
            z - reach > maxZ
        ) {
            return;
        }

        if (lodeConsumer != null) {
            lodeConsumer.accept(new Lode(
                districtId,
                mineral,
                companion,
                x,
                y,
                z,
                length,
                radius,
                angle,
                tilt,
                bend
            ));
        }
    }

    private static double unitHash(long value) {
        return (double)(value >>> 11) * 0x1.0p-53;
    }

    private static long cellSeed(
        long seed,
        int x,
        int y,
        int z
    ) {
        long h = seed ^ 0x9E3779B97F4A7C15L;
        h ^= (long)x * 0xBF58476D1CE4E5B9L;
        h ^= (long)y * 0x94D049BB133111EBL;
        h ^= (long)z * 0xD6E8FEB86659FD93L;
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        return h ^ (h >>> 31);
    }
}
