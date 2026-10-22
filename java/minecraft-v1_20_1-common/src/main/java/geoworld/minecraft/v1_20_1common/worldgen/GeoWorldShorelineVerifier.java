package geoworld.minecraft.v1_20_1common.worldgen;

import geoworld.minecraft.v1_20_1common.Minecraft1_20_1Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26ShorelineVerifier;

/** Minecraft 1.20.1 wrapper for shared 26.x shoreline verification. */
public final class GeoWorldShorelineVerifier {
    private GeoWorldShorelineVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26ShorelineVerifier.run(
            Minecraft1_20_1Profile.INSTANCE,
            args
        );
    }
}
