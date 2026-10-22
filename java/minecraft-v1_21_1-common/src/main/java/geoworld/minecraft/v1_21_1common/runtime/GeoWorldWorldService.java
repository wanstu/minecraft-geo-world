package geoworld.minecraft.v1_21_1common.runtime;

import geoworld.minecraft.v26common.runtime.GeoWorldV26WorldService;
import geoworld.minecraft.v1_21_1common.Minecraft1_21_1Profile;
import net.minecraft.server.MinecraftServer;

import java.util.Optional;

/** Minecraft 1.21.1 facade over the shared 26.x world service. */
public final class GeoWorldWorldService {
    private static final GeoWorldV26WorldService CORE =
        new GeoWorldV26WorldService(Minecraft1_21_1Profile.INSTANCE);

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
