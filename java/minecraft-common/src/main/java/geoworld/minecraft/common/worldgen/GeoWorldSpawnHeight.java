package geoworld.minecraft.common.worldgen;

import geoworld.runtime.GeoWorldV1;

/**
 * Computes a safe spawn ground boundary from the frozen GeoWorld model.
 *
 * World.getHeight(...) may return the dimension's minimum Y when the
 * destination chunk has not yet populated its heightmap during first boot.
 * That must never become the persisted world spawn height.
 */
public final class GeoWorldSpawnHeight {
    private GeoWorldSpawnHeight() {}

    public static int surfaceY(
        GeoWorldV1 model,
        HeightMapping heightMapping,
        int blockX,
        int blockZ
    ) {
        java.util.Objects.requireNonNull(model, "model");
        java.util.Objects.requireNonNull(heightMapping, "heightMapping");
        double normalized = model.sample(blockX, blockZ).groundHeight();
        if (!Double.isFinite(normalized)) {
            throw new IllegalStateException(
                "GeoWorld spawn ground height is not finite at " +
                blockX + "," + blockZ
            );
        }
        return heightMapping.boundaryY(normalized);
    }
}
