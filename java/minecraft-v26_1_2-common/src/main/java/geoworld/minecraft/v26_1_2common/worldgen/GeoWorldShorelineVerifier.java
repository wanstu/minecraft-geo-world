package geoworld.minecraft.v26_1_2common.worldgen;

import geoworld.minecraft.v26_1_2common.Minecraft26_1_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26ShorelineVerifier;

/** Minecraft 26.1.2 wrapper for shared 26.x shoreline verification. */
public final class GeoWorldShorelineVerifier {
    private GeoWorldShorelineVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26ShorelineVerifier.run(
            Minecraft26_1_2Profile.INSTANCE,
            args
        );
    }
}
