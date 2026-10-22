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
                123456L, 0x9f3de994862521fbL,
                -563659577384754347L, 0x71bfd1e1968b4db6L
            ),
            args
        );
    }
}
