package geoworld.neoforge.worldgen;

import net.minecraft.core.Holder;
import net.minecraft.core.HolderSet;
import net.minecraft.resources.Identifier;
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
final class GeoWorldDecorationPolicy {
    private static final Identifier VANILLA_SPRING_WATER =
        Identifier.withDefaultNamespace("spring_water");
    // GeoWorld replaces these overworld ore families with deterministic
    // mineral veins. Keep vanilla decorative stone, gravel, dirt, flowers,
    // trees, structures, and Nether-only ancient debris untouched.
    private static final List<String> REPLACED_ORE_PREFIXES = List.of(
        "ore_coal", "ore_iron", "ore_copper", "ore_gold",
        "ore_lapis", "ore_redstone", "ore_diamond", "ore_emerald"
    );

    private static final Map<
        BiomeGenerationSettings,
        BiomeGenerationSettings
    > CACHE = Collections.synchronizedMap(new WeakHashMap<>());

    private GeoWorldDecorationPolicy() {}

    static BiomeGenerationSettings filter(
        BiomeGenerationSettings source
    ) {
        synchronized (CACHE) {
            return CACHE.computeIfAbsent(
                source,
                GeoWorldDecorationPolicy::buildFiltered
            );
        }
    }

    static boolean suppresses(Holder<PlacedFeature> feature) {
        return feature.unwrapKey()
            .map(GeoWorldDecorationPolicy::suppresses)
            .orElse(false);
    }

    static boolean suppresses(ResourceKey<PlacedFeature> key) {
        if (key.identifier().equals(VANILLA_SPRING_WATER)) {
            return true;
        }
        if (!"minecraft".equals(key.identifier().getNamespace())) {
            return false;
        }
        String path = key.identifier().getPath();
        for (String prefix : REPLACED_ORE_PREFIXES) {
            if (path.equals(prefix) || path.startsWith(prefix + "_")) {
                return true;
            }
        }
        return false;
    }

    static List<HolderSet<PlacedFeature>> filterFeatureSteps(
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
        BiomeGenerationSettings source
    ) {
        List<HolderSet<PlacedFeature>> filteredSteps =
            filterFeatureSteps(source.features());
        if (filteredSteps == source.features()) {
            return source;
        }

        List<Holder<WorldCarver>> carvers = new ArrayList<>();
        source.getCarvers().forEach(carvers::add);

        return new BiomeGenerationSettings(
            HolderSet.direct(carvers),
            filteredSteps
        );
    }
}
