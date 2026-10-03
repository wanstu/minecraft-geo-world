package geoworld.neoforge.worldgen;

import com.mojang.serialization.MapCodec;
import com.mojang.serialization.codecs.RecordCodecBuilder;
import geoworld.minecraft.v26_3common.Minecraft26_3BiomeRealizer;
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
            biomes.getOrThrow(Biomes.PLAINS),
            biomes.getOrThrow(Biomes.SUNFLOWER_PLAINS),
            biomes.getOrThrow(Biomes.SNOWY_PLAINS),
            biomes.getOrThrow(Biomes.ICE_SPIKES),
            biomes.getOrThrow(Biomes.DESERT),
            biomes.getOrThrow(Biomes.SWAMP),
            biomes.getOrThrow(Biomes.MANGROVE_SWAMP),
            biomes.getOrThrow(Biomes.FOREST),
            biomes.getOrThrow(Biomes.FLOWER_FOREST),
            biomes.getOrThrow(Biomes.BIRCH_FOREST),
            biomes.getOrThrow(Biomes.DAPPLED_FOREST),
            biomes.getOrThrow(Biomes.DARK_FOREST),
            biomes.getOrThrow(Biomes.PALE_GARDEN),
            biomes.getOrThrow(Biomes.OLD_GROWTH_BIRCH_FOREST),
            biomes.getOrThrow(Biomes.OLD_GROWTH_PINE_TAIGA),
            biomes.getOrThrow(Biomes.OLD_GROWTH_SPRUCE_TAIGA),
            biomes.getOrThrow(Biomes.TAIGA),
            biomes.getOrThrow(Biomes.SNOWY_TAIGA),
            biomes.getOrThrow(Biomes.SAVANNA),
            biomes.getOrThrow(Biomes.SAVANNA_PLATEAU),
            biomes.getOrThrow(Biomes.WINDSWEPT_HILLS),
            biomes.getOrThrow(Biomes.WINDSWEPT_GRAVELLY_HILLS),
            biomes.getOrThrow(Biomes.WINDSWEPT_FOREST),
            biomes.getOrThrow(Biomes.WINDSWEPT_SAVANNA),
            biomes.getOrThrow(Biomes.JUNGLE),
            biomes.getOrThrow(Biomes.SPARSE_JUNGLE),
            biomes.getOrThrow(Biomes.BAMBOO_JUNGLE),
            biomes.getOrThrow(Biomes.BADLANDS),
            biomes.getOrThrow(Biomes.ERODED_BADLANDS),
            biomes.getOrThrow(Biomes.WOODED_BADLANDS),
            biomes.getOrThrow(Biomes.MEADOW),
            biomes.getOrThrow(Biomes.CHERRY_GROVE),
            biomes.getOrThrow(Biomes.GROVE),
            biomes.getOrThrow(Biomes.SNOWY_SLOPES),
            biomes.getOrThrow(Biomes.FROZEN_PEAKS),
            biomes.getOrThrow(Biomes.JAGGED_PEAKS),
            biomes.getOrThrow(Biomes.STONY_PEAKS),
            biomes.getOrThrow(Biomes.RIVER),
            biomes.getOrThrow(Biomes.FROZEN_RIVER),
            biomes.getOrThrow(Biomes.BEACH),
            biomes.getOrThrow(Biomes.SNOWY_BEACH),
            biomes.getOrThrow(Biomes.STONY_SHORE),
            biomes.getOrThrow(Biomes.WARM_OCEAN),
            biomes.getOrThrow(Biomes.LUKEWARM_OCEAN),
            biomes.getOrThrow(Biomes.DEEP_LUKEWARM_OCEAN),
            biomes.getOrThrow(Biomes.OCEAN),
            biomes.getOrThrow(Biomes.DEEP_OCEAN),
            biomes.getOrThrow(Biomes.COLD_OCEAN),
            biomes.getOrThrow(Biomes.DEEP_COLD_OCEAN),
            biomes.getOrThrow(Biomes.FROZEN_OCEAN),
            biomes.getOrThrow(Biomes.DEEP_FROZEN_OCEAN),
            biomes.getOrThrow(Biomes.MUSHROOM_FIELDS),
            biomes.getOrThrow(Biomes.DRIPSTONE_CAVES),
            biomes.getOrThrow(Biomes.LUSH_CAVES),
            biomes.getOrThrow(Biomes.DEEP_DARK),
            biomes.getOrThrow(Biomes.SULFUR_CAVES)
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
        int blockY = QuartPos.toBlock(quartY);
        int blockZ = QuartPos.toBlock(quartZ);
        String biomeId = minecraftBiomeId(current, blockX, blockY, blockZ);
        return this.byId.getOrDefault(biomeId, this.fallback);
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

    @Override
    public String toString() {
        return "GeoWorldBiomeSource[biomes=" + this.biomes.size() + "]";
    }
}
