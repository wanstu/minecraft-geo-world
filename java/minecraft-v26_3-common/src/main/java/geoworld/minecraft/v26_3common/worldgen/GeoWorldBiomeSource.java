package geoworld.minecraft.v26_3common.worldgen;

import com.mojang.serialization.MapCodec;
import com.mojang.serialization.codecs.RecordCodecBuilder;
import geoworld.minecraft.v26_3common.Minecraft26_3BiomeRealizer;
import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeSourceBase;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderGetter;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.biome.BiomeResolver;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.biome.Climate;

import java.util.List;

public final class GeoWorldBiomeSource extends GeoWorldV26BiomeSourceBase {
    public static final MapCodec<GeoWorldBiomeSource> CODEC =
        RecordCodecBuilder.mapCodec(instance -> instance.group(
            Biome.CODEC.listOf()
                .fieldOf("biomes")
                .forGetter(GeoWorldBiomeSource::biomes)
        ).apply(instance, GeoWorldBiomeSource::new));

    public GeoWorldBiomeSource(List<Holder<Biome>> biomes) {
        super(biomes, Minecraft26_3Profile.INSTANCE);
    }

    public static GeoWorldBiomeSource createDefault(
        HolderGetter<Biome> biomes
    ) {
        return new GeoWorldBiomeSource(
            defaultBiomeHolders(
                biomes,
                Minecraft26_3Profile.INSTANCE
            )
        );
    }

    @Override
    public BiomeResolver createResolver(Climate.Sampler sampler) {
        return this::getNoiseBiome;
    }

    public Holder<Biome> getNoiseBiome(
        int quartX,
        int quartY,
        int quartZ
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
        return Minecraft26_3BiomeRealizer.biomeId(
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
        return Minecraft26_3BiomeRealizer.biomeId(
            model,
            blockX,
            blockY,
            blockZ
        );
    }
}
