package geoworld.neoforge.worldgen;

import geoworld.neoforge.GeoWorldMod;
import net.minecraft.core.registries.Registries;
import net.minecraft.resources.Identifier;
import net.minecraft.resources.ResourceKey;
import net.minecraft.world.level.levelgen.presets.WorldPreset;

public final class GeoWorldWorldPresets {
    public static final ResourceKey<WorldPreset> GEOWORLD = ResourceKey.create(
        Registries.WORLD_PRESET,
        Identifier.fromNamespaceAndPath(GeoWorldMod.MOD_ID, "geoworld")
    );

    private GeoWorldWorldPresets() {}
}
