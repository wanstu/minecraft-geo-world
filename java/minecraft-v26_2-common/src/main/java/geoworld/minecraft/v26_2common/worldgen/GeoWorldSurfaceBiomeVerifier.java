package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26SurfaceBiomeVerifier;

/** Minecraft 26.2 wrapper for shared 26.x verification/benchmark logic. */
public final class GeoWorldSurfaceBiomeVerifier {
    private GeoWorldSurfaceBiomeVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26SurfaceBiomeVerifier.run(
            Minecraft26_2Profile.INSTANCE,
            args
        );
    }
}
