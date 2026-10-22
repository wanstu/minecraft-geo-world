package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomePresetVerifier;

/** Minecraft 26.2 wrapper for shared 26.x verification. */
public final class GeoWorldBiomePresetVerifier {
    private GeoWorldBiomePresetVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomePresetVerifier.run(
            Minecraft26_2Profile.INSTANCE,
            args
        );
    }
}
