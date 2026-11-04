package geoworld.runtime;

import java.util.Locale;

/**
 * Version-neutral mineral affinity shared by runtime queries and Minecraft
 * mineral placement. It contains no block or loader dependencies.
 */
public final class GeoWorldMineralAffinity {
    public static final String[] MINERALS = {
        "COAL", "IRON", "COPPER", "GOLD",
        "LAPIS", "REDSTONE", "DIAMOND", "EMERALD"
    };

    private GeoWorldMineralAffinity() {}

    public static double affinity(
        String mineral,
        String terrain,
        String biome,
        double y
    ) {
        String kind = normalize(mineral);
        Range range = range(kind);
        if (range == null || y < range.minY() || y > range.maxY()) {
            return 0.0;
        }

        boolean mountains = "Mountain".equals(terrain);
        boolean foothills =
            "Foothills".equals(terrain) || "Hills".equals(terrain);
        boolean sedimentary = switch (terrain) {
            case "Plain", "Rolling Plain", "Floodplain",
                 "Alluvial Plain", "Alluvial Fan" -> true;
            default -> false;
        };
        boolean arid =
            biome != null &&
            (biome.contains("Desert") || "Savanna".equals(biome));
        boolean highlands =
            mountains || foothills || "Plateau".equals(terrain);

        double multiplier = switch (kind) {
            case "COAL" -> sedimentary ? 2.1 : 0.80;
            case "IRON" -> mountains ? 2.2 : foothills ? 1.6 : 1.0;
            case "COPPER" -> highlands ? 2.0 : 0.9;
            case "GOLD" ->
                ("Plateau".equals(terrain) ||
                 "Canyon".equals(terrain) ||
                 arid) ? 2.1 : 0.75;
            case "LAPIS" -> sedimentary ? 1.5 : 1.0;
            case "REDSTONE" -> highlands ? 1.25 : 1.0;
            case "DIAMOND" -> 1.0;
            case "EMERALD" -> mountains ? 2.4 : foothills ? 1.4 : 0.08;
            default -> 0.0;
        };

        double vertical = switch (kind) {
            case "COAL" -> y < 0 ? 0.10 : y < 25 ? 0.7 : 1.3;
            case "DIAMOND" -> y > 0 ? 0.18 : 0.5 + (-y / 63.0);
            case "REDSTONE" -> y > 0 ? 0.3 : 1.0 + (-y / 100.0);
            case "EMERALD" -> y < 0 ? 0.08 : y > 55 ? 1.2 : 0.6;
            default -> 1.0;
        };
        return range.weight() * multiplier * vertical;
    }

    public static double[] normalizedProfile(
        String terrain,
        String biome,
        double y
    ) {
        double[] out = new double[MINERALS.length];
        double total = 0.0;
        for (int i = 0; i < MINERALS.length; i++) {
            out[i] = affinity(MINERALS[i], terrain, biome, y);
            total += out[i];
        }
        if (total <= 0.0) {
            return out;
        }
        for (int i = 0; i < out.length; i++) {
            out[i] /= total;
        }
        return out;
    }

    private static String normalize(String value) {
        return value == null
            ? ""
            : value.trim().toUpperCase(Locale.ROOT);
    }

    private record Range(int minY, int maxY, double weight) {}

    private static Range range(String mineral) {
        return switch (mineral) {
            case "COAL" -> new Range(-63, 215, 7.5);
            case "IRON" -> new Range(-60, 319, 9.0);
            case "COPPER" -> new Range(-20, 150, 6.0);
            case "GOLD" -> new Range(-63, 60, 3.2);
            case "LAPIS" -> new Range(-63, 80, 2.7);
            case "REDSTONE" -> new Range(-63, 24, 4.2);
            case "DIAMOND" -> new Range(-63, 24, 1.0);
            case "EMERALD" -> new Range(-63, 319, 1.2);
            default -> null;
        };
    }
}
