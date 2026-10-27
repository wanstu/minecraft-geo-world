package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner;
import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner.Body;
import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner.BodySource;
import geoworld.minecraft.v26common.MinecraftV26HeightMapping;
import geoworld.runtime.GeoWorldV1;

import java.io.BufferedReader;
import java.io.InputStreamReader;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.Locale;
import java.util.SplittableRandom;
import java.util.function.Consumer;

/**
 * Lightweight bridge used by the embedded Web tool. It exposes the exact
 * deterministic mineral bodies consumed by Minecraft without booting the game.
 */
public final class GeoWorldV26MineralPreviewCli {
    private GeoWorldV26MineralPreviewCli() {}

    public static void main(String[] args) throws Exception {
        if (args.length == 1 && "--server".equals(args[0])) {
            serve();
            return;
        }
        System.out.println(render(args, new ModelCache()));
    }

    private static void serve() throws Exception {
        ModelCache cache = new ModelCache();
        try (
            BufferedReader input = new BufferedReader(
                new InputStreamReader(System.in, StandardCharsets.UTF_8)
            );
            PrintWriter output = new PrintWriter(
                System.out, true, StandardCharsets.UTF_8
            )
        ) {
            String line;
            while ((line = input.readLine()) != null) {
                if ("QUIT".equals(line)) return;
                try {
                    String[] fields = line.split("\\t", -1);
                    output.println(render(fields, cache));
                } catch (Throwable error) {
                    output.println(
                        "{\"error\":\"" +
                        escape(error.getClass().getSimpleName() + ": " + error.getMessage()) +
                        "\"}"
                    );
                }
            }
        }
    }

    private static String render(String[] args, ModelCache cache) throws Exception {
        if (args.length < 6) {
            throw new IllegalArgumentException(
                "expected <model> <minX> <maxX> <minZ> <maxZ> <major|all> [limit]"
            );
        }

        Path modelPath = Path.of(args[0]).toAbsolutePath().normalize();
        int minX = Integer.parseInt(args[1]);
        int maxX = Integer.parseInt(args[2]);
        int minZ = Integer.parseInt(args[3]);
        int maxZ = Integer.parseInt(args[4]);
        String mode = args[5].toLowerCase(Locale.ROOT);
        int requestedLimit = args.length >= 7 ? Integer.parseInt(args[6]) : 2500;
        final int limit = Math.max(1, Math.min(10000, requestedLimit));

        if (minX > maxX || minZ > maxZ) {
            throw new IllegalArgumentException("invalid viewport bounds");
        }
        if (!mode.equals("major") && !mode.equals("all")) {
            throw new IllegalArgumentException("mode must be major or all");
        }

        GeoWorldV1 model = cache.load(modelPath);
        GeoWorldMineralPlanner planner = new GeoWorldMineralPlanner(
            new MinecraftV26HeightMapping()
        );

        List<Body> bodies = new ArrayList<>(limit);
        int[] total = {0};
        long sampleSeed =
            model.manifest().seed() ^
            ((long) minX * 0x9E3779B97F4A7C15L) ^
            ((long) maxX * 0xC2B2AE3D27D4EB4FL) ^
            ((long) minZ * 0x165667B19E3779F9L) ^
            ((long) maxZ * 0x85EBCA77C2B2AE63L) ^
            (mode.equals("all") ? 0xA11D37A11D37L : 0x6D616A6F72L);
        SplittableRandom previewSampler = new SplittableRandom(sampleSeed);
        Consumer<Body> collect = body -> {
            if (mode.equals("major") &&
                body.source() != BodySource.GUARANTEE &&
                body.source() != BodySource.OUTCROP &&
                !body.large()) {
                return;
            }
            int seen = ++total[0];
            if (bodies.size() < limit) {
                bodies.add(body);
                return;
            }
            int replacement = previewSampler.nextInt(seen);
            if (replacement < limit) {
                bodies.set(replacement, body);
            }
        };

        planner.planBodies(
            model, model.manifest().seed(),
            minX, maxX, minZ, maxZ, collect
        );
        planner.planSurfaceOutcropBodies(
            model, model.manifest().seed(),
            minX, maxX, minZ, maxZ, collect
        );
        if (mode.equals("all")) {
            planner.planScatterBodies(
                model, model.manifest().seed(),
                minX, maxX, minZ, maxZ, collect
            );
        }
        bodies.sort(
            Comparator.comparingDouble(Body::centerZ)
                .thenComparingDouble(Body::centerX)
                .thenComparing(body -> body.mineral().ordinal())
        );

        StringBuilder out = new StringBuilder(
            Math.max(1024, bodies.size() * 320)
        );
        out.append('{');
        field(out, "seed", model.manifest().seed()).append(',');
        field(out, "rules", model.manifest().generationRulesVersion()).append(',');
        field(out, "mode", mode).append(',');
        out.append("\"bounds\":{");
        field(out, "minX", minX).append(',');
        field(out, "maxX", maxX).append(',');
        field(out, "minZ", minZ).append(',');
        field(out, "maxZ", maxZ).append("},");
        field(out, "total", total[0]).append(',');
        field(out, "returned", bodies.size()).append(',');
        out.append("\"truncated\":").append(total[0] > bodies.size()).append(',');
        out.append("\"bodies\":[");

        for (int i = 0; i < bodies.size(); i++) {
            if (i > 0) out.append(',');
            appendBody(out, bodies.get(i), i);
        }
        out.append("]}");
        return out.toString();
    }

    private static void appendBody(StringBuilder out, Body body, int index) {
        double dx = Math.cos(body.angle());
        double dz = Math.sin(body.angle());
        double half = body.source() == BodySource.SCATTER
            ? body.length() * 0.5
            : body.length() * 0.34;
        double startX = body.centerX() - half * dx;
        double startZ = body.centerZ() - half * dz;
        double endX = body.centerX() + half * dx;
        double endZ = body.centerZ() + half * dz;
        double midX = body.centerX() - body.bend() * dz;
        double midZ = body.centerZ() + body.bend() * dx;

        out.append('{');
        field(out, "id", index).append(',');
        field(out, "mineral", body.mineral().name()).append(',');
        field(out, "source", body.source().name()).append(',');
        field(out, "x", body.centerX()).append(',');
        field(out, "y", body.centerY()).append(',');
        field(out, "z", body.centerZ()).append(',');
        field(out, "length", body.length()).append(',');
        field(out, "radius", body.radius()).append(',');
        field(out, "angle", body.angle()).append(',');
        field(out, "tilt", body.tilt()).append(',');
        field(out, "bend", body.bend()).append(',');
        field(out, "startX", startX).append(',');
        field(out, "startZ", startZ).append(',');
        field(out, "midX", midX).append(',');
        field(out, "midZ", midZ).append(',');
        field(out, "endX", endX).append(',');
        field(out, "endZ", endZ).append(',');
        out.append("\"large\":").append(body.large()).append(',');
        out.append("\"outcrop\":").append(body.rockyOutcrop());
        out.append('}');
    }

    private static StringBuilder field(
        StringBuilder out, String name, String value
    ) {
        return out.append('"').append(name).append("\":\"")
            .append(escape(value)).append('"');
    }

    private static StringBuilder field(
        StringBuilder out, String name, long value
    ) {
        return out.append('"').append(name).append("\":").append(value);
    }

    private static StringBuilder field(
        StringBuilder out, String name, double value
    ) {
        return out.append('"').append(name).append("\":")
            .append(Double.toString(value));
    }

    private static String escape(String value) {
        if (value == null) return "";
        return value
            .replace("\\", "\\\\")
            .replace("\"", "\\\"")
            .replace("\r", "\\r")
            .replace("\n", "\\n");
    }

    private static final class ModelCache {
        private Path path;
        private long size = -1;
        private long modifiedMillis = -1;
        private GeoWorldV1 model;

        GeoWorldV1 load(Path requested) throws Exception {
            long requestedSize = Files.size(requested);
            long requestedModifiedMillis =
                Files.getLastModifiedTime(requested).toMillis();
            if (
                model == null ||
                !requested.equals(path) ||
                requestedSize != size ||
                requestedModifiedMillis != modifiedMillis
            ) {
                model = GeoWorldV1.load(requested);
                path = requested;
                size = requestedSize;
                modifiedMillis = requestedModifiedMillis;
            }
            return model;
        }
    }
}
