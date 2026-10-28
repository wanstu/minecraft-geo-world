package geoworld.minecraft.common.worldgen;

import geoworld.runtime.GeoWorldV1;

import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.Map;

public class GeoWorldShorelinePlanner {
    public static final int MAX_SHORE_DISTANCE = ShorelineRules.MAX_SHORE_DISTANCE;
    // Legacy rules use the historical fixed 14-block corridor. v1.1.10
    // switches to convergence-based banks; its larger value is a safety
    // search bound, never a visual endpoint.
    public static final int MAX_GENTLE_BANK_DISTANCE =
        ShorelineRules.MAX_GENTLE_BANK_DISTANCE;
    public static final int MAX_ADAPTIVE_BANK_DISTANCE =
        ShorelineRules.MAX_ADAPTIVE_BANK_DISTANCE;

    private final HeightMapping heights;
    private record AdaptiveSearchKey(GeoWorldV1 runtime, int chunkX, int chunkZ) {}
    private record ChunkPlanKey(GeoWorldV1 runtime, int baseX, int baseZ) {}
    private final Map<AdaptiveSearchKey, Integer> adaptiveSearchDistanceCache =
        Collections.synchronizedMap(
            new LinkedHashMap<>(256, 0.75f, true) {
                @Override
                protected boolean removeEldestEntry(
                    Map.Entry<AdaptiveSearchKey, Integer> eldest
                ) {
                    return size() > 2048;
                }
            }
        );
    private final Map<ChunkPlanKey, ChunkPlan> chunkPlanCache =
        Collections.synchronizedMap(
            new LinkedHashMap<>(128, 0.75f, true) {
                @Override
                protected boolean removeEldestEntry(
                    Map.Entry<ChunkPlanKey, ChunkPlan> eldest
                ) {
                    return size() > 512;
                }
            }
        );

    public record Column(
        GeoWorldV1.RuntimeSample sample,
        int originalGroundBoundary,
        int groundBoundary,
        int waterBoundary,
        int shoreDistance,
        String shoreWaterKind,
        int shoreWaterBoundary,
        boolean littoralShelf,
        String surfaceOverride,
        String underOverride
    ) {
        public boolean hasShore() {
            return shoreDistance > 0 &&
                shoreDistance <= MAX_SHORE_DISTANCE;
        }
    }

    public static final class ChunkPlan {
        private final Column[] columns = new Column[16 * 16];

        private ChunkPlan() {}

        public Column column(int localX, int localZ) {
            if (localX < 0 || localX >= 16 || localZ < 0 || localZ >= 16) {
                throw new IllegalArgumentException(
                    "local column outside chunk: " + localX + "," + localZ
                );
            }
            return this.columns[localZ * 16 + localX];
        }
    }

    private interface Neighborhood {
        GeoWorldV1.RuntimeSample sample(int dx, int dz);
    }

    private static final class SampleCache {
        private final GeoWorldV1 runtime;
        private final int originX;
        private final int originZ;
        private final int stride;
        private final GeoWorldV1.RuntimeSample[] samples;

        private SampleCache(
            GeoWorldV1 runtime,
            int originX,
            int originZ,
            int stride
        ) {
            this.runtime = runtime;
            this.originX = originX;
            this.originZ = originZ;
            this.stride = stride;
            this.samples = new GeoWorldV1.RuntimeSample[stride * stride];
        }

        private GeoWorldV1.RuntimeSample sample(int x, int z) {
            if (x < 0 || x >= stride || z < 0 || z >= stride) {
                throw new IllegalArgumentException(
                    "shoreline sample outside cache: " + x + "," + z
                );
            }
            int index = z * stride + x;
            GeoWorldV1.RuntimeSample sample = samples[index];
            if (sample == null) {
                sample = runtime.sample(originX + x, originZ + z);
                samples[index] = sample;
            }
            return sample;
        }
    }

    private record WaterNeighbor(
        int distance,
        int waterBoundary,
        String waterKind
    ) {}

    private static boolean smoothRiverBanks(GeoWorldV1 runtime) {
        String rules = runtime.manifest().generationRulesVersion();
        return "geoworld-rules-v1.1.5".equals(rules) ||
            "geoworld-rules-v1.1.6".equals(rules) ||
            "geoworld-rules-v1.1.7".equals(rules) ||
            "geoworld-rules-v1.1.8".equals(rules) ||
            "geoworld-rules-v1.1.9".equals(rules) ||
            "geoworld-rules-v1.1.10".equals(rules) ||
            "geoworld-rules-v1.1.11".equals(rules);
    }

    private static boolean sealHighRiverBanks(GeoWorldV1 runtime) {
        String rules = runtime.manifest().generationRulesVersion();
        return "geoworld-rules-v1.1.6".equals(rules) ||
            "geoworld-rules-v1.1.7".equals(rules) ||
            "geoworld-rules-v1.1.8".equals(rules) ||
            "geoworld-rules-v1.1.9".equals(rules) ||
            "geoworld-rules-v1.1.10".equals(rules) ||
            "geoworld-rules-v1.1.11".equals(rules);
    }

    private static boolean adaptiveGentleBanks(GeoWorldV1 runtime) {
        String rules = runtime.manifest().generationRulesVersion();
        return "geoworld-rules-v1.1.10".equals(rules) ||
            "geoworld-rules-v1.1.11".equals(rules);
    }

    private int adaptiveSearchDistanceForChunk(
        GeoWorldV1 runtime,
        int baseX,
        int baseZ
    ) {
        if (!adaptiveGentleBanks(runtime)) {
            return MAX_GENTLE_BANK_DISTANCE;
        }
        int chunkX = Math.floorDiv(baseX, 16);
        int chunkZ = Math.floorDiv(baseZ, 16);
        AdaptiveSearchKey key = new AdaptiveSearchKey(runtime, chunkX, chunkZ);
        Integer cached = adaptiveSearchDistanceCache.get(key);
        if (cached != null) {
            return cached;
        }
        GeoWorldV1.NaturalFeatureSample features =
            runtime.sampleNaturalFeatures(
                chunkX * 16 + 7.5,
                chunkZ * 16 + 7.5
            );
        double nearest = Double.POSITIVE_INFINITY;
        if (features.distanceToRiver() != null) {
            nearest = Math.min(nearest, features.distanceToRiver());
        }
        if (features.distanceToLake() != null) {
            nearest = Math.min(nearest, features.distanceToLake());
        }
        if (features.distanceToOcean() != null) {
            nearest = Math.min(nearest, features.distanceToOcean());
        }
        int computed = nearest <= MAX_ADAPTIVE_BANK_DISTANCE + 16.0
            ? MAX_ADAPTIVE_BANK_DISTANCE
            : MAX_GENTLE_BANK_DISTANCE;
        adaptiveSearchDistanceCache.put(key, computed);
        return computed;
    }

    private record WetlandPond(
        int waterBoundary,
        boolean inside,
        boolean rim
    ) {}

    public GeoWorldShorelinePlanner(HeightMapping heights) {
        this.heights = java.util.Objects.requireNonNull(heights, "heights");
    }

    public ChunkPlan planChunkModel(
        GeoWorldV1 runtime,
        int baseX,
        int baseZ
    ) {
        ChunkPlanKey key = new ChunkPlanKey(runtime, baseX, baseZ);
        ChunkPlan cached = chunkPlanCache.get(key);
        if (cached != null) {
            return cached;
        }
        ChunkPlan computed = computeChunkModel(runtime, baseX, baseZ);
        chunkPlanCache.put(key, computed);
        return computed;
    }

    private ChunkPlan computeChunkModel(
        GeoWorldV1 runtime,
        int baseX,
        int baseZ
    ) {
        final int fringe = 3;
        final boolean adaptiveBank = adaptiveGentleBanks(runtime);
        final int bankSearchDistance = adaptiveSearchDistanceForChunk(
            runtime,
            baseX,
            baseZ
        );
        final int radius = bankSearchDistance + fringe;
        final int stride = 16 + radius * 2;
        SampleCache samples = new SampleCache(
            runtime,
            baseX - radius,
            baseZ - radius,
            stride
        );

        final int plannedStride = 16 + fringe * 2;
        Column[] preliminary =
            new Column[plannedStride * plannedStride];
        long seed = runtime.manifest().seed();
        boolean modernRiverBank = smoothRiverBanks(runtime);
        boolean sealedHighRiverBank = sealHighRiverBanks(runtime);
        for (int localZ = -fringe; localZ < 16 + fringe; localZ++) {
            for (int localX = -fringe; localX < 16 + fringe; localX++) {
                preliminary[
                    (localZ + fringe) * plannedStride +
                    (localX + fringe)
                ] = planCachedColumn(
                    samples,
                    stride,
                    localX + radius,
                    localZ + radius,
                    seed,
                    baseX + localX,
                    baseZ + localZ,
                    modernRiverBank,
                    sealedHighRiverBank,
                    adaptiveBank,
                    bankSearchDistance
                );
            }
        }

        Column[] grooveClosed = preliminary.clone();
        for (int localZ = -2; localZ <= 17; localZ++) {
            for (int localX = -2; localX <= 17; localX++) {
                int index =
                    (localZ + fringe) * plannedStride +
                    (localX + fringe);
                grooveClosed[index] = closeNarrowBackshoreGroove(
                    preliminary[index],
                    preliminary[index - 1],
                    preliminary[index + 1],
                    preliminary[index - plannedStride],
                    preliminary[index + plannedStride],
                    sealedHighRiverBank
                );
            }
        }

        Column[] pitClosed = grooveClosed.clone();
        for (int localZ = -1; localZ <= 16; localZ++) {
            for (int localX = -1; localX <= 16; localX++) {
                int index =
                    (localZ + fringe) * plannedStride +
                    (localX + fringe);
                pitClosed[index] = closeSmallBackshorePit(
                    grooveClosed[index],
                    grooveClosed[index - 1],
                    grooveClosed[index + 1],
                    grooveClosed[index - plannedStride],
                    grooveClosed[index + plannedStride],
                    sealedHighRiverBank,
                    adaptiveBank
                );
            }
        }

        Column[] bankClosed = pitClosed;
        if (adaptiveBank) {
            for (int pass = 0; pass < MAX_ADAPTIVE_BANK_DISTANCE; pass++) {
                Column[] next = bankClosed.clone();
                boolean changed = false;
                for (int localZ = -2; localZ <= 17; localZ++) {
                    for (int localX = -2; localX <= 17; localX++) {
                        int index =
                            (localZ + fringe) * plannedStride +
                            (localX + fringe);
                        Column relaxed = relaxAdaptiveBankStep(
                            bankClosed[index],
                            bankClosed[index - 1],
                            bankClosed[index + 1],
                            bankClosed[index - plannedStride],
                            bankClosed[index + plannedStride]
                        );
                        next[index] = relaxed;
                        if (relaxed != bankClosed[index]) {
                            changed = true;
                        }
                    }
                }
                bankClosed = next;
                if (!changed) break;
            }

            // Relaxation can expose a narrow one-cell groove or pit at the
            // edge of the raised bank. Iterate the small morphology repairs
            // to a fixed point (bounded by the four-block river raise cap).
            for (int cleanupPass = 0; cleanupPass < 4; cleanupPass++) {
                boolean cleanupChanged = false;
                Column[] grooves = bankClosed.clone();
                for (int localZ = -2; localZ <= 17; localZ++) {
                    for (int localX = -2; localX <= 17; localX++) {
                        int index =
                            (localZ + fringe) * plannedStride +
                            (localX + fringe);
                        Column closed = closeNarrowBackshoreGroove(
                            bankClosed[index],
                            bankClosed[index - 1],
                            bankClosed[index + 1],
                            bankClosed[index - plannedStride],
                            bankClosed[index + plannedStride],
                            sealedHighRiverBank
                        );
                        grooves[index] = closed;
                        if (closed != bankClosed[index]) cleanupChanged = true;
                    }
                }

                Column[] pits = grooves.clone();
                for (int localZ = -1; localZ <= 16; localZ++) {
                    for (int localX = -1; localX <= 16; localX++) {
                        int index =
                            (localZ + fringe) * plannedStride +
                            (localX + fringe);
                        Column closed = closeSmallBackshorePit(
                            grooves[index],
                            grooves[index - 1],
                            grooves[index + 1],
                            grooves[index - plannedStride],
                            grooves[index + plannedStride],
                            sealedHighRiverBank,
                            true
                        );
                        pits[index] = closed;
                        if (closed != grooves[index]) cleanupChanged = true;
                    }
                }
                bankClosed = pits;
                if (!cleanupChanged) break;
            }

            // Morphology cleanup can lift one cell by one block. Re-run the
            // 2D bank relaxation afterwards so cleanup itself cannot recreate
            // a two-block terminal wall. This pass only moves generated bank
            // cells within their original/max-raise envelope.
            for (int pass = 0; pass < MAX_ADAPTIVE_BANK_DISTANCE; pass++) {
                Column[] next = bankClosed.clone();
                boolean changed = false;
                for (int localZ = -2; localZ <= 17; localZ++) {
                    for (int localX = -2; localX <= 17; localX++) {
                        int index =
                            (localZ + fringe) * plannedStride +
                            (localX + fringe);
                        Column relaxed = relaxAdaptiveBankStep(
                            bankClosed[index],
                            bankClosed[index - 1],
                            bankClosed[index + 1],
                            bankClosed[index - plannedStride],
                            bankClosed[index + plannedStride]
                        );
                        next[index] = relaxed;
                        if (relaxed != bankClosed[index]) changed = true;
                    }
                }
                bankClosed = next;
                if (!changed) break;
            }
        }

        Column[] materialPass1 = bankClosed.clone();
        Column[] materialPass2 = bankClosed.clone();
        if (sealedHighRiverBank) {
            for (int localZ = -1; localZ <= 16; localZ++) {
                for (int localX = -1; localX <= 16; localX++) {
                    int index =
                        (localZ + fringe) * plannedStride +
                        (localX + fringe);
                    materialPass1[index] = fillRiverMaterialGap(
                        bankClosed[index],
                        bankClosed[index - 1],
                        bankClosed[index + 1],
                        bankClosed[index - plannedStride],
                        bankClosed[index + plannedStride]
                    );
                }
            }
            materialPass2 = materialPass1.clone();
            for (int localZ = 0; localZ < 16; localZ++) {
                for (int localX = 0; localX < 16; localX++) {
                    int index =
                        (localZ + fringe) * plannedStride +
                        (localX + fringe);
                    materialPass2[index] = fillRiverMaterialGap(
                        materialPass1[index],
                        materialPass1[index - 1],
                        materialPass1[index + 1],
                        materialPass1[index - plannedStride],
                        materialPass1[index + plannedStride]
                    );
                }
            }
        }

        ChunkPlan result = new ChunkPlan();
        for (int localZ = 0; localZ < 16; localZ++) {
            for (int localX = 0; localX < 16; localX++) {
                int index =
                    (localZ + fringe) * plannedStride +
                    (localX + fringe);
                Column geometryOnce = bankClosed[index];
                Column once = materialPass2[index];
                Column shaped = adaptiveBank
                    ? once
                    : geometryOnce != grooveClosed[index]
                        ? once
                        : closeSmallBackshorePit(
                            once,
                            materialPass2[index - 1],
                            materialPass2[index + 1],
                            materialPass2[index - plannedStride],
                            materialPass2[index + plannedStride],
                            sealedHighRiverBank,
                            false
                        );
                result.columns[localZ * 16 + localX] =
                    sealWetlandPoolRim(
                        shaped,
                        preliminary[index - 1],
                        preliminary[index + 1],
                        preliminary[index - plannedStride],
                        preliminary[index + plannedStride]
                    );
            }
        }
        return result;
    }

    public Column planModel(
        GeoWorldV1 runtime,
        int worldX,
        int worldZ
    ) {
        if (!adaptiveGentleBanks(runtime)) {
            return planModelLegacy(runtime, worldX, worldZ);
        }
        int baseX = Math.floorDiv(worldX, 16) * 16;
        int baseZ = Math.floorDiv(worldZ, 16) * 16;
        return planChunkModel(runtime, baseX, baseZ).column(
            worldX - baseX,
            worldZ - baseZ
        );
    }

    private Column planModelLegacy(
        GeoWorldV1 runtime,
        int worldX,
        int worldZ
    ) {
        final int localRadius = 3;
        final int radius = MAX_GENTLE_BANK_DISTANCE + localRadius;
        final int stride = radius * 2 + 1;
        SampleCache samples = new SampleCache(
            runtime,
            worldX - radius,
            worldZ - radius,
            stride
        );

        final int localStride = localRadius * 2 + 1;
        Column[] preliminary =
            new Column[localStride * localStride];
        long seed = runtime.manifest().seed();
        boolean modernRiverBank = smoothRiverBanks(runtime);
        boolean sealedHighRiverBank = sealHighRiverBanks(runtime);
        for (int dz = -localRadius; dz <= localRadius; dz++) {
            for (int dx = -localRadius; dx <= localRadius; dx++) {
                preliminary[
                    (dz + localRadius) * localStride +
                    (dx + localRadius)
                ] = planCachedColumn(
                    samples,
                    stride,
                    radius + dx,
                    radius + dz,
                    seed,
                    worldX + dx,
                    worldZ + dz,
                    modernRiverBank,
                    sealedHighRiverBank,
                    false,
                    MAX_GENTLE_BANK_DISTANCE
                );
            }
        }

        Column[] grooveClosed = preliminary.clone();
        for (int dz = -2; dz <= 2; dz++) {
            for (int dx = -2; dx <= 2; dx++) {
                int index =
                    (dz + localRadius) * localStride +
                    (dx + localRadius);
                grooveClosed[index] = closeNarrowBackshoreGroove(
                    preliminary[index],
                    preliminary[index - 1],
                    preliminary[index + 1],
                    preliminary[index - localStride],
                    preliminary[index + localStride],
                    sealedHighRiverBank
                );
            }
        }

        Column[] pitClosed = grooveClosed.clone();
        for (int dz = -1; dz <= 1; dz++) {
            for (int dx = -1; dx <= 1; dx++) {
                int index =
                    (dz + localRadius) * localStride +
                    (dx + localRadius);
                pitClosed[index] = closeSmallBackshorePit(
                    grooveClosed[index],
                    grooveClosed[index - 1],
                    grooveClosed[index + 1],
                    grooveClosed[index - localStride],
                    grooveClosed[index + localStride],
                    sealedHighRiverBank,
                    false
                );
            }
        }

        int center = localRadius * localStride + localRadius;
        Column[] materialPass1 = pitClosed.clone();
        Column[] materialPass2 = pitClosed.clone();
        if (sealedHighRiverBank) {
            for (int dz = -1; dz <= 1; dz++) {
                for (int dx = -1; dx <= 1; dx++) {
                    int index =
                        (dz + localRadius) * localStride +
                        (dx + localRadius);
                    materialPass1[index] = fillRiverMaterialGap(
                        pitClosed[index],
                        pitClosed[index - 1],
                        pitClosed[index + 1],
                        pitClosed[index - localStride],
                        pitClosed[index + localStride]
                    );
                }
            }
            materialPass2 = materialPass1.clone();
            materialPass2[center] = fillRiverMaterialGap(
                materialPass1[center],
                materialPass1[center - 1],
                materialPass1[center + 1],
                materialPass1[center - localStride],
                materialPass1[center + localStride]
            );
        }

        Column geometryShaped = pitClosed[center];
        Column shaped = materialPass2[center];
        if (geometryShaped == grooveClosed[center]) {
            shaped = closeSmallBackshorePit(
                shaped,
                materialPass2[center - 1],
                materialPass2[center + 1],
                materialPass2[center - localStride],
                materialPass2[center + localStride],
                sealedHighRiverBank,
                false
            );
        }
        return sealWetlandPoolRim(
            shaped,
            preliminary[center - 1],
            preliminary[center + 1],
            preliminary[center - localStride],
            preliminary[center + localStride]
        );
    }

    private Column planCachedColumn(
        SampleCache samples,
        int stride,
        int centerX,
        int centerZ,
        long seed,
        int worldX,
        int worldZ,
        boolean modernRiverBank,
        boolean sealedHighRiverBank,
        boolean adaptiveBank,
        int bankSearchDistance
    ) {
        GeoWorldV1.RuntimeSample center =
            samples.sample(centerX, centerZ);
        Neighborhood neighborhood = (dx, dz) ->
            samples.sample(centerX + dx, centerZ + dz);
        return planColumn(
            center,
            neighborhood,
            seed,
            worldX,
            worldZ,
            modernRiverBank,
            sealedHighRiverBank,
            adaptiveBank,
            bankSearchDistance
        );
    }

    public int waterBoundaryModel(
        GeoWorldV1.RuntimeSample sample,
        int groundBoundary
    ) {
        if (
            !sample.waterSurfaceDefined() ||
            sample.waterSurfaceElevation() == null
        ) {
            return groundBoundary;
        }

        int mapped = "River".equals(sample.waterKind())
            ? this.heights.boundaryYCeil(sample.waterSurfaceElevation())
            : this.heights.boundaryY(sample.waterSurfaceElevation());

        // A river's free surface belongs to the hydrology model. Do not raise
        // it merely because voxel quantization maps a shallow riverbed onto
        // the same Y boundary. Doing that creates one-block-high source-water
        // lips along the banks. River columns lower the bed by one block when
        // needed instead, preserving a flat cross-section water plane.
        if ("River".equals(sample.waterKind())) {
            return mapped;
        }
        return Math.max(groundBoundary, mapped);
    }

    private Column planColumn(
        GeoWorldV1.RuntimeSample center,
        Neighborhood neighborhood,
        long seed,
        int worldX,
        int worldZ,
        boolean modernRiverBank,
        boolean sealedHighRiverBank,
        boolean adaptiveBank,
        int bankSearchDistance
    ) {
        int originalGround = this.heights.boundaryY(center.groundHeight());
        int ownWater = waterBoundaryModel(center, originalGround);

        if (isWater(center)) {
            int waterGround = originalGround;
            if (
                "River".equals(center.waterKind()) &&
                ownWater <= waterGround
            ) {
                waterGround = ownWater - 1;
            }
            if (("Ocean".equals(center.waterKind()) ||
                 "Lake".equals(center.waterKind())) &&
                ownWater - waterGround >= 2 &&
                ownWater - waterGround <= 8) {
                // The old planner only leveled dry sand. A four-block-deep
                // ocean bed could therefore meet a raised sand rim as a
                // vertical wall. Lift just the near-shore *wet* bed through
                // a 1,2,3,4-block bathymetric profile without touching the
                // model-owned water surface or adding water sources.
                int dryDistance = nearestGentleDryBank(neighborhood);
                if (dryDistance > 0) {
                    waterGround = Math.max(
                        waterGround, ownWater - dryDistance
                    );
                }
            }
            return new Column(
                center,
                originalGround,
                waterGround,
                ownWater,
                0,
                center.waterKind(),
                ownWater,
                false,
                null,
                null
            );
        }

        WetlandPond wetlandPond = findWetlandPond(
            neighborhood,
            seed,
            worldX,
            worldZ
        );
        if (wetlandPond != null && wetlandPond.inside()) {
            // A pond uses one shared water boundary across its 3x3 basin.
            // The bed is exactly one block below that surface; the surrounding
            // one-block rim is planned by the same deterministic pond spec.
            return new Column(
                center,
                originalGround,
                wetlandPond.waterBoundary() - 1,
                wetlandPond.waterBoundary(),
                Integer.MAX_VALUE,
                "WetlandPool",
                wetlandPond.waterBoundary(),
                false,
                "minecraft:mud",
                "minecraft:dirt"
            );
        }

        WaterNeighbor shore = findNearestWater(
            neighborhood,
            bankSearchDistance
        );
        if (!adaptiveBank && shore != null &&
            shore.distance() > MAX_SHORE_DISTANCE &&
            !"Ocean".equals(shore.waterKind()) &&
            !"Lake".equals(shore.waterKind())) {
            shore = null;
        }
        if (shore == null) {
            int dryGround = originalGround;
            boolean pondRimRaised =
                wetlandPond != null &&
                wetlandPond.rim() &&
                dryGround < wetlandPond.waterBoundary();
            if (pondRimRaised) {
                dryGround = wetlandPond.waterBoundary();
            }
            return new Column(
                center,
                originalGround,
                dryGround,
                dryGround,
                Integer.MAX_VALUE,
                "None",
                dryGround,
                false,
                pondRimRaised ? "minecraft:mud" : null,
                pondRimRaised ? "minecraft:dirt" : null
            );
        }

        int adjustedGround = originalGround;
        boolean wetlandRimRaised =
            wetlandPond != null &&
            wetlandPond.rim() &&
            adjustedGround < wetlandPond.waterBoundary();
        if (wetlandRimRaised) {
            adjustedGround = wetlandPond.waterBoundary();
        }
        boolean shallowFringe = false;
        int deltaToWater = originalGround - shore.waterBoundary();

        // A dry column immediately next to source water must contain that
        // source layer after Minecraft fluid updates. v1.1.6 seals high river
        // banks completely; older rules keep the historical two-block repair.
        if (
            shore.distance() == 1 &&
            adjustedGround < shore.waterBoundary()
        ) {
            if (
                sealedHighRiverBank &&
                "River".equals(shore.waterKind())
            ) {
                int gap = shore.waterBoundary() - adjustedGround;
                final int maxSolidRaise = 4;
                if (gap <= maxSolidRaise) {
                    adjustedGround = shore.waterBoundary();
                    shallowFringe = false;
                } else {
                    adjustedGround = Math.min(
                        shore.waterBoundary() - 1,
                        adjustedGround + maxSolidRaise
                    );
                    shallowFringe = true;
                }
            } else {
                adjustedGround = Math.min(
                    shore.waterBoundary(),
                    adjustedGround + 2
                );
                int remainingGap =
                    shore.waterBoundary() - adjustedGround;
                if (remainingGap == 1) {
                    shallowFringe = true;
                }
            }
        }

        // v1.1.10 no longer gives an upward-built bank a fixed inland end.
        // The bank changes by 0 or 1 block per column, usually only once per
        // 2-3 inland columns. It stops as soon as it reconnects to the natural
        // terrain. Older locked worlds retain the historical river backing.
        if (
            adaptiveBank &&
            shore.distance() >= 2 &&
            originalGround < shore.waterBoundary() &&
            shore.waterBoundary() - originalGround <=
                adaptiveGentleBankMaxRelief()
        ) {
            int backingTarget = adaptiveGentleBankFloor(
                originalGround,
                shore.waterBoundary(),
                shore.distance()
            );
            int maxRaise = adaptiveMaxRaise(shore.waterKind());
            backingTarget = Math.min(
                backingTarget,
                originalGround + maxRaise
            );
            if (
                backingTarget > originalGround &&
                adjustedGround < backingTarget
            ) {
                adjustedGround = backingTarget;
                shallowFringe = false;
            }
        } else if (
            sealedHighRiverBank &&
            "River".equals(shore.waterKind()) &&
            shore.distance() >= 2 &&
            shore.distance() <= 3
        ) {
            int backingTarget =
                shore.waterBoundary() - (shore.distance() - 1);
            backingTarget = Math.min(
                backingTarget,
                originalGround + 4
            );
            if (adjustedGround < backingTarget) {
                adjustedGround = backingTarget;
                shallowFringe = false;
            }
        }

        if ("Ocean".equals(shore.waterKind())) {
            int oceanMinimum = oceanBackshoreMinimum(
                shore.distance(),
                shore.waterBoundary(),
                originalGround
            );
            if (adjustedGround < oceanMinimum) {
                adjustedGround = oceanMinimum;
                shallowFringe = false;
            }
        }

        boolean rocky = isRocky(center);
        // Preserve genuinely high mountain/canyon sea-cliffs. Low and
        // medium rocky banks still need a short walkable transition.
        boolean naturalCliff =
            "Canyon".equals(center.terrainVariant()) ||
            ("Mountain".equals(center.terrainVariant()) &&
                deltaToWater > 24);
        boolean taperedBank = false;
        // v1.1.10 uses a convergence envelope instead of a fixed 14-block
        // blend. The dry bank rises on a 0/1 staircase (about 2 vertical
        // blocks per 5 horizontal blocks), not a forced 45-degree ramp.
        // If that gentle envelope cannot reach the natural terrain inside the
        // adaptive corridor, leave the natural escarpment alone rather than
        // manufacture a hard terminal wall.
        if (
            adaptiveBank &&
            ("Ocean".equals(shore.waterKind()) ||
             "Lake".equals(shore.waterKind())) &&
            !naturalCliff &&
            deltaToWater >= 2 &&
            deltaToWater <= adaptiveGentleBankMaxRelief() &&
            shore.distance() <= MAX_ADAPTIVE_BANK_DISTANCE
        ) {
            int cap = adaptiveGentleBankCap(
                originalGround,
                shore.waterBoundary(),
                shore.distance()
            );
            if (
                cap < originalGround &&
                cap < adjustedGround
            ) {
                adjustedGround = cap;
                shallowFringe = false;
                taperedBank = true;
            }
        } else if (
            !adaptiveBank &&
            ("Ocean".equals(shore.waterKind()) ||
             "Lake".equals(shore.waterKind())) &&
            !naturalCliff &&
            deltaToWater >= 4 && deltaToWater <= 32 &&
            shore.distance() <= MAX_GENTLE_BANK_DISTANCE
        ) {
            int cap = gentleBankCap(
                originalGround, shore.waterBoundary(), shore.distance()
            );
            cap = Math.max(
                cap,
                "Ocean".equals(shore.waterKind())
                    ? oceanBackshoreMinimum(
                        shore.distance(), shore.waterBoundary(),
                        originalGround)
                    : shore.waterBoundary()
            );
            if (cap < adjustedGround) {
                adjustedGround = cap;
                shallowFringe = false;
                taperedBank = true;
            }
        }
        double shapeNoise = smoothValueNoise(
            seed ^ 0x53484F52455F5348L,
            worldX,
            worldZ,
            28.0
        );
        boolean steep = (taperedBank
            ? adjustedGround - shore.waterBoundary()
            : deltaToWater) >= 2;
        boolean hardBank =
            steep ||
            (!"River".equals(shore.waterKind()) &&
             deltaToWater >= 1 &&
             shapeNoise < (rocky ? 0.42 : 0.18));

        int shelfWidth = shelfWidth(
            shore.waterKind(),
            shapeNoise,
            modernRiverBank
        );
        boolean littoralShelf = false;
        if (
            !hardBank &&
            shore.distance() <= shelfWidth &&
            deltaToWater >= -1 &&
            deltaToWater <= 1
        ) {
            // Only ever lower a natural bank by one block. Columns that were
            // already below the water boundary may be raised farther because
            // that is the stability correction above, not aesthetic shaping.
            int shelfGround = Math.max(
                originalGround - 1,
                shore.waterBoundary()
            );
            adjustedGround = Math.max(
                shore.waterBoundary(),
                Math.min(adjustedGround, shelfGround)
            );
            littoralShelf = adjustedGround == shore.waterBoundary();
        }

        String surface =
            wetlandRimRaised ? "minecraft:mud" : null;
        String under =
            wetlandRimRaised ? "minecraft:dirt" : null;
        int materialWidth = materialBandWidth(shore.waterKind());
        int verticalOffset = adjustedGround - shore.waterBoundary();
        if (
            shore.distance() <= materialWidth &&
            verticalOffset >= 0 &&
            verticalOffset <= 1 &&
            !steep
        ) {
            String material = chooseMaterial(
                center,
                shore.waterKind(),
                shore.distance(),
                seed,
                worldX,
                worldZ
            );
            if (material != null && !wetlandRimRaised) {
                surface = material;
                under = underMaterial(material);
            }
        }

        return new Column(
            center,
            originalGround,
            adjustedGround,
            shallowFringe
                ? shore.waterBoundary()
                : adjustedGround,
            shore.distance(),
            shore.waterKind(),
            shore.waterBoundary(),
            littoralShelf,
            surface,
            under
        );
    }

    private WetlandPond findWetlandPond(
        Neighborhood neighborhood,
        long seed,
        int worldX,
        int worldZ
    ) {
        final int spacing = 10;
        final int anchorOffset = spacing / 2;
        int cellX = Math.floorDiv(worldX - anchorOffset, spacing);
        int cellZ = Math.floorDiv(worldZ - anchorOffset, spacing);

        for (int azCell = cellZ - 1; azCell <= cellZ + 1; azCell++) {
            int anchorZ = azCell * spacing + anchorOffset;
            int relZ = worldZ - anchorZ;
            if (Math.abs(relZ) > 2) {
                continue;
            }
            for (int axCell = cellX - 1; axCell <= cellX + 1; axCell++) {
                int anchorX = axCell * spacing + anchorOffset;
                int relX = worldX - anchorX;
                if (Math.abs(relX) > 2) {
                    continue;
                }

                Integer waterBoundary = wetlandPondBoundary(
                    neighborhood,
                    seed,
                    worldX,
                    worldZ,
                    anchorX,
                    anchorZ
                );
                if (waterBoundary == null) {
                    continue;
                }

                int maxAbs = Math.max(
                    Math.abs(relX),
                    Math.abs(relZ)
                );
                if (maxAbs <= 1) {
                    return new WetlandPond(
                        waterBoundary,
                        true,
                        false
                    );
                }
                if (maxAbs == 2) {
                    return new WetlandPond(
                        waterBoundary,
                        false,
                        true
                    );
                }
            }
        }
        return null;
    }

    private Integer wetlandPondBoundary(
        Neighborhood neighborhood,
        long seed,
        int worldX,
        int worldZ,
        int anchorX,
        int anchorZ
    ) {
        int anchorDX = anchorX - worldX;
        int anchorDZ = anchorZ - worldZ;
        GeoWorldV1.RuntimeSample anchor =
            neighborhood.sample(anchorDX, anchorDZ);
        if (
            !isWetGround(anchor) ||
            isRocky(anchor) ||
            isWater(anchor)
        ) {
            return null;
        }

        // Terrain/wetness already make valid basins irregular. Use one
        // uniform deterministic density gate instead of stacking smooth-noise
        // thresholds that accidentally rejected almost every real basin.
        if (
            hashUnit(
                seed ^ 0x5745544C414E4450L,
                anchorX,
                anchorZ
            ) > 0.65
        ) {
            return null;
        }

        int wetInterior = 0;
        int minGround = Integer.MAX_VALUE;
        int maxGround = Integer.MIN_VALUE;
        for (int dz = -1; dz <= 1; dz++) {
            for (int dx = -1; dx <= 1; dx++) {
                GeoWorldV1.RuntimeSample sample = neighborhood.sample(
                    anchorDX + dx,
                    anchorDZ + dz
                );
                if (isWater(sample) || isRocky(sample)) {
                    return null;
                }
                if (isWetGround(sample)) {
                    wetInterior++;
                }
                int ground = this.heights.boundaryY(
                    sample.groundHeight()
                );
                minGround = Math.min(minGround, ground);
                maxGround = Math.max(maxGround, ground);
            }
        }
        if (wetInterior < 5 || maxGround - minGround > 1) {
            return null;
        }

        // Use the highest interior ground as the common water boundary. Lower
        // interior cells need no excavation; higher cells are carved by one
        // block. A one-block-low outer rim can be raised by the planner, but a
        // deeper escape route rejects the pond.
        int waterBoundary = maxGround;
        for (int dz = -2; dz <= 2; dz++) {
            for (int dx = -2; dx <= 2; dx++) {
                if (Math.max(Math.abs(dx), Math.abs(dz)) != 2) {
                    continue;
                }
                GeoWorldV1.RuntimeSample sample = neighborhood.sample(
                    anchorDX + dx,
                    anchorDZ + dz
                );
                if (isWater(sample)) {
                    return null;
                }
                int ground = this.heights.boundaryY(
                    sample.groundHeight()
                );
                if (ground < waterBoundary - 1) {
                    return null;
                }
            }
        }
        return waterBoundary;
    }

    private boolean isWetGround(
        GeoWorldV1.RuntimeSample sample
    ) {
        return "Wetland".equals(sample.biomeFamily()) ||
            "Wet Mud".equals(sample.surfaceProfile()) ||
            "Peat".equals(sample.surfaceProfile());
    }

    private int nearestGentleDryBank(Neighborhood neighborhood) {
        for (int distance = 1; distance <= MAX_SHORE_DISTANCE; distance++) {
            for (int dx = -distance; dx <= distance; dx++) {
                int dz = distance - Math.abs(dx);
                if (isGentleDryBank(neighborhood.sample(dx, dz)) ||
                    (dz != 0 &&
                     isGentleDryBank(neighborhood.sample(dx, -dz)))) {
                    return distance;
                }
            }
        }
        return 0;
    }

    private boolean isGentleDryBank(GeoWorldV1.RuntimeSample sample) {
        if (isWater(sample) || isRocky(sample)) return false;
        return !"Canyon".equals(sample.terrainVariant());
    }

    private WaterNeighbor findNearestWater(
        Neighborhood neighborhood,
        int maxDistance
    ) {
        for (int distance = 1; distance <= maxDistance; distance++) {
            WaterNeighbor best = null;
            for (int dx = -distance; dx <= distance; dx++) {
                int dzAbs = distance - Math.abs(dx);
                best = betterWater(
                    best,
                    waterAt(neighborhood, dx, dzAbs, distance)
                );
                if (dzAbs != 0) {
                    best = betterWater(
                        best,
                        waterAt(neighborhood, dx, -dzAbs, distance)
                    );
                }
            }
            if (best != null) {
                return best;
            }
        }
        return null;
    }

    private WaterNeighbor waterAt(
        Neighborhood neighborhood,
        int dx,
        int dz,
        int distance
    ) {
        GeoWorldV1.RuntimeSample sample = neighborhood.sample(dx, dz);
        if (!isWater(sample)) {
            return null;
        }
        int ground = this.heights.boundaryY(sample.groundHeight());
        int water = waterBoundaryModel(sample, ground);
        if (
            water <= ground &&
            !"River".equals(sample.waterKind())
        ) {
            return null;
        }
        return new WaterNeighbor(
            distance,
            water,
            sample.waterKind()
        );
    }

    private WaterNeighbor betterWater(
        WaterNeighbor current,
        WaterNeighbor candidate
    ) {
        if (candidate == null) {
            return current;
        }
        if (current == null) {
            return candidate;
        }
        if (candidate.waterBoundary() != current.waterBoundary()) {
            return candidate.waterBoundary() > current.waterBoundary()
                ? candidate
                : current;
        }
        return waterPriority(candidate.waterKind()) >
            waterPriority(current.waterKind())
            ? candidate
            : current;
    }

    private int waterPriority(String kind) {
        return switch (kind) {
            case "River" -> 3;
            case "Lake" -> 2;
            case "Ocean" -> 1;
            default -> 0;
        };
    }

    private boolean isWater(GeoWorldV1.RuntimeSample sample) {
        if (
            "None".equals(sample.waterKind()) ||
            !sample.waterSurfaceDefined() ||
            sample.waterSurfaceElevation() == null
        ) {
            return false;
        }

        // A Runtime Lake/Ocean label can extend onto the exact detailed shore
        // where voxelized terrain already reaches the water boundary. That is
        // a dry bank, not a water column. River is different: shallow river
        // columns deliberately lower the voxel bed by one block rather than
        // raising the hydrologic surface.
        if ("River".equals(sample.waterKind())) {
            return true;
        }
        int ground = this.heights.boundaryY(sample.groundHeight());
        int water = this.heights.boundaryY(sample.waterSurfaceElevation());
        return water > ground;
    }

    private Column sealWetlandPoolRim(
        Column center,
        Column west,
        Column east,
        Column north,
        Column south
    ) {
        if (
            !"None".equals(center.sample().waterKind()) ||
            center.waterBoundary() > center.groundBoundary()
        ) {
            return center;
        }

        int target = center.groundBoundary();
        for (Column neighbor :
            new Column[]{west, east, north, south}) {
            if (
                "WetlandPool".equals(neighbor.shoreWaterKind()) &&
                neighbor.waterBoundary() > neighbor.groundBoundary()
            ) {
                target = Math.max(target, neighbor.waterBoundary());
            }
        }
        if (target <= center.groundBoundary()) {
            return center;
        }

        // Wetland pools are deliberately shallow source-water basins. Their
        // final cardinal rim must be solid at the pool water boundary even
        // when another shoreline rule (for example a nearby river) also owns
        // this dry cell. Minecraft fluids escape only through cardinal faces,
        // so a one-block mud rim is sufficient and does not reshape the wider
        // floodplain.
        return new Column(
            center.sample(),
            center.originalGroundBoundary(),
            target,
            target,
            1,
            "WetlandPool",
            target,
            false,
            "minecraft:mud",
            "minecraft:dirt"
        );
    }

    // Kept as facade methods so existing 26.3 generation and verification
    // code remains stable while the actual rules live in minecraft-common.
    public static int gentleBankCap(
        int originalGround,
        int waterBoundary,
        int distance
    ) {
        return ShorelineRules.gentleBankCap(
            originalGround,
            waterBoundary,
            distance
        );
    }

    public static int adaptiveGentleBankCap(
        int originalGround,
        int waterBoundary,
        int distance
    ) {
        return ShorelineRules.adaptiveGentleBankCap(
            originalGround,
            waterBoundary,
            distance
        );
    }

    public static int adaptiveGentleBankFloor(
        int originalGround,
        int waterBoundary,
        int distance
    ) {
        return ShorelineRules.adaptiveGentleBankFloor(
            originalGround,
            waterBoundary,
            distance
        );
    }

    public static int adaptiveGentleBankMaxRelief() {
        return ShorelineRules.adaptiveGentleBankMaxRelief();
    }

    public static int oceanBackshoreMinimum(
        int shoreDistance,
        int waterBoundary,
        int originalGround
    ) {
        return ShorelineRules.oceanBackshoreMinimum(
            shoreDistance,
            waterBoundary,
            originalGround
        );
    }

    private Column closeNarrowBackshoreGroove(
        Column center,
        Column west,
        Column east,
        Column north,
        Column south,
        boolean blendRepairMaterial
    ) {
        if (
            !"None".equals(center.sample().waterKind()) ||
            center.shoreDistance() < 2 ||
            center.shoreDistance() > 3
        ) {
            return center;
        }

        int target = center.groundBoundary();
        target = Math.max(
            target,
            bracketedPlannedGround(center, west, east)
        );
        target = Math.max(
            target,
            bracketedPlannedGround(center, north, south)
        );
        if (target <= center.groundBoundary()) {
            return center;
        }

        // This is a final-voxel morphology repair, not a terrain rewrite.
        // Close only a one-block-wide row that is lower than both the
        // water-side and land-side planned rows. Never raise more than one
        // block or above the low bank cap, so broad valleys and drainage
        // swales remain untouched.
        target = Math.min(target, center.groundBoundary() + 1);
        target = Math.min(target, center.shoreWaterBoundary() + 1);
        if (target <= center.groundBoundary()) {
            return center;
        }

        Column materialSource = blendRepairMaterial
            ? shorewardMaterialSource(
                center, west, east, north, south
            )
            : null;
        return new Column(
            center.sample(),
            center.originalGroundBoundary(),
            target,
            target,
            center.shoreDistance(),
            center.shoreWaterKind(),
            center.shoreWaterBoundary(),
            center.littoralShelf(),
            materialSource != null
                ? materialSource.surfaceOverride()
                : center.surfaceOverride(),
            materialSource != null
                ? materialSource.underOverride()
                : center.underOverride()
        );
    }

    private static int adaptiveMaxRaise(String waterKind) {
        return switch (waterKind) {
            case "River" -> 4;
            case "Ocean" -> 3;
            case "Lake" -> 2;
            default -> 0;
        };
    }

    private Column relaxAdaptiveBankStep(
        Column center,
        Column west,
        Column east,
        Column north,
        Column south
    ) {
        if (
            !"None".equals(center.sample().waterKind()) ||
            center.shoreDistance() < 2 ||
            center.groundBoundary() == center.originalGroundBoundary()
        ) {
            return center;
        }

        int original = center.originalGroundBoundary();
        int current = center.groundBoundary();
        int target = current;
        boolean raised = current > original;
        boolean lowered = current < original;
        for (Column neighbor : new Column[]{west, east, north, south}) {
            if (!"None".equals(neighbor.sample().waterKind())) continue;
            boolean compatible =
                center.shoreWaterKind().equals(neighbor.shoreWaterKind()) ||
                neighbor.shoreDistance() == Integer.MAX_VALUE;
            if (!compatible) continue;

            if (raised) {
                // Raised banks converge toward lower/natural neighbors only.
                // Never pull a generated bank back upward to chase another
                // generated row: the untouched terrain is the closure anchor.
                if (target > neighbor.groundBoundary() + 1) {
                    target = Math.max(
                        original,
                        Math.min(target, neighbor.groundBoundary() + 1)
                    );
                }
            } else if (
                lowered && target < neighbor.groundBoundary() - 1
            ) {
                target = Math.min(
                    original,
                    Math.max(target, neighbor.groundBoundary() - 1)
                );
            }
        }
        if (target == current) return center;

        int waterBoundary = center.waterBoundary() == current
            ? target
            : center.waterBoundary();
        return new Column(
            center.sample(),
            original,
            target,
            waterBoundary,
            center.shoreDistance(),
            center.shoreWaterKind(),
            center.shoreWaterBoundary(),
            center.littoralShelf(),
            center.surfaceOverride(),
            center.underOverride()
        );
    }

    private Column closeSmallBackshorePit(
        Column center,
        Column west,
        Column east,
        Column north,
        Column south,
        boolean blendRepairMaterial,
        boolean adaptiveBank
    ) {
        if (
            !"None".equals(center.sample().waterKind()) ||
            center.shoreDistance() < 2 ||
            center.shoreDistance() > 4
        ) {
            return center;
        }

        Column[] neighbors = {west, east, north, south};
        int higherDry = 0;
        int cap = Integer.MAX_VALUE;
        boolean adaptiveEscape = false;
        for (Column neighbor : neighbors) {
            if (
                !"None".equals(neighbor.sample().waterKind()) ||
                !center.shoreWaterKind().equals(
                    neighbor.shoreWaterKind()
                )
            ) {
                continue;
            }
            if (
                adaptiveBank &&
                neighbor.groundBoundary() <= center.groundBoundary()
            ) {
                // A convergence ramp has a real downhill/equal continuation.
                // It is not an isolated pit merely because the other three
                // cardinal neighbors are higher.
                adaptiveEscape = true;
            }
            if (
                neighbor.groundBoundary() >
                center.groundBoundary()
            ) {
                higherDry++;
                cap = Math.min(cap, neighbor.groundBoundary());
            }
        }
        if (
            adaptiveEscape ||
            higherDry < 3 ||
            cap == Integer.MAX_VALUE
        ) {
            return center;
        }

        int waterCap = center.shoreWaterBoundary() +
            (adaptiveBank ? Math.max(1, center.shoreDistance() - 2) : 1);
        int target = Math.min(
            center.groundBoundary() + 1,
            Math.min(cap, waterCap)
        );
        if (target <= center.groundBoundary()) {
            return center;
        }

        Column materialSource = blendRepairMaterial
            ? shorewardMaterialSource(
                center, west, east, north, south
            )
            : null;
        return new Column(
            center.sample(),
            center.originalGroundBoundary(),
            target,
            target,
            center.shoreDistance(),
            center.shoreWaterKind(),
            center.shoreWaterBoundary(),
            center.littoralShelf(),
            materialSource != null
                ? materialSource.surfaceOverride()
                : center.surfaceOverride(),
            materialSource != null
                ? materialSource.underOverride()
                : center.underOverride()
        );
    }

    private Column fillRiverMaterialGap(
        Column center,
        Column west,
        Column east,
        Column north,
        Column south
    ) {
        if (
            !"None".equals(center.sample().waterKind()) ||
            !"River".equals(center.shoreWaterKind()) ||
            center.shoreDistance() < 2 ||
            center.shoreDistance() > 3 ||
            center.surfaceOverride() != null
        ) {
            return center;
        }
        Column materialSource = shorewardMaterialSource(
            center, west, east, north, south
        );
        if (materialSource == null) {
            return center;
        }
        return new Column(
            center.sample(),
            center.originalGroundBoundary(),
            center.groundBoundary(),
            center.waterBoundary(),
            center.shoreDistance(),
            center.shoreWaterKind(),
            center.shoreWaterBoundary(),
            center.littoralShelf(),
            materialSource.surfaceOverride(),
            materialSource.underOverride()
        );
    }

    private Column shorewardMaterialSource(
        Column center,
        Column... neighbors
    ) {
        Column best = null;
        for (Column neighbor : neighbors) {
            if (
                neighbor == null ||
                !"None".equals(neighbor.sample().waterKind()) ||
                !center.shoreWaterKind().equals(
                    neighbor.shoreWaterKind()
                ) ||
                neighbor.shoreDistance() <= 0 ||
                neighbor.shoreDistance() >= center.shoreDistance() ||
                neighbor.surfaceOverride() == null
            ) {
                continue;
            }
            if (
                best == null ||
                neighbor.shoreDistance() < best.shoreDistance()
            ) {
                best = neighbor;
            }
        }
        return best;
    }

    private int bracketedPlannedGround(
        Column center,
        Column a,
        Column b
    ) {
        if (
            !"None".equals(a.sample().waterKind()) ||
            !"None".equals(b.sample().waterKind()) ||
            !center.shoreWaterKind().equals(a.shoreWaterKind()) ||
            !center.shoreWaterKind().equals(b.shoreWaterKind())
        ) {
            return center.groundBoundary();
        }

        int distance = center.shoreDistance();
        boolean bracketsShore =
            (a.shoreDistance() == distance - 1 &&
             b.shoreDistance() == distance + 1) ||
            (b.shoreDistance() == distance - 1 &&
             a.shoreDistance() == distance + 1);
        if (
            !bracketsShore ||
            a.groundBoundary() <= center.groundBoundary() ||
            b.groundBoundary() <= center.groundBoundary()
        ) {
            return center.groundBoundary();
        }
        return Math.min(
            a.groundBoundary(),
            b.groundBoundary()
        );
    }

    private int shelfWidth(
        String waterKind,
        double noise,
        boolean modernRiverBank
    ) {
        return switch (waterKind) {
            // Low-gradient river banks keep a continuous two-block dry shelf.
            // v1.1.4 accidentally lowered a third river row even though the
            // River material rules stop at distance 2. On diagonal banks that
            // left low grass teeth between the sandy shelf and inland slope.
            case "River" -> modernRiverBank ? 2 : 3;
            case "Lake" -> noise >= 0.62 ? 3 : 2;
            case "Ocean" -> noise >= 0.68 ? 3 : 2;
            default -> 1;
        };
    }

    private int materialBandWidth(String waterKind) {
        return switch (waterKind) {
            case "River" -> 3;
            case "Lake" -> 3;
            case "Ocean" -> 4;
            default -> 1;
        };
    }

    private boolean isRocky(GeoWorldV1.RuntimeSample sample) {
        return "Mountain".equals(sample.terrainVariant()) ||
            "Canyon".equals(sample.terrainVariant()) ||
            "Alpine Rock".equals(sample.surfaceProfile()) ||
            "Desert Gravel".equals(sample.surfaceProfile());
    }

    private String chooseMaterial(
        GeoWorldV1.RuntimeSample sample,
        String waterKind,
        int distance,
        long seed,
        int worldX,
        int worldZ
    ) {
        // Shore materials should read as long natural reaches, not
        // one-block salt-and-pepper noise. Keep the secondary field weak so
        // diagonal river/coast voxelization does not look like a zipper.
        double patch =
            0.92 * smoothValueNoise(
                seed ^ 0x53484F52455F4D31L,
                worldX,
                worldZ,
                92.0
            ) +
            0.08 * smoothValueNoise(
                seed ^ 0x53484F52455F4D32L,
                worldX,
                worldZ,
                38.0
            );

        String family = sample.biomeFamily();
        String surface = sample.surfaceProfile();

        if (isRocky(sample)) {
            if (distance == 1) {
                return patch < 0.72
                    ? "minecraft:gravel"
                    : "minecraft:coarse_dirt";
            }
            return patch < 0.36
                ? "minecraft:gravel"
                : null;
        }

        if (
            "Wetland".equals(family) ||
            "Wet Mud".equals(surface) ||
            "Peat".equals(surface)
        ) {
            if (distance == 1) {
                return patch < 0.68
                    ? "minecraft:mud"
                    : "minecraft:dirt";
            }
            return patch < 0.52
                ? "minecraft:mud"
                : (patch < 0.78 ? "minecraft:dirt" : null);
        }

        if (
            "Temperate Desert".equals(family) ||
            "Hot Desert".equals(family) ||
            "Desert Sand".equals(surface)
        ) {
            if (distance <= 2) {
                return patch < 0.88
                    ? "minecraft:sand"
                    : "minecraft:gravel";
            }
            return patch < 0.55 ? "minecraft:sand" : null;
        }

        if ("River".equals(waterKind)) {
            if (distance == 1) {
                if (patch < 0.16) return "minecraft:gravel";
                if (patch < 0.70) return "minecraft:sand";
                if (patch < 0.88) return "minecraft:mud";
                return "minecraft:dirt";
            }
            if (distance == 2) {
                if (patch < 0.18) return "minecraft:gravel";
                if (patch < 0.54) return "minecraft:sand";
                if (patch < 0.68) return "minecraft:mud";
                return null;
            }
        }

        if ("Ocean".equals(waterKind)) {
            if (distance <= 2) {
                if (patch < 0.68) return "minecraft:sand";
                if (patch < 0.83) return "minecraft:gravel";
                if (patch < 0.91) return "minecraft:clay";
                return "minecraft:dirt";
            }
            if (distance == 3) {
                if (patch < 0.46) return "minecraft:sand";
                if (patch < 0.56) return "minecraft:gravel";
                return null;
            }
            return patch < 0.30 ? "minecraft:sand" : null;
        }

        if (distance == 1) {
            // 75%+ of ordinary immediate river/lake shore remains a substrate
            // accepted by vanilla SUPPORTS_SUGAR_CANE (dirt/mud/sand).
            if (patch < 0.10) return "minecraft:clay";
            if (patch < 0.24) return "minecraft:gravel";
            if (patch < 0.48) return "minecraft:mud";
            if (patch < 0.69) return "minecraft:sand";
            if (patch < 0.86) return "minecraft:coarse_dirt";
            return "minecraft:dirt";
        }

        if (distance == 2) {
            if (patch < 0.16) return "minecraft:gravel";
            if (patch < 0.34) return "minecraft:sand";
            if (patch < 0.50) return "minecraft:mud";
            if (patch < 0.70) return "minecraft:coarse_dirt";
            return null;
        }

        if ("Lake".equals(waterKind) && distance == 3) {
            if (patch < 0.18) return "minecraft:mud";
            if (patch < 0.30) return "minecraft:sand";
        }
        return null;
    }

    private String underMaterial(String surface) {
        return switch (surface) {
            case "minecraft:sand" -> "minecraft:sand";
            case "minecraft:gravel" -> "minecraft:gravel";
            case "minecraft:clay" -> "minecraft:clay";
            case "minecraft:mud",
                 "minecraft:dirt",
                 "minecraft:coarse_dirt" -> "minecraft:dirt";
            default -> null;
        };
    }

    private double smoothValueNoise(
        long seed,
        double x,
        double z,
        double scale
    ) {
        double gx = x / scale;
        double gz = z / scale;
        int x0 = (int)Math.floor(gx);
        int z0 = (int)Math.floor(gz);
        double tx = smoothstep(gx - x0);
        double tz = smoothstep(gz - z0);

        double v00 = hashUnit(seed, x0, z0);
        double v10 = hashUnit(seed, x0 + 1, z0);
        double v01 = hashUnit(seed, x0, z0 + 1);
        double v11 = hashUnit(seed, x0 + 1, z0 + 1);
        double a = v00 + (v10 - v00) * tx;
        double b = v01 + (v11 - v01) * tx;
        return a + (b - a) * tz;
    }

    private double hashUnit(long seed, int x, int z) {
        long h = seed;
        h ^= (long)x * 0x9E3779B97F4A7C15L;
        h ^= (long)z * 0xC2B2AE3D27D4EB4FL;
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        h ^= h >>> 31;
        return (h >>> 11) * 0x1.0p-53;
    }

    private double smoothstep(double t) {
        t = Math.max(0.0, Math.min(1.0, t));
        return t * t * (3.0 - 2.0 * t);
    }
}
