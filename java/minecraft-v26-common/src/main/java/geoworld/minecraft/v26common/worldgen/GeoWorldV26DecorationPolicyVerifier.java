package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.v26common.GeoWorldMinecraftIds;
import net.minecraft.core.Holder;
import net.minecraft.core.HolderOwner;
import net.minecraft.core.HolderSet;
import net.minecraft.core.registries.Registries;
import net.minecraft.resources.ResourceKey;
import net.minecraft.world.level.levelgen.GenerationStep;
import net.minecraft.world.level.levelgen.placement.PlacedFeature;

import java.util.ArrayList;
import java.util.List;

public final class GeoWorldV26DecorationPolicyVerifier {
    private GeoWorldV26DecorationPolicyVerifier() {}

    public static void main(String[] args) {
        ResourceKey<PlacedFeature> waterKey = key("spring_water");
        ResourceKey<PlacedFeature> lavaKey = key("spring_lava");
        ResourceKey<PlacedFeature> flowerKey = key("flower_default");
        ResourceKey<PlacedFeature> gravelKey = key("ore_gravel");
        ResourceKey<PlacedFeature> geodeKey = key("amethyst_geode");
        ResourceKey<PlacedFeature> tuffKey = key("ore_tuff");
        ResourceKey<PlacedFeature> ironKey = key("ore_iron_middle");
        ResourceKey<PlacedFeature> copperKey = key("ore_copper_large");

        List<String> oreFamilies = List.of(
            "coal", "iron", "copper", "gold", "lapis",
            "redstone", "diamond", "emerald"
        );
        if (!GeoWorldV26DecorationPolicy.suppresses(waterKey)) {
            throw new AssertionError("minecraft:spring_water must be suppressed");
        }
        for (String ore : oreFamilies) {
            if (!GeoWorldV26DecorationPolicy.suppresses(key("ore_" + ore)) ||
                !GeoWorldV26DecorationPolicy.suppresses(key("ore_" + ore + "_upper"))) {
                throw new AssertionError("unfiltered overworld ore family: " + ore);
            }
        }
        ResourceKey<PlacedFeature> foreignIron =
            GeoWorldMinecraftIds.resourceKey(
                Registries.PLACED_FEATURE,
                "examplemod:ore_iron"
            );
        for (ResourceKey<PlacedFeature> keep : List.of(
            lavaKey, flowerKey, gravelKey, geodeKey, tuffKey, foreignIron
        )) {
            if (GeoWorldV26DecorationPolicy.suppresses(keep)) {
                throw new AssertionError(
                    "unexpectedly suppresses " +
                    GeoWorldMinecraftIds.resourceKeyId(keep)
                );
            }
        }

        HolderOwner<PlacedFeature> owner = new HolderOwner<>() {};
        Holder<PlacedFeature> water = holder(owner, waterKey);
        Holder<PlacedFeature> lava = holder(owner, lavaKey);
        Holder<PlacedFeature> flower = holder(owner, flowerKey);
        Holder<PlacedFeature> gravel = holder(owner, gravelKey);
        Holder<PlacedFeature> geode = holder(owner, geodeKey);
        Holder<PlacedFeature> iron = holder(owner, ironKey);
        Holder<PlacedFeature> copper = holder(owner, copperKey);

        int springStep = GenerationStep.Decoration.FLUID_SPRINGS.ordinal();
        int oreStep = GenerationStep.Decoration.UNDERGROUND_ORES.ordinal();
        int flowerStep = GenerationStep.Decoration.VEGETAL_DECORATION.ordinal();
        int stepCount = Math.max(Math.max(springStep, oreStep), flowerStep) + 1;

        List<HolderSet<PlacedFeature>> source = new ArrayList<>();
        for (int i = 0; i < stepCount; i++) {
            source.add(HolderSet.<PlacedFeature>direct());
        }
        source.set(springStep, HolderSet.direct(List.of(water, lava)));
        source.set(oreStep, HolderSet.direct(List.of(iron, gravel, copper, geode)));
        source.set(flowerStep, HolderSet.direct(List.of(flower)));

        List<HolderSet<PlacedFeature>> filtered =
            GeoWorldV26DecorationPolicy.filterFeatureSteps(source);
        if (filtered == source || filtered.size() != source.size()) {
            throw new AssertionError("decoration filter failed to preserve steps");
        }
        if (!keys(filtered.get(springStep)).equals(List.of(lavaKey))) {
            throw new AssertionError("spring_lava was removed or reordered");
        }
        if (!keys(filtered.get(oreStep)).equals(List.of(gravelKey, geodeKey))) {
            throw new AssertionError("non-ore underground features were removed or reordered");
        }
        if (!keys(filtered.get(flowerStep)).equals(List.of(flowerKey))) {
            throw new AssertionError("vegetation was changed");
        }
        if (!keys(source.get(oreStep)).equals(
            List.of(ironKey, gravelKey, copperKey, geodeKey)
        )) {
            throw new AssertionError("source decoration list was mutated");
        }
        List<HolderSet<PlacedFeature>> untouched =
            List.of(HolderSet.direct(List.of(lava, geode)));
        if (GeoWorldV26DecorationPolicy.filterFeatureSteps(untouched) != untouched) {
            throw new AssertionError("unfiltered features must reuse original steps");
        }
        System.out.printf(
            "GeoWorld decoration policy PASS: spring_water removed, " +
            "8 vanilla ore families replaced, amethyst geodes/gravel/tuff/lava " +
            "and foreign mod ores retained, steps=%d%n", stepCount
        );
    }

    private static Holder<PlacedFeature> holder(
        HolderOwner<PlacedFeature> owner, ResourceKey<PlacedFeature> key
    ) {
        return Holder.Reference.createStandAlone(owner, key);
    }

    private static ResourceKey<PlacedFeature> key(String path) {
        return GeoWorldMinecraftIds.resourceKey(
            Registries.PLACED_FEATURE,
            "minecraft:" + path
        );
    }

    private static List<ResourceKey<PlacedFeature>> keys(HolderSet<PlacedFeature> set) {
        return set.stream().map(h -> h.unwrapKey().orElseThrow()).toList();
    }
}
