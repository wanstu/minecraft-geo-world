package geoworld.neoforge.worldgen;

import geoworld.neoforge.GeoWorldMod;

import java.io.IOException;
import java.io.InputStream;
import java.util.Locale;
import java.util.Properties;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.LongAdder;

/**
 * Reports real GeoWorld chunk generation timings to Minecraft latest.log.
 *
 * HMCL development JARs enable profiling automatically via a build resource.
 * Release builds default to off. The JVM property geoworld.profileChunks
 * always overrides the build resource (true or false).
 *
 * This class is invoked only by the GeoWorld chunk generator, never for
 * vanilla or third-party worlds.
 */
final class GeoWorldChunkPerformance {
    private static final int FIRST_REPORT_CHUNKS = 32;
    private static final int REPORT_EVERY_CHUNKS = 128;
    private static final long MIN_REPORT_INTERVAL_NS = 30_000_000_000L;

    private static final boolean ENABLED = enabledByConfiguration();
    private static final AtomicLong CHUNKS = new AtomicLong();
    private static final AtomicLong LAST_REPORT_NS = new AtomicLong();
    private static final LongAdder[] STAGES = {
        new LongAdder(), new LongAdder(), new LongAdder(),
        new LongAdder(), new LongAdder(), new LongAdder()
    };
    private static final String[] STAGE_NAMES = {
        "model", "shoreline", "biomes", "voxel", "minerals", "geodes"
    };

    private GeoWorldChunkPerformance() {}

    static boolean enabled() {
        return ENABLED;
    }

    private static boolean enabledByConfiguration() {
        String override = System.getProperty("geoworld.profileChunks");
        if (override != null) {
            return Boolean.parseBoolean(override);
        }
        try (InputStream in = GeoWorldChunkPerformance.class.getClassLoader()
            .getResourceAsStream("geoworld-build.properties")) {
            if (in == null) return false;
            Properties flags = new Properties();
            flags.load(in);
            return Boolean.parseBoolean(flags.getProperty("devBuild", "false"));
        } catch (IOException exception) {
            return false; // No profiling or client startup failure if absent.
        }
    }

    static void record(
        long model, long shore, long biome, long blocks,
        long minerals, long geodes
    ) {
        if (!ENABLED) return;
        STAGES[0].add(model);
        STAGES[1].add(shore);
        STAGES[2].add(biome);
        STAGES[3].add(blocks);
        STAGES[4].add(minerals);
        STAGES[5].add(geodes);

        long count = CHUNKS.incrementAndGet();
        if (count != FIRST_REPORT_CHUNKS &&
            count % REPORT_EVERY_CHUNKS != 0) {
            return;
        }
        long now = System.nanoTime();
        long previous = LAST_REPORT_NS.get();
        if (previous != 0 && now - previous < MIN_REPORT_INTERVAL_NS) {
            return;
        }
        if (!LAST_REPORT_NS.compareAndSet(previous, now)) {
            return; // One reporting thread only.
        }

        StringBuilder message = new StringBuilder("GeoWorld generated ")
            .append(count)
            .append(" chunks; average milliseconds per chunk: ");
        for (int i = 0; i < STAGES.length; i++) {
            if (i != 0) message.append(", ");
            message.append(STAGE_NAMES[i]).append('=')
                .append(String.format(Locale.ROOT, "%.3f",
                    STAGES[i].sum() / (1e6 * count)));
        }
        GeoWorldMod.LOGGER.info(message.toString());
    }
}
