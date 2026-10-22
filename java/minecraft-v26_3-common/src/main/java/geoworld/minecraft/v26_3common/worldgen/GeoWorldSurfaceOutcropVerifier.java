package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26SurfaceOutcropVerifier;

/** Minecraft 26.3 wrapper for shared 26.x verification/benchmark logic. */
public final class GeoWorldSurfaceOutcropVerifier {
    private GeoWorldSurfaceOutcropVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26SurfaceOutcropVerifier.run(
            Minecraft26_3Profile.INSTANCE,
            args
        );
    }
}
