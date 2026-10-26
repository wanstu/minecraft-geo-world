package geoworld.minecraft.common.preview;

import geoworld.runtime.GeoWorldV1;

import java.nio.file.Path;

/**
 * Executable acceptance verifier for PROFILE-03. Preview must read the exact
 * locked model used by runtime generation/query rather than rebuilding facts
 * from current defaults.
 */
public final class GeoWorldPreviewModelVerifier {
    private GeoWorldPreviewModelVerifier() {}

    public static void main(String[] args) throws Exception {
        Path modelPath = args.length >= 1
            ? Path.of(args[0])
            : Path.of("internal/goldenvectors/testdata/runtime-v1.geoworld");

        GeoWorldV1 model = GeoWorldV1.load(modelPath);
        GeoWorldPreviewModel preview = new GeoWorldPreviewModel(model);

        require(
            preview.lockedModel() == model,
            "preview wrapper did not retain the exact locked model instance"
        );

        var manifest = model.manifest();
        var previewManifest = preview.manifest();
        require(
            previewManifest.worldProfileVersion().equals(manifest.worldProfileVersion()) &&
                previewManifest.generationRulesVersion().equals(manifest.generationRulesVersion()) &&
                previewManifest.worldModelVersion().equals(manifest.worldModelVersion()) &&
                previewManifest.preset().equals(manifest.preset()) &&
                previewManifest.parameterSnapshotSha256().equals(manifest.parameterSnapshotSha256()),
            "preview identity diverged from locked model identity"
        );

        var g = manifest.geometry();
        double[][] samples = {
            {g.originX() + g.spanXBlocks() * 0.13, g.originZ() + g.spanZBlocks() * 0.17},
            {g.originX() + g.spanXBlocks() * 0.51, g.originZ() + g.spanZBlocks() * 0.49},
            {g.originX() + g.spanXBlocks() * 0.87, g.originZ() + g.spanZBlocks() * 0.79},
            {g.originX() - g.spanXBlocks() * 0.35, g.originZ() + g.spanZBlocks() * 0.50},
            {g.originX() + g.spanXBlocks() * 0.50, g.originZ() - g.spanZBlocks() * 0.60}
        };

        for (double[] point : samples) {
            double x = point[0];
            double z = point[1];

            var direct = model.sample(x, z);
            var shown = preview.sample(x, z);
            require(
                Double.compare(shown.groundHeight(), direct.groundHeight()) == 0 &&
                    shown.waterKind().equals(direct.waterKind()) &&
                    shown.sourceLand() == direct.sourceLand() &&
                    shown.biomeFamily().equals(direct.biomeFamily()) &&
                    shown.terrainVariant().equals(direct.terrainVariant()) &&
                    shown.surfaceProfile().equals(direct.surfaceProfile()) &&
                    shown.vegetationProfile().equals(direct.vegetationProfile()),
                "preview runtime sample diverged at " + x + "," + z
            );

            var directClimate = model.sampleClimate(x, z);
            var shownClimate = preview.sampleClimate(x, z);
            require(
                shownClimate.available() == directClimate.available() &&
                    Double.compare(shownClimate.annualMeanTempC(), directClimate.annualMeanTempC()) == 0 &&
                    Double.compare(shownClimate.annualPrecipMM(), directClimate.annualPrecipMM()) == 0 &&
                    Double.compare(shownClimate.monsoonIndex(), directClimate.monsoonIndex()) == 0 &&
                    Double.compare(shownClimate.windX(), directClimate.windX()) == 0 &&
                    Double.compare(shownClimate.windZ(), directClimate.windZ()) == 0,
                "preview climate sample diverged at " + x + "," + z
            );
        }

        System.out.println("PASS PROFILE-03");
        System.out.println("GeoWorld Preview Model Verifier: passed=1/1");
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new AssertionError(message);
        }
    }
}
