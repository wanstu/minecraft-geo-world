package geoworld.runtime;

/**
 * Deterministic coarse geology facts derived from the locked natural model.
 *
 * This is intentionally static world-generation geology, not a simulation of
 * real plate chemistry. The same functions are consumed by ore placement and
 * the read-only Query API so "ore province" has one source of truth.
 */
public final class GeoWorldGeology {
    private static final double BASE_PROVINCE_SIZE_BLOCKS = 1536.0;

    private GeoWorldGeology() {}

    public static String geologicalRegion(
        GeoWorldV1 model,
        double worldX,
        double worldZ
    ) {
        GeoWorldV1.RuntimeSample sample = model.sample(worldX, worldZ);
        String family = switch (sample.terrainVariant()) {
            case "Ocean" -> "oceanic-crust";
            case "Mountain", "Foothills" -> "orogenic-belt";
            case "Plateau", "Hills" -> "uplifted-block";
            case "Canyon" -> "exposed-crust";
            case "Floodplain", "Alluvial Plain", "Alluvial Fan" ->
                "sedimentary-basin";
            default -> "continental-interior";
        };
        long region = coarseRegionId(model, worldX, worldZ, 2.25);
        return family + ":" + Long.toUnsignedString(region, 16);
    }

    public static String oreProvince(
        GeoWorldV1 model,
        double worldX,
        double worldZ
    ) {
        long region = coarseRegionId(model, worldX, worldZ, 1.0);
        int dominant = (int)Math.floor(provinceStrength(model, worldX, worldZ) * 8.0);
        dominant = Math.max(0, Math.min(7, dominant));
        return "ore-" + dominant + "-" + Long.toUnsignedString(region, 16);
    }

    /**
     * Stable 0..1 regional affinity. At the Standard regionalization value
     * callers are free to ignore this value to preserve historical placement;
     * non-standard profiles use it to strengthen or soften province clustering.
     */
    public static double provinceStrength(
        GeoWorldV1 model,
        double worldX,
        double worldZ
    ) {
        double scale = provinceSize(model);
        long cellX = (long)Math.floor(worldX / scale);
        long cellZ = (long)Math.floor(worldZ / scale);
        long hash = hash(
            model.manifest().seed() ^ 0x4F524550524F5631L,
            cellX,
            cellZ
        );
        return unit(hash);
    }

    private static long coarseRegionId(
        GeoWorldV1 model,
        double worldX,
        double worldZ,
        double scaleMultiplier
    ) {
        double scale = provinceSize(model) * scaleMultiplier;
        long cellX = (long)Math.floor(worldX / scale);
        long cellZ = (long)Math.floor(worldZ / scale);
        return hash(
            model.manifest().seed() ^ 0x47454F4C4F475931L,
            cellX,
            cellZ
        ) & 0xFFFFFFL;
    }

    private static double provinceSize(GeoWorldV1 model) {
        double regional = model.worldProfile().parameters().mineralRegionalization();
        // Stronger regionalization means fewer, broader provinces.
        return BASE_PROVINCE_SIZE_BLOCKS * Math.sqrt(Math.max(0.25, regional));
    }

    private static long hash(long seed, long x, long z) {
        long h =
            seed ^
            (x * 0x9E3779B185EBCA87L) ^
            (z * 0xC2B2AE3D27D4EB4FL);
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        h ^= h >>> 31;
        return h;
    }

    private static double unit(long hash) {
        return (double)(hash >>> 11) / (double)(1L << 53);
    }
}
