package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26BiomeCoverageVerifier;

/** Minecraft 26.2 wrapper for shared 26.x verification. */
public final class GeoWorldBiomeCoverageVerifier {
    private GeoWorldBiomeCoverageVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26BiomeCoverageVerifier.run(
            Minecraft26_2Profile.INSTANCE,
            args
        );
    }
}
