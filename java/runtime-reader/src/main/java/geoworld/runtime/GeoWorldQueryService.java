package geoworld.runtime;

import java.util.ArrayList;
import java.util.Collections;
import java.util.EnumSet;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;
import java.util.Set;
import java.util.WeakHashMap;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;

/**
 * Stable, read-only query facade over the locked GeoWorld natural model.
 *
 * The service deliberately does not know how to load Minecraft chunks. Natural
 * queries are model-only; Local Reality and Dynamic State require caller
 * supplied providers and return unavailable when their data is not already
 * loaded. This keeps query traffic incapable of forcing world generation.
 */
public final class GeoWorldQueryService {
    public static final String API_VERSION = "geoworld-query-v1";
    private static final Map<GeoWorldV1, GeoWorldQueryService> SHARED =
        new WeakHashMap<>();

    public static GeoWorldQueryService shared(GeoWorldV1 model) {
        Objects.requireNonNull(model, "model");
        synchronized (SHARED) {
            return SHARED.computeIfAbsent(
                model,
                GeoWorldQueryService::new
            );
        }
    }

    public enum Resolution {
        BLOCK_1(1),
        BLOCK_4(4),
        BLOCK_16(16);

        private final int blocks;

        Resolution(int blocks) {
            this.blocks = blocks;
        }

        public int blocks() {
            return blocks;
        }
    }

    public enum NaturalField {
        ELEVATION,
        SLOPE,
        ASPECT_DEGREES,
        ECO_REGION,
        BIOME_FAMILY,
        TERRAIN_VARIANT,
        SURFACE_PROFILE,
        VEGETATION_PROFILE,
        WATER_KIND,
        WATER_SURFACE_ELEVATION,
        WATER_DEPTH,
        OCEAN_ELIGIBILITY,
        RIVER_WEIGHT,
        ANNUAL_MEAN_TEMP_C,
        ANNUAL_PRECIP_MM,
        MONSOON_INDEX,
        WIND_X,
        WIND_Z,

        // Fields backed by persisted natural-model facts. Legacy rc.1
        // models may still report individual fields as unavailable.
        WATERSHED_ID,
        FLOW_ACCUMULATION,
        RIVER_ID,
        RIVER_ORDER,
        NATURAL_FLOW_DIRECTION,
        DISTANCE_TO_RIVER,
        DISTANCE_TO_LAKE,
        DISTANCE_TO_COAST,
        DISTANCE_TO_OCEAN,
        GEOLOGICAL_REGION,
        ORE_PROVINCE,
        SPAWN_SUITABILITY,
        SETTLEMENT_BUILDABILITY,
        FRESH_WATER_ACCESS,
        FLOOD_RISK,
        AGRICULTURE_SUITABILITY,
        CLIMATE_COMFORT,
        TRANSPORT_ACCESS,
        RESOURCE_ACCESS,
        SETTLEMENT_SUITABILITY,
        NEAREST_RIVER_HALF_WIDTH,
        NEAREST_RIVER_GRADIENT,
        WATERWAY_NAVIGABILITY,
        PORT_SUITABILITY,
        NAVIGATION_BOTTLENECK_RISK,
        WATERWAY_CLASS,
        PORT_TYPE,
        MINING_SUITABILITY,
        SURFACE_MINING_SUITABILITY,
        SHALLOW_MINING_SUITABILITY,
        DEEP_MINING_SUITABILITY,
        INDUSTRIAL_MINERAL_VALUE,
        MINERAL_RESOURCE_DIVERSITY,
        DOMINANT_MINERAL_RESOURCE,
        MINING_DISTRICT_CLASS,
        MINERAL_RESOURCE_PROFILE,
        CIVILIZATION_AGRICULTURAL_SETTLEMENT_SUITABILITY,
        CIVILIZATION_PORT_CITY_SUITABILITY,
        CIVILIZATION_MINING_TOWN_SUITABILITY,
        CIVILIZATION_TRANSPORT_NODE_SUITABILITY,
        CIVILIZATION_NODE_SUITABILITY,
        CIVILIZATION_SPECIALIZATION_STRENGTH,
        CIVILIZATION_NODE_CLASS,
        REGIONAL_BUILDING_ADVANTAGE,
        REGIONAL_AGRICULTURE_ADVANTAGE,
        REGIONAL_PORT_ADVANTAGE,
        REGIONAL_MINING_ADVANTAGE,
        REGIONAL_TRANSPORT_ADVANTAGE,
        REGIONAL_SPECIALIZATION_STRENGTH,
        REGIONAL_ADVANTAGE_OVERLAP_RISK,
        REGIONAL_HIGH_ADVANTAGE_COUNT,
        REGIONAL_DOMINANT_ADVANTAGE,
        REGIONAL_ADVANTAGE_CLASS,
        REGIONAL_REGION_KEY,
        REGIONAL_REGION_CELL_COUNT,
        REGIONAL_REGION_AREA_BLOCKS2,
        REGIONAL_REGION_CENTROID_X,
        REGIONAL_REGION_CENTROID_Z,
        NATURAL_CORRIDOR_SUITABILITY,
        RIVER_VALLEY_CORRIDOR,
        HILL_PASS_CORRIDOR,
        MOUNTAIN_PASS_CORRIDOR,
        PLATEAU_ACCESS_CORRIDOR,
        BASIN_GATEWAY_CORRIDOR,
        LOW_DIVIDE_CORRIDOR,
        STRAIT_ISTHMUS_POTENTIAL,
        TRANSPORT_CHOKEPOINT_STRENGTH,
        NATURAL_CORRIDOR_CLASS
    }

    public record QueryRequest(
        double worldX,
        double worldZ,
        Resolution resolution,
        Set<NaturalField> fields
    ) {
        public QueryRequest {
            Objects.requireNonNull(resolution, "resolution");
            Objects.requireNonNull(fields, "fields");
            fields = Set.copyOf(fields);
        }

        public static QueryRequest all(
            double worldX,
            double worldZ,
            Resolution resolution
        ) {
            return new QueryRequest(
                worldX,
                worldZ,
                resolution,
                EnumSet.allOf(NaturalField.class)
            );
        }
    }

    public record NaturalResult(
        String apiVersion,
        String worldProfileVersion,
        String generationRulesVersion,
        String worldModelVersion,
        String preset,
        double sampleX,
        double sampleZ,
        Resolution resolution,
        Map<NaturalField, Object> values,
        Set<NaturalField> unavailableFields
    ) {}

    /**
     * Local Reality is deliberately opaque to runtime-reader. Minecraft
     * adapters may expose block/fluid/biome facts only for already-loaded
     * chunks through this provider.
     */
    public interface LocalRealityProvider {
        boolean isLoaded(int blockX, int blockZ);
        Optional<Map<String, Object>> queryLoaded(int blockX, int blockZ);

        /**
         * Increment when caller-visible loaded-world facts change materially.
         * Providers that cannot expose a revision may keep the default and rely
         * on the short cache TTL.
         */
        default long cacheRevision() {
            return 0L;
        }

        /**
         * Local Reality is intentionally short lived: it describes the loaded
         * Minecraft world, not the immutable Natural Model.
         */
        default long cacheTtlMillis() {
            return 250L;
        }
    }

    public record LocalRealityResult(
        boolean available,
        int blockX,
        int blockZ,
        Map<String, Object> values
    ) {
        public static LocalRealityResult unavailable(int x, int z) {
            return new LocalRealityResult(false, x, z, Map.of());
        }
    }

    /**
     * Dynamic state (snow cover, flood state, seasonal state, etc.) must not be
     * inferred from natural-model values. A provider explicitly decides whether
     * a dynamic field currently exists.
     */
    public interface DynamicStateProvider {
        Optional<Object> queryLoaded(String field, int blockX, int blockZ);
    }

    private record CacheKey(
        long xBits,
        long zBits,
        Resolution resolution,
        Set<NaturalField> fields
    ) {}

    private static final class LocalCacheKey {
        private final LocalRealityProvider provider;
        private final int blockX;
        private final int blockZ;

        private LocalCacheKey(
            LocalRealityProvider provider,
            int blockX,
            int blockZ
        ) {
            this.provider = provider;
            this.blockX = blockX;
            this.blockZ = blockZ;
        }

        @Override
        public boolean equals(Object other) {
            if (!(other instanceof LocalCacheKey key)) return false;
            return provider == key.provider &&
                blockX == key.blockX &&
                blockZ == key.blockZ;
        }

        @Override
        public int hashCode() {
            int h = System.identityHashCode(provider);
            h = 31 * h + blockX;
            h = 31 * h + blockZ;
            return h;
        }
    }

    private record LocalCacheEntry(
        long revision,
        long expiresAtNanos,
        LocalRealityResult result
    ) {}

    /**
     * Read-only diagnostic counters used by acceptance verifiers to prove that
     * expensive natural-model field groups are evaluated only when requested.
     */
    public record ComputationStats(
        long climateQueries,
        long hydrologyQueries,
        long naturalFeatureQueries
    ) {}

    private final GeoWorldV1 model;
    private final int maxCacheEntries;
    private final int maxLocalCacheEntries;
    private final Map<CacheKey, NaturalResult> cache;
    private final Map<LocalCacheKey, LocalCacheEntry> localCache;
    private final ConcurrentHashMap<CacheKey, CompletableFuture<NaturalResult>>
        inFlight = new ConcurrentHashMap<>();
    private final AtomicLong climateQueries = new AtomicLong();
    private final AtomicLong hydrologyQueries = new AtomicLong();
    private final AtomicLong naturalFeatureQueries = new AtomicLong();
    private final GeoWorldRegionalAdvantageRegions regionalAdvantageRegions;

    public GeoWorldQueryService(GeoWorldV1 model) {
        this(model, 8192);
    }

    public GeoWorldQueryService(GeoWorldV1 model, int maxCacheEntries) {
        this.model = Objects.requireNonNull(model, "model");
        if (maxCacheEntries < 64) {
            throw new IllegalArgumentException("maxCacheEntries must be >= 64");
        }
        this.maxCacheEntries = maxCacheEntries;
        this.maxLocalCacheEntries = Math.max(64, maxCacheEntries / 4);
        this.cache = Collections.synchronizedMap(
            new LinkedHashMap<>(256, 0.75f, true) {
                @Override
                protected boolean removeEldestEntry(
                    Map.Entry<CacheKey, NaturalResult> eldest
                ) {
                    return size() > GeoWorldQueryService.this.maxCacheEntries;
                }
            }
        );
        this.localCache = Collections.synchronizedMap(
            new LinkedHashMap<>(128, 0.75f, true) {
                @Override
                protected boolean removeEldestEntry(
                    Map.Entry<LocalCacheKey, LocalCacheEntry> eldest
                ) {
                    return size() >
                        GeoWorldQueryService.this.maxLocalCacheEntries;
                }
            }
        );
        this.regionalAdvantageRegions =
            new GeoWorldRegionalAdvantageRegions(
                model.manifest().geometry(),
                this::regionalAdvantageAtMacroNode
            );
    }

    public GeoWorldV1 model() {
        return model;
    }

    public NaturalResult queryNatural(QueryRequest request) {
        Objects.requireNonNull(request, "request");
        double x = quantize(request.worldX(), request.resolution().blocks());
        double z = quantize(request.worldZ(), request.resolution().blocks());
        CacheKey key = new CacheKey(
            Double.doubleToLongBits(x),
            Double.doubleToLongBits(z),
            request.resolution(),
            request.fields()
        );

        NaturalResult cached = cache.get(key);
        if (cached != null) {
            return cached;
        }

        CompletableFuture<NaturalResult> mine = new CompletableFuture<>();
        CompletableFuture<NaturalResult> existing = inFlight.putIfAbsent(key, mine);
        if (existing != null) {
            return existing.join();
        }

        try {
            NaturalResult result = computeNatural(
                x,
                z,
                request.resolution(),
                request.fields()
            );
            cache.put(key, result);
            mine.complete(result);
            return result;
        } catch (RuntimeException error) {
            mine.completeExceptionally(error);
            throw error;
        } finally {
            inFlight.remove(key, mine);
        }
    }

    public LocalRealityResult queryLocalReality(
        int blockX,
        int blockZ,
        LocalRealityProvider provider
    ) {
        Objects.requireNonNull(provider, "provider");
        LocalCacheKey key = new LocalCacheKey(provider, blockX, blockZ);

        // This check is deliberately first. Querying Local Reality must never
        // become a reason for a provider to load or generate a chunk.
        if (!provider.isLoaded(blockX, blockZ)) {
            localCache.remove(key);
            return LocalRealityResult.unavailable(blockX, blockZ);
        }

        long now = System.nanoTime();
        long revision = provider.cacheRevision();
        LocalCacheEntry cached = localCache.get(key);
        if (
            cached != null &&
            cached.revision() == revision &&
            cached.expiresAtNanos() >= now
        ) {
            return cached.result();
        }

        Optional<Map<String, Object>> values = provider.queryLoaded(blockX, blockZ);
        LocalRealityResult result = values
            .map(value -> new LocalRealityResult(
                true,
                blockX,
                blockZ,
                Map.copyOf(value)
            ))
            .orElseGet(() ->
                LocalRealityResult.unavailable(blockX, blockZ)
            );

        long ttlMillis = Math.max(0L, provider.cacheTtlMillis());
        if (ttlMillis > 0L) {
            long ttlNanos;
            try {
                ttlNanos = Math.multiplyExact(ttlMillis, 1_000_000L);
            } catch (ArithmeticException overflow) {
                ttlNanos = Long.MAX_VALUE;
            }
            long expiresAt = ttlNanos == Long.MAX_VALUE ||
                Long.MAX_VALUE - now < ttlNanos
                ? Long.MAX_VALUE
                : now + ttlNanos;
            localCache.put(
                key,
                new LocalCacheEntry(revision, expiresAt, result)
            );
        } else {
            localCache.remove(key);
        }
        return result;
    }

    public Optional<Object> queryDynamicState(
        String field,
        int blockX,
        int blockZ,
        DynamicStateProvider provider
    ) {
        Objects.requireNonNull(field, "field");
        Objects.requireNonNull(provider, "provider");
        return provider.queryLoaded(field, blockX, blockZ);
    }

    public int cacheSize() {
        synchronized (cache) {
            return cache.size();
        }
    }

    public int localRealityCacheSize() {
        synchronized (localCache) {
            return localCache.size();
        }
    }

    public ComputationStats computationStats() {
        return new ComputationStats(
            climateQueries.get(),
            hydrologyQueries.get(),
            naturalFeatureQueries.get()
        );
    }

    public void resetComputationStats() {
        climateQueries.set(0L);
        hydrologyQueries.set(0L);
        naturalFeatureQueries.set(0L);
    }

    public void clearCache() {
        cache.clear();
    }

    public void clearLocalRealityCache() {
        localCache.clear();
    }

    public void clearAllCaches() {
        cache.clear();
        localCache.clear();
    }

    private NaturalResult computeNatural(
        double x,
        double z,
        Resolution resolution,
        Set<NaturalField> fields
    ) {
        GeoWorldV1.RuntimeSample sample = model.sample(x, z);
        Map<NaturalField, Object> values = new LinkedHashMap<>();
        EnumSet<NaturalField> unavailable = EnumSet.noneOf(NaturalField.class);

        GeoWorldV1.ClimateSample climate = null;
        if (needsClimate(fields)) {
            climateQueries.incrementAndGet();
            climate = model.sampleClimate(x, z);
        }
        GeoWorldV1.HydrologySample hydrology = null;
        if (needsHydrology(fields)) {
            hydrologyQueries.incrementAndGet();
            hydrology = model.sampleHydrology(x, z);
        }
        GeoWorldV1.NaturalFeatureSample features = null;
        if (needsNaturalFeatures(fields)) {
            naturalFeatureQueries.incrementAndGet();
            features = model.sampleNaturalFeatures(x, z);
        }
        GeoWorldSettlementSuitability.Score settlement = null;
        if (needsSettlement(fields)) {
            settlement = GeoWorldSettlementSuitability.evaluate(
                sample,
                slopeAt(x, z, Math.max(4, resolution.blocks())),
                climate,
                features
            );
        }
        GeoWorldV1.NavigationFeatureSample navigationFeatures = null;
        if (needsNavigationFeatures(fields)) {
            navigationFeatures = model.sampleNavigationFeatures(x, z);
        }
        GeoWorldNavigationSuitability.Score navigation = null;
        if (needsNavigationScore(fields)) {
            navigation = GeoWorldNavigationSuitability.evaluate(
                sample,
                features,
                navigationFeatures,
                settlement
            );
        }
        GeoWorldTransportGeography.Score transportGeography = null;
        if (needsTransportGeography(fields)) {
            transportGeography = GeoWorldTransportGeography.evaluate(
                model,
                sample,
                hydrology,
                settlement,
                navigation,
                x,
                z
            );
        }
        GeoWorldMiningDistrict.Score mining = null;
        if (needsMining(fields)) {
            mining = GeoWorldMiningDistrict.evaluate(
                model,
                sample,
                x,
                z
            );
        }
        GeoWorldCivilizationSuitability.Score civilization = null;
        if (needsCivilization(fields)) {
            civilization = GeoWorldCivilizationSuitability.evaluate(
                sample,
                settlement,
                navigation,
                mining
            );
        }
        GeoWorldRegionalAdvantage.Score regionalAdvantage = null;
        if (needsRegionalAdvantage(fields)) {
            regionalAdvantage = GeoWorldRegionalAdvantage.evaluate(
                sample,
                settlement,
                navigation,
                mining
            );
        }
        GeoWorldRegionalAdvantageRegions.Membership regionalRegion = null;
        if (needsRegionalRegion(fields)) {
            regionalRegion = regionalAdvantageRegions.membershipAt(x, z);
        }

        for (NaturalField field : fields) {
            switch (field) {
                case ELEVATION -> values.put(field, sample.groundHeight());
                case SLOPE -> values.put(
                    field,
                    slopeAt(x, z, resolution.blocks())
                );
                case ASPECT_DEGREES -> values.put(
                    field,
                    aspectAt(x, z, resolution.blocks())
                );
                case ECO_REGION -> values.put(field, sample.ecoRegion());
                case BIOME_FAMILY -> values.put(field, sample.biomeFamily());
                case TERRAIN_VARIANT -> values.put(field, sample.terrainVariant());
                case SURFACE_PROFILE -> values.put(field, sample.surfaceProfile());
                case VEGETATION_PROFILE -> values.put(field, sample.vegetationProfile());
                case WATER_KIND -> values.put(field, sample.waterKind());
                case WATER_SURFACE_ELEVATION -> {
                    if (sample.waterSurfaceDefined()) {
                        values.put(field, sample.waterSurfaceElevation());
                    } else {
                        unavailable.add(field);
                    }
                }
                case WATER_DEPTH -> values.put(field, sample.waterDepth());
                case OCEAN_ELIGIBILITY -> values.put(field, sample.oceanEligibility());
                case RIVER_WEIGHT -> values.put(field, sample.riverWeight());
                case ANNUAL_MEAN_TEMP_C -> putClimate(
                    values, unavailable, field, climate,
                    climate == null ? 0 : climate.annualMeanTempC()
                );
                case ANNUAL_PRECIP_MM -> putClimate(
                    values, unavailable, field, climate,
                    climate == null ? 0 : climate.annualPrecipMM()
                );
                case MONSOON_INDEX -> putClimate(
                    values, unavailable, field, climate,
                    climate == null ? 0 : climate.monsoonIndex()
                );
                case WIND_X -> putClimate(
                    values, unavailable, field, climate,
                    climate == null ? 0 : climate.windX()
                );
                case WIND_Z -> putClimate(
                    values, unavailable, field, climate,
                    climate == null ? 0 : climate.windZ()
                );
                case WATERSHED_ID -> {
                    if (
                        hydrology != null &&
                        hydrology.available() &&
                        hydrology.watershedId() > 0
                    ) {
                        values.put(field, hydrology.watershedId());
                    } else {
                        unavailable.add(field);
                    }
                }
                case FLOW_ACCUMULATION -> {
                    if (hydrology != null && hydrology.available()) {
                        values.put(field, hydrology.flowAccumulation());
                    } else {
                        unavailable.add(field);
                    }
                }
                case RIVER_ORDER -> {
                    if (hydrology != null && hydrology.available()) {
                        values.put(field, hydrology.riverOrder());
                    } else {
                        unavailable.add(field);
                    }
                }
                case NATURAL_FLOW_DIRECTION -> {
                    if (
                        hydrology != null &&
                        hydrology.available() &&
                        hydrology.flowDirectionX() != null &&
                        hydrology.flowDirectionZ() != null
                    ) {
                        values.put(
                            field,
                            Map.of(
                                "x", hydrology.flowDirectionX(),
                                "z", hydrology.flowDirectionZ()
                            )
                        );
                    } else {
                        unavailable.add(field);
                    }
                }
                case RIVER_ID -> {
                    if (features != null && features.riverId() != null) {
                        values.put(field, features.riverId());
                    } else {
                        unavailable.add(field);
                    }
                }
                case DISTANCE_TO_RIVER -> {
                    if (features != null && features.distanceToRiver() != null) {
                        values.put(field, features.distanceToRiver());
                    } else {
                        unavailable.add(field);
                    }
                }
                case DISTANCE_TO_LAKE -> {
                    if (features != null && features.distanceToLake() != null) {
                        values.put(field, features.distanceToLake());
                    } else {
                        unavailable.add(field);
                    }
                }
                case DISTANCE_TO_COAST -> {
                    if (features != null && features.distanceToCoast() != null) {
                        values.put(field, features.distanceToCoast());
                    } else {
                        unavailable.add(field);
                    }
                }
                case DISTANCE_TO_OCEAN -> {
                    if (features != null && features.distanceToOcean() != null) {
                        values.put(field, features.distanceToOcean());
                    } else {
                        unavailable.add(field);
                    }
                }
                case GEOLOGICAL_REGION -> values.put(
                    field,
                    GeoWorldGeology.geologicalRegion(model, x, z)
                );
                case ORE_PROVINCE -> values.put(
                    field,
                    GeoWorldGeology.oreProvince(model, x, z)
                );
                case SPAWN_SUITABILITY -> values.put(
                    field,
                    spawnSuitability(sample, x, z, resolution.blocks())
                );
                case SETTLEMENT_BUILDABILITY -> values.put(
                    field,
                    settlement.buildability()
                );
                case FRESH_WATER_ACCESS -> values.put(
                    field,
                    settlement.freshWaterAccess()
                );
                case FLOOD_RISK -> values.put(
                    field,
                    settlement.floodRisk()
                );
                case AGRICULTURE_SUITABILITY -> values.put(
                    field,
                    settlement.agricultureSuitability()
                );
                case CLIMATE_COMFORT -> values.put(
                    field,
                    settlement.climateComfort()
                );
                case TRANSPORT_ACCESS -> values.put(
                    field,
                    settlement.transportAccess()
                );
                case RESOURCE_ACCESS -> values.put(
                    field,
                    settlement.resourceAccess()
                );
                case SETTLEMENT_SUITABILITY -> values.put(
                    field,
                    settlement.settlementSuitability()
                );
                case NEAREST_RIVER_HALF_WIDTH -> {
                    if (
                        navigationFeatures != null &&
                        navigationFeatures.available() &&
                        navigationFeatures.riverHalfWidth() != null
                    ) {
                        values.put(
                            field,
                            navigationFeatures.riverHalfWidth()
                        );
                    } else {
                        unavailable.add(field);
                    }
                }
                case NEAREST_RIVER_GRADIENT -> {
                    if (
                        navigationFeatures != null &&
                        navigationFeatures.available() &&
                        navigationFeatures.riverGradient() != null
                    ) {
                        values.put(
                            field,
                            navigationFeatures.riverGradient()
                        );
                    } else {
                        unavailable.add(field);
                    }
                }
                case WATERWAY_NAVIGABILITY -> values.put(
                    field,
                    navigation.waterwayNavigability()
                );
                case PORT_SUITABILITY -> values.put(
                    field,
                    navigation.portSuitability()
                );
                case NAVIGATION_BOTTLENECK_RISK -> values.put(
                    field,
                    navigation.navigationBottleneckRisk()
                );
                case WATERWAY_CLASS -> values.put(
                    field,
                    navigation.waterwayClass()
                );
                case PORT_TYPE -> values.put(
                    field,
                    navigation.portType()
                );
                case MINING_SUITABILITY -> values.put(
                    field,
                    mining.miningSuitability()
                );
                case SURFACE_MINING_SUITABILITY -> values.put(
                    field,
                    mining.surfaceMiningSuitability()
                );
                case SHALLOW_MINING_SUITABILITY -> values.put(
                    field,
                    mining.shallowMiningSuitability()
                );
                case DEEP_MINING_SUITABILITY -> values.put(
                    field,
                    mining.deepMiningSuitability()
                );
                case INDUSTRIAL_MINERAL_VALUE -> values.put(
                    field,
                    mining.industrialValue()
                );
                case MINERAL_RESOURCE_DIVERSITY -> values.put(
                    field,
                    mining.resourceDiversity()
                );
                case DOMINANT_MINERAL_RESOURCE -> values.put(
                    field,
                    mining.dominantResource()
                );
                case MINING_DISTRICT_CLASS -> values.put(
                    field,
                    mining.miningClass()
                );
                case MINERAL_RESOURCE_PROFILE -> values.put(
                    field,
                    mining.resourceProfile()
                );
                case CIVILIZATION_AGRICULTURAL_SETTLEMENT_SUITABILITY ->
                    values.put(
                        field,
                        civilization.agriculturalSettlementSuitability()
                    );
                case CIVILIZATION_PORT_CITY_SUITABILITY -> values.put(
                    field,
                    civilization.portCitySuitability()
                );
                case CIVILIZATION_MINING_TOWN_SUITABILITY -> values.put(
                    field,
                    civilization.miningTownSuitability()
                );
                case CIVILIZATION_TRANSPORT_NODE_SUITABILITY -> values.put(
                    field,
                    civilization.transportNodeSuitability()
                );
                case CIVILIZATION_NODE_SUITABILITY -> values.put(
                    field,
                    civilization.nodeSuitability()
                );
                case CIVILIZATION_SPECIALIZATION_STRENGTH -> values.put(
                    field,
                    civilization.specializationStrength()
                );
                case CIVILIZATION_NODE_CLASS -> values.put(
                    field,
                    civilization.nodeClass()
                );
                case REGIONAL_BUILDING_ADVANTAGE -> values.put(
                    field,
                    regionalAdvantage.buildingAdvantage()
                );
                case REGIONAL_AGRICULTURE_ADVANTAGE -> values.put(
                    field,
                    regionalAdvantage.agricultureAdvantage()
                );
                case REGIONAL_PORT_ADVANTAGE -> values.put(
                    field,
                    regionalAdvantage.portAdvantage()
                );
                case REGIONAL_MINING_ADVANTAGE -> values.put(
                    field,
                    regionalAdvantage.miningAdvantage()
                );
                case REGIONAL_TRANSPORT_ADVANTAGE -> values.put(
                    field,
                    regionalAdvantage.transportAdvantage()
                );
                case REGIONAL_SPECIALIZATION_STRENGTH -> values.put(
                    field,
                    regionalAdvantage.specializationStrength()
                );
                case REGIONAL_ADVANTAGE_OVERLAP_RISK -> values.put(
                    field,
                    regionalAdvantage.advantageOverlapRisk()
                );
                case REGIONAL_HIGH_ADVANTAGE_COUNT -> values.put(
                    field,
                    regionalAdvantage.highAdvantageCount()
                );
                case REGIONAL_DOMINANT_ADVANTAGE -> values.put(
                    field,
                    regionalAdvantage.dominantAdvantage()
                );
                case REGIONAL_ADVANTAGE_CLASS -> values.put(
                    field,
                    regionalAdvantage.regionalClass()
                );
                case REGIONAL_REGION_KEY -> values.put(
                    field,
                    regionalRegion.key()
                );
                case REGIONAL_REGION_CELL_COUNT -> values.put(
                    field,
                    regionalRegion.cellCount()
                );
                case REGIONAL_REGION_AREA_BLOCKS2 -> values.put(
                    field,
                    regionalRegion.areaBlocks2()
                );
                case REGIONAL_REGION_CENTROID_X -> values.put(
                    field,
                    regionalRegion.centroidX()
                );
                case REGIONAL_REGION_CENTROID_Z -> values.put(
                    field,
                    regionalRegion.centroidZ()
                );
                case NATURAL_CORRIDOR_SUITABILITY -> values.put(
                    field,
                    transportGeography.corridorSuitability()
                );
                case RIVER_VALLEY_CORRIDOR -> values.put(
                    field,
                    transportGeography.riverValleyCorridor()
                );
                case HILL_PASS_CORRIDOR -> values.put(
                    field,
                    transportGeography.hillPassCorridor()
                );
                case MOUNTAIN_PASS_CORRIDOR -> values.put(
                    field,
                    transportGeography.mountainPassCorridor()
                );
                case PLATEAU_ACCESS_CORRIDOR -> values.put(
                    field,
                    transportGeography.plateauAccessCorridor()
                );
                case BASIN_GATEWAY_CORRIDOR -> values.put(
                    field,
                    transportGeography.basinGatewayCorridor()
                );
                case LOW_DIVIDE_CORRIDOR -> values.put(
                    field,
                    transportGeography.lowDivideCorridor()
                );
                case STRAIT_ISTHMUS_POTENTIAL -> values.put(
                    field,
                    transportGeography.straitIsthmusPotential()
                );
                case TRANSPORT_CHOKEPOINT_STRENGTH -> values.put(
                    field,
                    transportGeography.chokepointStrength()
                );
                case NATURAL_CORRIDOR_CLASS -> values.put(
                    field,
                    transportGeography.corridorClass()
                );
                default -> unavailable.add(field);
            }
        }

        GeoWorldV1.WorldProfile profile = model.worldProfile();
        return new NaturalResult(
            API_VERSION,
            profile.worldProfileVersion(),
            profile.generationRulesVersion(),
            profile.worldModelVersion(),
            profile.preset(),
            x,
            z,
            resolution,
            Map.copyOf(values),
            Set.copyOf(unavailable)
        );
    }

    private static void putClimate(
        Map<NaturalField, Object> values,
        Set<NaturalField> unavailable,
        NaturalField field,
        GeoWorldV1.ClimateSample climate,
        double value
    ) {
        if (climate == null || !climate.available()) {
            unavailable.add(field);
        } else {
            values.put(field, value);
        }
    }

    private boolean needsClimate(Set<NaturalField> fields) {
        return fields.contains(NaturalField.ANNUAL_MEAN_TEMP_C) ||
            fields.contains(NaturalField.ANNUAL_PRECIP_MM) ||
            fields.contains(NaturalField.MONSOON_INDEX) ||
            fields.contains(NaturalField.WIND_X) ||
            fields.contains(NaturalField.WIND_Z) ||
            needsSettlement(fields);
    }

    private boolean needsHydrology(Set<NaturalField> fields) {
        return fields.contains(NaturalField.WATERSHED_ID) ||
            fields.contains(NaturalField.FLOW_ACCUMULATION) ||
            fields.contains(NaturalField.RIVER_ORDER) ||
            fields.contains(NaturalField.NATURAL_FLOW_DIRECTION) ||
            needsTransportGeography(fields);
    }

    private boolean needsNaturalFeatures(Set<NaturalField> fields) {
        return fields.contains(NaturalField.RIVER_ID) ||
            fields.contains(NaturalField.DISTANCE_TO_RIVER) ||
            fields.contains(NaturalField.DISTANCE_TO_LAKE) ||
            fields.contains(NaturalField.DISTANCE_TO_COAST) ||
            fields.contains(NaturalField.DISTANCE_TO_OCEAN) ||
            needsSettlement(fields);
    }

    private boolean needsSettlement(Set<NaturalField> fields) {
        return fields.contains(NaturalField.SETTLEMENT_BUILDABILITY) ||
            fields.contains(NaturalField.FRESH_WATER_ACCESS) ||
            fields.contains(NaturalField.FLOOD_RISK) ||
            fields.contains(NaturalField.AGRICULTURE_SUITABILITY) ||
            fields.contains(NaturalField.CLIMATE_COMFORT) ||
            fields.contains(NaturalField.TRANSPORT_ACCESS) ||
            fields.contains(NaturalField.RESOURCE_ACCESS) ||
            fields.contains(NaturalField.SETTLEMENT_SUITABILITY) ||
            needsNavigationScore(fields) ||
            needsCivilization(fields) ||
            needsRegionalAdvantage(fields) ||
            needsTransportGeography(fields);
    }

    private boolean needsNavigationFeatures(Set<NaturalField> fields) {
        return fields.contains(NaturalField.NEAREST_RIVER_HALF_WIDTH) ||
            fields.contains(NaturalField.NEAREST_RIVER_GRADIENT) ||
            needsNavigationScore(fields);
    }

    private boolean needsNavigationScore(Set<NaturalField> fields) {
        return fields.contains(NaturalField.WATERWAY_NAVIGABILITY) ||
            fields.contains(NaturalField.PORT_SUITABILITY) ||
            fields.contains(NaturalField.NAVIGATION_BOTTLENECK_RISK) ||
            fields.contains(NaturalField.WATERWAY_CLASS) ||
            fields.contains(NaturalField.PORT_TYPE) ||
            needsCivilization(fields) ||
            needsRegionalAdvantage(fields) ||
            needsTransportGeography(fields);
    }

    private boolean needsMining(Set<NaturalField> fields) {
        return fields.contains(NaturalField.MINING_SUITABILITY) ||
            fields.contains(NaturalField.SURFACE_MINING_SUITABILITY) ||
            fields.contains(NaturalField.SHALLOW_MINING_SUITABILITY) ||
            fields.contains(NaturalField.DEEP_MINING_SUITABILITY) ||
            fields.contains(NaturalField.INDUSTRIAL_MINERAL_VALUE) ||
            fields.contains(NaturalField.MINERAL_RESOURCE_DIVERSITY) ||
            fields.contains(NaturalField.DOMINANT_MINERAL_RESOURCE) ||
            fields.contains(NaturalField.MINING_DISTRICT_CLASS) ||
            fields.contains(NaturalField.MINERAL_RESOURCE_PROFILE) ||
            needsCivilization(fields) ||
            needsRegionalAdvantage(fields);
    }

    private boolean needsCivilization(Set<NaturalField> fields) {
        return fields.contains(
            NaturalField.CIVILIZATION_AGRICULTURAL_SETTLEMENT_SUITABILITY
        ) ||
            fields.contains(NaturalField.CIVILIZATION_PORT_CITY_SUITABILITY) ||
            fields.contains(NaturalField.CIVILIZATION_MINING_TOWN_SUITABILITY) ||
            fields.contains(NaturalField.CIVILIZATION_TRANSPORT_NODE_SUITABILITY) ||
            fields.contains(NaturalField.CIVILIZATION_NODE_SUITABILITY) ||
            fields.contains(NaturalField.CIVILIZATION_SPECIALIZATION_STRENGTH) ||
            fields.contains(NaturalField.CIVILIZATION_NODE_CLASS);
    }

    private boolean needsRegionalAdvantage(Set<NaturalField> fields) {
        return fields.contains(NaturalField.REGIONAL_BUILDING_ADVANTAGE) ||
            fields.contains(NaturalField.REGIONAL_AGRICULTURE_ADVANTAGE) ||
            fields.contains(NaturalField.REGIONAL_PORT_ADVANTAGE) ||
            fields.contains(NaturalField.REGIONAL_MINING_ADVANTAGE) ||
            fields.contains(NaturalField.REGIONAL_TRANSPORT_ADVANTAGE) ||
            fields.contains(NaturalField.REGIONAL_SPECIALIZATION_STRENGTH) ||
            fields.contains(NaturalField.REGIONAL_ADVANTAGE_OVERLAP_RISK) ||
            fields.contains(NaturalField.REGIONAL_HIGH_ADVANTAGE_COUNT) ||
            fields.contains(NaturalField.REGIONAL_DOMINANT_ADVANTAGE) ||
            fields.contains(NaturalField.REGIONAL_ADVANTAGE_CLASS);
    }

    private boolean needsRegionalRegion(Set<NaturalField> fields) {
        return fields.contains(NaturalField.REGIONAL_REGION_KEY) ||
            fields.contains(NaturalField.REGIONAL_REGION_CELL_COUNT) ||
            fields.contains(NaturalField.REGIONAL_REGION_AREA_BLOCKS2) ||
            fields.contains(NaturalField.REGIONAL_REGION_CENTROID_X) ||
            fields.contains(NaturalField.REGIONAL_REGION_CENTROID_Z);
    }

    private boolean needsTransportGeography(Set<NaturalField> fields) {
        return fields.contains(NaturalField.NATURAL_CORRIDOR_SUITABILITY) ||
            fields.contains(NaturalField.RIVER_VALLEY_CORRIDOR) ||
            fields.contains(NaturalField.HILL_PASS_CORRIDOR) ||
            fields.contains(NaturalField.MOUNTAIN_PASS_CORRIDOR) ||
            fields.contains(NaturalField.PLATEAU_ACCESS_CORRIDOR) ||
            fields.contains(NaturalField.BASIN_GATEWAY_CORRIDOR) ||
            fields.contains(NaturalField.LOW_DIVIDE_CORRIDOR) ||
            fields.contains(NaturalField.STRAIT_ISTHMUS_POTENTIAL) ||
            fields.contains(NaturalField.TRANSPORT_CHOKEPOINT_STRENGTH) ||
            fields.contains(NaturalField.NATURAL_CORRIDOR_CLASS);
    }

    private GeoWorldRegionalAdvantage.Score regionalAdvantageAtMacroNode(
        int gridX,
        int gridZ
    ) {
        GeoWorldV1.Geometry geometry = model.manifest().geometry();
        double x = geometry.originX() + gridX * geometry.spacingX();
        double z = geometry.originZ() + gridZ * geometry.spacingZ();
        GeoWorldV1.RuntimeSample sample = model.sample(x, z);
        GeoWorldV1.ClimateSample climate = model.sampleClimate(x, z);
        GeoWorldV1.NaturalFeatureSample features =
            model.sampleNaturalFeatures(x, z);
        int slopeStep = Math.max(4, (int)Math.round(geometry.cellSize()));
        GeoWorldSettlementSuitability.Score settlement =
            GeoWorldSettlementSuitability.evaluate(
                sample,
                slopeAt(x, z, slopeStep),
                climate,
                features
            );
        GeoWorldV1.NavigationFeatureSample navigationFeatures =
            model.sampleNavigationFeatures(x, z);
        GeoWorldNavigationSuitability.Score navigation =
            GeoWorldNavigationSuitability.evaluate(
                sample,
                features,
                navigationFeatures,
                settlement
            );
        GeoWorldMiningDistrict.Score mining =
            GeoWorldMiningDistrict.evaluate(model, sample, x, z);
        return GeoWorldRegionalAdvantage.evaluate(
            sample,
            settlement,
            navigation,
            mining
        );
    }

    private double slopeAt(double x, double z, int step) {
        double hL = model.sample(x - step, z).groundHeight();
        double hR = model.sample(x + step, z).groundHeight();
        double hD = model.sample(x, z - step).groundHeight();
        double hU = model.sample(x, z + step).groundHeight();
        double dx = (hR - hL) / (2.0 * step);
        double dz = (hU - hD) / (2.0 * step);
        return Math.hypot(dx, dz);
    }

    private double aspectAt(double x, double z, int step) {
        double hL = model.sample(x - step, z).groundHeight();
        double hR = model.sample(x + step, z).groundHeight();
        double hD = model.sample(x, z - step).groundHeight();
        double hU = model.sample(x, z + step).groundHeight();
        double dx = (hR - hL) / (2.0 * step);
        double dz = (hU - hD) / (2.0 * step);
        double degrees = Math.toDegrees(Math.atan2(-dx, -dz));
        return degrees < 0 ? degrees + 360.0 : degrees;
    }

    private double spawnSuitability(
        GeoWorldV1.RuntimeSample sample,
        double x,
        double z,
        int step
    ) {
        if (!sample.sourceLand() || !"None".equals(sample.waterKind())) {
            return 0.0;
        }
        double sea = model.manifest().seaLevelNormalized();
        double heightPenalty = Math.min(1.0, Math.abs(sample.groundHeight() - (sea + 0.08)) * 4.0);
        double slope = Math.min(1.0, slopeAt(x, z, Math.max(4, step)) * 30.0);
        return Math.max(0.0, 1.0 - 0.55 * slope - 0.45 * heightPenalty);
    }

    private static double quantize(double coordinate, int resolution) {
        if (resolution <= 1) {
            return Math.floor(coordinate);
        }
        return Math.floor(coordinate / resolution) * resolution +
            resolution * 0.5;
    }

}
