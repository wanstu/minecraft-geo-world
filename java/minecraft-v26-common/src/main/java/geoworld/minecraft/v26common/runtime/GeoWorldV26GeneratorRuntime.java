package geoworld.minecraft.v26common.runtime;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.io.IOException;
import java.nio.file.Path;
import java.util.List;
import java.util.Optional;
import java.util.function.Consumer;

/**
 * Shared GeoWorld model lifecycle for Minecraft 26.x generators.
 */
public final class GeoWorldV26GeneratorRuntime {
    private final GeoWorldCreationConfig config;
    private final Consumer<GeoWorldV1> modelAttacher;
    private final Object modelLock = new Object();

    private volatile long modelSeed = Long.MIN_VALUE;
    private volatile GeoWorldV1 model;
    private volatile Path modelPath;

    public GeoWorldV26GeneratorRuntime(
        GeoWorldCreationConfig config,
        Consumer<GeoWorldV1> modelAttacher
    ) {
        if (config == null) throw new NullPointerException("config");
        List<String> errors = config.validationErrors();
        if (!errors.isEmpty()) {
            throw new IllegalArgumentException(errors.get(0));
        }
        this.config = config;
        this.modelAttacher = java.util.Objects.requireNonNull(
            modelAttacher,
            "modelAttacher"
        );
    }

    public GeoWorldCreationConfig config() {
        return config;
    }

    public long modelSeed() {
        return modelSeed;
    }

    public GeoWorldV1 currentModel() {
        return model;
    }

    public GeoWorldV1 ensureModel(long seed) {
        GeoWorldV1 current = this.model;
        if (current != null && this.modelSeed == seed) {
            return current;
        }

        synchronized (this.modelLock) {
            current = this.model;
            if (current != null && this.modelSeed == seed) {
                return current;
            }

            try {
                GeoWorldEngineBridge.GeneratedModel generated =
                    GeoWorldEngineBridge.generate(seed, this.config);
                attachModel(
                    seed,
                    generated.model(),
                    generated.path()
                );
                return generated.model();
            } catch (IOException | InterruptedException e) {
                if (e instanceof InterruptedException) {
                    Thread.currentThread().interrupt();
                }
                throw new IllegalStateException(
                    "Failed to prepare GeoWorld model for seed " + seed,
                    e
                );
            }
        }
    }

    public void attachModel(
        long seed,
        GeoWorldV1 model,
        Path sourcePath
    ) {
        if (model == null) throw new NullPointerException("model");
        if (model.manifest().seed() != seed) {
            throw new IllegalArgumentException(
                "GeoWorld model seed mismatch: model=" +
                model.manifest().seed() +
                " expected=" +
                seed
            );
        }

        synchronized (this.modelLock) {
            this.modelSeed = seed;
            this.model = model;
            this.modelPath = sourcePath;
            this.modelAttacher.accept(model);
        }
    }

    public Optional<Path> modelPath() {
        return Optional.ofNullable(modelPath);
    }
}
