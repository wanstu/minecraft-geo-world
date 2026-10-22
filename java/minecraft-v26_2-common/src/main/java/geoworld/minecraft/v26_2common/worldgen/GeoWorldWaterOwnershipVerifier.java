package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26WaterOwnershipVerifier;

/** Minecraft 26.2 wrapper for shared 26.x water verification. */
public final class GeoWorldWaterOwnershipVerifier {
    private GeoWorldWaterOwnershipVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26WaterOwnershipVerifier.run(
            Minecraft26_2Profile.INSTANCE,
            args
        );
    }
}
