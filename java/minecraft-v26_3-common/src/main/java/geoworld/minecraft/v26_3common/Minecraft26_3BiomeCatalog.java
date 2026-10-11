package geoworld.minecraft.v26_3common;

import java.util.ArrayList;
import java.util.EnumSet;
import java.util.List;
import java.util.Set;

/**
 * Tagged vanilla-biome catalogue.
 *
 * GeoWorld describes a physical region first (climate, vegetation, landform,
 * soil/wetness). Vanilla biomes describe ecological/rendering archetypes.
 * Matching those two semantic tag sets lets several valid vanilla biomes
 * coexist inside one GeoWorld region without hard-coding a single biome per
 * family.
 */
public final class Minecraft26_3BiomeCatalog {
    public enum Tag {
        POLAR,
        COLD,
        COOL,
        TEMPERATE,
        WARM,
        HOT,

        ARID,
        DRY,
        MESIC,
        HUMID,
        WET,

        FOREST,
        GRASSLAND,
        SHRUBLAND,
        DESERT,
        TUNDRA,
        ALPINE,
        TAIGA,
        JUNGLE,
        SAVANNA,
        WETLAND,

        PLAINS,
        HILLS,
        FOOTHILLS,
        MOUNTAIN,
        PLATEAU,
        CANYON,
        FLOODPLAIN,

        SNOWY,
        ROCKY,
        SANDY,
        PODZOLIC,
        ALLUVIAL,
        PEATY,
        MONSOON,
        COASTAL
    }

    public record Region(
        Set<Tag> tags,
        double annualMeanTempC,
        double annualPrecipMM,
        double monsoonIndex
    ) {
        public Region {
            tags = Set.copyOf(tags);
        }
    }

    private record Candidate(
        String id,
        Set<Tag> requiredAll,
        Set<Tag> requiredAny,
        Set<Tag> preferred,
        Set<Tag> avoided,
        double tempMin,
        double tempMax,
        double precipMin,
        double precipMax,
        double baseWeight
    ) {}

    private record Scored(Candidate candidate, double score) {}

    private static final List<Candidate> CANDIDATES = List.of(
        c("minecraft:snowy_plains", all(Tag.TUNDRA), none(),
            tags(Tag.POLAR, Tag.COLD, Tag.SNOWY, Tag.PLAINS), none(),
            -40, 3, 0, 1400, 1.00),
        c("minecraft:ice_spikes", all(Tag.TUNDRA), none(),
            tags(Tag.POLAR, Tag.SNOWY, Tag.PLAINS), none(),
            -40, -4, 0, 900, 0.36),

        c("minecraft:plains", all(Tag.GRASSLAND), none(),
            tags(Tag.PLAINS, Tag.TEMPERATE, Tag.MESIC), tags(Tag.WET),
            2, 23, 350, 1500, 1.20),
        c("minecraft:sunflower_plains", all(Tag.GRASSLAND), none(),
            tags(Tag.PLAINS, Tag.WARM, Tag.MESIC), tags(Tag.COLD, Tag.WET),
            8, 26, 450, 1450, 0.42),
        c("minecraft:meadow", all(Tag.GRASSLAND),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.COOL, Tag.TEMPERATE, Tag.HUMID), tags(Tag.ARID),
            -2, 18, 500, 1800, 0.78),

        c("minecraft:forest", all(Tag.FOREST), none(),
            tags(Tag.TEMPERATE, Tag.MESIC), tags(Tag.HOT, Tag.POLAR),
            3, 21, 500, 1700, 1.15),
        c("minecraft:flower_forest", all(Tag.FOREST), none(),
            tags(Tag.TEMPERATE, Tag.HUMID, Tag.PLAINS), tags(Tag.ARID),
            6, 20, 700, 1700, 0.34),
        c("minecraft:dappled_forest", all(Tag.FOREST), none(),
            tags(Tag.TEMPERATE, Tag.HUMID), tags(Tag.ARID, Tag.POLAR),
            5, 22, 650, 1800, 0.62),
        c("minecraft:dark_forest", all(Tag.FOREST), none(),
            tags(Tag.HUMID, Tag.WET, Tag.TEMPERATE), tags(Tag.ARID),
            4, 21, 1050, 2600, 0.55),
        c("minecraft:pale_garden", all(Tag.FOREST), none(),
            tags(Tag.COOL, Tag.HUMID, Tag.WET), tags(Tag.HOT, Tag.ARID),
            0, 16, 1100, 2800, 0.13),
        c("minecraft:birch_forest", all(Tag.FOREST), none(),
            tags(Tag.TEMPERATE, Tag.WARM, Tag.MESIC), tags(Tag.POLAR),
            6, 25, 500, 1700, 0.90),
        c("minecraft:old_growth_birch_forest", all(Tag.FOREST),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.TEMPERATE, Tag.HUMID), tags(Tag.ARID),
            5, 22, 700, 1900, 0.46),
        c("minecraft:cherry_grove", all(Tag.FOREST),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.COOL, Tag.TEMPERATE, Tag.HUMID), tags(Tag.ARID, Tag.HOT),
            3, 18, 650, 1700, 0.30),
        c("minecraft:windswept_forest", all(Tag.FOREST),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.COOL, Tag.ROCKY), none(),
            -4, 18, 350, 1800, 0.54),

        c("minecraft:taiga", all(Tag.FOREST, Tag.TAIGA), none(),
            tags(Tag.COLD, Tag.COOL, Tag.MESIC), tags(Tag.HOT),
            -8, 12, 350, 1700, 1.05),
        c("minecraft:snowy_taiga", all(Tag.FOREST, Tag.TAIGA), none(),
            tags(Tag.COLD, Tag.SNOWY), tags(Tag.WARM, Tag.HOT),
            -20, 2, 250, 1400, 0.65),
        c("minecraft:old_growth_pine_taiga", all(Tag.FOREST, Tag.TAIGA),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.COOL, Tag.DRY, Tag.PODZOLIC), tags(Tag.WET),
            -5, 13, 300, 1000, 0.50),
        c("minecraft:old_growth_spruce_taiga", all(Tag.FOREST, Tag.TAIGA),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.COOL, Tag.HUMID, Tag.PODZOLIC), tags(Tag.ARID),
            -5, 13, 700, 1900, 0.58),

        c("minecraft:jungle", all(Tag.FOREST, Tag.JUNGLE), none(),
            tags(Tag.HOT, Tag.HUMID, Tag.WET), tags(Tag.COLD, Tag.ARID),
            18, 40, 1200, 5000, 1.05),
        c("minecraft:sparse_jungle", all(Tag.FOREST, Tag.JUNGLE), none(),
            tags(Tag.HOT, Tag.MESIC, Tag.DRY), tags(Tag.COLD),
            17, 40, 650, 1900, 0.76),
        c("minecraft:bamboo_jungle", all(Tag.FOREST, Tag.JUNGLE), none(),
            tags(Tag.HOT, Tag.WET, Tag.MONSOON), tags(Tag.ARID),
            19, 40, 1500, 5500, 0.55),

        c("minecraft:savanna", all(Tag.SAVANNA), none(),
            tags(Tag.HOT, Tag.DRY, Tag.PLAINS), tags(Tag.WET),
            16, 38, 180, 1000, 1.05),
        c("minecraft:savanna_plateau", all(Tag.SAVANNA),
            any(Tag.PLATEAU, Tag.HILLS, Tag.FOOTHILLS),
            tags(Tag.HOT, Tag.DRY), tags(Tag.WET),
            15, 38, 150, 1000, 0.72),
        c("minecraft:windswept_savanna", all(Tag.SAVANNA),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.HOT, Tag.DRY, Tag.ROCKY), tags(Tag.WET),
            14, 38, 120, 950, 0.62),

        c("minecraft:desert", all(Tag.DESERT), none(),
            tags(Tag.HOT, Tag.ARID, Tag.SANDY, Tag.PLAINS), tags(Tag.WET),
            10, 45, 0, 500, 1.15),
        c("minecraft:badlands", all(Tag.DESERT),
            any(Tag.PLATEAU, Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.ARID, Tag.ROCKY), tags(Tag.WET),
            8, 42, 0, 650, 0.85),
        c("minecraft:eroded_badlands", all(Tag.DESERT),
            any(Tag.CANYON, Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.ARID, Tag.ROCKY), tags(Tag.WET),
            8, 42, 0, 550, 0.58),
        c("minecraft:wooded_badlands", all(Tag.DESERT),
            any(Tag.PLATEAU, Tag.HILLS, Tag.FOOTHILLS),
            tags(Tag.DRY, Tag.SHRUBLAND), tags(Tag.WET),
            6, 35, 120, 750, 0.48),

        c("minecraft:swamp", all(Tag.WETLAND), none(),
            tags(Tag.WET, Tag.FLOODPLAIN), tags(Tag.ARID, Tag.POLAR),
            2, 30, 800, 5000, 1.05),
        c("minecraft:mangrove_swamp", all(Tag.WETLAND, Tag.COASTAL), none(),
            tags(Tag.HOT, Tag.WET, Tag.MONSOON), tags(Tag.COLD, Tag.ARID),
            18, 40, 900, 5500, 0.82),

        c("minecraft:grove", all(Tag.ALPINE),
            any(Tag.FOOTHILLS, Tag.HILLS),
            tags(Tag.COLD, Tag.FOREST, Tag.SNOWY), tags(Tag.HOT),
            -12, 5, 400, 1800, 0.72),
        c("minecraft:snowy_slopes", all(Tag.ALPINE),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.COLD, Tag.SNOWY), tags(Tag.HOT),
            -20, 4, 150, 1800, 0.92),
        c("minecraft:jagged_peaks", all(Tag.ALPINE, Tag.MOUNTAIN), none(),
            tags(Tag.COLD, Tag.ROCKY, Tag.SNOWY), tags(Tag.HOT),
            -25, 2, 100, 1700, 0.82),
        c("minecraft:frozen_peaks", all(Tag.ALPINE, Tag.MOUNTAIN), none(),
            tags(Tag.POLAR, Tag.COLD, Tag.SNOWY), tags(Tag.WARM),
            -30, -2, 100, 1800, 0.64),
        c("minecraft:stony_peaks", all(Tag.ALPINE),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.ROCKY, Tag.COOL, Tag.TEMPERATE), tags(Tag.WET),
            -2, 16, 100, 1300, 0.72),
        c("minecraft:windswept_gravelly_hills", none(),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.ROCKY, Tag.DRY), tags(Tag.WET),
            -4, 24, 100, 1200, 0.48),
        c("minecraft:windswept_hills", none(),
            any(Tag.HILLS, Tag.FOOTHILLS, Tag.MOUNTAIN),
            tags(Tag.COOL, Tag.TEMPERATE), tags(Tag.HOT, Tag.WET),
            -6, 20, 250, 1500, 0.58)
    );

    private Minecraft26_3BiomeCatalog() {}

    public static Set<String> candidateBiomeIds() {
        return Set.copyOf(
            CANDIDATES.stream().map(Candidate::id).toList()
        );
    }

    public static String choose(
        Region region,
        long worldSeed,
        double x,
        double z,
        String fallback
    ) {
        ArrayList<Scored> scored = new ArrayList<>();
        double best = Double.NEGATIVE_INFINITY;

        for (Candidate candidate : CANDIDATES) {
            double score = score(region, candidate);
            if (!Double.isFinite(score)) {
                continue;
            }
            scored.add(new Scored(candidate, score));
            best = Math.max(best, score);
        }
        if (scored.isEmpty()) {
            return fallback;
        }

        final double band = 1.65;
        final double sharpness = 1.35;
        double total = 0.0;
        ArrayList<Scored> finalists = new ArrayList<>();
        for (Scored item : scored) {
            if (item.score() < best - band) {
                continue;
            }
            finalists.add(item);
            total += item.candidate().baseWeight() *
                Math.exp((item.score() - best) * sharpness);
        }
        if (finalists.isEmpty() || total <= 0.0) {
            return fallback;
        }

        // Smooth region-scale selector: deterministic for the same seed, but
        // spatially coherent so oak/birch/etc. form real patches instead of
        // block-by-block salt-and-pepper noise.
        double chooser =
            0.72 * smoothValueNoise(
                worldSeed ^ 0x42494F4D455F5441L,
                x,
                z,
                360.0
            ) +
            0.28 * smoothValueNoise(
                worldSeed ^ 0x56415249414E5453L,
                x,
                z,
                150.0
            );
        double target = clamp(chooser, 0.0, 1.0) * total;
        double cursor = 0.0;
        for (Scored item : finalists) {
            cursor += item.candidate().baseWeight() *
                Math.exp((item.score() - best) * sharpness);
            if (target <= cursor) {
                return item.candidate().id();
            }
        }
        return finalists.getLast().candidate().id();
    }

    private static double score(Region region, Candidate candidate) {
        Set<Tag> tags = region.tags();
        // A dry, rocky desert hillside belongs to the badlands/desert
        // families, not windswept temperate grassland. The old generic
        // windswept-gravelly-hills candidate had no biome-family constraint
        // and regularly displaced true desert biomes in hot arid terrain.
        if (tags.contains(Tag.DESERT) &&
            (candidate.id().equals("minecraft:windswept_hills") ||
             candidate.id().equals("minecraft:windswept_gravelly_hills"))) {
            return Double.NEGATIVE_INFINITY;
        }
        if (!tags.containsAll(candidate.requiredAll())) {
            return Double.NEGATIVE_INFINITY;
        }
        if (!candidate.requiredAny().isEmpty() &&
            candidate.requiredAny().stream().noneMatch(tags::contains)) {
            return Double.NEGATIVE_INFINITY;
        }

        double score = 0.0;
        for (Tag tag : candidate.preferred()) {
            if (tags.contains(tag)) {
                score += 1.05;
            }
        }
        for (Tag tag : candidate.avoided()) {
            if (tags.contains(tag)) {
                score -= 1.25;
            }
        }

        score += rangeScore(
            region.annualMeanTempC(),
            candidate.tempMin(),
            candidate.tempMax(),
            8.0
        );
        score += rangeScore(
            region.annualPrecipMM(),
            candidate.precipMin(),
            candidate.precipMax(),
            650.0
        );
        return score;
    }

    private static double rangeScore(
        double value,
        double min,
        double max,
        double falloff
    ) {
        if (value >= min && value <= max) {
            double center = (min + max) * 0.5;
            double radius = Math.max(1.0e-9, (max - min) * 0.5);
            double centrality = 1.0 - Math.min(
                1.0,
                Math.abs(value - center) / radius
            );
            return 0.65 + 0.55 * centrality;
        }
        double distance = value < min ? min - value : value - max;
        return -Math.min(4.0, distance / Math.max(1.0e-9, falloff));
    }

    private static Candidate c(
        String id,
        Set<Tag> requiredAll,
        Set<Tag> requiredAny,
        Set<Tag> preferred,
        Set<Tag> avoided,
        double tempMin,
        double tempMax,
        double precipMin,
        double precipMax,
        double baseWeight
    ) {
        return new Candidate(
            id,
            Set.copyOf(requiredAll),
            Set.copyOf(requiredAny),
            Set.copyOf(preferred),
            Set.copyOf(avoided),
            tempMin,
            tempMax,
            precipMin,
            precipMax,
            baseWeight
        );
    }

    private static Set<Tag> all(Tag... tags) {
        return tags(tags);
    }

    private static Set<Tag> any(Tag... tags) {
        return tags(tags);
    }

    private static Set<Tag> none() {
        return Set.of();
    }

    private static Set<Tag> tags(Tag... tags) {
        if (tags.length == 0) {
            return Set.of();
        }
        EnumSet<Tag> set = EnumSet.noneOf(Tag.class);
        for (Tag tag : tags) {
            set.add(tag);
        }
        return set;
    }

    private static double smoothValueNoise(
        long seed,
        double x,
        double z,
        double scale
    ) {
        double gx = x / scale;
        double gz = z / scale;
        int x0 = (int)Math.floor(gx);
        int z0 = (int)Math.floor(gz);
        double tx = smoothstep(gx - x0);
        double tz = smoothstep(gz - z0);

        double v00 = hashUnit(seed, x0, z0);
        double v10 = hashUnit(seed, x0 + 1, z0);
        double v01 = hashUnit(seed, x0, z0 + 1);
        double v11 = hashUnit(seed, x0 + 1, z0 + 1);
        double a = v00 + (v10 - v00) * tx;
        double b = v01 + (v11 - v01) * tx;
        return a + (b - a) * tz;
    }

    private static double hashUnit(long seed, int x, int z) {
        long h = seed;
        h ^= (long)x * 0x9E3779B97F4A7C15L;
        h ^= (long)z * 0xC2B2AE3D27D4EB4FL;
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        h ^= h >>> 31;
        return (h >>> 11) * 0x1.0p-53;
    }

    private static double smoothstep(double t) {
        t = clamp(t, 0.0, 1.0);
        return t * t * (3.0 - 2.0 * t);
    }

    private static double clamp(double value, double lo, double hi) {
        return Math.max(lo, Math.min(hi, value));
    }
}
