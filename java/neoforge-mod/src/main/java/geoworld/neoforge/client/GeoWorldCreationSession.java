package geoworld.neoforge.client;

import geoworld.neoforge.engine.GeoWorldEngineBridge;

import geoworld.neoforge.creation.GeoWorldCreationConfig;
import net.minecraft.client.gui.screens.worldselection.WorldCreationUiState;

import java.util.Map;
import java.util.Optional;
import java.util.WeakHashMap;

public final class GeoWorldCreationSession {
    private static final Map<WorldCreationUiState, Entry> ENTRIES =
        new WeakHashMap<>();

    private GeoWorldCreationSession() {}

    public static synchronized GeoWorldCreationConfig get(
        WorldCreationUiState state
    ) {
        Entry entry = ENTRIES.get(state);
        return entry == null
            ? GeoWorldCreationConfig.defaults()
            : entry.config();
    }

    public static synchronized void set(
        WorldCreationUiState state,
        GeoWorldCreationConfig config
    ) {
        if (state == null || config == null) {
            throw new NullPointerException("state/config");
        }
        Entry old = ENTRIES.get(state);
        GeoWorldEngineBridge.GeneratedModel generated =
            old != null &&
            old.generated() != null &&
            old.generated().config().equals(config)
                ? old.generated()
                : null;
        ENTRIES.put(state, new Entry(config, generated));
    }

    public static synchronized Optional<GeoWorldEngineBridge.GeneratedModel>
    generated(WorldCreationUiState state) {
        Entry entry = ENTRIES.get(state);
        return entry == null
            ? Optional.empty()
            : Optional.ofNullable(entry.generated());
    }

    public static synchronized void setGenerated(
        WorldCreationUiState state,
        GeoWorldEngineBridge.GeneratedModel generated
    ) {
        if (state == null || generated == null) {
            throw new NullPointerException("state/generated");
        }
        ENTRIES.put(state, new Entry(generated.config(), generated));
    }

    public static synchronized void invalidateGenerated(
        WorldCreationUiState state
    ) {
        Entry entry = ENTRIES.get(state);
        if (entry != null && entry.generated() != null) {
            ENTRIES.put(state, new Entry(entry.config(), null));
        }
    }

    public static synchronized void clear(WorldCreationUiState state) {
        ENTRIES.remove(state);
    }

    private record Entry(
        GeoWorldCreationConfig config,
        GeoWorldEngineBridge.GeneratedModel generated
    ) {}
}
