package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeDecisionFingerprint;

/** Minecraft 26.3 wrapper for shared 26.x verification. */
public final class GeoWorldBiomeDecisionFingerprint {
    private GeoWorldBiomeDecisionFingerprint() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomeDecisionFingerprint.run(
            Minecraft26_3Profile.INSTANCE,
            java.util.Map.of(
                // Current baseline also matches v1.1.15 at all 24,576 sites.
                123456L, 0x261671873169ae5cL,
                -563659577384754347L, 0x8e74d3a605e736faL
            ),
            args
        );
    }
}
