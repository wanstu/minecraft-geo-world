package geoworld.minecraft.v1_21_1common.worldgen;

import geoworld.minecraft.v1_21_1common.Minecraft1_21_1Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26SpawnVerifier;

/** Minecraft 1.21.1 wrapper for shared 26.x verification. */
public final class GeoWorldSpawnVerifier {
    private GeoWorldSpawnVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26SpawnVerifier.run(
            Minecraft1_21_1Profile.INSTANCE,
            args
        );
    }
}
