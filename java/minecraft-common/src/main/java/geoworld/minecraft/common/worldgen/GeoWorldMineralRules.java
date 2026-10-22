package geoworld.minecraft.common.worldgen;

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
        if (y < mineral.minY() || y > mineral.maxY()) return 0;

        boolean mountains = "Mountain".equals(terrain);
        boolean foothills =
            "Foothills".equals(terrain) || "Hills".equals(terrain);
        boolean sedimentary = switch (terrain) {
            case "Plain", "Rolling Plain", "Floodplain",
                 "Alluvial Plain", "Alluvial Fan" -> true;
            default -> false;
        };
        boolean arid =
            biome.contains("Desert") || "Savanna".equals(biome);
        boolean highlands =
            mountains || foothills || "Plateau".equals(terrain);

        double multiplier = switch (mineral) {
            case COAL -> sedimentary ? 2.1 : 0.80;
            case IRON -> mountains ? 2.2 : foothills ? 1.6 : 1.0;
            case COPPER -> highlands ? 2.0 : 0.9;
            case GOLD ->
                ("Plateau".equals(terrain) ||
                 "Canyon".equals(terrain) ||
                 arid) ? 2.1 : 0.75;
            case LAPIS -> sedimentary ? 1.5 : 1.0;
            case REDSTONE -> highlands ? 1.25 : 1.0;
            case DIAMOND -> 1.0;
            case EMERALD -> mountains ? 2.4 : foothills ? 1.4 : 0.08;
        };

        double vertical = switch (mineral) {
            case COAL -> y < 0 ? 0.10 : y < 25 ? 0.7 : 1.3;
            case DIAMOND -> y > 0 ? 0.18 : 0.5 + (-y / 63.0);
            case REDSTONE -> y > 0 ? 0.3 : 1.0 + (-y / 100.0);
            case EMERALD -> y < 0 ? 0.08 : y > 55 ? 1.2 : 0.6;
            default -> 1.0;
        };
        return mineral.weight() * multiplier * vertical;
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
