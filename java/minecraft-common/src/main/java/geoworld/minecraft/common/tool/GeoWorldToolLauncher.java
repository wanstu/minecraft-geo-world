package geoworld.minecraft.common.tool;

import java.io.IOException;
import java.io.InputStream;
import java.net.URI;
import java.net.URL;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.concurrent.TimeUnit;

public final class GeoWorldToolLauncher {
    private GeoWorldToolLauncher() {}

    public static void main(String[] args) throws Exception {
        try (ManagedProcess managed = startManaged(args)) {
            int exitCode = managed.waitFor();
            if (exitCode != 0) {
                throw new IllegalStateException(
                    "GeoWorld Web Tool exited with code " + exitCode
                );
            }
        }
    }

    public static ManagedProcess startManaged(String... args) throws Exception {
        Platform platform = detectPlatform();
        String executableName = platform.windows()
            ? "geoworld-tool.exe"
            : "geoworld-tool";
        String resource =
            "/geoworld/tool/native/" + platform.id() + "/" + executableName;

        // Keep native children in GeoWorld's own writable runtime directory.
        // Some Windows environments deny Java-created executables in the
        // global Temp directory even when the identical binary runs here.
        String explicitCache = System.getProperty("geoworld.cache.dir");
        Path cacheRoot = explicitCache != null && !explicitCache.isBlank()
            ? Path.of(explicitCache)
            : Path.of(System.getProperty("user.dir"), "geoworld-cache");
        Path toolRoot = cacheRoot.toAbsolutePath().normalize().resolve("tool");
        Files.createDirectories(toolRoot);
        Path tempDir = Files.createTempDirectory(toolRoot, "run-");
        Path executable = tempDir.resolve(executableName);
        boolean started = false;
        try {
            extract(resource, executable);
            if (
                !platform.windows() &&
                !executable.toFile().setExecutable(true, true)
            ) {
                throw new IOException(
                    "Unable to make GeoWorld Web Tool executable: " + executable
                );
            }

            List<String> command = new ArrayList<>();
            command.add(executable.toString());
            for (String arg : args) {
                command.add(arg);
            }

            ProcessBuilder builder = new ProcessBuilder(command).inheritIO();
            Path plannerClasspath = resolvePlannerClasspath();
            if (plannerClasspath != null) {
                builder.environment().put(
                    "GEOWORLD_TOOL_CLASSPATH",
                    plannerClasspath.toString()
                );
                if (Files.isRegularFile(plannerClasspath)) {
                    builder.environment().put(
                        "GEOWORLD_TOOL_JAR",
                        plannerClasspath.toString()
                    );
                }
            } else {
                System.err.println(
                    "GeoWorld Web Tool warning: unable to resolve Java planner " +
                    "classpath; exact minerals/strongholds will be unavailable"
                );
            }
            Path javaExecutable = Path.of(
                System.getProperty("java.home"),
                "bin",
                platform.windows() ? "java.exe" : "java"
            );
            builder.environment().put(
                "GEOWORLD_JAVA",
                javaExecutable.toString()
            );

            Process process = builder.start();
            started = true;
            return new ManagedProcess(process, tempDir);
        } finally {
            if (!started) {
                deleteTree(tempDir);
            }
        }
    }

    private static Path resolvePlannerClasspath() {
        try {
            URL location = GeoWorldToolLauncher.class.getProtectionDomain()
                .getCodeSource().getLocation();
            if (location != null && "file".equalsIgnoreCase(location.getProtocol())) {
                Path direct = Path.of(location.toURI()).toAbsolutePath().normalize();
                if (Files.exists(direct)) {
                    return direct;
                }
            }
        } catch (Exception ignored) {
            // NeoForge may expose transformed classes through a union filesystem.
        }

        URL resource = GeoWorldToolLauncher.class.getResource(
            "GeoWorldToolLauncher.class"
        );
        Path physicalJar = resolvePhysicalJarFromLocation(
            resource == null ? null : resource.toExternalForm()
        );
        if (physicalJar != null && Files.isRegularFile(physicalJar)) {
            return physicalJar;
        }
        return null;
    }

    static Path resolvePhysicalJarFromLocation(String location) {
        if (location == null || location.isBlank()) {
            return null;
        }
        String candidate = location.trim();
        while (candidate.startsWith("jar:")) {
            candidate = candidate.substring(4);
        }
        int jarEnd = candidate.toLowerCase(Locale.ROOT).indexOf(".jar");
        if (jarEnd < 0) {
            return null;
        }
        candidate = candidate.substring(0, jarEnd + 4);

        while (candidate.startsWith("union:")) {
            candidate = candidate.substring("union:".length());
        }
        try {
            if (candidate.startsWith("file:")) {
                return Path.of(URI.create(candidate))
                    .toAbsolutePath().normalize();
            }
            if (!candidate.startsWith("/")) {
                candidate = "/" + candidate;
            }
            return Path.of(URI.create("file:" + candidate))
                .toAbsolutePath().normalize();
        } catch (Exception ignored) {
            return null;
        }
    }

    private static void extract(String resource, Path target) throws IOException {
        try (
            InputStream in =
                GeoWorldToolLauncher.class.getResourceAsStream(resource)
        ) {
            if (in == null) {
                throw new IOException(
                    "GeoWorld Web Tool is not bundled for this platform: " +
                    resource
                );
            }
            Files.copy(in, target, StandardCopyOption.REPLACE_EXISTING);
        }
    }

    private static Platform detectPlatform() {
        String os = System.getProperty("os.name", "").toLowerCase(Locale.ROOT);
        String arch = System.getProperty("os.arch", "").toLowerCase(Locale.ROOT);

        String osPart;
        boolean windows;
        if (os.contains("win")) {
            osPart = "windows";
            windows = true;
        } else if (os.contains("mac") || os.contains("darwin")) {
            osPart = "macos";
            windows = false;
        } else if (os.contains("linux")) {
            osPart = "linux";
            windows = false;
        } else {
            throw new IllegalStateException(
                "Unsupported operating system: " + os
            );
        }

        String archPart;
        if (
            arch.equals("amd64") ||
            arch.equals("x86_64") ||
            arch.equals("x64")
        ) {
            archPart = "x64";
        } else if (
            arch.equals("aarch64") ||
            arch.equals("arm64")
        ) {
            archPart = "arm64";
        } else {
            throw new IllegalStateException(
                "Unsupported CPU architecture: " + arch
            );
        }
        return new Platform(osPart + "-" + archPart, windows);
    }

    private static void deleteTree(Path root) {
        if (root == null || !Files.exists(root)) {
            return;
        }
        try (var stream = Files.walk(root)) {
            stream.sorted((a, b) -> b.getNameCount() - a.getNameCount())
                .forEach(path -> {
                    try {
                        Files.deleteIfExists(path);
                    } catch (IOException ignored) {
                        path.toFile().deleteOnExit();
                    }
                });
        } catch (IOException ignored) {
            root.toFile().deleteOnExit();
        }
    }

    public static final class ManagedProcess implements AutoCloseable {
        private final Process process;
        private final Path tempDir;
        private boolean closed;

        private ManagedProcess(Process process, Path tempDir) {
            this.process = process;
            this.tempDir = tempDir;
        }

        public boolean isAlive() {
            return process.isAlive();
        }

        public long pid() {
            return process.pid();
        }

        public Integer exitCodeIfExited() {
            if (process.isAlive()) {
                return null;
            }
            try {
                return process.exitValue();
            } catch (IllegalThreadStateException ignored) {
                return null;
            }
        }

        public int waitFor() throws InterruptedException {
            return process.waitFor();
        }

        @Override
        public synchronized void close() {
            if (closed) {
                return;
            }
            closed = true;
            if (process.isAlive()) {
                process.destroy();
                try {
                    if (!process.waitFor(2, TimeUnit.SECONDS)) {
                        process.destroyForcibly();
                        process.waitFor(2, TimeUnit.SECONDS);
                    }
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    process.destroyForcibly();
                }
            }
            deleteTree(tempDir);
        }
    }

    private record Platform(String id, boolean windows) {}
}
