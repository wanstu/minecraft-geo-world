package geoworld.minecraft.v26_3common;

import geoworld.minecraft.v26common.MinecraftV26Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26GeneratorSupport;

/** Minecraft 26.3 capability binding. */
public final class Minecraft26_3Profile {
    public static final MinecraftV26Profile INSTANCE =
        new MinecraftV26Profile(
            GeoWorldConstants.MINECRAFT_VERSION,
            true,
            true,
            true
        );

    public static final GeoWorldV26GeneratorSupport.MinecraftApi GENERATOR_API =
        new GeoWorldV26GeneratorSupport.MinecraftApi(
            (chunk, pos, state) -> chunk.setBlockState(pos, state),
            height -> height.getMinY(),
            height -> height.getMaxY(),
            (source, builder) ->
                source.getCarvers().forEach(builder::addCarver)
        );

    private Minecraft26_3Profile() {}
}
