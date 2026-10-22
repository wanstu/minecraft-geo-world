package geoworld.minecraft.v26common;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.common.biome.GeoWorldBiomeRegion;
import geoworld.minecraft.common.biome.GeoWorldBiomeTag;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.runtime.GeoWorldV1;

import java.util.EnumSet;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.Map;
import java.util.Set;

public final class GeoWorldV26BiomeRealizer {
    private final MinecraftVersionProfile profile;
    private final HeightMapping heights;
    private final Map<GeoWorldV1, GeoWorldV26BiomeCoverage> coverageCache =
        java.util.Collections.synchronizedMap(new java.util.WeakHashMap<>());

    public GeoWorldV26BiomeRealizer(MinecraftVersionProfile profile) {
        this.profile = java.util.Objects.requireNonNull(profile, "profile");
        this.heights = profile.heightMapping();
    }

    public GeoWorldV26BiomeCoverage coverageForModel(GeoWorldV1 model) {
        synchronized (coverageCache) {
            return coverageCache.computeIfAbsent(
                model,
                value -> new GeoWorldV26BiomeCoverage(
                    value,
                    this.heights,
                    requiredBiomeIds()
                )
            );
        }
    }

    public Set<String> requiredBiomeIds() {
        LinkedHashSet<String> ids = new LinkedHashSet<>(
            this.profile.biomeCatalog().candidateBiomeIds()
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
        if (supportsSulfurCaves()) {
            ids.add("minecraft:sulfur_caves");
        }
        return Set.copyOf(ids);
    }

    public String biomeId(
        GeoWorldV1 model,
        double blockX,
        double blockZ
    ) {
        String ensured = coverageForModel(model).override(
            model, blockX, 319, blockZ
        );
        return ensured != null ? ensured
            : surfaceBiomeId(
                model, blockX, blockZ, model.sample(blockX, blockZ),
                model.sampleClimate(blockX, blockZ)
            );
    }

    public String biomeId(
        GeoWorldV1 model,
        double blockX,
        double blockY,
        double blockZ
    ) {
        String ensured = coverageForModel(model).override(
            model, blockX, blockY, blockZ
        );
        if (ensured != null) {
            return ensured;
        }
        return biomeWithoutOverride(
            model, blockX, blockY, blockZ, model.sample(blockX, blockZ)
        );
    }

    /**
     * Same biome resolution, reusing an existing GeoWorld terrain sample.
     * The chunk shoreline planner has already sampled each surface column,
     * so sampling it again is wasted work.
     */
    public String biomeIdFromSample(
        GeoWorldV1 model,
        double blockX,
        double blockY,
        double blockZ,
        GeoWorldV1.RuntimeSample sample
    ) {
        String ensured = coverageForModel(model).override(
            model, blockX, blockY, blockZ
        );
        return ensured != null ? ensured
            : biomeWithoutOverride(model, blockX, blockY, blockZ, sample);
    }

    private String biomeWithoutOverride(
        GeoWorldV1 model,
        double blockX,
        double blockY,
        double blockZ,
        GeoWorldV1.RuntimeSample sample
    ) {
        GeoWorldV1.ClimateSample climate =
            model.sampleClimate(blockX, blockZ);
        int surfaceY = this.heights.boundaryY(sample.groundHeight()) - 1;
        String cave = caveBiomeId(
            model, sample, climate, blockX, blockY, blockZ, surfaceY
        );
        return cave != null ? cave
            : surfaceBiomeId(model, blockX, blockZ, sample, climate);
    }

    private String surfaceBiomeId(
        GeoWorldV1 model,
        double blockX,
        double blockZ,
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.ClimateSample climate
    ) {
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

        // Both mushroom coasts and region tags require the same 96-block
        // ocean-neighbor test. Reuse it rather than sampling eight model
        // positions twice for every temperate-forest or grassland column.
        boolean nearOcean96 = nearOcean(model, blockX, blockZ, 96.0);
        if (isMushroomCandidate(model, sample, blockX, blockZ, nearOcean96)) {
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
            blockZ,
            nearOcean96
        );
        return erodedBoundaryBiome(
            model,
            blockX,
            blockZ,
            family,
            base
        );
    }

    private String oceanBiomeId(
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
                : this.profile.biomeFamilyMapping().biomeId(family);
        };
    }

    private String coastBiomeId(GeoWorldV1.RuntimeSample sample) {
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


    private String landBiomeId(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample sample,
        GeoWorldV1.ClimateSample climate,
        double x,
        double z,
        boolean nearOcean96
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

        EnumSet<GeoWorldBiomeTag> tags = regionTags(
            model,
            sample,
            temp,
            precip,
            monsoon,
            nearOcean96
        );
        String fallback = this.profile.biomeFamilyMapping().biomeId(
            sample.biomeFamily()
        );
        return this.profile.biomeCatalog().choose(
            new GeoWorldBiomeRegion(
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

    private EnumSet<GeoWorldBiomeTag> regionTags(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample sample,
        double temp,
        double precip,
        double monsoon,
        boolean nearOcean96
    ) {
        EnumSet<GeoWorldBiomeTag> tags =
            EnumSet.noneOf(GeoWorldBiomeTag.class);

        if (temp <= -8.0) {
            tags.add(GeoWorldBiomeTag.POLAR);
        } else if (temp <= 3.0) {
            tags.add(GeoWorldBiomeTag.COLD);
        } else if (temp <= 10.0) {
            tags.add(GeoWorldBiomeTag.COOL);
        } else if (temp <= 20.0) {
            tags.add(GeoWorldBiomeTag.TEMPERATE);
        } else if (temp <= 27.0) {
            tags.add(GeoWorldBiomeTag.WARM);
        } else {
            tags.add(GeoWorldBiomeTag.HOT);
        }

        if (precip < 200.0) {
            tags.add(GeoWorldBiomeTag.ARID);
        } else if (precip < 500.0) {
            tags.add(GeoWorldBiomeTag.DRY);
        } else if (precip < 1000.0) {
            tags.add(GeoWorldBiomeTag.MESIC);
        } else if (precip < 1800.0) {
            tags.add(GeoWorldBiomeTag.HUMID);
        } else {
            tags.add(GeoWorldBiomeTag.WET);
        }
        if (monsoon >= 0.45) {
            tags.add(GeoWorldBiomeTag.MONSOON);
        }

        switch (sample.biomeFamily()) {
            case "Ice Sheet", "Tundra" -> {
                tags.add(GeoWorldBiomeTag.TUNDRA);
                tags.add(GeoWorldBiomeTag.SNOWY);
            }
            case "Alpine" -> {
                tags.add(GeoWorldBiomeTag.ALPINE);
                tags.add(GeoWorldBiomeTag.ROCKY);
                if (temp <= 4.0) {
                    tags.add(GeoWorldBiomeTag.SNOWY);
                }
            }
            case "Boreal Forest" -> {
                tags.add(GeoWorldBiomeTag.FOREST);
                tags.add(GeoWorldBiomeTag.TAIGA);
            }
            case "Temperate Forest", "Subtropical Forest" ->
                tags.add(GeoWorldBiomeTag.FOREST);
            case "Temperate Grassland" ->
                tags.add(GeoWorldBiomeTag.GRASSLAND);
            case "Temperate Desert", "Hot Desert" ->
                tags.add(GeoWorldBiomeTag.DESERT);
            case "Mediterranean Shrubland" -> {
                tags.add(GeoWorldBiomeTag.SHRUBLAND);
                tags.add(GeoWorldBiomeTag.SAVANNA);
            }
            case "Savanna" -> {
                tags.add(GeoWorldBiomeTag.SAVANNA);
                tags.add(GeoWorldBiomeTag.GRASSLAND);
            }
            case "Tropical Monsoon Forest", "Tropical Rainforest" -> {
                tags.add(GeoWorldBiomeTag.FOREST);
                tags.add(GeoWorldBiomeTag.JUNGLE);
            }
            case "Wetland" ->
                tags.add(GeoWorldBiomeTag.WETLAND);
            default -> {
            }
        }

        switch (sample.terrainVariant()) {
            case "Plain", "Rolling Plain" ->
                tags.add(GeoWorldBiomeTag.PLAINS);
            case "Hills" ->
                tags.add(GeoWorldBiomeTag.HILLS);
            case "Foothills" ->
                tags.add(GeoWorldBiomeTag.FOOTHILLS);
            case "Mountain" ->
                tags.add(GeoWorldBiomeTag.MOUNTAIN);
            case "Plateau" ->
                tags.add(GeoWorldBiomeTag.PLATEAU);
            case "Floodplain", "Alluvial Plain", "Alluvial Fan" -> {
                tags.add(GeoWorldBiomeTag.FLOODPLAIN);
                tags.add(GeoWorldBiomeTag.PLAINS);
            }
            case "Canyon" -> {
                tags.add(GeoWorldBiomeTag.CANYON);
                tags.add(GeoWorldBiomeTag.ROCKY);
            }
            default -> {
            }
        }

        switch (sample.surfaceProfile()) {
            case "Ice / Snow", "Alpine Snow" ->
                tags.add(GeoWorldBiomeTag.SNOWY);
            case "Alpine Rock", "Desert Gravel" ->
                tags.add(GeoWorldBiomeTag.ROCKY);
            case "Desert Sand" ->
                tags.add(GeoWorldBiomeTag.SANDY);
            case "Podzol" ->
                tags.add(GeoWorldBiomeTag.PODZOLIC);
            case "Alluvial Silt" ->
                tags.add(GeoWorldBiomeTag.ALLUVIAL);
            case "Wet Mud" ->
                // Wet alluvium is common beside rivers, but it does not turn
                // the surrounding climate biome into a swamp by itself.
                tags.add(GeoWorldBiomeTag.WET);
            case "Peat" -> {
                tags.add(GeoWorldBiomeTag.PEATY);
                tags.add(GeoWorldBiomeTag.WET);
                tags.add(GeoWorldBiomeTag.WETLAND);
            }
            case "Dry Grass Soil", "Shrub Soil" ->
                tags.add(GeoWorldBiomeTag.DRY);
            default -> {
            }
        }

        switch (sample.vegetationProfile()) {
            case "Conifer Forest", "Mixed Forest", "Broadleaf Forest" ->
                tags.add(GeoWorldBiomeTag.FOREST);
            case "Dense Rainforest" -> {
                tags.add(GeoWorldBiomeTag.FOREST);
                tags.add(GeoWorldBiomeTag.JUNGLE);
            }
            case "Monsoon Forest" -> {
                tags.add(GeoWorldBiomeTag.FOREST);
                tags.add(GeoWorldBiomeTag.JUNGLE);
                tags.add(GeoWorldBiomeTag.MONSOON);
            }
            case "Grassland" ->
                tags.add(GeoWorldBiomeTag.GRASSLAND);
            case "Dry Grassland" -> {
                tags.add(GeoWorldBiomeTag.GRASSLAND);
                tags.add(GeoWorldBiomeTag.DRY);
            }
            case "Shrubland" ->
                tags.add(GeoWorldBiomeTag.SHRUBLAND);
            case "Savanna" -> {
                tags.add(GeoWorldBiomeTag.SAVANNA);
                tags.add(GeoWorldBiomeTag.GRASSLAND);
            }
            case "Desert Sparse" ->
                tags.add(GeoWorldBiomeTag.DESERT);
            case "Wetland" ->
                tags.add(GeoWorldBiomeTag.WETLAND);
            default -> {
            }
        }

        // Floodplains remain the surrounding climate biome unless the
        // ecological model itself selected Wetland (or peat/wetland vegetation
        // added the tag above). High rainfall alone must not make every major
        // river corridor a Minecraft swamp.
        if (nearOcean96) {
            tags.add(GeoWorldBiomeTag.COASTAL);
        }
        return tags;
    }

    private String caveBiomeId(
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
        if (supportsSulfurCaves() &&
            y <= 24.0 && hotDry && temp >= 15.0 && detail > 0.62) {
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

    private boolean isMushroomCandidate(
        GeoWorldV1 model,
        GeoWorldV1.RuntimeSample sample,
        double x,
        double z,
        boolean nearOcean96
    ) {
        if (!sample.sourceLand() ||
            (!"Temperate Grassland".equals(sample.biomeFamily()) &&
             !"Temperate Forest".equals(sample.biomeFamily())) ||
            "Mountain".equals(sample.terrainVariant()) ||
            !nearOcean96) {
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

    private String erodedBoundaryBiome(
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
            String candidate = this.profile.biomeFamilyMapping().biomeId(
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

    private boolean nearOcean(
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

    private boolean nearTropicalLand(
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

    private boolean nearNonTropicalLand(
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

    private boolean anyNearby(
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

    private double smoothValueNoise(
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

    private double hashUnit(long seed, int x, int z) {
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

    private double smoothstep(double t) {
        t = Math.max(0.0, Math.min(1.0, t));
        return t * t * (3.0 - 2.0 * t);
    }

    private boolean supportsSulfurCaves() {
        return this.profile instanceof MinecraftV26Profile v26 &&
            v26.supportsSulfurCaves();
    }

    private boolean isTropicalForest(String family) {
        return "Tropical Rainforest".equals(family) ||
            "Tropical Monsoon Forest".equals(family);
    }

    private boolean isFrozenRiverFamily(String family) {
        return "Frozen Ocean".equals(family) ||
            "Ice Sheet".equals(family) ||
            "Tundra".equals(family) ||
            "Alpine".equals(family);
    }
}
