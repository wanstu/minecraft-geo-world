package geoworld.minecraft.v26_3common;

import geoworld.minecraft.v26common.MinecraftV26ProfileMapping;

/** Compatibility facade; shared 26.x profile mapping lives in v26-common. */
public final class Minecraft26_3ProfileMapping {
    private Minecraft26_3ProfileMapping() {}

    public static String surfaceBlock(String profile) {
        return MinecraftV26ProfileMapping.surfaceBlock(profile);
    }

    public static String underBlock(String profile) {
        return MinecraftV26ProfileMapping.underBlock(profile);
    }

    public static String biomeId(String family) {
        return MinecraftV26ProfileMapping.biomeId(family);
    }
}
