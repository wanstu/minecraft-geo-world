package geoworld.minecraft.v1_21_1common.worldgen;

import geoworld.minecraft.v1_21_1common.Minecraft1_21_1Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomePresetVerifier;

/** Minecraft 1.21.1 wrapper for shared 26.x verification. */
public final class GeoWorldBiomePresetVerifier {
    private GeoWorldBiomePresetVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomePresetVerifier.run(
            Minecraft1_21_1Profile.INSTANCE,
            args
        );
    }
}
