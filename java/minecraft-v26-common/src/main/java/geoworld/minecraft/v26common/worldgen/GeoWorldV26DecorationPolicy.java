package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.v26common.GeoWorldMinecraftIds;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderSet;
import net.minecraft.resources.ResourceKey;
import net.minecraft.world.level.biome.BiomeGenerationSettings;
import net.minecraft.world.level.levelgen.carver.WorldCarver;
import net.minecraft.world.level.levelgen.placement.PlacedFeature;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.WeakHashMap;

/**
 * GeoWorld owns open surface water through its Runtime hydrology model.
 *
 * Vanilla spring_water deliberately places a source block next to one air
 * opening and immediately schedules a fluid tick. On steep GeoWorld terrain
 * that can create large cliff-side water sheets unrelated to Hydrology.
 *
 * This policy is attached only to GeoWorldChunkGenerator. Vanilla worlds and
 * every other biome feature remain unchanged.
 */
public final class GeoWorldV26DecorationPolicy {
    private static final String VANILLA_SPRING_WATER =
        "minecraft:spring_water";
    // GeoWorld filters these vanilla placed-feature ore families only inside
    // GeoWorld worlds, then v1.1.7+ restores the ordinary gameplay layer with
    // deterministic small scattered veins and adds separate large geological
    // bodies. This keeps old locked worlds stable while retaining both small
    // ores and GeoWorld-scale deposits in new worlds. Decorative stone, gravel,
    // dirt, flowers, trees, structures and Nether-only ancient debris remain.
    private static final List<String> REPLACED_ORE_PREFIXES = List.of(
        "ore_coal", "ore_iron", "ore_copper", "ore_gold",
        "ore_lapis", "ore_redstone", "ore_diamond", "ore_emerald"
    );

    private static final Map<
        BiomeGenerationSettings,
        BiomeGenerationSettings
    > CACHE = Collections.synchronizedMap(new WeakHashMap<>());

    private GeoWorldV26DecorationPolicy() {}

    public static BiomeGenerationSettings filter(
        BiomeGenerationSettings source,
        GeoWorldV26GeneratorSupport.MinecraftApi api
    ) {
        synchronized (CACHE) {
            return CACHE.computeIfAbsent(
                source,
                value -> buildFiltered(value, api)
            );
        }
    }

    public static boolean suppresses(Holder<PlacedFeature> feature) {
        return feature.unwrapKey()
            .map(GeoWorldV26DecorationPolicy::suppresses)
            .orElse(false);
    }

    public static boolean suppresses(ResourceKey<PlacedFeature> key) {
        String id = GeoWorldMinecraftIds.resourceKeyId(key);
        if (id.equals(VANILLA_SPRING_WATER)) {
            return true;
        }
        if (!"minecraft".equals(GeoWorldMinecraftIds.namespace(id))) {
            return false;
        }
        String path = GeoWorldMinecraftIds.path(id);
        for (String prefix : REPLACED_ORE_PREFIXES) {
            if (path.equals(prefix) || path.startsWith(prefix + "_")) {
                return true;
            }
        }
        return false;
    }

    public static List<HolderSet<PlacedFeature>> filterFeatureSteps(
        List<HolderSet<PlacedFeature>> source
    ) {
        boolean removed = false;
        List<HolderSet<PlacedFeature>> filtered =
            new ArrayList<>(source.size());

        for (HolderSet<PlacedFeature> step : source) {
            List<Holder<PlacedFeature>> kept =
                new ArrayList<>(step.size());
            for (Holder<PlacedFeature> feature : step) {
                if (suppresses(feature)) {
                    removed = true;
                } else {
                    kept.add(feature);
                }
            }
            filtered.add(HolderSet.direct(kept));
        }
        return removed ? List.copyOf(filtered) : source;
    }

    private static BiomeGenerationSettings buildFiltered(
        BiomeGenerationSettings source,
        GeoWorldV26GeneratorSupport.MinecraftApi api
    ) {
        List<HolderSet<PlacedFeature>> filteredSteps =
            filterFeatureSteps(source.features());
        if (filteredSteps == source.features()) {
            return source;
        }

        BiomeGenerationSettings.PlainBuilder builder =
            new BiomeGenerationSettings.PlainBuilder();
        // GeoWorld owns cave density. Do not copy Vanilla noise carvers here;
        // deterministic sparse caves are carved by GeoWorldV26SparseCaves.
        for (int step = 0; step < filteredSteps.size(); step++) {
            for (Holder<PlacedFeature> feature : filteredSteps.get(step)) {
                builder.addFeature(step, feature);
            }
        }
        return builder.build();
    }
}
