package geoworld.minecraft.v26common.runtime;

import com.mojang.logging.LogUtils;
import geoworld.minecraft.common.MinecraftVersionProfile;
import org.slf4j.Logger;
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

public final class GeoWorldV26WorldService {
    private static final Logger LOGGER = LogUtils.getLogger();

    private final MinecraftVersionProfile profile;
    private final Map<MinecraftServer, ActiveWorld> active =
        new ConcurrentHashMap<>();

    public GeoWorldV26WorldService(MinecraftVersionProfile profile) {
        this.profile = java.util.Objects.requireNonNull(profile, "profile");
    }

    public void activate(MinecraftServer server) {
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

            if (currentGenerator instanceof GeoWorldDynamicGenerator generator) {
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

            this.active.remove(server);
            LOGGER.info(
                "World {} is not a GeoWorld world; leaving vanilla worldgen untouched",
                worldDirectory
            );
        } catch (IOException | RuntimeException e) {
            this.active.remove(server);
            throw new IllegalStateException(
                "Failed to activate GeoWorld world at " + worldDirectory,
                e
            );
        }
    }

    private void activateDynamicWorld(
        MinecraftServer server,
        Path worldDirectory,
        long seed,
        GeoWorldDynamicGenerator generator,
        Optional<GeoWorldBinding.Binding> found
    ) throws IOException {
        GeoWorldBinding.Binding binding;
        GeoWorldV1 model;

        if (found.isPresent()) {
            binding = requireCompatibleBinding(found.get(), seed);
            model = GeoWorldBinding.loadModel(binding);
            generator.attachModel(seed, model, binding.modelPath());

            LOGGER.info(
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
                this.profile.minecraftVersion()
            );
            generator.attachModel(
                seed,
                model,
                binding.modelPath()
            );

            LOGGER.info(
                "Persisted GeoWorld model into save: {}",
                binding.modelPath()
            );
        }

        this.active.put(
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

    private void activateStaticBoundWorld(
        MinecraftServer server,
        Path worldDirectory,
        GeoWorldBinding.Binding binding
    ) throws IOException {
        GeoWorldV1 model = GeoWorldBinding.loadModel(binding);
        this.active.put(
            server,
            new ActiveWorld(
                worldDirectory,
                binding,
                model,
                false
            )
        );

        LOGGER.info(
            "Opened GeoWorld-bound static Preview save; current chunk generator is not GeoWorld, so only pre-generated chunks are GeoWorld-controlled"
        );
        logActivated(model, binding, false);
    }

    private GeoWorldBinding.Binding requireCompatibleBinding(
        GeoWorldBinding.Binding binding,
        long seed
    ) {
        if (!this.profile.minecraftVersion().equals(binding.minecraftVersion())) {
            throw new IllegalStateException(
                "GeoWorld binding targets Minecraft " +
                binding.minecraftVersion() +
                ", but this mod build targets " + this.profile.minecraftVersion()
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

    private void logActivated(
        GeoWorldV1 model,
        GeoWorldBinding.Binding binding,
        boolean dynamicWorldgen
    ) {
        GeoWorldV1.Manifest manifest = model.manifest();
        LOGGER.info(
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

    public void deactivate(MinecraftServer server) {
        ActiveWorld removed = this.active.remove(server);
        if (removed != null) {
            LOGGER.info(
                "Released GeoWorld model for {}",
                removed.worldDirectory()
            );
        }
    }

    public Optional<ActiveWorld> active(MinecraftServer server) {
        return Optional.ofNullable(this.active.get(server));
    }

    public boolean isActive(MinecraftServer server) {
        return this.active.containsKey(server);
    }

    public record ActiveWorld(
        Path worldDirectory,
        GeoWorldBinding.Binding binding,
        GeoWorldV1 model,
        boolean dynamicWorldgen
    ) {}
}
