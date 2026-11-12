package geoworld.minecraft.v1_21_1common.runtime;

import geoworld.minecraft.v26common.runtime.GeoWorldV26WorldService;
import geoworld.minecraft.v1_21_1common.Minecraft1_21_1Profile;
import geoworld.minecraft.common.worldgen.GeoWorldSpawnPlanner;
import geoworld.minecraft.common.worldgen.GeoWorldSpawnHeight;
import net.minecraft.core.BlockPos;
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

    public static void ensureSafeSpawn(MinecraftServer server) {
        CORE.active(server).ifPresent(active -> {
            if (!active.dynamicWorldgen()) {
                return;
            }

            var overworld = server.overworld();
            BlockPos current = overworld.getSharedSpawnPos();
            if (
                Math.floorDiv(current.getX(), 16) != 0 ||
                Math.floorDiv(current.getZ(), 16) != 0
            ) {
                return;
            }

            var model = active.model();
            var candidate = GeoWorldSpawnPlanner.find(
                model,
                model.worldProfile().parameters().spawnPreference()
            );
            if (candidate.chunkX() == 0 && candidate.chunkZ() == 0) {
                return;
            }

            // Minecraft 26.3 consumes the planner result as a ChunkPos and
            // persists that chunk's block origin as the shared spawn.
            // Mirror that final behavior on 1.21.1 instead of using the
            // planner's center sample directly.
            int x = candidate.chunkX() * 16;
            int z = candidate.chunkZ() * 16;
            // The destination chunk may not have a populated Minecraft
            // heightmap on first boot. An uninitialized heightmap can report
            // minY (-64) even when GeoWorld has solid terrain above Y=100.
            // Sample the already-bound immutable model instead, preserving
            // the existing X/Z choice and the game's height mapping.
            int y = GeoWorldSpawnHeight.surfaceY(
                model,
                Minecraft1_21_1Profile.INSTANCE.heightMapping(),
                x,
                z
            );
            BlockPos spawn = new BlockPos(x, y, z);
            overworld.setDefaultSpawnPos(spawn, 0.0F);
        });
    }
}
