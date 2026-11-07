package geoworld.minecraft.client;

import geoworld.minecraft.common.tool.GeoWorldToolLauncher;
import geoworld.minecraft.v26common.runtime.GeoWorldV26WorldService;
import geoworld.minecraft.v26common.worldgen.GeoWorldOreCatalog;
import geoworld.minecraft.v26common.GeoWorldMinecraftIds;
import net.minecraft.world.level.Level;
import geoworld.runtime.GeoWorldBinding;
import geoworld.runtime.GeoWorldV1;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import net.minecraft.client.Minecraft;
import net.minecraft.server.MinecraftServer;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.resources.ResourceLocation;
import net.minecraft.server.packs.resources.ResourceManager;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.InetSocketAddress;
import java.net.ServerSocket;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.time.Duration;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.Properties;
import java.util.UUID;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

public final class GeoWorldGameWebUi {
    private static final Logger LOGGER =
        LoggerFactory.getLogger("GeoWorld/WebUI");
    private static final String LISTEN_HOST = "0.0.0.0";
    private static final int PREFERRED_PORT = 8788;
    private static final String TOKEN_HEADER = "X-GeoWorld-Game-Token";
    private static final String TOKEN = UUID.randomUUID().toString();
    private static final long HEALTH_CHECK_INTERVAL_NANOS =
        TimeUnit.SECONDS.toNanos(5);
    private static final long RESTART_BACKOFF_NANOS =
        TimeUnit.SECONDS.toNanos(5);
    private static final int HEALTH_FAILURES_BEFORE_RESTART = 3;
    private static final HttpClient HTTP = HttpClient.newBuilder()
        .connectTimeout(Duration.ofSeconds(1))
        .build();
    private static final ExecutorService CONTROL = Executors.newSingleThreadExecutor(
        runnable -> {
            Thread thread = new Thread(runnable, "GeoWorld-WebUI-Control");
            thread.setDaemon(true);
            return thread;
        }
    );
    private static final ScheduledExecutorService GAME_SYNC =
        Executors.newSingleThreadScheduledExecutor(runnable -> {
            Thread thread = new Thread(runnable, "GeoWorld-WebUI-GameSync");
            thread.setDaemon(true);
            return thread;
        });

    private static GeoWorldToolLauncher.ManagedProcess managed;
    private static volatile URI webUiUri;
    private static volatile URI sessionUri;
    private static volatile int webUiPort = -1;
    private static volatile MinecraftServer boundServer;
    private static volatile String pendingBindPayload;
    private static volatile boolean gameSessionBound;
    private static volatile long webUiGeneration;
    private static final AtomicBoolean gameSyncInFlight = new AtomicBoolean();
    private static final AtomicBoolean recoveryInFlight = new AtomicBoolean();
    private static volatile long lastChunkyStatusNanos;
    private static volatile long lastHealthCheckNanos;
    private static volatile long lastRecoveryAttemptNanos;
    private static volatile int consecutiveHealthFailures;
    private static volatile boolean expectedRunning;
    private static boolean syncLoopStarted;
    private static boolean shutdownHookRegistered;

    private static volatile Platform platform;

    private GeoWorldGameWebUi() {}

    public interface Platform {
        String gameClient();
        String minecraftVersion();
        Optional<GeoWorldV26WorldService.ActiveWorld> activeWorld(
            MinecraftServer server
        );
        boolean isSingleplayerOwner(
            MinecraftServer server,
            net.minecraft.server.level.ServerPlayer player
        );
        Object chunkyInstance();
        void openUri(Screen source, URI uri);
        void setScreen(Screen screen);
    }

    public static synchronized void configure(Platform value) {
        Objects.requireNonNull(value, "platform");
        if (platform != null && platform != value && isRunning()) {
            throw new IllegalStateException(
                "GeoWorld WebUI platform cannot change while running"
            );
        }
        platform = value;
    }

    private static Platform platform() {
        Platform value = platform;
        if (value == null) {
            throw new IllegalStateException(
                "GeoWorld WebUI platform bridge is not configured"
            );
        }
        return value;
    }

    public static void openInBrowser(Screen source) {
        if (!isRunning()) {
            start();
        }
        URI uri = currentWebUiUri();
        if (uri != null) {
            platform().openUri(source, uri);
        }
    }

    public static void setScreen(Screen screen) {
        platform().setScreen(screen);
    }
    public static synchronized void start() {
        start(-1);
    }

    private static synchronized void start(int preferredRecoveryPort) {
        expectedRunning = true;
        if (managed != null && managed.isAlive()) {
            return;
        }
        webUiGeneration++;
        GeoWorldToolLauncher.ManagedProcess stale = managed;
        managed = null;
        webUiUri = null;
        sessionUri = null;
        webUiPort = -1;
        pendingBindPayload = null;
        gameSessionBound = false;
        if (stale != null) {
            stale.close();
        }
        try {
            WebUiLaunch launch =
                launchWebUiWithRetry(preferredRecoveryPort);
            managed = launch.process();
            webUiPort = launch.port();
            webUiUri = launch.webUiUri();
            sessionUri = launch.sessionUri();
            System.setProperty("geoworld.webui.url", webUiUri.toString());
            consecutiveHealthFailures = 0;
            lastHealthCheckNanos = System.nanoTime();
            LOGGER.info(
                "GeoWorld WebUI started at {} pid={} (browser auto-open disabled)",
                webUiUri,
                managed.pid()
            );
            ensureGameSyncLoop();
            if (!shutdownHookRegistered) {
                Runtime.getRuntime().addShutdownHook(
                    new Thread(GeoWorldGameWebUi::stop, "GeoWorld-WebUI-Shutdown")
                );
                shutdownHookRegistered = true;
            }
            // Startup status is best-effort. Do not let a long retry
            // occupy the single control executor and delay the real world bind.
            postSessionAsync(
                unboundPayload(
                    "menu",
                    "Minecraft 已启动，当前尚未进入 GeoWorld 世界"
                ),
                3
            );
        } catch (Exception e) {
            GeoWorldToolLauncher.ManagedProcess failed = managed;
            managed = null;
            webUiUri = null;
            sessionUri = null;
            webUiPort = -1;
            System.clearProperty("geoworld.webui.url");
            pendingBindPayload = null;
            gameSessionBound = false;
            gameSyncInFlight.set(false);
            if (failed != null) {
                failed.close();
            }
            LOGGER.warn(
                "Unable to start GeoWorld WebUI; game startup will continue normally",
                e
            );
        }
    }

    public static void bindAfterServerStarted(MinecraftServer server) {
        // SERVER_STARTED listeners are not an ordering contract across the
        // common and client entrypoints. Queue from our own executor so the
        // actual bind runs on the server only after the current lifecycle
        // event (including GeoWorldWorldService.activate) has completed.
        GAME_SYNC.schedule(() -> {
            try {
                server.execute(() -> bind(server));
            } catch (Exception e) {
                LOGGER.warn(
                    "Unable to queue GeoWorld WebUI bind after server startup",
                    e
                );
            }
        }, 100L, TimeUnit.MILLISECONDS);
    }

    static void bind(MinecraftServer server) {
        start();
        LOGGER.info("GeoWorld WebUI bind check running after server startup");
        boundServer = null;
        pendingBindPayload = null;
        gameSessionBound = false;
        var active = platform().activeWorld(server);
        if (active.isEmpty()) {
            LOGGER.info(
                "GeoWorld WebUI bind check found no active GeoWorld model"
            );
            postSessionAsync(
                unboundPayload(
                    "non_geoworld",
                    "当前世界不是 GeoWorld 世界"
                ),
                8
            );
            return;
        }

        GeoWorldV26WorldService.ActiveWorld world = active.get();
        String compatibilityError = compatibilityError(world);
        if (compatibilityError != null) {
            LOGGER.warn(
                "GeoWorld WebUI refused automatic world binding: {}",
                compatibilityError
            );
            postSessionAsync(
                unboundPayload("incompatible", compatibilityError),
                8
            );
            return;
        }

        boundServer = server;
        pendingBindPayload = boundPayload(world);
        gameSessionBound = false;
        LOGGER.info(
            "GeoWorld WebUI prepared bind for world {}",
            world.worldDirectory()
        );
        // Do not use a finite one-shot bind retry here. World/model generation
        // can keep the WebUI busy for tens of seconds; the sync loop will keep
        // retrying this exact binding until it is acknowledged.
        syncGameState();
    }

    public static void unbind(String message) {
        boundServer = null;
        pendingBindPayload = null;
        gameSessionBound = false;
        postSessionAsync(
            unboundPayload(
                "menu",
                message == null || message.isBlank()
                    ? "已退出当前世界"
                    : message
            ),
            6
        );
    }

    public static synchronized void stop() {
        expectedRunning = false;
        webUiGeneration++;
        GeoWorldToolLauncher.ManagedProcess current = managed;
        managed = null;
        webUiUri = null;
        sessionUri = null;
        webUiPort = -1;
        System.clearProperty("geoworld.webui.url");
        boundServer = null;
        pendingBindPayload = null;
        gameSessionBound = false;
        gameSyncInFlight.set(false);
        consecutiveHealthFailures = 0;
        lastHealthCheckNanos = 0L;
        lastRecoveryAttemptNanos = 0L;
        if (current != null) {
            current.close();
            LOGGER.info("GeoWorld WebUI stopped");
        }
        // Executors are daemon singletons for the client JVM. Keep them alive
        // so a stopped WebUI can be started again in the same Minecraft process.
    }

    public static synchronized boolean isRunning() {
        return managed != null && managed.isAlive() && webUiUri != null;
    }

    public static URI currentWebUiUri() {
        return webUiUri;
    }

    public static int currentWebUiPort() {
        return webUiPort;
    }

    private record WebUiLaunch(
        GeoWorldToolLauncher.ManagedProcess process,
        int port,
        URI webUiUri,
        URI sessionUri
    ) {}

    private static WebUiLaunch launchWebUiWithRetry(
        int preferredRecoveryPort
    ) throws Exception {
        Exception lastError = null;
        for (int attempt = 1; attempt <= 3; attempt++) {
            int port;
            if (
                attempt == 1 &&
                preferredRecoveryPort > 0 &&
                canBind(preferredRecoveryPort)
            ) {
                port = preferredRecoveryPort;
            } else if (attempt == 1) {
                port = selectAvailablePort();
            } else {
                port = selectRandomAvailablePort();
            }
            URI ui = URI.create("http://127.0.0.1:" + port + "/");
            GeoWorldToolLauncher.ManagedProcess process = null;
            try {
                process = GeoWorldToolLauncher.startManaged(
                    "-addr=" + LISTEN_HOST + ":" + port,
                    "-open=false",
                    "-game-token=" + TOKEN,
                    "-game-client=" + platform().gameClient(),
                    "-game-minecraft-version=" +
                        platform().minecraftVersion()
                );
                if (waitForWebUiReady(process, ui)) {
                    return new WebUiLaunch(
                        process,
                        port,
                        ui,
                        ui.resolve("api/game/session")
                    );
                }
                lastError = new IllegalStateException(
                    "GeoWorld WebUI did not become ready on port " + port
                );
            } catch (Exception e) {
                lastError = e;
            }
            if (process != null) {
                process.close();
            }
            LOGGER.warn(
                "GeoWorld WebUI launch attempt {} failed on port {}; retrying with another port",
                attempt,
                port
            );
        }
        throw new IllegalStateException(
            "GeoWorld WebUI failed to start after dynamic-port retries",
            lastError
        );
    }

    private static boolean waitForWebUiReady(
        GeoWorldToolLauncher.ManagedProcess process,
        URI uri
    ) {
        URI statusUri = uri.resolve("api/status");
        for (int i = 0; i < 20; i++) {
            if (!process.isAlive()) {
                return false;
            }
            try {
                HttpRequest request = HttpRequest.newBuilder(statusUri)
                    .timeout(Duration.ofMillis(350))
                    .GET()
                    .build();
                HttpResponse<Void> response = HTTP.send(
                    request,
                    HttpResponse.BodyHandlers.discarding()
                );
                if (
                    response.statusCode() >= 200 &&
                    response.statusCode() < 300
                ) {
                    return true;
                }
            } catch (Exception ignored) {
                // The native WebUI process may still be binding its socket.
            }
            try {
                Thread.sleep(100L);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                return false;
            }
        }
        return false;
    }

    private static int selectRandomAvailablePort() throws Exception {
        try (ServerSocket socket = new ServerSocket()) {
            socket.setReuseAddress(false);
            socket.bind(new InetSocketAddress(LISTEN_HOST, 0));
            int port = socket.getLocalPort();
            if (port <= 0) {
                throw new IllegalStateException("无法分配 GeoWorld WebUI 动态端口");
            }
            return port;
        }
    }

    private static int selectAvailablePort() throws Exception {
        if (canBind(PREFERRED_PORT)) {
            return PREFERRED_PORT;
        }
        int port = selectRandomAvailablePort();
        LOGGER.info(
            "Preferred GeoWorld WebUI port {} is busy; using dynamic port {}",
            PREFERRED_PORT,
            port
        );
        return port;
    }

    private static boolean canBind(int port) {
        try (ServerSocket socket = new ServerSocket()) {
            socket.setReuseAddress(false);
            socket.bind(new InetSocketAddress(LISTEN_HOST, port));
            return true;
        } catch (Exception ignored) {
            return false;
        }
    }

    private static synchronized void ensureGameSyncLoop() {
        if (syncLoopStarted) {
            return;
        }
        GAME_SYNC.scheduleWithFixedDelay(
            GeoWorldGameWebUi::syncGameState,
            200L,
            1000L,
            TimeUnit.MILLISECONDS
        );
        syncLoopStarted = true;
    }

    private static void syncGameState() {
        if (!expectedRunning) {
            return;
        }

        if (!isRunning()) {
            scheduleRecovery("native WebUI process is not running");
            return;
        }

        long now = System.nanoTime();
        if (
            now - lastHealthCheckNanos >= HEALTH_CHECK_INTERVAL_NANOS
        ) {
            lastHealthCheckNanos = now;
            if (currentWebUiHealthy()) {
                consecutiveHealthFailures = 0;
            } else {
                int failures = ++consecutiveHealthFailures;
                LOGGER.warn(
                    "GeoWorld WebUI health check failed ({}/{}); uri={}",
                    failures,
                    HEALTH_FAILURES_BEFORE_RESTART,
                    webUiUri
                );
                if (failures >= HEALTH_FAILURES_BEFORE_RESTART) {
                    scheduleRecovery(
                        "health endpoint failed repeatedly"
                    );
                }
                return;
            }
        }

        MinecraftServer server = boundServer;
        if (server == null) {
            return;
        }

        long generation = webUiGeneration;
        if (!gameSyncInFlight.compareAndSet(false, true)) {
            return;
        }

        String bindPayload = pendingBindPayload;
        if (!gameSessionBound && bindPayload != null) {
            CONTROL.execute(() -> {
                try {
                    if (
                        postSession(bindPayload, 1, generation) &&
                        boundServer == server &&
                        generation == webUiGeneration
                    ) {
                        gameSessionBound = true;
                        LOGGER.info(
                            "GeoWorld WebUI game session bound; live player sync enabled"
                        );
                    }
                } finally {
                    gameSyncInFlight.set(false);
                }
            });
            return;
        }

        if (!gameSessionBound) {
            gameSyncInFlight.set(false);
            return;
        }

        try {
            server.execute(() -> {
                if (
                    boundServer != server ||
                    generation != webUiGeneration ||
                    !gameSessionBound
                ) {
                    gameSyncInFlight.set(false);
                    return;
                }
                var player = singleplayerOwner(server);
                if (player == null) {
                    gameSyncInFlight.set(false);
                    return;
                }
                String payload = playerPayload(
                    player.getX(),
                    player.getY(),
                    player.getZ(),
                    player.getYRot(),
                    player.getXRot(),
                    GeoWorldMinecraftIds.resourceKeyId(player.level().dimension())
                );
                CONTROL.execute(() -> {
                    try {
                        // Game control is latency-sensitive. Poll pending
                        // teleport/chunk commands before posting the periodic
                        // player-position heartbeat so a busy WebUI cannot
                        // starve direct control behind telemetry traffic.
                        pollGameControl(server, generation);
                        if (!postSession(payload, 1, generation)) {
                            // The native WebUI may have restarted or dropped its
                            // session. Re-bind before sending any more player
                            // updates instead of hammering a permanent 409.
                            gameSessionBound = false;
                            return;
                        }
                    } finally {
                        gameSyncInFlight.set(false);
                    }
                });
            });
        } catch (Exception e) {
            gameSyncInFlight.set(false);
            LOGGER.debug("Unable to schedule GeoWorld game-state sync", e);
        }
    }


    private static boolean currentWebUiHealthy() {
        GeoWorldToolLauncher.ManagedProcess current;
        URI base;
        synchronized (GeoWorldGameWebUi.class) {
            current = managed;
            base = webUiUri;
        }
        if (
            current == null ||
            !current.isAlive() ||
            base == null
        ) {
            return false;
        }

        try {
            HttpRequest request = HttpRequest.newBuilder(
                    base.resolve("api/status")
                )
                .timeout(Duration.ofSeconds(1))
                .GET()
                .build();
            HttpResponse<Void> response = HTTP.send(
                request,
                HttpResponse.BodyHandlers.discarding()
            );
            return (
                response.statusCode() >= 200 &&
                response.statusCode() < 300
            );
        } catch (Exception ignored) {
            return false;
        }
    }

    private static void scheduleRecovery(String reason) {
        if (!expectedRunning) {
            return;
        }

        long now = System.nanoTime();
        if (
            now - lastRecoveryAttemptNanos <
            RESTART_BACKOFF_NANOS
        ) {
            return;
        }
        if (!recoveryInFlight.compareAndSet(false, true)) {
            return;
        }
        lastRecoveryAttemptNanos = now;

        CONTROL.execute(() -> {
            try {
                if (!expectedRunning) {
                    return;
                }

                GeoWorldToolLauncher.ManagedProcess previous;
                Integer exitCode;
                long previousPid;
                int previousPort;
                synchronized (GeoWorldGameWebUi.class) {
                    previous = managed;
                    exitCode = previous == null
                        ? null
                        : previous.exitCodeIfExited();
                    previousPid = previous == null
                        ? -1L
                        : previous.pid();
                    previousPort = webUiPort;

                    webUiGeneration++;
                    managed = null;
                    webUiUri = null;
                    sessionUri = null;
                    webUiPort = -1;
                    System.clearProperty("geoworld.webui.url");
                    pendingBindPayload = null;
                    gameSessionBound = false;
                    gameSyncInFlight.set(false);
                }

                if (previous != null) {
                    previous.close();
                }
                if (!expectedRunning) {
                    return;
                }

                LOGGER.warn(
                    "GeoWorld WebUI automatic recovery: reason={}, oldPid={}, exitCode={}",
                    reason,
                    previousPid,
                    exitCode == null ? "running/unavailable" : exitCode
                );

                start(previousPort);
                if (!isRunning()) {
                    LOGGER.warn(
                        "GeoWorld WebUI automatic recovery did not restart the service; retrying after backoff"
                    );
                    return;
                }

                consecutiveHealthFailures = 0;
                MinecraftServer server = boundServer;
                if (server == null) {
                    return;
                }
                try {
                    server.execute(() -> {
                        if (expectedRunning && boundServer == server) {
                            bind(server);
                        }
                    });
                } catch (Exception e) {
                    LOGGER.warn(
                        "GeoWorld WebUI restarted but failed to queue world rebind",
                        e
                    );
                }
            } finally {
                recoveryInFlight.set(false);
            }
        });
    }

    private static void pollGameControl(
        MinecraftServer server,
        long generation
    ) {
        URI base = webUiUri;
        if (
            base == null ||
            boundServer != server ||
            generation != webUiGeneration
        ) {
            return;
        }
        try {
            postChunkyStatusIfDue(generation);
            HttpRequest request = HttpRequest.newBuilder(
                    base.resolve("api/game/control")
                )
                .timeout(Duration.ofSeconds(2))
                .header(TOKEN_HEADER, TOKEN)
                .GET()
                .build();
            HttpResponse<String> response = HTTP.send(
                request,
                HttpResponse.BodyHandlers.ofString()
            );
            if (
                generation != webUiGeneration ||
                boundServer != server
            ) {
                return;
            }
            if (response.statusCode() == 204) {
                return;
            }
            if (response.statusCode() != 200) {
                LOGGER.debug(
                    "GeoWorld WebUI control poll returned HTTP {}",
                    response.statusCode()
                );
                return;
            }
            String[] parts = response.body().trim().split("\\t");
            if (parts.length < 2) {
                LOGGER.warn("Ignoring malformed GeoWorld game-control payload");
                return;
            }
            switch (parts[0]) {
                case "teleport" -> handleTeleportControl(
                    server,
                    generation,
                    parts
                );
                case "chunky" -> handleChunkyControl(
                    server,
                    generation,
                    parts
                );
                default -> LOGGER.warn(
                    "Ignoring unknown GeoWorld game-control type: {}",
                    parts[0]
                );
            }
        } catch (Exception e) {
            LOGGER.debug("GeoWorld WebUI game-control poll failed", e);
        }
    }

    private static void handleTeleportControl(
        MinecraftServer server,
        long generation,
        String[] parts
    ) {
        if (parts.length != 5) {
            LOGGER.warn("Ignoring malformed GeoWorld teleport control payload");
            return;
        }
        long commandId = Long.parseLong(parts[1]);
        double x = Double.parseDouble(parts[2]);
        double y = Double.parseDouble(parts[3]);
        double z = Double.parseDouble(parts[4]);
        if (
            !Double.isFinite(x) ||
            !Double.isFinite(y) ||
            !Double.isFinite(z) ||
            y < -64.0 ||
            y > 320.0
        ) {
            LOGGER.warn(
                "Ignoring invalid GeoWorld teleport target: {}, {}, {}",
                x,
                y,
                z
            );
            return;
        }
        server.execute(() -> {
            if (
                boundServer != server ||
                generation != webUiGeneration
            ) {
                return;
            }
            var player = singleplayerOwner(server);
            if (player == null) {
                String message =
                    "local singleplayer owner is unavailable";
                LOGGER.warn(
                    "Ignoring GeoWorld teleport #{} because {}",
                    commandId,
                    message
                );
                postSessionAsync(
                    teleportResultPayload(commandId, false, message),
                    4
                );
                return;
            }
            if (!Level.OVERWORLD.equals(player.level().dimension())) {
                postSessionAsync(
                    teleportResultPayload(
                        commandId, false, "当前不在主世界，已暂停地图传送"
                    ),
                    4
                );
                return;
            }
            try {
                player.teleportTo(x, y, z);
                LOGGER.info(
                    "GeoWorld WebUI teleport #{} executed: {}, {}, {}",
                    commandId,
                    x,
                    y,
                    z
                );
                postSessionAsync(
                    teleportResultPayload(commandId, true, "executed"),
                    4
                );
            } catch (Exception e) {
                LOGGER.warn(
                    "GeoWorld WebUI teleport #{} failed",
                    commandId,
                    e
                );
                postSessionAsync(
                    teleportResultPayload(
                        commandId,
                        false,
                        e.getMessage() == null
                            ? e.getClass().getSimpleName()
                            : e.getMessage()
                    ),
                    4
                );
            }
        });
    }

    private static void handleChunkyControl(
        MinecraftServer server,
        long generation,
        String[] parts
    ) {
        if (parts.length != 7) {
            LOGGER.warn("Ignoring malformed GeoWorld Chunky control payload");
            return;
        }
        long commandId = Long.parseLong(parts[1]);
        String action = parts[2];
        int x1 = Integer.parseInt(parts[3]);
        int z1 = Integer.parseInt(parts[4]);
        int x2 = Integer.parseInt(parts[5]);
        int z2 = Integer.parseInt(parts[6]);
        server.execute(() -> {
            if (
                boundServer != server ||
                generation != webUiGeneration
            ) {
                return;
            }
            try {
                executeChunkyAction(server, action, x1, z1, x2, z2);
                LOGGER.info(
                    "GeoWorld WebUI Chunky #{} executed: {} [{},{} -> {},{}]",
                    commandId,
                    action,
                    x1,
                    z1,
                    x2,
                    z2
                );
                postSessionAsync(
                    chunkyResultPayload(commandId, true, "executed"),
                    4
                );
            } catch (Exception e) {
                String message = e.getMessage() == null
                    ? e.getClass().getSimpleName()
                    : e.getMessage();
                LOGGER.warn(
                    "GeoWorld WebUI Chunky #{} failed: {}",
                    commandId,
                    message
                );
                postSessionAsync(
                    chunkyResultPayload(commandId, false, message),
                    4
                );
            }
        });
    }

    private static void executeChunkyAction(
        MinecraftServer server,
        String action,
        int x1,
        int z1,
        int x2,
        int z2
    ) {
        if (action.equals("start") || action.equals("continue")) {
            var owner = singleplayerOwner(server);
            if (owner == null || !Level.OVERWORLD.equals(owner.level().dimension())) {
                throw new IllegalStateException(
                    "当前不在主世界，已暂停新的区块预生成操作"
                );
            }
        }
        if (!chunkyAvailable()) {
            throw new IllegalStateException(
                "Chunky 未安装或当前版本不可用"
            );
        }
        switch (action) {
            case "start" -> {
                if (chunkyTaskIsActive()) {
                    throw new IllegalStateException(
                        "Chunky has an active task; continue or cancel it first"
                    );
                }
                runServerCommand(server, "chunky world minecraft:overworld");
                runServerCommand(
                    server,
                    "chunky corners " + x1 + " " + z1 + " " + x2 + " " + z2
                );
                runServerCommand(server, "chunky start");
            }
            case "pause" ->
                runServerCommand(server, "chunky pause minecraft:overworld");
            case "continue" ->
                runServerCommand(server, "chunky continue minecraft:overworld");
            case "cancel" -> {
                runServerCommand(server, "chunky cancel minecraft:overworld");
                runServerCommand(server, "chunky confirm");
            }
            default -> throw new IllegalArgumentException(
                "unsupported Chunky action: " + action
            );
        }
    }

    private static void runServerCommand(
        MinecraftServer server,
        String command
    ) {
        server.getCommands().performPrefixedCommand(
            server.createCommandSourceStack(),
            command
        );
    }

    private static boolean chunkyAvailable() {
        try {
            Class.forName(
                "org.popcraft.chunky.Chunky",
                false,
                GeoWorldGameWebUi.class.getClassLoader()
            );
            return true;
        } catch (ClassNotFoundException e) {
            return false;
        }
    }

    private static Path chunkyTaskPath() {
        return Path.of(
            "config",
            "chunky",
            "tasks",
            "minecraft",
            "overworld.properties"
        );
    }

    private static boolean chunkyTaskIsActive() {
        ChunkyRuntimeStatus status = chunkyRuntimeStatus();
        return status.taskExists() &&
            !status.cancelled() &&
            !status.complete();
    }

    private static ChunkyRuntimeStatus chunkyRuntimeStatus() {
        boolean available = chunkyAvailable();
        if (!available) {
            return new ChunkyRuntimeStatus(
                false, false, false, false,
                0L, 0L, 0.0, 0.0,
                0, 0, 0, 0, 0, 0,
                0L, false
            );
        }

        try {
            Object chunky = chunkyInstance();
            if (chunky != null) {
                Object rawTasks = chunky
                    .getClass()
                    .getMethod("getGenerationTasks")
                    .invoke(chunky);
                if (rawTasks instanceof java.util.Map<?, ?> tasks) {
                    Object task = tasks.get("minecraft:overworld");
                    if (task != null) {
                        Object progress = task
                            .getClass()
                            .getMethod("getProgress")
                            .invoke(task);
                        long chunks = ((Number) progress
                            .getClass()
                            .getMethod("getChunkCount")
                            .invoke(progress))
                            .longValue();
                        boolean complete = (Boolean) progress
                            .getClass()
                            .getMethod("isComplete")
                            .invoke(progress);
                        double percent = ((Number) progress
                            .getClass()
                            .getMethod("getPercentComplete")
                            .invoke(progress))
                            .doubleValue();
                        double rate = ((Number) progress
                            .getClass()
                            .getMethod("getRate")
                            .invoke(progress))
                            .doubleValue();
                        int currentX = ((Number) progress
                            .getClass()
                            .getMethod("getChunkX")
                            .invoke(progress))
                            .intValue();
                        int currentZ = ((Number) progress
                            .getClass()
                            .getMethod("getChunkZ")
                            .invoke(progress))
                            .intValue();
                        long elapsed = ((Number) task
                            .getClass()
                            .getMethod("getTotalTime")
                            .invoke(task))
                            .longValue();
                        boolean cancelled = (Boolean) task
                            .getClass()
                            .getMethod("isCancelled")
                            .invoke(task);

                        Object selection = task
                            .getClass()
                            .getMethod("getSelection")
                            .invoke(task);
                        int diameterX = ((Number) selection
                            .getClass()
                            .getMethod("diameterChunksX")
                            .invoke(selection))
                            .intValue();
                        int diameterZ = ((Number) selection
                            .getClass()
                            .getMethod("diameterChunksZ")
                            .invoke(selection))
                            .intValue();
                        double centerX = ((Number) selection
                            .getClass()
                            .getMethod("centerX")
                            .invoke(selection))
                            .doubleValue();
                        double centerZ = ((Number) selection
                            .getClass()
                            .getMethod("centerZ")
                            .invoke(selection))
                            .doubleValue();
                        double radiusX = ((Number) selection
                            .getClass()
                            .getMethod("radiusX")
                            .invoke(selection))
                            .doubleValue();
                        double radiusZ = ((Number) selection
                            .getClass()
                            .getMethod("radiusZ")
                            .invoke(selection))
                            .doubleValue();
                        int x1 = (int) Math.floor(centerX - radiusX);
                        int z1 = (int) Math.floor(centerZ - radiusZ);
                        int x2 = (int) Math.floor(centerX + radiusX);
                        int z2 = (int) Math.floor(centerZ + radiusZ);
                        long total = Math.max(
                            0L,
                            (long) diameterX * (long) diameterZ
                        );

                        return new ChunkyRuntimeStatus(
                            true,
                            true,
                            true,
                            complete,
                            Math.max(0L, chunks),
                            total,
                            Math.max(0.0, Math.min(100.0, percent)),
                            Math.max(0.0, rate),
                            currentX,
                            currentZ,
                            x1,
                            z1,
                            x2,
                            z2,
                            Math.max(0L, elapsed),
                            cancelled
                        );
                    }
                }
            }
        } catch (Exception e) {
            LOGGER.debug("Unable to read live Chunky task state", e);
        }

        Path taskPath = chunkyTaskPath();
        if (!Files.isRegularFile(taskPath)) {
            return new ChunkyRuntimeStatus(
                true, false, false, false,
                0L, 0L, 0.0, 0.0,
                0, 0, 0, 0, 0, 0,
                0L, false
            );
        }
        Properties properties = new Properties();
        try (var input = Files.newInputStream(taskPath)) {
            properties.load(input);
            long processed = parseLongProperty(properties, "chunks");
            long elapsed = parseLongProperty(properties, "time");
            boolean cancelled = Boolean.parseBoolean(
                properties.getProperty("cancelled", "false")
            );
            double centerX = parseDoubleProperty(properties, "center-x");
            double centerZ = parseDoubleProperty(properties, "center-z");
            double radiusX = parseDoubleProperty(properties, "radius");
            double radiusZ = parseDoubleProperty(properties, "radius-z");
            int x1 = (int) Math.floor(centerX - radiusX);
            int z1 = (int) Math.floor(centerZ - radiusZ);
            int x2 = (int) Math.floor(centerX + radiusX);
            int z2 = (int) Math.floor(centerZ + radiusZ);
            long chunksX = Math.max(0L, (long) (x2 - x1 + 1) / 16L);
            long chunksZ = Math.max(0L, (long) (z2 - z1 + 1) / 16L);
            long total = chunksX * chunksZ;
            double percent = total > 0L
                ? Math.max(0.0, Math.min(100.0, processed * 100.0 / total))
                : 0.0;
            return new ChunkyRuntimeStatus(
                true,
                true,
                false,
                total > 0L && processed >= total,
                processed,
                total,
                percent,
                0.0,
                0,
                0,
                x1,
                z1,
                x2,
                z2,
                elapsed,
                cancelled
            );
        } catch (Exception e) {
            LOGGER.warn(
                "Unable to inspect saved Chunky task state; treating it as active",
                e
            );
            return new ChunkyRuntimeStatus(
                true, true, false, false,
                0L, 0L, 0.0, 0.0,
                0, 0, 0, 0, 0, 0,
                0L, false
            );
        }
    }

    private static Object chunkyInstance() {
        return platform().chunkyInstance();
    }

    private static void postChunkyStatusIfDue(long generation) {
        long now = System.nanoTime();
        if (now - lastChunkyStatusNanos < TimeUnit.SECONDS.toNanos(1)) {
            return;
        }
        lastChunkyStatusNanos = now;

        ChunkyRuntimeStatus status = chunkyRuntimeStatus();
        postSession(
            chunkyStatusPayload(status),
            1,
            generation
        );
    }

    private record ChunkyRuntimeStatus(
        boolean available,
        boolean taskExists,
        boolean live,
        boolean complete,
        long processedChunks,
        long totalChunks,
        double percent,
        double rate,
        int currentChunkX,
        int currentChunkZ,
        int x1,
        int z1,
        int x2,
        int z2,
        long elapsedMillis,
        boolean cancelled
    ) {}

    private static long parseLongProperty(
        Properties properties,
        String key
    ) {
        try {
            return Math.max(
                0L,
                Long.parseLong(properties.getProperty(key, "0").trim())
            );
        } catch (RuntimeException ignored) {
            return 0L;
        }
    }

    private static double parseDoubleProperty(
        Properties properties,
        String key
    ) {
        try {
            return Double.parseDouble(
                properties.getProperty(key, "0").trim()
            );
        } catch (RuntimeException ignored) {
            return 0.0;
        }
    }

    private static net.minecraft.server.level.ServerPlayer singleplayerOwner(
        MinecraftServer server
    ) {
        for (var player : server.getPlayerList().getPlayers()) {
            if (platform().isSingleplayerOwner(server, player)) {
                return player;
            }
        }
        return null;
    }

    private static String playerPayload(
        double x,
        double y,
        double z,
        double yaw,
        double pitch,
        String dimension
    ) {
        return "{" +
            field("action", "player") + "," +
            field("client", platform().gameClient()) + "," +
            doubleField("playerX", x) + "," +
            doubleField("playerY", y) + "," +
            doubleField("playerZ", z) + "," +
            doubleField("playerYaw", yaw) + "," +
            doubleField("playerPitch", pitch) + "," +
            field("dimension", dimension) +
            "}";
    }

    private static String teleportResultPayload(
        long commandId,
        boolean success,
        String message
    ) {
        return "{" +
            field("action", "teleport_result") + "," +
            numberField("commandId", commandId) + "," +
            booleanField("success", success) + "," +
            field("message", message == null ? "" : message) +
            "}";
    }

    private static String chunkyResultPayload(
        long commandId,
        boolean success,
        String message
    ) {
        return "{" +
            field("action", "chunky_result") + "," +
            numberField("commandId", commandId) + "," +
            booleanField("success", success) + "," +
            field("message", message == null ? "" : message) +
            "}";
    }

    private static String chunkyStatusPayload(
        ChunkyRuntimeStatus status
    ) {
        return "{" +
            field("action", "chunky_status") + "," +
            booleanField("chunkyAvailable", status.available()) + "," +
            booleanField("chunkyTaskExists", status.taskExists()) + "," +
            booleanField("chunkyTaskLive", status.live()) + "," +
            booleanField("chunkyComplete", status.complete()) + "," +
            numberField("chunkyProcessedChunks", status.processedChunks()) + "," +
            numberField("chunkyTotalChunks", status.totalChunks()) + "," +
            doubleField("chunkyPercent", status.percent()) + "," +
            doubleField("chunkyRate", status.rate()) + "," +
            numberField("chunkyCurrentChunkX", status.currentChunkX()) + "," +
            numberField("chunkyCurrentChunkZ", status.currentChunkZ()) + "," +
            numberField("chunkyX1", status.x1()) + "," +
            numberField("chunkyZ1", status.z1()) + "," +
            numberField("chunkyX2", status.x2()) + "," +
            numberField("chunkyZ2", status.z2()) + "," +
            numberField("chunkyElapsedMillis", status.elapsedMillis()) + "," +
            booleanField("chunkyCancelled", status.cancelled()) +
            "}";
    }

    private static String compatibilityError(
        GeoWorldV26WorldService.ActiveWorld active
    ) {
        GeoWorldBinding.Binding binding = active.binding();
        GeoWorldV1.Manifest manifest = active.model().manifest();
        String expectedMinecraft =
            platform().minecraftVersion();

        if (binding.bindingVersion() < GeoWorldBinding.VERSION) {
            return "当前 GeoWorld 存档缺少完整的版本锁定信息，拒绝自动绑定 WebUI";
        }
        if (!expectedMinecraft.equals(binding.minecraftVersion())) {
            return "Minecraft 版本不一致：存档 " +
                binding.minecraftVersion() +
                "，当前 MOD " +
                expectedMinecraft;
        }
        if (binding.seed() != manifest.seed()) {
            return "Seed 不一致，拒绝自动绑定 WebUI";
        }
        if (
            !binding.generatorVersion().equals(manifest.generatorVersion()) ||
            !binding.worldProfileVersion().equals(manifest.worldProfileVersion()) ||
            !binding.generationRulesVersion().equals(manifest.generationRulesVersion()) ||
            !binding.worldModelVersion().equals(manifest.worldModelVersion()) ||
            !binding.preset().equals(manifest.preset()) ||
            !binding.parameterSnapshotSha256().equals(
                manifest.parameterSnapshotSha256()
            )
        ) {
            return "GeoWorld 世界身份与当前游戏进程实际加载的模型不一致，拒绝自动绑定 WebUI";
        }
        return null;
    }

    private static String boundPayload(
        GeoWorldV26WorldService.ActiveWorld active
    ) {
        GeoWorldBinding.Binding binding = active.binding();
        String worldName = active.worldDirectory().getFileName() == null
            ? active.worldDirectory().toString()
            : active.worldDirectory().getFileName().toString();

        return "{" +
            field("action", "bind") + "," +
            field("state", "bound") + "," +
            field("client", platform().gameClient()) + "," +
            field("worldName", worldName) + "," +
            field("worldDirectory", active.worldDirectory().toString()) + "," +
            field("modelPath", binding.modelPath().toString()) + "," +
            numberField("bindingVersion", binding.bindingVersion()) + "," +
            field("minecraftVersion", binding.minecraftVersion()) + "," +
            numberField("modelSchemaVersion", binding.modelSchemaVersion()) + "," +
            field("generatorVersion", binding.generatorVersion()) + "," +
            numberField("seed", binding.seed()) + "," +
            field("worldProfileVersion", binding.worldProfileVersion()) + "," +
            field("generationRulesVersion", binding.generationRulesVersion()) + "," +
            field("worldModelVersion", binding.worldModelVersion()) + "," +
            field("preset", binding.preset()) + "," +
            field(
                "parameterSnapshotSha256",
                binding.parameterSnapshotSha256()
            ) + "," +
            stringArrayField("oreFamilies", GeoWorldOreCatalog.families()) + "," +
            stringMapField("oreIcons", oreIconDataUris()) + "," +
            booleanField("dynamicWorldgen", active.dynamicWorldgen()) +
            "}";
    }

    private static String unboundPayload(String state, String message) {
        return "{" +
            field("action", "unbind") + "," +
            field("state", state) + "," +
            field("client", platform().gameClient()) + "," +
            field("message", message) +
            "}";
    }

    private static void postSessionAsync(String payload, int attempts) {
        long generation = webUiGeneration;
        CONTROL.execute(() -> postSession(payload, attempts, generation));
    }

    private static boolean postSession(
        String payload,
        int attempts,
        long generation
    ) {
        for (int attempt = 1; attempt <= attempts; attempt++) {
            GeoWorldToolLauncher.ManagedProcess current;
            synchronized (GeoWorldGameWebUi.class) {
                if (generation != webUiGeneration) {
                    return false;
                }
                current = managed;
            }
            if (current == null || !current.isAlive()) {
                return false;
            }
            try {
                URI currentSessionUri = sessionUri;
                if (currentSessionUri == null) {
                    return false;
                }
                HttpRequest request = HttpRequest.newBuilder(currentSessionUri)
                    .timeout(Duration.ofSeconds(3))
                    .header("Content-Type", "application/json")
                    .header(TOKEN_HEADER, TOKEN)
                    .POST(HttpRequest.BodyPublishers.ofString(payload))
                    .build();
                HttpResponse<String> response = HTTP.send(
                    request,
                    HttpResponse.BodyHandlers.ofString()
                );
                if (
                    response.statusCode() >= 200 &&
                    response.statusCode() < 300
                ) {
                    return true;
                }
                if (response.statusCode() == 409 && attempts == 1) {
                    LOGGER.debug(
                        "GeoWorld WebUI session is temporarily unavailable: {}",
                        response.body()
                    );
                } else {
                    LOGGER.warn(
                        "GeoWorld WebUI session update returned HTTP {}: {}",
                        response.statusCode(),
                        response.body()
                    );
                }
            } catch (Exception e) {
                if (attempt == attempts) {
                    if (attempts == 1) {
                        LOGGER.debug(
                            "GeoWorld WebUI session update will be retried by the sync loop",
                            e
                        );
                    } else {
                        LOGGER.warn(
                            "Unable to update GeoWorld WebUI game session after {} attempts",
                            attempts,
                            e
                        );
                    }
                    return false;
                }
            }

            try {
                Thread.sleep(250L);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                return false;
            }
        }
        return false;
    }

    private static Map<String, String> oreIconDataUris() {
        Map<String, String> out = new LinkedHashMap<>();
        try {
            ResourceManager resources =
                Minecraft.getInstance().getResourceManager();
            for (
                Map.Entry<String, String> entry :
                GeoWorldOreCatalog.previewBlockIds().entrySet()
            ) {
                String icon = blockTextureDataUri(
                    resources,
                    entry.getValue()
                );
                if (icon != null && !icon.isBlank()) {
                    out.put(entry.getKey(), icon);
                }
            }
        } catch (Exception e) {
            LOGGER.debug(
                "Unable to collect GeoWorld ore icons from Minecraft resources",
                e
            );
        }
        return Map.copyOf(out);
    }

    private static String blockTextureDataUri(
        ResourceManager resources,
        String blockId
    ) {
        if (
            resources == null ||
            blockId == null ||
            blockId.isBlank()
        ) {
            return null;
        }

        ResourceLocation block = parseResourceId(
            blockId,
            "minecraft"
        );
        if (block == null) {
            return null;
        }

        ResourceLocation texture = null;
        ResourceLocation blockState = ResourceLocation.fromNamespaceAndPath(
            block.getNamespace(),
            "blockstates/" + block.getPath() + ".json"
        );
        JsonObject state = readJsonObject(resources, blockState);
        if (state != null) {
            String modelRef = firstModelReference(state);
            if (modelRef != null) {
                texture = resolveModelTexture(
                    resources,
                    modelRef,
                    block.getNamespace(),
                    Map.of(),
                    0
                );
            }
        }

        if (texture == null) {
            texture = resolveModelTexture(
                resources,
                block.getNamespace() + ":block/" + block.getPath(),
                block.getNamespace(),
                Map.of(),
                0
            );
        }

        if (texture == null) {
            String[] candidates = new String[]{
                "textures/block/" + block.getPath() + ".png",
                "textures/block/ores/" + block.getPath() + ".png",
                "textures/block/ore/" + block.getPath() + ".png",
                "textures/blocks/" + block.getPath() + ".png"
            };
            for (String path : candidates) {
                ResourceLocation candidate =
                    ResourceLocation.fromNamespaceAndPath(
                        block.getNamespace(),
                        path
                    );
                if (resources.getResource(candidate).isPresent()) {
                    texture = candidate;
                    break;
                }
            }
        }

        if (texture == null) {
            return null;
        }
        try {
            Optional<net.minecraft.server.packs.resources.Resource> resource =
                resources.getResource(texture);
            if (resource.isEmpty()) {
                return null;
            }
            try (InputStream input = resource.get().open()) {
                byte[] bytes = input.readAllBytes();
                if (bytes.length == 0 || bytes.length > 512 * 1024) {
                    return null;
                }
                return "data:image/png;base64," +
                    Base64.getEncoder().encodeToString(bytes);
            }
        } catch (Exception e) {
            LOGGER.debug(
                "Unable to read GeoWorld ore texture {} for {}",
                texture,
                blockId,
                e
            );
            return null;
        }
    }

    private static ResourceLocation resolveModelTexture(
        ResourceManager resources,
        String modelRef,
        String defaultNamespace,
        Map<String, String> inheritedTextures,
        int depth
    ) {
        if (depth > 8 || modelRef == null || modelRef.isBlank()) {
            return null;
        }
        ResourceLocation model = parseResourceId(
            modelRef,
            defaultNamespace
        );
        if (model == null) {
            return null;
        }
        ResourceLocation modelJson =
            ResourceLocation.fromNamespaceAndPath(
                model.getNamespace(),
                "models/" + model.getPath() + ".json"
            );
        JsonObject json = readJsonObject(resources, modelJson);
        if (json == null) {
            return null;
        }

        Map<String, String> textures =
            new LinkedHashMap<>(inheritedTextures);
        if (
            json.has("textures") &&
            json.get("textures").isJsonObject()
        ) {
            for (
                Map.Entry<String, JsonElement> entry :
                json.getAsJsonObject("textures").entrySet()
            ) {
                if (entry.getValue().isJsonPrimitive()) {
                    textures.put(
                        entry.getKey(),
                        entry.getValue().getAsString()
                    );
                }
            }
        }

        for (
            String key :
            new String[]{"all", "particle", "texture", "side", "top"}
        ) {
            ResourceLocation resolved = resolveTextureReference(
                textures.get(key),
                textures,
                model.getNamespace()
            );
            if (
                resolved != null &&
                resources.getResource(resolved).isPresent()
            ) {
                return resolved;
            }
        }
        for (String value : textures.values()) {
            ResourceLocation resolved = resolveTextureReference(
                value,
                textures,
                model.getNamespace()
            );
            if (
                resolved != null &&
                resources.getResource(resolved).isPresent()
            ) {
                return resolved;
            }
        }

        if (
            json.has("parent") &&
            json.get("parent").isJsonPrimitive()
        ) {
            return resolveModelTexture(
                resources,
                json.get("parent").getAsString(),
                model.getNamespace(),
                textures,
                depth + 1
            );
        }
        return null;
    }

    private static ResourceLocation resolveTextureReference(
        String value,
        Map<String, String> textures,
        String defaultNamespace
    ) {
        if (value == null || value.isBlank()) {
            return null;
        }
        String resolved = value;
        for (int i = 0; i < 8 && resolved.startsWith("#"); i++) {
            resolved = textures.get(resolved.substring(1));
            if (resolved == null || resolved.isBlank()) {
                return null;
            }
        }
        if (resolved.startsWith("#")) {
            return null;
        }
        ResourceLocation texture = parseResourceId(
            resolved,
            defaultNamespace
        );
        if (texture == null) {
            return null;
        }
        String path = texture.getPath();
        if (!path.startsWith("textures/")) {
            path = "textures/" + path;
        }
        if (!path.endsWith(".png")) {
            path += ".png";
        }
        return ResourceLocation.fromNamespaceAndPath(
            texture.getNamespace(),
            path
        );
    }

    private static String firstModelReference(JsonElement element) {
        if (element == null || element.isJsonNull()) {
            return null;
        }
        if (element.isJsonObject()) {
            JsonObject object = element.getAsJsonObject();
            if (
                object.has("model") &&
                object.get("model").isJsonPrimitive()
            ) {
                return object.get("model").getAsString();
            }
            for (
                Map.Entry<String, JsonElement> entry :
                object.entrySet()
            ) {
                String model = firstModelReference(entry.getValue());
                if (model != null) {
                    return model;
                }
            }
        } else if (element.isJsonArray()) {
            for (JsonElement child : element.getAsJsonArray()) {
                String model = firstModelReference(child);
                if (model != null) {
                    return model;
                }
            }
        }
        return null;
    }

    private static JsonObject readJsonObject(
        ResourceManager resources,
        ResourceLocation location
    ) {
        try {
            Optional<net.minecraft.server.packs.resources.Resource> resource =
                resources.getResource(location);
            if (resource.isEmpty()) {
                return null;
            }
            try (
                java.io.BufferedReader reader =
                    resource.get().openAsReader()
            ) {
                JsonElement element = JsonParser.parseReader(reader);
                return element.isJsonObject()
                    ? element.getAsJsonObject()
                    : null;
            }
        } catch (Exception ignored) {
            return null;
        }
    }

    private static ResourceLocation parseResourceId(
        String value,
        String defaultNamespace
    ) {
        if (value == null || value.isBlank()) {
            return null;
        }
        try {
            int separator = value.indexOf(':');
            if (separator >= 0) {
                return ResourceLocation.parse(value);
            }
            return ResourceLocation.fromNamespaceAndPath(
                defaultNamespace == null || defaultNamespace.isBlank()
                    ? "minecraft"
                    : defaultNamespace,
                value
            );
        } catch (Exception ignored) {
            return null;
        }
    }

    private static String stringMapField(
        String name,
        Map<String, String> values
    ) {
        StringBuilder out = new StringBuilder();
        out.append(quote(name)).append(":{");
        boolean first = true;
        for (
            Map.Entry<String, String> entry :
            values.entrySet()
        ) {
            if (!first) out.append(',');
            first = false;
            out.append(quote(entry.getKey()))
                .append(':')
                .append(quote(entry.getValue()));
        }
        return out.append('}').toString();
    }

    private static String field(String name, String value) {
        return quote(name) + ":" + quote(value == null ? "" : value);
    }

    private static String numberField(String name, long value) {
        return quote(name) + ":" + value;
    }

    private static String doubleField(String name, double value) {
        return quote(name) + ":" + Double.toString(value);
    }

    private static String booleanField(String name, boolean value) {
        return quote(name) + ":" + value;
    }

    private static String stringArrayField(
        String name,
        Iterable<String> values
    ) {
        StringBuilder out = new StringBuilder();
        out.append(quote(name)).append(":[");
        boolean first = true;
        for (String value : values) {
            if (!first) out.append(',');
            first = false;
            out.append(quote(value == null ? "" : value));
        }
        return out.append(']').toString();
    }

    private static String quote(String value) {
        StringBuilder out = new StringBuilder(value.length() + 8);
        out.append('"');
        for (int i = 0; i < value.length(); i++) {
            char c = value.charAt(i);
            switch (c) {
                case '\\' -> {
                    out.append('\\');
                    out.append('\\');
                }
                case '"' -> {
                    out.append('\\');
                    out.append('"');
                }
                case '\n' -> {
                    out.append('\\');
                    out.append('n');
                }
                case '\r' -> {
                    out.append('\\');
                    out.append('r');
                }
                case '\t' -> {
                    out.append('\\');
                    out.append('t');
                }
                default -> {
                    if (c < 0x20) {
                        out.append(String.format("\\u%04x", (int) c));
                    } else {
                        out.append(c);
                    }
                }
            }
        }
        return out.append('"').toString();
    }
}
