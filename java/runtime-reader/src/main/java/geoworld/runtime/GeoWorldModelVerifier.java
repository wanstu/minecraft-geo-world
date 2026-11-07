package geoworld.runtime;

import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Comparator;
import java.util.EnumSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * Executable black-box acceptance verifier for the public World Model query
 * contract. It only uses persisted .geoworld files and public runtime APIs.
 */
public final class GeoWorldModelVerifier {
    private GeoWorldModelVerifier() {}

    private static final List<String> VERIFIED_IDS = List.of(
        "MODEL-01",
        "MODEL-02",
        "MODEL-03",
        "MODEL-04",
        "MODEL-05",
        "MODEL-06",
        "MODEL-07",
        "MODEL-08",
        "MODEL-09",
        "MODEL-10",
        "MODEL-11",
        "MODEL-12",
        "MODEL-13",
        "MODEL-14",
        "MODEL-15",
        "RUNTIME-01",
        "RUNTIME-02",
        "RUNTIME-03",
        "RUNTIME-04",
        "RUNTIME-05",
        "PROFILE-01",
        "PROFILE-02",
        "PROFILE-04",
        "PARAM-05",
        "QUERY-01", "QUERY-02", "QUERY-03", "QUERY-04", "QUERY-05",
        "QUERY-06", "QUERY-07", "QUERY-08", "QUERY-09"
    );

    public static void main(String[] args) throws Exception {
        Path input = args.length >= 1
            ? Path.of(args[0])
            : Path.of("internal/goldenvectors/testdata/runtime-v1.geoworld");

        List<Path> models = discoverModels(input);
        if (models.isEmpty()) {
            throw new AssertionError("no .geoworld models found under " + input);
        }

        Path registry = args.length >= 3
            ? Path.of(args[2])
            : findRegistry(input);
        Map<String, List<String>> failures = new LinkedHashMap<>();
        for (String id : VERIFIED_IDS) {
            failures.put(id, new ArrayList<>());
        }

        for (Path path : models) {
            try {
                verifyModel(path);
            } catch (VerificationFailure failure) {
                failures.get(failure.id).add(path + ": " + failure.getMessage());
            } catch (Throwable failure) {
                for (String id : VERIFIED_IDS) {
                    if (id.startsWith("QUERY-")) continue;
                    failures.get(id).add(path + ": unexpected verifier failure: " + failure);
                }
            }
            GeoWorldQueryContractVerifier.verify(path, registry, failures);
        }

        int passed = 0;
        for (String id : VERIFIED_IDS) {
            List<String> problems = failures.get(id);
            if (problems.isEmpty()) {
                passed++;
                System.out.printf("PASS %s models=%d%n", id, models.size());
            } else {
                System.out.printf("FAIL %s models=%d failures=%d%n", id, models.size(), problems.size());
                for (String problem : problems) {
                    System.out.println("  " + problem);
                }
            }
        }

        System.out.printf(
            "GeoWorld Model Verifier: passed=%d/%d models=%d%n",
            passed,
            VERIFIED_IDS.size(),
            models.size()
        );
        if (passed != VERIFIED_IDS.size()) {
            throw new AssertionError("World Model verifier failed");
        }
        if (args.length >= 2) {
            writeEvidence(Path.of(args[1]), models.size());
        }
    }

    private static Path findRegistry(Path input) {
        Path start = input.toAbsolutePath().normalize();
        if (!Files.isDirectory(start)) start = start.getParent();
        for (Path dir = start; dir != null; dir = dir.getParent()) {
            Path candidate = dir.resolve("docs/FACT_REGISTRY.md");
            if (Files.isRegularFile(candidate)) return candidate;
        }
        throw new IllegalArgumentException("cannot locate docs/FACT_REGISTRY.md from " + input);
    }

    private static void writeEvidence(Path output, int modelCount)
        throws IOException {
        Path parent = output.toAbsolutePath().getParent();
        if (parent != null) {
            Files.createDirectories(parent);
        }
        StringBuilder json = new StringBuilder();
        json.append("{\n");
        json.append("  \"format\": \"geoworld-model-verifier-evidence\",\n");
        json.append("  \"version\": 1,\n");
        json.append("  \"modelCount\": ").append(modelCount).append(",\n");
        json.append("  \"passedIds\": [\n");
        // Preserve the original 177-item Go report schema. Query acceptance
        // has its own supplementary IDs, recorded separately in this evidence.
        List<String> evidenceIds = VERIFIED_IDS.stream()
            .filter(id -> !id.startsWith("QUERY-")).toList();
        for (int i = 0; i < evidenceIds.size(); i++) {
            json.append("    \"").append(evidenceIds.get(i)).append("\"");
            if (i + 1 < evidenceIds.size()) {
                json.append(",");
            }
            json.append("\n");
        }
        json.append("  ],\n");
        json.append("  \"queryPassedIds\": [");
        json.append(String.join(", ", GeoWorldQueryContractVerifier.IDS.stream()
            .map(id -> "\"" + id + "\"").toList()));
        json.append("]\n");
        json.append("}\n");
        Files.writeString(output, json.toString(), StandardCharsets.UTF_8);
        System.out.println("Verifier evidence: " + output);
    }

    private static List<Path> discoverModels(Path input) throws IOException {
        if (Files.isRegularFile(input)) {
            return List.of(input);
        }
        if (!Files.isDirectory(input)) {
            return List.of();
        }
        try (var stream = Files.walk(input)) {
            return stream
                .filter(Files::isRegularFile)
                .filter(path -> path.getFileName().toString().endsWith(".geoworld"))
                .sorted(Comparator.comparing(Path::toString))
                .toList();
        }
    }

    private static void verifyModel(Path path) throws Exception {
        GeoWorldV1 model = GeoWorldV1.load(path);
        verifyProfileIdentity(model, path);
        verifyDeterminism(model, path);
        verifyUnloadedNaturalQuery(model, path);
        verifyNoForcedLoadAndLocalReality(model, path);
        verifyDynamicUnavailable(model, path);
        verifyResolutions(model, path);
        verifySharedCache(model, path);
        verifyFieldLaziness(model, path);
        verifySettlementSuitability(model, path);
        verifyNavigationSuitability(model, path);
        verifyMiningDistrict(model, path);
        verifyCivilizationSuitability(model, path);
        verifyRegionalAdvantage(model, path);
        verifyRegionalRegionIdentity(model, path);
        verifyTransportGeography(model, path);
        verifyInFlightCoalescing(model, path);
        verifyQueryIdentity(model, path);
    }

    private static void verifyProfileIdentity(GeoWorldV1 model, Path path) throws IOException {
        GeoWorldV1.Manifest manifest = model.manifest();
        GeoWorldV1.WorldProfile profile = model.worldProfile();

        require(
            "PROFILE-01",
            !profile.legacy(),
            "current acceptance model unexpectedly loaded as legacy"
        );
        require(
            "PROFILE-01",
            nonBlank(manifest.worldProfileVersion()) &&
                nonBlank(manifest.generationRulesVersion()) &&
                nonBlank(manifest.worldModelVersion()) &&
                nonBlank(manifest.preset()),
            "incomplete persisted world identity"
        );
        require(
            "PROFILE-01",
            manifest.worldProfileVersion().equals(profile.worldProfileVersion()) &&
                manifest.generationRulesVersion().equals(profile.generationRulesVersion()) &&
                manifest.worldModelVersion().equals(profile.worldModelVersion()) &&
                manifest.preset().equals(profile.preset()),
            "manifest and locked WorldProfile identity differ"
        );

        double[] parameters = profile.parameters().values();
        require(
            "PROFILE-02",
            parameters.length == 25,
            "parameter snapshot count=" + parameters.length + " want=25"
        );
        for (int i = 0; i < parameters.length; i++) {
            require(
                "PROFILE-02",
                Double.isFinite(parameters[i]),
                "parameter[" + i + "] is not finite"
            );
        }
        require(
            "PROFILE-02",
            manifest.parameterSnapshotSha256() != null &&
                manifest.parameterSnapshotSha256().matches("[0-9a-f]{64}"),
            "missing/invalid parameter snapshot SHA-256"
        );

        GeoWorldV1 reopened = GeoWorldV1.load(path);
        require(
            "PARAM-05",
            manifest.parameterSnapshotSha256().equals(
                reopened.manifest().parameterSnapshotSha256()
            ) &&
                Arrays.equals(
                    parameters,
                    reopened.worldProfile().parameters().values()
                ),
            "persisted parameter snapshot changed after reopening the world model"
        );
    }

    private static void verifyDeterminism(GeoWorldV1 model, Path path) throws Exception {
        GeoWorldV1.Geometry g = model.manifest().geometry();
        double x = g.originX() + g.spanXBlocks() * 0.371 + 0.37;
        double z = g.originZ() + g.spanZBlocks() * 0.433 + 0.61;

        var request = GeoWorldQueryService.QueryRequest.all(
            x,
            z,
            GeoWorldQueryService.Resolution.BLOCK_4
        );
        GeoWorldQueryService firstService = new GeoWorldQueryService(model);
        var first = firstService.queryNatural(request);
        var second = firstService.queryNatural(request);
        require(
            "MODEL-01",
            first.equals(second),
            "same model/request returned different values"
        );

        GeoWorldV1 reopened = GeoWorldV1.load(path);
        var reopenedResult = new GeoWorldQueryService(reopened).queryNatural(request);
        require(
            "MODEL-01",
            first.equals(reopenedResult),
            "reopening the same persisted model changed Natural Model result"
        );
    }

    private static void verifyUnloadedNaturalQuery(GeoWorldV1 model, Path path) {
        GeoWorldV1.Geometry g = model.manifest().geometry();
        double x = g.originX() + g.spanXBlocks() * 3.5;
        double z = g.originZ() + g.spanZBlocks() * 0.5;
        var request = new GeoWorldQueryService.QueryRequest(
            x,
            z,
            GeoWorldQueryService.Resolution.BLOCK_16,
            EnumSet.of(
                GeoWorldQueryService.NaturalField.ELEVATION,
                GeoWorldQueryService.NaturalField.BIOME_FAMILY,
                GeoWorldQueryService.NaturalField.WATER_KIND
            )
        );
        var result = new GeoWorldQueryService(model).queryNatural(request);
        require(
            "MODEL-02",
            result.values().containsKey(GeoWorldQueryService.NaturalField.ELEVATION) &&
                result.values().containsKey(GeoWorldQueryService.NaturalField.BIOME_FAMILY) &&
                result.values().containsKey(GeoWorldQueryService.NaturalField.WATER_KIND),
            "Natural Model query outside loaded/model area did not return persisted/model-derived facts"
        );
    }

    private static void verifyNoForcedLoadAndLocalReality(
        GeoWorldV1 model,
        Path path
    ) {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        TrapLocalProvider unloaded = new TrapLocalProvider();

        // Natural queries cannot even receive a chunk provider. Keep a trap
        // provider beside the call and assert it remains untouched.
        var naturalRequest = new GeoWorldQueryService.QueryRequest(
            19.75,
            -33.25,
            GeoWorldQueryService.Resolution.BLOCK_1,
            EnumSet.of(GeoWorldQueryService.NaturalField.ELEVATION)
        );
        service.queryNatural(naturalRequest);
        require(
            "MODEL-03",
            unloaded.loadedChecks.get() == 0 && unloaded.loadedQueries.get() == 0,
            "Natural Model query touched Local Reality provider"
        );

        var unavailable = service.queryLocalReality(128, -64, unloaded);
        require(
            "MODEL-03",
            unloaded.loadedQueries.get() == 0,
            "unloaded Local Reality query attempted to query/load chunk facts"
        );
        require(
            "MODEL-04",
            !unavailable.available() &&
                unloaded.loadedChecks.get() == 1 &&
                unloaded.loadedQueries.get() == 0,
            "Local Reality did not stop at isLoaded=false"
        );

        var stableNaturalRequest = new GeoWorldQueryService.QueryRequest(
            141.25,
            -72.75,
            GeoWorldQueryService.Resolution.BLOCK_4,
            EnumSet.of(GeoWorldQueryService.NaturalField.ELEVATION)
        );
        var stableNatural = service.queryNatural(stableNaturalRequest);
        int naturalCacheBeforeLocalChange = service.cacheSize();

        LoadedLocalProvider loaded = new LoadedLocalProvider();
        var first = service.queryLocalReality(128, -64, loaded);
        var second = service.queryLocalReality(128, -64, loaded);
        require(
            "MODEL-04",
            first.available() &&
                second.available() &&
                loaded.loadedQueries.get() == 1 &&
                service.localRealityCacheSize() == 1,
            "loaded Local Reality was not reused from its short-lived cache"
        );
        require(
            "RUNTIME-04",
            first == second &&
                loaded.loadedQueries.get() == 1 &&
                service.localRealityCacheSize() == 1,
            "multiple consumers did not reuse the same loaded-area sample"
        );

        loaded.revision++;
        service.queryLocalReality(128, -64, loaded);
        require(
            "MODEL-04",
            loaded.loadedQueries.get() == 2,
            "Local Reality cache revision did not invalidate cached facts"
        );
        var stableNaturalAfterLocalChange = service.queryNatural(stableNaturalRequest);
        require(
            "RUNTIME-03",
            stableNaturalAfterLocalChange == stableNatural &&
                service.cacheSize() == naturalCacheBeforeLocalChange,
            "local Reality revision invalidated or recomputed Natural Model cache"
        );
        require(
            "RUNTIME-05",
            loaded.loadedQueries.get() == 2 &&
                loaded.lastBlockX == 128 &&
                loaded.lastBlockZ == -64,
            "Local Reality provider touched coordinates outside the explicitly requested active cell"
        );

        int localCacheSize = service.localRealityCacheSize();
        service.clearCache();
        require(
            "MODEL-07",
            service.localRealityCacheSize() == localCacheSize,
            "Natural Model cache clear interfered with Local Reality cache"
        );
    }

    private static void verifyDynamicUnavailable(GeoWorldV1 model, Path path) {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        AtomicInteger dynamicQueries = new AtomicInteger();
        GeoWorldQueryService.DynamicStateProvider provider = (field, x, z) -> {
            dynamicQueries.incrementAndGet();
            return Optional.empty();
        };

        service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                11.25,
                17.75,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.WIND_X,
                    GeoWorldQueryService.NaturalField.WATER_KIND
                )
            )
        );
        require(
            "RUNTIME-01",
            dynamicQueries.get() == 0,
            "Natural Model query invoked a global real-time dynamic provider"
        );
        require(
            "RUNTIME-02",
            dynamicQueries.get() == 0,
            "dynamic computation ran without an explicit consumer request"
        );

        Optional<Object> value = service.queryDynamicState(
            "snow_cover",
            0,
            0,
            provider
        );
        require(
            "MODEL-05",
            value.isEmpty(),
            "missing dynamic state was synthesized instead of unavailable"
        );
        require(
            "RUNTIME-02",
            dynamicQueries.get() == 1,
            "explicit dynamic consumer request did not invoke exactly one provider query"
        );
    }

    private static void verifyResolutions(GeoWorldV1 model, Path path) {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        double x = 123.75;
        double z = -45.25;
        for (GeoWorldQueryService.Resolution resolution :
            GeoWorldQueryService.Resolution.values()) {
            var request = new GeoWorldQueryService.QueryRequest(
                x,
                z,
                resolution,
                EnumSet.of(GeoWorldQueryService.NaturalField.ELEVATION)
            );
            var result = service.queryNatural(request);
            double expectedX = quantize(x, resolution.blocks());
            double expectedZ = quantize(z, resolution.blocks());
            require(
                "MODEL-06",
                result.resolution() == resolution &&
                    Double.compare(result.sampleX(), expectedX) == 0 &&
                    Double.compare(result.sampleZ(), expectedZ) == 0,
                "resolution " + resolution + " did not quantize to expected cell"
            );
        }
        require(
            "MODEL-06",
            GeoWorldQueryService.Resolution.BLOCK_1.blocks() == 1 &&
                GeoWorldQueryService.Resolution.BLOCK_4.blocks() == 4 &&
                GeoWorldQueryService.Resolution.BLOCK_16.blocks() == 16,
            "resolution contract is not exactly 1/4/16 blocks"
        );
    }

    private static void verifySharedCache(GeoWorldV1 model, Path path) {
        GeoWorldQueryService sharedA = GeoWorldQueryService.shared(model);
        GeoWorldQueryService sharedB = GeoWorldQueryService.shared(model);
        require(
            "MODEL-07",
            sharedA == sharedB,
            "shared(model) returned different services"
        );

        sharedA.clearCache();
        var request = new GeoWorldQueryService.QueryRequest(
            77.25,
            91.75,
            GeoWorldQueryService.Resolution.BLOCK_4,
            EnumSet.of(
                GeoWorldQueryService.NaturalField.ELEVATION,
                GeoWorldQueryService.NaturalField.WATER_KIND
            )
        );
        var first = sharedA.queryNatural(request);
        var second = sharedB.queryNatural(request);
        require(
            "MODEL-07",
            first == second && sharedA.cacheSize() == 1,
            "same cell/request was not reused from shared cache"
        );

        GeoWorldQueryService bounded = new GeoWorldQueryService(model, 64);
        for (int i = 0; i < 160; i++) {
            bounded.queryNatural(new GeoWorldQueryService.QueryRequest(
                i * 17.0 + 0.25,
                i * -11.0 - 0.75,
                GeoWorldQueryService.Resolution.BLOCK_1,
                EnumSet.of(GeoWorldQueryService.NaturalField.ELEVATION)
            ));
        }
        require(
            "MODEL-07",
            bounded.cacheSize() <= 64,
            "bounded natural cache exceeded configured maximum: " + bounded.cacheSize()
        );
    }

    private static void verifyFieldLaziness(
        GeoWorldV1 model,
        Path path
    ) {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        service.resetComputationStats();

        var elevation = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                321.5,
                654.5,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(GeoWorldQueryService.NaturalField.ELEVATION)
            )
        );
        var afterElevation = service.computationStats();
        require(
            "MODEL-08",
            elevation.values().size() == 1 &&
                elevation.values().containsKey(
                    GeoWorldQueryService.NaturalField.ELEVATION
                ) &&
                afterElevation.climateQueries() == 0 &&
                afterElevation.hydrologyQueries() == 0 &&
                afterElevation.naturalFeatureQueries() == 0,
            "elevation-only query evaluated unrelated field groups"
        );

        service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                325.5,
                654.5,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(GeoWorldQueryService.NaturalField.ANNUAL_MEAN_TEMP_C)
            )
        );
        var afterClimate = service.computationStats();
        require(
            "MODEL-08",
            afterClimate.climateQueries() == 1 &&
                afterClimate.hydrologyQueries() == 0 &&
                afterClimate.naturalFeatureQueries() == 0,
            "climate field did not remain isolated from other field groups"
        );

        service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                329.5,
                654.5,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(GeoWorldQueryService.NaturalField.FLOW_ACCUMULATION)
            )
        );
        var afterHydrology = service.computationStats();
        require(
            "MODEL-08",
            afterHydrology.climateQueries() == 1 &&
                afterHydrology.hydrologyQueries() == 1 &&
                afterHydrology.naturalFeatureQueries() == 0,
            "hydrology field did not remain isolated from other field groups"
        );

        service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                333.5,
                654.5,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(GeoWorldQueryService.NaturalField.DISTANCE_TO_COAST)
            )
        );
        var afterFeatures = service.computationStats();
        require(
            "MODEL-08",
            afterFeatures.climateQueries() == 1 &&
                afterFeatures.hydrologyQueries() == 1 &&
                afterFeatures.naturalFeatureQueries() == 1,
            "natural-feature field did not remain isolated from other field groups"
        );

        service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                337.5,
                654.5,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.SETTLEMENT_SUITABILITY
                )
            )
        );
        var afterSettlement = service.computationStats();
        require(
            "MODEL-08",
            afterSettlement.climateQueries() == 2 &&
                afterSettlement.hydrologyQueries() == 1 &&
                afterSettlement.naturalFeatureQueries() == 2,
            "settlement query did not lazily reuse exactly its climate/natural-feature dependencies"
        );

        service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                341.5,
                654.5,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.NEAREST_RIVER_HALF_WIDTH
                )
            )
        );
        var afterNavigationGeometry = service.computationStats();
        require(
            "MODEL-08",
            afterNavigationGeometry.climateQueries() == 2 &&
                afterNavigationGeometry.hydrologyQueries() == 1 &&
                afterNavigationGeometry.naturalFeatureQueries() == 2,
            "river-geometry navigation query evaluated unrelated field groups"
        );

        service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                345.5,
                654.5,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.PORT_SUITABILITY
                )
            )
        );
        var afterNavigationScore = service.computationStats();
        require(
            "MODEL-08",
            afterNavigationScore.climateQueries() == 3 &&
                afterNavigationScore.hydrologyQueries() == 1 &&
                afterNavigationScore.naturalFeatureQueries() == 3,
            "navigation score did not lazily reuse settlement/natural-feature dependencies"
        );

        service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                349.5,
                654.5,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.NATURAL_CORRIDOR_SUITABILITY
                )
            )
        );
        var afterTransport = service.computationStats();
        require(
            "MODEL-08",
            afterTransport.climateQueries() == 4 &&
                afterTransport.hydrologyQueries() == 2 &&
                afterTransport.naturalFeatureQueries() == 4,
            "transport geography did not lazily load exactly its natural dependencies"
        );
    }

    private static void verifySettlementSuitability(
        GeoWorldV1 model,
        Path path
    ) {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        double landX = Double.NaN;
        double landZ = Double.NaN;
        double waterX = Double.NaN;
        double waterZ = Double.NaN;

        for (int z = 0; z < geometry.height(); z++) {
            double worldZ = geometry.originZ() + z * geometry.spacingZ();
            for (int x = 0; x < geometry.width(); x++) {
                double worldX = geometry.originX() + x * geometry.spacingX();
                GeoWorldV1.RuntimeSample sample = model.sample(worldX, worldZ);
                if (
                    Double.isNaN(landX) &&
                    sample.sourceLand() &&
                    "None".equals(sample.waterKind())
                ) {
                    landX = worldX;
                    landZ = worldZ;
                }
                if (
                    Double.isNaN(waterX) &&
                    !"None".equals(sample.waterKind())
                ) {
                    waterX = worldX;
                    waterZ = worldZ;
                }
                if (!Double.isNaN(landX) && !Double.isNaN(waterX)) {
                    break;
                }
            }
            if (!Double.isNaN(landX) && !Double.isNaN(waterX)) {
                break;
            }
        }

        require(
            "MODEL-09",
            !Double.isNaN(landX) && !Double.isNaN(waterX),
            "model did not expose both a dry-land and water sample"
        );

        EnumSet<GeoWorldQueryService.NaturalField> settlementFields =
            EnumSet.of(
                GeoWorldQueryService.NaturalField.SETTLEMENT_BUILDABILITY,
                GeoWorldQueryService.NaturalField.FRESH_WATER_ACCESS,
                GeoWorldQueryService.NaturalField.FLOOD_RISK,
                GeoWorldQueryService.NaturalField.AGRICULTURE_SUITABILITY,
                GeoWorldQueryService.NaturalField.CLIMATE_COMFORT,
                GeoWorldQueryService.NaturalField.TRANSPORT_ACCESS,
                GeoWorldQueryService.NaturalField.RESOURCE_ACCESS,
                GeoWorldQueryService.NaturalField.SETTLEMENT_SUITABILITY
            );

        var land = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                landX,
                landZ,
                GeoWorldQueryService.Resolution.BLOCK_16,
                settlementFields
            )
        );
        require(
            "MODEL-09",
            land.unavailableFields().isEmpty() &&
                land.values().size() == settlementFields.size(),
            "current model did not expose all settlement suitability fields"
        );
        for (Object value : land.values().values()) {
            require(
                "MODEL-09",
                value instanceof Number number &&
                    number.doubleValue() >= 0 &&
                    number.doubleValue() <= 1 &&
                    Double.isFinite(number.doubleValue()),
                "settlement suitability field escaped [0,1]"
            );
        }

        var water = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                waterX,
                waterZ,
                GeoWorldQueryService.Resolution.BLOCK_1,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.SETTLEMENT_SUITABILITY,
                    GeoWorldQueryService.NaturalField.FLOOD_RISK
                )
            )
        );
        double waterSuitability = ((Number) water.values().get(
            GeoWorldQueryService.NaturalField.SETTLEMENT_SUITABILITY
        )).doubleValue();
        double waterRisk = ((Number) water.values().get(
            GeoWorldQueryService.NaturalField.FLOOD_RISK
        )).doubleValue();
        require(
            "MODEL-09",
            waterSuitability == 0.0 && waterRisk == 1.0,
            "water sample was incorrectly considered a land settlement site"
        );
    }

    private static void verifyNavigationSuitability(
        GeoWorldV1 model,
        Path path
    ) {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        GeoWorldV1.Geometry geometry = model.manifest().geometry();

        double landX = Double.NaN;
        double landZ = Double.NaN;
        double farLandX = Double.NaN;
        double farLandZ = Double.NaN;
        double openWaterX = Double.NaN;
        double openWaterZ = Double.NaN;
        double riverProbeX = Double.NaN;
        double riverProbeZ = Double.NaN;

        for (int z = 0; z < geometry.height(); z++) {
            double worldZ = geometry.originZ() + z * geometry.spacingZ();
            for (int x = 0; x < geometry.width(); x++) {
                double worldX = geometry.originX() + x * geometry.spacingX();
                GeoWorldV1.RuntimeSample sample = model.sample(worldX, worldZ);

                if (
                    Double.isNaN(landX) &&
                    sample.sourceLand() &&
                    "None".equals(sample.waterKind())
                ) {
                    landX = worldX;
                    landZ = worldZ;
                }
                if (
                    Double.isNaN(farLandX) &&
                    sample.sourceLand() &&
                    "None".equals(sample.waterKind())
                ) {
                    GeoWorldV1.NaturalFeatureSample natural =
                        model.sampleNaturalFeatures(worldX, worldZ);
                    if (
                        natural.distanceToRiver() != null &&
                        natural.distanceToRiver() > 256
                    ) {
                        farLandX = worldX;
                        farLandZ = worldZ;
                    }
                }
                if (
                    Double.isNaN(openWaterX) &&
                    (
                        "Ocean".equals(sample.waterKind()) ||
                        "Lake".equals(sample.waterKind())
                    )
                ) {
                    openWaterX = worldX;
                    openWaterZ = worldZ;
                }
                if (Double.isNaN(riverProbeX)) {
                    GeoWorldV1.NavigationFeatureSample river =
                        model.sampleNavigationFeatures(worldX, worldZ);
                    if (river.available()) {
                        riverProbeX = worldX;
                        riverProbeZ = worldZ;
                    }
                }

                if (
                    !Double.isNaN(landX) &&
                    !Double.isNaN(farLandX) &&
                    !Double.isNaN(openWaterX) &&
                    !Double.isNaN(riverProbeX)
                ) {
                    break;
                }
            }
            if (
                !Double.isNaN(landX) &&
                !Double.isNaN(farLandX) &&
                !Double.isNaN(openWaterX) &&
                !Double.isNaN(riverProbeX)
            ) {
                break;
            }
        }

        require(
            "MODEL-10",
            !Double.isNaN(landX) &&
                !Double.isNaN(farLandX) &&
                !Double.isNaN(openWaterX) &&
                !Double.isNaN(riverProbeX),
            "model did not expose land, far land, open water and persisted river geometry"
        );

        EnumSet<GeoWorldQueryService.NaturalField> scoreFields =
            EnumSet.of(
                GeoWorldQueryService.NaturalField.WATERWAY_NAVIGABILITY,
                GeoWorldQueryService.NaturalField.PORT_SUITABILITY,
                GeoWorldQueryService.NaturalField.NAVIGATION_BOTTLENECK_RISK,
                GeoWorldQueryService.NaturalField.WATERWAY_CLASS,
                GeoWorldQueryService.NaturalField.PORT_TYPE
            );

        var land = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                landX,
                landZ,
                GeoWorldQueryService.Resolution.BLOCK_16,
                scoreFields
            )
        );
        require(
            "MODEL-10",
            land.unavailableFields().isEmpty() &&
                land.values().size() == scoreFields.size(),
            "current model did not expose all navigation suitability fields"
        );
        for (
            GeoWorldQueryService.NaturalField field :
            List.of(
                GeoWorldQueryService.NaturalField.WATERWAY_NAVIGABILITY,
                GeoWorldQueryService.NaturalField.PORT_SUITABILITY,
                GeoWorldQueryService.NaturalField.NAVIGATION_BOTTLENECK_RISK
            )
        ) {
            Object value = land.values().get(field);
            require(
                "MODEL-10",
                value instanceof Number number &&
                    Double.isFinite(number.doubleValue()) &&
                    number.doubleValue() >= 0 &&
                    number.doubleValue() <= 1,
                field + " escaped [0,1]"
            );
        }
        require(
            "MODEL-10",
            land.values().get(
                GeoWorldQueryService.NaturalField.WATERWAY_CLASS
            ) instanceof String waterwayClass &&
                nonBlank(waterwayClass) &&
            land.values().get(
                GeoWorldQueryService.NaturalField.PORT_TYPE
            ) instanceof String portType &&
                nonBlank(portType),
            "navigation classification fields were blank"
        );

        var river = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                riverProbeX,
                riverProbeZ,
                GeoWorldQueryService.Resolution.BLOCK_4,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.NEAREST_RIVER_HALF_WIDTH,
                    GeoWorldQueryService.NaturalField.NEAREST_RIVER_GRADIENT
                )
            )
        );
        require(
            "MODEL-10",
            river.unavailableFields().isEmpty(),
            "persisted river geometry was unavailable to navigation query"
        );
        double halfWidth = ((Number) river.values().get(
            GeoWorldQueryService.NaturalField.NEAREST_RIVER_HALF_WIDTH
        )).doubleValue();
        double gradient = ((Number) river.values().get(
            GeoWorldQueryService.NaturalField.NEAREST_RIVER_GRADIENT
        )).doubleValue();
        require(
            "MODEL-10",
            Double.isFinite(halfWidth) &&
                halfWidth > 0 &&
                Double.isFinite(gradient) &&
                gradient >= 0,
            "river width/gradient facts were invalid"
        );

        var water = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                openWaterX,
                openWaterZ,
                GeoWorldQueryService.Resolution.BLOCK_1,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.WATERWAY_NAVIGABILITY,
                    GeoWorldQueryService.NaturalField.PORT_SUITABILITY,
                    GeoWorldQueryService.NaturalField.NAVIGATION_BOTTLENECK_RISK,
                    GeoWorldQueryService.NaturalField.WATERWAY_CLASS,
                    GeoWorldQueryService.NaturalField.PORT_TYPE
                )
            )
        );
        double waterNavigability = ((Number) water.values().get(
            GeoWorldQueryService.NaturalField.WATERWAY_NAVIGABILITY
        )).doubleValue();
        double waterPort = ((Number) water.values().get(
            GeoWorldQueryService.NaturalField.PORT_SUITABILITY
        )).doubleValue();
        Object waterPortType = water.values().get(
            GeoWorldQueryService.NaturalField.PORT_TYPE
        );
        require(
            "MODEL-10",
            waterNavigability > 0 &&
                waterNavigability <= 1 &&
                waterPort == 0.0 &&
                "none".equals(waterPortType),
            "open water was not navigable or was incorrectly considered a port site"
        );

        var farLand = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                farLandX,
                farLandZ,
                GeoWorldQueryService.Resolution.BLOCK_16,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.WATERWAY_NAVIGABILITY,
                    GeoWorldQueryService.NaturalField.WATERWAY_CLASS
                )
            )
        );
        double farNavigability = ((Number) farLand.values().get(
            GeoWorldQueryService.NaturalField.WATERWAY_NAVIGABILITY
        )).doubleValue();
        Object farClass = farLand.values().get(
            GeoWorldQueryService.NaturalField.WATERWAY_CLASS
        );
        require(
            "MODEL-10",
            farNavigability == 0.0 && "none".equals(farClass),
            "far inland land was incorrectly classified as a river waterway"
        );
    }

    private static void verifyMiningDistrict(
        GeoWorldV1 model,
        Path path
    ) {
        require(
            "MODEL-11",
            Math.abs(GeoWorldMineralAffinity.affinity(
                "IRON", "Mountain", "Alpine", 80
            ) - 19.8) <= 1.0e-12,
            "shared mineral affinity drifted from mineral rules"
        );

        GeoWorldQueryService service = new GeoWorldQueryService(model);
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        double[] land = findMiningProbe(model, geometry, false);
        double[] water = findMiningProbe(model, geometry, true);
        require(
            "MODEL-11",
            land != null && water != null,
            "model did not expose land and water mining probes"
        );

        var fields = EnumSet.of(
            GeoWorldQueryService.NaturalField.MINING_SUITABILITY,
            GeoWorldQueryService.NaturalField.SURFACE_MINING_SUITABILITY,
            GeoWorldQueryService.NaturalField.SHALLOW_MINING_SUITABILITY,
            GeoWorldQueryService.NaturalField.DEEP_MINING_SUITABILITY,
            GeoWorldQueryService.NaturalField.INDUSTRIAL_MINERAL_VALUE,
            GeoWorldQueryService.NaturalField.MINERAL_RESOURCE_DIVERSITY,
            GeoWorldQueryService.NaturalField.DOMINANT_MINERAL_RESOURCE,
            GeoWorldQueryService.NaturalField.MINING_DISTRICT_CLASS,
            GeoWorldQueryService.NaturalField.MINERAL_RESOURCE_PROFILE
        );
        verifyMiningLand(service, land, fields);
        verifyMiningWater(service, water, fields);
    }

    private static double[] findMiningProbe(
        GeoWorldV1 model,
        GeoWorldV1.Geometry geometry,
        boolean wantWater
    ) {
        for (int z = 0; z < geometry.height(); z++) {
            double worldZ = geometry.originZ() + z * geometry.spacingZ();
            for (int x = 0; x < geometry.width(); x++) {
                double worldX = geometry.originX() + x * geometry.spacingX();
                GeoWorldV1.RuntimeSample sample = model.sample(worldX, worldZ);
                boolean water = !"None".equals(sample.waterKind());
                if (
                    water == wantWater &&
                    (wantWater || sample.sourceLand())
                ) {
                    return new double[]{worldX, worldZ};
                }
            }
        }
        return null;
    }

    private static void verifyMiningLand(
        GeoWorldQueryService service,
        double[] point,
        EnumSet<GeoWorldQueryService.NaturalField> fields
    ) {
        var result = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                point[0],
                point[1],
                GeoWorldQueryService.Resolution.BLOCK_16,
                fields
            )
        );
        require(
            "MODEL-11",
            result.unavailableFields().isEmpty() &&
                result.values().size() == fields.size(),
            "current model did not expose all mining district fields"
        );

        for (
            GeoWorldQueryService.NaturalField field :
            List.of(
                GeoWorldQueryService.NaturalField.MINING_SUITABILITY,
                GeoWorldQueryService.NaturalField.SURFACE_MINING_SUITABILITY,
                GeoWorldQueryService.NaturalField.SHALLOW_MINING_SUITABILITY,
                GeoWorldQueryService.NaturalField.DEEP_MINING_SUITABILITY,
                GeoWorldQueryService.NaturalField.INDUSTRIAL_MINERAL_VALUE,
                GeoWorldQueryService.NaturalField.MINERAL_RESOURCE_DIVERSITY
            )
        ) {
            double value = ((Number)result.values().get(field)).doubleValue();
            require(
                "MODEL-11",
                Double.isFinite(value) && value >= 0 && value <= 1,
                field + " escaped [0,1]"
            );
        }

        Object profileObject = result.values().get(
            GeoWorldQueryService.NaturalField.MINERAL_RESOURCE_PROFILE
        );
        require(
            "MODEL-11",
            profileObject instanceof Map<?, ?>,
            "mineral resource profile was not a map"
        );
        Map<?, ?> profile = (Map<?, ?>)profileObject;
        double sum = 0.0;
        for (Object value : profile.values()) {
            sum += ((Number)value).doubleValue();
        }
        require(
            "MODEL-11",
            profile.size() == 8 && Math.abs(sum - 1.0) <= 1.0e-9,
            "mineral resource profile was not normalized across eight minerals"
        );
    }

    private static void verifyMiningWater(
        GeoWorldQueryService service,
        double[] point,
        EnumSet<GeoWorldQueryService.NaturalField> fields
    ) {
        var result = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                point[0],
                point[1],
                GeoWorldQueryService.Resolution.BLOCK_1,
                fields
            )
        );
        double score = ((Number)result.values().get(
            GeoWorldQueryService.NaturalField.MINING_SUITABILITY
        )).doubleValue();
        require(
            "MODEL-11",
            score == 0.0 &&
                "none".equals(result.values().get(
                    GeoWorldQueryService.NaturalField.DOMINANT_MINERAL_RESOURCE
                )) &&
                "none".equals(result.values().get(
                    GeoWorldQueryService.NaturalField.MINING_DISTRICT_CLASS
                )),
            "water was incorrectly classified as a mining district"
        );
    }

    private static void verifyCivilizationSuitability(
        GeoWorldV1 model,
        Path path
    ) {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        double[] land = findMiningProbe(model, geometry, false);
        double[] water = findMiningProbe(model, geometry, true);
        require(
            "MODEL-12",
            land != null && water != null,
            "model did not expose land and water civilization probes"
        );

        EnumSet<GeoWorldQueryService.NaturalField> fields = EnumSet.of(
            GeoWorldQueryService.NaturalField.CIVILIZATION_AGRICULTURAL_SETTLEMENT_SUITABILITY,
            GeoWorldQueryService.NaturalField.CIVILIZATION_PORT_CITY_SUITABILITY,
            GeoWorldQueryService.NaturalField.CIVILIZATION_MINING_TOWN_SUITABILITY,
            GeoWorldQueryService.NaturalField.CIVILIZATION_TRANSPORT_NODE_SUITABILITY,
            GeoWorldQueryService.NaturalField.CIVILIZATION_NODE_SUITABILITY,
            GeoWorldQueryService.NaturalField.CIVILIZATION_SPECIALIZATION_STRENGTH,
            GeoWorldQueryService.NaturalField.CIVILIZATION_NODE_CLASS
        );

        for (GeoWorldQueryService.Resolution resolution :
            GeoWorldQueryService.Resolution.values()) {
            var result = service.queryNatural(
                new GeoWorldQueryService.QueryRequest(
                    land[0],
                    land[1],
                    resolution,
                    fields
                )
            );
            require(
                "MODEL-12",
                result.unavailableFields().isEmpty() &&
                    result.values().size() == fields.size(),
                "civilization fields unavailable at resolution " + resolution
            );
            for (
                GeoWorldQueryService.NaturalField field :
                List.of(
                    GeoWorldQueryService.NaturalField.CIVILIZATION_AGRICULTURAL_SETTLEMENT_SUITABILITY,
                    GeoWorldQueryService.NaturalField.CIVILIZATION_PORT_CITY_SUITABILITY,
                    GeoWorldQueryService.NaturalField.CIVILIZATION_MINING_TOWN_SUITABILITY,
                    GeoWorldQueryService.NaturalField.CIVILIZATION_TRANSPORT_NODE_SUITABILITY,
                    GeoWorldQueryService.NaturalField.CIVILIZATION_NODE_SUITABILITY,
                    GeoWorldQueryService.NaturalField.CIVILIZATION_SPECIALIZATION_STRENGTH
                )
            ) {
                Object value = result.values().get(field);
                require(
                    "MODEL-12",
                    value instanceof Number number &&
                        Double.isFinite(number.doubleValue()) &&
                        number.doubleValue() >= 0 &&
                        number.doubleValue() <= 1,
                    field + " escaped [0,1] at " + resolution
                );
            }
            require(
                "MODEL-12",
                result.values().get(
                    GeoWorldQueryService.NaturalField.CIVILIZATION_NODE_CLASS
                ) instanceof String nodeClass &&
                    nonBlank(nodeClass) &&
                    !"none".equals(nodeClass),
                "land civilization class was blank or none at " + resolution
            );
        }

        var waterResult = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                water[0],
                water[1],
                GeoWorldQueryService.Resolution.BLOCK_1,
                fields
            )
        );
        require(
            "MODEL-12",
            "none".equals(waterResult.values().get(
                GeoWorldQueryService.NaturalField.CIVILIZATION_NODE_CLASS
            )),
            "water was assigned a civilization node class"
        );
        for (
            GeoWorldQueryService.NaturalField field :
            List.of(
                GeoWorldQueryService.NaturalField.CIVILIZATION_AGRICULTURAL_SETTLEMENT_SUITABILITY,
                GeoWorldQueryService.NaturalField.CIVILIZATION_PORT_CITY_SUITABILITY,
                GeoWorldQueryService.NaturalField.CIVILIZATION_MINING_TOWN_SUITABILITY,
                GeoWorldQueryService.NaturalField.CIVILIZATION_TRANSPORT_NODE_SUITABILITY,
                GeoWorldQueryService.NaturalField.CIVILIZATION_NODE_SUITABILITY,
                GeoWorldQueryService.NaturalField.CIVILIZATION_SPECIALIZATION_STRENGTH
            )
        ) {
            double value = ((Number)waterResult.values().get(field)).doubleValue();
            require(
                "MODEL-12",
                value == 0.0,
                "water civilization field was non-zero: " + field
            );
        }

        GeoWorldV1.RuntimeSample landSample = model.sample(land[0], land[1]);
        GeoWorldSettlementSuitability.Score agriculturalSettlement =
            new GeoWorldSettlementSuitability.Score(
                0.90, 0.90, 0.08, 0.95,
                0.90, 0.72, 0.40, 0.88
            );
        GeoWorldNavigationSuitability.Score noNavigation =
            new GeoWorldNavigationSuitability.Score(
                0.0, 0.0, 0.0, "none", "none"
            );
        GeoWorldMiningDistrict.Score lowMining =
            new GeoWorldMiningDistrict.Score(
                0.20, 0.20, 0.20, 0.20,
                0.20, 0.30,
                "iron", "low_potential", Map.of()
            );
        var agricultural = GeoWorldCivilizationSuitability.evaluate(
            landSample,
            agriculturalSettlement,
            noNavigation,
            lowMining
        );
        require(
            "MODEL-12",
            "agricultural_settlement".equals(agricultural.nodeClass()),
            "synthetic agricultural site was misclassified"
        );

        GeoWorldSettlementSuitability.Score miningSettlement =
            new GeoWorldSettlementSuitability.Score(
                0.70, 0.55, 0.12, 0.35,
                0.80, 0.72, 0.95, 0.72
            );
        GeoWorldMiningDistrict.Score strongMining =
            new GeoWorldMiningDistrict.Score(
                0.94, 0.78, 0.90, 0.84,
                0.91, 0.82,
                "iron", "mixed_district", Map.of()
            );
        var miningTown = GeoWorldCivilizationSuitability.evaluate(
            landSample,
            miningSettlement,
            noNavigation,
            strongMining
        );
        require(
            "MODEL-12",
            "mining_town".equals(miningTown.nodeClass()),
            "synthetic mining site was misclassified"
        );
    }

    private static void verifyRegionalAdvantage(
        GeoWorldV1 model,
        Path path
    ) {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        double[] land = findMiningProbe(model, geometry, false);
        double[] water = findMiningProbe(model, geometry, true);
        require(
            "MODEL-13",
            land != null && water != null,
            "model did not expose land and water regional-advantage probes"
        );

        EnumSet<GeoWorldQueryService.NaturalField> fields = EnumSet.of(
            GeoWorldQueryService.NaturalField.REGIONAL_BUILDING_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_AGRICULTURE_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_PORT_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_MINING_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_TRANSPORT_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_SPECIALIZATION_STRENGTH,
            GeoWorldQueryService.NaturalField.REGIONAL_ADVANTAGE_OVERLAP_RISK,
            GeoWorldQueryService.NaturalField.REGIONAL_HIGH_ADVANTAGE_COUNT,
            GeoWorldQueryService.NaturalField.REGIONAL_DOMINANT_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_ADVANTAGE_CLASS
        );
        List<GeoWorldQueryService.NaturalField> normalized = List.of(
            GeoWorldQueryService.NaturalField.REGIONAL_BUILDING_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_AGRICULTURE_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_PORT_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_MINING_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_TRANSPORT_ADVANTAGE,
            GeoWorldQueryService.NaturalField.REGIONAL_SPECIALIZATION_STRENGTH,
            GeoWorldQueryService.NaturalField.REGIONAL_ADVANTAGE_OVERLAP_RISK
        );
        List<String> dominantValues = List.of(
            "building",
            "agriculture",
            "port",
            "mining",
            "transport"
        );
        List<String> classes = List.of(
            "low_potential",
            "building_region",
            "agriculture_region",
            "port_region",
            "mining_region",
            "transport_region",
            "multi_advantage_region"
        );

        for (GeoWorldQueryService.Resolution resolution :
            GeoWorldQueryService.Resolution.values()) {
            var result = service.queryNatural(
                new GeoWorldQueryService.QueryRequest(
                    land[0],
                    land[1],
                    resolution,
                    fields
                )
            );
            require(
                "MODEL-13",
                result.unavailableFields().isEmpty() &&
                    result.values().size() == fields.size(),
                "regional advantage fields unavailable at " + resolution
            );
            for (GeoWorldQueryService.NaturalField field : normalized) {
                Object value = result.values().get(field);
                require(
                    "MODEL-13",
                    value instanceof Number number &&
                        Double.isFinite(number.doubleValue()) &&
                        number.doubleValue() >= 0 &&
                        number.doubleValue() <= 1,
                    field + " escaped [0,1] at " + resolution
                );
            }
            Object highCount = result.values().get(
                GeoWorldQueryService.NaturalField.REGIONAL_HIGH_ADVANTAGE_COUNT
            );
            require(
                "MODEL-13",
                highCount instanceof Number number &&
                    number.intValue() >= 0 &&
                    number.intValue() <= 5 &&
                    Math.abs(number.doubleValue() - number.intValue()) < 1e-12,
                "regional high-advantage count escaped 0..5"
            );
            Object dominant = result.values().get(
                GeoWorldQueryService.NaturalField.REGIONAL_DOMINANT_ADVANTAGE
            );
            Object regionalClass = result.values().get(
                GeoWorldQueryService.NaturalField.REGIONAL_ADVANTAGE_CLASS
            );
            require(
                "MODEL-13",
                dominant instanceof String value &&
                    dominantValues.contains(value),
                "land regional dominant advantage was invalid: " + dominant
            );
            require(
                "MODEL-13",
                regionalClass instanceof String value &&
                    classes.contains(value),
                "land regional class was invalid: " + regionalClass
            );
        }

        var waterResult = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                water[0],
                water[1],
                GeoWorldQueryService.Resolution.BLOCK_1,
                fields
            )
        );
        for (GeoWorldQueryService.NaturalField field : normalized) {
            require(
                "MODEL-13",
                ((Number)waterResult.values().get(field)).doubleValue() == 0.0,
                "water regional advantage field was non-zero: " + field
            );
        }
        require(
            "MODEL-13",
            ((Number)waterResult.values().get(
                GeoWorldQueryService.NaturalField.REGIONAL_HIGH_ADVANTAGE_COUNT
            )).intValue() == 0 &&
                "none".equals(waterResult.values().get(
                    GeoWorldQueryService.NaturalField.REGIONAL_DOMINANT_ADVANTAGE
                )) &&
                "none".equals(waterResult.values().get(
                    GeoWorldQueryService.NaturalField.REGIONAL_ADVANTAGE_CLASS
                )),
            "water was incorrectly assigned a regional advantage"
        );

        GeoWorldQueryService.NaturalField[] all =
            GeoWorldQueryService.NaturalField.values();
        require(
            "MODEL-13",
            all.length > 64,
            "regional query expansion did not cross the former 64-field cache boundary"
        );
        GeoWorldQueryService.NaturalField low = all[0];
        GeoWorldQueryService.NaturalField high = all[64];
        service.clearCache();
        var lowResult = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                land[0],
                land[1],
                GeoWorldQueryService.Resolution.BLOCK_1,
                EnumSet.of(low)
            )
        );
        var highResult = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                land[0],
                land[1],
                GeoWorldQueryService.Resolution.BLOCK_1,
                EnumSet.of(high)
            )
        );
        require(
            "MODEL-13",
            lowResult.values().size() == 1 &&
                lowResult.values().containsKey(low) &&
                !lowResult.values().containsKey(high) &&
                highResult.values().size() == 1 &&
                highResult.values().containsKey(high) &&
                !highResult.values().containsKey(low) &&
                service.cacheSize() == 2,
            "natural query cache collided across field ordinals 0 and 64"
        );

        GeoWorldV1.RuntimeSample landSample = model.sample(land[0], land[1]);
        GeoWorldSettlementSuitability.Score miningSettlement =
            new GeoWorldSettlementSuitability.Score(
                0.55, 0.48, 0.08, 0.20,
                0.70, 0.55, 0.92, 0.55
            );
        GeoWorldNavigationSuitability.Score lowNavigation =
            new GeoWorldNavigationSuitability.Score(
                0.10, 0.05, 0.10, "none", "none"
            );
        GeoWorldMiningDistrict.Score strongMining =
            new GeoWorldMiningDistrict.Score(
                0.96, 0.80, 0.92, 0.88,
                0.94, 0.88,
                "iron", "mixed_district", Map.of()
            );
        var miningRegion = GeoWorldRegionalAdvantage.evaluate(
            landSample,
            miningSettlement,
            lowNavigation,
            strongMining
        );
        require(
            "MODEL-13",
            "mining".equals(miningRegion.dominantAdvantage()) &&
                "mining_region".equals(miningRegion.regionalClass()),
            "synthetic mining advantage site was misclassified"
        );
    }

    private static void verifyRegionalRegionIdentity(
        GeoWorldV1 model,
        Path path
    ) throws IOException {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        EnumSet<GeoWorldQueryService.NaturalField> fields = EnumSet.of(
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_KEY,
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_CELL_COUNT,
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_AREA_BLOCKS2,
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_CENTROID_X,
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_CENTROID_Z
        );

        GeoWorldQueryService.NaturalResult region = null;
        double regionX = 0;
        double regionZ = 0;
        search:
        for (int gz = 0; gz < geometry.height(); gz++) {
            double z = geometry.originZ() + gz * geometry.spacingZ();
            for (int gx = 0; gx < geometry.width(); gx++) {
                double x = geometry.originX() + gx * geometry.spacingX();
                var candidate = service.queryNatural(
                    new GeoWorldQueryService.QueryRequest(
                        x,
                        z,
                        GeoWorldQueryService.Resolution.BLOCK_1,
                        fields
                    )
                );
                Object key = candidate.values().get(
                    GeoWorldQueryService.NaturalField.REGIONAL_REGION_KEY
                );
                if (key instanceof String value && !"none".equals(value)) {
                    region = candidate;
                    regionX = x;
                    regionZ = z;
                    break search;
                }
            }
        }
        require(
            "MODEL-15",
            region != null,
            "locked model exposed no connected regional-advantage component"
        );

        String key = (String)region.values().get(
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_KEY
        );
        int cellCount = ((Number)region.values().get(
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_CELL_COUNT
        )).intValue();
        double area = ((Number)region.values().get(
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_AREA_BLOCKS2
        )).doubleValue();
        double centroidX = ((Number)region.values().get(
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_CENTROID_X
        )).doubleValue();
        double centroidZ = ((Number)region.values().get(
            GeoWorldQueryService.NaturalField.REGIONAL_REGION_CENTROID_Z
        )).doubleValue();
        double expectedArea =
            cellCount * geometry.spacingX() * geometry.spacingZ();
        require(
            "MODEL-15",
            region.unavailableFields().isEmpty() &&
                region.values().size() == fields.size() &&
                key.contains("-g") &&
                cellCount > 0 &&
                Double.isFinite(area) &&
                Math.abs(area - expectedArea) <=
                    Math.max(1e-9, expectedArea * 1e-12) &&
                Double.isFinite(centroidX) &&
                Double.isFinite(centroidZ) &&
                centroidX >= geometry.originX() &&
                centroidX <= geometry.originX() + geometry.spanXBlocks() &&
                centroidZ >= geometry.originZ() &&
                centroidZ <= geometry.originZ() + geometry.spanZBlocks(),
            "connected regional identity metadata was invalid: " + region.values()
        );

        var repeated = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                regionX,
                regionZ,
                GeoWorldQueryService.Resolution.BLOCK_1,
                fields
            )
        );
        require(
            "MODEL-15",
            region.equals(repeated),
            "same regional component query was not deterministic"
        );

        if (cellCount > 1) {
            int gx = (int)Math.round(
                (regionX - geometry.originX()) / geometry.spacingX()
            );
            int gz = (int)Math.round(
                (regionZ - geometry.originZ()) / geometry.spacingZ()
            );
            boolean foundAdjacent = false;
            int[][] offsets = {{-1, 0}, {1, 0}, {0, -1}, {0, 1}};
            for (int[] offset : offsets) {
                int nx = gx + offset[0];
                int nz = gz + offset[1];
                if (
                    nx < 0 || nx >= geometry.width() ||
                    nz < 0 || nz >= geometry.height()
                ) {
                    continue;
                }
                double x = geometry.originX() + nx * geometry.spacingX();
                double z = geometry.originZ() + nz * geometry.spacingZ();
                var adjacent = service.queryNatural(
                    new GeoWorldQueryService.QueryRequest(
                        x,
                        z,
                        GeoWorldQueryService.Resolution.BLOCK_1,
                        fields
                    )
                );
                if (key.equals(adjacent.values().get(
                    GeoWorldQueryService.NaturalField.REGIONAL_REGION_KEY
                ))) {
                    foundAdjacent = true;
                    break;
                }
            }
            require(
                "MODEL-15",
                foundAdjacent,
                "multi-cell regional component had no four-connected neighbor"
            );
        }

        GeoWorldV1 reopened = GeoWorldV1.load(path);
        var reopenedRegion = new GeoWorldQueryService(reopened).queryNatural(
            new GeoWorldQueryService.QueryRequest(
                regionX,
                regionZ,
                GeoWorldQueryService.Resolution.BLOCK_1,
                fields
            )
        );
        require(
            "MODEL-15",
            key.equals(reopenedRegion.values().get(
                GeoWorldQueryService.NaturalField.REGIONAL_REGION_KEY
            )) &&
                cellCount == ((Number)reopenedRegion.values().get(
                    GeoWorldQueryService.NaturalField.REGIONAL_REGION_CELL_COUNT
                )).intValue(),
            "regional identity changed after reopening the locked World Model"
        );

        double[] water = findMiningProbe(model, geometry, true);
        require(
            "MODEL-15",
            water != null,
            "model exposed no water probe for regional identity"
        );
        var waterRegion = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                water[0],
                water[1],
                GeoWorldQueryService.Resolution.BLOCK_1,
                fields
            )
        );
        require(
            "MODEL-15",
            "none".equals(waterRegion.values().get(
                GeoWorldQueryService.NaturalField.REGIONAL_REGION_KEY
            )) &&
                ((Number)waterRegion.values().get(
                    GeoWorldQueryService.NaturalField.REGIONAL_REGION_CELL_COUNT
                )).intValue() == 0 &&
                ((Number)waterRegion.values().get(
                    GeoWorldQueryService.NaturalField.REGIONAL_REGION_AREA_BLOCKS2
                )).doubleValue() == 0.0,
            "water was incorrectly assigned to a connected advantage region"
        );
    }

    private static void verifyTransportGeography(
        GeoWorldV1 model,
        Path path
    ) {
        GeoWorldQueryService service = new GeoWorldQueryService(model);
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        double[] land = findMiningProbe(model, geometry, false);
        double[] ocean = null;
        for (int gz = 0; gz < geometry.height() && ocean == null; gz++) {
            double z = geometry.originZ() + gz * geometry.spacingZ();
            for (int gx = 0; gx < geometry.width(); gx++) {
                double x = geometry.originX() + gx * geometry.spacingX();
                GeoWorldV1.RuntimeSample sample = model.sample(x, z);
                if (!sample.sourceLand() && "Ocean".equals(sample.waterKind())) {
                    ocean = new double[]{x, z};
                    break;
                }
            }
        }
        require(
            "MODEL-14",
            land != null && ocean != null,
            "model did not expose land and ocean transport probes"
        );

        EnumSet<GeoWorldQueryService.NaturalField> fields = EnumSet.of(
            GeoWorldQueryService.NaturalField.NATURAL_CORRIDOR_SUITABILITY,
            GeoWorldQueryService.NaturalField.RIVER_VALLEY_CORRIDOR,
            GeoWorldQueryService.NaturalField.HILL_PASS_CORRIDOR,
            GeoWorldQueryService.NaturalField.MOUNTAIN_PASS_CORRIDOR,
            GeoWorldQueryService.NaturalField.PLATEAU_ACCESS_CORRIDOR,
            GeoWorldQueryService.NaturalField.BASIN_GATEWAY_CORRIDOR,
            GeoWorldQueryService.NaturalField.LOW_DIVIDE_CORRIDOR,
            GeoWorldQueryService.NaturalField.STRAIT_ISTHMUS_POTENTIAL,
            GeoWorldQueryService.NaturalField.TRANSPORT_CHOKEPOINT_STRENGTH,
            GeoWorldQueryService.NaturalField.NATURAL_CORRIDOR_CLASS
        );
        List<GeoWorldQueryService.NaturalField> normalized = List.of(
            GeoWorldQueryService.NaturalField.NATURAL_CORRIDOR_SUITABILITY,
            GeoWorldQueryService.NaturalField.RIVER_VALLEY_CORRIDOR,
            GeoWorldQueryService.NaturalField.HILL_PASS_CORRIDOR,
            GeoWorldQueryService.NaturalField.MOUNTAIN_PASS_CORRIDOR,
            GeoWorldQueryService.NaturalField.PLATEAU_ACCESS_CORRIDOR,
            GeoWorldQueryService.NaturalField.BASIN_GATEWAY_CORRIDOR,
            GeoWorldQueryService.NaturalField.LOW_DIVIDE_CORRIDOR,
            GeoWorldQueryService.NaturalField.STRAIT_ISTHMUS_POTENTIAL,
            GeoWorldQueryService.NaturalField.TRANSPORT_CHOKEPOINT_STRENGTH
        );
        List<String> classes = List.of(
            "low_potential",
            "river_valley",
            "hill_pass",
            "mountain_pass",
            "plateau_access",
            "basin_gateway",
            "low_divide"
        );

        for (GeoWorldQueryService.Resolution resolution :
            GeoWorldQueryService.Resolution.values()) {
            var result = service.queryNatural(
                new GeoWorldQueryService.QueryRequest(
                    land[0], land[1], resolution, fields
                )
            );
            require(
                "MODEL-14",
                result.unavailableFields().isEmpty() &&
                    result.values().size() == fields.size(),
                "transport geography fields unavailable at " + resolution
            );
            for (GeoWorldQueryService.NaturalField field : normalized) {
                Object value = result.values().get(field);
                require(
                    "MODEL-14",
                    value instanceof Number number &&
                        Double.isFinite(number.doubleValue()) &&
                        number.doubleValue() >= 0 &&
                        number.doubleValue() <= 1,
                    field + " escaped [0,1] at " + resolution
                );
            }
            Object corridorClass = result.values().get(
                GeoWorldQueryService.NaturalField.NATURAL_CORRIDOR_CLASS
            );
            require(
                "MODEL-14",
                corridorClass instanceof String value &&
                    classes.contains(value),
                "land natural corridor class was invalid: " + corridorClass
            );
        }

        var oceanResult = service.queryNatural(
            new GeoWorldQueryService.QueryRequest(
                ocean[0],
                ocean[1],
                GeoWorldQueryService.Resolution.BLOCK_1,
                fields
            )
        );
        for (GeoWorldQueryService.NaturalField field : List.of(
            GeoWorldQueryService.NaturalField.NATURAL_CORRIDOR_SUITABILITY,
            GeoWorldQueryService.NaturalField.RIVER_VALLEY_CORRIDOR,
            GeoWorldQueryService.NaturalField.HILL_PASS_CORRIDOR,
            GeoWorldQueryService.NaturalField.MOUNTAIN_PASS_CORRIDOR,
            GeoWorldQueryService.NaturalField.PLATEAU_ACCESS_CORRIDOR,
            GeoWorldQueryService.NaturalField.BASIN_GATEWAY_CORRIDOR,
            GeoWorldQueryService.NaturalField.LOW_DIVIDE_CORRIDOR
        )) {
            require(
                "MODEL-14",
                ((Number)oceanResult.values().get(field)).doubleValue() == 0.0,
                "ocean gained a land corridor field: " + field
            );
        }
        require(
            "MODEL-14",
            "none".equals(oceanResult.values().get(
                GeoWorldQueryService.NaturalField.NATURAL_CORRIDOR_CLASS
            )),
            "ocean was assigned a land corridor class"
        );
    }

    private static void verifyInFlightCoalescing(
        GeoWorldV1 model,
        Path path
    ) throws Exception {
        GeoWorldQueryService service = new GeoWorldQueryService(model, 64);
        var request = new GeoWorldQueryService.QueryRequest(
            777.25,
            -333.75,
            GeoWorldQueryService.Resolution.BLOCK_1,
            EnumSet.allOf(GeoWorldQueryService.NaturalField.class)
        );
        int workers = 8;
        CountDownLatch ready = new CountDownLatch(workers);
        CountDownLatch start = new CountDownLatch(1);
        List<GeoWorldQueryService.NaturalResult> results =
            java.util.Collections.synchronizedList(new ArrayList<>());
        List<Throwable> errors =
            java.util.Collections.synchronizedList(new ArrayList<>());
        List<Thread> threads = new ArrayList<>();

        for (int i = 0; i < workers; i++) {
            Thread thread = new Thread(() -> {
                try {
                    ready.countDown();
                    start.await();
                    results.add(service.queryNatural(request));
                } catch (Throwable failure) {
                    errors.add(failure);
                }
            }, "geoworld-model-verifier-" + i);
            threads.add(thread);
            thread.start();
        }
        ready.await();
        start.countDown();
        for (Thread thread : threads) {
            thread.join();
        }

        require(
            "MODEL-07",
            errors.isEmpty() &&
                results.size() == workers &&
                service.cacheSize() == 1,
            "same in-flight query was not coalesced cleanly: errors=" + errors
        );
        GeoWorldQueryService.NaturalResult first = results.get(0);
        for (GeoWorldQueryService.NaturalResult result : results) {
            require(
                "MODEL-07",
                first.equals(result),
                "coalesced callers observed divergent Natural Model results"
            );
        }
    }

    private static void verifyQueryIdentity(GeoWorldV1 model, Path path) {
        GeoWorldV1.Manifest manifest = model.manifest();
        var result = new GeoWorldQueryService(model).queryNatural(
            new GeoWorldQueryService.QueryRequest(
                0.25,
                0.75,
                GeoWorldQueryService.Resolution.BLOCK_1,
                EnumSet.of(
                    GeoWorldQueryService.NaturalField.ELEVATION,
                    GeoWorldQueryService.NaturalField.BIOME_FAMILY,
                    GeoWorldQueryService.NaturalField.TERRAIN_VARIANT
                )
            )
        );
        require(
            "PROFILE-04",
            result.worldProfileVersion().equals(manifest.worldProfileVersion()) &&
                result.generationRulesVersion().equals(manifest.generationRulesVersion()) &&
                result.worldModelVersion().equals(manifest.worldModelVersion()) &&
                result.preset().equals(manifest.preset()),
            "Query API identity does not match locked model identity"
        );

        GeoWorldV1.RuntimeSample direct =
            model.sample(result.sampleX(), result.sampleZ());
        require(
            "PROFILE-04",
            Double.compare(
                ((Number) result.values().get(
                    GeoWorldQueryService.NaturalField.ELEVATION
                )).doubleValue(),
                direct.groundHeight()
            ) == 0 &&
                direct.biomeFamily().equals(
                    result.values().get(
                        GeoWorldQueryService.NaturalField.BIOME_FAMILY
                    )
                ) &&
                direct.terrainVariant().equals(
                    result.values().get(
                        GeoWorldQueryService.NaturalField.TERRAIN_VARIANT
                    )
                ),
            "Query API facts diverged from the locked Natural Model"
        );
    }

    private static double quantize(double coordinate, int resolution) {
        if (resolution <= 1) {
            return Math.floor(coordinate);
        }
        return Math.floor(coordinate / resolution) * resolution +
            resolution * 0.5;
    }

    private static boolean nonBlank(String value) {
        return value != null && !value.isBlank();
    }

    private static void require(String id, boolean condition, String message) {
        if (!condition) {
            throw new VerificationFailure(id, message);
        }
    }

    private static final class VerificationFailure extends AssertionError {
        private final String id;

        private VerificationFailure(String id, String message) {
            super(message);
            this.id = id;
        }
    }

    private static final class TrapLocalProvider
        implements GeoWorldQueryService.LocalRealityProvider {
        private final AtomicInteger loadedChecks = new AtomicInteger();
        private final AtomicInteger loadedQueries = new AtomicInteger();

        @Override
        public boolean isLoaded(int blockX, int blockZ) {
            loadedChecks.incrementAndGet();
            return false;
        }

        @Override
        public Optional<Map<String, Object>> queryLoaded(
            int blockX,
            int blockZ
        ) {
            loadedQueries.incrementAndGet();
            throw new AssertionError("queryLoaded must not be called for unloaded chunks");
        }
    }

    private static final class LoadedLocalProvider
        implements GeoWorldQueryService.LocalRealityProvider {
        private final AtomicInteger loadedQueries = new AtomicInteger();
        private long revision;
        private int lastBlockX = Integer.MIN_VALUE;
        private int lastBlockZ = Integer.MIN_VALUE;

        @Override
        public boolean isLoaded(int blockX, int blockZ) {
            return true;
        }

        @Override
        public Optional<Map<String, Object>> queryLoaded(
            int blockX,
            int blockZ
        ) {
            loadedQueries.incrementAndGet();
            lastBlockX = blockX;
            lastBlockZ = blockZ;
            return Optional.of(Map.of(
                "height", 88,
                "revision", revision
            ));
        }

        @Override
        public long cacheRevision() {
            return revision;
        }

        @Override
        public long cacheTtlMillis() {
            return 60_000L;
        }
    }
}
