package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner;
import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner.Body;
import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner.BodySource;
import geoworld.minecraft.common.worldgen.GeoWorldMiningDistrictPlanner;
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
    private static final List<String> VANILLA_FAMILIES = List.of(
        "coal", "iron", "copper", "gold",
        "lapis", "redstone", "diamond", "emerald"
    );

    private record PreviewBody(
        String mineral,
        String source,
        double centerX,
        double centerY,
        double centerZ,
        double length,
        double radius,
        double angle,
        double tilt,
        double bend,
        boolean large,
        boolean outcrop
    ) {}

    private record PreviewDistrict(
        long id,
        double centerX,
        double centerZ,
        double radius,
        String primary,
        String companion,
        int primaryLodes,
        int companionLodes
    ) {}

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
                "expected <model> <minX> <maxX> <minZ> <maxZ> <major|all> [limit] [oreFamiliesCsv]"
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
        MinecraftV26HeightMapping heightMapping =
            new MinecraftV26HeightMapping();
        GeoWorldMineralPlanner planner = new GeoWorldMineralPlanner(
            heightMapping
        );
        GeoWorldMiningDistrictPlanner districtPlanner =
            new GeoWorldMiningDistrictPlanner(heightMapping);
        List<String> oreFamilies = parseOreFamilies(
            args.length >= 8 ? args[7] : ""
        );
        boolean miningDistricts = ("geoworld-rules-v1.1.15".equals(
            model.manifest().generationRulesVersion()
        ) || "geoworld-rules-v1.1.16".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.18".equals(
            model.manifest().generationRulesVersion()
        ) || "geoworld-rules-v1.1.19".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.20".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.21".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.22".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.23".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.24".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.25".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.26".equals(
            model.manifest().generationRulesVersion()
        ) || "geoworld-rules-v1.1.27".equals(
            model.manifest().generationRulesVersion()
        ))))))))));

        List<PreviewBody> bodies = new ArrayList<>(limit);
        List<PreviewDistrict> districts = new ArrayList<>();
        int[] total = {0};
        long sampleSeed =
            model.manifest().seed() ^
            ((long) minX * 0x9E3779B97F4A7C15L) ^
            ((long) maxX * 0xC2B2AE3D27D4EB4FL) ^
            ((long) minZ * 0x165667B19E3779F9L) ^
            ((long) maxZ * 0x85EBCA77C2B2AE63L) ^
            (mode.equals("all") ? 0xA11D37A11D37L : 0x6D616A6F72L);
        SplittableRandom previewSampler = new SplittableRandom(sampleSeed);
        Consumer<PreviewBody> collect = body -> {
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

        Consumer<Body> collectLegacy = body -> {
            if (mode.equals("major") &&
                body.source() != BodySource.GUARANTEE &&
                body.source() != BodySource.OUTCROP &&
                !body.large()) {
                return;
            }
            collect.accept(fromBody(body));
        };

        if (miningDistricts) {
            districtPlanner.planDistricts(
                model, model.manifest().seed(),
                minX, maxX, minZ, maxZ,
                district -> districts.add(fromDistrict(model, district, oreFamilies))
            );
            districtPlanner.planLodes(
                model, model.manifest().seed(),
                minX, maxX, minZ, maxZ,
                lode -> collect.accept(fromDistrictLode(model, lode, oreFamilies))
            );
            // v1.1.15 keeps only a light background resource layer outside
            // districts. It is shown only in the detailed "all" mode.
            if (mode.equals("all")) {
                planner.planScatterBodies(
                    model, model.manifest().seed(),
                    minX, maxX, minZ, maxZ,
                    body -> collect.accept(fromBody(body))
                );
                planner.planSurfaceOutcropBodies(
                    model, model.manifest().seed(),
                    minX, maxX, minZ, maxZ,
                    body -> collect.accept(fromBody(body))
                );
            }
        } else {
            planner.planBodies(
                model, model.manifest().seed(),
                minX, maxX, minZ, maxZ, collectLegacy
            );
            planner.planSurfaceOutcropBodies(
                model, model.manifest().seed(),
                minX, maxX, minZ, maxZ, collectLegacy
            );
            if (mode.equals("all")) {
                planner.planScatterBodies(
                    model, model.manifest().seed(),
                    minX, maxX, minZ, maxZ, collectLegacy
                );
            }
        }
        bodies.sort(
            Comparator.comparingDouble(PreviewBody::centerZ)
                .thenComparingDouble(PreviewBody::centerX)
                .thenComparing(PreviewBody::mineral)
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
        appendStringArray(out, "families", oreFamilies).append(',');
        out.append("\"districts\":[");
        for (int i = 0; i < districts.size(); i++) {
            if (i > 0) out.append(',');
            appendDistrict(out, districts.get(i));
        }
        out.append("],");
        out.append("\"bodies\":[");

        for (int i = 0; i < bodies.size(); i++) {
            if (i > 0) out.append(',');
            appendBody(out, bodies.get(i), i);
        }
        out.append("]}");
        return out.toString();
    }

    private static void appendBody(StringBuilder out, PreviewBody body, int index) {
        double dx = Math.cos(body.angle());
        double dz = Math.sin(body.angle());
        double half = body.source().equals("SCATTER") ||
            body.source().startsWith("DISTRICT_")
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
        field(out, "mineral", body.mineral()).append(',');
        field(out, "source", body.source()).append(',');
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
        out.append("\"outcrop\":").append(body.outcrop());
        out.append('}');
    }

    private static PreviewDistrict fromDistrict(
        GeoWorldV1 model,
        GeoWorldMiningDistrictPlanner.District district,
        List<String> families
    ) {
        GeoWorldV1.MineralDistrictAssignment assignment =
            model.mineralDistrictAssignment(district.id());
        String primary;
        String companion;
        if (assignment != null) {
            primary = assignment.primary();
            companion = assignment.companion();
        } else {
            String primaryFallback =
                district.primary().name().toLowerCase(Locale.ROOT);
            String companionFallback =
                district.companion().name().toLowerCase(Locale.ROOT);
            primary = primaryFamily(
                district.id(),
                primaryFallback,
                families
            );
            companion = companionFamily(
                district.id(),
                companionFallback,
                families
            );
        }
        return new PreviewDistrict(
            district.id(),
            district.centerX(),
            district.centerZ(),
            district.radius(),
            primary,
            companion,
            district.primaryLodes(),
            district.companionLodes()
        );
    }

    private static void appendDistrict(
        StringBuilder out,
        PreviewDistrict district
    ) {
        out.append('{');
        field(out, "id", district.id()).append(',');
        field(out, "x", district.centerX()).append(',');
        field(out, "z", district.centerZ()).append(',');
        field(out, "radius", district.radius()).append(',');
        field(out, "primary", district.primary()).append(',');
        field(out, "companion", district.companion()).append(',');
        field(out, "primaryLodes", district.primaryLodes()).append(',');
        field(out, "companionLodes", district.companionLodes());
        out.append('}');
    }

    private static PreviewBody fromBody(Body body) {
        return new PreviewBody(
            body.mineral().name(),
            body.source().name(),
            body.centerX(), body.centerY(), body.centerZ(),
            body.length(), body.radius(), body.angle(),
            body.tilt(), body.bend(), body.large(),
            body.rockyOutcrop()
        );
    }

    private static PreviewBody fromDistrictLode(
        GeoWorldV1 model,
        GeoWorldMiningDistrictPlanner.Lode lode,
        List<String> families
    ) {
        GeoWorldV1.MineralDistrictAssignment assignment =
            model.mineralDistrictAssignment(lode.districtId());
        String family;
        if (assignment != null) {
            family = lode.companion()
                ? assignment.companion()
                : assignment.primary();
        } else {
            String fallback =
                lode.mineral().name().toLowerCase(Locale.ROOT);
            family = lode.companion()
                ? companionFamily(lode.districtId(), fallback, families)
                : primaryFamily(lode.districtId(), fallback, families);
        }
        return new PreviewBody(
            family,
            lode.companion() ? "DISTRICT_COMPANION" : "DISTRICT_PRIMARY",
            lode.centerX(), lode.centerY(), lode.centerZ(),
            lode.length(), lode.radius(), lode.angle(),
            lode.tilt(), lode.bend(), true, false
        );
    }

    private static List<String> parseOreFamilies(String csv) {
        List<String> families = new ArrayList<>();
        if (csv != null && !csv.isBlank()) {
            for (String part : csv.split(",")) {
                String value = part.trim().toLowerCase(Locale.ROOT);
                if (!value.isBlank() && !families.contains(value)) {
                    families.add(value);
                }
            }
        }
        if (families.isEmpty()) {
            families.addAll(VANILLA_FAMILIES);
        }
        families.sort(String::compareTo);
        return List.copyOf(families);
    }

    private static String primaryFamily(
        long districtId,
        String fallback,
        List<String> families
    ) {
        List<String> extras = extraFamilies(families);
        if (usesExtraPrimary(districtId, extras)) {
            return extras.get(Math.floorMod(
                mix32(districtId ^ 0x5052494D415259L),
                extras.size()
            ));
        }
        if (families.contains(fallback)) return fallback;
        return families.get(0);
    }

    private static String companionFamily(
        long districtId,
        String fallback,
        List<String> families
    ) {
        List<String> extras = extraFamilies(families);
        if (usesExtraPrimary(districtId, extras)) {
            if (families.contains(fallback)) return fallback;
            return families.get(0);
        }
        if (!extras.isEmpty()) {
            return extras.get(Math.floorMod(
                mix32(districtId ^ 0x434F4D50414E494FL),
                extras.size()
            ));
        }
        if (families.contains(fallback)) return fallback;
        return families.get(0);
    }

    private static List<String> extraFamilies(List<String> families) {
        return families.stream()
            .filter(name -> !VANILLA_FAMILIES.contains(name))
            .toList();
    }

    private static boolean usesExtraPrimary(
        long districtId,
        List<String> extras
    ) {
        return !extras.isEmpty() &&
            Math.floorMod(
                mix32(districtId ^ 0x4558545241505249L),
                100
            ) < 32;
    }

    private static int mix32(long value) {
        long h = value;
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        h ^= h >>> 31;
        return (int)(h ^ (h >>> 32));
    }

    private static StringBuilder appendStringArray(
        StringBuilder out,
        String name,
        List<String> values
    ) {
        out.append('"').append(name).append("\":[");
        for (int i = 0; i < values.size(); i++) {
            if (i > 0) out.append(',');
            out.append('"').append(escape(values.get(i))).append('"');
        }
        return out.append(']');
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
