package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26WaterOwnershipVerifier;

/** Minecraft 26.3 wrapper for shared 26.x water verification. */
public final class GeoWorldWaterOwnershipVerifier {
    private GeoWorldWaterOwnershipVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26WaterOwnershipVerifier.run(
            Minecraft26_3Profile.INSTANCE,
            args
        );
    }
}
