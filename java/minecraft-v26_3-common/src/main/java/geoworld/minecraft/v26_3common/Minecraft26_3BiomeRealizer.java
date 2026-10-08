package geoworld.minecraft.v26_3common;

import geoworld.runtime.GeoWorldV1;

import java.util.EnumSet;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.Map;
import java.util.Set;

public final class Minecraft26_3BiomeRealizer {
    private static final Minecraft26_3HeightMapping HEIGHTS =
        new Minecraft26_3HeightMapping();

    private Minecraft26_3BiomeRealizer() {}

    public static Set<String> requiredBiomeIds() {
        LinkedHashSet<String> ids = new LinkedHashSet<>(
            Minecraft26_3BiomeCatalog.candidateBiomeIds()
        );
        ids.add("minecraft:river");
        ids.add("minecraft:frozen_river");
        ids.add("minecraft:beach");
        ids.add("minecraft:snowy_beach");
        ids.add("minecraft:stony_shore");
        ids.add("minecraft:warm_ocean");
        ids.add("minecraft:lukewarm_ocean");
        ids.add("minecraft:deep_lukewarm_ocean");
        ids.add("minecraft:ocean");
        ids.add("minecraft:deep_ocean");
        ids.add("minecraft:cold_ocean");
        ids.add("minecraft:deep_cold_ocean");
        ids.add("minecraft:frozen_ocean");
        ids.add("minecraft:deep_frozen_ocean");
        ids.add("minecraft:mushroom_fields");
        ids.add("minecraft:dripstone_caves");
        ids.add("minecraft:lush_caves");
        ids.add("minecraft:deep_dark");
        ids.add("minecraft:sulfur_caves");
        return Set.copyOf(ids);
    }

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
        double temp = climate.available()
            ? climate.annualMeanTempC()
            : 12.0;
        double precip = climate.available()
            ? climate.annualPrecipMM()
            : 800.0;
        double monsoon = climate.available()
            ? climate.monsoonIndex()
            : 0.0;

        EnumSet<Minecraft26_3BiomeCatalog.Tag> tags = regionTags(
            model,
            sample,
            temp,
            precip,
            monsoon,
            x,
            z
        );
        String fallback = Minecraft26_3ProfileMapping.biomeId(
            sample.biomeFamily()
        );
        return Minecraft26_3BiomeCatalog.choose(
            new Minecraft26_3BiomeCatalog.Region(
                tags,
                temp,
                precip,
                monsoon
            ),
            model.manifest().seed(),
            x,
            z,
            fallback
        );
    }

    private static EnumSet<Minecraft26_3BiomeCatalog.Tag> regionTags(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample sample,
        double temp,
        double precip,
        double monsoon,
        double x,
        double z
    ) {
        EnumSet<Minecraft26_3BiomeCatalog.Tag> tags =
            EnumSet.noneOf(Minecraft26_3BiomeCatalog.Tag.class);

        if (temp <= -8.0) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.POLAR);
        } else if (temp <= 3.0) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.COLD);
        } else if (temp <= 10.0) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.COOL);
        } else if (temp <= 20.0) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.TEMPERATE);
        } else if (temp <= 27.0) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.WARM);
        } else {
            tags.add(Minecraft26_3BiomeCatalog.Tag.HOT);
        }

        if (precip < 200.0) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.ARID);
        } else if (precip < 500.0) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.DRY);
        } else if (precip < 1000.0) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.MESIC);
        } else if (precip < 1800.0) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.HUMID);
        } else {
            tags.add(Minecraft26_3BiomeCatalog.Tag.WET);
        }
        if (monsoon >= 0.45) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.MONSOON);
        }

        switch (sample.biomeFamily()) {
            case "Ice Sheet", "Tundra" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.TUNDRA);
                tags.add(Minecraft26_3BiomeCatalog.Tag.SNOWY);
            }
            case "Alpine" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.ALPINE);
                tags.add(Minecraft26_3BiomeCatalog.Tag.ROCKY);
                if (temp <= 4.0) {
                    tags.add(Minecraft26_3BiomeCatalog.Tag.SNOWY);
                }
            }
            case "Boreal Forest" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.FOREST);
                tags.add(Minecraft26_3BiomeCatalog.Tag.TAIGA);
            }
            case "Temperate Forest", "Subtropical Forest" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.FOREST);
            case "Temperate Grassland" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.GRASSLAND);
            case "Temperate Desert", "Hot Desert" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.DESERT);
            case "Mediterranean Shrubland" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.SHRUBLAND);
                tags.add(Minecraft26_3BiomeCatalog.Tag.SAVANNA);
            }
            case "Savanna" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.SAVANNA);
                tags.add(Minecraft26_3BiomeCatalog.Tag.GRASSLAND);
            }
            case "Tropical Monsoon Forest", "Tropical Rainforest" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.FOREST);
                tags.add(Minecraft26_3BiomeCatalog.Tag.JUNGLE);
            }
            case "Wetland" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.WETLAND);
            default -> {
            }
        }

        switch (sample.terrainVariant()) {
            case "Plain", "Rolling Plain" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.PLAINS);
            case "Hills" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.HILLS);
            case "Foothills" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.FOOTHILLS);
            case "Mountain" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.MOUNTAIN);
            case "Plateau" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.PLATEAU);
            case "Floodplain", "Alluvial Plain", "Alluvial Fan" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.FLOODPLAIN);
                tags.add(Minecraft26_3BiomeCatalog.Tag.PLAINS);
            }
            case "Canyon" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.CANYON);
                tags.add(Minecraft26_3BiomeCatalog.Tag.ROCKY);
            }
            default -> {
            }
        }

        switch (sample.surfaceProfile()) {
            case "Ice / Snow", "Alpine Snow" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.SNOWY);
            case "Alpine Rock", "Desert Gravel" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.ROCKY);
            case "Desert Sand" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.SANDY);
            case "Podzol" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.PODZOLIC);
            case "Alluvial Silt" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.ALLUVIAL);
            case "Wet Mud" ->
                // Wet alluvium is common beside rivers, but it does not turn
                // the surrounding climate biome into a swamp by itself.
                tags.add(Minecraft26_3BiomeCatalog.Tag.WET);
            case "Peat" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.PEATY);
                tags.add(Minecraft26_3BiomeCatalog.Tag.WET);
                tags.add(Minecraft26_3BiomeCatalog.Tag.WETLAND);
            }
            case "Dry Grass Soil", "Shrub Soil" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.DRY);
            default -> {
            }
        }

        switch (sample.vegetationProfile()) {
            case "Conifer Forest", "Mixed Forest", "Broadleaf Forest" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.FOREST);
            case "Dense Rainforest" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.FOREST);
                tags.add(Minecraft26_3BiomeCatalog.Tag.JUNGLE);
            }
            case "Monsoon Forest" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.FOREST);
                tags.add(Minecraft26_3BiomeCatalog.Tag.JUNGLE);
                tags.add(Minecraft26_3BiomeCatalog.Tag.MONSOON);
            }
            case "Grassland" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.GRASSLAND);
            case "Dry Grassland" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.GRASSLAND);
                tags.add(Minecraft26_3BiomeCatalog.Tag.DRY);
            }
            case "Shrubland" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.SHRUBLAND);
            case "Savanna" -> {
                tags.add(Minecraft26_3BiomeCatalog.Tag.SAVANNA);
                tags.add(Minecraft26_3BiomeCatalog.Tag.GRASSLAND);
            }
            case "Desert Sparse" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.DESERT);
            case "Wetland" ->
                tags.add(Minecraft26_3BiomeCatalog.Tag.WETLAND);
            default -> {
            }
        }

        // Floodplains remain the surrounding climate biome unless the
        // ecological model itself selected Wetland (or peat/wetland vegetation
        // added the tag above). High rainfall alone must not make every major
        // river corridor a Minecraft swamp.
        if (nearOcean(model, x, z, 96.0)) {
            tags.add(Minecraft26_3BiomeCatalog.Tag.COASTAL);
        }
        return tags;
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
