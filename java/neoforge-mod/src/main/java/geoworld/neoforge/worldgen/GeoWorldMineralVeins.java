package geoworld.neoforge.worldgen;

import geoworld.runtime.GeoWorldV1;
import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
import net.minecraft.core.BlockPos;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ChunkAccess;

import java.util.BitSet;
import java.util.SplittableRandom;
import java.util.function.Consumer;

/**
 * Deterministic, chunk-independent mineral deposits for GeoWorld terrain.
 *
 * Deposits belong to global 64x64x48 source cells. Each chunk clips the same
 * world-space vein geometry to its own footprint, so an ore seam can continue
 * across chunk boundaries without depending on generation order.
 *
 * Ore affinity uses persisted terrain/biome labels as a first geological
 * approximation. It does not pretend to reconstruct missing deep strata.
 */
final class GeoWorldMineralVeins {
    private static final int CELL_XZ = 64;
    private static final int CELL_Y = 48;
    private static final int SEARCH_MARGIN = 56;
    private static final int MIN_Y = -63;
    private static final int MAX_Y = 319;
    private static final Minecraft26_3HeightMapping HEIGHTS =
        new Minecraft26_3HeightMapping();
    private static final long OUTCROP_SALT = 0x4F555443524F5053L;

    enum Mineral {
        COAL(-63, 215, 7.5),
        IRON(-60, 319, 9),
        COPPER(-20, 150, 6.0),
        GOLD(-63, 60, 3.2),
        LAPIS(-63, 80, 2.7),
        REDSTONE(-63, 24, 4.2),
        DIAMOND(-63, 24, 1.0),
        EMERALD(-63, 319, 1.2);

        final int minY;
        final int maxY;
        final double weight;

        Mineral(int minY, int maxY, double weight) {
            this.minY = minY;
            this.maxY = maxY;
            this.weight = weight;
        }

        // Resolve Minecraft blocks only in the actual generation runtime.
        // Pure geometry verifiers must not need an FML registry bootstrap.
        BlockState blockAt(int y) {
            boolean deep = y < 0;
            return switch (this) {
                case COAL -> deep ? Blocks.DEEPSLATE_COAL_ORE.defaultBlockState() : Blocks.COAL_ORE.defaultBlockState();
                case IRON -> deep ? Blocks.DEEPSLATE_IRON_ORE.defaultBlockState() : Blocks.IRON_ORE.defaultBlockState();
                case COPPER -> deep ? Blocks.DEEPSLATE_COPPER_ORE.defaultBlockState() : Blocks.COPPER_ORE.defaultBlockState();
                case GOLD -> deep ? Blocks.DEEPSLATE_GOLD_ORE.defaultBlockState() : Blocks.GOLD_ORE.defaultBlockState();
                case LAPIS -> deep ? Blocks.DEEPSLATE_LAPIS_ORE.defaultBlockState() : Blocks.LAPIS_ORE.defaultBlockState();
                case REDSTONE -> deep ? Blocks.DEEPSLATE_REDSTONE_ORE.defaultBlockState() : Blocks.REDSTONE_ORE.defaultBlockState();
                case DIAMOND -> deep ? Blocks.DEEPSLATE_DIAMOND_ORE.defaultBlockState() : Blocks.DIAMOND_ORE.defaultBlockState();
                case EMERALD -> deep ? Blocks.DEEPSLATE_EMERALD_ORE.defaultBlockState() : Blocks.EMERALD_ORE.defaultBlockState();
            };
        }
    }

    record Placement(int x, int y, int z, Mineral mineral) {}

    private GeoWorldMineralVeins() {}

    static int populate(ChunkAccess chunk, GeoWorldV1 model, long seed, GeoWorldShoreline.ChunkPlan shoreline) {
        int minX = chunk.getPos().getMinBlockX();
        int minZ = chunk.getPos().getMinBlockZ();
        BlockPos.MutableBlockPos position = new BlockPos.MutableBlockPos();
        final int[] added = {0};
        plan(model, seed, minX, minX + 15, minZ, minZ + 15, placement -> {
            position.set(placement.x(), placement.y(), placement.z());
            BlockState host = chunk.getBlockState(position);
            // Leave bedrock, ocean/river water, soil, snow, and decorations intact.
            // The vanilla ore features are suppressed for this generator only.
            int localX = placement.x() - minX;
            int localZ = placement.z() - minZ;
            GeoWorldShoreline.Column column = shoreline.column(localX, localZ);
            int safeBoundary = column.groundBoundary() - 4;
            int surfaceY = column.groundBoundary() - 1;
            boolean solidHost = host.is(Blocks.STONE) || host.is(Blocks.DEEPSLATE);
            // Rare natural ore exposures occur only where a planned vein
            // intersects an actual STONE mountain face. Never expose ore on
            // grass, snow, a shoreline, or underwater. Default remains buried.
            boolean outcrop = placement.y() == surfaceY &&
                host.is(Blocks.STONE) &&
                column.surfaceOverride() == null &&
                isOutcropLocation(
                    placement.mineral(), column.sample().terrainVariant(),
                    column.sample().waterKind(),
                    placement.x(), placement.z(), seed
                );
            if (solidHost && (placement.y() < safeBoundary || outcrop)) {
                chunk.setBlockState(position, placement.mineral().blockAt(placement.y()));
                added[0]++;
            }
        });
        // The swept ellipsoids overlap heavily. Deduplicate their voxel
        // positions before even probing host blocks, and cache the expensive
        // mountain/rock-surface validation once per x/z column.
        BitSet seenOutcrop = new BitSet(384 * 256);
        byte[] rockSurface = new byte[256];
        // Independently plan rare lodes intersecting natural high rock.
        // Subsurface mineral reserves remain intact; only real exposed
        // stone on mountains can receive these surface-visible ores.
        planSurfaceOutcrops(model, seed, minX, minX + 15, minZ, minZ + 15,
            placement -> {
                int localX = placement.x() - minX;
                int localZ = placement.z() - minZ;
                int voxel = ((placement.y() + 63) * 256) +
                    localZ * 16 + localX;
                if (seenOutcrop.get(voxel)) return;
                seenOutcrop.set(voxel);

                GeoWorldShoreline.Column column = shoreline.column(localX, localZ);
                int surface = column.groundBoundary() - 1;
                if (placement.y() > surface || placement.y() < surface - 13) {
                    return;
                }
                // Confirm the entire host column is a naturally exposed rock
                // formation, not buried beneath snow, grass or an ocean.
                int columnIndex = localZ * 16 + localX;
                if (rockSurface[columnIndex] == 0) {
                    if (column.surfaceOverride() != null ||
                        !"None".equals(column.sample().waterKind()) ||
                        !isMountain(column.sample().terrainVariant())) {
                        rockSurface[columnIndex] = 1;
                    } else {
                        position.set(placement.x(), surface, placement.z());
                        rockSurface[columnIndex] = (byte)(
                            chunk.getBlockState(position).is(Blocks.STONE) ? 2 : 1
                        );
                    }
                }
                if (rockSurface[columnIndex] != 2) return;
                position.set(placement.x(), placement.y(), placement.z());
                if (!chunk.getBlockState(position).is(Blocks.STONE)) return;
                chunk.setBlockState(
                    position, placement.mineral().blockAt(placement.y())
                );
                added[0]++;
            });
        return added[0];
    }

    private static boolean isMountain(String terrain) {
        return "Mountain".equals(terrain) ||
            "Foothills".equals(terrain);
    }

    /**
     * Separate geography-anchored lodes; unlike underground random sources,
     * they are intentionally centered within a few blocks of a mountain's
     * true voxel surface, so erosion can expose coherent clusters.
     */
    static void planSurfaceOutcrops(
        GeoWorldV1 model, long seed, int minX, int maxX,
        int minZ, int maxZ, Consumer<Placement> consumer
    ) {
        for (int cellZ = Math.floorDiv(minZ - SEARCH_MARGIN, CELL_XZ);
             cellZ <= Math.floorDiv(maxZ + SEARCH_MARGIN, CELL_XZ); cellZ++) {
            for (int cellX = Math.floorDiv(minX - SEARCH_MARGIN, CELL_XZ);
                 cellX <= Math.floorDiv(maxX + SEARCH_MARGIN, CELL_XZ); cellX++) {
                SplittableRandom random = new SplittableRandom(
                    cellSeed(seed ^ OUTCROP_SALT, cellX, 0, cellZ)
                );
                // Lodes are less rare than the old 1/12 sources, but still
                // clustered in individual 64x64 geological cells.
                if (random.nextInt(8) != 0) continue;
                // Do not pin every lode exactly to the center of its cell.
                double centerX = cellX * (double)CELL_XZ + CELL_XZ / 2.0
                    + (random.nextDouble() - 0.5) * 24.0;
                double centerZ = cellZ * (double)CELL_XZ + CELL_XZ / 2.0
                    + (random.nextDouble() - 0.5) * 24.0;
                var geology = model.sample(centerX, centerZ);
                if (!geology.sourceLand() ||
                    !"None".equals(geology.waterKind()) ||
                    !isMountain(geology.terrainVariant())) {
                    continue;
                }
                int surface = HEIGHTS.boundaryY(geology.groundHeight()) - 1;
                if (surface < 85 || surface >= MAX_Y) continue;
                Mineral mineral = surface > 215
                    ? (random.nextBoolean() ? Mineral.IRON : Mineral.EMERALD)
                    : surface > 150
                        ? switch (random.nextInt(3)) {
                            case 0 -> Mineral.COAL;
                            case 1 -> Mineral.IRON;
                            default -> Mineral.EMERALD;
                        }
                        : switch (random.nextInt(4)) {
                            case 0 -> Mineral.COAL;
                            case 1 -> Mineral.IRON;
                            case 2 -> Mineral.COPPER;
                            default -> Mineral.EMERALD;
                        };
                emitVein(
                    mineral, centerX, surface - 4.0, centerZ,
                    random, minX, maxX, minZ, maxZ, consumer, true
                );
            }
        }
    }

    /** Pure placement geometry, used for both chunk generation and seam tests. */
    static void plan(
        GeoWorldV1 model, long seed,
        int minX, int maxX, int minZ, int maxZ,
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
                // One fixed sparse seam of every overworld ore family in
                // each 1024x1024 supercell. Ordinary veins below still follow
                // geology; these anchors prevent rare ores disappearing due
                // to deterministic random choice on finite world seeds.
                for (Mineral guarantee : Mineral.values()) {
                    int slot = guarantee.ordinal();
                    int targetX = 2 + (slot % 4) * 4;
                    int targetZ = 4 + (slot / 4) * 8;
                    if (Math.floorMod(cellX, 16) == targetX &&
                        Math.floorMod(cellZ, 16) == targetZ) {
                        // Both stone and deepslate variants exist for every
                        // vanilla overworld ore, including rare deep coal
                        // and emerald deposits.
                        for (int guaranteedY : new int[]{
                            guaranteeY(guarantee, false),
                            guaranteeY(guarantee, true)
                        }) {
                            emitVein(
                                guarantee,
                                cellX * (double)CELL_XZ + 32,
                                guaranteedY,
                                cellZ * (double)CELL_XZ + 32,
                                new SplittableRandom(
                                    cellSeed(seed, cellX, guaranteedY, cellZ) ^
                                    0x4D494E4552414C31L
                                ),
                                minX, maxX, minZ, maxZ, consumer
                            );
                        }
                    }
                }
                for (int cellY = Math.floorDiv(MIN_Y, CELL_Y);
                     cellY <= Math.floorDiv(MAX_Y, CELL_Y); cellY++) {
                    SplittableRandom random = new SplittableRandom(
                        cellSeed(seed, cellX, cellY, cellZ)
                    );
                    for (int attempt = 0; attempt < 2; attempt++) {
                        if (random.nextDouble() > (attempt == 0 ? 0.88 : 0.43)) {
                            continue;
                        }
                        double x = (cellX + 0.28 + random.nextDouble() * 0.44) * CELL_XZ;
                        double z = (cellZ + 0.28 + random.nextDouble() * 0.44) * CELL_XZ;
                        double y = (cellY + 0.22 + random.nextDouble() * 0.56) * CELL_Y;
                        Mineral mineral = chooseMineral(geology.terrainVariant(), geology.biomeFamily(), y, random.nextDouble());
                        if (mineral == null) {
                            continue;
                        }
                        emitVein(mineral, x, y, z, random, minX, maxX, minZ, maxZ, consumer);
                    }
                }
            }
        }
    }

    static boolean isOutcropLocation(
        Mineral mineral, String terrain, String water,
        int x, int z, long seed
    ) {
        if (!"None".equals(water) ||
            !("Mountain".equals(terrain) || "Foothills".equals(terrain))) {
            return false;
        }
        if (mineral != Mineral.COAL && mineral != Mineral.IRON &&
            mineral != Mineral.COPPER && mineral != Mineral.EMERALD) {
            return false;
        }
        long patch = cellSeed(
            seed ^ 0x4F555443524F5053L,
            Math.floorDiv(x, 8), 0, Math.floorDiv(z, 8)
        );
        // 1/16 rocky micro-regions: low density and coherent outcrop patches.
        return (patch & 15L) == 0L;
    }

    static Mineral chooseMineral(String terrain, String biome, double y, double roll) {
        double total = 0;
        for (Mineral mineral : Mineral.values()) {
            total += affinity(mineral, terrain, biome, y);
        }
        if (total == 0) {
            return null;
        }
        double target = roll * total;
        for (Mineral mineral : Mineral.values()) {
            double weight = affinity(mineral, terrain, biome, y);
            if (weight <= 0) {
                continue;
            }
            target -= weight;
            if (target < 0) {
                return mineral;
            }
        }
        return Mineral.IRON;
    }

    static double affinity(Mineral mineral, String terrain, String biome, double y) {
        if (y < mineral.minY || y > mineral.maxY) {
            return 0;
        }
        boolean mountains = "Mountain".equals(terrain);
        boolean foothills = "Foothills".equals(terrain) || "Hills".equals(terrain);
        boolean sedimentary = switch (terrain) {
            case "Plain", "Rolling Plain", "Floodplain", "Alluvial Plain", "Alluvial Fan" -> true;
            default -> false;
        };
        boolean arid = biome.contains("Desert") || "Savanna".equals(biome);
        boolean highlands = mountains || foothills || "Plateau".equals(terrain);
        double multiplier = switch (mineral) {
            case COAL -> sedimentary ? 2.1 : 0.80;
            case IRON -> mountains ? 2.2 : foothills ? 1.6 : 1.0;
            case COPPER -> highlands ? 2.0 : 0.9;
            case GOLD -> ("Plateau".equals(terrain) || "Canyon".equals(terrain) || arid) ? 2.1 : 0.75;
            case LAPIS -> sedimentary ? 1.5 : 1.0;
            case REDSTONE -> highlands ? 1.25 : 1.0;
            case DIAMOND -> 1.0; // depth first, no invented surface-rock certainty
            case EMERALD -> mountains ? 2.4 : foothills ? 1.4 : 0.08;
        };
        // Coal favors upper deposits; diamonds/redstone are more common deep
        // underground. Other mineral types retain broad vertical intervals.
        double vertical = switch (mineral) {
            case COAL -> y < 0 ? 0.10 : y < 25 ? 0.7 : 1.3;
            case DIAMOND -> y > 0 ? 0.18 : 0.5 + (-y / 63.0);
            case REDSTONE -> y > 0 ? 0.3 : 1.0 + (-y / 100.0);
            case EMERALD -> y < 0 ? 0.08 : y > 55 ? 1.2 : 0.6;
            default -> 1.0;
        };
        return mineral.weight * multiplier * vertical;
    }

    static int guaranteeY(Mineral mineral, boolean deep) {
        if (deep) {
            return switch (mineral) {
                case COAL, COPPER, EMERALD -> -12;
                case IRON, GOLD, LAPIS -> -25;
                case REDSTONE -> -36;
                case DIAMOND -> -52;
            };
        }
        return switch (mineral) {
            case COAL -> 80;
            case IRON -> 28;
            case COPPER -> 30;
            case GOLD -> 32;
            case LAPIS -> 20;
            case REDSTONE -> 8;
            case DIAMOND -> 6;
            case EMERALD -> 64;
        };
    }

    private static void emitVein(
        Mineral mineral, double x, double y, double z,
        SplittableRandom random,
        int minX, int maxX, int minZ, int maxZ,
        Consumer<Placement> consumer
    ) {
        emitVein(mineral, x, y, z, random, minX, maxX, minZ, maxZ,
            consumer, false);
    }

    private static void emitVein(
        Mineral mineral, double x, double y, double z,
        SplittableRandom random,
        int minX, int maxX, int minZ, int maxZ,
        Consumer<Placement> consumer, boolean rockyOutcrop
    ) {
        boolean large = random.nextDouble() < 0.025;
        double length = rockyOutcrop
            ? (large ? 50 + random.nextDouble() * 10
                : 30 + random.nextDouble() * 22)
            : (large ? 32 + random.nextDouble() * 16
                : 13 + random.nextDouble() * 18);
        double radius = rockyOutcrop
            ? (large ? 3.6 + random.nextDouble() * 0.8
                : 2.5 + random.nextDouble() * 1.1)
            : (large ? 2.4 + random.nextDouble() * 0.8
                : 1.2 + random.nextDouble() * 0.9);
        double angle = random.nextDouble() * Math.PI * 2;
        double dx = Math.cos(angle), dz = Math.sin(angle);
        double tilt = (random.nextDouble() - 0.5) * 0.32;
        double bend = (random.nextDouble() - 0.5) * 4;
        // After consuming all the same PRNG draws, reject veins whose
        // complete swept envelope cannot touch this chunk. Keeps both
        // ore placement and subsequent attempts bit-for-bit unchanged.
        double reach = length * 0.5 + Math.abs(bend) + radius + 1;
        if (x + reach < minX || x - reach > maxX ||
            z + reach < minZ || z - reach > maxZ) {
            return;
        }
        // Rocky lodes have >=2.5-block radius, so sampling their centers
        // every 2.5 blocks stays continuous and avoids re-emitting many
        // identical voxels. Preserve original underground vein geometry.
        int steps = (int)Math.ceil(length / (rockyOutcrop ? 2.5 : 1.5));
        for (int step = 0; step <= steps; step++) {
            double t = (double) step / steps;
            double longitudinal = (t - 0.5) * length;
            double lateral = Math.sin(t * Math.PI) * bend;
            double cx = x + longitudinal * dx - lateral * dz;
            double cz = z + longitudinal * dz + lateral * dx;
            double cy = y + longitudinal * tilt;
            double r = radius * (0.82 + 0.18 * Math.sin(t * Math.PI));
            double verticalRadius = rockyOutcrop ? r * 1.75 : r;
            int left = Math.max(minX, (int)Math.floor(cx - r));
            int right = Math.min(maxX, (int)Math.ceil(cx + r));
            int north = Math.max(minZ, (int)Math.floor(cz - r));
            int south = Math.min(maxZ, (int)Math.ceil(cz + r));
            int low = Math.max(MIN_Y, (int)Math.floor(cy - verticalRadius));
            int high = Math.min(MAX_Y, (int)Math.ceil(cy + verticalRadius));
            if (left > right || north > south || low > high) {
                continue;
            }
            for (int bx = left; bx <= right; bx++) {
                for (int bz = north; bz <= south; bz++) {
                    for (int by = low; by <= high; by++) {
                        double horizontal2 = (bx - cx) * (bx - cx) +
                            (bz - cz) * (bz - cz);
                        double vertical = (by - cy) / verticalRadius;
                        if (horizontal2 / (r * r) + vertical * vertical <= 1.0) {
                            consumer.accept(new Placement(bx, by, bz, mineral));
                        }
                    }
                }
            }
        }
    }

    private static long cellSeed(long seed, int x, int y, int z) {
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
