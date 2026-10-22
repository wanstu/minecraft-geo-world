package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26SurfaceOutcropVerifier;

/** Minecraft 26.2 wrapper for shared 26.x verification/benchmark logic. */
public final class GeoWorldSurfaceOutcropVerifier {
    private GeoWorldSurfaceOutcropVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26SurfaceOutcropVerifier.run(
            Minecraft26_2Profile.INSTANCE,
            args
        );
    }
}
