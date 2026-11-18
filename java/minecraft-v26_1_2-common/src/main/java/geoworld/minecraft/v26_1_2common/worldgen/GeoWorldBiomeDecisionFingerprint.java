package geoworld.minecraft.v26_1_2common.worldgen;

import geoworld.minecraft.v26_1_2common.Minecraft26_1_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeDecisionFingerprint;

/** Minecraft 26.1.2 wrapper for shared 26.x verification. */
public final class GeoWorldBiomeDecisionFingerprint {
    private GeoWorldBiomeDecisionFingerprint() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomeDecisionFingerprint.run(
            Minecraft26_1_2Profile.INSTANCE,
            // v1.1.27 surface biome distribution, sampled at 24,576 positions per seed.
            java.util.Map.of(
                123456L, 0x9e0454016a938132L,
                -563659577384754347L, 0x02c39d0573249c76L
            ),
            args
        );
    }
}
