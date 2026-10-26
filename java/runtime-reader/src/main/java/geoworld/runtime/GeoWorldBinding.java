package geoworld.runtime;

import java.io.IOException;
import java.io.Reader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.util.Optional;
import java.util.Properties;

public final class GeoWorldBinding {
    public static final String FORMAT = "geoworld-binding";
    public static final int VERSION = 2;
    public static final String DIRECTORY = "data/geoworld";
    public static final String CONFIG_FILE = "geoworld.properties";
    public static final String DEFAULT_MODEL_FILE = "world.geoworld";

    private GeoWorldBinding() {}

    public record Binding(
        Path worldDirectory,
        Path configPath,
        Path modelPath,
        int bindingVersion,
        String minecraftVersion,
        int modelSchemaVersion,
        String generatorVersion,
        long seed,
        String worldProfileVersion,
        String generationRulesVersion,
        String worldModelVersion,
        String preset,
        String parameterSnapshotSha256
    ) {}

    public static Binding write(
        Path worldDirectory,
        Path sourceModel,
        String minecraftVersion
    ) throws IOException {
        if (worldDirectory == null || sourceModel == null || minecraftVersion == null) {
            throw new NullPointerException("worldDirectory/sourceModel/minecraftVersion");
        }

        Path world = worldDirectory.toAbsolutePath().normalize();
        Path source = sourceModel.toAbsolutePath().normalize();
        if (!Files.isRegularFile(source)) {
            throw new IOException("GeoWorld model not found: " + source);
        }

        GeoWorldV1 model = GeoWorldV1.load(source);
        Path directory = world.resolve(DIRECTORY);
        Files.createDirectories(directory);

        Path modelPath = directory.resolve(DEFAULT_MODEL_FILE);
        Files.copy(source, modelPath, StandardCopyOption.REPLACE_EXISTING);

        GeoWorldV1.Manifest manifest = model.manifest();
        boolean lockedProfile =
            !manifest.worldProfileVersion().isBlank() &&
            !manifest.generationRulesVersion().isBlank() &&
            !manifest.worldModelVersion().isBlank() &&
            !manifest.preset().isBlank() &&
            !manifest.parameterSnapshotSha256().isBlank();
        int bindingVersion = lockedProfile ? VERSION : 1;
        StringBuilder contents = new StringBuilder()
            .append("format=").append(FORMAT).append('\n')
            .append("version=").append(bindingVersion).append('\n')
            .append("model=").append(DEFAULT_MODEL_FILE).append('\n')
            .append("minecraftVersion=").append(escapeProperty(minecraftVersion)).append('\n')
            .append("modelSchemaVersion=").append(manifest.schemaVersion()).append('\n')
            .append("generatorVersion=").append(escapeProperty(manifest.generatorVersion())).append('\n')
            .append("seed=").append(manifest.seed()).append('\n');
        if (lockedProfile) {
            contents
                .append("worldProfileVersion=").append(escapeProperty(manifest.worldProfileVersion())).append('\n')
                .append("generationRulesVersion=").append(escapeProperty(manifest.generationRulesVersion())).append('\n')
                .append("worldModelVersion=").append(escapeProperty(manifest.worldModelVersion())).append('\n')
                .append("preset=").append(escapeProperty(manifest.preset())).append('\n')
                .append("parameterSnapshotSha256=").append(escapeProperty(manifest.parameterSnapshotSha256())).append('\n');
        }

        Path configPath = directory.resolve(CONFIG_FILE);
        Files.writeString(configPath, contents.toString(), StandardCharsets.UTF_8);

        return new Binding(
            world,
            configPath,
            modelPath,
            bindingVersion,
            minecraftVersion,
            manifest.schemaVersion(),
            manifest.generatorVersion(),
            manifest.seed(),
            manifest.worldProfileVersion(),
            manifest.generationRulesVersion(),
            manifest.worldModelVersion(),
            manifest.preset(),
            manifest.parameterSnapshotSha256()
        );
    }

    public static Optional<Binding> detect(Path worldDirectory) throws IOException {
        if (worldDirectory == null) throw new NullPointerException("worldDirectory");

        Path world = worldDirectory.toAbsolutePath().normalize();
        Path configPath = world.resolve(DIRECTORY).resolve(CONFIG_FILE);
        if (!Files.isRegularFile(configPath)) {
            return Optional.empty();
        }

        Properties properties = new Properties();
        try (Reader reader = Files.newBufferedReader(configPath, StandardCharsets.UTF_8)) {
            properties.load(reader);
        }

        require(FORMAT.equals(properties.getProperty("format")), "unsupported binding format");
        int version = parseInt(properties, "version");
        require(version == 1 || version == VERSION, "unsupported binding version: " + version);

        String modelName = required(properties, "model");
        Path relativeModel = Path.of(modelName);
        require(
            !relativeModel.isAbsolute() &&
            relativeModel.getNameCount() == 1 &&
            !modelName.equals(".") &&
            !modelName.equals(".."),
            "binding model must be a plain file name"
        );

        String minecraftVersion = required(properties, "minecraftVersion");
        int modelSchemaVersion = parseInt(properties, "modelSchemaVersion");
        String generatorVersion = required(properties, "generatorVersion");
        long seed = parseLong(properties, "seed");
        String worldProfileVersion = version >= 2 ? required(properties, "worldProfileVersion") : "";
        String generationRulesVersion = version >= 2 ? required(properties, "generationRulesVersion") : "";
        String worldModelVersion = version >= 2 ? required(properties, "worldModelVersion") : "";
        String preset = version >= 2 ? required(properties, "preset") : "legacy";
        String parameterSnapshotSha256 = version >= 2 ? required(properties, "parameterSnapshotSha256") : "";

        Path modelPath = configPath.getParent().resolve(relativeModel).normalize();
        require(
            modelPath.getParent().equals(configPath.getParent()),
            "binding model escapes GeoWorld directory"
        );
        require(Files.isRegularFile(modelPath), "bound GeoWorld model is missing");

        return Optional.of(new Binding(
            world,
            configPath,
            modelPath,
            version,
            minecraftVersion,
            modelSchemaVersion,
            generatorVersion,
            seed,
            worldProfileVersion,
            generationRulesVersion,
            worldModelVersion,
            preset,
            parameterSnapshotSha256
        ));
    }

    public static GeoWorldV1 loadModel(Binding binding) throws IOException {
        if (binding == null) throw new NullPointerException("binding");

        GeoWorldV1 model = GeoWorldV1.load(binding.modelPath());
        GeoWorldV1.Manifest manifest = model.manifest();

        require(
            manifest.schemaVersion() == binding.modelSchemaVersion(),
            "binding/model schemaVersion mismatch"
        );
        require(
            manifest.generatorVersion().equals(binding.generatorVersion()),
            "binding/model generatorVersion mismatch"
        );
        require(
            manifest.seed() == binding.seed(),
            "binding/model seed mismatch"
        );
        if (binding.bindingVersion() >= 2) {
            require(
                manifest.worldProfileVersion().equals(binding.worldProfileVersion()),
                "binding/model worldProfileVersion mismatch"
            );
            require(
                manifest.generationRulesVersion().equals(binding.generationRulesVersion()),
                "binding/model generationRulesVersion mismatch"
            );
            require(
                manifest.worldModelVersion().equals(binding.worldModelVersion()),
                "binding/model worldModelVersion mismatch"
            );
            require(
                manifest.preset().equals(binding.preset()),
                "binding/model preset mismatch"
            );
            require(
                manifest.parameterSnapshotSha256().equals(binding.parameterSnapshotSha256()),
                "binding/model parameter snapshot mismatch"
            );
        }
        return model;
    }

    private static String required(Properties properties, String key) {
        String value = properties.getProperty(key);
        if (value == null || value.isBlank()) {
            throw new IllegalArgumentException("missing binding property: " + key);
        }
        return value.trim();
    }

    private static int parseInt(Properties properties, String key) {
        try {
            return Integer.parseInt(required(properties, key));
        } catch (NumberFormatException e) {
            throw new IllegalArgumentException("invalid integer binding property: " + key, e);
        }
    }

    private static long parseLong(Properties properties, String key) {
        try {
            return Long.parseLong(required(properties, key));
        } catch (NumberFormatException e) {
            throw new IllegalArgumentException("invalid long binding property: " + key, e);
        }
    }

    private static void require(boolean condition, String message) {
        if (!condition) throw new IllegalArgumentException(message);
    }

    private static String escapeProperty(String value) {
        return value
            .replace("\\", "\\\\")
            .replace("\n", "\\n")
            .replace("\r", "\\r")
            .replace("=", "\\=")
            .replace(":", "\\:");
    }
}
