package geoworld.minecraft.v1_21_1common.worldgen;

import geoworld.minecraft.v1_21_1common.Minecraft1_21_1Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeDecisionFingerprint;

/** Minecraft 1.21.1 wrapper for shared 26.x verification. */
public final class GeoWorldBiomeDecisionFingerprint {
    private GeoWorldBiomeDecisionFingerprint() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomeDecisionFingerprint.run(
            Minecraft1_21_1Profile.INSTANCE,
            java.util.Map.of(),
            args
        );
    }
}
