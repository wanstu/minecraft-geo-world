package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.v26common.GeoWorldMinecraftIds;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderGetter;
import net.minecraft.core.QuartPos;
import net.minecraft.core.registries.Registries;
import net.minecraft.resources.ResourceKey;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.biome.BiomeSource;

import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Stream;

/**
 * Shared 26.x BiomeSource state and GeoWorld holder resolution.
 *
 * Minecraft 26.x releases deliberately keep their own biome lifecycle
 * overrides because Mojang changed that abstract method's signature.
 */
public abstract class GeoWorldV26BiomeSourceBase extends BiomeSource {
    private static final List<String> DEFAULT_BIOME_IDS = List.of(
        "minecraft:plains",
        "minecraft:sunflower_plains",
        "minecraft:snowy_plains",
        "minecraft:ice_spikes",
        "minecraft:desert",
        "minecraft:swamp",
        "minecraft:mangrove_swamp",
        "minecraft:forest",
        "minecraft:flower_forest",
        "minecraft:birch_forest",
        "minecraft:dappled_forest",
        "minecraft:dark_forest",
        "minecraft:pale_garden",
        "minecraft:old_growth_birch_forest",
        "minecraft:old_growth_pine_taiga",
        "minecraft:old_growth_spruce_taiga",
        "minecraft:taiga",
        "minecraft:snowy_taiga",
        "minecraft:savanna",
        "minecraft:savanna_plateau",
        "minecraft:windswept_hills",
        "minecraft:windswept_gravelly_hills",
        "minecraft:windswept_forest",
        "minecraft:windswept_savanna",
        "minecraft:jungle",
        "minecraft:sparse_jungle",
        "minecraft:bamboo_jungle",
        "minecraft:badlands",
        "minecraft:eroded_badlands",
        "minecraft:wooded_badlands",
        "minecraft:meadow",
        "minecraft:cherry_grove",
        "minecraft:grove",
        "minecraft:snowy_slopes",
        "minecraft:frozen_peaks",
        "minecraft:jagged_peaks",
        "minecraft:stony_peaks",
        "minecraft:river",
        "minecraft:frozen_river",
        "minecraft:beach",
        "minecraft:snowy_beach",
        "minecraft:stony_shore",
        "minecraft:warm_ocean",
        "minecraft:lukewarm_ocean",
        "minecraft:deep_lukewarm_ocean",
        "minecraft:ocean",
        "minecraft:deep_ocean",
        "minecraft:cold_ocean",
        "minecraft:deep_cold_ocean",
        "minecraft:frozen_ocean",
        "minecraft:deep_frozen_ocean",
        "minecraft:mushroom_fields",
        "minecraft:dripstone_caves",
        "minecraft:lush_caves",
        "minecraft:deep_dark",
        "minecraft:sulfur_caves"
    );

    @FunctionalInterface
    public interface BiomeIdResolver {
        String biomeId(
            GeoWorldV1 model,
            double blockX,
            double blockY,
            double blockZ
        );
    }

    private final List<Holder<Biome>> biomes;
    private final Map<String, Holder<Biome>> byId;
    private final Holder<Biome> fallback;
    private final BiomeIdResolver resolver;
    private volatile GeoWorldV1 model;

    protected GeoWorldV26BiomeSourceBase(
        List<Holder<Biome>> biomes,
        MinecraftVersionProfile profile
    ) {
        this(
            biomes,
            new GeoWorldV26BiomeRealizer(profile)::biomeId
        );
    }

    protected GeoWorldV26BiomeSourceBase(
        List<Holder<Biome>> biomes,
        BiomeIdResolver resolver
    ) {
        if (biomes == null || biomes.isEmpty()) {
            throw new IllegalArgumentException(
                "GeoWorld biome source requires at least one biome"
            );
        }
        this.resolver = java.util.Objects.requireNonNull(
            resolver,
            "resolver"
        );
        this.biomes = List.copyOf(biomes);

        LinkedHashMap<String, Holder<Biome>> map = new LinkedHashMap<>();
        for (Holder<Biome> holder : this.biomes) {
            holder.unwrapKey().ifPresent(
                key -> map.put(GeoWorldMinecraftIds.resourceKeyId(key), holder)
            );
        }
        this.byId = Map.copyOf(map);
        this.fallback = this.byId.getOrDefault(
            "minecraft:plains",
            this.biomes.get(0)
        );
    }

    protected static List<Holder<Biome>> defaultBiomeHolders(
        HolderGetter<Biome> biomes,
        MinecraftVersionProfile profile
    ) {
        java.util.Set<String> required =
            new GeoWorldV26BiomeRealizer(profile).requiredBiomeIds();
        java.util.ArrayList<Holder<Biome>> holders =
            new java.util.ArrayList<>(required.size());

        for (String id : DEFAULT_BIOME_IDS) {
            if (!required.contains(id)) {
                continue;
            }
            ResourceKey<Biome> key = GeoWorldMinecraftIds.resourceKey(
                Registries.BIOME,
                id
            );
            holders.add(biomes.getOrThrow(key));
        }

        if (holders.size() != required.size()) {
            java.util.LinkedHashSet<String> missing =
                new java.util.LinkedHashSet<>(required);
            for (Holder<Biome> holder : holders) {
                holder.unwrapKey().ifPresent(
                    key -> missing.remove(GeoWorldMinecraftIds.resourceKeyId(key))
                );
            }
            throw new IllegalStateException(
                "GeoWorld default biome holder list is incomplete: " +
                missing
            );
        }
        return List.copyOf(holders);
    }

    public final List<Holder<Biome>> biomes() {
        return this.biomes;
    }

    public final void attachModel(GeoWorldV1 model) {
        this.model = model;
    }

    @Override
    protected final Stream<Holder<Biome>> collectPossibleBiomes() {
        return this.biomes.stream();
    }

    protected final Holder<Biome> resolveNoiseBiome(
        int quartX,
        int quartY,
        int quartZ
    ) {
        GeoWorldV1 current = this.model;
        if (current == null) return this.fallback;

        int blockX = QuartPos.toBlock(quartX);
        int blockY = QuartPos.toBlock(quartY);
        int blockZ = QuartPos.toBlock(quartZ);
        String biomeId = this.resolver.biomeId(
            current,
            blockX,
            blockY,
            blockZ
        );
        return resolveBiomeHolder(biomeId);
    }

    private Holder<Biome> resolveBiomeHolder(String biomeId) {
        Holder<Biome> exact = this.byId.get(biomeId);
        if (exact != null) return exact;

        for (String candidate : compatibilityFallbacks(biomeId)) {
            Holder<Biome> compatible = this.byId.get(candidate);
            if (compatible != null) return compatible;
        }
        return this.fallback;
    }

    private static List<String> compatibilityFallbacks(String biomeId) {
        return switch (biomeId) {
            case "minecraft:stony_peaks",
                 "minecraft:frozen_peaks",
                 "minecraft:snowy_slopes",
                 "minecraft:grove" ->
                List.of("minecraft:jagged_peaks", "minecraft:snowy_plains");
            case "minecraft:meadow",
                 "minecraft:sunflower_plains" ->
                List.of("minecraft:plains");
            case "minecraft:flower_forest",
                 "minecraft:dappled_forest",
                 "minecraft:dark_forest",
                 "minecraft:pale_garden",
                 "minecraft:old_growth_birch_forest",
                 "minecraft:cherry_grove",
                 "minecraft:windswept_forest" ->
                List.of("minecraft:forest", "minecraft:birch_forest");
            case "minecraft:snowy_taiga",
                 "minecraft:old_growth_pine_taiga",
                 "minecraft:old_growth_spruce_taiga" ->
                List.of("minecraft:taiga", "minecraft:snowy_plains");
            case "minecraft:sparse_jungle",
                 "minecraft:bamboo_jungle" ->
                List.of("minecraft:jungle");
            case "minecraft:savanna_plateau",
                 "minecraft:windswept_savanna" ->
                List.of("minecraft:savanna");
            case "minecraft:badlands",
                 "minecraft:eroded_badlands",
                 "minecraft:wooded_badlands" ->
                List.of("minecraft:desert");
            case "minecraft:mangrove_swamp" ->
                List.of("minecraft:swamp");
            case "minecraft:windswept_hills",
                 "minecraft:windswept_gravelly_hills" ->
                List.of("minecraft:jagged_peaks", "minecraft:plains");
            case "minecraft:river" ->
                List.of("minecraft:ocean", "minecraft:plains");
            case "minecraft:frozen_river" ->
                List.of("minecraft:frozen_ocean", "minecraft:snowy_plains");
            case "minecraft:beach",
                 "minecraft:stony_shore" ->
                List.of("minecraft:plains");
            case "minecraft:snowy_beach" ->
                List.of("minecraft:snowy_plains");
            case "minecraft:lukewarm_ocean",
                 "minecraft:deep_lukewarm_ocean" ->
                List.of("minecraft:warm_ocean", "minecraft:ocean");
            case "minecraft:deep_ocean" ->
                List.of("minecraft:ocean");
            case "minecraft:deep_cold_ocean" ->
                List.of("minecraft:cold_ocean", "minecraft:ocean");
            case "minecraft:deep_frozen_ocean" ->
                List.of("minecraft:frozen_ocean");
            case "minecraft:mushroom_fields",
                 "minecraft:dripstone_caves",
                 "minecraft:lush_caves",
                 "minecraft:deep_dark",
                 "minecraft:sulfur_caves" ->
                List.of("minecraft:plains");
            default -> List.of();
        };
    }

    @Override
    public String toString() {
        return "GeoWorldBiomeSource[biomes=" + this.biomes.size() + "]";
    }
}
