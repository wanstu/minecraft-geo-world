package geoworld.runtime.api;

import geoworld.runtime.GeoWorldFactsBinding;
import geoworld.runtime.GeoWorldQueryService;
import java.util.Optional;

/** The single consumer entry point for logical-server Overworld facts. */
public final class GeoWorldFacade {
    private GeoWorldFacade() {}

    public static String apiVersion() {
        return GeoWorldQueryService.API_VERSION;
    }

    public static String minSupportedApiVersion() {
        return GeoWorldQueryService.API_VERSION;
    }

    /**
     * Returns a ready snapshot, or empty before activation and after shutdown.
     * Performs no sampling, waiting, chunk access or file access on any thread.
     * The dimension is always Overworld, irrespective of a player's dimension.
     */
    public static Optional<WorldFacts> currentWorld() {
        return GeoWorldFactsBinding.currentWorld();
    }
}
