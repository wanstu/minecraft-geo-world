package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.v26common.MinecraftV26ProfileMapping;

/**
 * Maps the *final* Minecraft Overworld biome, not GeoWorld's intermediate
 * soil/vegetation profile alone, to the blocks that will actually be placed.
 *
 * Geographical elevation, temperature, vegetation and hydrology still belong
 * to GeoWorld; this is their Minecraft-compatible surface realization.
 */
public final class GeoWorldV26SurfaceMaterials {
    public enum Cover {
        SAND, BADLANDS, SNOW, SNOWY_BEACH, SHORE_SAND, SHORE_STONE,
        OCEAN, FROZEN_OCEAN, FROZEN_RIVER, RIVER, GRASS, PODZOL,
        MYCELIUM, PALE_MOSS, MUD, SWAMP, STONE, GRAVEL, ALPINE,
        WINDSWEPT, CAVE
    }

    private GeoWorldV26SurfaceMaterials() {}

    /** Exact classification for every supported Minecraft 26.x vanilla Overworld biome. */
    public static Cover cover(String biome) {
        return switch (biome) {
            case "minecraft:desert" -> Cover.SAND;
            case "minecraft:badlands", "minecraft:eroded_badlands",
                "minecraft:wooded_badlands" -> Cover.BADLANDS;
            case "minecraft:beach" -> Cover.SHORE_SAND;
            case "minecraft:snowy_beach" -> Cover.SNOWY_BEACH;
            case "minecraft:stony_shore" -> Cover.SHORE_STONE;
            case "minecraft:river" -> Cover.RIVER;
            case "minecraft:frozen_river" -> Cover.FROZEN_RIVER;
            case "minecraft:warm_ocean", "minecraft:lukewarm_ocean",
                "minecraft:deep_lukewarm_ocean", "minecraft:ocean",
                "minecraft:deep_ocean", "minecraft:cold_ocean",
                "minecraft:deep_cold_ocean" -> Cover.OCEAN;
            case "minecraft:frozen_ocean", "minecraft:deep_frozen_ocean" ->
                Cover.FROZEN_OCEAN;
            case "minecraft:snowy_plains", "minecraft:ice_spikes",
                "minecraft:snowy_taiga", "minecraft:grove",
                "minecraft:snowy_slopes", "minecraft:frozen_peaks" -> Cover.SNOW;
            case "minecraft:stony_peaks" -> Cover.STONE;
            case "minecraft:jagged_peaks" -> Cover.ALPINE;
            case "minecraft:windswept_gravelly_hills" -> Cover.GRAVEL;
            case "minecraft:windswept_hills", "minecraft:windswept_forest" ->
                Cover.WINDSWEPT;
            case "minecraft:old_growth_pine_taiga",
                "minecraft:old_growth_spruce_taiga" -> Cover.PODZOL;
            case "minecraft:mushroom_fields" -> Cover.MYCELIUM;
            case "minecraft:pale_garden" -> Cover.PALE_MOSS;
            case "minecraft:mangrove_swamp" -> Cover.MUD;
            case "minecraft:swamp" -> Cover.SWAMP;
            case "minecraft:dripstone_caves", "minecraft:lush_caves",
                "minecraft:deep_dark", "minecraft:sulfur_caves" -> Cover.CAVE;
            case "minecraft:plains", "minecraft:sunflower_plains",
                "minecraft:meadow", "minecraft:forest",
                "minecraft:flower_forest", "minecraft:dappled_forest",
                "minecraft:dark_forest", "minecraft:birch_forest",
                "minecraft:old_growth_birch_forest", "minecraft:cherry_grove",
                "minecraft:taiga", "minecraft:jungle",
                "minecraft:sparse_jungle", "minecraft:bamboo_jungle",
                "minecraft:savanna", "minecraft:savanna_plateau",
                "minecraft:windswept_savanna" -> Cover.GRASS;
            default -> throw new IllegalArgumentException(
                "Missing Overworld surface mapping: " + biome
            );
        };
    }

    public static String top(String biome, String profile, String shoreline) {
        // Shoreline water ownership (including wetland pond rims) takes
        // precedence over decorative ecological biome surface materials.
        if (shoreline != null) return shoreline;
        return switch (cover(biome)) {
            case SAND, SHORE_SAND -> "minecraft:sand";
            // Badlands have red sand on weathered caps and exposed terracotta
            // on eroded/rocky crests; never a wall-to-wall dirt or grass plain.
            case BADLANDS -> "minecraft:eroded_badlands".equals(biome) ||
                "Desert Gravel".equals(profile) || "Alpine Rock".equals(profile)
                ? "minecraft:terracotta" : "minecraft:red_sand";
            case SNOW, SNOWY_BEACH -> "minecraft:snow_block";
            case SHORE_STONE, STONE -> "minecraft:stone";
            case ALPINE -> "Alpine Snow".equals(profile)
                ? "minecraft:snow_block" : "minecraft:stone";
            case GRAVEL -> "minecraft:gravel";
            case WINDSWEPT -> "Alpine Rock".equals(profile) ||
                "Desert Gravel".equals(profile)
                ? "minecraft:stone" : "minecraft:grass_block";
            case PODZOL -> "minecraft:podzol";
            case MYCELIUM -> "minecraft:mycelium";
            case PALE_MOSS -> "minecraft:pale_moss_block";
            case MUD -> "minecraft:mud";
            case SWAMP -> "Peat".equals(profile) || "Wet Mud".equals(profile)
                ? "minecraft:mud" : "minecraft:grass_block";
            case RIVER, FROZEN_RIVER, OCEAN, FROZEN_OCEAN, CAVE ->
                MinecraftV26ProfileMapping.surfaceBlock(profile);
            case GRASS -> "minecraft:grass_block";
        };
    }

    public static String topForRules(
        String biome,
        String profile,
        String shoreline,
        String generationRulesVersion
    ) {
        if (shoreline != null) return shoreline;
        Cover cover = cover(biome);
        if (
            (
                "geoworld-rules-v1.1.7".equals(generationRulesVersion) ||
                "geoworld-rules-v1.1.8".equals(generationRulesVersion) ||
                "geoworld-rules-v1.1.9".equals(generationRulesVersion)
            ) &&
            "Alpine Rock".equals(profile) &&
            (cover == Cover.GRASS || cover == Cover.PODZOL)
        ) {
            return "minecraft:stone";
        }
        return top(biome, profile, null);
    }

    public static String belowForRules(
        String biome,
        String profile,
        String shoreUnder,
        int blockX,
        int y,
        int blockZ,
        int depth,
        String generationRulesVersion
    ) {
        if (shoreUnder != null && depth <= 2) return shoreUnder;
        Cover cover = cover(biome);
        if (
            depth <= 3 &&
            (
                "geoworld-rules-v1.1.7".equals(generationRulesVersion) ||
                "geoworld-rules-v1.1.8".equals(generationRulesVersion) ||
                "geoworld-rules-v1.1.9".equals(generationRulesVersion)
            ) &&
            "Alpine Rock".equals(profile) &&
            (cover == Cover.GRASS || cover == Cover.PODZOL)
        ) {
            return "minecraft:stone";
        }
        return below(
            biome,
            profile,
            shoreUnder,
            blockX,
            y,
            blockZ,
            depth
        );
    }

    public static String below(
        String biome, String profile, String shoreUnder,
        int blockX, int y, int blockZ, int depth
    ) {
        if (shoreUnder != null && depth <= 2) return shoreUnder;
        Cover cover = cover(biome);
        if (cover == Cover.BADLANDS) {
            // A rocky exposed terracotta cap must not sit on two layers of
            // loose red sand. Weathered red-sand caps get a thin sand layer;
            // exposed ridges are terracotta all the way into the strata.
            if (depth <= 2 && "minecraft:red_sand".equals(
                top(biome, profile, null)
            )) {
                return "minecraft:red_sand";
            }
            if (depth <= 50) return badlandsStratum(blockX, y, blockZ);
        }
        if (depth > 3) return null; // caller uses STONE/DEEPSLATE host
        return switch (cover) {
            case SAND, SHORE_SAND -> "minecraft:sandstone";
            case SNOWY_BEACH -> "minecraft:sand";
            case SNOW, ALPINE, STONE, SHORE_STONE, GRAVEL, WINDSWEPT ->
                "minecraft:stone";
            case OCEAN, FROZEN_OCEAN, RIVER, FROZEN_RIVER, CAVE ->
                MinecraftV26ProfileMapping.underBlock(profile);
            case MUD, SWAMP, PALE_MOSS, PODZOL, GRASS, MYCELIUM ->
                "minecraft:dirt";
            case BADLANDS -> throw new AssertionError("handled above");
        };
    }

    /** Y-stratified, long contiguous terracotta bands; never checkerboard. */
    private static String badlandsStratum(int x, int y, int z) {
        int slowShift = Math.floorDiv(x, 160) + Math.floorDiv(z, 192);
        int band = Math.floorMod(Math.floorDiv(y + slowShift, 4), 17);
        if (band <= 3 || band >= 15) return "minecraft:terracotta";
        if (band <= 5) return "minecraft:orange_terracotta";
        if (band <= 7) return "minecraft:brown_terracotta";
        if (band <= 9) return "minecraft:yellow_terracotta";
        if (band == 10) return "minecraft:white_terracotta";
        if (band == 11) return "minecraft:light_gray_terracotta";
        return "minecraft:red_terracotta";
    }
}
