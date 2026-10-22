package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;

/** Stable comparison fingerprints for biome decisions before/after optimizations. */
public final class GeoWorldV26BiomeDecisionFingerprint {
    private GeoWorldV26BiomeDecisionFingerprint() {}

    public static void run(
        MinecraftVersionProfile profile,
        java.util.Map<Long, Long> expectedFingerprints,
        String[] args
    ) throws Exception {
        GeoWorldV26BiomeRealizer realizer =
            new GeoWorldV26BiomeRealizer(profile);
        long seed = args.length > 0 ? Long.parseLong(args[0]) : 123456L;
        var model = GeoWorldEngineBridge.generate(seed, GeoWorldCreationConfig.defaults()).model();
        var geo = model.manifest().geometry();
        var mapper = profile.heightMapping();
        long hash = 0xcbf29ce484222325L;
        int n = 0;
        // Broad geography, coastline, rivers and high peaks, at fixed heights.
        for (int row = 0; row < 64; row++) {
            for (int col = 0; col < 96; col++) {
                double x = geo.originX() + (col + 0.5) * geo.spanXBlocks() / 96;
                double z = geo.originZ() + (row + 0.5) * geo.spanZBlocks() / 64;
                int ground = mapper.boundaryY(model.sample(x, z).groundHeight()) - 1;
                var sample = model.sample(x, z);
                for (int y : new int[]{ground, -30, 16, 92}) {
                    String id = realizer.biomeId(model, x, y, z);
                    String reused = realizer.biomeIdFromSample(
                        model, x, y, z, sample
                    );
                    if (!id.equals(reused)) {
                        throw new AssertionError("biome/sample mismatch at "+x+","+y+","+z);
                    }
                    for (int i = 0; i < id.length(); i++) {
                        hash ^= id.charAt(i);
                        hash *= 0x100000001b3L;
                    }
                    n++;
                }
            }
        }
        Long expected = expectedFingerprints.get(seed);
        if (expected != null && hash != expected) {
            throw new AssertionError(
                "biome decisions changed for seed=" + seed +
                ": expected=" + Long.toHexString(expected) +
                " actual=" + Long.toHexString(hash)
            );
        }
        System.out.printf("BiomeDecisionFingerprint seed=%d points=%d fnv64=%016x PASS%n",seed,n,hash);
    }
}
