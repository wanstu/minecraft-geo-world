package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomePresetVerifier;

/** Minecraft 26.3 wrapper for shared 26.x verification. */
public final class GeoWorldBiomePresetVerifier {
    private GeoWorldBiomePresetVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomePresetVerifier.run(
            Minecraft26_3Profile.INSTANCE,
            args
        );
    }
}
