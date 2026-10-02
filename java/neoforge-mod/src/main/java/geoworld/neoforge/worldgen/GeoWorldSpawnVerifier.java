package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

public final class GeoWorldSpawnVerifier {
    private static final long OCEAN_CENTER_SEED = -1273393314970095624L;

    private GeoWorldSpawnVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldCreationConfig config = GeoWorldCreationConfig.defaults();
        GeoWorldEngineBridge.GeneratedModel generated =
            GeoWorldEngineBridge.generate(OCEAN_CENTER_SEED, config);
        GeoWorldV1 runtime = generated.model();

        GeoWorldV1.RuntimeSample center = runtime.sample(8.0, 8.0);
        if (!"Ocean".equals(center.waterKind()) || center.sourceLand()) {
            throw new AssertionError(
                "regression seed no longer has ocean center: " + center
            );
        }

        GeoWorldSpawnPlanner.Candidate origin =
            GeoWorldSpawnPlanner.find(runtime);
        GeoWorldV1.RuntimeSample spawn = runtime.sample(
            origin.blockX(),
            origin.blockZ()
        );

        if (!spawn.sourceLand()) {
            throw new AssertionError("spawn is not source land: " + spawn);
        }
        if (!"None".equals(spawn.waterKind())) {
            throw new AssertionError(
                "spawn contains water: " + spawn.waterKind()
            );
        }

        int[] offsets = {-12, 0, 12};
        for (int dz : offsets) {
            for (int dx : offsets) {
                GeoWorldV1.RuntimeSample neighbor = runtime.sample(
                    origin.blockX() + dx,
                    origin.blockZ() + dz
                );
                if (
                    !neighbor.sourceLand() ||
                    !"None".equals(neighbor.waterKind())
                ) {
                    throw new AssertionError(
                        "spawn neighborhood unsafe at " +
                        (origin.blockX() + dx) + "," +
                        (origin.blockZ() + dz) + ": " + neighbor
                    );
                }
            }
        }

        Minecraft26_3HeightMapping heights =
            new Minecraft26_3HeightMapping();
        int y = heights.boundaryY(spawn.groundHeight());

        System.out.printf(
            "GeoWorld spawn verification passed: seed=%d, center=%s/%s, " +
            "originChunk=%d,%d, spawn=%d,%d,%d, biome=%s, ground=%.6f, score=%.6f%n",
            OCEAN_CENTER_SEED,
            center.waterKind(),
            center.biomeFamily(),
            origin.chunkX(),
            origin.chunkZ(),
            origin.blockX(),
            y,
            origin.blockZ(),
            spawn.biomeFamily(),
            spawn.groundHeight(),
            origin.score()
        );
    }
}
