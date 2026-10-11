package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.util.EnumMap;
import java.util.HashMap;
import java.util.Map;

/**
 * Confirms all 8 vanilla Overworld ore families are real placed underground
 * in both stone and deepslate variants, not merely listed in an enum.
 */
public final class GeoWorldMineralVerifier {
    private GeoWorldMineralVerifier() {}

    public static void main(String[] args) throws Exception {
        long seed = args.length > 0 ? Long.parseLong(args[0]) : 123456L;
        GeoWorldV1 model = GeoWorldEngineBridge.generate(
            seed, GeoWorldCreationConfig.defaults()
        ).model();

        // Geological tendency remains meaningful even with reserve seams.
        if (GeoWorldMineralVeins.affinity(
                GeoWorldMineralVeins.Mineral.IRON, "Mountain", "Alpine", 30
            ) <= GeoWorldMineralVeins.affinity(
                GeoWorldMineralVeins.Mineral.IRON, "Plain", "Temperate Forest", 30
            )) {
            throw new AssertionError("iron mineral bias not shaped by geology");
        }
        if (GeoWorldMineralVeins.affinity(
                GeoWorldMineralVeins.Mineral.COAL, "Plain", "Temperate Forest", 30
            ) <= GeoWorldMineralVeins.affinity(
                GeoWorldMineralVeins.Mineral.COAL, "Mountain", "Alpine", 30
            )) {
            throw new AssertionError("coal sedimentary preference disappeared");
        }

        var g = model.manifest().geometry();
        var heightMapping = new Minecraft26_3HeightMapping();
        var found = new EnumMap<GeoWorldMineralVeins.Mineral, boolean[]>(
            GeoWorldMineralVeins.Mineral.class
        );
        for (var mineral : GeoWorldMineralVeins.Mineral.values()) {
            found.put(mineral, new boolean[2]);
        }
        int verified = 0;
        int checks = 0;
        // 1024-block supercell guarantees are searched across the finite
        // geographic world; success requires at least one actual solid host
        // for each normal and deepslate ore variant.
        int firstCellX = (int)Math.floor(g.originX() / 1024);
        int lastCellX = (int)Math.floor(
            (g.originX() + g.spanXBlocks() - 1024) / 1024
        );
        int firstCellZ = (int)Math.floor(g.originZ() / 1024);
        int lastCellZ = (int)Math.floor(
            (g.originZ() + g.spanZBlocks() - 1024) / 1024
        );
        for (int superX = firstCellX; superX <= lastCellX && verified < 16; superX++) {
            for (int superZ = firstCellZ; superZ <= lastCellZ && verified < 16; superZ++) {
                for (var mineral : GeoWorldMineralVeins.Mineral.values()) {
                    boolean[] variants = found.get(mineral);
                    for (int deep = 0; deep < 2; deep++) {
                        if (variants[deep]) continue;
                        int slot = mineral.ordinal();
                        int cellX = superX * 16 + 2 + (slot % 4) * 4;
                        int cellZ = superZ * 16 + 4 + (slot / 4) * 8;
                        int x = cellX * 64 + 32;
                        int z = cellZ * 64 + 32;
                        int y = GeoWorldMineralVeins.guaranteeY(mineral, deep == 1);
                        if (x < g.originX() + 32 || x > g.originX() + g.spanXBlocks() - 32 ||
                            z < g.originZ() + 32 || z > g.originZ() + g.spanZBlocks() - 32) {
                            continue;
                        }
                        int ground = heightMapping.boundaryY(
                            model.sample(x, z).groundHeight()
                        );
                        // y lower than the 4-block soil/shore surface guarantees
                        // the host is STONE or DEEPSLATE before the ore pass.
                        if (y >= ground - 5) continue;
                        final boolean[] placed = {false};
                        GeoWorldMineralVeins.plan(
                            model, seed, x - 6, x + 6, z - 6, z + 6,
                            p -> {
                                if (p.mineral() == mineral &&
                                    Math.abs(p.y() - y) <= 2 &&
                                    p.y() < ground - 5) {
                                    placed[0] = true;
                                }
                            }
                        );
                        checks++;
                        if (placed[0]) {
                            variants[deep] = true;
                            verified++;
                        }
                    }
                }
            }
        }

        if (verified != 16) {
            throw new AssertionError(
                "missing ore variants from actual underground planned deposits: " +
                found + " verified=" + verified + "/16, checks=" + checks
            );
        }

        boolean geodePresent = false;
        for (int tileX = firstCellX; tileX <= lastCellX && !geodePresent; tileX++) {
            for (int tileZ = firstCellZ; tileZ <= lastCellZ && !geodePresent; tileZ++) {
                int x = tileX * 1024 + 928;
                int z = tileZ * 1024 + 160;
                if (x < g.originX() + 16 || x > g.originX() + g.spanXBlocks() - 16 ||
                    z < g.originZ() + 16 || z > g.originZ() + g.spanZBlocks() - 16 ||
                    heightMapping.boundaryY(model.sample(x, z).groundHeight()) < -10) {
                    continue;
                }
                final boolean[] budding = {false};
                GeoWorldAmethystGeodes.plan(
                    seed, x - 8, x + 8, z - 8, z + 8, p -> {
                        if (p.material() == GeoWorldAmethystGeodes.Material.BUDDING_AMETHYST) {
                            budding[0] = true;
                        }
                    }
                );
                geodePresent = budding[0];
            }
        }
        if (!geodePresent) {
            throw new AssertionError("no physically embedded amethyst geode in the finite GeoWorld world");
        }

        // Replanning as one region or as two adjacent chunks must return the
        // same world-space ore blocks, independent of chunk request order.
        Map<GeoWorldMineralVeins.Placement, Integer> joined = new HashMap<>();
        GeoWorldMineralVeins.plan(model, seed, -16, -1, 0, 15,
            p -> joined.merge(p, 1, Integer::sum));
        GeoWorldMineralVeins.plan(model, seed, 0, 15, 0, 15,
            p -> joined.merge(p, 1, Integer::sum));
        Map<GeoWorldMineralVeins.Placement, Integer> whole = new HashMap<>();
        GeoWorldMineralVeins.plan(model, seed, -16, 15, 0, 15,
            p -> whole.merge(p, 1, Integer::sum));
        if (!joined.equals(whole)) {
            throw new AssertionError("ore plan differs across chunk boundaries");
        }
        System.out.printf(
            "GeoWorld mineral veins PASS seed=%d oreFamilies=8 " +
            "stoneVariants=8 deepVariants=8 amethystGeode=PASS " +
            "geologicalBias=PASS solidHosts=%d chunkOrder=PASS%n", seed, verified
        );
    }
}
