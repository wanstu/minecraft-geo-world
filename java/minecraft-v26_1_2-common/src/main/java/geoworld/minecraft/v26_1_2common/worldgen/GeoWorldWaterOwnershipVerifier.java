package geoworld.minecraft.v26_1_2common.worldgen;

import geoworld.minecraft.v26_1_2common.Minecraft26_1_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26WaterOwnershipVerifier;

/** Minecraft 26.1.2 wrapper for shared 26.x water verification. */
public final class GeoWorldWaterOwnershipVerifier {
    private GeoWorldWaterOwnershipVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26WaterOwnershipVerifier.run(
            Minecraft26_1_2Profile.INSTANCE,
            args
        );
    }
}
