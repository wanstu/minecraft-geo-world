package geoworld.minecraft.common.performance;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.io.InputStream;
import java.util.Locale;
import java.util.Properties;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.LongAdder;

/**
 * Reports real GeoWorld chunk generation timings to Minecraft latest.log.
 *
 * HMCL development JARs enable profiling automatically via a build resource.
 * Release builds default to off. The JVM property geoworld.profileChunks
 * always overrides the build resource (true or false).
 *
 * Runtime commands may temporarily enable/disable profiling without changing
 * the packaged default.
 *
 * This class is invoked only by the GeoWorld chunk generator, never for
 * vanilla or third-party worlds.
 */
public final class GeoWorldChunkPerformance {
    private static final Logger LOGGER = LoggerFactory.getLogger("GeoWorld");
    private static final int FIRST_REPORT_CHUNKS = 32;
    private static final int REPORT_EVERY_CHUNKS = 128;
    private static final long MIN_REPORT_INTERVAL_NS = 30_000_000_000L;

    private static final AtomicBoolean ENABLED =
        new AtomicBoolean(enabledByConfiguration());
    private static final AtomicLong CHUNKS = new AtomicLong();
    private static final AtomicLong LAST_REPORT_NS = new AtomicLong();
    private static final LongAdder[] STAGES = {
        new LongAdder(), new LongAdder(), new LongAdder(), new LongAdder(),
        new LongAdder(), new LongAdder(), new LongAdder(), new LongAdder()
    };
    private static final LongAdder FAST_TERRAIN_CHUNKS = new LongAdder();
    private static final String[] STAGE_NAMES = {
        "model", "shoreline", "biomes", "voxel",
        "structures", "caves", "minerals", "geodes"
    };

    private GeoWorldChunkPerformance() {}

    public static boolean enabled() {
        return ENABLED.get();
    }

    public static void setEnabled(boolean enabled) {
        ENABLED.set(enabled);
    }

    public static void reset() {
        CHUNKS.set(0L);
        LAST_REPORT_NS.set(0L);
        for (LongAdder stage : STAGES) {
            stage.reset();
        }
        FAST_TERRAIN_CHUNKS.reset();
    }

    public static String summary() {
        long count = CHUNKS.get();
        StringBuilder message = new StringBuilder()
            .append("enabled=")
            .append(enabled())
            .append(", chunks=")
            .append(count);
        if (count <= 0L) {
            return message.append(", no samples yet").toString();
        }

        message.append(", avg ms/chunk: ");
        for (int i = 0; i < STAGES.length; i++) {
            if (i != 0) message.append(", ");
            message.append(STAGE_NAMES[i]).append('=')
                .append(String.format(
                    Locale.ROOT,
                    "%.3f",
                    STAGES[i].sum() / (1e6 * count)
                ));
        }
        message.append(", fastTerrain=")
            .append(FAST_TERRAIN_CHUNKS.sum())
            .append('/')
            .append(count);
        return message.toString();
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
            return false;
        }
    }

    public static void record(
        long model, long shore, long biome, long blocks,
        long structures, long caves, long minerals, long geodes,
        boolean fastTerrain
    ) {
        if (!ENABLED.get()) return;
        STAGES[0].add(model);
        STAGES[1].add(shore);
        STAGES[2].add(biome);
        STAGES[3].add(blocks);
        STAGES[4].add(structures);
        STAGES[5].add(caves);
        STAGES[6].add(minerals);
        STAGES[7].add(geodes);
        if (fastTerrain) {
            FAST_TERRAIN_CHUNKS.increment();
        }

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
            return;
        }

        LOGGER.info(
            "GeoWorld generated {} chunks; average milliseconds per chunk: {}",
            count,
            summary()
        );
    }
}
