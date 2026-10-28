package geoworld.minecraft.common.worldgen;

import geoworld.runtime.GeoWorldV1;
import geoworld.runtime.GeoWorldGeology;

import java.util.SplittableRandom;
import java.util.function.Consumer;

/**
 * Version-neutral deterministic GeoWorld mineral geometry.
 *
 * This class decides where connected mineral bodies exist. Version modules
 * only decide which block state represents one mineral kind and whether a
 * candidate voxel is a replaceable host block.
 */
public final class GeoWorldMineralPlanner {
    private static final int CELL_XZ = 64;
    private static final int CELL_Y = 48;
    private static final int SEARCH_MARGIN = 56;
    private static final int SCATTER_CELL_XZ = 32;
    private static final int SCATTER_CELL_Y = 32;
    private static final int SCATTER_SEARCH_MARGIN = 20;
    private static final long OUTCROP_SALT = 0x4F555443524F5053L;
    private static final long SCATTER_SALT = 0x5343415454455231L;

    private final HeightMapping heights;

    public GeoWorldMineralPlanner(HeightMapping heights) {
        this.heights = java.util.Objects.requireNonNull(heights, "heights");
    }

    public record Placement(
        int x,
        int y,
        int z,
        GeoWorldMineralRules.Kind mineral
    ) {}

    public enum BodySource {
        VEIN,
        GUARANTEE,
        SCATTER,
        OUTCROP
    }

    /**
     * One deterministic mineral body before Minecraft host-block filtering.
     * The game and the Web preview consume the same body geometry.
     */
    public record Body(
        GeoWorldMineralRules.Kind mineral,
        BodySource source,
        double centerX,
        double centerY,
        double centerZ,
        double length,
        double radius,
        double angle,
        double tilt,
        double bend,
        boolean large,
        boolean rockyOutcrop
    ) {}

    private static boolean usesV117Minerals(GeoWorldV1 model) {
        String rules = model.manifest().generationRulesVersion();
        return "geoworld-rules-v1.1.7".equals(rules) ||
            "geoworld-rules-v1.1.8".equals(rules) ||
            "geoworld-rules-v1.1.9".equals(rules) ||
            "geoworld-rules-v1.1.10".equals(rules) ||
            "geoworld-rules-v1.1.11".equals(rules);
    }

    public void planSurfaceOutcrops(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> consumer
    ) {
        planSurfaceOutcropsInternal(
            model, seed, minX, maxX, minZ, maxZ, consumer, null
        );
    }

    public void planSurfaceOutcropBodies(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Body> consumer
    ) {
        planSurfaceOutcropsInternal(
            model, seed, minX, maxX, minZ, maxZ, null, consumer
        );
    }

    private void planSurfaceOutcropsInternal(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> placementConsumer,
        Consumer<Body> bodyConsumer
    ) {
        boolean modernMinerals = usesV117Minerals(model);
        for (
            int cellZ = Math.floorDiv(minZ - SEARCH_MARGIN, CELL_XZ);
            cellZ <= Math.floorDiv(maxZ + SEARCH_MARGIN, CELL_XZ);
            cellZ++
        ) {
            for (
                int cellX = Math.floorDiv(minX - SEARCH_MARGIN, CELL_XZ);
                cellX <= Math.floorDiv(maxX + SEARCH_MARGIN, CELL_XZ);
                cellX++
            ) {
                SplittableRandom random = new SplittableRandom(
                    cellSeed(seed ^ OUTCROP_SALT, cellX, 0, cellZ)
                );
                // Keep rocky outcrops sparse, but not so sparse that exposed
                // stone mountains lose coherent visible lodes. Mineral kind
                // selection and body geometry still control rarity/coverage.
                if (random.nextInt(6) != 0) continue;

                double centerX =
                    cellX * (double) CELL_XZ + CELL_XZ / 2.0 +
                    (random.nextDouble() - 0.5) * 24.0;
                double centerZ =
                    cellZ * (double) CELL_XZ + CELL_XZ / 2.0 +
                    (random.nextDouble() - 0.5) * 24.0;
                var geology = model.sample(centerX, centerZ);
                if (
                    !geology.sourceLand() ||
                    !"None".equals(geology.waterKind()) ||
                    (!modernMinerals &&
                     !GeoWorldMineralRules.isMountainTerrain(
                         geology.terrainVariant()
                     ))
                ) {
                    continue;
                }

                int surface = heights.boundaryY(geology.groundHeight()) - 1;
                int minimumSurface = modernMinerals
                    ? heights.minY() + 4
                    : 85;
                if (
                    surface < minimumSurface ||
                    surface >= heights.maxYExclusive() - 1
                ) {
                    continue;
                }

                GeoWorldMineralRules.Kind mineral = modernMinerals
                    ? chooseSurfaceOutcropMineral(
                        geology.terrainVariant(),
                        geology.biomeFamily(),
                        surface,
                        random
                    )
                    : surface > 215
                        ? (
                            random.nextBoolean()
                                ? GeoWorldMineralRules.Kind.IRON
                                : GeoWorldMineralRules.Kind.EMERALD
                        )
                        : surface > 150
                            ? switch (random.nextInt(3)) {
                                case 0 -> GeoWorldMineralRules.Kind.COAL;
                                case 1 -> GeoWorldMineralRules.Kind.IRON;
                                default -> GeoWorldMineralRules.Kind.EMERALD;
                            }
                            : switch (random.nextInt(4)) {
                                case 0 -> GeoWorldMineralRules.Kind.COAL;
                                case 1 -> GeoWorldMineralRules.Kind.IRON;
                                case 2 -> GeoWorldMineralRules.Kind.COPPER;
                                default -> GeoWorldMineralRules.Kind.EMERALD;
                            };
                if (mineral == null) continue;

                double regionalization =
                    model.worldProfile().parameters().mineralRegionalization();
                double provinceStrength =
                    GeoWorldGeology.provinceStrength(model, centerX, centerZ);
                emitVein(
                    mineral,
                    centerX,
                    surface - 2.5,
                    centerZ,
                    random,
                    minX,
                    maxX,
                    minZ,
                    maxZ,
                    placementConsumer,
                    true,
                    regionalization,
                    provinceStrength,
                    modernMinerals,
                    BodySource.OUTCROP,
                    bodyConsumer
                );
            }
        }
    }

    private static GeoWorldMineralRules.Kind chooseSurfaceOutcropMineral(
        String terrain,
        String biome,
        double y,
        SplittableRandom random
    ) {
        for (int attempt = 0; attempt < 8; attempt++) {
            GeoWorldMineralRules.Kind selected =
                GeoWorldMineralRules.chooseMineral(
                    terrain, biome, y, random.nextDouble()
                );
            if (
                selected == GeoWorldMineralRules.Kind.COAL ||
                selected == GeoWorldMineralRules.Kind.IRON ||
                selected == GeoWorldMineralRules.Kind.COPPER ||
                selected == GeoWorldMineralRules.Kind.GOLD ||
                selected == GeoWorldMineralRules.Kind.EMERALD
            ) {
                return selected;
            }
        }
        if (
            GeoWorldMineralRules.affinity(
                GeoWorldMineralRules.Kind.IRON, terrain, biome, y
            ) > 0
        ) {
            return GeoWorldMineralRules.Kind.IRON;
        }
        if (
            GeoWorldMineralRules.affinity(
                GeoWorldMineralRules.Kind.COAL, terrain, biome, y
            ) > 0
        ) {
            return GeoWorldMineralRules.Kind.COAL;
        }
        return null;
    }

    public void planScatter(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> consumer
    ) {
        planScatterInternal(model, seed, minX, maxX, minZ, maxZ, consumer, null);
    }

    public void planScatterBodies(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Body> consumer
    ) {
        planScatterInternal(model, seed, minX, maxX, minZ, maxZ, null, consumer);
    }

    private void planScatterInternal(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> placementConsumer,
        Consumer<Body> bodyConsumer
    ) {
        if (!usesV117Minerals(model)) return;

        int firstX = Math.floorDiv(
            minX - SCATTER_SEARCH_MARGIN,
            SCATTER_CELL_XZ
        );
        int lastX = Math.floorDiv(
            maxX + SCATTER_SEARCH_MARGIN,
            SCATTER_CELL_XZ
        );
        int firstZ = Math.floorDiv(
            minZ - SCATTER_SEARCH_MARGIN,
            SCATTER_CELL_XZ
        );
        int lastZ = Math.floorDiv(
            maxZ + SCATTER_SEARCH_MARGIN,
            SCATTER_CELL_XZ
        );
        int minY = heights.minY() + 1;
        int maxY = heights.maxYExclusive() - 1;

        for (int cellZ = firstZ; cellZ <= lastZ; cellZ++) {
            for (int cellX = firstX; cellX <= lastX; cellX++) {
                double geologyX =
                    (cellX + 0.5) * SCATTER_CELL_XZ;
                double geologyZ =
                    (cellZ + 0.5) * SCATTER_CELL_XZ;
                GeoWorldV1.RuntimeSample geology =
                    model.sample(geologyX, geologyZ);

                for (
                    int cellY = Math.floorDiv(minY, SCATTER_CELL_Y);
                    cellY <= Math.floorDiv(maxY, SCATTER_CELL_Y);
                    cellY++
                ) {
                    SplittableRandom random = new SplittableRandom(
                        cellSeed(
                            seed ^ SCATTER_SALT,
                            cellX,
                            cellY,
                            cellZ
                        )
                    );
                    for (int attempt = 0; attempt < 2; attempt++) {
                        double chance = attempt == 0 ? 0.82 : 0.34;
                        if (random.nextDouble() > chance) continue;

                        double x =
                            (cellX + 0.18 + random.nextDouble() * 0.64) *
                            SCATTER_CELL_XZ;
                        double z =
                            (cellZ + 0.18 + random.nextDouble() * 0.64) *
                            SCATTER_CELL_XZ;
                        double y =
                            (cellY + 0.16 + random.nextDouble() * 0.68) *
                            SCATTER_CELL_Y;
                        GeoWorldMineralRules.Kind mineral =
                            GeoWorldMineralRules.chooseMineral(
                                geology.terrainVariant(),
                                geology.biomeFamily(),
                                y,
                                random.nextDouble()
                            );
                        if (mineral == null) continue;
                        emitScatterVein(
                            mineral,
                            x,
                            y,
                            z,
                            random,
                            minX,
                            maxX,
                            minZ,
                            maxZ,
                            placementConsumer,
                            bodyConsumer
                        );
                    }
                }
            }
        }
    }

    public void plan(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> consumer
    ) {
        planInternal(model, seed, minX, maxX, minZ, maxZ, consumer, null);
    }

    public void planBodies(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Body> consumer
    ) {
        planInternal(model, seed, minX, maxX, minZ, maxZ, null, consumer);
    }

    private void planInternal(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> placementConsumer,
        Consumer<Body> bodyConsumer
    ) {
        boolean modernMinerals = usesV117Minerals(model);
        int firstX = Math.floorDiv(minX - SEARCH_MARGIN, CELL_XZ);
        int lastX = Math.floorDiv(maxX + SEARCH_MARGIN, CELL_XZ);
        int firstZ = Math.floorDiv(minZ - SEARCH_MARGIN, CELL_XZ);
        int lastZ = Math.floorDiv(maxZ + SEARCH_MARGIN, CELL_XZ);

        for (int cellZ = firstZ; cellZ <= lastZ; cellZ++) {
            for (int cellX = firstX; cellX <= lastX; cellX++) {
                double geologyX = (double) cellX * CELL_XZ + CELL_XZ / 2.0;
                double geologyZ = (double) cellZ * CELL_XZ + CELL_XZ / 2.0;
                GeoWorldV1.RuntimeSample geology = model.sample(
                    geologyX,
                    geologyZ
                );
                double regionalization =
                    model.worldProfile().parameters().mineralRegionalization();
                double provinceStrength =
                    GeoWorldGeology.provinceStrength(model, geologyX, geologyZ);

                for (GeoWorldMineralRules.Kind guarantee :
                    GeoWorldMineralRules.Kind.values()) {
                    int slot = guarantee.ordinal();
                    int targetX = 2 + (slot % 4) * 4;
                    int targetZ = 4 + (slot / 4) * 8;
                    if (
                        Math.floorMod(cellX, 16) == targetX &&
                        Math.floorMod(cellZ, 16) == targetZ
                    ) {
                        for (int guaranteedY : new int[]{
                            GeoWorldMineralRules.guaranteeY(
                                guarantee, false
                            ),
                            GeoWorldMineralRules.guaranteeY(
                                guarantee, true
                            )
                        }) {
                            emitVein(
                                guarantee,
                                cellX * (double) CELL_XZ + 32,
                                guaranteedY,
                                cellZ * (double) CELL_XZ + 32,
                                new SplittableRandom(
                                    cellSeed(
                                        seed,
                                        cellX,
                                        guaranteedY,
                                        cellZ
                                    ) ^ 0x4D494E4552414C31L
                                ),
                                minX,
                                maxX,
                                minZ,
                                maxZ,
                                placementConsumer,
                                false,
                                regionalization,
                                provinceStrength,
                                modernMinerals,
                                BodySource.GUARANTEE,
                                bodyConsumer
                            );
                        }
                    }
                }

                int minY = heights.minY() + 1;
                int maxY = heights.maxYExclusive() - 1;
                for (
                    int cellY = Math.floorDiv(minY, CELL_Y);
                    cellY <= Math.floorDiv(maxY, CELL_Y);
                    cellY++
                ) {
                    SplittableRandom random = new SplittableRandom(
                        cellSeed(seed, cellX, cellY, cellZ)
                    );
                    for (int attempt = 0; attempt < 2; attempt++) {
                        double chance = attempt == 0 ? 0.88 : 0.43;
                        if (Math.abs(regionalization - 1.0) > 1.0e-12) {
                            double provinceBias = 0.70 + 0.60 * provinceStrength;
                            chance *= 1.0 +
                                (regionalization - 1.0) * (provinceBias - 1.0);
                            chance = Math.max(0.08, Math.min(0.98, chance));
                        }
                        if (random.nextDouble() > chance) {
                            continue;
                        }
                        double x = (
                            cellX + 0.28 + random.nextDouble() * 0.44
                        ) * CELL_XZ;
                        double z = (
                            cellZ + 0.28 + random.nextDouble() * 0.44
                        ) * CELL_XZ;
                        double y = (
                            cellY + 0.22 + random.nextDouble() * 0.56
                        ) * CELL_Y;

                        GeoWorldMineralRules.Kind mineral =
                            GeoWorldMineralRules.chooseMineral(
                                geology.terrainVariant(),
                                geology.biomeFamily(),
                                y,
                                random.nextDouble()
                            );
                        if (mineral == null) continue;

                        emitVein(
                            mineral,
                            x,
                            y,
                            z,
                            random,
                            minX,
                            maxX,
                            minZ,
                            maxZ,
                            placementConsumer,
                            false,
                            regionalization,
                            provinceStrength,
                            modernMinerals,
                            BodySource.VEIN,
                            bodyConsumer
                        );
                    }
                }
            }
        }
    }

    private void emitScatterVein(
        GeoWorldMineralRules.Kind mineral,
        double x,
        double y,
        double z,
        SplittableRandom random,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> placementConsumer,
        Consumer<Body> bodyConsumer
    ) {
        double baseLength = switch (mineral) {
            case COAL -> 9.0;
            case IRON, COPPER -> 8.0;
            case GOLD, REDSTONE -> 6.5;
            case LAPIS -> 5.5;
            case DIAMOND, EMERALD -> 5.0;
        };
        double baseRadius = switch (mineral) {
            case COAL, COPPER -> 2.2;
            case IRON -> 2.0;
            case GOLD, REDSTONE -> 1.7;
            case LAPIS -> 1.6;
            case DIAMOND, EMERALD -> 1.45;
        };
        double length = baseLength * (0.78 + random.nextDouble() * 0.44);
        double radius = baseRadius * (0.82 + random.nextDouble() * 0.36);
        double angle = random.nextDouble() * Math.PI * 2;
        double dx = Math.cos(angle);
        double dz = Math.sin(angle);
        double tilt = (random.nextDouble() - 0.5) * 0.18;
        if (bodyConsumer != null) {
            bodyConsumer.accept(new Body(
                mineral, BodySource.SCATTER,
                x, y, z, length, radius, angle, tilt, 0.0,
                false, false
            ));
        }
        int steps = Math.max(3, (int)Math.ceil(length / 2.2));
        int minY = heights.minY() + 1;
        int maxY = heights.maxYExclusive() - 1;

        for (int step = 0; step <= steps; step++) {
            double t = (double)step / steps;
            double along = (t - 0.5) * length;
            double cx = x + along * dx;
            double cz = z + along * dz;
            double cy = y + along * tilt;
            double localRadius = radius * (
                0.78 +
                0.18 * Math.sin(t * Math.PI) +
                0.10 * (random.nextDouble() - 0.5)
            );
            double verticalRadius = localRadius * 0.82;
            if (placementConsumer == null) continue;

            int left = Math.max(minX, (int)Math.floor(cx - localRadius));
            int right = Math.min(maxX, (int)Math.ceil(cx + localRadius));
            int north = Math.max(minZ, (int)Math.floor(cz - localRadius));
            int south = Math.min(maxZ, (int)Math.ceil(cz + localRadius));
            int low = Math.max(minY, (int)Math.floor(cy - verticalRadius));
            int high = Math.min(maxY, (int)Math.ceil(cy + verticalRadius));
            for (int bx = left; bx <= right; bx++) {
                for (int bz = north; bz <= south; bz++) {
                    for (int by = low; by <= high; by++) {
                        double hx = (bx - cx) / localRadius;
                        double hz = (bz - cz) / localRadius;
                        double vy = (by - cy) / verticalRadius;
                        if (hx * hx + hz * hz + vy * vy <= 1.0) {
                            placementConsumer.accept(
                                new Placement(bx, by, bz, mineral)
                            );
                        }
                    }
                }
            }
        }
    }

    public void planReferenceBody(
        GeoWorldMineralRules.Kind mineral,
        boolean exposed,
        long seed,
        Consumer<Placement> consumer
    ) {
        emitVein(
            mineral,
            0,
            100,
            0,
            new SplittableRandom(seed),
            -70,
            70,
            -70,
            70,
            consumer,
            exposed,
            1.0,
            0.5,
            true,
            BodySource.VEIN,
            null
        );
    }

    private void emitVein(
        GeoWorldMineralRules.Kind mineral,
        double x,
        double y,
        double z,
        SplittableRandom random,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> placementConsumer,
        boolean rockyOutcrop,
        double regionalization,
        double provinceStrength,
        boolean modernMinerals,
        BodySource source,
        Consumer<Body> bodyConsumer
    ) {
        double largeChance = modernMinerals
            ? switch (mineral) {
                case COAL, IRON -> 0.028;
                case COPPER -> 0.024;
                case GOLD, LAPIS, REDSTONE -> 0.017;
                case DIAMOND, EMERALD -> 0.011;
            }
            : 0.035;
        double bodyScale = 1.0;
        if (Math.abs(regionalization - 1.0) > 1.0e-12) {
            double provinceBias = 0.65 + 0.70 * provinceStrength;
            largeChance = Math.max(
                0.008,
                Math.min(
                    0.12,
                    largeChance * (
                        1.0 +
                        (regionalization - 1.0) * (provinceBias - 0.82)
                    )
                )
            );
            bodyScale = Math.max(
                0.72,
                Math.min(
                    1.45,
                    1.0 +
                    (regionalization - 1.0) *
                    (0.10 + 0.22 * provinceStrength)
                )
            );
        }
        boolean large = random.nextDouble() < largeChance;
        double majorLengthScale = !large
            ? 1.0
            : modernMinerals
                ? switch (mineral) {
                    case COAL -> 3.6;
                    case IRON -> 3.8;
                    case COPPER -> 3.6;
                    case GOLD, LAPIS, REDSTONE -> 3.0;
                    case DIAMOND, EMERALD -> 2.8;
                }
                : 1.22;
        double majorRadiusScale = !large
            ? 1.0
            : modernMinerals
                ? switch (mineral) {
                    case COAL -> 2.5;
                    case IRON -> 2.7;
                    case COPPER -> 2.6;
                    case GOLD, LAPIS, REDSTONE -> 2.3;
                    case DIAMOND, EMERALD -> 2.2;
                }
                : 1.17;
        double normalBodyScale = modernMinerals ? 1.18 : 1.0;
        double length =
            mineral.seamLength() *
            (0.90 + random.nextDouble() * 0.22) *
            normalBodyScale *
            majorLengthScale *
            bodyScale;
        double radius =
            mineral.seamRadius() *
            (0.94 + random.nextDouble() * 0.15) *
            (modernMinerals ? 1.12 : 1.0) *
            majorRadiusScale *
            (rockyOutcrop ? 1.12 : 1.0) *
            Math.sqrt(bodyScale);

        long bodySalt = random.nextLong();
        double angle = random.nextDouble() * Math.PI * 2;
        double dx = Math.cos(angle);
        double dz = Math.sin(angle);
        double tilt = (random.nextDouble() - 0.5) * 0.27;
        double bend =
            (random.nextDouble() - 0.5) * radius * 1.15;

        double reach =
            length * 0.5 +
            Math.abs(bend) +
            radius * 1.3 +
            2;
        if (
            x + reach < minX ||
            x - reach > maxX ||
            z + reach < minZ ||
            z - reach > maxZ
        ) {
            return;
        }

        if (bodyConsumer != null) {
            bodyConsumer.accept(new Body(
                mineral, source, x, y, z, length, radius,
                angle, tilt, bend, large, rockyOutcrop
            ));
        }

        int steps = Math.max(4, (int) Math.ceil(length / 3.7));
        int minY = heights.minY() + 1;
        int maxY = heights.maxYExclusive() - 1;

        for (int step = 0; step <= steps; step++) {
            double t = (double) step / steps;
            double longitudinal = (t - 0.5) * length * 0.68;
            double lateral = Math.sin(t * Math.PI) * bend;
            double cx = x + longitudinal * dx - lateral * dz;
            double cz = z + longitudinal * dz + lateral * dx;
            double cy = y + longitudinal * tilt;
            double lobe =
                0.76 +
                0.19 * Math.sin(t * Math.PI) +
                0.14 * (random.nextDouble() - 0.5);
            double r = radius * lobe;
            double verticalRadius =
                rockyOutcrop ? r * 1.70 : r * 1.10;
            if (placementConsumer == null) continue;

            int left = Math.max(minX, (int) Math.floor(cx - r));
            int right = Math.min(maxX, (int) Math.ceil(cx + r));
            int north = Math.max(minZ, (int) Math.floor(cz - r));
            int south = Math.min(maxZ, (int) Math.ceil(cz + r));
            int low = Math.max(
                minY,
                (int) Math.floor(cy - verticalRadius)
            );
            int high = Math.min(
                maxY,
                (int) Math.ceil(cy + verticalRadius)
            );

            if (
                left > right ||
                north > south ||
                low > high
            ) {
                continue;
            }

            for (int bx = left; bx <= right; bx++) {
                for (int bz = north; bz <= south; bz++) {
                    for (int by = low; by <= high; by++) {
                        double horizontal2 =
                            (bx - cx) * (bx - cx) +
                            (bz - cz) * (bz - cz);
                        double vertical =
                            (by - cy) / verticalRadius;
                        double bodyDistance =
                            horizontal2 / (r * r) +
                            vertical * vertical;
                        if (
                            bodyDistance <= 1.0 &&
                            (
                                !modernMinerals ||
                                // Preserve an overlapping solid core so the
                                // vein remains one granite-style body. The
                                // outer shell still uses the mineralization
                                // mask for an irregular natural boundary.
                                bodyDistance <= 0.48 ||
                                mineralizedBodyVoxel(
                                    mineral,
                                    bodySalt,
                                    bx,
                                    by,
                                    bz
                                )
                            )
                        ) {
                            placementConsumer.accept(
                                new Placement(
                                    bx,
                                    by,
                                    bz,
                                    mineral
                                )
                            );
                        }
                    }
                }
            }
        }
    }

    private static boolean mineralizedBodyVoxel(
        GeoWorldMineralRules.Kind mineral,
        long bodySalt,
        int x,
        int y,
        int z
    ) {
        double richness = switch (mineral) {
            case COAL -> 0.66;
            case IRON -> 0.60;
            case COPPER -> 0.57;
            case GOLD -> 0.48;
            case LAPIS -> 0.50;
            case REDSTONE -> 0.52;
            case DIAMOND -> 0.36;
            case EMERALD -> 0.34;
        };
        double coarse = unitHash(cellSeed(
            bodySalt ^ 0x434F415253453031L,
            Math.floorDiv(x, 4),
            Math.floorDiv(y, 4),
            Math.floorDiv(z, 4)
        ));
        double fine = unitHash(cellSeed(
            bodySalt ^ 0x46494E454F524531L,
            Math.floorDiv(x, 2),
            Math.floorDiv(y, 2),
            Math.floorDiv(z, 2)
        ));
        return coarse * 0.68 + fine * 0.32 < richness;
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
        h ^= (long) x * 0xBF58476D1CE4E5B9L;
        h ^= (long) y * 0x94D049BB133111EBL;
        h ^= (long) z * 0xD6E8FEB86659FD93L;
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        return h ^ (h >>> 31);
    }
}
