package geoworld.minecraft.v26_3;

import geoworld.minecraft.nbt.Nbt;

import java.io.IOException;
import java.nio.file.Path;
import java.time.Instant;
import java.util.List;

public final class Minecraft26_3Metadata {
    public static final int DATA_VERSION = 5023;
    public static final int LEVEL_DATA_VERSION = 19133;
    public static final String VERSION_NAME = "26.3";
    public static final String DATA_VERSION_SERIES = "main";

    public void writeLevelDat(
        Path worldDirectory,
        String levelName,
        int spawnX,
        int spawnY,
        int spawnZ
    ) throws IOException {
        writeLevelDat(worldDirectory, levelName, spawnX, spawnY, spawnZ, false);
    }

    public void writeLevelDat(
        Path worldDirectory,
        String levelName,
        int spawnX,
        int spawnY,
        int spawnZ,
        boolean qaMode
    ) throws IOException {
        if (worldDirectory == null || levelName == null) {
            throw new NullPointerException("worldDirectory/levelName");
        }

        long lastPlayed = Instant.now().toEpochMilli();

        Nbt.CompoundTag version = Nbt.compound()
            .put("Name", Nbt.string(VERSION_NAME))
            .put("Id", Nbt.intTag(DATA_VERSION))
            .put("Snapshot", Nbt.byteTag(0))
            .put("Series", Nbt.string(DATA_VERSION_SERIES));

        Nbt.CompoundTag spawn = Nbt.compound()
            .put("dimension", Nbt.string("minecraft:overworld"))
            .put("pos", Nbt.intArray(new int[] {spawnX, spawnY, spawnZ}))
            .put("yaw", Nbt.floatTag(0))
            .put("pitch", Nbt.floatTag(0));

        Nbt.CompoundTag difficulty = Nbt.compound()
            .put("difficulty", Nbt.string("normal"))
            .put("hardcore", Nbt.byteTag(0))
            .put("locked", Nbt.byteTag(0));

        Nbt.CompoundTag dataPacks = Nbt.compound()
            .put("Enabled", Nbt.list(List.of(Nbt.string("vanilla"))))
            .put("Disabled", Nbt.list(List.of()));

        Nbt.CompoundTag data = Nbt.compound()
            .put("ServerBrands", Nbt.list(List.of()))
            .put("WasModded", Nbt.byteTag(0))
            .put("Version", version)
            .put("version_history", Nbt.list(List.of()))
            .put("DataVersion", Nbt.intTag(DATA_VERSION))
            .put("GameType", Nbt.intTag(qaMode ? 1 : 0))
            .put("spawn", spawn)
            .put("Time", Nbt.longTag(0))
            .put("LastPlayed", Nbt.longTag(lastPlayed))
            .put("LevelName", Nbt.string(levelName))
            .put("version", Nbt.intTag(LEVEL_DATA_VERSION))
            .put("allowCommands", Nbt.byteTag(qaMode ? 1 : 0))
            .put("initialized", Nbt.byteTag(1))
            .put("difficulty_settings", difficulty)
            .put("DataPacks", dataPacks);

        Nbt.CompoundTag root = Nbt.compound().put("Data", data);
        Nbt.writeRootCompressed(worldDirectory.resolve("level.dat"), "", root);
    }

    public void writeWorldGenSettings(Path worldDirectory, long seed) throws IOException {
        if (worldDirectory == null) throw new NullPointerException("worldDirectory");

        Nbt.CompoundTag dimensions = Nbt.compound()
            .put(
                "minecraft:overworld",
                levelStem(
                    "minecraft:overworld",
                    noiseGenerator(
                        multiNoiseBiomeSource("minecraft:overworld"),
                        "minecraft:overworld"
                    )
                )
            )
            .put(
                "minecraft:the_nether",
                levelStem(
                    "minecraft:the_nether",
                    noiseGenerator(
                        multiNoiseBiomeSource("minecraft:nether"),
                        "minecraft:nether"
                    )
                )
            )
            .put(
                "minecraft:the_end",
                levelStem(
                    "minecraft:the_end",
                    noiseGenerator(
                        Nbt.compound().put("type", Nbt.string("minecraft:the_end")),
                        "minecraft:end"
                    )
                )
            );

        Nbt.CompoundTag data = Nbt.compound()
            .put("seed", Nbt.longTag(seed))
            .put("generate_structures", Nbt.byteTag(1))
            .put("bonus_chest", Nbt.byteTag(0))
            .put("dimensions", dimensions);

        Nbt.CompoundTag root = Nbt.compound()
            .put("data", data)
            .put("DataVersion", Nbt.intTag(DATA_VERSION));

        Nbt.writeRootCompressed(
            worldDirectory
                .resolve("data")
                .resolve("minecraft")
                .resolve("world_gen_settings.dat"),
            "",
            root
        );
    }

    private static Nbt.CompoundTag levelStem(
        String dimensionType,
        Nbt.CompoundTag generator
    ) {
        return Nbt.compound()
            .put("type", Nbt.string(dimensionType))
            .put("generator", generator);
    }

    private static Nbt.CompoundTag noiseGenerator(
        Nbt.CompoundTag biomeSource,
        String settings
    ) {
        return Nbt.compound()
            .put("type", Nbt.string("minecraft:noise"))
            .put("biome_source", biomeSource)
            .put("settings", Nbt.string(settings));
    }

    private static Nbt.CompoundTag multiNoiseBiomeSource(String preset) {
        return Nbt.compound()
            .put("type", Nbt.string("minecraft:multi_noise"))
            .put("preset", Nbt.string(preset));
    }
}
