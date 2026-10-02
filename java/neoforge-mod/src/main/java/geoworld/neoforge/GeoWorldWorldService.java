package geoworld.neoforge;

import geoworld.runtime.GeoWorldBinding;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.server.MinecraftServer;
import net.minecraft.world.level.storage.LevelResource;

import java.io.IOException;
import java.nio.file.Path;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;

public final class GeoWorldWorldService {
    private static final Map<MinecraftServer, ActiveWorld> ACTIVE =
        new ConcurrentHashMap<>();

    private GeoWorldWorldService() {}

    public static void activate(MinecraftServer server) {
        Path worldDirectory = server
            .getWorldPath(LevelResource.ROOT)
            .toAbsolutePath()
            .normalize();

        try {
            Optional<GeoWorldBinding.Binding> found =
                GeoWorldBinding.detect(worldDirectory);

            if (found.isEmpty()) {
                ACTIVE.remove(server);
                GeoWorldMod.LOGGER.info(
                    "World {} has no GeoWorld binding; vanilla worldgen remains untouched",
                    worldDirectory
                );
                return;
            }

            GeoWorldBinding.Binding binding = found.get();
            if (!"26.3".equals(binding.minecraftVersion())) {
                throw new IllegalStateException(
                    "GeoWorld binding targets Minecraft " +
                    binding.minecraftVersion() +
                    ", but this mod build targets 26.3"
                );
            }

            GeoWorldV1 model = GeoWorldBinding.loadModel(binding);
            ActiveWorld active = new ActiveWorld(
                worldDirectory,
                binding,
                model
            );
            ACTIVE.put(server, active);

            GeoWorldV1.Manifest manifest = model.manifest();
            GeoWorldMod.LOGGER.info(
                "Activated GeoWorld model: seed={}, schema={}, generator={}, grid={}x{}, world={}x{} blocks, model={}",
                manifest.seed(),
                manifest.schemaVersion(),
                manifest.generatorVersion(),
                manifest.geometry().width(),
                manifest.geometry().height(),
                manifest.geometry().spanXBlocks(),
                manifest.geometry().spanZBlocks(),
                binding.modelPath()
            );
        } catch (IOException | RuntimeException e) {
            ACTIVE.remove(server);
            throw new IllegalStateException(
                "Failed to activate GeoWorld world at " + worldDirectory,
                e
            );
        }
    }

    public static void deactivate(MinecraftServer server) {
        ActiveWorld removed = ACTIVE.remove(server);
        if (removed != null) {
            GeoWorldMod.LOGGER.info(
                "Released GeoWorld model for {}",
                removed.worldDirectory()
            );
        }
    }

    public static Optional<ActiveWorld> active(MinecraftServer server) {
        return Optional.ofNullable(ACTIVE.get(server));
    }

    public static boolean isActive(MinecraftServer server) {
        return ACTIVE.containsKey(server);
    }

    public record ActiveWorld(
        Path worldDirectory,
        GeoWorldBinding.Binding binding,
        GeoWorldV1 model
    ) {}
}
