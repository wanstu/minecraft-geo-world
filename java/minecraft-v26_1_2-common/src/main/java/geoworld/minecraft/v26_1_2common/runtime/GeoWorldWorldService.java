package geoworld.minecraft.v26_1_2common.runtime;

import geoworld.minecraft.v26common.runtime.GeoWorldV26WorldService;
import geoworld.minecraft.v26_1_2common.Minecraft26_1_2Profile;
import net.minecraft.server.MinecraftServer;

import java.util.Optional;

/** Minecraft 26.1.2 facade over the shared 26.x world service. */
public final class GeoWorldWorldService {
    private static final GeoWorldV26WorldService CORE =
        new GeoWorldV26WorldService(Minecraft26_1_2Profile.INSTANCE);

    private GeoWorldWorldService() {}

    public static void activate(MinecraftServer server) {
        CORE.activate(server);
    }

    public static void deactivate(MinecraftServer server) {
        CORE.deactivate(server);
    }

    public static Optional<GeoWorldV26WorldService.ActiveWorld> active(
        MinecraftServer server
    ) {
        return CORE.active(server);
    }

    public static boolean isActive(MinecraftServer server) {
        return CORE.isActive(server);
    }
}
