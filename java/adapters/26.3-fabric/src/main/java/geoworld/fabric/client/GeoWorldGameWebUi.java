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

import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.time.Duration;
import java.util.UUID;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

final class GeoWorldGameWebUi {
    private static final Logger LOGGER =
        LoggerFactory.getLogger("GeoWorld/WebUI");
    private static final String LISTEN_ADDRESS = "0.0.0.0:8788";
    private static final URI SESSION_URI =
        URI.create("http://127.0.0.1:8788/api/game/session");
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

    private static GeoWorldToolLauncher.ManagedProcess managed;
    private static boolean shutdownHookRegistered;

    private GeoWorldGameWebUi() {}

    static synchronized void start() {
        if (managed != null && managed.isAlive()) {
            return;
        }
        try {
            managed = GeoWorldToolLauncher.startManaged(
                "-addr=" + LISTEN_ADDRESS,
                "-open=false",
                "-game-token=" + TOKEN,
                "-game-client=" + GAME_CLIENT
            );
            LOGGER.info(
                "GeoWorld WebUI started at http://127.0.0.1:8788 (browser auto-open disabled)"
            );
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
            managed = null;
            LOGGER.warn(
                "Unable to start GeoWorld WebUI; game startup will continue normally",
                e
            );
        }
    }

    static void bind(MinecraftServer server) {
        start();
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

        postSessionAsync(boundPayload(world), 12);
    }

    static void unbind(String message) {
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
        GeoWorldToolLauncher.ManagedProcess current = managed;
        managed = null;
        if (current != null) {
            current.close();
            LOGGER.info("GeoWorld WebUI stopped");
        }
        CONTROL.shutdownNow();
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
        CONTROL.execute(() -> postSession(payload, attempts));
    }

    private static void postSession(String payload, int attempts) {
        for (int attempt = 1; attempt <= attempts; attempt++) {
            GeoWorldToolLauncher.ManagedProcess current;
            synchronized (GeoWorldGameWebUi.class) {
                current = managed;
            }
            if (current == null || !current.isAlive()) {
                return;
            }
            try {
                HttpRequest request = HttpRequest.newBuilder(SESSION_URI)
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
