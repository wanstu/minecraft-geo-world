package geoworld.neoforge.engine;

import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.runtime.GeoWorldV1;

import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.time.Duration;
import java.util.HexFormat;
import java.util.List;
import java.util.Locale;
import java.util.Optional;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;

public final class GeoWorldEngineBridge {
    private static final Duration GENERATION_TIMEOUT = Duration.ofMinutes(5);
    private static final ExecutorService EXECUTOR = Executors.newSingleThreadExecutor(runnable -> {
        Thread thread = new Thread(runnable, "GeoWorld-Engine");
        thread.setDaemon(true);
        return thread;
    });
    private static volatile Path resolvedEngine;

    private GeoWorldEngineBridge() {}

    public static CompletableFuture<GeneratedModel> generateAsync(
        long seed,
        GeoWorldCreationConfig config
    ) {
        return CompletableFuture.supplyAsync(() -> {
            try {
                return generate(seed, config);
            } catch (IOException | InterruptedException e) {
                throw new RuntimeException(e);
            }
        }, EXECUTOR);
    }

    public static CompletableFuture<Optional<GeneratedModel>> loadCachedAsync(
        long seed,
        GeoWorldCreationConfig config
    ) {
        return CompletableFuture.supplyAsync(() -> {
            try {
                return loadCached(seed, config);
            } catch (IOException e) {
                throw new RuntimeException(e);
            }
        }, EXECUTOR);
    }

    public static Optional<GeneratedModel> loadCached(
        long seed,
        GeoWorldCreationConfig config
    ) throws IOException {
        List<String> errors = config.validationErrors();
        if (!errors.isEmpty()) {
            throw new IllegalArgumentException(errors.getFirst());
        }

        String cacheKey = cacheKey(seed, config);
        Path output = resolveCacheRoot().resolve(cacheKey + ".geoworld");
        if (!Files.isRegularFile(output)) {
            return Optional.empty();
        }

        long started = System.nanoTime();
        try {
            GeoWorldV1 model = GeoWorldV1.load(output);
            validateModel(seed, config, model);
            return Optional.of(new GeneratedModel(
                seed,
                config,
                cacheKey,
                output,
                model,
                elapsedMillis(started),
                true
            ));
        } catch (IOException | RuntimeException invalidCache) {
            Files.deleteIfExists(output);
            return Optional.empty();
        }
    }

    public static GeneratedModel generate(
        long seed,
        GeoWorldCreationConfig config
    ) throws IOException, InterruptedException {
        List<String> errors = config.validationErrors();
        if (!errors.isEmpty()) {
            throw new IllegalArgumentException(errors.getFirst());
        }

        Path engine = resolveEngine();
        String cacheKey = cacheKey(seed, config);
        Path cacheRoot = resolveCacheRoot();
        Files.createDirectories(cacheRoot);
        Path output = cacheRoot.resolve(cacheKey + ".geoworld");

        long started = System.nanoTime();
        Optional<GeneratedModel> cached = loadCached(seed, config);
        if (cached.isPresent()) {
            return cached.get();
        }

        Path engineLog = cacheRoot.resolve(cacheKey + ".engine.log");
        Files.deleteIfExists(engineLog);

        ProcessBuilder processBuilder = new ProcessBuilder(
            engine.toString(),
            "--seed", Long.toString(seed),
            "--width", Integer.toString(config.width()),
            "--height", Integer.toString(config.height()),
            "--cell-size", Integer.toString(config.cellSizeBlocks()),
            "--plates", Integer.toString(config.plateCount()),
            "--ocean", format(config.oceanFraction()),
            "--fragmentation", format(config.continentFragmentation()),
            "--separation", format(config.continentSeparation()),
            "--output", output.toString(),
            "--force"
        );
        processBuilder.redirectErrorStream(true);
        processBuilder.redirectOutput(engineLog.toFile());

        Process process = processBuilder.start();
        boolean completed = process.waitFor(
            GENERATION_TIMEOUT.toMillis(),
            TimeUnit.MILLISECONDS
        );
        String outputText = Files.isRegularFile(engineLog)
            ? Files.readString(engineLog, StandardCharsets.UTF_8)
            : "";
        Files.deleteIfExists(engineLog);

        if (!completed) {
            process.destroyForcibly();
            throw new IOException(
                "GeoWorld engine timed out after " +
                GENERATION_TIMEOUT.toSeconds() +
                " seconds"
            );
        }
        if (process.exitValue() != 0) {
            throw new IOException(
                "GeoWorld engine failed with exit code " +
                process.exitValue() +
                ": " +
                outputText.trim()
            );
        }

        GeoWorldV1 model = GeoWorldV1.load(output);
        validateModel(seed, config, model);
        return new GeneratedModel(
            seed,
            config,
            cacheKey,
            output,
            model,
            elapsedMillis(started),
            false
        );
    }

    public static Path resolveEngine() throws IOException {
        Path cached = resolvedEngine;
        if (cached != null && Files.isRegularFile(cached)) {
            return cached;
        }

        boolean embeddedOnly = Boolean.getBoolean(
            "geoworld.engine.embeddedOnly"
        );
        String explicit = embeddedOnly
            ? null
            : System.getProperty("geoworld.engine.path");
        if (!embeddedOnly && (explicit == null || explicit.isBlank())) {
            explicit = System.getenv("GEOWORLD_ENGINE_PATH");
        }
        if (explicit != null && !explicit.isBlank()) {
            Path path = Path.of(explicit).toAbsolutePath().normalize();
            if (Files.isRegularFile(path)) {
                resolvedEngine = path;
                return path;
            }
            throw new IOException("Configured GeoWorld engine not found: " + path);
        }

        String executable = isWindows() ? "geoworld-engine.exe" : "geoworld-engine";
        Path current = embeddedOnly ? null : Path.of(System.getProperty("user.dir"))
            .toAbsolutePath()
            .normalize();

        for (int i = 0; i < 8 && current != null; i++) {
            Path candidate = current
                .resolve("output")
                .resolve("geoworld-engine")
                .resolve(executable);
            if (Files.isRegularFile(candidate)) {
                resolvedEngine = candidate;
                return candidate;
            }
            current = current.getParent();
        }

        Path embedded = extractEmbeddedEngine();
        if (embedded != null) {
            resolvedEngine = embedded;
            return embedded;
        }

        throw new IOException(
            "GeoWorld engine executable was not found. " +
            "The MOD did not contain a native engine for this platform, " +
            "and GEOWORLD_ENGINE_PATH was not configured."
        );
    }

    private static Path extractEmbeddedEngine() throws IOException {
        String resource;
        String executable;
        if (isWindows()) {
            resource = "/native/windows-x64/geoworld-engine.exe";
            executable = "geoworld-engine.exe";
        } else {
            return null;
        }

        try (InputStream input = GeoWorldEngineBridge.class.getResourceAsStream(resource)) {
            if (input == null) {
                return null;
            }

            Path directory = resolveCacheRoot()
                .resolve("native")
                .resolve("windows-x64");
            Files.createDirectories(directory);
            Path target = directory.resolve(executable);
            Path temp = directory.resolve(executable + ".tmp");
            Files.copy(input, temp, StandardCopyOption.REPLACE_EXISTING);
            try {
                Files.move(
                    temp,
                    target,
                    StandardCopyOption.REPLACE_EXISTING,
                    StandardCopyOption.ATOMIC_MOVE
                );
            } catch (IOException atomicMoveUnavailable) {
                Files.move(
                    temp,
                    target,
                    StandardCopyOption.REPLACE_EXISTING
                );
            }
            target.toFile().setExecutable(true, true);
            return target;
        }
    }

    private static Path resolveCacheRoot() {
        String explicit = System.getProperty("geoworld.cache.dir");
        if (explicit != null && !explicit.isBlank()) {
            return Path.of(explicit).toAbsolutePath().normalize();
        }
        return Path.of(System.getProperty("user.dir"))
            .toAbsolutePath()
            .normalize()
            .resolve("geoworld-cache");
    }

    private static String cacheKey(
        long seed,
        GeoWorldCreationConfig config
    ) {
        String canonical = String.join(
            "|",
            "v4",
            Long.toString(seed),
            Integer.toString(config.width()),
            Integer.toString(config.height()),
            Integer.toString(config.cellSizeBlocks()),
            Integer.toString(config.plateCount()),
            format(config.oceanFraction()),
            format(config.continentFragmentation()),
            format(config.continentSeparation())
        );
        try {
            byte[] digest = MessageDigest
                .getInstance("SHA-256")
                .digest(canonical.getBytes(StandardCharsets.UTF_8));
            return HexFormat.of().formatHex(digest, 0, 16);
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException("SHA-256 unavailable", e);
        }
    }

    private static void validateModel(
        long seed,
        GeoWorldCreationConfig config,
        GeoWorldV1 model
    ) {
        if (model.manifest().seed() != seed) {
            throw new IllegalArgumentException(
                "GeoWorld cache seed mismatch: model=" +
                model.manifest().seed() +
                " expected=" +
                seed
            );
        }

        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        if (
            geometry.width() != config.width() ||
            geometry.height() != config.height()
        ) {
            throw new IllegalArgumentException(
                "GeoWorld cache geometry mismatch: model=" +
                geometry.width() +
                "x" +
                geometry.height() +
                " expected=" +
                config.width() +
                "x" +
                config.height()
            );
        }

        double spacingX = geometry.width() <= 1
            ? 0.0
            : geometry.spanXBlocks() / (geometry.width() - 1.0);
        double spacingZ = geometry.height() <= 1
            ? 0.0
            : geometry.spanZBlocks() / (geometry.height() - 1.0);
        if (
            Math.abs(spacingX - config.cellSizeBlocks()) > 1.0e-9 ||
            Math.abs(spacingZ - config.cellSizeBlocks()) > 1.0e-9
        ) {
            throw new IllegalArgumentException(
                "GeoWorld cache cell-size mismatch: model=" +
                spacingX +
                "x" +
                spacingZ +
                " expected=" +
                config.cellSizeBlocks()
            );
        }
    }

    private static boolean isWindows() {
        return System.getProperty("os.name", "")
            .toLowerCase(Locale.ROOT)
            .contains("win");
    }

    private static String format(double value) {
        return String.format(Locale.ROOT, "%.6f", value);
    }

    private static long elapsedMillis(long startedNanos) {
        return TimeUnit.NANOSECONDS.toMillis(System.nanoTime() - startedNanos);
    }

    public record GeneratedModel(
        long seed,
        GeoWorldCreationConfig config,
        String cacheKey,
        Path path,
        GeoWorldV1 model,
        long durationMs,
        boolean cached
    ) {}
}
