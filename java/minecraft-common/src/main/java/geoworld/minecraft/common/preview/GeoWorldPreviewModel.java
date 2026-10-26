package geoworld.minecraft.common.preview;

import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.runtime.GeoWorldV1;

import java.util.Objects;

/**
 * Loader- and UI-neutral view of the exact locked GeoWorld model used by map
 * previews. Keeping preview reads behind this wrapper prevents preview code
 * from silently regenerating facts with current defaults or a different
 * profile.
 */
public final class GeoWorldPreviewModel {
    private final GeoWorldV1 model;

    public GeoWorldPreviewModel(GeoWorldV1 model) {
        this.model = Objects.requireNonNull(model, "model");
    }

    public static GeoWorldPreviewModel fromGenerated(
        GeoWorldEngineBridge.GeneratedModel generated
    ) {
        Objects.requireNonNull(generated, "generated");
        return new GeoWorldPreviewModel(generated.model());
    }

    public GeoWorldV1.Manifest manifest() {
        return this.model.manifest();
    }

    public GeoWorldV1.RuntimeSample sample(double blockX, double blockZ) {
        return this.model.sample(blockX, blockZ);
    }

    public GeoWorldV1.ClimateSample sampleClimate(
        double blockX,
        double blockZ
    ) {
        return this.model.sampleClimate(blockX, blockZ);
    }

    public GeoWorldV1 lockedModel() {
        return this.model;
    }
}
