package geoworld.minecraft.v26_3;

import geoworld.minecraft.anvil.RegionFile;
import geoworld.runtime.GeoWorldV1;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

public final class PreviewWorldGenerator {
    private PreviewWorldGenerator() {}

    public static void main(String[] args) throws Exception {
        Arguments parsed = Arguments.parse(args);
        generate(parsed);
    }

    public static void generate(Arguments args) throws Exception {
        GeoWorldV1 model = GeoWorldV1.load(args.model());
        HeightMapping heights = new HeightMapping();
        Spawn spawn = findSpawn(model, heights);

        Path world = args.output().toAbsolutePath();
        if (Files.exists(world)) {
            if (!args.force()) {
                throw new IllegalArgumentException(
                    "output already exists: " + world + " (pass --force to replace it)"
                );
            }
            deleteTree(world);
        }
        Files.createDirectories(world);
        Files.createDirectories(world.resolve("region"));

        Minecraft26_3Metadata metadata = new Minecraft26_3Metadata();
        metadata.writeLevelDat(
            world,
            args.levelName(),
            spawn.x(),
            spawn.y(),
            spawn.z()
        );
        metadata.writeWorldGenSettings(world, model.manifest().seed());

        Minecraft26_3Chunk chunkWriter = new Minecraft26_3Chunk(model, heights);
        int centerChunkX = Math.floorDiv(spawn.x(), 16);
        int centerChunkZ = Math.floorDiv(spawn.z(), 16);

        Map<RegionKey, List<RegionFile.Chunk>> regions = new LinkedHashMap<>();
        int chunkCount = 0;

        for (
            int chunkZ = centerChunkZ - args.radiusChunks();
            chunkZ <= centerChunkZ + args.radiusChunks();
            chunkZ++
        ) {
            for (
                int chunkX = centerChunkX - args.radiusChunks();
                chunkX <= centerChunkX + args.radiusChunks();
                chunkX++
            ) {
                byte[] nbt = chunkWriter.writeChunkNbt(chunkX, chunkZ);
                RegionKey key = new RegionKey(
                    RegionFile.regionXForChunk(chunkX),
                    RegionFile.regionZForChunk(chunkZ)
                );
                regions.computeIfAbsent(key, ignored -> new ArrayList<>())
                    .add(new RegionFile.Chunk(chunkX, chunkZ, nbt));
                chunkCount++;
            }
        }

        int timestamp = 1;
        for (Map.Entry<RegionKey, List<RegionFile.Chunk>> entry : regions.entrySet()) {
            RegionKey key = entry.getKey();
            Path regionPath = RegionFile.regionPath(
                world.resolve("region"),
                key.x(),
                key.z()
            );
            RegionFile.write(
                regionPath,
                key.x(),
                key.z(),
                entry.getValue(),
                timestamp
            );
            verifyRegion(regionPath, entry.getValue());
        }

        System.out.printf(
            "Generated Minecraft Java 26.3 preview world%n" +
            "  model: %s%n" +
            "  seed: %d%n" +
            "  output: %s%n" +
            "  spawn: (%d, %d, %d)%n" +
            "  center chunk: (%d, %d)%n" +
            "  radius: %d chunks%n" +
            "  chunks: %d%n" +
            "  regions: %d%n",
            args.model().toAbsolutePath(),
            model.manifest().seed(),
            world,
            spawn.x(),
            spawn.y(),
            spawn.z(),
            centerChunkX,
            centerChunkZ,
            args.radiusChunks(),
            chunkCount,
            regions.size()
        );
    }

    private static Spawn findSpawn(GeoWorldV1 model, HeightMapping heights) {
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        int minX = (int) Math.ceil(geometry.originX());
        int minZ = (int) Math.ceil(geometry.originZ());
        int maxX = (int) Math.floor(geometry.originX() + geometry.spanXBlocks());
        int maxZ = (int) Math.floor(geometry.originZ() + geometry.spanZBlocks());

        List<Candidate> candidates = new ArrayList<>();
        int step = 16;
        for (int z = alignUp(minZ, step); z <= maxZ; z += step) {
            for (int x = alignUp(minX, step); x <= maxX; x += step) {
                GeoWorldV1.RuntimeSample sample = model.sample(x, z);
                if (
                    !"None".equals(sample.waterKind()) ||
                    !sample.sourceLand() ||
                    sample.semanticFallback() ||
                    sample.semanticMismatch()
                ) {
                    continue;
                }
                long distance2 = (long) x * x + (long) z * z;
                candidates.add(new Candidate(x, z, distance2, sample));
            }
        }

        Candidate best = candidates.stream()
            .min(Comparator.comparingLong(Candidate::distance2))
            .orElseThrow(() -> new IllegalStateException("no safe land spawn found"));

        int surfaceBoundary = heights.boundaryY(best.sample().groundHeight());
        int spawnY = Math.min(
            HeightMapping.MAX_Y_EXCLUSIVE - 2,
            Math.max(HeightMapping.MIN_Y + 2, surfaceBoundary + 1)
        );
        return new Spawn(best.x(), spawnY, best.z());
    }

    private static int alignUp(int value, int step) {
        return Math.floorDiv(value + step - 1, step) * step;
    }

    private static void verifyRegion(
        Path regionPath,
        List<RegionFile.Chunk> chunks
    ) throws IOException {
        for (RegionFile.Chunk chunk : chunks) {
            byte[] got = RegionFile.readChunkNbt(
                regionPath,
                chunk.chunkX(),
                chunk.chunkZ()
            );
            if (!Arrays.equals(got, chunk.nbt())) {
                throw new IOException(
                    "Region round-trip mismatch for chunk " +
                    chunk.chunkX() + "," + chunk.chunkZ()
                );
            }
        }
    }

    private static void deleteTree(Path root) throws IOException {
        if (!Files.exists(root)) return;
        try (var stream = Files.walk(root)) {
            List<Path> paths = stream
                .sorted(Comparator.reverseOrder())
                .toList();
            for (Path path : paths) Files.deleteIfExists(path);
        }
    }

    private record RegionKey(int x, int z) {}

    private record Spawn(int x, int y, int z) {}

    private record Candidate(
        int x,
        int z,
        long distance2,
        GeoWorldV1.RuntimeSample sample
    ) {}

    public record Arguments(
        Path model,
        Path output,
        String levelName,
        int radiusChunks,
        boolean force
    ) {
        static Arguments parse(String[] args) {
            Path model = Path.of(
                "internal/goldenvectors/testdata/runtime-v1.geoworld"
            );
            Path output = Path.of("output/minecraft-26.3-preview/GeoWorld-Preview");
            String levelName = "GeoWorld 26.3 Preview";
            int radius = 8;
            boolean force = false;

            for (int i = 0; i < args.length; i++) {
                switch (args[i]) {
                    case "--model" -> model = Path.of(requireValue(args, ++i, "--model"));
                    case "--output" -> output = Path.of(requireValue(args, ++i, "--output"));
                    case "--name" -> levelName = requireValue(args, ++i, "--name");
                    case "--radius" -> {
                        radius = Integer.parseInt(requireValue(args, ++i, "--radius"));
                        if (radius < 0 || radius > 31) {
                            throw new IllegalArgumentException(
                                "--radius must be in 0..31 chunks"
                            );
                        }
                    }
                    case "--force" -> force = true;
                    case "--help", "-h" -> {
                        System.out.println(
                            "Usage: generate-preview-world.cmd [options]\n" +
                            "  --model <file.geoworld>\n" +
                            "  --output <world-directory>\n" +
                            "  --name <level-name>\n" +
                            "  --radius <0..31 chunks>\n" +
                            "  --force"
                        );
                        System.exit(0);
                    }
                    default -> throw new IllegalArgumentException(
                        "unknown argument: " + args[i]
                    );
                }
            }

            return new Arguments(model, output, levelName, radius, force);
        }

        private static String requireValue(
            String[] args,
            int index,
            String option
        ) {
            if (index >= args.length) {
                throw new IllegalArgumentException(option + " requires a value");
            }
            return args[index];
        }
    }
}
