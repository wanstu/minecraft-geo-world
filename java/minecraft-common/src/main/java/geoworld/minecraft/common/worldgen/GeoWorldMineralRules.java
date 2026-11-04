package geoworld.minecraft.common.worldgen;

import geoworld.runtime.GeoWorldMineralAffinity;

/**
 * Version-neutral GeoWorld mineral selection and geology rules.
 * Minecraft-version modules are responsible only for mapping Kind to blocks.
 */
public final class GeoWorldMineralRules {
    public enum Kind {
        COAL(-63, 215, 7.5, 17, 3.3),
        IRON(-60, 319, 9, 16, 3.2),
        COPPER(-20, 150, 6.0, 16, 3.3),
        GOLD(-63, 60, 3.2, 13, 2.6),
        LAPIS(-63, 80, 2.7, 13, 2.6),
        REDSTONE(-63, 24, 4.2, 14, 2.7),
        DIAMOND(-63, 24, 1.0, 11, 2.1),
        EMERALD(-63, 319, 1.2, 11, 2.1);

        private final int minY;
        private final int maxY;
        private final double weight;
        private final double seamLength;
        private final double seamRadius;

        Kind(
            int minY,
            int maxY,
            double weight,
            double seamLength,
            double seamRadius
        ) {
            this.minY = minY;
            this.maxY = maxY;
            this.weight = weight;
            this.seamLength = seamLength;
            this.seamRadius = seamRadius;
        }

        public int minY() { return minY; }
        public int maxY() { return maxY; }
        public double weight() { return weight; }
        public double seamLength() { return seamLength; }
        public double seamRadius() { return seamRadius; }
    }

    private GeoWorldMineralRules() {}

    public static boolean isMountainTerrain(String terrain) {
        return "Mountain".equals(terrain) ||
            "Foothills".equals(terrain);
    }

    public static boolean isOutcropLocation(
        Kind mineral,
        String terrain,
        String water,
        int x,
        int z,
        long seed
    ) {
        if (
            !"None".equals(water) ||
            !isMountainTerrain(terrain)
        ) {
            return false;
        }
        if (
            mineral != Kind.COAL &&
            mineral != Kind.IRON &&
            mineral != Kind.COPPER &&
            mineral != Kind.EMERALD
        ) {
            return false;
        }
        long patch = cellSeed(
            seed ^ 0x4F555443524F5053L,
            Math.floorDiv(x, 8),
            0,
            Math.floorDiv(z, 8)
        );
        return (patch & 15L) == 0L;
    }

    public static Kind chooseMineral(
        String terrain,
        String biome,
        double y,
        double roll
    ) {
        double total = 0;
        for (Kind mineral : Kind.values()) {
            total += affinity(mineral, terrain, biome, y);
        }
        if (total == 0) return null;

        double target = roll * total;
        for (Kind mineral : Kind.values()) {
            double weight = affinity(mineral, terrain, biome, y);
            if (weight <= 0) continue;
            target -= weight;
            if (target < 0) return mineral;
        }
        return Kind.IRON;
    }

    public static double affinity(
        Kind mineral,
        String terrain,
        String biome,
        double y
    ) {
        return GeoWorldMineralAffinity.affinity(
            mineral.name(),
            terrain,
            biome,
            y
        );
    }

    public static int guaranteeY(Kind mineral, boolean deep) {
        if (deep) {
            return switch (mineral) {
                case COAL, COPPER, EMERALD -> -12;
                case IRON, GOLD, LAPIS -> -25;
                case REDSTONE -> -36;
                case DIAMOND -> -52;
            };
        }
        return switch (mineral) {
            case COAL -> 80;
            case IRON -> 28;
            case COPPER -> 30;
            case GOLD -> 32;
            case LAPIS -> 20;
            case REDSTONE -> 8;
            case DIAMOND -> 6;
            case EMERALD -> 64;
        };
    }

    private static long cellSeed(long seed, int x, int y, int z) {
        long h = seed ^ 0x9E3779B97F4A7C15L;
        h ^= (long)x * 0xBF58476D1CE4E5B9L;
        h ^= (long)y * 0x94D049BB133111EBL;
        h ^= (long)z * 0xD6E8FEB86659FD93L;
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        return h ^ (h >>> 31);
    }
}
