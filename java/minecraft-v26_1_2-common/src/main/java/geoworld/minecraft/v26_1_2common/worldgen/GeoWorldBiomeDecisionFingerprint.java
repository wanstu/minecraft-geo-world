package geoworld.minecraft.v26_1_2common.worldgen;

import geoworld.minecraft.v26_1_2common.Minecraft26_1_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeDecisionFingerprint;

/** Minecraft 26.1.2 wrapper for shared 26.x verification. */
public final class GeoWorldBiomeDecisionFingerprint {
    private GeoWorldBiomeDecisionFingerprint() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomeDecisionFingerprint.run(
            Minecraft26_1_2Profile.INSTANCE,
            // Current-rule baselines; v1.1.15/v1.1.16 comparisons are identical.
            java.util.Map.of(
                123456L, 0xad15332a552036dcL,
                -563659577384754347L, 0xd628c61f9055ff2aL
            ),
            args
        );
    }
}
