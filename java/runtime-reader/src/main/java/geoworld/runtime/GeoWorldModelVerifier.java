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
        "RUNTIME-01",
        "RUNTIME-02",
        "RUNTIME-03",
        "RUNTIME-04",
        "RUNTIME-05",
        "PROFILE-01",
        "PROFILE-02",
        "PROFILE-04",
        "PARAM-05"
    );

    public static void main(String[] args) throws Exception {
        Path input = args.length >= 1
            ? Path.of(args[0])
            : Path.of("internal/goldenvectors/testdata/runtime-v1.geoworld");

        List<Path> models = discoverModels(input);
        if (models.isEmpty()) {
            throw new AssertionError("no .geoworld models found under " + input);
        }

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
                    failures.get(id).add(path + ": unexpected verifier failure: " + failure);
                }
            }
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
        for (int i = 0; i < VERIFIED_IDS.size(); i++) {
            json.append("    \"").append(VERIFIED_IDS.get(i)).append("\"");
            if (i + 1 < VERIFIED_IDS.size()) {
                json.append(",");
            }
            json.append("\n");
        }
        json.append("  ]\n");
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
