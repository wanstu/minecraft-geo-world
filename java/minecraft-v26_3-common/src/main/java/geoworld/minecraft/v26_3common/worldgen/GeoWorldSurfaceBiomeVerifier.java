package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26SurfaceBiomeVerifier;

/** Minecraft 26.3 wrapper for shared 26.x verification/benchmark logic. */
public final class GeoWorldSurfaceBiomeVerifier {
    private GeoWorldSurfaceBiomeVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26SurfaceBiomeVerifier.run(
            Minecraft26_3Profile.INSTANCE,
            args
        );
    }
}
