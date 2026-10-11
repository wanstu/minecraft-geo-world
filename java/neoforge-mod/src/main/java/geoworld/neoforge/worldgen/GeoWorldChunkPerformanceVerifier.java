package geoworld.neoforge.worldgen;

/** Ensures packaged build flags match the effective default profiling mode. */
public final class GeoWorldChunkPerformanceVerifier {
    private GeoWorldChunkPerformanceVerifier() {}

    public static void main(String[] args) {
        boolean expected = Boolean.parseBoolean(
            System.getProperty("geoworld.expectedBuildProfile", "false")
        );
        boolean actual = GeoWorldChunkPerformance.enabled();
        if (expected != actual) {
            throw new AssertionError(
                "chunk profiling mode expected=" + expected + ", actual=" + actual
            );
        }
        System.out.printf(
            "GeoWorld chunk performance profile PASS automaticDevMode=%s%n",
            actual
        );
    }
}
