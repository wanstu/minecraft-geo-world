package geoworld.runtime;

import geoworld.runtime.GeoWorldQueryService.NaturalField;
import geoworld.runtime.GeoWorldQueryService.NaturalResult;
import geoworld.runtime.GeoWorldQueryService.QueryRequest;
import geoworld.runtime.GeoWorldQueryService.Resolution;
import geoworld.runtime.api.GeoWorldFacade;

import java.io.IOException;
import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Base64;
import java.util.HexFormat;
import java.security.MessageDigest;
import java.util.zip.GZIPInputStream;
import java.util.zip.GZIPOutputStream;
import java.util.EnumMap;
import java.util.EnumSet;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.Set;
import java.util.regex.Pattern;

/** Black-box checks shared by GeoWorldModelVerifier; never changes the model. */
final class GeoWorldQueryContractVerifier {
    static final List<String> IDS = List.of(
        "QUERY-01", "QUERY-02", "QUERY-03", "QUERY-04", "QUERY-05",
        "QUERY-06", "QUERY-07", "QUERY-08", "QUERY-09", "QUERY-10"
    );
    private record Point(double x, double z) {}
    private static final Set<NaturalField> REGION = EnumSet.of(
        NaturalField.REGIONAL_REGION_KEY, NaturalField.REGIONAL_REGION_CELL_COUNT,
        NaturalField.REGIONAL_REGION_AREA_BLOCKS2,
        NaturalField.REGIONAL_REGION_CENTROID_X, NaturalField.REGIONAL_REGION_CENTROID_Z
    );
    private static final Set<NaturalField> NORMALIZED = EnumSet.of(
        NaturalField.ELEVATION, NaturalField.WATER_SURFACE_ELEVATION,
        NaturalField.WATER_DEPTH, NaturalField.OCEAN_ELIGIBILITY,
        NaturalField.RIVER_WEIGHT, NaturalField.MONSOON_INDEX,
        NaturalField.SPAWN_SUITABILITY, NaturalField.SETTLEMENT_BUILDABILITY,
        NaturalField.FRESH_WATER_ACCESS, NaturalField.FLOOD_RISK,
        NaturalField.AGRICULTURE_SUITABILITY, NaturalField.CLIMATE_COMFORT,
        NaturalField.TRANSPORT_ACCESS, NaturalField.RESOURCE_ACCESS,
        NaturalField.SETTLEMENT_SUITABILITY, NaturalField.WATERWAY_NAVIGABILITY,
        NaturalField.PORT_SUITABILITY, NaturalField.NAVIGATION_BOTTLENECK_RISK,
        NaturalField.MINING_SUITABILITY, NaturalField.SURFACE_MINING_SUITABILITY,
        NaturalField.SHALLOW_MINING_SUITABILITY, NaturalField.DEEP_MINING_SUITABILITY,
        NaturalField.INDUSTRIAL_MINERAL_VALUE, NaturalField.MINERAL_RESOURCE_DIVERSITY,
        NaturalField.CIVILIZATION_AGRICULTURAL_SETTLEMENT_SUITABILITY,
        NaturalField.CIVILIZATION_PORT_CITY_SUITABILITY,
        NaturalField.CIVILIZATION_MINING_TOWN_SUITABILITY,
        NaturalField.CIVILIZATION_TRANSPORT_NODE_SUITABILITY,
        NaturalField.CIVILIZATION_NODE_SUITABILITY,
        NaturalField.CIVILIZATION_SPECIALIZATION_STRENGTH,
        NaturalField.REGIONAL_BUILDING_ADVANTAGE, NaturalField.REGIONAL_AGRICULTURE_ADVANTAGE,
        NaturalField.REGIONAL_PORT_ADVANTAGE, NaturalField.REGIONAL_MINING_ADVANTAGE,
        NaturalField.REGIONAL_TRANSPORT_ADVANTAGE, NaturalField.REGIONAL_SPECIALIZATION_STRENGTH,
        NaturalField.REGIONAL_ADVANTAGE_OVERLAP_RISK, NaturalField.NATURAL_CORRIDOR_SUITABILITY,
        NaturalField.RIVER_VALLEY_CORRIDOR, NaturalField.HILL_PASS_CORRIDOR,
        NaturalField.MOUNTAIN_PASS_CORRIDOR, NaturalField.PLATEAU_ACCESS_CORRIDOR,
        NaturalField.BASIN_GATEWAY_CORRIDOR, NaturalField.LOW_DIVIDE_CORRIDOR,
        NaturalField.STRAIT_ISTHMUS_POTENTIAL, NaturalField.TRANSPORT_CHOKEPOINT_STRENGTH
    );
    private static final Map<NaturalField, Set<String>> DOMAINS = domains();
    private final Path path;
    private final Map<String, List<String>> failures;

    private GeoWorldQueryContractVerifier(Path path, Map<String, List<String>> failures) {
        this.path = path;
        this.failures = failures;
    }

    static void verify(Path path, Path registry, Map<String, List<String>> failures) {
        var verifier = new GeoWorldQueryContractVerifier(path, failures);
        try {
            verifier.verifyRegistry(registry);
            verifier.verifySamples(GeoWorldV1.load(path));
        } catch (Exception | AssertionError failure) {
            // No unexecuted check may be reported as PASS after a setup failure.
            for (String id : IDS) {
                verifier.check(id, false, 0, 0, "setup", failure, "successful verifier setup");
            }
        }
    }

    private void verifyRegistry(Path registry) throws IOException {
        String text = Files.readString(registry, StandardCharsets.UTF_8);
        var matcher = Pattern.compile("(?m)^\\|\\s*`([A-Z][A-Z0-9_]+)`\\s*\\|").matcher(text);
        Set<String> registered = new HashSet<>();
        int rows = 0;
        while (matcher.find()) {
            rows++;
            registered.add(matcher.group(1));
        }
        Set<String> expected = new HashSet<>();
        for (NaturalField field : NaturalField.values()) expected.add(field.name());
        Set<String> missing = new HashSet<>(expected);
        missing.removeAll(registered);
        Set<String> extra = new HashSet<>(registered);
        extra.removeAll(expected);
        check("QUERY-01", registered.equals(expected) && rows == expected.size(),
            0, 0, "NaturalField", "rows=" + rows + ", missing=" + missing + ", extra=" + extra,
            expected.size() + " unique exact field rows");
        for (String id : IDS) GeoWorldAcceptanceRegistry.require(id);
    }

    private void verifySamples(GeoWorldV1 model) throws Exception {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        List<Point> points = samplePoints(model, path);
        Map<String, Map<NaturalField, Object>> regions = new HashMap<>();
        Map<String, Integer> regionCounts = new HashMap<>();
        EnumSet<NaturalField> observed = EnumSet.noneOf(NaturalField.class);
        double landRisk = 0, floodRisk = 0;
        int landCount = 0, floodCount = 0, outerCount = 0;
        Point lastFlood = new Point(0, 0);
        Map<Integer, Point> nearRiverNodes = new java.util.LinkedHashMap<>();
        for (Point point : points) {
            QueryRequest request = QueryRequest.all(point.x, point.z, Resolution.BLOCK_4);
            NaturalResult result = service.queryNatural(request);
            observed.addAll(result.values().keySet());
            verifyResult(model, request, result);
            verifyQuantization(service, point);
            NaturalResult regional = service.queryNatural(
                new QueryRequest(point.x, point.z, Resolution.BLOCK_16, REGION));
            String key = (String) regional.values().get(NaturalField.REGIONAL_REGION_KEY);
            if (key != null && !"none".equals(key)) {
                Map<NaturalField, Object> previous = regions.putIfAbsent(key, regional.values());
                regionCounts.merge(key, 1, Integer::sum);
                if (previous != null) {
                    for (NaturalField field : REGION) {
                        check("QUERY-05", Objects.equals(previous.get(field), regional.values().get(field)),
                            point.x, point.z, field, regional.values().get(field), previous.get(field));
                    }
                }
            }
            var g = model.manifest().geometry();
            boolean outside = result.sampleX() < g.originX() || result.sampleZ() < g.originZ() ||
                result.sampleX() > g.originX() + g.spanXBlocks() ||
                result.sampleZ() > g.originZ() + g.spanZBlocks();
            if (outside) {
                outerCount++;
                verifyOutside(result);
            } else if ("None".equals(result.values().get(NaturalField.WATER_KIND))) {
                double risk = number(result, NaturalField.FLOOD_RISK);
                landRisk += risk;
                landCount++;
                if (result.values().get(NaturalField.DISTANCE_TO_RIVER) instanceof Number d &&
                    d.doubleValue() <= 256 && nearRiverNodes.size() < 2) {
                    int node = model.sample(result.sampleX(), result.sampleZ()).sourceNode();
                    if (node >= 0) nearRiverNodes.putIfAbsent(node, point);
                }
                Object terrain = result.values().get(NaturalField.TERRAIN_VARIANT);
                if (("Floodplain".equals(terrain) || "Alluvial Plain".equals(terrain)) &&
                    result.values().get(NaturalField.DISTANCE_TO_RIVER) instanceof Number distance &&
                    distance.doubleValue() <= 256) {
                    floodRisk += risk;
                    floodCount++;
                    lastFlood = point;
                }
            }
        }
        long repeatedRegions = regionCounts.values().stream().filter(count -> count >= 2).count();
        check("QUERY-05", repeatedRegions > 0, 0, 0, NaturalField.REGIONAL_REGION_KEY,
            regionCounts, "at least one region sampled twice");
        check("QUERY-08", outerCount > 0, 0, 0, "outside samples", outerCount, ">0");
        if (floodCount > 0) {
            check("QUERY-09", landCount > 0 && floodRisk / floodCount > landRisk / landCount,
                lastFlood.x, lastFlood.z, NaturalField.FLOOD_RISK,
                "floodMean=" + floodRisk / floodCount + " count=" + floodCount +
                    ", landMean=" + landRisk / landCount + " count=" + landCount,
                "near-river floodplain mean > dry land mean, both nonempty");
        } else {
            // The six legacy goldens contain no Floodplain/Alluvial Plain
            // nodes. Verify the query contract with controlled persisted input,
            // without changing a golden or claiming generator coverage.
            verifyNegativeScoreFixture(model, nearRiverNodes, landRisk / landCount);
        }
        verifyFacade(model, service, points);
        System.out.printf("Query samples: model=%s points=%d availableFields=%d repeatedRegions=%d floodplain=%d land=%d%n",
            path.getFileName(), points.size(), observed.size(), repeatedRegions, floodCount, landCount);
    }

    private void verifyFacade(GeoWorldV1 model, GeoWorldQueryService direct, List<Point> points) {
        Object owner = new Object();
        GeoWorldFactsBinding.bind(owner, model);
        try {
            var facts = GeoWorldFacade.currentWorld().orElseThrow();
            for (Point point : points) {
                QueryRequest request = QueryRequest.all(point.x, point.z, Resolution.BLOCK_4);
                NaturalResult actual = facts.queryNatural(request);
                check("QUERY-10", direct.queryNatural(request).equals(actual), point.x, point.z,
                    "facade result", actual, "equals direct GeoWorldQueryService result");
            }
            var g = model.manifest().geometry();
            NaturalResult center = direct.queryNatural(QueryRequest.all(
                g.originX() + g.spanXBlocks() * 0.5, g.originZ() + g.spanZBlocks() * 0.5, Resolution.BLOCK_16));
            Set<NaturalField> conditional = EnumSet.of(
                NaturalField.WATER_SURFACE_ELEVATION, NaturalField.WATERSHED_ID,
                NaturalField.NATURAL_FLOW_DIRECTION, NaturalField.RIVER_ID,
                NaturalField.DISTANCE_TO_RIVER, NaturalField.DISTANCE_TO_LAKE,
                NaturalField.DISTANCE_TO_COAST, NaturalField.DISTANCE_TO_OCEAN,
                NaturalField.NEAREST_RIVER_HALF_WIDTH, NaturalField.NEAREST_RIVER_GRADIENT
            );
            for (NaturalField field : NaturalField.values()) {
                boolean expected = !center.unavailableFields().contains(field) || conditional.contains(field);
                check("QUERY-10", facts.supports(field) == expected, center.sampleX(), center.sampleZ(),
                    "supports(" + field + ")", facts.supports(field), expected);
            }
            check("QUERY-10", facts.worldProfileVersion().equals(center.worldProfileVersion()) &&
                facts.generationRulesVersion().equals(center.generationRulesVersion()) &&
                facts.worldModelVersion().equals(center.worldModelVersion()), center.sampleX(), center.sampleZ(),
                "facade versions", facts.worldProfileVersion() + "/" + facts.generationRulesVersion() + "/" + facts.worldModelVersion(),
                "direct query identity");
        } finally { GeoWorldFactsBinding.unbind(owner); }
        check("QUERY-10", GeoWorldFacade.currentWorld().isEmpty(), 0, 0,
            "currentWorld after verification", GeoWorldFacade.currentWorld(), "empty");
    }

    private void verifyNegativeScoreFixture(
        GeoWorldV1 source, Map<Integer, Point> nodes, double landMean
    ) throws Exception {
        check("QUERY-09", nodes.size() == 2 && Double.isFinite(landMean), 0, 0,
            "negative-score fixture", nodes.size(), "two distinct dry near-river nodes");
        if (nodes.size() != 2) return;
        byte[] original = Files.readAllBytes(path);
        String envelope;
        try (var gzip = new GZIPInputStream(new ByteArrayInputStream(original))) {
            envelope = new String(gzip.readAllBytes(), StandardCharsets.UTF_8);
        }
        Json.Root parsed = Json.parseEnvelope(envelope);
        String payload = parsed.rawPayload();
        var object = Json.object(Json.required(parsed.object(), "payload"), "payload");
        var profile = Json.object(Json.required(object, "profile"), "profile");
        String encoded = Json.string(profile, "terrainVariant");
        byte[] terrain = Base64.getDecoder().decode(encoded);
        List<Map.Entry<Integer, Point>> selected = new ArrayList<>(nodes.entrySet());
        for (int i = 0; i < selected.size(); i++) terrain[selected.get(i).getKey()] = (byte)(8 + i);
        String replacement = Base64.getEncoder().encodeToString(terrain);
        var pattern = Pattern.compile("(\"terrainVariant\"\\s*:\\s*\")" + Pattern.quote(encoded) + "(\")");
        var matcher = pattern.matcher(payload);
        if (!matcher.find()) throw new AssertionError("cannot locate fixture terrain profile");
        String changedPayload = matcher.replaceFirst("$1" + java.util.regex.Matcher.quoteReplacement(replacement) + "$2");
        String checksum = HexFormat.of().formatHex(MessageDigest.getInstance("SHA-256")
            .digest(changedPayload.getBytes(StandardCharsets.UTF_8)));
        // Base64 has constant length for the same byte array, so payloadBytes
        // and all other persisted identity remain unchanged.
        String changedEnvelope = envelope.replace(payload, changedPayload)
            .replace(source.manifest().payloadSha256(), checksum);
        Path fixture = Files.createTempFile("geoworld-query-negative-", ".geoworld");
        try {
            try (var output = new GZIPOutputStream(Files.newOutputStream(fixture))) {
                output.write(changedEnvelope.getBytes(StandardCharsets.UTF_8));
            }
            GeoWorldV1 controlled = GeoWorldV1.load(fixture);
            GeoWorldQueryService query = new GeoWorldQueryService(controlled);
            double sum = 0;
            for (int i = 0; i < selected.size(); i++) {
                Point point = selected.get(i).getValue();
                NaturalResult result = query.queryNatural(QueryRequest.all(point.x, point.z, Resolution.BLOCK_4));
                String expected = i == 0 ? "Floodplain" : "Alluvial Plain";
                check("QUERY-09", expected.equals(result.values().get(NaturalField.TERRAIN_VARIANT)) &&
                    "None".equals(result.values().get(NaturalField.WATER_KIND)) &&
                    number(result, NaturalField.DISTANCE_TO_RIVER) <= 256,
                    point.x, point.z, "controlled terrain/water/distance", result.values(),
                    "dry " + expected + " within 256 blocks of river");
                sum += number(result, NaturalField.FLOOD_RISK);
            }
            double mean = sum / selected.size();
            Point point = selected.get(0).getValue();
            check("QUERY-09", Double.isFinite(mean) && mean > landMean,
                point.x, point.z, NaturalField.FLOOD_RISK, mean, "> original dry-land mean " + landMean);
            System.out.printf("Query negative-score fixture: model=%s controlledPoints=2 mean=%f dryLandMean=%f%n",
                path.getFileName(), mean, landMean);
        } finally {
            Files.deleteIfExists(fixture);
        }
        check("QUERY-09", Arrays.equals(original, Files.readAllBytes(path)), 0, 0,
            "source fixture bytes", path.getFileName(), "unchanged committed golden");
    }

    private void verifyResult(GeoWorldV1 model, QueryRequest request, NaturalResult result) {
        double x = request.worldX(), z = request.worldZ();
        Set<NaturalField> union = new HashSet<>(result.values().keySet());
        Set<NaturalField> intersection = new HashSet<>(union);
        intersection.retainAll(result.unavailableFields());
        union.addAll(result.unavailableFields());
        check("QUERY-03", intersection.isEmpty() && union.equals(request.fields()), x, z,
            "values/unavailableFields", result, "disjoint partition of request fields");
        for (var entry : result.values().entrySet()) {
            NaturalField field = entry.getKey();
            Object value = entry.getValue();
            check("QUERY-03", finiteNonNull(value), x, z, field, value, "non-null recursively finite value");
            if (DOMAINS.containsKey(field)) {
                check("QUERY-02", DOMAINS.get(field).contains(value), x, z, field, value, DOMAINS.get(field));
            } else if (field == NaturalField.GEOLOGICAL_REGION) {
                check("QUERY-02", value instanceof String s && s.matches(
                    "^(oceanic-crust|orogenic-belt|uplifted-block|exposed-crust|sedimentary-basin|continental-interior):[0-9a-f]+$"),
                    x, z, field, value, "geological-family:hex");
            } else if (field == NaturalField.ORE_PROVINCE) {
                check("QUERY-02", value instanceof String s && s.matches("^ore-[0-7]-[0-9a-f]+$"),
                    x, z, field, value, "ore-[0-7]-hex");
            } else if (field == NaturalField.DOMINANT_MINERAL_RESOURCE) {
                check("QUERY-02", "none".equals(value) || Arrays.stream(GeoWorldMineralAffinity.MINERALS)
                    .map(name -> name.toLowerCase(Locale.ROOT)).anyMatch(name -> name.equals(value)),
                    x, z, field, value, "none or a query mineral family");
            }
            if (NORMALIZED.contains(field)) range(result, field, 0, 1);
            if (field.name().startsWith("DISTANCE_TO_") || field == NaturalField.ANNUAL_PRECIP_MM ||
                field == NaturalField.RIVER_ORDER || field == NaturalField.SLOPE ||
                field == NaturalField.NEAREST_RIVER_HALF_WIDTH || field == NaturalField.NEAREST_RIVER_GRADIENT ||
                field == NaturalField.FLOW_ACCUMULATION) range(result, field, 0, Double.MAX_VALUE);
            if (field == NaturalField.ANNUAL_MEAN_TEMP_C) range(result, field, -60, 60);
            if (field == NaturalField.REGIONAL_HIGH_ADVANTAGE_COUNT) range(result, field, 0, 5);
            if (field == NaturalField.ASPECT_DEGREES) {
                double degrees = number(result, field);
                check("QUERY-04", degrees >= 0 && degrees < 360, x, z, field, value, "[0,360)");
            }
            if (field == NaturalField.MINERAL_RESOURCE_PROFILE && value instanceof Map<?, ?> profile) {
                for (var mineral : profile.entrySet()) {
                    check("QUERY-04", mineral.getValue() instanceof Number n && n.doubleValue() >= 0 && n.doubleValue() <= 1,
                        x, z, field + "." + mineral.getKey(), mineral.getValue(), "[0,1]");
                }
            }
        }
        var manifest = model.manifest();
        check("QUERY-07", GeoWorldQueryService.API_VERSION.equals(result.apiVersion()) &&
            notBlank(result.worldProfileVersion()) && notBlank(result.generationRulesVersion()) &&
            notBlank(result.worldModelVersion()) &&
            Objects.equals(manifest.worldProfileVersion(), result.worldProfileVersion()) &&
            Objects.equals(manifest.generationRulesVersion(), result.generationRulesVersion()) &&
            Objects.equals(manifest.worldModelVersion(), result.worldModelVersion()), x, z,
            "version identity", result.apiVersion() + "/" + result.worldProfileVersion() + "/" +
                result.generationRulesVersion() + "/" + result.worldModelVersion(), "nonempty manifest identity");
    }

    private void verifyQuantization(GeoWorldQueryService service, Point point) {
        for (Resolution resolution : Resolution.values()) {
            int blocks = resolution.blocks();
            double cellX = Math.floor(point.x / blocks) * blocks;
            double cellZ = Math.floor(point.z / blocks) * blocks;
            NaturalResult first = service.queryNatural(QueryRequest.all(
                cellX + 0.125, cellZ + 0.125, resolution));
            NaturalResult second = service.queryNatural(QueryRequest.all(
                cellX + blocks - 0.125, cellZ + blocks - 0.125, resolution));
            double expectedX = cellX + (blocks == 1 ? 0 : blocks * 0.5);
            double expectedZ = cellZ + (blocks == 1 ? 0 : blocks * 0.5);
            check("QUERY-06", first.equals(second) && first.sampleX() == expectedX && first.sampleZ() == expectedZ,
                point.x, point.z, "sampleX/sampleZ/" + resolution,
                first.sampleX() + "," + first.sampleZ() + ", equal=" + first.equals(second),
                expectedX + "," + expectedZ + ", equal=true");
        }
    }

    private void verifyOutside(NaturalResult result) {
        for (NaturalField field : List.of(NaturalField.RIVER_ID, NaturalField.DISTANCE_TO_RIVER,
            NaturalField.DISTANCE_TO_LAKE, NaturalField.DISTANCE_TO_COAST)) {
            check("QUERY-08", result.unavailableFields().contains(field) && !result.values().containsKey(field),
                result.sampleX(), result.sampleZ(), field, result.values().get(field), "unavailable outside model");
        }
        boolean ocean = "Ocean".equals(result.values().get(NaturalField.WATER_KIND));
        check("QUERY-08", ocean ? Double.valueOf(0).equals(result.values().get(NaturalField.DISTANCE_TO_OCEAN)) :
            result.unavailableFields().contains(NaturalField.DISTANCE_TO_OCEAN),
            result.sampleX(), result.sampleZ(), NaturalField.DISTANCE_TO_OCEAN,
            result.values().get(NaturalField.DISTANCE_TO_OCEAN), ocean ? "0 on ocean" : "unavailable on outer island");
    }

    static List<double[]> coordinates(GeoWorldV1 model, Path path) throws IOException {
        return samplePoints(model, path).stream().map(p -> new double[] {p.x, p.z}).toList();
    }

    private static List<Point> samplePoints(GeoWorldV1 model, Path path) throws IOException {
        List<Point> points = new ArrayList<>();
        var g = model.manifest().geometry();
        for (int z = 0; z < 32; z++) for (int x = 0; x < 32; x++) {
            points.add(new Point(g.originX() + g.spanXBlocks() * x / 31.0,
                g.originZ() + g.spanZBlocks() * z / 31.0));
        }
        String filename = path.getFileName().toString();
        Path golden = path.resolveSibling(filename.substring(0, filename.length() - ".geoworld".length()) + ".json");
        var document = Json.object(Json.parse(Files.readString(golden, StandardCharsets.UTF_8)), "golden");
        for (Object entry : Json.array(Json.required(document, "cases"), "golden.cases")) {
            var point = Json.object(entry, "golden point");
            points.add(new Point(Json.doubleValue(point, "x"), Json.doubleValue(point, "z")));
        }
        return points;
    }

    private void range(NaturalResult result, NaturalField field, double min, double max) {
        double value = number(result, field);
        check("QUERY-04", value >= min && value <= max, result.sampleX(), result.sampleZ(), field,
            result.values().get(field), "[" + min + "," + max + "]");
    }

    private void check(String id, boolean condition, double x, double z, Object field, Object actual, Object expected) {
        List<String> errors = failures.get(id);
        if (!condition && errors.size() < 12) errors.add(path.getFileName() + " (" + x + "," + z +
            ") field=" + field + " actual=" + actual + " expected=" + expected);
    }

    private static double number(NaturalResult result, NaturalField field) {
        Object value = result.values().get(field);
        return value instanceof Number n ? n.doubleValue() : Double.NaN;
    }
    private static boolean notBlank(String value) { return value != null && !value.isBlank(); }
    private static boolean finiteNonNull(Object value) {
        if (value == null) return false;
        if (value instanceof Number number) return Double.isFinite(number.doubleValue());
        if (value instanceof Map<?, ?> map) return map.entrySet().stream()
            .allMatch(entry -> entry.getKey() != null && finiteNonNull(entry.getValue()));
        return true;
    }

    private static Map<NaturalField, Set<String>> domains() {
        Map<NaturalField, Set<String>> out = new EnumMap<>(NaturalField.class);
        Set<String> ecology = words("Unknown|Warm Ocean|Temperate Ocean|Cold Ocean|Frozen Ocean|Ice Sheet|Tundra|Alpine|Boreal Forest|Temperate Forest|Temperate Grassland|Temperate Desert|Mediterranean Shrubland|Subtropical Forest|Hot Desert|Savanna|Tropical Monsoon Forest|Tropical Rainforest|Wetland");
        out.put(NaturalField.ECO_REGION, ecology);
        out.put(NaturalField.BIOME_FAMILY, ecology);
        out.put(NaturalField.TERRAIN_VARIANT, words("Unknown|Ocean|Plain|Rolling Plain|Hills|Foothills|Mountain|Plateau|Floodplain|Alluvial Plain|Alluvial Fan|Canyon"));
        out.put(NaturalField.SURFACE_PROFILE, words("Unknown|Ocean Sand|Ocean Gravel|Frozen Ocean|Ice / Snow|Tundra Soil|Alpine Rock|Alpine Snow|Forest Loam|Podzol|Grass Loam|Dry Grass Soil|Shrub Soil|Desert Sand|Desert Gravel|Alluvial Silt|Wet Mud|Peat"));
        out.put(NaturalField.VEGETATION_PROFILE, words("Unknown|None|Tundra Sparse|Alpine Sparse|Conifer Forest|Mixed Forest|Broadleaf Forest|Dense Rainforest|Monsoon Forest|Grassland|Dry Grassland|Shrubland|Savanna|Desert Sparse|Wetland"));
        out.put(NaturalField.WATER_KIND, words("None|Ocean|Lake|River"));
        out.put(NaturalField.MINING_DISTRICT_CLASS, words("none|low_potential|surface_district|shallow_district|deep_district|mixed_district"));
        out.put(NaturalField.WATERWAY_CLASS, words("open_sea|shallow_sea|lake|shallow_lake|major_river|river|local_waterway|non_navigable|none"));
        out.put(NaturalField.PORT_TYPE, words("none|sea|lake|river"));
        out.put(NaturalField.CIVILIZATION_NODE_CLASS, words("none|agricultural_settlement|port_city|mining_town|transport_node|low_potential|multi_role_hub"));
        out.put(NaturalField.REGIONAL_DOMINANT_ADVANTAGE, words("none|building|agriculture|port|mining|transport"));
        out.put(NaturalField.REGIONAL_ADVANTAGE_CLASS, words("none|building_region|agriculture_region|port_region|mining_region|transport_region|low_potential|multi_advantage_region"));
        out.put(NaturalField.NATURAL_CORRIDOR_CLASS, words("none|river_valley|hill_pass|mountain_pass|plateau_access|basin_gateway|low_divide|low_potential"));
        return Map.copyOf(out);
    }
    private static Set<String> words(String values) { return Set.of(values.split("\\|")); }
}
