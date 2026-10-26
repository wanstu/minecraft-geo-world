package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.util.Arrays;

/**
 * Repeatable model-side chunk generation microbenchmark.
 * Keeps model loading out of timings and measures the actual three planning
 * subsystems independently. No Minecraft client or world writes required.
 */
public final class GeoWorldV26ChunkPlanningBenchmark {
    private GeoWorldV26ChunkPlanningBenchmark() {}

    private static volatile long sink;

    private static GeoWorldV26BiomeRealizer BIOMES;
    private static GeoWorldShorelinePlanner SHORELINE;
    private static GeoWorldMineralPlanner MINERALS;

    public static void run(
        MinecraftVersionProfile profile,
        String[] args
    ) throws Exception {
        BIOMES = new GeoWorldV26BiomeRealizer(profile);
        SHORELINE = new GeoWorldShorelinePlanner(profile.heightMapping());
        MINERALS = new GeoWorldMineralPlanner(profile.heightMapping());
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
            GeoWorldShorelinePlanner.ChunkPlan shore =
                SHORELINE.planChunkModel(model, bx, bz);
            shoreTime[i] = System.nanoTime() - start;
            start = System.nanoTime();
            for (int dz = 0; dz < 16; dz++) {
                for (int dx = 0; dx < 16; dx++) {
                    var column = shore.column(dx, dz);
                    String id = BIOMES.biomeIdFromSample(
                        model, bx + dx, column.groundBoundary() - 1,
                        bz + dz, column.sample()
                    );
                    sink += id.length();
                }
            }
            biomeTime[i] = System.nanoTime() - start;
            start = System.nanoTime();
            MINERALS.plan(
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
