package geoworld.minecraft.v26common.runtime;

import geoworld.runtime.GeoWorldV1;

import java.nio.file.Path;
import java.util.Optional;

/**
 * Loader- and 26.x-version-neutral contract required by the shared world
 * binding service.
 */
public interface GeoWorldDynamicGenerator {
    GeoWorldV1 ensureModel(long seed);

    void attachModel(long seed, GeoWorldV1 model, Path sourcePath);

    Optional<Path> modelPath();
}
