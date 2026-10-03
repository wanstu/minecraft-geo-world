package geoworld.minecraft.v26_3common;

import geoworld.runtime.GeoWorldV1;

import java.util.LinkedHashMap;
import java.util.Map;

public final class Minecraft26_3BiomeRealizer {
    private static final Minecraft26_3HeightMapping HEIGHTS =
        new Minecraft26_3HeightMapping();

    private Minecraft26_3BiomeRealizer() {}

    public static String biomeId(
        GeoWorldV1 model,
        double blockX,
        double blockZ
    ) {
        return surfaceBiomeId(model, blockX, blockZ);
    }

    public static String biomeId(
        GeoWorldV1 model,
        double blockX,
        double blockY,
        double blockZ
    ) {
        GeoWorldV1.RuntimeSample sample = model.sample(blockX, blockZ);
        int surfaceY = HEIGHTS.boundaryY(sample.groundHeight()) - 1;
        String cave = caveBiomeId(
            model,
            sample,
            model.sampleClimate(blockX, blockZ),
            blockX,
            blockY,
            blockZ,
            surfaceY
        );
        return cave != null
            ? cave
            : surfaceBiomeId(model, blockX, blockZ);
    }

    private static String surfaceBiomeId(
        GeoWorldV1 model,
        double blockX,
        double blockZ
    ) {
        GeoWorldV1.RuntimeSample sample = model.sample(blockX, blockZ);
        GeoWorldV1.ClimateSample climate =
            model.sampleClimate(blockX, blockZ);
        String family = sample.biomeFamily();

        if ("River".equals(sample.waterKind()) ||
            "Lake".equals(sample.waterKind())) {
            return isFrozenRiverFamily(family)
                ? "minecraft:frozen_river"
                : "minecraft:river";
        }
        if ("Ocean".equals(sample.waterKind())) {
            return oceanBiomeId(sample, climate);
        }

        if (!"Wetland".equals(family) &&
            sample.sourceLand() &&
            nearOcean(model, blockX, blockZ, 28.0)) {
            return coastBiomeId(sample);
        }

        if (isMushroomCandidate(model, sample, blockX, blockZ)) {
            return "minecraft:mushroom_fields";
        }

        if (isTropicalForest(family)) {
            if (nearNonTropicalLand(model, blockX, blockZ, 64.0)) {
                return "minecraft:sparse_jungle";
            }
        } else if ("Subtropical Forest".equals(family)) {
            if (nearTropicalLand(model, blockX, blockZ, 40.0)) {
                return "minecraft:sparse_jungle";
            }
            if (nearTropicalLand(model, blockX, blockZ, 80.0)) {
                return "minecraft:forest";
            }
        }

        String base = landBiomeId(
            model,
            sample,
            climate,
            blockX,
            blockZ
        );
        return erodedBoundaryBiome(
            model,
            blockX,
            blockZ,
            family,
            base
        );
    }

    private static String oceanBiomeId(
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.ClimateSample climate
    ) {
        String family = sample.biomeFamily();
        double depth = Math.max(0.0, sample.waterDepth());
        boolean deep = depth >= 0.08;
        double temp = climate.available()
            ? climate.annualMeanTempC()
            : 12.0;

        return switch (family) {
            case "Warm Ocean" -> {
                if (deep) {
                    yield "minecraft:deep_lukewarm_ocean";
                }
                yield temp >= 23.0
                    ? "minecraft:warm_ocean"
                    : "minecraft:lukewarm_ocean";
            }
            case "Temperate Ocean" -> {
                if (temp >= 16.0) {
                    yield deep
                        ? "minecraft:deep_lukewarm_ocean"
                        : "minecraft:lukewarm_ocean";
                }
                yield deep
                    ? "minecraft:deep_ocean"
                    : "minecraft:ocean";
            }
            case "Cold Ocean" -> deep
                ? "minecraft:deep_cold_ocean"
                : "minecraft:cold_ocean";
            case "Frozen Ocean" -> deep
                ? "minecraft:deep_frozen_ocean"
                : "minecraft:frozen_ocean";
            default -> deep
                ? "minecraft:deep_ocean"
                : Minecraft26_3ProfileMapping.biomeId(family);
        };
    }

    private static String coastBiomeId(GeoWorldV1.RuntimeSample sample) {
        if (isFrozenRiverFamily(sample.biomeFamily())) {
            return "minecraft:snowy_beach";
        }
        String terrain = sample.terrainVariant();
        String surface = sample.surfaceProfile();
        if ("Mountain".equals(terrain) ||
            "Canyon".equals(terrain) ||
            "Alpine Rock".equals(surface) ||
            "Desert Gravel".equals(surface)) {
            return "minecraft:stony_shore";
        }
        return "minecraft:beach";
    }

    private static String landBiomeId(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.ClimateSample climate,
        double x,
        double z
    ) {
        String family = sample.biomeFamily();
        String terrain = sample.terrainVariant();
        String surface = sample.surfaceProfile();
        String vegetation = sample.vegetationProfile();
        double temp = climate.available()
            ? climate.annualMeanTempC()
            : 12.0;
        double precip = climate.available()
            ? climate.annualPrecipMM()
            : 800.0;
        double monsoon = climate.available()
            ? climate.monsoonIndex()
            : 0.0;
        double broad = smoothValueNoise(
            model.manifest().seed() ^ 0x22A5B94D3C11L,
            x,
            z,
            224.0
        );
        double detail = smoothValueNoise(
            model.manifest().seed() ^ 0x7D13A1C94E63L,
            x,
            z,
            88.0
        );

        // Hydrology may classify a broad lowland as grassland/forest while
        // Runtime still exposes floodplain/alluvial terrain and a wet climate.
        // Realize some of those physically wet pockets as vanilla swamps,
        // with mangroves restricted to warm coastal floodplains.
        boolean floodLowland =
            "Floodplain".equals(terrain) ||
            "Alluvial Plain".equals(terrain) ||
            "Alluvial Fan".equals(terrain);
        boolean hydricSoil =
            "Wet Mud".equals(surface) ||
            "Peat".equals(surface);
        if (hydricSoil ||
            (floodLowland && precip >= 1250.0 && detail > 0.66)) {
            if (temp >= 18.0 &&
                nearOcean(model, x, z, 96.0) &&
                (precip >= 1050.0 || monsoon >= 0.45)) {
                return "minecraft:mangrove_swamp";
            }
            return "minecraft:swamp";
        }

        switch (family) {
            case "Ice Sheet":
                return broad > 0.48
                    ? "minecraft:ice_spikes"
                    : "minecraft:snowy_plains";
            case "Tundra":
                if ("Hills".equals(terrain) ||
                    "Foothills".equals(terrain)) {
                    return detail > 0.58
                        ? "minecraft:grove"
                        : "minecraft:snowy_plains";
                }
                return broad > 0.76
                    ? "minecraft:ice_spikes"
                    : "minecraft:snowy_plains";
            case "Alpine":
                if ("Foothills".equals(terrain)) {
                    if (temp > -2.0 && precip >= 500.0) {
                        return detail > 0.58
                            ? "minecraft:grove"
                            : "minecraft:meadow";
                    }
                    return "minecraft:snowy_slopes";
                }
                if ("Hills".equals(terrain)) {
                    return "Alpine Rock".equals(surface)
                        ? "minecraft:windswept_gravelly_hills"
                        : "minecraft:snowy_slopes";
                }
                if ("Alpine Rock".equals(surface) || temp > 3.5) {
                    return detail > 0.62
                        ? "minecraft:stony_peaks"
                        : "minecraft:windswept_gravelly_hills";
                }
                return broad > 0.62
                    ? "minecraft:frozen_peaks"
                    : "minecraft:jagged_peaks";
            case "Boreal Forest":
                if (temp <= -3.0) {
                    return "minecraft:snowy_taiga";
                }
                if ("Mountain".equals(terrain) ||
                    "Foothills".equals(terrain) ||
                    "Hills".equals(terrain) ||
                    broad > 0.80) {
                    return precip >= 800.0
                        ? "minecraft:old_growth_spruce_taiga"
                        : "minecraft:old_growth_pine_taiga";
                }
                return "minecraft:taiga";
            case "Temperate Forest":
                if ("Mountain".equals(terrain)) {
                    return "minecraft:windswept_forest";
                }
                if ("Foothills".equals(terrain) &&
                    temp >= 5.0 && temp <= 18.0 &&
                    precip >= 650.0 && precip <= 1600.0 &&
                    detail > 0.60) {
                    return "minecraft:cherry_grove";
                }
                if (precip >= 1450.0 && broad > 0.67) {
                    return detail > 0.88
                        ? "minecraft:pale_garden"
                        : "minecraft:dark_forest";
                }
                if (detail > 0.82) {
                    return "minecraft:flower_forest";
                }
                if (detail > 0.65) {
                    return "minecraft:dappled_forest";
                }
                return "minecraft:forest";
            case "Temperate Grassland":
                if ("Mountain".equals(terrain)) {
                    return "minecraft:windswept_hills";
                }
                if ("Hills".equals(terrain) ||
                    "Foothills".equals(terrain)) {
                    if (precip >= 500.0 && temp <= 18.0) {
                        return detail > 0.60
                            ? "minecraft:meadow"
                            : "minecraft:windswept_hills";
                    }
                    return "minecraft:windswept_hills";
                }
                return broad > 0.76
                    ? "minecraft:sunflower_plains"
                    : "minecraft:plains";
            case "Temperate Desert":
                if ("Canyon".equals(terrain)) {
                    return "minecraft:eroded_badlands";
                }
                if ("Desert Gravel".equals(surface) &&
                    ("Hills".equals(terrain) ||
                     "Mountain".equals(terrain))) {
                    return "minecraft:windswept_gravelly_hills";
                }
                if ("Plateau".equals(terrain) ||
                    "Hills".equals(terrain) ||
                    broad > 0.76) {
                    if (detail > 0.76 &&
                        ("Hills".equals(terrain) ||
                         "Foothills".equals(terrain) ||
                         broad > 0.86)) {
                        return "minecraft:eroded_badlands";
                    }
                    if (precip > 140.0 &&
                        detail < 0.44) {
                        return "minecraft:wooded_badlands";
                    }
                    return "minecraft:badlands";
                }
                return "minecraft:desert";
            case "Mediterranean Shrubland":
                if ("Plateau".equals(terrain)) {
                    return "minecraft:savanna_plateau";
                }
                if ("Mountain".equals(terrain) ||
                    "Hills".equals(terrain) ||
                    "Foothills".equals(terrain)) {
                    return detail > 0.50
                        ? "minecraft:windswept_savanna"
                        : "minecraft:savanna_plateau";
                }
                return "minecraft:savanna";
            case "Subtropical Forest":
                if ("Mountain".equals(terrain)) {
                    return "minecraft:windswept_forest";
                }
                if ("Foothills".equals(terrain) &&
                    precip >= 700.0 && detail > 0.65) {
                    return "minecraft:cherry_grove";
                }
                if (precip >= 1500.0 && broad > 0.76) {
                    return "minecraft:dark_forest";
                }
                if (detail > 0.78) {
                    return "minecraft:old_growth_birch_forest";
                }
                if (detail > 0.58) {
                    return "minecraft:dappled_forest";
                }
                return "minecraft:birch_forest";
            case "Hot Desert":
                if ("Canyon".equals(terrain)) {
                    return "minecraft:eroded_badlands";
                }
                if ("Plateau".equals(terrain) ||
                    "Mountain".equals(terrain) ||
                    "Hills".equals(terrain) ||
                    broad > 0.70) {
                    if (detail > 0.74 &&
                        ("Hills".equals(terrain) ||
                         "Foothills".equals(terrain) ||
                         broad > 0.84)) {
                        return "minecraft:eroded_badlands";
                    }
                    if (precip > 120.0 &&
                        detail < 0.42) {
                        return "minecraft:wooded_badlands";
                    }
                    return "minecraft:badlands";
                }
                return "minecraft:desert";
            case "Savanna":
                if ("Plateau".equals(terrain)) {
                    return "minecraft:savanna_plateau";
                }
                if ("Mountain".equals(terrain) ||
                    "Hills".equals(terrain) ||
                    "Foothills".equals(terrain)) {
                    return detail > 0.42
                        ? "minecraft:windswept_savanna"
                        : "minecraft:savanna_plateau";
                }
                return "minecraft:savanna";
            case "Tropical Monsoon Forest":
                if (precip >= 1500.0 &&
                    (monsoon >= 0.55 || detail > 0.72)) {
                    return "minecraft:bamboo_jungle";
                }
                if (detail < 0.18) {
                    return "minecraft:sparse_jungle";
                }
                return "minecraft:jungle";
            case "Tropical Rainforest":
                if (precip >= 1750.0 && detail > 0.62) {
                    return "minecraft:bamboo_jungle";
                }
                return "minecraft:jungle";
            case "Wetland":
                if (temp >= 18.0 &&
                    (precip >= 1000.0 || monsoon >= 0.48) &&
                    nearOcean(model, x, z, 72.0)) {
                    return "minecraft:mangrove_swamp";
                }
                return "minecraft:swamp";
            default:
                return Minecraft26_3ProfileMapping.biomeId(family);
        }
    }

    private static String caveBiomeId(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.ClimateSample climate,
        double x,
        double y,
        double z,
        int surfaceY
    ) {
        if (!sample.sourceLand() ||
            y > Math.min(52.0, surfaceY - 20.0)) {
            return null;
        }

        String family = sample.biomeFamily();
        String terrain = sample.terrainVariant();
        double precip = climate.available()
            ? climate.annualPrecipMM()
            : 800.0;
        double temp = climate.available()
            ? climate.annualMeanTempC()
            : 12.0;
        double cave = smoothValueNoise(
            model.manifest().seed() ^ 0x4C4156455F31L,
            x,
            z,
            112.0
        );
        double detail = smoothValueNoise(
            model.manifest().seed() ^ 0x535542544552L,
            x,
            z,
            52.0
        );

        boolean highRelief =
            "Mountain".equals(terrain) ||
            "Plateau".equals(terrain) ||
            "Foothills".equals(terrain);
        if (y <= -20.0 && highRelief && cave > 0.54) {
            return "minecraft:deep_dark";
        }

        boolean hotDry =
            "Hot Desert".equals(family) ||
            "Temperate Desert".equals(family) ||
            "Canyon".equals(terrain);
        if (y <= 24.0 && hotDry && temp >= 15.0 && detail > 0.62) {
            return "minecraft:sulfur_caves";
        }

        boolean wet =
            precip >= 950.0 ||
            "Wetland".equals(family) ||
            isTropicalForest(family);
        if (wet && cave > 0.36) {
            return "minecraft:lush_caves";
        }

        boolean dryOrRocky =
            precip < 700.0 ||
            "Alpine".equals(family) ||
            "Canyon".equals(terrain) ||
            "Desert Gravel".equals(sample.surfaceProfile());
        if (dryOrRocky && detail > 0.32) {
            return "minecraft:dripstone_caves";
        }
        return null;
    }

    private static boolean isMushroomCandidate(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample sample,
        double x,
        double z
    ) {
        if (!sample.sourceLand() ||
            (!"Temperate Grassland".equals(sample.biomeFamily()) &&
             !"Temperate Forest".equals(sample.biomeFamily())) ||
            "Mountain".equals(sample.terrainVariant()) ||
            !nearOcean(model, x, z, 96.0)) {
            return false;
        }
        double broad = smoothValueNoise(
            model.manifest().seed() ^ 0x4D555348524F4F4DL,
            x,
            z,
            520.0
        );
        double detail = smoothValueNoise(
            model.manifest().seed() ^ 0x49534C414E44L,
            x,
            z,
            180.0
        );
        return broad > 0.78 && detail > 0.48;
    }

    private static String erodedBoundaryBiome(
        GeoWorldV1 model,
        double x,
        double z,
        String family,
        String base
    ) {
        final double radius = 56.0;
        final double diagonal = radius * 0.7071067811865476;
        double[][] offsets = {
            {radius, 0.0},
            {-radius, 0.0},
            {0.0, radius},
            {0.0, -radius},
            {diagonal, diagonal},
            {diagonal, -diagonal},
            {-diagonal, diagonal},
            {-diagonal, -diagonal}
        };

        LinkedHashMap<String, Integer> candidates = new LinkedHashMap<>();
        int different = 0;
        for (double[] offset : offsets) {
            GeoWorldV1.RuntimeSample nearby = model.sample(
                x + offset[0],
                z + offset[1]
            );
            if (
                !nearby.sourceLand() ||
                !"None".equals(nearby.waterKind()) ||
                nearby.biomeFamily().equals(family)
            ) {
                continue;
            }
            String candidate = Minecraft26_3ProfileMapping.biomeId(
                nearby.biomeFamily()
            );
            candidates.merge(candidate, 1, Integer::sum);
            different++;
        }
        if (different == 0 || candidates.isEmpty()) {
            return base;
        }

        String neighbor = candidates
            .entrySet()
            .stream()
            .max(Map.Entry.comparingByValue())
            .map(Map.Entry::getKey)
            .orElse(base);
        if (neighbor.equals(base)) {
            return base;
        }

        double strength = Math.min(1.0, different / 4.0);
        double noise = smoothValueNoise(
            model.manifest().seed() ^ 0x5A17C9E3L,
            x,
            z,
            24.0
        );
        return noise < 0.34 * strength ? neighbor : base;
    }

    private static boolean nearOcean(
        GeoWorldV1 model,
        double x,
        double z,
        double radius
    ) {
        return anyNearby(
            model,
            x,
            z,
            radius,
            sample -> "Ocean".equals(sample.waterKind())
        );
    }

    private static boolean nearTropicalLand(
        GeoWorldV1 model,
        double x,
        double z,
        double radius
    ) {
        return anyNearby(
            model,
            x,
            z,
            radius,
            sample -> sample.sourceLand() &&
                isTropicalForest(sample.biomeFamily())
        );
    }

    private static boolean nearNonTropicalLand(
        GeoWorldV1 model,
        double x,
        double z,
        double radius
    ) {
        return anyNearby(
            model,
            x,
            z,
            radius,
            sample -> sample.sourceLand() &&
                !isTropicalForest(sample.biomeFamily())
        );
    }

    private static boolean anyNearby(
        GeoWorldV1 model,
        double x,
        double z,
        double radius,
        java.util.function.Predicate<GeoWorldV1.RuntimeSample> predicate
    ) {
        double diagonal = radius * 0.7071067811865476;
        double[][] offsets = {
            {radius, 0.0},
            {-radius, 0.0},
            {0.0, radius},
            {0.0, -radius},
            {diagonal, diagonal},
            {diagonal, -diagonal},
            {-diagonal, diagonal},
            {-diagonal, -diagonal}
        };
        for (double[] offset : offsets) {
            if (predicate.test(model.sample(x + offset[0], z + offset[1]))) {
                return true;
            }
        }
        return false;
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
        t = Math.max(0.0, Math.min(1.0, t));
        return t * t * (3.0 - 2.0 * t);
    }

    private static boolean isTropicalForest(String family) {
        return "Tropical Rainforest".equals(family) ||
            "Tropical Monsoon Forest".equals(family);
    }

    private static boolean isFrozenRiverFamily(String family) {
        return "Frozen Ocean".equals(family) ||
            "Ice Sheet".equals(family) ||
            "Tundra".equals(family) ||
            "Alpine".equals(family);
    }
}
