package geoworld.minecraft.v26_3common;

public final class Minecraft26_3ProfileMapping {
    private Minecraft26_3ProfileMapping() {}

    public static String surfaceBlock(String profile) {
        return switch (profile) {
            case "Ocean Sand", "Desert Sand", "Desert Gravel" -> "minecraft:sand";
            case "Ocean Gravel" -> "minecraft:gravel";
            case "Frozen Ocean" -> "minecraft:gravel";
            case "Ice / Snow", "Alpine Snow" -> "minecraft:snow_block";
            case "Alpine Rock" -> "minecraft:stone";
            case "Forest Loam", "Grass Loam", "Dry Grass Soil", "Alluvial Silt" -> "minecraft:grass_block";
            case "Podzol" -> "minecraft:podzol";
            case "Tundra Soil", "Shrub Soil" -> "minecraft:coarse_dirt";
            case "Wet Mud", "Peat" -> "minecraft:mud";
            default -> "minecraft:grass_block";
        };
    }

    public static String underBlock(String profile) {
        return switch (profile) {
            case "Ocean Sand", "Desert Sand", "Desert Gravel" -> "minecraft:sandstone";
            case "Ocean Gravel", "Frozen Ocean", "Alpine Rock" -> "minecraft:stone";
            case "Ice / Snow", "Alpine Snow",
                 "Forest Loam", "Grass Loam", "Podzol",
                 "Tundra Soil", "Dry Grass Soil", "Shrub Soil",
                 "Alluvial Silt", "Wet Mud", "Peat" -> "minecraft:dirt";
            default -> "minecraft:dirt";
        };
    }

    public static String biomeId(String family) {
        return switch (family) {
            case "Warm Ocean" -> "minecraft:warm_ocean";
            case "Temperate Ocean" -> "minecraft:ocean";
            case "Cold Ocean" -> "minecraft:cold_ocean";
            case "Frozen Ocean" -> "minecraft:frozen_ocean";
            case "Ice Sheet" -> "minecraft:ice_spikes";
            case "Tundra" -> "minecraft:snowy_plains";
            case "Alpine" -> "minecraft:jagged_peaks";
            case "Boreal Forest" -> "minecraft:taiga";
            case "Temperate Forest" -> "minecraft:forest";
            case "Temperate Grassland" -> "minecraft:plains";
            case "Temperate Desert", "Hot Desert" -> "minecraft:desert";
            case "Mediterranean Shrubland" -> "minecraft:savanna";
            case "Subtropical Forest" -> "minecraft:birch_forest";
            case "Savanna" -> "minecraft:savanna";
            case "Tropical Monsoon Forest", "Tropical Rainforest" -> "minecraft:jungle";
            case "Wetland" -> "minecraft:swamp";
            default -> "minecraft:plains";
        };
    }
}
