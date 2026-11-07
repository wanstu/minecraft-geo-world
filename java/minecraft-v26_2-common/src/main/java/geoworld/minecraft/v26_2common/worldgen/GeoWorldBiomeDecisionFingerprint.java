package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeDecisionFingerprint;

/** Minecraft 26.2 wrapper for shared 26.x verification. */
public final class GeoWorldBiomeDecisionFingerprint {
    private GeoWorldBiomeDecisionFingerprint() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomeDecisionFingerprint.run(
            Minecraft26_2Profile.INSTANCE,
            // Locked to independently observed pre-Query 386aace and candidate results.
            java.util.Map.of(
                123456L, 0x6eff9f92530b16e6L,
                -563659577384754347L, 0xdf976bd29eaf4429L
            ),
            args
        );
    }
}
