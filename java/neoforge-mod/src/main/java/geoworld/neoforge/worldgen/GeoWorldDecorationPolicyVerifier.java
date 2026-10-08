package geoworld.neoforge.worldgen;

import net.minecraft.core.Holder;
import net.minecraft.core.HolderOwner;
import net.minecraft.core.HolderSet;
import net.minecraft.core.registries.Registries;
import net.minecraft.resources.Identifier;
import net.minecraft.resources.ResourceKey;
import net.minecraft.world.level.levelgen.GenerationStep;
import net.minecraft.world.level.levelgen.placement.PlacedFeature;

import java.util.ArrayList;
import java.util.List;

public final class GeoWorldDecorationPolicyVerifier {
    private GeoWorldDecorationPolicyVerifier() {}

    public static void main(String[] args) {
        ResourceKey<PlacedFeature> waterKey = key("spring_water");
        ResourceKey<PlacedFeature> lavaKey = key("spring_lava");
        ResourceKey<PlacedFeature> flowerKey = key("flower_default");

        if (!GeoWorldDecorationPolicy.suppresses(waterKey)) {
            throw new AssertionError(
                "minecraft:spring_water must be suppressed in GeoWorld"
            );
        }
        if (
            GeoWorldDecorationPolicy.suppresses(lavaKey) ||
            GeoWorldDecorationPolicy.suppresses(flowerKey)
        ) {
            throw new AssertionError(
                "GeoWorld decoration policy suppresses a non-water feature"
            );
        }

        HolderOwner<PlacedFeature> owner = new HolderOwner<>() {};
        Holder<PlacedFeature> water =
            Holder.Reference.createStandAlone(owner, waterKey);
        Holder<PlacedFeature> lava =
            Holder.Reference.createStandAlone(owner, lavaKey);
        Holder<PlacedFeature> flower =
            Holder.Reference.createStandAlone(owner, flowerKey);

        int springStep = GenerationStep.Decoration.FLUID_SPRINGS.ordinal();
        int vegetationStep =
            GenerationStep.Decoration.VEGETAL_DECORATION.ordinal();
        int stepCount = Math.max(springStep, vegetationStep) + 1;

        List<HolderSet<PlacedFeature>> steps =
            new ArrayList<>(stepCount);
        for (int i = 0; i < stepCount; i++) {
            steps.add(HolderSet.empty());
        }
        steps.set(
            springStep,
            HolderSet.direct(List.of(water, lava))
        );
        steps.set(
            vegetationStep,
            HolderSet.direct(List.of(flower))
        );

        List<HolderSet<PlacedFeature>> filtered =
            GeoWorldDecorationPolicy.filterFeatureSteps(steps);

        if (filtered == steps) {
            throw new AssertionError(
                "feature list containing spring_water was not filtered"
            );
        }
        if (filtered.size() != steps.size()) {
            throw new AssertionError(
                "filter changed decoration step count: source=" +
                steps.size() +
                " filtered=" +
                filtered.size()
            );
        }
        if (contains(filtered, waterKey)) {
            throw new AssertionError(
                "spring_water remains in filtered feature steps"
            );
        }
        if (!contains(filtered, lavaKey) || !contains(filtered, flowerKey)) {
            throw new AssertionError(
                "filter removed an unrelated biome feature"
            );
        }
        if (!contains(steps, waterKey)) {
            throw new AssertionError(
                "filter mutated source feature steps"
            );
        }

        List<HolderSet<PlacedFeature>> untouched =
            List.of(HolderSet.direct(List.of(lava, flower)));
        if (
            GeoWorldDecorationPolicy.filterFeatureSteps(untouched) !=
            untouched
        ) {
            throw new AssertionError(
                "feature list without spring_water should be reused unchanged"
            );
        }

        List<ResourceKey<PlacedFeature>> springOrder =
            keys(filtered.get(springStep));
        if (!springOrder.equals(List.of(lavaKey))) {
            throw new AssertionError(
                "fluid-springs feature order changed: " + springOrder
            );
        }
        List<ResourceKey<PlacedFeature>> vegetationOrder =
            keys(filtered.get(vegetationStep));
        if (!vegetationOrder.equals(List.of(flowerKey))) {
            throw new AssertionError(
                "vegetation feature order changed: " + vegetationOrder
            );
        }

        System.out.printf(
            "GeoWorld decoration policy verification passed: " +
            "suppressed=minecraft:spring_water, preserved=%s,%s, " +
            "steps=%d, order=stable%n",
            lavaKey.identifier(),
            flowerKey.identifier(),
            filtered.size()
        );
    }

    private static ResourceKey<PlacedFeature> key(String path) {
        return ResourceKey.create(
            Registries.PLACED_FEATURE,
            Identifier.withDefaultNamespace(path)
        );
    }

    private static boolean contains(
        List<HolderSet<PlacedFeature>> steps,
        ResourceKey<PlacedFeature> key
    ) {
        return steps.stream()
            .flatMap(HolderSet::stream)
            .anyMatch(holder -> holder.unwrapKey()
                .map(key::equals)
                .orElse(false));
    }

    private static List<ResourceKey<PlacedFeature>> keys(
        HolderSet<PlacedFeature> set
    ) {
        return set.stream()
            .map(holder -> holder.unwrapKey().orElseThrow())
            .toList();
    }
}
