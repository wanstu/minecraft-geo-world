package geoworld.minecraft.common.worldgen;

import geoworld.runtime.GeoWorldV1;
import geoworld.minecraft.v26common.MinecraftV26HeightMapping;

import java.nio.file.Path;

/**
 * Standalone regression verifier for the 1.21.1 spawn-height bug.
 * Invoke with one already-persisted GeoWorld model path, expected seed and
 * expected spawn chunk origin X/Z. Never creates or modifies world data.
 */
public final class GeoWorldSpawnHeightVerifier {
    private GeoWorldSpawnHeightVerifier() {}

    public static void main(String[] args) throws Exception {
        if (args.length != 4) {
            throw new IllegalArgumentException(
                "usage: <world.geoworld> <seed> <expectedX> <expectedZ>"
            );
        }
        GeoWorldV1 model = GeoWorldV1.load(Path.of(args[0]));
        long seed = Long.parseLong(args[1]);
        int expectedX = Integer.parseInt(args[2]);
        int expectedZ = Integer.parseInt(args[3]);
        if (model.manifest().seed() != seed) {
            throw new AssertionError("seed mismatch");
        }
        GeoWorldSpawnPlanner.Candidate choice =
            GeoWorldSpawnPlanner.find(
                model,
                model.worldProfile().parameters().spawnPreference()
            );
        int x = choice.chunkX() * 16;
        int z = choice.chunkZ() * 16;
        if (x != expectedX || z != expectedZ) {
            throw new AssertionError(
                "spawn X/Z unexpectedly changed: " + x + "," + z +
                " expected " + expectedX + "," + expectedZ
            );
        }

        var heights = new MinecraftV26HeightMapping();
        var sample = model.sample(x, z);
        if (!GeoWorldSpawnPlanner.isSpawnLand(
            sample, model.manifest().seaLevelNormalized()
        )) {
            throw new AssertionError("planned corner is not valid spawn land");
        }

        int safeY = GeoWorldSpawnHeight.surfaceY(model, heights, x, z);
        if (safeY <= heights.seaLevel() ||
            safeY <= heights.minY() ||
            safeY > heights.maxYExclusive()) {
            throw new AssertionError(
                "unsafe spawn model Y=" + safeY +
                " at " + x + "," + z
            );
        }
        if (safeY != heights.boundaryY(sample.groundHeight())) {
            throw new AssertionError("spawn height mapping differs from GeoWorld ground");
        }

        System.out.printf(
            java.util.Locale.ROOT,
            "PASS seed=%d spawnX=%d spawnY=%d spawnZ=%d ground=%.12f candidateScore=%.9f%n",
            seed, x, safeY, z, sample.groundHeight(), choice.score()
        );
    }
}
