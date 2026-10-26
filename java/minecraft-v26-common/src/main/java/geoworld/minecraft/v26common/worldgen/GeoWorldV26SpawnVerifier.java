package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.worldgen.GeoWorldSpawnPlanner;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.common.worldgen.HeightMapping;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

public final class GeoWorldV26SpawnVerifier {
    private static final long OCEAN_CENTER_SEED = -1273393314970095624L;

    private GeoWorldV26SpawnVerifier() {}

    public static void run(
        MinecraftVersionProfile profile,
        String[] args
    ) throws Exception {
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

        double defaultInfluence = runtime
            .worldProfile()
            .parameters()
            .spawnPreference();
        GeoWorldSpawnPlanner.Candidate disabled =
            GeoWorldSpawnPlanner.find(runtime, 0.0);
        if (
            disabled.chunkX() != 0 ||
            disabled.chunkZ() != 0 ||
            disabled.blockX() != 8 ||
            disabled.blockZ() != 8
        ) {
            throw new AssertionError(
                "spawn preference 0 must disable GeoWorld origin bias: " +
                disabled
            );
        }

        GeoWorldSpawnPlanner.Candidate origin =
            GeoWorldSpawnPlanner.find(runtime, defaultInfluence);
        if (
            origin.chunkX() == disabled.chunkX() &&
            origin.chunkZ() == disabled.chunkZ()
        ) {
            throw new AssertionError(
                "default spawn preference did not move the ocean-center " +
                "regression seed onto habitable mainland"
            );
        }
        GeoWorldV1.RuntimeSample spawn = assertSafeSpawn(
            runtime,
            origin,
            runtime.manifest().seaLevelNormalized()
        );

        GeoWorldSpawnPlanner.Candidate strong =
            GeoWorldSpawnPlanner.find(runtime, 2.5);
        assertSafeSpawn(
            runtime,
            strong,
            runtime.manifest().seaLevelNormalized()
        );
        GeoWorldSpawnPlanner.Candidate strongAgain =
            GeoWorldSpawnPlanner.find(runtime, 2.5);
        if (!strong.equals(strongAgain)) {
            throw new AssertionError(
                "spawn planner is not deterministic at fixed influence: " +
                strong + " vs " + strongAgain
            );
        }

        HeightMapping heights = profile.heightMapping();
        int y = heights.boundaryY(spawn.groundHeight());

        System.out.printf(
            "GeoWorld spawn verification passed: seed=%d, center=%s/%s, " +
            "originChunk=%d,%d, spawn=%d,%d,%d, biome=%s, ground=%.6f, score=%.6f, " +
            "influence=%.2f, strongChunk=%d,%d%n",
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
            origin.score(),
            defaultInfluence,
            strong.chunkX(),
            strong.chunkZ()
        );
    }

    private static GeoWorldV1.RuntimeSample assertSafeSpawn(
        GeoWorldV1 runtime,
        GeoWorldSpawnPlanner.Candidate origin,
        double sea
    ) {
        GeoWorldV1.RuntimeSample spawn = runtime.sample(
            origin.blockX(),
            origin.blockZ()
        );
        if (!GeoWorldSpawnPlanner.isSpawnLand(spawn, sea)) {
            throw new AssertionError(
                "spawn is not safe habitable land: " + spawn
            );
        }

        int[] offsets = {-12, 0, 12};
        for (int dz : offsets) {
            for (int dx : offsets) {
                GeoWorldV1.RuntimeSample neighbor = runtime.sample(
                    origin.blockX() + dx,
                    origin.blockZ() + dz
                );
                if (!GeoWorldSpawnPlanner.isSpawnLand(neighbor, sea)) {
                    throw new AssertionError(
                        "spawn neighborhood unsafe at " +
                        (origin.blockX() + dx) + "," +
                        (origin.blockZ() + dz) + ": " + neighbor
                    );
                }
            }
        }
        return spawn;
    }
}
