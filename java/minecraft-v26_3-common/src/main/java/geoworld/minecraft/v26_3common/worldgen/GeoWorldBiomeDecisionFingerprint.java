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
                // v1.1.28 coastal bathymetry and surface biome distribution, sampled at 24,576 positions per seed.
                123456L, 0x59c05de109371b59L,
                -563659577384754347L, 0x3604e5dfa830d878L
            ),
            args
        );
    }
}
