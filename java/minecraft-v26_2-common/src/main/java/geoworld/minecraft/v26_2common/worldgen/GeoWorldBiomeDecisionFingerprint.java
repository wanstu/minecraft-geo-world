package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeDecisionFingerprint;

/** Minecraft 26.2 wrapper for shared 26.x verification. */
public final class GeoWorldBiomeDecisionFingerprint {
    private GeoWorldBiomeDecisionFingerprint() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomeDecisionFingerprint.run(
            Minecraft26_2Profile.INSTANCE,
            // Current-rule baselines; v1.1.15/v1.1.16 comparisons are identical.
            java.util.Map.of(
                123456L, 0xef19ea2e85ece8eaL,
                -563659577384754347L, 0x40e63264bc16545dL
            ),
            args
        );
    }
}
