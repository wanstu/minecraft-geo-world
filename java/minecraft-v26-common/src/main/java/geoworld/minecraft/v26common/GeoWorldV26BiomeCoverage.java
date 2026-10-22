package geoworld.minecraft.v26common;

import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.runtime.GeoWorldV1;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * Guaranteed ecological representation across a finite GeoWorld world.
 *
 * The ordinary realizer determines almost all biomes. This supplemental plan
 * places one coherent habitat patch per registered overworld biome at the
 * best available geographic candidate. It does not modify Terrain, Water,
 * or the actual Minecraft blocks. Rare biomes are thus genuinely queryable
 * by the chunk generator rather than merely present in the registry.
 */
public final class GeoWorldV26BiomeCoverage {
    public record Habitat(String biome, double x, double z, int y, double radius) {}

    private record Site(
        double x, double z, int surfaceY, String terrain, String family,
        String water, double temperature, double precipitation, double depth
    ) {}

    private static final int HABITAT_BUCKET = 256;
    private final HeightMapping heights;
    private final List<Habitat> habitats;
    // Nearly every biome query is outside a reserved rare habitat. Use
    // spatial buckets instead of scanning all 56 reserved areas per column.
    private final Map<Long, List<Habitat>> habitatBuckets;

    public GeoWorldV26BiomeCoverage(
        GeoWorldV1 model,
        HeightMapping heights,
        Set<String> requiredBiomeIds
    ) {
        this.heights = java.util.Objects.requireNonNull(heights, "heights");
        var g = model.manifest().geometry();
        ArrayList<Site> candidates = new ArrayList<>();
        // Bound the search independently of the number of chunks. Grid
        // spacing is roughly 320 blocks in a default 320x200 macro world.
        final double step = Math.max(96.0, g.cellSize() * 2.5);
        for (double x = g.originX() + step / 2;
             x <= g.originX() + g.spanXBlocks() - step / 2; x += step) {
            for (double z = g.originZ() + step / 2;
                 z <= g.originZ() + g.spanZBlocks() - step / 2; z += step) {
                candidates.add(siteAt(model, x, z));
            }
        }

        // A 320-block ecological grid can skip narrow river channels
        // entirely. Refine candidate discovery near hydrographic receivers
        // without needing a dense grid for every biome family.
        final double riverStep = Math.max(96.0, g.cellSize());
        for (double x = g.originX() + riverStep / 2;
             x <= g.originX() + g.spanXBlocks() - riverStep / 2;
             x += riverStep) {
            for (double z = g.originZ() + riverStep / 2;
                 z <= g.originZ() + g.spanZBlocks() - riverStep / 2;
                 z += riverStep) {
                var sample = model.sample(x, z);
                if ("River".equals(sample.waterKind()) ||
                    "Lake".equals(sample.waterKind())) {
                    candidates.add(siteAt(model, x, z));
                }
            }
        }

        // Every biome (including caves, deep-water and coastal variants)
        // receives a deterministic distinct representative site.
        List<String> required = requiredBiomeIds.stream().sorted().toList();
        ArrayList<Habitat> selected = new ArrayList<>(required.size());
        for (String biome : required) {
            Site best = null;
            double bestScore = Double.NEGATIVE_INFINITY;
            for (Site site : candidates) {
                double score = affinity(biome, site);
                if (!Double.isFinite(score)) {
                    continue;
                }
                double spacing = selected.stream()
                    .mapToDouble(a -> Math.hypot(site.x() - a.x(), site.z() - a.z()))
                    .min().orElse(Double.POSITIVE_INFINITY);
                // Each guaranteed habitat must own its entire 116-block
                // interior. Penalizing collisions was insufficient on some
                // seeds: river and frozen_river could share a source cell.
                // Reserve nonoverlapping sites instead of masking collisions
                // through habitat iteration order.
                if (spacing < 256.0) {
                    continue;
                } else if (spacing < 850.0) {
                    score -= (850.0 - spacing) / 150.0;
                }
                // Stable scan order handles ties without random HashSet order.
                if (score > bestScore) {
                    bestScore = score;
                    best = site;
                }
            }
            if (best == null) {
                throw new IllegalStateException(
                    "GeoWorld cannot reserve required overworld biome " + biome
                );
            }
            int y = representativeY(biome, best.surfaceY());
            selected.add(new Habitat(biome, best.x(), best.z(), y, 116.0));
        }
        this.habitats = List.copyOf(selected);
        Map<Long, List<Habitat>> indexed = new HashMap<>();
        for (Habitat habitat : this.habitats) {
            int minX = bucket(habitat.x() - habitat.radius());
            int maxX = bucket(habitat.x() + habitat.radius());
            int minZ = bucket(habitat.z() - habitat.radius());
            int maxZ = bucket(habitat.z() + habitat.radius());
            for (int bx = minX; bx <= maxX; bx++) {
                for (int bz = minZ; bz <= maxZ; bz++) {
                    indexed.computeIfAbsent(bucketKey(bx, bz), unused -> new ArrayList<>())
                        .add(habitat);
                }
            }
        }
        Map<Long, List<Habitat>> frozen = new HashMap<>();
        indexed.forEach((key, value) -> frozen.put(key, List.copyOf(value)));
        habitatBuckets = Map.copyOf(frozen);
    }

    private Site siteAt(GeoWorldV1 model, double x, double z) {
        var sample = model.sample(x, z);
        var climate = model.sampleClimate(x, z);
        return new Site(
            x, z, this.heights.boundaryY(sample.groundHeight()) - 1,
            sample.terrainVariant(), sample.biomeFamily(),
            sample.waterKind(),
            climate.available() ? climate.annualMeanTempC() : 12.0,
            climate.available() ? climate.annualPrecipMM() : 800.0,
            sample.waterDepth()
        );
    }

    public List<Habitat> habitats() {
        return habitats;
    }

    private static int bucket(double coordinate) {
        return (int)Math.floor(coordinate / HABITAT_BUCKET);
    }

    private static long bucketKey(int bx, int bz) {
        return ((long)bx << 32) | (bz & 0xffffffffL);
    }

    public String override(GeoWorldV1 model, double x, double y, double z) {
        List<Habitat> nearby = habitatBuckets.get(
            bucketKey(bucket(x), bucket(z))
        );
        if (nearby == null) return null;
        Habitat chosen = null;
        double closest = Double.POSITIVE_INFINITY;
        for (Habitat habitat : nearby) {
            double dx = x - habitat.x();
            double dz = z - habitat.z();
            double distance2 = dx * dx + dz * dz;
            if (distance2 > habitat.radius() * habitat.radius() ||
                distance2 >= closest) {
                continue;
            }
            String biome = habitat.biome();
            if (isCave(biome)) {
                if (Math.abs(y - habitat.y()) > 24.0) continue;
            } else {
                // Surface patches do not suppress underground cave biomes.
                int ground = this.heights.boundaryY(model.sample(x, z).groundHeight()) - 1;
                if (y < ground - 16.0) continue;
            }
            chosen = habitat;
            closest = distance2;
        }
        return chosen == null ? null : chosen.biome();
    }

    private int representativeY(String biome, int surface) {
        if (biome.endsWith("deep_dark")) return -44;
        if (biome.endsWith("sulfur_caves")) return -16;
        if (biome.endsWith("lush_caves")) return 8;
        if (biome.endsWith("dripstone_caves")) return 12;
        return Math.min(this.heights.maxYExclusive() - 1, surface + 3);
    }

    private static boolean isCave(String biome) {
        return biome.endsWith("_caves") || biome.endsWith("deep_dark");
    }

    private static double affinity(String biome, Site s) {
        boolean ocean = "Ocean".equals(s.water());
        boolean river = "River".equals(s.water());
        boolean land = !ocean && s.surfaceY() >= 63;
        String name = biome.substring("minecraft:".length());
        boolean cave = isCave(biome);
        boolean oceanBiome = name.contains("ocean");
        boolean riverBiome = name.endsWith("river");
        boolean shore = name.contains("beach") || name.contains("shore");
        if (oceanBiome && !ocean) return Double.NEGATIVE_INFINITY;
        if (riverBiome && !river) return Double.NEGATIVE_INFINITY;
        if (cave && !land) return Double.NEGATIVE_INFINITY;
        if (!oceanBiome && !riverBiome && !cave && (ocean || !land)) {
            return Double.NEGATIVE_INFINITY;
        }

        double score = 0.0;
        double temp = s.temperature();
        double rain = s.precipitation();
        String family = s.family();
        String terrain = s.terrain();

        if (oceanBiome) {
            if (name.contains("frozen")) score -= Math.abs(temp + 8.0) * 0.48;
            else if (name.contains("cold")) score -= Math.abs(temp - 4.0) * 0.45;
            else if (name.contains("warm")) score -= Math.abs(temp - 28.0) * 0.45;
            else if (name.contains("lukewarm")) score -= Math.abs(temp - 20.0) * 0.4;
            else score -= Math.abs(temp - 13.0) * 0.38;
            score += name.startsWith("deep") ? s.depth() * 10 : -s.depth() * 18;
        } else if (riverBiome) {
            score -= Math.abs(temp - (name.startsWith("frozen") ? -10 : 17)) * 0.5;
        } else if (shore) {
            score -= Math.abs(s.surfaceY() - 65) * 0.30;
            if (name.startsWith("snowy")) score -= Math.abs(temp + 4.0) * 0.4;
            if (name.startsWith("stony")) score += terrain.equals("Mountain") ? 5 : 0;
        } else if (cave) {
            if (name.equals("deep_dark")) score += terrain.equals("Mountain") ? 12 : 0;
            if (name.equals("sulfur_caves")) score +=
                (family.contains("Desert") ? 12 : 0) + Math.min(6, temp / 6.0);
            if (name.equals("lush_caves")) score += Math.min(12, rain / 180.0);
            if (name.equals("dripstone_caves")) score +=
                family.contains("Desert") || terrain.equals("Mountain") ? 10 : 0;
        } else {
            if (name.contains("snowy") || name.contains("frozen") ||
                name.equals("ice_spikes")) {
                score -= Math.abs(temp + 9.0) * 0.5;
            } else if (name.contains("jungle") || name.contains("mangrove")) {
                score -= Math.abs(temp - 29.0) * 0.38;
                score += Math.min(8, rain / 270.0);
            } else if (name.contains("desert") || name.contains("badlands")) {
                score -= Math.abs(temp - 29.0) * 0.30;
                score += Math.max(-12, 8 - rain / 140.0);
            } else if (name.contains("taiga")) {
                score -= Math.abs(temp - 2.0) * 0.44;
            } else if (name.contains("swamp")) {
                score -= Math.abs(temp - 23.0) * 0.30;
                score += Math.min(7.0, rain / 250.0);
            } else if (name.contains("savanna")) {
                score -= Math.abs(temp - 27.0) * 0.38;
                score += Math.max(-8, 6 - rain / 230.0);
            } else if (name.contains("forest") || name.contains("grove") ||
                name.contains("garden")) {
                score -= Math.abs(temp - 14.0) * 0.30;
                score += Math.min(4, rain / 350.0);
            } else {
                score -= Math.abs(temp - 16.0) * 0.22;
            }
            if (name.contains("mushroom")) score += s.surfaceY() < 100 ? 8 : 0;
            if (name.contains("hill") || name.contains("peak") ||
                name.contains("slope") || name.contains("grove") ||
                name.contains("meadow") || name.contains("windswept")) {
                score += terrain.equals("Mountain") ? 9 :
                    terrain.equals("Foothills") || terrain.equals("Hills") ? 5 : -6;
            }
            if (name.contains("badlands") || name.contains("plateau"))
                score += terrain.equals("Plateau") || terrain.equals("Canyon") ? 8 : 0;
            if (name.contains("swamp")) score += family.equals("Wetland") ? 9 : 0;
            if (name.contains("jungle")) score += family.contains("Tropical") ? 9 : 0;
            if (name.contains("taiga")) score += family.equals("Boreal Forest") ? 9 : 0;
            if (name.contains("forest")) score += family.contains("Forest") ? 4 : 0;
        }
        return score;
    }
}
