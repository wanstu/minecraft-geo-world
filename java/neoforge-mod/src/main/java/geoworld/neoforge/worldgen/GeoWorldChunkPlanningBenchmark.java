package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3BiomeRealizer;
import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.util.Arrays;

/**
 * Repeatable model-side chunk generation microbenchmark.
 * Keeps model loading out of timings and measures the actual three planning
 * subsystems independently. No Minecraft client or world writes required.
 */
public final class GeoWorldChunkPlanningBenchmark {
    private GeoWorldChunkPlanningBenchmark() {}

    private static volatile long sink;

    public static void main(String[] args) throws Exception {
        long seed = args.length > 0 ? Long.parseLong(args[0]) : 123456L;
        GeoWorldV1 model = GeoWorldEngineBridge.generate(
            seed, GeoWorldCreationConfig.defaults()
        ).model();
        int[][] positions = new int[48][2];
        for (int i = 0; i < positions.length; i++) {
            // Explicit fixed world coordinates; a mix of sea/land, away from
            // large teleports and independent of chunk generation ordering.
            positions[i][0] = (-32 + (i % 8) * 8) * 16;
            positions[i][1] = (-24 + (i / 8) * 8) * 16;
        }
        for (int i = 0; i < 3; i++) {
            measure(model, seed, positions, false);
        }
        measure(model, seed, positions, true);
    }

    private static void measure(
        GeoWorldV1 model, long seed, int[][] positions, boolean report
    ) {
        long[] shoreTime = new long[positions.length];
        long[] biomeTime = new long[positions.length];
        long[] mineralTime = new long[positions.length];
        long[] planCount = {0};
        for (int i = 0; i < positions.length; i++) {
            int bx = positions[i][0];
            int bz = positions[i][1];
            long start = System.nanoTime();
            GeoWorldShoreline.ChunkPlan shore =
                GeoWorldShoreline.planChunk(model, bx, bz);
            shoreTime[i] = System.nanoTime() - start;
            start = System.nanoTime();
            for (int dz = 0; dz < 16; dz++) {
                for (int dx = 0; dx < 16; dx++) {
                    var column = shore.column(dx, dz);
                    String id = Minecraft26_3BiomeRealizer.biomeId(
                        model, bx + dx, column.groundBoundary() - 1, bz + dz
                    );
                    sink += id.length();
                }
            }
            biomeTime[i] = System.nanoTime() - start;
            start = System.nanoTime();
            GeoWorldMineralVeins.plan(
                model, seed, bx, bx + 15, bz, bz + 15,
                placement -> planCount[0]++
            );
            mineralTime[i] = System.nanoTime() - start;
        }
        if (report) {
            stats("shoreline", shoreTime);
            stats("biomes", biomeTime);
            stats("minerals", mineralTime);
            System.out.printf("chunks=%d plans=%d sink=%d%n",
                positions.length, planCount[0], sink);
        }
    }

    private static void stats(String name, long[] time) {
        Arrays.sort(time);
        long sum = 0;
        for (long ns : time) sum += ns;
        System.out.printf(
            "GeoWorld benchmark %-10s average_ms=%.3f p50_ms=%.3f p95_ms=%.3f%n",
            name, sum / 1e6 / time.length,
            time[time.length / 2] / 1e6,
            time[(int)(time.length * .95)] / 1e6
        );
    }
}
