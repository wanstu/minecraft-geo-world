package geoworld.minecraft.common;

import geoworld.minecraft.common.biome.BiomeCatalog;
import geoworld.minecraft.common.worldgen.HeightMapping;

/**
 * Minimal identity/capability root for one supported Minecraft version.
 * Add capabilities only when real cross-version differences require them.
 */
public interface MinecraftVersionProfile {
    String minecraftVersion();

    HeightMapping heightMapping();

    SurfaceProfileMapping surfaceProfileMapping();

    BiomeFamilyMapping biomeFamilyMapping();

    BiomeCatalog biomeCatalog();
}
