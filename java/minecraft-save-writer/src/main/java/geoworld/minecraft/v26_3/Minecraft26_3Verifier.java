package geoworld.minecraft.v26_3;

import geoworld.minecraft.anvil.RegionFile;
import geoworld.runtime.GeoWorldBinding;
import geoworld.runtime.GeoWorldV1;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Arrays;
import java.util.Comparator;

public final class Minecraft26_3Verifier {
    private Minecraft26_3Verifier() {}

    public static void main(String[] args) throws Exception {
        Path model = args.length > 0
            ? Path.of(args[0])
            : Path.of("internal/goldenvectors/testdata/runtime-v1.geoworld");

        verifyHeightMapping();
        verifyPackedValues();
        verifyPreviewGeneration(model);

        System.out.println("Minecraft 26.3 preview verification passed.");
    }

    private static void verifyHeightMapping() {
        HeightMapping mapping = new HeightMapping();

        exact("height 0.0", mapping.boundaryY(0.0), -64);
        exact("height 0.5", mapping.boundaryY(0.5), 63);
        exact("height 1.0", mapping.boundaryY(1.0), 320);
        exact("relative min", mapping.relativeHeightmapValue(-64), 0);
        exact("relative sea", mapping.relativeHeightmapValue(63), 127);
        exact("relative max", mapping.relativeHeightmapValue(320), 384);

        System.out.println("PASS 26.3 height mapping anchors");
    }

    private static void verifyPackedValues() {
        int[] fourBit = new int[4096];
        for (int i = 0; i < fourBit.length; i++) {
            fourBit[i] = i & 0x0f;
        }
        long[] fourPacked = PackedValues.pack(fourBit, 4);
        exact("4-bit block-state long count", fourPacked.length, 256);
        if (!Arrays.equals(
            fourBit,
            PackedValues.unpack(fourPacked, fourBit.length, 4)
        )) {
            throw new AssertionError("4-bit block-state packing mismatch");
        }

        int[] nineBit = new int[256];
        for (int i = 0; i < nineBit.length; i++) {
            nineBit[i] = i % 385;
        }
        long[] ninePacked = PackedValues.pack(nineBit, 9);
        exact("9-bit heightmap long count", ninePacked.length, 37);
        if (!Arrays.equals(
            nineBit,
            PackedValues.unpack(ninePacked, nineBit.length, 9)
        )) {
            throw new AssertionError("9-bit heightmap packing mismatch");
        }

        exact("block palette 2 entries", PackedValues.blockStateBits(2), 4);
        exact("block palette 16 entries", PackedValues.blockStateBits(16), 4);
        exact("block palette 17 entries", PackedValues.blockStateBits(17), 5);
        exact("biome palette 2 entries", PackedValues.biomeBits(2), 1);
        exact("biome palette 4 entries", PackedValues.biomeBits(4), 2);
        exact("biome palette 8 entries", PackedValues.biomeBits(8), 3);

        System.out.println("PASS 26.3 palette/heightmap bit packing");
    }

    private static void verifyPreviewGeneration(Path model) throws Exception {
        Path root = Files.createTempDirectory("geoworld-mc26_3-");
        Path world = root.resolve("world");
        try {
            PreviewWorldGenerator.generate(
                new PreviewWorldGenerator.Arguments(
                    model,
                    world,
                    "GeoWorld 26.3 Smoke",
                    1,
                    false
                )
            );

            requireFile(world.resolve("level.dat"));
            requireFile(
                world
                    .resolve("data")
                    .resolve("minecraft")
                    .resolve("world_gen_settings.dat")
            );
            if (Files.exists(world.resolve("data").resolve("world_gen_settings.dat"))) {
                throw new AssertionError(
                    "legacy wrong world_gen_settings.dat path must not be written"
                );
            }

            GeoWorldBinding.Binding binding = GeoWorldBinding.detect(world)
                .orElseThrow(() -> new AssertionError("GeoWorld binding was not written"));
            requireFile(binding.configPath());
            requireFile(binding.modelPath());
            if (!"26.3".equals(binding.minecraftVersion())) {
                throw new AssertionError(
                    "binding minecraftVersion=" + binding.minecraftVersion()
                );
            }
            GeoWorldV1 boundModel = GeoWorldBinding.loadModel(binding);
            if (boundModel.manifest().seed() != binding.seed()) {
                throw new AssertionError("bound model seed mismatch");
            }

            long regionFiles;
            try (var files = Files.list(world.resolve("region"))) {
                regionFiles = files
                    .filter(path -> path.getFileName().toString().endsWith(".mca"))
                    .count();
            }
            exact("region file count", (int) regionFiles, 4);

            int chunks = 0;
            for (int chunkZ = -1; chunkZ <= 1; chunkZ++) {
                for (int chunkX = -1; chunkX <= 1; chunkX++) {
                    Path region = RegionFile.regionPath(
                        world.resolve("region"),
                        RegionFile.regionXForChunk(chunkX),
                        RegionFile.regionZForChunk(chunkZ)
                    );
                    byte[] nbt = RegionFile.readChunkNbt(
                        region,
                        chunkX,
                        chunkZ
                    );
                    if (nbt == null || nbt.length == 0) {
                        throw new AssertionError(
                            "missing preview chunk " + chunkX + "," + chunkZ
                        );
                    }
                    chunks++;
                }
            }
            exact("preview chunk count", chunks, 9);
            System.out.println("PASS 26.3 preview save generation");
        } finally {
            deleteTree(root);
        }
    }

    private static void requireFile(Path path) throws Exception {
        if (!Files.isRegularFile(path) || Files.size(path) == 0) {
            throw new AssertionError("missing/empty file: " + path);
        }
    }

    private static void exact(String name, int got, int want) {
        if (got != want) {
            throw new AssertionError(name + " got=" + got + " want=" + want);
        }
    }

    private static void deleteTree(Path root) throws Exception {
        if (!Files.exists(root)) return;
        try (var stream = Files.walk(root)) {
            for (Path path : stream.sorted(Comparator.reverseOrder()).toList()) {
                Files.deleteIfExists(path);
            }
        }
    }
}
