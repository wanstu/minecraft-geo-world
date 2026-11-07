package geoworld.runtime;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Set;

/**
 * Stable v1.1 acceptance registry. Acceptance IDs are API-like contracts: once
 * published they are never silently reused for a different requirement.
 */
public final class GeoWorldAcceptanceRegistry {
    public static final String VERSION = "geoworld-acceptance-v1.1";

    public enum RequirementKind {
        CORE,
        LATER,
        CONSTRAINT
    }

    public enum InspectionKind {
        AUTOMATIC,
        AUTOMATIC_AND_MANUAL,
        POINT,
        ROUTE,
        REGION,
        RELATION,
        UI,
        API,
        PERFORMANCE
    }

    public enum ResultStatus {
        PASS,
        FAIL,
        NOT_IMPLEMENTED,
        MANUAL_REQUIRED
    }

    public record Entry(
        String id,
        RequirementKind requirement,
        String name,
        InspectionKind inspection,
        String passStandard,
        String method
    ) {}

    public record AcceptancePoint(
        int x,
        int z,
        String expected,
        String failure
    ) {}

    public record Result(
        String id,
        ResultStatus status,
        String message,
        List<AcceptancePoint> points
    ) {
        public Result {
            Objects.requireNonNull(id, "id");
            Objects.requireNonNull(status, "status");
            message = message == null ? "" : message;
            points = points == null ? List.of() : List.copyOf(points);
        }
    }

    private static final List<Entry> ENTRIES = buildEntries();
    // Query acceptance supplements, rather than changes, the canonical 177 IDs.
    private static final List<Entry> QUERY_ENTRIES = buildQueryEntries();
    private static final Map<String, Entry> BY_ID = index(
        java.util.stream.Stream.concat(ENTRIES.stream(), QUERY_ENTRIES.stream()).toList()
    );

    private GeoWorldAcceptanceRegistry() {}

    public static List<Entry> entries() {
        return ENTRIES;
    }

    public static List<Entry> queryEntries() {
        return QUERY_ENTRIES;
    }

    public static Entry require(String id) {
        Entry entry = BY_ID.get(id);
        if (entry == null) {
            throw new IllegalArgumentException("unknown GeoWorld Acceptance ID: " + id);
        }
        return entry;
    }

    public static void validateComplete(List<Result> results) {
        Objects.requireNonNull(results, "results");
        Map<String, Result> byId = new HashMap<>();
        for (Result result : results) {
            require(result.id());
            Result previous = byId.put(result.id(), result);
            if (previous != null) {
                throw new IllegalArgumentException(
                    "duplicate GeoWorld acceptance result: " + result.id()
                );
            }
        }
        ArrayList<String> missing = new ArrayList<>();
        for (Entry entry : ENTRIES) {
            if (!byId.containsKey(entry.id())) {
                missing.add(entry.id());
            }
        }
        if (!missing.isEmpty()) {
            throw new IllegalArgumentException(
                "acceptance report omitted registered IDs: " + String.join(",", missing)
            );
        }
    }

    /**
     * NOT_IMPLEMENTED is explicit, and is only release-tolerable for entries
     * classified as LATER by the specification. Core/constraint omissions must
     * fail the release gate instead of disappearing from a report.
     */
    public static List<String> releaseBlockingIds(List<Result> results) {
        validateComplete(results);
        ArrayList<String> blocking = new ArrayList<>();
        for (Result result : results) {
            Entry entry = require(result.id());
            if (result.status() == ResultStatus.FAIL) {
                blocking.add(result.id());
            } else if (
                result.status() == ResultStatus.NOT_IMPLEMENTED &&
                entry.requirement() != RequirementKind.LATER
            ) {
                blocking.add(result.id());
            }
        }
        return List.copyOf(blocking);
    }

    private static Map<String, Entry> index(List<Entry> entries) {
        LinkedHashMap<String, Entry> out = new LinkedHashMap<>();
        for (Entry entry : entries) {
            if (out.put(entry.id(), entry) != null) {
                throw new ExceptionInInitializerError(
                    "duplicate GeoWorld Acceptance ID: " + entry.id()
                );
            }
        }
        return Map.copyOf(out);
    }

    private static List<Entry> buildQueryEntries() {
        String[][] rows = {
            {"QUERY-01", "Exact field registration", "NaturalField and FACT_REGISTRY contain the same unique field names"},
            {"QUERY-02", "String value domains", "Every sampled classification belongs to its documented domain"},
            {"QUERY-03", "Unavailable and finite values", "Values and unavailable fields partition the request without null or nonfinite values"},
            {"QUERY-04", "Numeric ranges", "Normalized scores, distances, angles and counts obey their documented bounds"},
            {"QUERY-05", "Regional identity", "Repeated nonempty region keys have identical count, area and centroid"},
            {"QUERY-06", "Coordinate quantization", "All offsets in one 1/4/16 block cell return the same result and sample coordinates"},
            {"QUERY-07", "Version identity", "Nonempty result versions match the persisted model manifest"},
            {"QUERY-08", "Outer model semantics", "Outer distances are unavailable except zero distance to ocean on ocean"},
            {"QUERY-09", "Negative score direction", "Near-river floodplain risk mean exceeds dry land mean with nonempty samples"},
            {"QUERY-10", "Facade equivalence", "Facade results, support flags and identity equal direct query results"}
        };
        ArrayList<Entry> entries = new ArrayList<>();
        for (String[] row : rows) {
            entries.add(new Entry(row[0], RequirementKind.CORE, row[1],
                InspectionKind.API, row[2], "GeoWorldModelVerifier"));
        }
        return List.copyOf(entries);
    }

    private static List<Entry> buildEntries() {
        InputStream resource = GeoWorldAcceptanceRegistry.class
            .getResourceAsStream("/geoworld/acceptance-v1.1.tsv");
        if (resource == null) {
            throw new ExceptionInInitializerError(
                "missing /geoworld/acceptance-v1.1.tsv"
            );
        }

        ArrayList<Entry> out = new ArrayList<>();
        try (
            BufferedReader reader = new BufferedReader(
                new InputStreamReader(resource, StandardCharsets.UTF_8)
            )
        ) {
            String line;
            while ((line = reader.readLine()) != null) {
                line = line.trim();
                if (line.isEmpty() || line.startsWith("#")) {
                    continue;
                }
                String[] parts = line.split("\\t", 6);
                if (parts.length != 6) {
                    throw new IllegalArgumentException(
                        "invalid acceptance registry row: " + line
                    );
                }
                out.add(new Entry(
                    parts[0],
                    RequirementKind.valueOf(parts[1]),
                    parts[3],
                    InspectionKind.valueOf(parts[2]),
                    parts[4],
                    parts[5]
                ));
            }
        } catch (IOException | RuntimeException error) {
            throw new ExceptionInInitializerError(error);
        }

        if (out.size() != 177) {
            throw new ExceptionInInitializerError(
                "acceptance registry entry count=" + out.size() +
                " expected=177"
            );
        }
        return List.copyOf(out);
    }

}
