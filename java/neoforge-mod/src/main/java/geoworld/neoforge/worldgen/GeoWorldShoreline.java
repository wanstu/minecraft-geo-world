package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;
import geoworld.runtime.GeoWorldV1;

final class GeoWorldShoreline {
    static final int MAX_SHORE_DISTANCE = 4;
    // Only Lake/Ocean dry shores can need a longer inland transition.
    // Do not widen the river shelf or move a modeled water surface.
    static final int MAX_GENTLE_BANK_DISTANCE = 10;

    private static final Minecraft26_3HeightMapping HEIGHTS =
        new Minecraft26_3HeightMapping();

    record Column(
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
        boolean hasShore() {
            return shoreDistance > 0 &&
                shoreDistance <= MAX_SHORE_DISTANCE;
        }
    }

    static final class ChunkPlan {
        private final Column[] columns = new Column[16 * 16];

        private ChunkPlan() {}

        Column column(int localX, int localZ) {
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

    private record WaterNeighbor(
        int distance,
        int waterBoundary,
        String waterKind
    ) {}

    private record WetlandPond(
        int waterBoundary,
        boolean inside,
        boolean rim
    ) {}

    private GeoWorldShoreline() {}

    static ChunkPlan planChunk(
        GeoWorldV1 runtime,
        int baseX,
        int baseZ
    ) {
        final int fringe = 3;
        final int radius = MAX_GENTLE_BANK_DISTANCE + fringe;
        final int stride = 16 + radius * 2;
        GeoWorldV1.RuntimeSample[] samples =
            new GeoWorldV1.RuntimeSample[stride * stride];

        for (int z = 0; z < stride; z++) {
            int worldZ = baseZ + z - radius;
            for (int x = 0; x < stride; x++) {
                int worldX = baseX + x - radius;
                samples[z * stride + x] = runtime.sample(worldX, worldZ);
            }
        }

        final int plannedStride = 16 + fringe * 2;
        Column[] preliminary =
            new Column[plannedStride * plannedStride];
        long seed = runtime.manifest().seed();
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
                    baseZ + localZ
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
                    preliminary[index + plannedStride]
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
                    grooveClosed[index + plannedStride]
                );
            }
        }

        ChunkPlan result = new ChunkPlan();
        for (int localZ = 0; localZ < 16; localZ++) {
            for (int localX = 0; localX < 16; localX++) {
                int index =
                    (localZ + fringe) * plannedStride +
                    (localX + fringe);
                Column once = pitClosed[index];
                Column shaped =
                    once != grooveClosed[index]
                        ? once
                        : closeSmallBackshorePit(
                            once,
                            pitClosed[index - 1],
                            pitClosed[index + 1],
                            pitClosed[index - plannedStride],
                            pitClosed[index + plannedStride]
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

    static Column plan(
        GeoWorldV1 runtime,
        int worldX,
        int worldZ
    ) {
        final int localRadius = 3;
        final int radius = MAX_GENTLE_BANK_DISTANCE + localRadius;
        final int stride = radius * 2 + 1;
        GeoWorldV1.RuntimeSample[] samples =
            new GeoWorldV1.RuntimeSample[stride * stride];
        for (int z = 0; z < stride; z++) {
            for (int x = 0; x < stride; x++) {
                samples[z * stride + x] = runtime.sample(
                    worldX + x - radius,
                    worldZ + z - radius
                );
            }
        }

        final int localStride = localRadius * 2 + 1;
        Column[] preliminary =
            new Column[localStride * localStride];
        long seed = runtime.manifest().seed();
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
                    worldZ + dz
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
                    preliminary[index + localStride]
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
                    grooveClosed[index + localStride]
                );
            }
        }

        int center = localRadius * localStride + localRadius;
        Column shaped = pitClosed[center];
        if (shaped == grooveClosed[center]) {
            shaped = closeSmallBackshorePit(
                shaped,
                pitClosed[center - 1],
                pitClosed[center + 1],
                pitClosed[center - localStride],
                pitClosed[center + localStride]
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

    private static Column planCachedColumn(
        GeoWorldV1.RuntimeSample[] samples,
        int stride,
        int centerX,
        int centerZ,
        long seed,
        int worldX,
        int worldZ
    ) {
        GeoWorldV1.RuntimeSample center =
            samples[centerZ * stride + centerX];
        Neighborhood neighborhood = (dx, dz) ->
            samples[
                (centerZ + dz) * stride +
                (centerX + dx)
            ];
        return planColumn(
            center,
            neighborhood,
            seed,
            worldX,
            worldZ
        );
    }

    static int waterBoundary(
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
            ? HEIGHTS.boundaryYCeil(sample.waterSurfaceElevation())
            : HEIGHTS.boundaryY(sample.waterSurfaceElevation());

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

    private static Column planColumn(
        GeoWorldV1.RuntimeSample center,
        Neighborhood neighborhood,
        long seed,
        int worldX,
        int worldZ
    ) {
        int originalGround = HEIGHTS.boundaryY(center.groundHeight());
        int ownWater = waterBoundary(center, originalGround);

        if (isWater(center)) {
            int waterGround = originalGround;
            if (
                "River".equals(center.waterKind()) &&
                ownWater <= waterGround
            ) {
                waterGround = ownWater - 1;
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

        WaterNeighbor shore = findNearestWater(neighborhood);
        if (shore != null && shore.distance() > MAX_SHORE_DISTANCE &&
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

        // A dry column immediately next to source water may never remain as an
        // empty slot below that source-water boundary. Repair at most two solid
        // blocks; if exactly one block of vertical gap remains, make it a
        // shallow source-water fringe instead of building a 3-block artificial
        // levee. A deeper remainder is left for the verifier to reject as a
        // real model problem.
        if (
            shore.distance() == 1 &&
            adjustedGround < shore.waterBoundary()
        ) {
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

        if ("Ocean".equals(shore.waterKind())) {
            int oceanMinimum = oceanBackshoreMinimum(
                shore.distance(),
                shore.waterBoundary()
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
        // The four-cell littoral shelf preserves water/shore ownership, but
        // nonrocky lake and sea edges can still rise into an abrupt vertical
        // wall just inland. Blend the *dry* terrain over a wider corridor
        // only where its raw height actually needs it. Leave rocky natural
        // cliffs, rivers, and large escarpments unchanged.
        if (("Ocean".equals(shore.waterKind()) ||
             "Lake".equals(shore.waterKind())) &&
            !naturalCliff &&
            deltaToWater >= 4 && deltaToWater <= 32 &&
            shore.distance() <= MAX_GENTLE_BANK_DISTANCE) {
            int cap = gentleBankCap(
                originalGround, shore.waterBoundary(), shore.distance()
            );
            cap = Math.max(
                cap,
                "Ocean".equals(shore.waterKind())
                    ? oceanBackshoreMinimum(
                        shore.distance(), shore.waterBoundary())
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
            (deltaToWater >= 1 &&
             shapeNoise < (rocky ? 0.42 : 0.18));

        int shelfWidth = shelfWidth(shore.waterKind(), shapeNoise);
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

    private static WetlandPond findWetlandPond(
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

    private static Integer wetlandPondBoundary(
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
                int ground = HEIGHTS.boundaryY(
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
                int ground = HEIGHTS.boundaryY(
                    sample.groundHeight()
                );
                if (ground < waterBoundary - 1) {
                    return null;
                }
            }
        }
        return waterBoundary;
    }

    private static boolean isWetGround(
        GeoWorldV1.RuntimeSample sample
    ) {
        return "Wetland".equals(sample.biomeFamily()) ||
            "Wet Mud".equals(sample.surfaceProfile()) ||
            "Peat".equals(sample.surfaceProfile());
    }

    private static WaterNeighbor findNearestWater(
        Neighborhood neighborhood
    ) {
        for (int distance = 1; distance <= MAX_GENTLE_BANK_DISTANCE; distance++) {
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

    private static WaterNeighbor waterAt(
        Neighborhood neighborhood,
        int dx,
        int dz,
        int distance
    ) {
        GeoWorldV1.RuntimeSample sample = neighborhood.sample(dx, dz);
        if (!isWater(sample)) {
            return null;
        }
        int ground = HEIGHTS.boundaryY(sample.groundHeight());
        int water = waterBoundary(sample, ground);
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

    private static WaterNeighbor betterWater(
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

    private static int waterPriority(String kind) {
        return switch (kind) {
            case "River" -> 3;
            case "Lake" -> 2;
            case "Ocean" -> 1;
            default -> 0;
        };
    }

    private static boolean isWater(GeoWorldV1.RuntimeSample sample) {
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
        int ground = HEIGHTS.boundaryY(sample.groundHeight());
        int water = HEIGHTS.boundaryY(sample.waterSurfaceElevation());
        return water > ground;
    }

    private static Column sealWetlandPoolRim(
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

    // Ease-in/ease-out instead of an abrupt four-block sea wall.
    // Leaves the inland height untouched at the end of the blending band.
    static int gentleBankCap(int originalGround, int waterBoundary, int distance) {
        if (distance >= MAX_GENTLE_BANK_DISTANCE) return originalGround;
        double t = Math.max(0.0,
            distance / (double)MAX_GENTLE_BANK_DISTANCE);
        double ease = t * t * (3.0 - 2.0 * t);
        return waterBoundary + Math.max(1,
            (int)Math.round((originalGround - waterBoundary) * ease));
    }

    private static int oceanBackshoreMinimum(
        int shoreDistance,
        int waterBoundary
    ) {
        return switch (shoreDistance) {
            case 1, 2 -> waterBoundary;
            case 3 -> waterBoundary - 1;
            case 4 -> waterBoundary - 2;
            default -> Integer.MIN_VALUE;
        };
    }

    private static Column closeNarrowBackshoreGroove(
        Column center,
        Column west,
        Column east,
        Column north,
        Column south
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

        return new Column(
            center.sample(),
            center.originalGroundBoundary(),
            target,
            target,
            center.shoreDistance(),
            center.shoreWaterKind(),
            center.shoreWaterBoundary(),
            center.littoralShelf(),
            center.surfaceOverride(),
            center.underOverride()
        );
    }

    private static Column closeSmallBackshorePit(
        Column center,
        Column west,
        Column east,
        Column north,
        Column south
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
                neighbor.groundBoundary() >
                center.groundBoundary()
            ) {
                higherDry++;
                cap = Math.min(cap, neighbor.groundBoundary());
            }
        }
        if (higherDry < 3 || cap == Integer.MAX_VALUE) {
            return center;
        }

        int target = Math.min(
            center.groundBoundary() + 1,
            Math.min(cap, center.shoreWaterBoundary() + 1)
        );
        if (target <= center.groundBoundary()) {
            return center;
        }

        return new Column(
            center.sample(),
            center.originalGroundBoundary(),
            target,
            target,
            center.shoreDistance(),
            center.shoreWaterKind(),
            center.shoreWaterBoundary(),
            center.littoralShelf(),
            center.surfaceOverride(),
            center.underOverride()
        );
    }

    private static int bracketedPlannedGround(
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

    private static int shelfWidth(String waterKind, double noise) {
        return switch (waterKind) {
            // Most low-gradient banks get a second dry shelf row toward land.
            // Water itself is never expanded; this only fills the visual
            // saw-tooth backward into the bank.
            case "River" -> noise >= 0.18 ? 2 : 1;
            case "Lake" -> noise >= 0.14 ? 2 : 1;
            case "Ocean" -> noise >= 0.30 ? 2 : 1;
            default -> 1;
        };
    }

    private static int materialBandWidth(String waterKind) {
        return switch (waterKind) {
            case "River" -> 2;
            case "Lake" -> 3;
            case "Ocean" -> 4;
            default -> 1;
        };
    }

    private static boolean isRocky(GeoWorldV1.RuntimeSample sample) {
        return "Mountain".equals(sample.terrainVariant()) ||
            "Canyon".equals(sample.terrainVariant()) ||
            "Alpine Rock".equals(sample.surfaceProfile()) ||
            "Desert Gravel".equals(sample.surfaceProfile());
    }

    private static String chooseMaterial(
        GeoWorldV1.RuntimeSample sample,
        String waterKind,
        int distance,
        long seed,
        int worldX,
        int worldZ
    ) {
        // Shore materials should read as short natural reaches, not
        // one-block salt-and-pepper noise. The broad field chooses the reach
        // material; the smaller field only bends its boundary.
        double patch =
            0.82 * smoothValueNoise(
                seed ^ 0x53484F52455F4D31L,
                worldX,
                worldZ,
                46.0
            ) +
            0.18 * smoothValueNoise(
                seed ^ 0x53484F52455F4D32L,
                worldX,
                worldZ,
                21.0
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

    private static String underMaterial(String surface) {
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

    private static double smoothValueNoise(
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

    private static double hashUnit(long seed, int x, int z) {
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

    private static double smoothstep(double t) {
        t = Math.max(0.0, Math.min(1.0, t));
        return t * t * (3.0 - 2.0 * t);
    }
}
