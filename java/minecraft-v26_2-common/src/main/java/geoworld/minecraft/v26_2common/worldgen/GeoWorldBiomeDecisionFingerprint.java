package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeDecisionFingerprint;

/** Minecraft 26.2 wrapper for shared 26.x verification. */
public final class GeoWorldBiomeDecisionFingerprint {
    private GeoWorldBiomeDecisionFingerprint() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomeDecisionFingerprint.run(
            Minecraft26_2Profile.INSTANCE,
            // v1.1.28 coastal bathymetry and surface biome distribution, sampled at 24,576 positions per seed.
            java.util.Map.of(
                123456L, 0x49de0d5905aa28f7L,
                -563659577384754347L, 0x647128003a630a3eL
            ),
            args
        );
    }
}
