package geoworld.neoforge.worldgen;

import com.mojang.serialization.MapCodec;
import com.mojang.serialization.codecs.RecordCodecBuilder;
import geoworld.minecraft.v26_3common.Minecraft26_3ProfileMapping;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderGetter;
import net.minecraft.core.QuartPos;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.biome.Biomes;
import net.minecraft.world.level.biome.BiomeResolver;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.biome.Climate;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Stream;

public final class GeoWorldBiomeSource extends BiomeSource {
    public static final MapCodec<GeoWorldBiomeSource> CODEC =
        RecordCodecBuilder.mapCodec(instance -> instance.group(
            Biome.CODEC.listOf()
                .fieldOf("biomes")
                .forGetter(GeoWorldBiomeSource::biomes)
        ).apply(instance, GeoWorldBiomeSource::new));

    private final List<Holder<Biome>> biomes;
    private final Map<String, Holder<Biome>> byId;
    private final Holder<Biome> fallback;
    private volatile GeoWorldV1 model;

    public GeoWorldBiomeSource(List<Holder<Biome>> biomes) {
        if (biomes == null || biomes.isEmpty()) {
            throw new IllegalArgumentException(
                "GeoWorld biome source requires at least one biome"
            );
        }

        this.biomes = List.copyOf(biomes);
        LinkedHashMap<String, Holder<Biome>> map = new LinkedHashMap<>();
        for (Holder<Biome> holder : this.biomes) {
            holder.unwrapKey().ifPresent(
                key -> map.put(key.identifier().toString(), holder)
            );
        }
        this.byId = Map.copyOf(map);
        this.fallback = this.byId.getOrDefault(
            "minecraft:plains",
            this.biomes.getFirst()
        );
    }

    public List<Holder<Biome>> biomes() {
        return this.biomes;
    }

    public static GeoWorldBiomeSource createDefault(
        HolderGetter<Biome> biomes
    ) {
        return new GeoWorldBiomeSource(List.of(
            biomes.getOrThrow(Biomes.WARM_OCEAN),
            biomes.getOrThrow(Biomes.OCEAN),
            biomes.getOrThrow(Biomes.COLD_OCEAN),
            biomes.getOrThrow(Biomes.FROZEN_OCEAN),
            biomes.getOrThrow(Biomes.ICE_SPIKES),
            biomes.getOrThrow(Biomes.SNOWY_PLAINS),
            biomes.getOrThrow(Biomes.JAGGED_PEAKS),
            biomes.getOrThrow(Biomes.TAIGA),
            biomes.getOrThrow(Biomes.FOREST),
            biomes.getOrThrow(Biomes.PLAINS),
            biomes.getOrThrow(Biomes.DESERT),
            biomes.getOrThrow(Biomes.SAVANNA),
            biomes.getOrThrow(Biomes.BIRCH_FOREST),
            biomes.getOrThrow(Biomes.JUNGLE),
            biomes.getOrThrow(Biomes.SWAMP)
        ));
    }

    public void attachModel(GeoWorldV1 model) {
        this.model = model;
    }

    @Override
    protected Stream<Holder<Biome>> collectPossibleBiomes() {
        return this.biomes.stream();
    }

    @Override
    protected MapCodec<? extends BiomeSource> codec() {
        return CODEC;
    }

    @Override
    public BiomeResolver createResolver(Climate.Sampler sampler) {
        return this::getNoiseBiome;
    }

    public Holder<Biome> getNoiseBiome(int quartX, int quartY, int quartZ) {
        GeoWorldV1 current = this.model;
        if (current == null) {
            return this.fallback;
        }

        int blockX = QuartPos.toBlock(quartX);
        int blockZ = QuartPos.toBlock(quartZ);
        GeoWorldV1.RuntimeSample sample = current.sample(blockX, blockZ);
        String biomeId = Minecraft26_3ProfileMapping.biomeId(
            sample.biomeFamily()
        );
        return this.byId.getOrDefault(biomeId, this.fallback);
    }

    @Override
    public String toString() {
        return "GeoWorldBiomeSource[biomes=" + this.biomes.size() + "]";
    }
}
