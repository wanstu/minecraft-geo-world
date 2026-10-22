package geoworld.minecraft.v26_3common;

import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.runtime.GeoWorldV1;

import java.util.Set;

/** Minecraft 26.3 facade over the shared 26.x biome realizer. */
public final class Minecraft26_3BiomeRealizer {
    private static final GeoWorldV26BiomeRealizer CORE =
        new GeoWorldV26BiomeRealizer(Minecraft26_3Profile.INSTANCE);

    private Minecraft26_3BiomeRealizer() {}

    static GeoWorldV26BiomeRealizer core() {
        return CORE;
    }

    public static Set<String> requiredBiomeIds() {
        return CORE.requiredBiomeIds();
    }

    public static String biomeId(
        GeoWorldV1 model,
        double blockX,
        double blockZ
    ) {
        return CORE.biomeId(model, blockX, blockZ);
    }

    public static String biomeId(
        GeoWorldV1 model,
        double blockX,
        double blockY,
        double blockZ
    ) {
        return CORE.biomeId(model, blockX, blockY, blockZ);
    }

    public static String biomeIdFromSample(
        GeoWorldV1 model,
        double blockX,
        double blockY,
        double blockZ,
        GeoWorldV1.RuntimeSample sample
    ) {
        return CORE.biomeIdFromSample(
            model,
            blockX,
            blockY,
            blockZ,
            sample
        );
    }
}
