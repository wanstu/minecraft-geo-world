package geoworld.minecraft.v1_20_1common;

import geoworld.minecraft.v26common.MinecraftV26Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26GeneratorSupport;
import net.minecraft.world.level.levelgen.GenerationStep;

/** Minecraft 1.20.1 capability binding. */
public final class Minecraft1_20_1Profile {
    public static final MinecraftV26Profile INSTANCE =
        new MinecraftV26Profile(
            GeoWorldConstants.MINECRAFT_VERSION,
            false,
            false,
            false
        );

    public static final GeoWorldV26GeneratorSupport.MinecraftApi GENERATOR_API =
        new GeoWorldV26GeneratorSupport.MinecraftApi(
            (chunk, pos, state) -> chunk.setBlockState(pos, state, false),
            height -> height.getMinBuildHeight(),
            height -> height.getMaxBuildHeight(),
            (source, builder) -> {
                for (GenerationStep.Carving step :
                    GenerationStep.Carving.values()) {
                    source.getCarvers(step).forEach(
                        carver -> builder.addCarver(step, carver)
                    );
                }
            }
        );

    private Minecraft1_20_1Profile() {}
}
