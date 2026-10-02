package geoworld.neoforge;

import geoworld.neoforge.worldgen.GeoWorldChunkGenerator;
import geoworld.runtime.GeoWorldBinding;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.server.MinecraftServer;
import net.minecraft.world.level.chunk.ChunkGenerator;
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

        ChunkGenerator currentGenerator =
            server.overworld().getChunkSource().getGenerator();
        long seed = server.overworld().getSeed();

        try {
            Optional<GeoWorldBinding.Binding> found =
                GeoWorldBinding.detect(worldDirectory);

            if (currentGenerator instanceof GeoWorldChunkGenerator generator) {
                activateDynamicWorld(
                    server,
                    worldDirectory,
                    seed,
                    generator,
                    found
                );
                return;
            }

            if (found.isPresent()) {
                activateStaticBoundWorld(
                    server,
                    worldDirectory,
                    found.get()
                );
                return;
            }

            ACTIVE.remove(server);
            GeoWorldMod.LOGGER.info(
                "World {} is not a GeoWorld world; leaving vanilla worldgen untouched",
                worldDirectory
            );
        } catch (IOException | RuntimeException e) {
            ACTIVE.remove(server);
            throw new IllegalStateException(
                "Failed to activate GeoWorld world at " + worldDirectory,
                e
            );
        }
    }

    private static void activateDynamicWorld(
        MinecraftServer server,
        Path worldDirectory,
        long seed,
        GeoWorldChunkGenerator generator,
        Optional<GeoWorldBinding.Binding> found
    ) throws IOException {
        GeoWorldBinding.Binding binding;
        GeoWorldV1 model;

        if (found.isPresent()) {
            binding = requireCompatibleBinding(found.get(), seed);
            model = GeoWorldBinding.loadModel(binding);
            generator.attachModel(seed, model, binding.modelPath());

            GeoWorldMod.LOGGER.info(
                "Loaded GeoWorld model from save binding: {}",
                binding.modelPath()
            );
        } else {
            model = generator.ensureModel(seed);
            Path sourcePath = generator
                .modelPath()
                .orElseThrow(() ->
                    new IllegalStateException(
                        "GeoWorld generator prepared a model without a source path"
                    )
                );
            binding = GeoWorldBinding.write(
                worldDirectory,
                sourcePath,
                "26.3"
            );
            generator.attachModel(
                seed,
                model,
                binding.modelPath()
            );

            GeoWorldMod.LOGGER.info(
                "Persisted GeoWorld model into save: {}",
                binding.modelPath()
            );
        }

        ACTIVE.put(
            server,
            new ActiveWorld(
                worldDirectory,
                binding,
                model,
                true
            )
        );
        logActivated(model, binding, true);
    }

    private static void activateStaticBoundWorld(
        MinecraftServer server,
        Path worldDirectory,
        GeoWorldBinding.Binding binding
    ) throws IOException {
        GeoWorldV1 model = GeoWorldBinding.loadModel(binding);
        ACTIVE.put(
            server,
            new ActiveWorld(
                worldDirectory,
                binding,
                model,
                false
            )
        );

        GeoWorldMod.LOGGER.info(
            "Opened GeoWorld-bound static Preview save; current chunk generator is not GeoWorld, so only pre-generated chunks are GeoWorld-controlled"
        );
        logActivated(model, binding, false);
    }

    private static GeoWorldBinding.Binding requireCompatibleBinding(
        GeoWorldBinding.Binding binding,
        long seed
    ) {
        if (!"26.3".equals(binding.minecraftVersion())) {
            throw new IllegalStateException(
                "GeoWorld binding targets Minecraft " +
                binding.minecraftVersion() +
                ", but this mod build targets 26.3"
            );
        }
        if (binding.seed() != seed) {
            throw new IllegalStateException(
                "GeoWorld binding seed " +
                binding.seed() +
                " does not match world seed " +
                seed
            );
        }
        return binding;
    }

    private static void logActivated(
        GeoWorldV1 model,
        GeoWorldBinding.Binding binding,
        boolean dynamicWorldgen
    ) {
        GeoWorldV1.Manifest manifest = model.manifest();
        GeoWorldMod.LOGGER.info(
            "Activated GeoWorld model: dynamicWorldgen={}, seed={}, schema={}, generator={}, grid={}x{}, world={}x{} blocks, model={}",
            dynamicWorldgen,
            manifest.seed(),
            manifest.schemaVersion(),
            manifest.generatorVersion(),
            manifest.geometry().width(),
            manifest.geometry().height(),
            manifest.geometry().spanXBlocks(),
            manifest.geometry().spanZBlocks(),
            binding.modelPath()
        );
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
        GeoWorldV1 model,
        boolean dynamicWorldgen
    ) {}
}
