package geoworld.minecraft.v1_21_1common.worldgen;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.v1_21_1common.Minecraft1_21_1Profile;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.runtime.GeoWorldV1;

import java.nio.file.Path;
import java.util.ArrayDeque;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

/** Measures final surface biomes at the persisted macro-node grid. */
public final class GeoWorldBiomeDistributionAudit {
    private static final long[] DEFAULT_SEEDS = {-1818694639628404530L, 123456L, 10400L, 71L};

    private static final List<String> ECOLOGICAL_FORESTS = List.of(
        "Boreal Forest", "Temperate Forest", "Subtropical Forest",
        "Tropical Monsoon Forest", "Tropical Rainforest"
    );
    private static final List<String> FOREST_BIOMES = List.of(
        "forest", "flower_forest", "birch_forest", "old_growth_birch_forest",
        "dark_forest", "windswept_forest", "taiga", "snowy_taiga",
        "old_growth_pine_taiga", "old_growth_spruce_taiga", "cherry_grove", "grove",
        "jungle", "sparse_jungle", "bamboo_jungle"
    );
    private static final List<String> LAND_BIOMES = List.of(
        "forest", "birch_forest", "dark_forest", "jungle", "sparse_jungle",
        "bamboo_jungle", "plains", "savanna"
    );

    private GeoWorldBiomeDistributionAudit() {}

    public static void main(String[] args) throws Exception {
        var realizer = new GeoWorldV26BiomeRealizer(Minecraft1_21_1Profile.INSTANCE);
        long[] seeds = args.length == 0
            ? DEFAULT_SEEDS
            : Arrays.stream(args).mapToLong(Long::parseLong).toArray();
        String baselineDir = System.getProperty("geoworld.biomeAudit.baselineDir");

        for (long seed : seeds) {
            GeoWorldV1 current = GeoWorldEngineBridge.generate(
                seed, GeoWorldCreationConfig.defaults()
            ).model();
            if (baselineDir != null) {
                var baseline = GeoWorldV1.load(Path.of(baselineDir, "seed_" + seed + ".geoworld"));
                if (!"geoworld-rules-v1.1.26".equals(baseline.manifest().generationRulesVersion()) ||
                    baseline.manifest().seed() != seed ||
                    !baseline.manifest().geometry().equals(current.manifest().geometry())) {
                    throw new IllegalArgumentException("Unequal baseline seed, rules or geometry: " + seed);
                }
                audit(baseline, realizer);
            }
            audit(current, realizer);
        }
    }

    private static void audit(GeoWorldV1 model, GeoWorldV26BiomeRealizer realizer) {
        var g = model.manifest().geometry();
        OceanCell[] ocean = new OceanCell[g.nodeCount()];
        var dryLand = new Counts();
        var ecology = new Counts();
        boolean reefRules = "geoworld-rules-v1.1.27".equals(
            model.manifest().generationRulesVersion()
        );

        for (int z = 0; z < g.height(); z++) {
            double worldZ = g.originZ() + z * g.spacingZ();
            for (int x = 0; x < g.width(); x++) {
                double worldX = g.originX() + x * g.spacingX();
                var sample = model.sample(worldX, worldZ);
                String biome = realizer.surfaceBiomeIdFromSample(model, worldX, 319, worldZ, sample);
                if ("Ocean".equals(sample.waterKind())) {
                    double sst = model.sampleSeaSurfaceTemperature(worldX, worldZ);
                    boolean warmFamily = "Warm Ocean".equals(sample.biomeFamily());
                    if (reefRules && "minecraft:warm_ocean".equals(biome) &&
                        (!warmFamily || sample.coastalSeaReceiver() || sample.riverWeight() >= 0.12 ||
                            sample.waterDepth() < 0.008 || sample.waterDepth() > 0.055 || !(sst >= 24.0))) {
                        throw new AssertionError("Reef outside habitat at " + worldX + ", " + worldZ);
                    }
                    ocean[z * g.width() + x] = new OceanCell(
                        biome.substring("minecraft:".length()), warmFamily, sst
                    );
                } else if ("None".equals(sample.waterKind())) {
                    dryLand.add(biome.substring("minecraft:".length()));
                    ecology.add(sample.ecoRegion());
                }
            }
        }

        int[] distance = oceanDistance(g.width(), g.height(), ocean);
        var all = new OceanCounts();
        var near = new OceanCounts();
        var coast = new OceanCounts();
        var offshore = new OceanCounts();
        for (int i = 0; i < ocean.length; i++) {
            if (ocean[i] == null) {
                continue;
            }
            all.add(ocean[i]);
            if (distance[i] == 1) {
                near.add(ocean[i]);
            }
            if (distance[i] <= 3) {
                coast.add(ocean[i]);
            } else if (distance[i] >= 6) {
                offshore.add(ocean[i]);
            }
        }

        System.out.printf(
            Locale.ROOT,
            "SLICEF seed=%d rules=%s dryLand=%d forestLike=%.2f%% ecoForest=%.2f%%%n",
            model.manifest().seed(), model.manifest().generationRulesVersion(),
            dryLand.total, dryLand.share(FOREST_BIOMES), ecology.share(ECOLOGICAL_FORESTS)
        );
        dryLand.report("  Biomes", LAND_BIOMES);
        ecology.report("  Ecology", ECOLOGICAL_FORESTS);
        all.report("all");
        near.report("coast_1");
        coast.report("coast_1_3");
        offshore.report("offshore_6_plus");
    }

    private static int[] oceanDistance(int width, int height, OceanCell[] ocean) {
        int[] distance = new int[ocean.length];
        Arrays.fill(distance, Integer.MAX_VALUE);
        var queue = new ArrayDeque<Integer>();
        for (int i = 0; i < ocean.length; i++) {
            if (ocean[i] == null) {
                distance[i] = 0;
                queue.addLast(i);
            }
        }
        while (!queue.isEmpty()) {
            int i = queue.removeFirst();
            int x = i % width;
            int z = i / width;
            int next = distance[i] + 1;
            if (x > 0) {
                relax(i - 1, next, distance, queue);
            }
            if (x + 1 < width) {
                relax(i + 1, next, distance, queue);
            }
            if (z > 0) {
                relax(i - width, next, distance, queue);
            }
            if (z + 1 < height) {
                relax(i + width, next, distance, queue);
            }
        }
        return distance;
    }

    private static void relax(int index, int next, int[] distance, ArrayDeque<Integer> queue) {
        if (next < distance[index]) {
            distance[index] = next;
            queue.addLast(index);
        }
    }

    private static double pct(int count, int total) {
        return total > 0 ? 100.0 * count / total : 0.0;
    }

    private static final class Counts {
        private final Map<String, Integer> names = new HashMap<>();
        private int total;

        void add(String name) {
            names.merge(name, 1, Integer::sum);
            total++;
        }

        double share(List<String> selected) {
            int count = selected.stream().mapToInt(name -> names.getOrDefault(name, 0)).sum();
            return pct(count, total);
        }

        void report(String label, List<String> selected) {
            System.out.print(label);
            for (String name : selected) {
                System.out.printf(Locale.ROOT, " %s=%.2f%%", name, pct(names.getOrDefault(name, 0), total));
            }
            System.out.println();
        }
    }

    private record OceanCell(String biome, boolean warmFamily, double sst) {}

    private static final class OceanCounts {
        private static final List<String> BIOMES = List.of(
            "warm_ocean", "lukewarm_ocean", "deep_lukewarm_ocean"
        );
        private final Counts counts = new Counts();
        private int warmFamily;
        private int withSst;
        private int warmSst;

        void add(OceanCell cell) {
            counts.add(cell.biome());
            if (cell.warmFamily()) {
                warmFamily++;
            }
            if (Double.isFinite(cell.sst())) {
                withSst++;
                if (cell.sst() >= 24.0) {
                    warmSst++;
                }
            }
        }

        void report(String region) {
            String sst = withSst == 0
                ? "unavailable"
                : String.format(Locale.ROOT, "%.2f%%", pct(warmSst, withSst));
            counts.report(String.format(
                Locale.ROOT, "  Ocean %s cells=%d familyWarm=%.2f%% sst24C=%s",
                region, counts.total, pct(warmFamily, counts.total), sst
            ), BIOMES);
        }
    }
}
