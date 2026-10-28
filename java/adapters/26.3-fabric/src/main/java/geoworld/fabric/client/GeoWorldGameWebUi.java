package geoworld.fabric.client;

import geoworld.minecraft.common.tool.GeoWorldToolLauncher;
import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26_3common.runtime.GeoWorldWorldService;
import geoworld.minecraft.v26common.runtime.GeoWorldV26WorldService;
import geoworld.runtime.GeoWorldBinding;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.server.MinecraftServer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.InetSocketAddress;
import java.net.ServerSocket;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.UUID;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.ScheduledExecutorService;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

final class GeoWorldGameWebUi {
    private static final Logger LOGGER =
        LoggerFactory.getLogger("GeoWorld/WebUI");
    private static final String LISTEN_HOST = "0.0.0.0";
    private static final int PREFERRED_PORT = 8788;
    private static final String TOKEN_HEADER = "X-GeoWorld-Game-Token";
    private static final String GAME_CLIENT = "Minecraft 26.3 Fabric";
    private static final String TOKEN = UUID.randomUUID().toString();
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
    private static volatile long webUiGeneration;
    private static final AtomicBoolean gameSyncInFlight = new AtomicBoolean();
    private static boolean syncLoopStarted;
    private static boolean shutdownHookRegistered;

    private GeoWorldGameWebUi() {}

    static synchronized void start() {
        if (managed != null && managed.isAlive()) {
            return;
        }
        webUiGeneration++;
        GeoWorldToolLauncher.ManagedProcess stale = managed;
        managed = null;
        webUiUri = null;
        sessionUri = null;
        webUiPort = -1;
        if (stale != null) {
            stale.close();
        }
        try {
            WebUiLaunch launch = launchWebUiWithRetry();
            managed = launch.process();
            webUiPort = launch.port();
            webUiUri = launch.webUiUri();
            sessionUri = launch.sessionUri();
            LOGGER.info(
                "GeoWorld WebUI started at {} (browser auto-open disabled)",
                webUiUri
            );
            ensureGameSyncLoop();
            if (!shutdownHookRegistered) {
                Runtime.getRuntime().addShutdownHook(
                    new Thread(GeoWorldGameWebUi::stop, "GeoWorld-WebUI-Shutdown")
                );
                shutdownHookRegistered = true;
            }
            postSessionAsync(
                unboundPayload(
                    "menu",
                    "Minecraft 已启动，当前尚未进入 GeoWorld 世界"
                ),
                20
            );
        } catch (Exception e) {
            GeoWorldToolLauncher.ManagedProcess failed = managed;
            managed = null;
            webUiUri = null;
            sessionUri = null;
            webUiPort = -1;
            boundServer = null;
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

    static void bind(MinecraftServer server) {
        start();
        boundServer = null;
        var active = GeoWorldWorldService.active(server);
        if (active.isEmpty()) {
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
        postSessionAsync(boundPayload(world), 12);
    }

    static void unbind(String message) {
        boundServer = null;
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

    static synchronized void stop() {
        webUiGeneration++;
        GeoWorldToolLauncher.ManagedProcess current = managed;
        managed = null;
        webUiUri = null;
        sessionUri = null;
        webUiPort = -1;
        boundServer = null;
        gameSyncInFlight.set(false);
        if (current != null) {
            current.close();
            LOGGER.info("GeoWorld WebUI stopped");
        }
        // Executors are daemon singletons for the client JVM. Keep them alive
        // so a stopped WebUI can be started again in the same Minecraft process.
    }

    static synchronized boolean isRunning() {
        return managed != null && managed.isAlive() && webUiUri != null;
    }

    static URI currentWebUiUri() {
        return webUiUri;
    }

    static int currentWebUiPort() {
        return webUiPort;
    }

    private record WebUiLaunch(
        GeoWorldToolLauncher.ManagedProcess process,
        int port,
        URI webUiUri,
        URI sessionUri
    ) {}

    private static WebUiLaunch launchWebUiWithRetry() throws Exception {
        Exception lastError = null;
        for (int attempt = 1; attempt <= 3; attempt++) {
            int port = attempt == 1
                ? selectAvailablePort()
                : selectRandomAvailablePort();
            URI ui = URI.create("http://127.0.0.1:" + port + "/");
            GeoWorldToolLauncher.ManagedProcess process = null;
            try {
                process = GeoWorldToolLauncher.startManaged(
                    "-addr=" + LISTEN_HOST + ":" + port,
                    "-open=false",
                    "-game-token=" + TOKEN,
                    "-game-client=" + GAME_CLIENT
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
            1L,
            1L,
            TimeUnit.SECONDS
        );
        syncLoopStarted = true;
    }

    private static void syncGameState() {
        MinecraftServer server = boundServer;
        long generation = webUiGeneration;
        if (
            server == null ||
            !isRunning() ||
            !gameSyncInFlight.compareAndSet(false, true)
        ) {
            return;
        }
        try {
            server.execute(() -> {
                if (
                    boundServer != server ||
                    generation != webUiGeneration
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
                    player.getZ()
                );
                CONTROL.execute(() -> {
                    try {
                        postSession(payload, 1, generation);
                        pollGameControl(server, generation);
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
            if (parts.length != 4) {
                LOGGER.warn("Ignoring malformed GeoWorld teleport control payload");
                return;
            }
            long commandId = Long.parseLong(parts[0]);
            double x = Double.parseDouble(parts[1]);
            double y = Double.parseDouble(parts[2]);
            double z = Double.parseDouble(parts[3]);
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
                    LOGGER.warn(
                        "Ignoring GeoWorld teleport #{} because the local singleplayer owner is unavailable",
                        commandId
                    );
                    return;
                }
                player.teleportTo(x, y, z);
                LOGGER.info(
                    "GeoWorld WebUI teleport #{} executed: {}, {}, {}",
                    commandId,
                    x,
                    y,
                    z
                );
            });
        } catch (Exception e) {
            LOGGER.debug("GeoWorld WebUI game-control poll failed", e);
        }
    }

    private static net.minecraft.server.level.ServerPlayer singleplayerOwner(
        MinecraftServer server
    ) {
        for (var player : server.getPlayerList().getPlayers()) {
            if (server.isSingleplayerOwner(player.nameAndId())) {
                return player;
            }
        }
        return null;
    }

    private static String playerPayload(double x, double y, double z) {
        return "{" +
            field("action", "player") + "," +
            field("client", GAME_CLIENT) + "," +
            doubleField("playerX", x) + "," +
            doubleField("playerY", y) + "," +
            doubleField("playerZ", z) +
            "}";
    }

    private static String compatibilityError(
        GeoWorldV26WorldService.ActiveWorld active
    ) {
        GeoWorldBinding.Binding binding = active.binding();
        GeoWorldV1.Manifest manifest = active.model().manifest();
        String expectedMinecraft =
            Minecraft26_3Profile.INSTANCE.minecraftVersion();

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
            field("client", GAME_CLIENT) + "," +
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
            booleanField("dynamicWorldgen", active.dynamicWorldgen()) +
            "}";
    }

    private static String unboundPayload(String state, String message) {
        return "{" +
            field("action", "unbind") + "," +
            field("state", state) + "," +
            field("client", GAME_CLIENT) + "," +
            field("message", message) +
            "}";
    }

    private static void postSessionAsync(String payload, int attempts) {
        long generation = webUiGeneration;
        CONTROL.execute(() -> postSession(payload, attempts, generation));
    }

    private static void postSession(
        String payload,
        int attempts,
        long generation
    ) {
        for (int attempt = 1; attempt <= attempts; attempt++) {
            GeoWorldToolLauncher.ManagedProcess current;
            synchronized (GeoWorldGameWebUi.class) {
                if (generation != webUiGeneration) {
                    return;
                }
                current = managed;
            }
            if (current == null || !current.isAlive()) {
                return;
            }
            try {
                URI currentSessionUri = sessionUri;
                if (currentSessionUri == null) {
                    return;
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
                    return;
                }
                LOGGER.warn(
                    "GeoWorld WebUI session update returned HTTP {}: {}",
                    response.statusCode(),
                    response.body()
                );
            } catch (Exception e) {
                if (attempt == attempts) {
                    LOGGER.warn(
                        "Unable to update GeoWorld WebUI game session after {} attempts",
                        attempts,
                        e
                    );
                    return;
                }
            }

            try {
                Thread.sleep(250L);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                return;
            }
        }
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
