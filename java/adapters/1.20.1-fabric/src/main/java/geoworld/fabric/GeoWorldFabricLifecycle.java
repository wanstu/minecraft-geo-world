package geoworld.fabric;

import net.minecraft.server.MinecraftServer;

import java.util.Objects;
import java.util.function.Consumer;

/**
 * Loader-only lifecycle bridge used by the MinecraftServer mixin.
 * GeoWorld common code still sees the same ServerLifecycleAdapter contract.
 */
public final class GeoWorldFabricLifecycle {
    private static volatile Consumer<MinecraftServer> onReady = server -> {};
    private static volatile Consumer<MinecraftServer> onStopped = server -> {};

    private GeoWorldFabricLifecycle() {}

    static void register(
        Consumer<MinecraftServer> ready,
        Consumer<MinecraftServer> stopped
    ) {
        onReady = Objects.requireNonNull(ready, "ready");
        onStopped = Objects.requireNonNull(stopped, "stopped");
    }

    public static void serverReady(MinecraftServer server) {
        onReady.accept(server);
    }

    public static void serverStopped(MinecraftServer server) {
        onStopped.accept(server);
    }
}
