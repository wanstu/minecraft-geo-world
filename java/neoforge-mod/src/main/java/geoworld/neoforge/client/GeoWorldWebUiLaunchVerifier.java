package geoworld.neoforge.client;

import geoworld.minecraft.client.GeoWorldGameWebUi;
import geoworld.minecraft.common.tool.GeoWorldToolLauncher;

import java.lang.reflect.Field;
import java.net.InetSocketAddress;
import java.net.ServerSocket;
import java.util.Arrays;

public final class GeoWorldWebUiLaunchVerifier {
    private GeoWorldWebUiLaunchVerifier() {}

    public static void main(String[] args) throws Exception {
        GeoWorldGameWebUi.configure(GeoWorldWebUiPlatform.INSTANCE);
        boolean expectFallback =
            Arrays.asList(args).contains("--expect-fallback");
        ServerSocket blocker = null;
        try {
            if (expectFallback) {
                blocker = new ServerSocket();
                blocker.setReuseAddress(false);
                try {
                    blocker.bind(new InetSocketAddress("0.0.0.0", 8788));
                } catch (java.net.BindException alreadyOccupied) {
                    blocker.close();
                    blocker = null;
                }
            }

            int firstPort = verifyRunning(expectFallback, "initial");
            long firstPid = managedPid();
            killManagedProcess();
            long recoveredPid = waitForAutomaticRecovery(firstPid);
            int recoveredPort = GeoWorldGameWebUi.currentWebUiPort();
            // The previous listening socket can remain temporarily
            // unavailable after a forced native-process exit: the HTTP
            // status/session connections may still be in TCP TIME_WAIT.
            // The launcher correctly retries on a free dynamic port in that
            // case. Require a live, healthy replacement instead of assuming
            // that a recently closed TCP port can always be rebound.
            var recoveredUri = GeoWorldGameWebUi.currentWebUiUri();
            if (
                recoveredPort <= 0 ||
                recoveredUri == null ||
                recoveredUri.getPort() != recoveredPort
            ) {
                throw new AssertionError(
                    "GeoWorld WebUI watchdog recovered with a stale URI/port: " +
                    recoveredUri + " / " + recoveredPort
                );
            }
            var status = java.net.http.HttpClient.newHttpClient().send(
                java.net.http.HttpRequest.newBuilder(
                    recoveredUri.resolve("api/status")
                ).timeout(java.time.Duration.ofSeconds(3)).GET().build(),
                java.net.http.HttpResponse.BodyHandlers.discarding()
            );
            if (status.statusCode() != 200) {
                throw new AssertionError(
                    "GeoWorld WebUI watchdog recovered without a healthy " +
                    "native HTTP service: port=" + recoveredPort +
                    " status=" + status.statusCode()
                );
            }

            GeoWorldGameWebUi.stop();
            if (GeoWorldGameWebUi.isRunning()) {
                throw new AssertionError(
                    "GeoWorld WebUI still reports running after stop"
                );
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
                "GeoWorld WebUI launch verification passed: " +
                "mode=%s, preferred=8788, first=%d, recovered=%d, " +
                "firstPid=%d, recoveredPid=%d, restart=%d%n",
                expectFallback ? "occupied-port-fallback" : "normal",
                firstPort,
                recoveredPort,
                firstPid,
                recoveredPid,
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

    private static long waitForAutomaticRecovery(long killedPid)
        throws Exception {
        long deadline = System.nanoTime() +
            java.util.concurrent.TimeUnit.SECONDS.toNanos(15);
        while (System.nanoTime() < deadline) {
            if (GeoWorldGameWebUi.isRunning()) {
                long pid = managedPid();
                if (pid > 0 && pid != killedPid) {
                    return pid;
                }
            }
            Thread.sleep(200L);
        }
        throw new AssertionError(
            "GeoWorld WebUI watchdog did not recover after native process death"
        );
    }

    private static void killManagedProcess() throws Exception {
        GeoWorldToolLauncher.ManagedProcess process = managedProcess();
        if (process == null) {
            throw new AssertionError(
                "GeoWorld WebUI managed process missing before crash test"
            );
        }
        process.close();
        if (process.isAlive()) {
            throw new AssertionError(
                "GeoWorld WebUI managed process survived forced crash test"
            );
        }
    }

    private static long managedPid() throws Exception {
        GeoWorldToolLauncher.ManagedProcess process = managedProcess();
        return process == null ? -1L : process.pid();
    }

    private static GeoWorldToolLauncher.ManagedProcess managedProcess()
        throws Exception {
        Field field = GeoWorldGameWebUi.class.getDeclaredField("managed");
        field.setAccessible(true);
        return (GeoWorldToolLauncher.ManagedProcess)field.get(null);
    }
}
