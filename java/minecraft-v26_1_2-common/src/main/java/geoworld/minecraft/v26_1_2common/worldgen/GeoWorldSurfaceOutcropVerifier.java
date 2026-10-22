package geoworld.minecraft.v26_1_2common.worldgen;

import geoworld.minecraft.v26_1_2common.Minecraft26_1_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26SurfaceOutcropVerifier;

/** Minecraft 26.1.2 wrapper for shared 26.x verification/benchmark logic. */
public final class GeoWorldSurfaceOutcropVerifier {
    private GeoWorldSurfaceOutcropVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26SurfaceOutcropVerifier.run(
            Minecraft26_1_2Profile.INSTANCE,
            args
        );
    }
}
