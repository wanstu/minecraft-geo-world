package geoworld.minecraft.v26_1_2common;

import geoworld.minecraft.v26common.MinecraftV26Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26GeneratorSupport;

/** Minecraft 26.1.2 capability binding. */
public final class Minecraft26_1_2Profile {
    public static final MinecraftV26Profile INSTANCE =
        new MinecraftV26Profile(
            GeoWorldConstants.MINECRAFT_VERSION,
            false,
            true,
            false
        );

    public static final GeoWorldV26GeneratorSupport.MinecraftApi GENERATOR_API =
        new GeoWorldV26GeneratorSupport.MinecraftApi(
            (chunk, pos, state) -> chunk.setBlockState(pos, state),
            height -> height.getMinY(),
            height -> height.getMaxY(),
            (source, builder) ->
                source.getCarvers().forEach(builder::addCarver)
        );

    private Minecraft26_1_2Profile() {}
}
