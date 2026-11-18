package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeDecisionFingerprint;

/** Minecraft 26.2 wrapper for shared 26.x verification. */
public final class GeoWorldBiomeDecisionFingerprint {
    private GeoWorldBiomeDecisionFingerprint() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomeDecisionFingerprint.run(
            Minecraft26_2Profile.INSTANCE,
            // v1.1.27 surface biome distribution, sampled at 24,576 positions per seed.
            java.util.Map.of(
                123456L, 0x47ad3bb89998ec1fL,
                -563659577384754347L, 0xcece3d706c8725e6L
            ),
            args
        );
    }
}
