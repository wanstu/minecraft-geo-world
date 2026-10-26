package geoworld.minecraft.v26common.worldgen;

import geoworld.runtime.GeoWorldV1;
import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner;
import geoworld.minecraft.common.worldgen.GeoWorldMineralRules;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.v26common.MinecraftV26HeightMapping;
import net.minecraft.core.BlockPos;
import net.minecraft.world.level.block.Block;
import net.minecraft.world.level.block.Blocks;
import net.minecraft.world.level.block.state.BlockState;
import net.minecraft.world.level.chunk.ChunkAccess;

import java.util.BitSet;
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
public final class GeoWorldV26MineralVeins {
    private static final HeightMapping HEIGHTS =
        new MinecraftV26HeightMapping();
    private static final GeoWorldMineralPlanner PLANNER =
        new GeoWorldMineralPlanner(HEIGHTS);
    // Optional vanilla blocks must not become compile-time dependencies of
    // version-common code. Resolve once so older supported Minecraft versions
    // simply omit hosts that did not exist yet.
    private static final Block OPTIONAL_PALE_MOSS_BLOCK =
        optionalVanillaBlock("PALE_MOSS_BLOCK");

    public enum Mineral {
        // Rarity primarily controls how frequently a vein starts, NOT
        // whether it is a 2-block speck. Even diamond/emerald deposits form
        // tangible three-dimensional lodes when discovered.
        // Vanilla granite/andesite-style lobes: short, thick and irregular,
        // rather than long thin strings of ore.
        COAL(GeoWorldMineralRules.Kind.COAL),
        IRON(GeoWorldMineralRules.Kind.IRON),
        COPPER(GeoWorldMineralRules.Kind.COPPER),
        GOLD(GeoWorldMineralRules.Kind.GOLD),
        LAPIS(GeoWorldMineralRules.Kind.LAPIS),
        REDSTONE(GeoWorldMineralRules.Kind.REDSTONE),
        DIAMOND(GeoWorldMineralRules.Kind.DIAMOND),
        EMERALD(GeoWorldMineralRules.Kind.EMERALD);

        final GeoWorldMineralRules.Kind kind;

        Mineral(GeoWorldMineralRules.Kind kind) {
            this.kind = kind;
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

    public record Placement(int x, int y, int z, Mineral mineral) {}

    private GeoWorldV26MineralVeins() {}

    private static boolean usesV117Minerals(GeoWorldV1 model) {
        String rules = model.manifest().generationRulesVersion();
        return "geoworld-rules-v1.1.7".equals(rules) ||
            "geoworld-rules-v1.1.8".equals(rules) ||
            "geoworld-rules-v1.1.9".equals(rules);
    }

    public static int populate(
        ChunkAccess chunk,
        GeoWorldV1 model,
        long seed,
        GeoWorldShorelinePlanner.ChunkPlan shoreline,
        GeoWorldV26TerrainPopulator.BlockSetter blockSetter
    ) {
        int minX = chunk.getPos().getMinBlockX();
        int minZ = chunk.getPos().getMinBlockZ();
        BlockPos.MutableBlockPos position = new BlockPos.MutableBlockPos();
        final int[] added = {0};
        boolean modernMinerals = usesV117Minerals(model);
        // Long, thick seams contain overlapping ellipsoid sweeps. Skip each
        // voxel after the first mineral candidate, before expensive chunk
        // state and surface checks. First-seen ordering is unchanged.
        BitSet seenUnderground = new BitSet(HEIGHTS.height() * 256);
        plan(model, seed, minX, minX + 15, minZ, minZ + 15, placement -> {
            int localX = placement.x() - minX;
            int localZ = placement.z() - minZ;
            int voxel = ((placement.y() - (HEIGHTS.minY() + 1)) * 256) +
                localZ * 16 + localX;
            if (seenUnderground.get(voxel)) return;
            seenUnderground.set(voxel);
            position.set(placement.x(), placement.y(), placement.z());
            BlockState host = chunk.getBlockState(position);
            // Leave bedrock, ocean/river water, soil, snow, and decorations intact.
            // The vanilla ore features are suppressed for this generator only.
            GeoWorldShorelinePlanner.Column column = shoreline.column(localX, localZ);
            int safeBoundary = column.groundBoundary() - 4;
            int surfaceY = column.groundBoundary() - 1;
            boolean solidHost = host.is(Blocks.STONE) || host.is(Blocks.DEEPSLATE);
            // Rare natural ore exposures occur only where a planned vein
            // intersects an actual STONE mountain face. Never expose ore on
            // grass, snow, a shoreline, or underwater. Default remains buried.
            boolean outcrop = !modernMinerals &&
                placement.y() == surfaceY &&
                host.is(Blocks.STONE) &&
                column.surfaceOverride() == null &&
                isOutcropLocation(
                    placement.mineral(), column.sample().terrainVariant(),
                    column.sample().waterKind(),
                    placement.x(), placement.z(), seed
                );
            if (solidHost && (placement.y() < safeBoundary || outcrop)) {
                blockSetter.set(
                    chunk,
                    position,
                    placement.mineral().blockAt(placement.y())
                );
                added[0]++;
            }
        });
        if (modernMinerals) {
            BitSet seenScatter = new BitSet(HEIGHTS.height() * 256);
            planScatter(
                model, seed,
                minX, minX + 15, minZ, minZ + 15,
                placement -> {
                    int localX = placement.x() - minX;
                    int localZ = placement.z() - minZ;
                    int voxel =
                        ((placement.y() - (HEIGHTS.minY() + 1)) * 256) +
                        localZ * 16 + localX;
                    if (seenScatter.get(voxel)) return;
                    seenScatter.set(voxel);

                    GeoWorldShorelinePlanner.Column column =
                        shoreline.column(localX, localZ);
                    if (placement.y() >= column.groundBoundary() - 4) {
                        return;
                    }
                    position.set(
                        placement.x(), placement.y(), placement.z()
                    );
                    BlockState host = chunk.getBlockState(position);
                    if (!host.is(Blocks.STONE) &&
                        !host.is(Blocks.DEEPSLATE)) {
                        return;
                    }
                    blockSetter.set(
                        chunk,
                        position,
                        placement.mineral().blockAt(placement.y())
                    );
                    added[0]++;
                }
            );
        }

        // Surface lodes are a separate, sparse geological layer. Legacy
        // worlds retain the old mountain+STONE exposure rule; v1.1.7 lets a
        // mineral body crop out through natural land covers such as grass,
        // soil, gravel, sand or badlands terracotta.
        BitSet seenOutcrop = new BitSet(HEIGHTS.height() * 256);
        byte[] rockSurface = new byte[256];
        planSurfaceOutcrops(model, seed, minX, minX + 15, minZ, minZ + 15,
            placement -> {
                int localX = placement.x() - minX;
                int localZ = placement.z() - minZ;
                int voxel = ((placement.y() - (HEIGHTS.minY() + 1)) * 256) +
                    localZ * 16 + localX;
                if (seenOutcrop.get(voxel)) return;
                seenOutcrop.set(voxel);

                GeoWorldShorelinePlanner.Column column =
                    shoreline.column(localX, localZ);
                int surface = column.groundBoundary() - 1;
                int outcropDepth = modernMinerals ? 31 : 23;
                if (placement.y() > surface ||
                    placement.y() < surface - outcropDepth) {
                    return;
                }

                if (modernMinerals) {
                    if (!"None".equals(column.sample().waterKind()) ||
                        column.surfaceOverride() != null) {
                        return;
                    }
                    position.set(
                        placement.x(), placement.y(), placement.z()
                    );
                    BlockState host = chunk.getBlockState(position);
                    if (!isNaturalOutcropHost(host)) return;
                } else {
                    int columnIndex = localZ * 16 + localX;
                    if (rockSurface[columnIndex] == 0) {
                        if (column.surfaceOverride() != null ||
                            !"None".equals(column.sample().waterKind()) ||
                            !GeoWorldMineralRules.isMountainTerrain(
                                column.sample().terrainVariant()
                            )) {
                            rockSurface[columnIndex] = 1;
                        } else {
                            position.set(
                                placement.x(), surface, placement.z()
                            );
                            rockSurface[columnIndex] = (byte)(
                                chunk.getBlockState(position).is(Blocks.STONE)
                                    ? 2 : 1
                            );
                        }
                    }
                    if (rockSurface[columnIndex] != 2) return;
                    position.set(
                        placement.x(), placement.y(), placement.z()
                    );
                    if (!chunk.getBlockState(position).is(Blocks.STONE)) {
                        return;
                    }
                }
                blockSetter.set(
                    chunk,
                    position,
                    placement.mineral().blockAt(placement.y())
                );
                added[0]++;
            });
        return added[0];
    }

    /**
     * Separate geography-anchored lodes near the natural land surface.
     * v1.1.7+ treats terrain as geological affinity rather than an eligibility
     * gate, so a coherent deposit may crop out on plains, plateaus, badlands
     * or mountains instead of requiring a literal STONE mountain face.
     */
    public static void planSurfaceOutcrops(
        GeoWorldV1 model, long seed, int minX, int maxX,
        int minZ, int maxZ, Consumer<Placement> consumer
    ) {
        PLANNER.planSurfaceOutcrops(
            model, seed, minX, maxX, minZ, maxZ,
            placement -> consumer.accept(fromCommon(placement))
        );
    }

    public static void planScatter(
        GeoWorldV1 model, long seed,
        int minX, int maxX, int minZ, int maxZ,
        Consumer<Placement> consumer
    ) {
        PLANNER.planScatter(
            model, seed, minX, maxX, minZ, maxZ,
            placement -> consumer.accept(fromCommon(placement))
        );
    }

    /** Pure placement geometry, used for both chunk generation and seam tests. */
    public static void plan(
        GeoWorldV1 model, long seed,
        int minX, int maxX, int minZ, int maxZ,
        Consumer<Placement> consumer
    ) {
        PLANNER.plan(
            model, seed, minX, maxX, minZ, maxZ,
            placement -> consumer.accept(fromCommon(placement))
        );
    }

    public static boolean isOutcropLocation(
        Mineral mineral, String terrain, String water,
        int x, int z, long seed
    ) {
        return GeoWorldMineralRules.isOutcropLocation(
            mineral.kind, terrain, water, x, z, seed
        );
    }

    public static Mineral chooseMineral(
        String terrain, String biome, double y, double roll
    ) {
        GeoWorldMineralRules.Kind selected =
            GeoWorldMineralRules.chooseMineral(
                terrain, biome, y, roll
            );
        return selected == null ? null : Mineral.valueOf(selected.name());
    }

    public static double affinity(
        Mineral mineral,
        String terrain,
        String biome,
        double y
    ) {
        return GeoWorldMineralRules.affinity(
            mineral.kind, terrain, biome, y
        );
    }

    public static int guaranteeY(Mineral mineral, boolean deep) {
        return GeoWorldMineralRules.guaranteeY(mineral.kind, deep);
    }

    // Deterministic, full-geometry fixture for auditing the lower bound of
    // each mineral body without bootstrapping a Minecraft server.
    public static void planReferenceBody(
        Mineral mineral, boolean exposed, long seed,
        Consumer<Placement> consumer
    ) {
        PLANNER.planReferenceBody(
            mineral.kind,
            exposed,
            seed,
            placement -> consumer.accept(fromCommon(placement))
        );
    }

    private static boolean isNaturalOutcropHost(BlockState host) {
        return host.is(Blocks.STONE) ||
            host.is(Blocks.DEEPSLATE) ||
            host.is(Blocks.GRANITE) ||
            host.is(Blocks.DIORITE) ||
            host.is(Blocks.ANDESITE) ||
            host.is(Blocks.TUFF) ||
            host.is(Blocks.CALCITE) ||
            host.is(Blocks.DIRT) ||
            host.is(Blocks.GRASS_BLOCK) ||
            host.is(Blocks.COARSE_DIRT) ||
            host.is(Blocks.PODZOL) ||
            host.is(Blocks.ROOTED_DIRT) ||
            host.is(Blocks.MUD) ||
            host.is(Blocks.PACKED_MUD) ||
            host.is(Blocks.CLAY) ||
            host.is(Blocks.SAND) ||
            host.is(Blocks.RED_SAND) ||
            host.is(Blocks.SANDSTONE) ||
            host.is(Blocks.RED_SANDSTONE) ||
            host.is(Blocks.GRAVEL) ||
            host.is(Blocks.TERRACOTTA) ||
            host.is(Blocks.MYCELIUM) ||
            (OPTIONAL_PALE_MOSS_BLOCK != null &&
                host.is(OPTIONAL_PALE_MOSS_BLOCK));
    }

    private static Block optionalVanillaBlock(String fieldName) {
        try {
            return (Block) Blocks.class.getField(fieldName).get(null);
        } catch (ReflectiveOperationException ignored) {
            return null;
        }
    }

    private static Placement fromCommon(
        GeoWorldMineralPlanner.Placement placement
    ) {
        return new Placement(
            placement.x(),
            placement.y(),
            placement.z(),
            Mineral.valueOf(placement.mineral().name())
        );
    }

}
