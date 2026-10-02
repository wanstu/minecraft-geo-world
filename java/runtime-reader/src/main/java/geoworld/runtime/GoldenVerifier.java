package geoworld.runtime;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;
import java.util.Map;

public final class GoldenVerifier {
    private GoldenVerifier() {}

    public static void main(String[] args) throws Exception {
        verifyJsonIntegerPrecision();

        Path modelPath = args.length >= 1
            ? Path.of(args[0])
            : Path.of("internal/goldenvectors/testdata/runtime-v1.geoworld");
        Path goldenPath = args.length >= 2
            ? Path.of(args[1])
            : Path.of("internal/goldenvectors/testdata/runtime-v1.json");

        verify(modelPath, goldenPath);
    }

    private static void verifyJsonIntegerPrecision() {
        long[] values = {
            Long.MIN_VALUE,
            -9007199254740993L,
            9007199254740993L,
            3873414348276034653L,
            Long.MAX_VALUE
        };
        for (long expected : values) {
            String json = "{\"value\":" + expected + "}";
            Map<String, Object> object = Json.object(
                Json.parse(json),
                "integer precision probe"
            );
            long actual = Json.longValue(object, "value");
            if (actual != expected) {
                throw new AssertionError(
                    "JSON integer precision lost: got=" +
                    actual +
                    " want=" +
                    expected
                );
            }
        }
        System.out.println("PASS JSON long integer precision");
    }

    static void verify(Path modelPath, Path goldenPath) throws IOException {
        String goldenJson = Files.readString(goldenPath, StandardCharsets.UTF_8);
        Map<String, Object> document = Json.object(Json.parse(goldenJson), "golden document");

        String format = Json.string(document, "format");
        int formatVersion = Json.intValue(document, "formatVersion");
        int modelSchemaVersion = Json.intValue(document, "modelSchemaVersion");
        String generatorVersion = Json.string(document, "generatorVersion");
        double tolerance = Json.doubleValue(document, "floatTolerance");

        if (!"geoworld-runtime-golden".equals(format) || formatVersion != 1) {
            throw new AssertionError("unsupported golden format: " + format + " v" + formatVersion);
        }
        if (tolerance <= 0 || !Double.isFinite(tolerance)) {
            throw new AssertionError("invalid golden tolerance: " + tolerance);
        }

        GeoWorldV1 model = GeoWorldV1.load(modelPath);
        if (model.manifest().schemaVersion() != modelSchemaVersion) {
            throw new AssertionError(
                "model schema mismatch: " +
                model.manifest().schemaVersion() +
                " != " +
                modelSchemaVersion
            );
        }
        if (!model.manifest().generatorVersion().equals(generatorVersion)) {
            throw new AssertionError(
                "generator version mismatch: " +
                model.manifest().generatorVersion() +
                " != " +
                generatorVersion
            );
        }

        Map<String, Object> source = Json.object(Json.required(document, "source"), "source");
        if (Json.longValue(source, "seed") != model.manifest().seed()) {
            throw new AssertionError("golden/model seed mismatch");
        }
        verifyGeometry(
            Json.object(Json.required(source, "geometry"), "source.geometry"),
            model.manifest().geometry(),
            tolerance
        );

        List<Object> cases = Json.array(Json.required(document, "cases"), "cases");
        int passed = 0;
        for (int i = 0; i < cases.size(); i++) {
            Map<String, Object> testCase = Json.object(cases.get(i), "cases[" + i + "]");
            String name = Json.string(testCase, "name");
            String category = Json.string(testCase, "category");
            double x = Json.doubleValue(testCase, "x");
            double z = Json.doubleValue(testCase, "z");
            Map<String, Object> expected = Json.object(
                Json.required(testCase, "expected"),
                name + ".expected"
            );

            GeoWorldV1.RuntimeSample actual = model.sample(x, z);
            compare(name, category, x, z, actual, expected, tolerance);
            passed++;
            System.out.printf(
                "PASS %-24s %-18s @ (%9.2f, %9.2f)  water=%-5s biome=%s%n",
                name,
                category,
                x,
                z,
                actual.waterKind(),
                actual.biomeFamily()
            );
        }

        System.out.printf(
            "Golden verification passed: %d/%d cases, tolerance=%g, schema=%d, generator=%s%n",
            passed,
            cases.size(),
            tolerance,
            model.manifest().schemaVersion(),
            model.manifest().generatorVersion()
        );
    }

    private static void verifyGeometry(
        Map<String, Object> expected,
        GeoWorldV1.Geometry actual,
        double tolerance
    ) {
        exact("geometry.width", actual.width(), Json.intValue(expected, "width"));
        exact("geometry.height", actual.height(), Json.intValue(expected, "height"));
        near("geometry.originX", actual.originX(), Json.doubleValue(expected, "originX"), tolerance);
        near("geometry.originZ", actual.originZ(), Json.doubleValue(expected, "originZ"), tolerance);
        near(
            "geometry.spanXBlocks",
            actual.spanXBlocks(),
            Json.doubleValue(expected, "spanXBlocks"),
            tolerance
        );
        near(
            "geometry.spanZBlocks",
            actual.spanZBlocks(),
            Json.doubleValue(expected, "spanZBlocks"),
            tolerance
        );
    }

    private static void compare(
        String name,
        String category,
        double x,
        double z,
        GeoWorldV1.RuntimeSample actual,
        Map<String, Object> expected,
        double tolerance
    ) {
        String prefix =
            name + " (" + category + ") @ (" + x + "," + z + "): ";

        near(prefix + "groundHeight", actual.groundHeight(), number(expected, "groundHeight"), tolerance);
        exact(prefix + "waterKind", actual.waterKind(), Json.string(expected, "waterKind"));
        exact(
            prefix + "waterSurfaceDefined",
            actual.waterSurfaceDefined(),
            Json.bool(expected, "waterSurfaceDefined")
        );

        Object expectedSurface = expected.get("waterSurfaceElevation");
        if (expectedSurface == null) {
            if (actual.waterSurfaceElevation() != null) {
                throw new AssertionError(
                    prefix + "waterSurfaceElevation expected null got " +
                    actual.waterSurfaceElevation()
                );
            }
        } else {
            if (actual.waterSurfaceElevation() == null) {
                throw new AssertionError(prefix + "waterSurfaceElevation expected a value got null");
            }
            near(
                prefix + "waterSurfaceElevation",
                actual.waterSurfaceElevation(),
                ((Number) expectedSurface).doubleValue(),
                tolerance
            );
        }

        near(prefix + "waterDepth", actual.waterDepth(), number(expected, "waterDepth"), tolerance);
        near(
            prefix + "oceanEligibility",
            actual.oceanEligibility(),
            number(expected, "oceanEligibility"),
            tolerance
        );
        near(prefix + "riverWeight", actual.riverWeight(), number(expected, "riverWeight"), tolerance);

        exact(prefix + "sourceNode", actual.sourceNode(), Json.intValue(expected, "sourceNode"));
        near(
            prefix + "sourceWeight",
            actual.sourceWeight(),
            number(expected, "sourceWeight"),
            tolerance
        );
        exact(prefix + "sourceLand", actual.sourceLand(), Json.bool(expected, "sourceLand"));

        exact(prefix + "ecoRegion", actual.ecoRegion(), Json.string(expected, "ecoRegion"));
        near(
            prefix + "regionConfidence",
            actual.regionConfidence(),
            number(expected, "regionConfidence"),
            tolerance
        );

        exact(prefix + "biomeFamily", actual.biomeFamily(), Json.string(expected, "biomeFamily"));
        exact(
            prefix + "terrainVariant",
            actual.terrainVariant(),
            Json.string(expected, "terrainVariant")
        );
        exact(
            prefix + "surfaceProfile",
            actual.surfaceProfile(),
            Json.string(expected, "surfaceProfile")
        );
        exact(
            prefix + "vegetationProfile",
            actual.vegetationProfile(),
            Json.string(expected, "vegetationProfile")
        );
        near(
            prefix + "profileConfidence",
            actual.profileConfidence(),
            number(expected, "profileConfidence"),
            tolerance
        );

        exact(
            prefix + "semanticFallback",
            actual.semanticFallback(),
            Json.bool(expected, "semanticFallback")
        );
        exact(
            prefix + "semanticMismatch",
            actual.semanticMismatch(),
            Json.bool(expected, "semanticMismatch")
        );
    }

    private static double number(Map<String, Object> object, String key) {
        Object value = Json.required(object, key);
        if (!(value instanceof Number number)) {
            throw new IllegalArgumentException(key + " must be numeric");
        }
        return number.doubleValue();
    }

    private static void near(String name, double got, double want, double tolerance) {
        if (!Double.isFinite(got) || Math.abs(got - want) > tolerance) {
            throw new AssertionError(
                name + " got=" + Double.toString(got) +
                " want=" + Double.toString(want) +
                " diff=" + Double.toString(Math.abs(got - want)) +
                " tolerance=" + tolerance
            );
        }
    }

    private static void exact(String name, Object got, Object want) {
        if (!got.equals(want)) {
            throw new AssertionError(name + " got=" + got + " want=" + want);
        }
    }

    private static void exact(String name, int got, int want) {
        if (got != want) {
            throw new AssertionError(name + " got=" + got + " want=" + want);
        }
    }

    private static void exact(String name, boolean got, boolean want) {
        if (got != want) {
            throw new AssertionError(name + " got=" + got + " want=" + want);
        }
    }
}
