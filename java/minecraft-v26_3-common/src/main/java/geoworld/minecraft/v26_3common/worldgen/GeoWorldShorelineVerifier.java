package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26ShorelineVerifier;

/** Minecraft 26.3 wrapper for shared 26.x shoreline verification. */
public final class GeoWorldShorelineVerifier {
    private GeoWorldShorelineVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26ShorelineVerifier.run(
            Minecraft26_3Profile.INSTANCE,
            args
        );
    }
}
