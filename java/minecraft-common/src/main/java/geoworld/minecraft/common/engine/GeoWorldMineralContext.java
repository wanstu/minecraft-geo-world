package geoworld.minecraft.common.engine;

import java.util.ArrayList;
import java.util.Comparator;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;

/**
 * Generation-time mineral context supplied by the Minecraft-version adapter.
 *
 * The common engine deliberately does not know how a loader exposes ore tags.
 * Version/loader code publishes the dictionary and the currently realizable
 * ore families immediately before generating or loading a World Model cache.
 */
public final class GeoWorldMineralContext {
    private static final List<String> VANILLA = List.of(
        "coal", "iron", "copper", "gold",
        "lapis", "redstone", "diamond", "emerald"
    );

    private static volatile Snapshot current =
        new Snapshot(VANILLA, VANILLA);

    private GeoWorldMineralContext() {}

    public static void publish(
        List<String> dictionary,
        List<String> available
    ) {
        List<String> normalizedAvailable = normalize(available);
        if (normalizedAvailable.isEmpty()) {
            normalizedAvailable = VANILLA;
        }

        LinkedHashSet<String> merged = new LinkedHashSet<>(
            normalize(dictionary)
        );
        merged.addAll(normalizedAvailable);
        if (merged.isEmpty()) {
            merged.addAll(VANILLA);
        }
        ArrayList<String> normalizedDictionary =
            new ArrayList<>(merged);
        normalizedDictionary.sort(Comparator.naturalOrder());

        current = new Snapshot(
            List.copyOf(normalizedDictionary),
            List.copyOf(normalizedAvailable)
        );
    }

    public static Snapshot snapshot() {
        return current;
    }

    private static List<String> normalize(List<String> values) {
        if (values == null || values.isEmpty()) {
            return List.of();
        }
        LinkedHashSet<String> unique = new LinkedHashSet<>();
        for (String value : values) {
            if (value == null) continue;
            String family = value.trim().toLowerCase(Locale.ROOT);
            if (family.isEmpty() || family.contains("/")) continue;
            unique.add(family);
        }
        ArrayList<String> out = new ArrayList<>(unique);
        out.sort(Comparator.naturalOrder());
        return List.copyOf(out);
    }

    public record Snapshot(
        List<String> dictionary,
        List<String> available
    ) {}
}
