package geoworld.minecraft.v1_21_1common.worldgen;

import com.mojang.serialization.MapCodec;
import com.mojang.serialization.codecs.RecordCodecBuilder;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.minecraft.v1_21_1common.Minecraft1_21_1Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeSourceBase;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderGetter;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.biome.Climate;

import java.util.List;

public final class GeoWorldBiomeSource extends GeoWorldV26BiomeSourceBase {
    private static final GeoWorldV26BiomeRealizer REALIZER =
        new GeoWorldV26BiomeRealizer(Minecraft1_21_1Profile.INSTANCE);
    public static final MapCodec<GeoWorldBiomeSource> CODEC =
        RecordCodecBuilder.mapCodec(instance -> instance.group(
            Biome.CODEC.listOf()
                .fieldOf("biomes")
                .forGetter(GeoWorldBiomeSource::biomes)
        ).apply(instance, GeoWorldBiomeSource::new));

    public GeoWorldBiomeSource(List<Holder<Biome>> biomes) {
        super(biomes, Minecraft1_21_1Profile.INSTANCE);
    }

    public static GeoWorldBiomeSource createDefault(
        HolderGetter<Biome> biomes
    ) {
        return new GeoWorldBiomeSource(
            defaultBiomeHolders(
                biomes,
                Minecraft1_21_1Profile.INSTANCE
            )
        );
    }

    @Override
    public Holder<Biome> getNoiseBiome(
        int quartX,
        int quartY,
        int quartZ,
        Climate.Sampler sampler
    ) {
        return resolveNoiseBiome(quartX, quartY, quartZ);
    }

    @Override
    protected MapCodec<? extends BiomeSource> codec() {
        return CODEC;
    }

    public static String minecraftBiomeId(
        GeoWorldV1 model,
        double blockX,
        double blockZ
    ) {
        return REALIZER.biomeId(
            model,
            blockX,
            blockZ
        );
    }

    public static String minecraftBiomeId(
        GeoWorldV1 model,
        double blockX,
        double blockY,
        double blockZ
    ) {
        return REALIZER.biomeId(
            model,
            blockX,
            blockY,
            blockZ
        );
    }
}
