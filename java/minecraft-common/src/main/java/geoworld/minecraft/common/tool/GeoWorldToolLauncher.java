package geoworld.minecraft.common.tool;

import java.io.IOException;
import java.io.InputStream;
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

        Path tempDir = Files.createTempDirectory("geoworld-tool-");
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
            Path launcher = Path.of(
                GeoWorldToolLauncher.class.getProtectionDomain()
                    .getCodeSource().getLocation().toURI()
            ).toAbsolutePath().normalize();
            if (Files.isRegularFile(launcher)) {
                builder.environment().put(
                    "GEOWORLD_TOOL_JAR",
                    launcher.toString()
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
