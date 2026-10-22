package geoworld.minecraft.v26common;

import java.util.ArrayList;
import geoworld.minecraft.common.biome.BiomeCatalog;
import geoworld.minecraft.common.biome.GeoWorldBiomeRegion;
import geoworld.minecraft.common.biome.GeoWorldBiomeTag;
import geoworld.minecraft.common.biome.TaggedBiomeCandidate;
import geoworld.minecraft.common.biome.TaggedBiomeSelector;

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
public final class MinecraftV26BiomeCatalog implements BiomeCatalog {
    private static final List<TaggedBiomeCandidate> ALL_CANDIDATES = List.of(
        c("minecraft:snowy_plains", all(GeoWorldBiomeTag.TUNDRA), none(),
            tags(GeoWorldBiomeTag.POLAR, GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.SNOWY, GeoWorldBiomeTag.PLAINS), none(),
            -40, 3, 0, 1400, 1.00),
        c("minecraft:ice_spikes", all(GeoWorldBiomeTag.TUNDRA), none(),
            tags(GeoWorldBiomeTag.POLAR, GeoWorldBiomeTag.SNOWY, GeoWorldBiomeTag.PLAINS), none(),
            -40, -4, 0, 900, 0.36),

        c("minecraft:plains", all(GeoWorldBiomeTag.GRASSLAND), none(),
            tags(GeoWorldBiomeTag.PLAINS, GeoWorldBiomeTag.TEMPERATE, GeoWorldBiomeTag.MESIC), tags(GeoWorldBiomeTag.WET),
            2, 23, 350, 1500, 1.20),
        c("minecraft:sunflower_plains", all(GeoWorldBiomeTag.GRASSLAND), none(),
            tags(GeoWorldBiomeTag.PLAINS, GeoWorldBiomeTag.WARM, GeoWorldBiomeTag.MESIC), tags(GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.WET),
            8, 26, 450, 1450, 0.42),
        c("minecraft:meadow", all(GeoWorldBiomeTag.GRASSLAND),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.COOL, GeoWorldBiomeTag.TEMPERATE, GeoWorldBiomeTag.HUMID), tags(GeoWorldBiomeTag.ARID),
            -2, 18, 500, 1800, 0.78),

        c("minecraft:forest", all(GeoWorldBiomeTag.FOREST), none(),
            tags(GeoWorldBiomeTag.TEMPERATE, GeoWorldBiomeTag.MESIC), tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.POLAR),
            3, 21, 500, 1700, 1.15),
        c("minecraft:flower_forest", all(GeoWorldBiomeTag.FOREST), none(),
            tags(GeoWorldBiomeTag.TEMPERATE, GeoWorldBiomeTag.HUMID, GeoWorldBiomeTag.PLAINS), tags(GeoWorldBiomeTag.ARID),
            6, 20, 700, 1700, 0.34),
        c("minecraft:dappled_forest", all(GeoWorldBiomeTag.FOREST), none(),
            tags(GeoWorldBiomeTag.TEMPERATE, GeoWorldBiomeTag.HUMID), tags(GeoWorldBiomeTag.ARID, GeoWorldBiomeTag.POLAR),
            5, 22, 650, 1800, 0.62),
        c("minecraft:dark_forest", all(GeoWorldBiomeTag.FOREST), none(),
            tags(GeoWorldBiomeTag.HUMID, GeoWorldBiomeTag.WET, GeoWorldBiomeTag.TEMPERATE), tags(GeoWorldBiomeTag.ARID),
            4, 21, 1050, 2600, 0.55),
        c("minecraft:pale_garden", all(GeoWorldBiomeTag.FOREST), none(),
            tags(GeoWorldBiomeTag.COOL, GeoWorldBiomeTag.HUMID, GeoWorldBiomeTag.WET), tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.ARID),
            0, 16, 1100, 2800, 0.13),
        c("minecraft:birch_forest", all(GeoWorldBiomeTag.FOREST), none(),
            tags(GeoWorldBiomeTag.TEMPERATE, GeoWorldBiomeTag.WARM, GeoWorldBiomeTag.MESIC), tags(GeoWorldBiomeTag.POLAR),
            6, 25, 500, 1700, 0.90),
        c("minecraft:old_growth_birch_forest", all(GeoWorldBiomeTag.FOREST),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.TEMPERATE, GeoWorldBiomeTag.HUMID), tags(GeoWorldBiomeTag.ARID),
            5, 22, 700, 1900, 0.46),
        c("minecraft:cherry_grove", all(GeoWorldBiomeTag.FOREST),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.COOL, GeoWorldBiomeTag.TEMPERATE, GeoWorldBiomeTag.HUMID), tags(GeoWorldBiomeTag.ARID, GeoWorldBiomeTag.HOT),
            3, 18, 650, 1700, 0.30),
        c("minecraft:windswept_forest", all(GeoWorldBiomeTag.FOREST),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.COOL, GeoWorldBiomeTag.ROCKY), none(),
            -4, 18, 350, 1800, 0.54),

        c("minecraft:taiga", all(GeoWorldBiomeTag.FOREST, GeoWorldBiomeTag.TAIGA), none(),
            tags(GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.COOL, GeoWorldBiomeTag.MESIC), tags(GeoWorldBiomeTag.HOT),
            -8, 12, 350, 1700, 1.05),
        c("minecraft:snowy_taiga", all(GeoWorldBiomeTag.FOREST, GeoWorldBiomeTag.TAIGA), none(),
            tags(GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.SNOWY), tags(GeoWorldBiomeTag.WARM, GeoWorldBiomeTag.HOT),
            -20, 2, 250, 1400, 0.65),
        c("minecraft:old_growth_pine_taiga", all(GeoWorldBiomeTag.FOREST, GeoWorldBiomeTag.TAIGA),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.COOL, GeoWorldBiomeTag.DRY, GeoWorldBiomeTag.PODZOLIC), tags(GeoWorldBiomeTag.WET),
            -5, 13, 300, 1000, 0.50),
        c("minecraft:old_growth_spruce_taiga", all(GeoWorldBiomeTag.FOREST, GeoWorldBiomeTag.TAIGA),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.COOL, GeoWorldBiomeTag.HUMID, GeoWorldBiomeTag.PODZOLIC), tags(GeoWorldBiomeTag.ARID),
            -5, 13, 700, 1900, 0.58),

        c("minecraft:jungle", all(GeoWorldBiomeTag.FOREST, GeoWorldBiomeTag.JUNGLE), none(),
            tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.HUMID, GeoWorldBiomeTag.WET), tags(GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.ARID),
            18, 40, 1200, 5000, 1.05),
        c("minecraft:sparse_jungle", all(GeoWorldBiomeTag.FOREST, GeoWorldBiomeTag.JUNGLE), none(),
            tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.MESIC, GeoWorldBiomeTag.DRY), tags(GeoWorldBiomeTag.COLD),
            17, 40, 650, 1900, 0.76),
        c("minecraft:bamboo_jungle", all(GeoWorldBiomeTag.FOREST, GeoWorldBiomeTag.JUNGLE), none(),
            tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.WET, GeoWorldBiomeTag.MONSOON), tags(GeoWorldBiomeTag.ARID),
            19, 40, 1500, 5500, 0.55),

        c("minecraft:savanna", all(GeoWorldBiomeTag.SAVANNA), none(),
            tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.DRY, GeoWorldBiomeTag.PLAINS), tags(GeoWorldBiomeTag.WET),
            16, 38, 180, 1000, 1.05),
        c("minecraft:savanna_plateau", all(GeoWorldBiomeTag.SAVANNA),
            any(GeoWorldBiomeTag.PLATEAU, GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS),
            tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.DRY), tags(GeoWorldBiomeTag.WET),
            15, 38, 150, 1000, 0.72),
        c("minecraft:windswept_savanna", all(GeoWorldBiomeTag.SAVANNA),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.DRY, GeoWorldBiomeTag.ROCKY), tags(GeoWorldBiomeTag.WET),
            14, 38, 120, 950, 0.62),

        c("minecraft:desert", all(GeoWorldBiomeTag.DESERT), none(),
            tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.ARID, GeoWorldBiomeTag.SANDY, GeoWorldBiomeTag.PLAINS), tags(GeoWorldBiomeTag.WET),
            10, 45, 0, 500, 1.15),
        c("minecraft:badlands", all(GeoWorldBiomeTag.DESERT),
            any(GeoWorldBiomeTag.PLATEAU, GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.ARID, GeoWorldBiomeTag.ROCKY), tags(GeoWorldBiomeTag.WET),
            8, 42, 0, 650, 0.85),
        c("minecraft:eroded_badlands", all(GeoWorldBiomeTag.DESERT),
            any(GeoWorldBiomeTag.CANYON, GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.ARID, GeoWorldBiomeTag.ROCKY), tags(GeoWorldBiomeTag.WET),
            8, 42, 0, 550, 0.58),
        c("minecraft:wooded_badlands", all(GeoWorldBiomeTag.DESERT),
            any(GeoWorldBiomeTag.PLATEAU, GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS),
            tags(GeoWorldBiomeTag.DRY, GeoWorldBiomeTag.SHRUBLAND), tags(GeoWorldBiomeTag.WET),
            6, 35, 120, 750, 0.48),

        c("minecraft:swamp", all(GeoWorldBiomeTag.WETLAND), none(),
            tags(GeoWorldBiomeTag.WET, GeoWorldBiomeTag.FLOODPLAIN), tags(GeoWorldBiomeTag.ARID, GeoWorldBiomeTag.POLAR),
            2, 30, 800, 5000, 1.05),
        c("minecraft:mangrove_swamp", all(GeoWorldBiomeTag.WETLAND, GeoWorldBiomeTag.COASTAL), none(),
            tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.WET, GeoWorldBiomeTag.MONSOON), tags(GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.ARID),
            18, 40, 900, 5500, 0.82),

        c("minecraft:grove", all(GeoWorldBiomeTag.ALPINE),
            any(GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.HILLS),
            tags(GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.FOREST, GeoWorldBiomeTag.SNOWY), tags(GeoWorldBiomeTag.HOT),
            -12, 5, 400, 1800, 0.72),
        c("minecraft:snowy_slopes", all(GeoWorldBiomeTag.ALPINE),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.SNOWY), tags(GeoWorldBiomeTag.HOT),
            -20, 4, 150, 1800, 0.92),
        c("minecraft:jagged_peaks", all(GeoWorldBiomeTag.ALPINE, GeoWorldBiomeTag.MOUNTAIN), none(),
            tags(GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.ROCKY, GeoWorldBiomeTag.SNOWY), tags(GeoWorldBiomeTag.HOT),
            -25, 2, 100, 1700, 0.82),
        c("minecraft:frozen_peaks", all(GeoWorldBiomeTag.ALPINE, GeoWorldBiomeTag.MOUNTAIN), none(),
            tags(GeoWorldBiomeTag.POLAR, GeoWorldBiomeTag.COLD, GeoWorldBiomeTag.SNOWY), tags(GeoWorldBiomeTag.WARM),
            -30, -2, 100, 1800, 0.64),
        c("minecraft:stony_peaks", all(GeoWorldBiomeTag.ALPINE),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.ROCKY, GeoWorldBiomeTag.COOL, GeoWorldBiomeTag.TEMPERATE), tags(GeoWorldBiomeTag.WET),
            -2, 16, 100, 1300, 0.72),
        c("minecraft:windswept_gravelly_hills", none(),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.ROCKY, GeoWorldBiomeTag.DRY), tags(GeoWorldBiomeTag.WET),
            -4, 24, 100, 1200, 0.48),
        c("minecraft:windswept_hills", none(),
            any(GeoWorldBiomeTag.HILLS, GeoWorldBiomeTag.FOOTHILLS, GeoWorldBiomeTag.MOUNTAIN),
            tags(GeoWorldBiomeTag.COOL, GeoWorldBiomeTag.TEMPERATE), tags(GeoWorldBiomeTag.HOT, GeoWorldBiomeTag.WET),
            -6, 20, 250, 1500, 0.58)
    );

    private final List<TaggedBiomeCandidate> candidates;

    public MinecraftV26BiomeCatalog(
        boolean supportsPaleGarden,
        boolean supportsDappledForest
    ) {
        this.candidates = ALL_CANDIDATES.stream()
            .filter(candidate -> supportsPaleGarden ||
                !candidate.id().equals("minecraft:pale_garden"))
            .filter(candidate -> supportsDappledForest ||
                !candidate.id().equals("minecraft:dappled_forest"))
            .toList();
    }

    @Override
    public Set<String> candidateBiomeIds() {
        return TaggedBiomeSelector.candidateBiomeIds(this.candidates);
    }

    @Override
    public String choose(
        GeoWorldBiomeRegion region,
        long worldSeed,
        double x,
        double z,
        String fallback
    ) {
        return TaggedBiomeSelector.choose(
            this.candidates,
            region,
            worldSeed,
            x,
            z,
            fallback,
            (candidateRegion, candidate) -> !(
                candidateRegion.tags().contains(GeoWorldBiomeTag.DESERT) &&
                (candidate.id().equals("minecraft:windswept_hills") ||
                 candidate.id().equals("minecraft:windswept_gravelly_hills"))
            )
        );
    }

    private static TaggedBiomeCandidate c(
        String id,
        Set<GeoWorldBiomeTag> requiredAll,
        Set<GeoWorldBiomeTag> requiredAny,
        Set<GeoWorldBiomeTag> preferred,
        Set<GeoWorldBiomeTag> avoided,
        double tempMin,
        double tempMax,
        double precipMin,
        double precipMax,
        double baseWeight
    ) {
        return new TaggedBiomeCandidate(
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

    private static Set<GeoWorldBiomeTag> all(GeoWorldBiomeTag... tags) {
        return tags(tags);
    }

    private static Set<GeoWorldBiomeTag> any(GeoWorldBiomeTag... tags) {
        return tags(tags);
    }

    private static Set<GeoWorldBiomeTag> none() {
        return Set.of();
    }

    private static Set<GeoWorldBiomeTag> tags(GeoWorldBiomeTag... tags) {
        if (tags.length == 0) {
            return Set.of();
        }
        EnumSet<GeoWorldBiomeTag> set = EnumSet.noneOf(GeoWorldBiomeTag.class);
        for (GeoWorldBiomeTag tag : tags) {
            set.add(tag);
        }
        return set;
    }

}
