package geoworld.minecraft.common.worldgen;

import geoworld.runtime.GeoWorldV1;

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
    private static final long OUTCROP_SALT = 0x4F555443524F5053L;

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

    public void planSurfaceOutcrops(
        GeoWorldV1 model,
        long seed,
        int minX,
        int maxX,
        int minZ,
        int maxZ,
        Consumer<Placement> consumer
    ) {
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
                    !GeoWorldMineralRules.isMountainTerrain(
                        geology.terrainVariant()
                    )
                ) {
                    continue;
                }

                int surface = heights.boundaryY(geology.groundHeight()) - 1;
                if (
                    surface < 85 ||
                    surface >= heights.maxYExclusive() - 1
                ) {
                    continue;
                }

                GeoWorldMineralRules.Kind mineral =
                    surface > 215
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
                    consumer,
                    true
                );
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
        int firstX = Math.floorDiv(minX - SEARCH_MARGIN, CELL_XZ);
        int lastX = Math.floorDiv(maxX + SEARCH_MARGIN, CELL_XZ);
        int firstZ = Math.floorDiv(minZ - SEARCH_MARGIN, CELL_XZ);
        int lastZ = Math.floorDiv(maxZ + SEARCH_MARGIN, CELL_XZ);

        for (int cellZ = firstZ; cellZ <= lastZ; cellZ++) {
            for (int cellX = firstX; cellX <= lastX; cellX++) {
                GeoWorldV1.RuntimeSample geology = model.sample(
                    (double) cellX * CELL_XZ + CELL_XZ / 2.0,
                    (double) cellZ * CELL_XZ + CELL_XZ / 2.0
                );

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
                                consumer,
                                false
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
                        if (
                            random.nextDouble() >
                            (attempt == 0 ? 0.88 : 0.43)
                        ) {
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
                            consumer,
                            false
                        );
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
            exposed
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
        Consumer<Placement> consumer,
        boolean rockyOutcrop
    ) {
        boolean large = random.nextDouble() < 0.035;
        double length =
            mineral.seamLength() *
            (0.90 + random.nextDouble() * 0.22) *
            (large ? 1.22 : 1.0);
        double radius =
            mineral.seamRadius() *
            (0.94 + random.nextDouble() * 0.15) *
            (large ? 1.17 : 1.0) *
            (rockyOutcrop ? 1.12 : 1.0);

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
                        if (
                            horizontal2 / (r * r) +
                            vertical * vertical <= 1.0
                        ) {
                            consumer.accept(
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
