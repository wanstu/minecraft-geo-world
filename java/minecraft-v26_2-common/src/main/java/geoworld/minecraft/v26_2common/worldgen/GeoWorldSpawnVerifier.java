package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26SpawnVerifier;

/** Minecraft 26.2 wrapper for shared 26.x verification. */
public final class GeoWorldSpawnVerifier {
    private GeoWorldSpawnVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26SpawnVerifier.run(
            Minecraft26_2Profile.INSTANCE,
            args
        );
    }
}
