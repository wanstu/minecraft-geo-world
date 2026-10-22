package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.v26common.GeoWorldMinecraftIds;
import net.minecraft.core.registries.Registries;
import net.minecraft.resources.ResourceKey;
import net.minecraft.world.level.levelgen.presets.WorldPreset;

public final class GeoWorldV26WorldPresets {
    public static final ResourceKey<WorldPreset> GEOWORLD =
        GeoWorldMinecraftIds.resourceKey(
            Registries.WORLD_PRESET,
            "geoworld:geoworld"
        );

    private GeoWorldV26WorldPresets() {}
}
