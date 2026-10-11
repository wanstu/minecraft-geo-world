package geoworld.neoforge.worldgen;

import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.LongAdder;

/**
 * Opt-in aggregate profiling for real Minecraft chunk generation.
 * Add -Dgeoworld.profileChunks=true to the client JVM arguments.
 * Nothing is logged or timed while the option is disabled.
 */
final class GeoWorldChunkPerformance {
    private static final boolean ENABLED =
        Boolean.getBoolean("geoworld.profileChunks");
    private static final System.Logger LOGGER =
        System.getLogger("GeoWorld.ChunkPerformance");
    private static final AtomicLong CHUNKS = new AtomicLong();
    private static final LongAdder[] STAGES = {
        new LongAdder(), new LongAdder(), new LongAdder(),
        new LongAdder(), new LongAdder(), new LongAdder()
    };

    static {
        for (LongAdder stage : STAGES) stage.reset();
    }

    private GeoWorldChunkPerformance() {}

    static boolean enabled() {
        return ENABLED;
    }

    static void record(
        long model, long shore, long biome, long blocks,
        long minerals, long geodes
    ) {
        if (!ENABLED) return;
        long[] timings = {model, shore, biome, blocks, minerals, geodes};
        for (int i = 0; i < timings.length; i++) STAGES[i].add(timings[i]);
        long count = CHUNKS.incrementAndGet();
        if (count % 128 == 0) {
            StringBuilder message = new StringBuilder(
                "GeoWorld generated ").append(count).append(" chunks; ")
                .append("average milliseconds per chunk: ");
            String[] names = {"model", "shoreline", "biomes", "voxel",
                "minerals", "geodes"};
            for (int i = 0; i < timings.length; i++) {
                if (i != 0) message.append(", ");
                message.append(names[i]).append('=')
                    .append(String.format(java.util.Locale.ROOT,
                        "%.3f", STAGES[i].sum() / (1e6 * count)));
            }
            LOGGER.log(System.Logger.Level.INFO, message.toString());
        }
    }
}
