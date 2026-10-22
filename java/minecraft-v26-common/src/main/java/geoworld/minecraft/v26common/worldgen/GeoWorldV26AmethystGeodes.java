package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;

import net.minecraft.core.BlockPos;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ChunkAccess;

import java.util.function.Consumer;

/** Sparse guaranteed amethyst geodes, supplementing retained vanilla geodes. */
public final class GeoWorldV26AmethystGeodes {
    private static final int SPACING = 1024;
    private static final int CENTER_OFFSET_X = 928;
    private static final int CENTER_OFFSET_Z = 160;
    private static final int CENTER_Y = -24;
    private static final int RADIUS = 7;

    public enum Material {
        SMOOTH_BASALT, CALCITE, AMETHYST, BUDDING_AMETHYST, AIR;

        BlockState block() {
            return switch (this) {
                case SMOOTH_BASALT -> Blocks.SMOOTH_BASALT.defaultBlockState();
                case CALCITE -> Blocks.CALCITE.defaultBlockState();
                case AMETHYST -> Blocks.AMETHYST_BLOCK.defaultBlockState();
                case BUDDING_AMETHYST -> Blocks.BUDDING_AMETHYST.defaultBlockState();
                case AIR -> Blocks.AIR.defaultBlockState();
            };
        }
    }

    public record Placement(int x, int y, int z, Material material) {}

    private GeoWorldV26AmethystGeodes() {}

    public static int populate(
        ChunkAccess chunk,
        long worldSeed,
        GeoWorldShorelinePlanner.ChunkPlan shoreline,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter
    ) {
        int minX = chunk.getPos().getMinBlockX();
        int minZ = chunk.getPos().getMinBlockZ();
        BlockPos.MutableBlockPos pos = new BlockPos.MutableBlockPos();
        final int[] placed = {0};
        plan(worldSeed, minX, minX + 15, minZ, minZ + 15, sample -> {
            pos.set(sample.x(), sample.y(), sample.z());
            BlockState original = chunk.getBlockState(pos);
            // Only carve/replace intact subterranean host; other features and
            // hydrology-owned water are never overwritten.
            int lx = sample.x() - minX;
            int lz = sample.z() - minZ;
            if (sample.y() < shoreline.column(lx, lz).groundBoundary() - 6 &&
                (original.is(Blocks.STONE) || original.is(Blocks.DEEPSLATE))) {
                blockSetter.set(chunk, pos, sample.material().block());
                placed[0]++;
            }
        });
        return placed[0];
    }

    public static void plan(
        long seed, int minX, int maxX, int minZ, int maxZ,
        Consumer<Placement> output
    ) {
        int firstX = Math.floorDiv(minX - RADIUS - 2 - CENTER_OFFSET_X, SPACING);
        int lastX = Math.floorDiv(maxX + RADIUS + 2 - CENTER_OFFSET_X, SPACING);
        int firstZ = Math.floorDiv(minZ - RADIUS - 2 - CENTER_OFFSET_Z, SPACING);
        int lastZ = Math.floorDiv(maxZ + RADIUS + 2 - CENTER_OFFSET_Z, SPACING);
        for (int tileX = firstX; tileX <= lastX; tileX++) {
            for (int tileZ = firstZ; tileZ <= lastZ; tileZ++) {
                int centerX = tileX * SPACING + CENTER_OFFSET_X;
                int centerZ = tileZ * SPACING + CENTER_OFFSET_Z;
                long h = seed ^ (tileX * 0x9E3779B97F4A7C15L) ^
                    (tileZ * 0xC2B2AE3D27D4EB4FL);
                // Slight local shifts keep deposits from appearing on an exact
                // supercell-wide grid, while their source remains deterministic.
                int shiftX = (int)Math.floorMod(h, 3L) - 1;
                int shiftZ = (int)Math.floorMod(h >>> 12, 3L) - 1;
                centerX += shiftX;
                centerZ += shiftZ;
                for (int x = Math.max(minX, centerX - RADIUS);
                     x <= Math.min(maxX, centerX + RADIUS); x++) {
                    for (int z = Math.max(minZ, centerZ - RADIUS);
                         z <= Math.min(maxZ, centerZ + RADIUS); z++) {
                        for (int y = CENTER_Y - 7; y <= CENTER_Y + 7; y++) {
                            double dx = (x - centerX) / 6.4;
                            double dy = (y - CENTER_Y) / 5.6;
                            double dz = (z - centerZ) / 6.4;
                            double d = Math.sqrt(dx * dx + dy * dy + dz * dz);
                            if (d > 1.0) continue;
                            Material material = d > 0.82
                                ? Material.SMOOTH_BASALT
                                : d > 0.68 ? Material.CALCITE
                                : d > 0.50 ? Material.AMETHYST
                                : d > 0.39
                                    ? (((x + y + z) & 3) == 0
                                        ? Material.BUDDING_AMETHYST
                                        : Material.AMETHYST)
                                : Material.AIR;
                            output.accept(new Placement(x, y, z, material));
                        }
                    }
                }
            }
        }
    }
}
