package geoworld.minecraft.common.worldgen;

import geoworld.runtime.GeoWorldV1;

public final class GeoWorldSpawnPlanner {
    private GeoWorldSpawnPlanner() {}

    public static Candidate find(GeoWorldV1 runtime) {
        GeoWorldV1.Geometry geometry = runtime.manifest().geometry();

        double centerX = geometry.originX() + geometry.spanXBlocks() * 0.5;
        double centerZ = geometry.originZ() + geometry.spanZBlocks() * 0.5;
        double stepX = geometry.width() > 1
            ? geometry.spanXBlocks() / (geometry.width() - 1.0)
            : 128.0;
        double stepZ = geometry.height() > 1
            ? geometry.spanZBlocks() / (geometry.height() - 1.0)
            : 128.0;
        double sea = runtime.manifest().seaLevelNormalized();

        Candidate best = new Candidate(0, 0, 8, 8, Double.POSITIVE_INFINITY);
        double bestScore = Double.POSITIVE_INFINITY;

        for (int row = 0; row < geometry.height(); row++) {
            double z = geometry.originZ() + row * stepZ;
            for (int column = 0; column < geometry.width(); column++) {
                double x = geometry.originX() + column * stepX;

                int chunkX = Math.floorDiv((int)Math.floor(x), 16);
                int chunkZ = Math.floorDiv((int)Math.floor(z), 16);
                int sampleX = chunkX * 16 + 8;
                int sampleZ = chunkZ * 16 + 8;

                GeoWorldV1.RuntimeSample center = runtime.sample(
                    sampleX,
                    sampleZ
                );
                if (!isSpawnLand(center, sea)) {
                    continue;
                }

                double relief = localRelief(
                    runtime,
                    sampleX,
                    sampleZ,
                    sea
                );
                if (!Double.isFinite(relief)) {
                    continue;
                }

                double nx = geometry.spanXBlocks() > 0.0
                    ? (sampleX - centerX) / geometry.spanXBlocks()
                    : 0.0;
                double nz = geometry.spanZBlocks() > 0.0
                    ? (sampleZ - centerZ) / geometry.spanZBlocks()
                    : 0.0;
                double distance = nx * nx + nz * nz;

                double targetHeight = sea + 0.08;
                double heightPenalty = Math.abs(
                    center.groundHeight() - targetHeight
                );
                double score =
                    distance * 12.0 +
                    relief * 6.0 +
                    heightPenalty;

                if (score < bestScore) {
                    bestScore = score;
                    best = new Candidate(
                        chunkX,
                        chunkZ,
                        sampleX,
                        sampleZ,
                        score
                    );
                }
            }
        }

        return best;
    }

    public static boolean isSpawnLand(
        GeoWorldV1.RuntimeSample sample,
        double sea
    ) {
        if (!sample.sourceLand()) {
            return false;
        }
        if (!"None".equals(sample.waterKind())) {
            return false;
        }
        double ground = sample.groundHeight();
        return ground >= sea + 0.02 && ground <= 0.82;
    }

    private static double localRelief(
        GeoWorldV1 runtime,
        int x,
        int z,
        double sea
    ) {
        int[] offsets = {-12, 0, 12};
        double min = Double.POSITIVE_INFINITY;
        double max = Double.NEGATIVE_INFINITY;

        for (int dz : offsets) {
            for (int dx : offsets) {
                GeoWorldV1.RuntimeSample sample = runtime.sample(
                    x + dx,
                    z + dz
                );
                if (!isSpawnLand(sample, sea)) {
                    return Double.POSITIVE_INFINITY;
                }
                min = Math.min(min, sample.groundHeight());
                max = Math.max(max, sample.groundHeight());
            }
        }
        return max - min;
    }

    public record Candidate(
        int chunkX,
        int chunkZ,
        int blockX,
        int blockZ,
        double score
    ) {}
}
