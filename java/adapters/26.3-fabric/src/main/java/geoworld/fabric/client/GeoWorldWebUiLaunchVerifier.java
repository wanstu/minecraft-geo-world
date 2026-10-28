package geoworld.fabric.client;

import java.net.InetSocketAddress;
import java.net.ServerSocket;
import java.util.Arrays;

public final class GeoWorldWebUiLaunchVerifier {
    private GeoWorldWebUiLaunchVerifier() {}

    public static void main(String[] args) throws Exception {
        boolean expectFallback =
            Arrays.asList(args).contains("--expect-fallback");
        ServerSocket blocker = null;
        try {
            if (expectFallback) {
                blocker = new ServerSocket();
                blocker.setReuseAddress(false);
                blocker.bind(new InetSocketAddress("0.0.0.0", 8788));
            }

            int firstPort = verifyRunning(expectFallback, "initial");

            GeoWorldGameWebUi.stop();
            if (GeoWorldGameWebUi.isRunning()) {
                throw new AssertionError("GeoWorld WebUI still reports running after stop");
            }
            if (
                GeoWorldGameWebUi.currentWebUiUri() != null ||
                GeoWorldGameWebUi.currentWebUiPort() != -1
            ) {
                throw new AssertionError(
                    "GeoWorld WebUI retained stale URI/port after stop"
                );
            }

            int restartPort = verifyRunning(expectFallback, "restart");
            System.out.printf(
                "GeoWorld WebUI launch verification passed: mode=%s, preferred=8788, first=%d, restart=%d%n",
                expectFallback ? "occupied-port-fallback" : "normal",
                firstPort,
                restartPort
            );
        } finally {
            GeoWorldGameWebUi.stop();
            if (blocker != null) {
                blocker.close();
            }
        }
    }

    private static int verifyRunning(
        boolean expectFallback,
        String phase
    ) {
        GeoWorldGameWebUi.start();
        if (!GeoWorldGameWebUi.isRunning()) {
            throw new AssertionError(
                "GeoWorld WebUI failed to start during " + phase
            );
        }
        int port = GeoWorldGameWebUi.currentWebUiPort();
        if (port <= 0) {
            throw new AssertionError(
                "GeoWorld WebUI did not expose its actual port during " + phase
            );
        }
        if (expectFallback && port == 8788) {
            throw new AssertionError(
                "GeoWorld WebUI used occupied preferred port 8788 during " +
                phase
            );
        }
        var uri = GeoWorldGameWebUi.currentWebUiUri();
        if (uri == null || uri.getPort() != port) {
            throw new AssertionError(
                "dynamic WebUI URI does not match actual port during " +
                phase + ": " + uri + " / " + port
            );
        }
        return port;
    }
}
