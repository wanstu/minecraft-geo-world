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
            biomes.getOrThrow(Biomes.SPARSE_JUNGLE),
            biomes.getOrThrow(Biomes.SWAMP),
            biomes.getOrThrow(Biomes.RIVER),
            biomes.getOrThrow(Biomes.FROZEN_RIVER)
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
        String biomeId = minecraftBiomeId(current, blockX, blockZ);
        return this.byId.getOrDefault(biomeId, this.fallback);
    }

    public static String minecraftBiomeId(
        GeoWorldV1 model,
        double blockX,
        double blockZ
    ) {
        GeoWorldV1.RuntimeSample sample = model.sample(blockX, blockZ);
        String family = sample.biomeFamily();

        // River is a first-class Runtime water semantic. Keep that semantic in
        // Minecraft's stored biome data too instead of inheriting the broad
        // terrestrial biome (for example Jungle) underneath the channel.
        if ("River".equals(sample.waterKind())) {
            return isFrozenRiverFamily(family)
                ? "minecraft:frozen_river"
                : "minecraft:river";
        }

        // The model already has a broad tropical -> subtropical -> temperate
        // climate transition, but mapping Tropical Rainforest directly to
        // vanilla JUNGLE makes the canopy stop like a wall. Use vanilla's
        // SPARSE_JUNGLE as a visual/ecological edge biome on both sides of the
        // tropical boundary, then a short FOREST shoulder before the normal
        // subtropical birch forest. This does not alter terrain/water/profile
        // semantics; it only makes the Minecraft biome realization match the
        // intended "mutual erosion" look.
        if (isTropicalForest(family)) {
            if (nearNonTropicalLand(model, blockX, blockZ, 64.0)) {
                return "minecraft:sparse_jungle";
            }
        } else if ("Subtropical Forest".equals(family)) {
            if (nearTropicalLand(model, blockX, blockZ, 40.0)) {
                return "minecraft:sparse_jungle";
            }
            if (nearTropicalLand(model, blockX, blockZ, 80.0)) {
                return "minecraft:forest";
            }
        }

        return Minecraft26_3ProfileMapping.biomeId(family);
    }

    private static boolean nearTropicalLand(
        GeoWorldV1 model,
        double x,
        double z,
        double radius
    ) {
        return anyNearby(
            model,
            x,
            z,
            radius,
            sample -> sample.sourceLand() &&
                isTropicalForest(sample.biomeFamily())
        );
    }

    private static boolean nearNonTropicalLand(
        GeoWorldV1 model,
        double x,
        double z,
        double radius
    ) {
        return anyNearby(
            model,
            x,
            z,
            radius,
            sample -> sample.sourceLand() &&
                !isTropicalForest(sample.biomeFamily())
        );
    }

    private static boolean anyNearby(
        GeoWorldV1 model,
        double x,
        double z,
        double radius,
        java.util.function.Predicate<GeoWorldV1.RuntimeSample> predicate
    ) {
        double diagonal = radius * 0.7071067811865476;
        double[][] offsets = {
            {radius, 0.0},
            {-radius, 0.0},
            {0.0, radius},
            {0.0, -radius},
            {diagonal, diagonal},
            {diagonal, -diagonal},
            {-diagonal, diagonal},
            {-diagonal, -diagonal}
        };
        for (double[] offset : offsets) {
            if (predicate.test(model.sample(x + offset[0], z + offset[1]))) {
                return true;
            }
        }
        return false;
    }

    private static boolean isTropicalForest(String family) {
        return "Tropical Rainforest".equals(family) ||
            "Tropical Monsoon Forest".equals(family);
    }

    private static boolean isFrozenRiverFamily(String family) {
        return "Frozen Ocean".equals(family) ||
            "Ice Sheet".equals(family) ||
            "Tundra".equals(family) ||
            "Alpine".equals(family);
    }

    @Override
    public String toString() {
        return "GeoWorldBiomeSource[biomes=" + this.biomes.size() + "]";
    }
}
