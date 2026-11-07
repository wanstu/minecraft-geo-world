package geoworld.runtime;

import geoworld.runtime.GeoWorldQueryService.NaturalField;
import geoworld.runtime.GeoWorldQueryService.NaturalResult;
import geoworld.runtime.GeoWorldQueryService.QueryRequest;
import geoworld.runtime.GeoWorldQueryService.Resolution;
import geoworld.runtime.api.GeoWorldFacade;
import geoworld.runtime.api.WorldFacts;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.EnumSet;
import java.util.List;
import java.util.Set;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.atomic.AtomicReference;

/** Pure Java lifecycle and concurrent-reader checks; does not start Minecraft. */
public final class GeoWorldFacadeLifecycleVerifier {
    private GeoWorldFacadeLifecycleVerifier() {}

    public static void main(String[] args) throws Exception {
        Path input = args.length > 0 ? Path.of(args[0]) : Path.of("internal/goldenvectors/testdata");
        List<Path> paths;
        try (var stream = Files.list(input)) {
            paths = stream.filter(path -> path.getFileName().toString().endsWith(".geoworld")).sorted().toList();
        }
        if (paths.size() < 2) throw new AssertionError("lifecycle needs two different persisted models");
        GeoWorldV1 a = GeoWorldV1.load(paths.get(0));
        GeoWorldV1 b = GeoWorldV1.load(paths.get(paths.size() - 1));
        Object ownerA = new Object(), ownerB = new Object();
        QueryRequest request = new QueryRequest(-17.1, 33.9, Resolution.BLOCK_4,
            EnumSet.of(NaturalField.ELEVATION, NaturalField.ANNUAL_MEAN_TEMP_C));
        NaturalResult expectedA = new GeoWorldQueryService(a).queryNatural(request);
        NaturalResult expectedB = new GeoWorldQueryService(b).queryNatural(request);
        require(!expectedA.equals(expectedB), "model fixtures must distinguish A from B");
        require(GeoWorldFacade.currentWorld().isEmpty(), "initial state must be empty");
        require(GeoWorldFacade.apiVersion().equals(GeoWorldQueryService.API_VERSION) &&
            GeoWorldFacade.minSupportedApiVersion().equals(GeoWorldQueryService.API_VERSION), "API identity");
        try {
            GeoWorldFactsBinding.bind(ownerA, a);
            WorldFacts viewA = GeoWorldFacade.currentWorld().orElseThrow();
            require(viewA.queryNatural(request).equals(expectedA), "bind A");
            GeoWorldFactsBinding.unbind(ownerA);
            require(GeoWorldFacade.currentWorld().isEmpty(), "A -> empty on close");
            GeoWorldFactsBinding.bind(ownerB, b);
            WorldFacts viewB = GeoWorldFacade.currentWorld().orElseThrow();
            require(viewB != viewA && viewB.queryNatural(request).equals(expectedB), "empty -> B");
            GeoWorldFactsBinding.unbind(ownerA);
            require(GeoWorldFacade.currentWorld().orElseThrow() == viewB, "late stop A must preserve B");
            require(viewA.queryNatural(request).equals(expectedA), "retained view is a stable A snapshot");
            require(!viewA.supports(null), "null is not a supported field");
            verifyImmutableResult(viewB.queryNatural(request));
            GeoWorldFactsBinding.bind(ownerB, a);
            require(GeoWorldFacade.currentWorld().orElseThrow().queryNatural(request).equals(expectedA), "rebind same owner");
            require(viewB.queryNatural(request).equals(expectedB), "old B view must not mutate on rebind");
            GeoWorldFactsBinding.unbind(ownerB);
            GeoWorldFactsBinding.unbind(ownerB);
            require(GeoWorldFacade.currentWorld().isEmpty(), "repeated stop is idempotent");
            verifyConcurrentReaders(ownerA, ownerB, a, b, request, expectedA, expectedB);
            verifyPublicSurface();
        } finally {
            GeoWorldFactsBinding.unbind(ownerA);
            GeoWorldFactsBinding.unbind(ownerB);
        }
        require(GeoWorldFacade.currentWorld().isEmpty(), "final state must be empty");
        System.out.println("PASS Query facade lifecycle: A -> empty -> B, late stop, rebind, immutable snapshots, concurrent reads");
    }

    private static void verifyConcurrentReaders(Object ownerA, Object ownerB, GeoWorldV1 a, GeoWorldV1 b,
        QueryRequest request, NaturalResult expectedA, NaturalResult expectedB) throws Exception {
        AtomicReference<Throwable> failure = new AtomicReference<>();
        CountDownLatch start = new CountDownLatch(1);
        CountDownLatch firstSnapshots = new CountDownLatch(4);
        GeoWorldFactsBinding.bind(ownerA, a);
        List<Thread> readers = java.util.stream.IntStream.range(0, 4).mapToObj(index -> new Thread(() -> {
            try {
                start.await();
                require(GeoWorldFacade.currentWorld().orElseThrow().queryNatural(request).equals(expectedA),
                    "initial concurrent read of A");
                firstSnapshots.countDown();
                for (int i = 0; i < 1000; i++) {
                    var current = GeoWorldFacade.currentWorld();
                    if (current.isPresent()) {
                        NaturalResult result = current.get().queryNatural(request);
                        require(result.equals(expectedA) || result.equals(expectedB), "mixed-world snapshot from concurrent reader");
                    }
                }
            } catch (Throwable error) { failure.compareAndSet(null, error); }
        }, "GeoWorld-facts-reader-" + index)).toList();
        readers.forEach(Thread::start);
        start.countDown();
        try {
            require(firstSnapshots.await(10, java.util.concurrent.TimeUnit.SECONDS),
                "every reader must observe a nonempty world before mutations");
            for (int i = 0; i < 32; i++) {
                GeoWorldFactsBinding.bind(ownerA, a);
                GeoWorldFactsBinding.bind(ownerB, b);
                GeoWorldFactsBinding.unbind(ownerA);
                GeoWorldFactsBinding.unbind(ownerB);
            }
        } finally {
            for (Thread reader : readers) reader.join(10_000);
            for (Thread reader : readers) require(!reader.isAlive(), "reader blocked during bind/unbind");
        }
        if (failure.get() != null) throw new AssertionError("concurrent facade reader failed", failure.get());
    }

    private static void verifyImmutableResult(NaturalResult result) {
        try {
            result.values().put(NaturalField.ELEVATION, -1.0);
            throw new AssertionError("result values are mutable");
        } catch (UnsupportedOperationException expected) { }
        try {
            result.unavailableFields().add(NaturalField.ELEVATION);
            throw new AssertionError("unavailable fields are mutable");
        } catch (UnsupportedOperationException expected) { }
    }

    private static void verifyPublicSurface() {
        Set<String> methods = java.util.Arrays.stream(GeoWorldFacade.class.getDeclaredMethods())
            .filter(method -> java.lang.reflect.Modifier.isPublic(method.getModifiers()))
            .map(java.lang.reflect.Method::getName).collect(java.util.stream.Collectors.toSet());
        require(methods.equals(Set.of("apiVersion", "minSupportedApiVersion", "currentWorld")), "facade must expose only three read methods");
        require(java.util.Arrays.stream(WorldFacts.class.getMethods()).noneMatch(method ->
            method.getReturnType() == GeoWorldV1.class), "WorldFacts exposes the mutable model implementation");
    }

    private static void require(boolean condition, String message) {
        if (!condition) throw new AssertionError(message);
    }
}
