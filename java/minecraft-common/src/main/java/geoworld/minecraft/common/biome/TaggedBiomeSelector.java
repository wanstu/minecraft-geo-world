package geoworld.minecraft.common.biome;

import java.util.ArrayList;
import java.util.List;
import java.util.Set;
import java.util.function.BiPredicate;

public final class TaggedBiomeSelector {
    private TaggedBiomeSelector() {}

    public static Set<String> candidateBiomeIds(
        List<TaggedBiomeCandidate> candidates
    ) {
        return Set.copyOf(
            candidates.stream().map(TaggedBiomeCandidate::id).toList()
        );
    }

    public static String choose(
        List<TaggedBiomeCandidate> candidates,
        GeoWorldBiomeRegion region,
        long worldSeed,
        double x,
        double z,
        String fallback,
        BiPredicate<GeoWorldBiomeRegion, TaggedBiomeCandidate> allowCandidate
    ) {
        SelectorScratch scratch = SELECTOR_SCRATCH.get();
        scratch.ensureCapacity(candidates.size());
        int scoredCount = 0;
        double best = Double.NEGATIVE_INFINITY;

        for (int i = 0, n = candidates.size(); i < n; i++) {
            TaggedBiomeCandidate candidate = candidates.get(i);
            if (!allowCandidate.test(region, candidate)) continue;
            double candidateScore = score(region, candidate);
            if (!Double.isFinite(candidateScore)) continue;
            scratch.candidates[scoredCount] = candidate;
            scratch.scores[scoredCount] = candidateScore;
            scoredCount++;
            best = Math.max(best, candidateScore);
        }
        if (scoredCount == 0) return fallback;

        final double band = 1.65;
        final double sharpness = 1.35;
        double total = 0.0;
        int finalistCount = 0;
        TaggedBiomeCandidate lastFinalist = null;
        for (int i = 0; i < scoredCount; i++) {
            double candidateScore = scratch.scores[i];
            if (candidateScore < best - band) continue;
            TaggedBiomeCandidate candidate = scratch.candidates[i];
            lastFinalist = candidate;
            finalistCount++;
            double weight = candidate.baseWeight() *
                Math.exp((candidateScore - best) * sharpness);
            scratch.weights[i] = weight;
            total += weight;
        }
        if (finalistCount == 0 || total <= 0.0) return fallback;

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
        for (int i = 0; i < scoredCount; i++) {
            double candidateScore = scratch.scores[i];
            if (candidateScore < best - band) continue;
            TaggedBiomeCandidate candidate = scratch.candidates[i];
            cursor += scratch.weights[i];
            if (target <= cursor) return candidate.id();
        }
        return lastFinalist == null ? fallback : lastFinalist.id();
    }
    private static final ThreadLocal<SelectorScratch> SELECTOR_SCRATCH =
        ThreadLocal.withInitial(SelectorScratch::new);

    private static final class SelectorScratch {
        TaggedBiomeCandidate[] candidates = new TaggedBiomeCandidate[16];
        double[] scores = new double[16];
        double[] weights = new double[16];

        void ensureCapacity(int capacity) {
            if (capacity <= candidates.length) {
                return;
            }
            int next = Math.max(capacity, candidates.length * 2);
            candidates = java.util.Arrays.copyOf(candidates, next);
            scores = java.util.Arrays.copyOf(scores, next);
            weights = java.util.Arrays.copyOf(weights, next);
        }
    }
    private static double score(
        GeoWorldBiomeRegion region,
        TaggedBiomeCandidate candidate
    ) {
        Set<GeoWorldBiomeTag> tags = region.tags();
        if (!tags.containsAll(candidate.requiredAll())) {
            return Double.NEGATIVE_INFINITY;
        }
        if (!candidate.requiredAny().isEmpty() &&
            !containsAny(tags, candidate.requiredAny())) {
            return Double.NEGATIVE_INFINITY;
        }

        double score = 0.0;
        for (GeoWorldBiomeTag tag : candidate.preferred()) {
            if (tags.contains(tag)) score += 1.05;
        }
        for (GeoWorldBiomeTag tag : candidate.avoided()) {
            if (tags.contains(tag)) score -= 1.25;
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

    private static boolean containsAny(
        Set<GeoWorldBiomeTag> tags,
        Set<GeoWorldBiomeTag> requiredAny
    ) {
        for (GeoWorldBiomeTag tag : requiredAny) {
            if (tags.contains(tag)) {
                return true;
            }
        }
        return false;
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

    private record Scored(
        TaggedBiomeCandidate candidate,
        double score
    ) {}
}
