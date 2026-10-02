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
        Result result = generate(parsed);
        if (parsed.resultJson() != null) {
            writeResultJson(parsed.resultJson(), result);
        }
    }

    public static Result generate(Arguments args) throws Exception {
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
            spawn.z(),
            args.qaMode()
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

        Result result = new Result(
            "26.3",
            model.manifest().seed(),
            world,
            spawn.x(),
            spawn.y(),
            spawn.z(),
            centerChunkX,
            centerChunkZ,
            args.radiusChunks(),
            chunkCount,
            regions.size(),
            args.qaMode()
        );

        System.out.printf(
            "Generated Minecraft Java 26.3 preview world%n" +
            "  model: %s%n" +
            "  seed: %d%n" +
            "  output: %s%n" +
            "  spawn: (%d, %d, %d)%n" +
            "  center chunk: (%d, %d)%n" +
            "  radius: %d chunks%n" +
            "  chunks: %d%n" +
            "  regions: %d%n" +
            "  QA mode: %s%n",
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
            regions.size(),
            args.qaMode()
        );
        return result;
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

    private static void writeResultJson(Path path, Result result) throws IOException {
        Path absolute = path.toAbsolutePath();
        Path parent = absolute.getParent();
        if (parent != null) Files.createDirectories(parent);

        String output = result.output().toAbsolutePath().toString();
        String json =
            "{" +
            "\"minecraftVersion\":\"" + jsonEscape(result.minecraftVersion()) + "\"," +
            "\"seed\":" + result.seed() + "," +
            "\"output\":\"" + jsonEscape(output) + "\"," +
            "\"spawnX\":" + result.spawnX() + "," +
            "\"spawnY\":" + result.spawnY() + "," +
            "\"spawnZ\":" + result.spawnZ() + "," +
            "\"centerChunkX\":" + result.centerChunkX() + "," +
            "\"centerChunkZ\":" + result.centerChunkZ() + "," +
            "\"radiusChunks\":" + result.radiusChunks() + "," +
            "\"chunks\":" + result.chunks() + "," +
            "\"regions\":" + result.regions() + "," +
            "\"qaMode\":" + result.qaMode() +
            "}\n";
        Files.writeString(absolute, json);
    }

    private static String jsonEscape(String value) {
        StringBuilder out = new StringBuilder(value.length() + 16);
        for (int i = 0; i < value.length(); i++) {
            char c = value.charAt(i);
            switch (c) {
                case '\\' -> out.append("\\\\");
                case '"' -> out.append("\\\"");
                case '\n' -> out.append("\\n");
                case '\r' -> out.append("\\r");
                case '\t' -> out.append("\\t");
                default -> {
                    if (c < 0x20) {
                        out.append(String.format("\\u%04x", (int) c));
                    } else {
                        out.append(c);
                    }
                }
            }
        }
        return out.toString();
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

    public record Result(
        String minecraftVersion,
        long seed,
        Path output,
        int spawnX,
        int spawnY,
        int spawnZ,
        int centerChunkX,
        int centerChunkZ,
        int radiusChunks,
        int chunks,
        int regions,
        boolean qaMode
    ) {}

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
        boolean force,
        boolean qaMode,
        Path resultJson
    ) {
        public Arguments(
            Path model,
            Path output,
            String levelName,
            int radiusChunks,
            boolean force
        ) {
            this(model, output, levelName, radiusChunks, force, true, null);
        }

        static Arguments parse(String[] args) {
            Path model = Path.of(
                "internal/goldenvectors/testdata/runtime-v1.geoworld"
            );
            Path output = Path.of("output/minecraft-26.3-preview/GeoWorld-Preview");
            String levelName = "GeoWorld 26.3 Preview";
            int radius = 8;
            boolean force = false;
            boolean qaMode = true;
            Path resultJson = null;

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
                    case "--qa" -> qaMode = true;
                    case "--survival" -> qaMode = false;
                    case "--result-json" ->
                        resultJson = Path.of(requireValue(args, ++i, "--result-json"));
                    case "--help", "-h" -> {
                        System.out.println(
                            "Usage: generate-preview-world.cmd [options]\n" +
                            "  --model <file.geoworld>\n" +
                            "  --output <world-directory>\n" +
                            "  --name <level-name>\n" +
                            "  --radius <0..31 chunks>\n" +
                            "  --qa (default: creative + cheats)\n" +
                            "  --survival\n" +
                            "  --result-json <file>\n" +
                            "  --force"
                        );
                        System.exit(0);
                    }
                    default -> throw new IllegalArgumentException(
                        "unknown argument: " + args[i]
                    );
                }
            }

            return new Arguments(
                model,
                output,
                levelName,
                radius,
                force,
                qaMode,
                resultJson
            );
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
