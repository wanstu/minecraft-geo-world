package geoworld.runtime;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.util.ArrayList;
import java.util.Base64;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.zip.GZIPInputStream;

public final class GeoWorldV1 {
    public static final int SCHEMA_VERSION = 1;
    public static final String GENERATOR_VERSION = "phase6.3-v1";
    public static final String FORMAT_MAGIC = "geoworld-model";
    public static final String FILE_ENCODING = "gzip+json";
    public static final String CHECKSUM_ALGORITHM = "sha256";
    private static final double RIVER_WET_CORE_FRACTION = 1.0;

    private static final String[] ECO_REGION_NAMES = {
        "Unknown",
        "Warm Ocean",
        "Temperate Ocean",
        "Cold Ocean",
        "Frozen Ocean",
        "Ice Sheet",
        "Tundra",
        "Alpine",
        "Boreal Forest",
        "Temperate Forest",
        "Temperate Grassland",
        "Temperate Desert",
        "Mediterranean Shrubland",
        "Subtropical Forest",
        "Hot Desert",
        "Savanna",
        "Tropical Monsoon Forest",
        "Tropical Rainforest",
        "Wetland"
    };

    private static final String[] BIOME_NAMES = ECO_REGION_NAMES;

    private static final String[] TERRAIN_NAMES = {
        "Unknown",
        "Ocean",
        "Plain",
        "Rolling Plain",
        "Hills",
        "Foothills",
        "Mountain",
        "Plateau",
        "Floodplain",
        "Alluvial Plain",
        "Alluvial Fan",
        "Canyon"
    };

    private static final String[] SURFACE_NAMES = {
        "Unknown",
        "Ocean Sand",
        "Ocean Gravel",
        "Frozen Ocean",
        "Ice / Snow",
        "Tundra Soil",
        "Alpine Rock",
        "Alpine Snow",
        "Forest Loam",
        "Podzol",
        "Grass Loam",
        "Dry Grass Soil",
        "Shrub Soil",
        "Desert Sand",
        "Desert Gravel",
        "Alluvial Silt",
        "Wet Mud",
        "Peat"
    };

    private static final String[] VEGETATION_NAMES = {
        "Unknown",
        "None",
        "Tundra Sparse",
        "Alpine Sparse",
        "Conifer Forest",
        "Mixed Forest",
        "Broadleaf Forest",
        "Dense Rainforest",
        "Monsoon Forest",
        "Grassland",
        "Dry Grassland",
        "Shrubland",
        "Savanna",
        "Desert Sparse",
        "Wetland"
    };

    public record Geometry(
        int width,
        int height,
        double originX,
        double originZ,
        double spanXBlocks,
        double spanZBlocks
    ) {
        public int nodeCount() {
            return Math.multiplyExact(width, height);
        }

        public double spacingX() {
            return spanXBlocks / (width - 1.0);
        }

        public double spacingZ() {
            return spanZBlocks / (height - 1.0);
        }

        public double cellSize() {
            double x = spacingX();
            double z = spacingZ();
            if (!finite(x) || !finite(z) || x <= 0 || z <= 0 || Math.abs(x - z) > 1e-9) {
                throw new IllegalArgumentException("V1 requires finite square macro cells");
            }
            return x;
        }
    }

    public record Manifest(
        String magic,
        int schemaVersion,
        String generatorVersion,
        String worldProfileVersion,
        String generationRulesVersion,
        String worldModelVersion,
        String preset,
        String parameterSnapshotSha256,
        String fileEncoding,
        String checksumAlgorithm,
        String payloadSha256,
        long payloadBytes,
        long seed,
        Geometry geometry,
        double seaLevelNormalized
    ) {}

    public record RuntimeSample(
        double groundHeight,
        String waterKind,
        boolean waterSurfaceDefined,
        Double waterSurfaceElevation,
        double waterDepth,
        double oceanEligibility,
        double riverWeight,
        int sourceNode,
        double sourceWeight,
        boolean sourceLand,
        String ecoRegion,
        double regionConfidence,
        String biomeFamily,
        String terrainVariant,
        String surfaceProfile,
        String vegetationProfile,
        double profileConfidence,
        boolean semanticFallback,
        boolean semanticMismatch
    ) {}

    public record ClimateSample(
        boolean available,
        double annualMeanTempC,
        double annualPrecipMM,
        double monsoonIndex,
        double windX,
        double windZ
    ) {}

    public record HydrologySample(
        boolean available,
        int watershedId,
        double flowAccumulation,
        int riverOrder,
        Double flowDirectionX,
        Double flowDirectionZ
    ) {}

    public record NaturalFeatureSample(
        Integer riverId,
        Double distanceToRiver,
        Double distanceToLake,
        Double distanceToCoast,
        Double distanceToOcean
    ) {}
    public record ParameterSnapshot(
        double mainlandScale, double regionScale, double transitionWidth,
        double mountainStrength, double plainContinuity, double plateauScale,
        double cliffRarity, double rareMacroTerrain, double riverDensity,
        double mainRiverScale, double lakeAbundance, double wetlandAbundance,
        double climateRegionScale, double climateTransitionWidth,
        double rainShadowStrength, double biomeContinuity, double caveRichness,
        double largeCaveRarity, double mineralRegionalization,
        double spawnPreference, double structureInfluence,
        double outerIslandAbundance, double polarTransitionWidth,
        double oceanClimateStrength, double prevailingWindStrength
    ) {
        public static ParameterSnapshot standard() {
            return new ParameterSnapshot(
                1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
                1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1
            );
        }

        public double[] values() {
            return new double[] {
                mainlandScale, regionScale, transitionWidth,
                mountainStrength, plainContinuity, plateauScale, cliffRarity,
                rareMacroTerrain, riverDensity, mainRiverScale, lakeAbundance,
                wetlandAbundance, climateRegionScale, climateTransitionWidth,
                rainShadowStrength, biomeContinuity, caveRichness,
                largeCaveRarity, mineralRegionalization, spawnPreference,
                structureInfluence, outerIslandAbundance, polarTransitionWidth,
                oceanClimateStrength, prevailingWindStrength
            };
        }
    }

    public record WorldProfile(
        String worldProfileVersion,
        String generationRulesVersion,
        String worldModelVersion,
        String preset,
        ParameterSnapshot parameters,
        boolean legacy
    ) {}

    private enum WaterKind {
        NONE("None"),
        OCEAN("Ocean"),
        LAKE("Lake"),
        RIVER("River");

        final String display;

        WaterKind(String display) {
            this.display = display;
        }
    }

    private record WorldSample(
        double groundHeight,
        WaterKind waterKind,
        boolean waterSurfaceDefined,
        double waterSurfaceElevation,
        double waterDepth,
        double oceanEligibility,
        double riverWeight
    ) {}

    private record GroundSample(
        double finalHeight,
        double riverWeight,
        double riverHalfWidth,
        boolean riverBedDefined,
        double riverBedElevation,
        boolean riverReceiverSurfaceDefined,
        double riverReceiverSurface,
        double riverReceiverWeight,
        WaterKind waterKind,
        boolean waterSurfaceDefined,
        double waterSurfaceElevation
    ) {}

    private record RiverSegment(
        int from,
        int to,
        double x0,
        double z0,
        double x1,
        double z1,
        double bed0,
        double bed1,
        double halfWidth,
        double meanderOffset,
        double[] profile,
        boolean receiverSurfaceDefined,
        double receiverSurface,
        boolean oceanMouth,
        boolean lakeMouth,
        boolean lakeOutlet,
        boolean lakeOutletSurfaceDefined,
        double lakeOutletSurface,
        int mouthExpansionVersion
    ) {}

    private record CoastSegment(
        double x0,
        double z0,
        double x1,
        double z1
    ) {}

    private record LakeNode(
        int index,
        int lakeId,
        double x,
        double z,
        double bedCeiling,
        double innerRadius,
        double outerRadius,
        double waterSurface,
        boolean coreCarve
    ) {}

    private record TerrainConfig(
        long seed,
        double plainAmplitude,
        double hillAmplitude,
        double foothillAmplitude,
        double mountainAmplitude,
        double mountainStrengthScale,
        double plateauAmplitude,
        double valleyDepth,
        double depositionLift,
        double alluvialFanLift,
        double canyonExtraDepth,
        double cliffFrequency
    ) {}

    private record WorldSampleConfig(
        double riverCenterDepthNormalized,
        double minRiverWaterDepthNormalized,
        double oceanEligibilityThreshold
    ) {}

    private record WaterbedConfig(
        double minRiverHalfWidthBlocks,
        double maxRiverHalfWidthBlocks,
        double minLakeWaterDepthNormalized
    ) {}

    private final Manifest manifest;
    private final WorldProfile worldProfile;
    private final Geometry geometry;
    private final boolean[] landMask;
    private final Terrain terrain;
    private final List<RiverSegment> rivers;
    private final List<LakeNode> lakes;
    private final double featureBucketSize;
    private final Map<Long, List<RiverSegment>> riverBuckets;
    private final Map<Long, List<LakeNode>> lakeBuckets;
    private final List<CoastSegment> coastSegments;
    private final WaterbedConfig waterbedConfig;
    private final ScalarGrid oceanEligibility;
    private final WorldSampleConfig worldSampleConfig;
    private final ScalarGrid climateTemp;
    private final ScalarGrid climatePrecip;
    private final ScalarGrid climateMonsoon;
    private final ScalarGrid climateWindX;
    private final ScalarGrid climateWindZ;
    private final int[] basinId;
    private final int[] flowTo;
    private final double[] flowAccumulation;
    private final byte[] riverOrder;
    private final byte[] ecoRegion;
    private final double[] regionConfidence;
    private final byte[] biomeFamily;
    private final byte[] terrainVariant;
    private final byte[] surfaceProfile;
    private final byte[] vegetationProfile;
    private final double[] profileConfidence;

    private GeoWorldV1(
        Manifest manifest,
        WorldProfile worldProfile,
        boolean[] landMask,
        Terrain terrain,
        List<RiverSegment> rivers,
        List<LakeNode> lakes,
        WaterbedConfig waterbedConfig,
        WorldSampleConfig worldSampleConfig,
        ScalarGrid climateTemp,
        ScalarGrid climatePrecip,
        ScalarGrid climateMonsoon,
        ScalarGrid climateWindX,
        ScalarGrid climateWindZ,
        int[] basinId,
        int[] flowTo,
        double[] flowAccumulation,
        byte[] riverOrder,
        byte[] ecoRegion,
        double[] regionConfidence,
        byte[] biomeFamily,
        byte[] terrainVariant,
        byte[] surfaceProfile,
        byte[] vegetationProfile,
        double[] profileConfidence
    ) {
        this.manifest = manifest;
        this.worldProfile = worldProfile;
        this.geometry = manifest.geometry();
        this.landMask = landMask;
        this.terrain = terrain;
        this.rivers = rivers;
        this.lakes = lakes;
        this.featureBucketSize = Math.max(64.0, this.geometry.cellSize());
        this.riverBuckets = this.buildRiverBuckets(rivers);
        this.lakeBuckets = this.buildLakeBuckets(lakes);
        this.coastSegments = this.buildCoastSegments();
        this.waterbedConfig = waterbedConfig;
        this.worldSampleConfig = worldSampleConfig;
        this.climateTemp = climateTemp;
        this.climatePrecip = climatePrecip;
        this.climateMonsoon = climateMonsoon;
        this.climateWindX = climateWindX;
        this.climateWindZ = climateWindZ;
        this.basinId = basinId;
        this.flowTo = flowTo;
        this.flowAccumulation = flowAccumulation;
        this.riverOrder = riverOrder;
        this.ecoRegion = ecoRegion;
        this.regionConfidence = regionConfidence;
        this.biomeFamily = biomeFamily;
        this.terrainVariant = terrainVariant;
        this.surfaceProfile = surfaceProfile;
        this.vegetationProfile = vegetationProfile;
        this.profileConfidence = profileConfidence;

        double[] ocean = new double[landMask.length];
        for (int i = 0; i < landMask.length; i++) {
            ocean[i] = landMask[i] ? 0.0 : 1.0;
        }
        this.oceanEligibility = new ScalarGrid(geometry, ocean);
    }

    private Map<Long, List<RiverSegment>> buildRiverBuckets(
        List<RiverSegment> segments
    ) {
        Map<Long, List<RiverSegment>> buckets = new HashMap<>();
        for (RiverSegment segment : segments) {
            double channelSupport =
                riverMaxHalfWidth(segment) +
                riverBankBlendBlocks(segment.halfWidth());
            double supportRadius =
                Math.max(
                    Math.max(channelSupport, riverFluvialReach(segment)),
                    riverDeltaSupportRadius(segment)
                ) +
                Math.abs(segment.meanderOffset());
            double minX =
                Math.min(segment.x0(), segment.x1()) - supportRadius;
            double maxX =
                Math.max(segment.x0(), segment.x1()) + supportRadius;
            double minZ =
                Math.min(segment.z0(), segment.z1()) - supportRadius;
            double maxZ =
                Math.max(segment.z0(), segment.z1()) + supportRadius;
            int minBX = this.featureBucketX(minX);
            int maxBX = this.featureBucketX(maxX);
            int minBZ = this.featureBucketZ(minZ);
            int maxBZ = this.featureBucketZ(maxZ);
            for (int bz = minBZ; bz <= maxBZ; bz++) {
                for (int bx = minBX; bx <= maxBX; bx++) {
                    buckets
                        .computeIfAbsent(
                            featureBucketKey(bx, bz),
                            ignored -> new ArrayList<>()
                        )
                        .add(segment);
                }
            }
        }
        return buckets;
    }

    private Map<Long, List<LakeNode>> buildLakeBuckets(
        List<LakeNode> nodes
    ) {
        Map<Long, List<LakeNode>> buckets = new HashMap<>();
        for (LakeNode lake : nodes) {
            int minBX = this.featureBucketX(lake.x() - lake.outerRadius());
            int maxBX = this.featureBucketX(lake.x() + lake.outerRadius());
            int minBZ = this.featureBucketZ(lake.z() - lake.outerRadius());
            int maxBZ = this.featureBucketZ(lake.z() + lake.outerRadius());
            for (int bz = minBZ; bz <= maxBZ; bz++) {
                for (int bx = minBX; bx <= maxBX; bx++) {
                    buckets
                        .computeIfAbsent(
                            featureBucketKey(bx, bz),
                            ignored -> new ArrayList<>()
                        )
                        .add(lake);
                }
            }
        }
        return buckets;
    }

    private List<CoastSegment> buildCoastSegments() {
        List<CoastSegment> segments = new ArrayList<>();
        double spacingX = this.geometry.spacingX();
        double spacingZ = this.geometry.spacingZ();
        for (int z = 0; z < this.geometry.height(); z++) {
            double nodeZ = this.geometry.originZ() + z * spacingZ;
            double z0 = Math.max(minZ(), nodeZ - spacingZ * 0.5);
            double z1 = Math.min(maxZ(), nodeZ + spacingZ * 0.5);
            for (int x = 0; x + 1 < this.geometry.width(); x++) {
                int left = z * this.geometry.width() + x;
                int right = left + 1;
                if (this.landMask[left] == this.landMask[right]) continue;
                double coastX = this.geometry.originX() + (x + 0.5) * spacingX;
                segments.add(new CoastSegment(coastX, z0, coastX, z1));
            }
        }
        for (int z = 0; z + 1 < this.geometry.height(); z++) {
            double coastZ = this.geometry.originZ() + (z + 0.5) * spacingZ;
            for (int x = 0; x < this.geometry.width(); x++) {
                int top = z * this.geometry.width() + x;
                int bottom = top + this.geometry.width();
                if (this.landMask[top] == this.landMask[bottom]) continue;
                double nodeX = this.geometry.originX() + x * spacingX;
                double x0 = Math.max(minX(), nodeX - spacingX * 0.5);
                double x1 = Math.min(maxX(), nodeX + spacingX * 0.5);
                segments.add(new CoastSegment(x0, coastZ, x1, coastZ));
            }
        }
        return List.copyOf(segments);
    }

    private List<RiverSegment> riverCandidates(double x, double z) {
        return this.riverBuckets.getOrDefault(
            featureBucketKey(this.featureBucketX(x), this.featureBucketZ(z)),
            List.of()
        );
    }

    private static boolean riverBankProtectedAt(
        List<RiverSegment> candidates,
        double x,
        double z
    ) {
        for (RiverSegment segment : candidates) {
            PointSegment point = pointRiverSegment(x, z, segment);
            double bankRadius =
                riverMouthHalfWidth(segment, point.t()) +
                riverBankBlendBlocks(segment.halfWidth());
            if (point.distance() < bankRadius) {
                return true;
            }
            DeltaPoint delta = riverDeltaPoint(
                segment,
                point.t(),
                x,
                z
            );
            if (delta != null && delta.distance() < delta.bankRadius()) {
                return true;
            }
        }
        return false;
    }

    private List<LakeNode> lakeCandidates(double x, double z) {
        return this.lakeBuckets.getOrDefault(
            featureBucketKey(this.featureBucketX(x), this.featureBucketZ(z)),
            List.of()
        );
    }

    private int featureBucketX(double x) {
        return (int)Math.floor(
            (x - this.geometry.originX()) / this.featureBucketSize
        );
    }

    private int featureBucketZ(double z) {
        return (int)Math.floor(
            (z - this.geometry.originZ()) / this.featureBucketSize
        );
    }

    private static long featureBucketKey(int x, int z) {
        return ((long)x << 32) ^ (z & 0xFFFFFFFFL);
    }

    public Manifest manifest() {
        return manifest;
    }

    public WorldProfile worldProfile() {
        return worldProfile;
    }

    public boolean hasClimatePreview() {
        return this.climateTemp != null &&
            this.climatePrecip != null &&
            this.climateMonsoon != null &&
            this.climateWindX != null &&
            this.climateWindZ != null;
    }

    public ClimateSample sampleClimate(double worldX, double worldZ) {
        if (!this.hasClimatePreview()) {
            return new ClimateSample(false, 0, 0, 0, 0, 0);
        }
        if (!this.worldProfile.legacy() && outsideModel(worldX, worldZ)) {
            return sampleOuterClimate(worldX, worldZ);
        }
        return new ClimateSample(
            true,
            this.climateTemp.sampleSmoothstepBilinear(worldX, worldZ),
            this.climatePrecip.sampleSmoothstepBilinear(worldX, worldZ),
            this.climateMonsoon.sampleSmoothstepBilinear(worldX, worldZ),
            this.climateWindX.sampleSmoothstepBilinear(worldX, worldZ),
            this.climateWindZ.sampleSmoothstepBilinear(worldX, worldZ)
        );
    }

    public HydrologySample sampleHydrology(double worldX, double worldZ) {
        if (
            this.basinId == null ||
            this.flowTo == null ||
            this.flowAccumulation == null ||
            this.riverOrder == null ||
            outsideModel(worldX, worldZ)
        ) {
            return new HydrologySample(false, 0, 0, 0, null, null);
        }
        double gx = clamp(
            (worldX - this.geometry.originX()) / this.geometry.spacingX(),
            0,
            this.geometry.width() - 1.0
        );
        double gz = clamp(
            (worldZ - this.geometry.originZ()) / this.geometry.spacingZ(),
            0,
            this.geometry.height() - 1.0
        );
        int x = clampInt((int)Math.round(gx), 0, this.geometry.width() - 1);
        int z = clampInt((int)Math.round(gz), 0, this.geometry.height() - 1);
        int node = z * this.geometry.width() + x;
        Double directionX = null;
        Double directionZ = null;
        int downstream = this.flowTo[node];
        if (downstream >= 0 && downstream < this.flowTo.length) {
            int downstreamX = downstream % this.geometry.width();
            int downstreamZ = downstream / this.geometry.width();
            double dx =
                (downstreamX - x) * this.geometry.spacingX();
            double dz =
                (downstreamZ - z) * this.geometry.spacingZ();
            double length = Math.hypot(dx, dz);
            if (length > 1.0e-9) {
                directionX = dx / length;
                directionZ = dz / length;
            }
        }
        return new HydrologySample(
            true,
            this.basinId[node],
            this.flowAccumulation[node],
            unsigned(this.riverOrder[node]),
            directionX,
            directionZ
        );
    }

    public NaturalFeatureSample sampleNaturalFeatures(
        double worldX,
        double worldZ
    ) {
        if (outsideModel(worldX, worldZ)) {
            RuntimeSample outer = sample(worldX, worldZ);
            return new NaturalFeatureSample(
                null,
                null,
                null,
                null,
                "Ocean".equals(outer.waterKind()) ? 0.0 : null
            );
        }

        Integer riverId = null;
        Double distanceToRiver = null;
        double bestRiver = Double.POSITIVE_INFINITY;
        for (int i = 0; i < this.rivers.size(); i++) {
            RiverSegment segment = this.rivers.get(i);
            PointSegment point = pointRiverSegment(worldX, worldZ, segment);
            double bankDistance = Math.max(
                0.0,
                point.distance() - riverMouthHalfWidth(segment, point.t())
            );
            if (bankDistance < bestRiver) {
                bestRiver = bankDistance;
                riverId = i + 1;
            }
        }
        if (riverId != null) {
            distanceToRiver = bestRiver;
        }

        Double distanceToLake = null;
        double bestLake = Double.POSITIVE_INFINITY;
        for (LakeNode lake : this.lakes) {
            double d = Math.max(
                0.0,
                lakeDistance(worldX - lake.x(), worldZ - lake.z()) -
                    lake.outerRadius()
            );
            bestLake = Math.min(bestLake, d);
        }
        if (Double.isFinite(bestLake)) {
            distanceToLake = bestLake;
        }

        Double distanceToCoast = null;
        double bestCoast = Double.POSITIVE_INFINITY;
        for (CoastSegment coast : this.coastSegments) {
            double d = pointSegment(
                worldX,
                worldZ,
                coast.x0(),
                coast.z0(),
                coast.x1(),
                coast.z1()
            ).distance();
            bestCoast = Math.min(bestCoast, d);
        }
        if (Double.isFinite(bestCoast)) {
            distanceToCoast = bestCoast;
        }

        RuntimeSample local = sample(worldX, worldZ);
        Double distanceToOcean = "Ocean".equals(local.waterKind())
            ? 0.0
            : distanceToCoast;

        return new NaturalFeatureSample(
            riverId,
            distanceToRiver,
            distanceToLake,
            distanceToCoast,
            distanceToOcean
        );
    }

    public RuntimeSample sample(double worldX, double worldZ) {
        if (!this.worldProfile.legacy() && outsideModel(worldX, worldZ)) {
            return sampleOuterOcean(worldX, worldZ);
        }
        WorldSample world = sampleWorld(worldX, worldZ);
        boolean wantLand = world.waterKind() != WaterKind.OCEAN;
        SourceSelection source = selectSourceNode(worldX, worldZ, wantLand);
        int node = source.node();

        int ecoCode = unsigned(ecoRegion[node]);
        int biomeCode = unsigned(biomeFamily[node]);
        int terrainCode = unsigned(terrainVariant[node]);
        int surfaceCode = unsigned(surfaceProfile[node]);
        int vegetationCode = unsigned(vegetationProfile[node]);

        boolean sourceLand = landMask[node];
        ProfileCodes zoned = applyElevationZoning(
            worldX,
            worldZ,
            world,
            sourceLand,
            terrainCode,
            biomeCode,
            surfaceCode,
            vegetationCode
        );
        biomeCode = zoned.biomeCode();
        surfaceCode = zoned.surfaceCode();
        vegetationCode = zoned.vegetationCode();

        boolean mismatch;
        if (world.waterKind() == WaterKind.OCEAN) {
            mismatch = sourceLand ||
                !isOceanFamily(biomeCode) ||
                terrainCode != 1 ||
                vegetationCode != 1;
        } else {
            mismatch = !sourceLand ||
                isOceanFamily(biomeCode) ||
                terrainCode == 1;
        }

        return new RuntimeSample(
            world.groundHeight(),
            world.waterKind().display,
            world.waterSurfaceDefined(),
            world.waterSurfaceDefined() ? world.waterSurfaceElevation() : null,
            world.waterDepth(),
            world.oceanEligibility(),
            world.riverWeight(),
            node,
            source.weight(),
            sourceLand,
            enumName(ECO_REGION_NAMES, ecoCode),
            regionConfidence[node],
            enumName(BIOME_NAMES, biomeCode),
            enumName(TERRAIN_NAMES, terrainCode),
            enumName(SURFACE_NAMES, surfaceCode),
            enumName(VEGETATION_NAMES, vegetationCode),
            profileConfidence[node],
            source.fallback(),
            mismatch
        );
    }

    private boolean outsideModel(double x, double z) {
        return x < minX() || x > maxX() || z < minZ() || z > maxZ();
    }

    private double minX() { return this.geometry.originX(); }
    private double maxX() { return this.geometry.originX() + this.geometry.spanXBlocks(); }
    private double minZ() { return this.geometry.originZ(); }
    private double maxZ() { return this.geometry.originZ() + this.geometry.spanZBlocks(); }

    private double clampModelX(double x) { return clamp(x, minX(), maxX()); }
    private double clampModelZ(double z) { return clamp(z, minZ(), maxZ()); }

    private double outsideDistance(double x, double z) {
        double dx = x < minX() ? minX() - x : x > maxX() ? x - maxX() : 0;
        double dz = z < minZ() ? minZ() - z : z > maxZ() ? z - maxZ() : 0;
        return Math.hypot(dx, dz);
    }

    private double polarFactor(double z) {
        double center = minZ() + this.geometry.spanZBlocks() * 0.5;
        double half = Math.max(1.0, this.geometry.spanZBlocks() * 0.5);
        double latitude = Math.abs((z - center) / half);
        double width = this.worldProfile.parameters().polarTransitionWidth();
        return smoothstepUnit(clamp((latitude - 0.72) / (0.40 * width), 0, 1));
    }

    private ClimateSample sampleOuterClimate(double x, double z) {
        double edgeX = clampModelX(x);
        double edgeZ = clampModelZ(z);
        double polar = polarFactor(z);
        double temp = this.climateTemp.sampleSmoothstepBilinear(edgeX, edgeZ);
        temp = lerp(temp, -18.0, polar);
        double precip = this.climatePrecip.sampleSmoothstepBilinear(edgeX, edgeZ);
        double monsoon = this.climateMonsoon.sampleSmoothstepBilinear(edgeX, edgeZ) * (1 - 0.75 * polar);
        double windX = this.climateWindX.sampleSmoothstepBilinear(edgeX, edgeZ);
        double windZ = this.climateWindZ.sampleSmoothstepBilinear(edgeX, edgeZ);
        return new ClimateSample(true, temp, precip, monsoon, windX, windZ);
    }

    private double guaranteedOuterIslandShape(
        double x,
        double z,
        double transition
    ) {
        String rules = this.worldProfile.generationRulesVersion();
        if ("geoworld-rules-v1.1".equals(rules)) {
            return guaranteedOuterIslandShapeV11(x, z, transition);
        }
        if ("geoworld-rules-v1.1.1".equals(rules)) {
            return guaranteedOuterIslandShapeV111(x, z, transition);
        }

        double centerZ = minZ() + this.geometry.spanZBlocks() * 0.5;
        double best = 0;
        for (int side = 0; side < 2; side++) {
            long key = side + 1L;
            double distanceJitter =
                0.5 + 0.5 * hashValue(this.manifest.seed() + 77201L, key, 11L);
            double zJitter =
                hashValue(this.manifest.seed() + 77203L, key, 17L);
            double radiusJitter =
                0.5 + 0.5 * hashValue(this.manifest.seed() + 77207L, key, 23L);
            double aspectJitter =
                0.5 + 0.5 * hashValue(this.manifest.seed() + 77211L, key, 29L);
            double rotation =
                Math.PI * hashValue(this.manifest.seed() + 77213L, key, 31L);

            double offshore =
                transition * (0.26 + 0.12 * distanceJitter);
            double anchorX = side == 0
                ? minX() - offshore
                : maxX() + offshore;
            double anchorZ =
                centerZ + zJitter * 0.22 * this.geometry.spanZBlocks();

            double radiusScale = clamp(
                0.75 +
                    0.25 *
                        this.worldProfile.parameters().outerIslandAbundance(),
                0.60,
                1.40
            );
            double radius = (420.0 + 300.0 * radiusJitter) * radiusScale;
            double aspect = 0.64 + 0.48 * aspectJitter;

            double warpX =
                radius * 0.16 * valueNoise(
                    this.manifest.seed() + 77217L + key * 97L,
                    x,
                    z,
                    540.0
                );
            double warpZ =
                radius * 0.13 * valueNoise(
                    this.manifest.seed() + 77219L + key * 101L,
                    x,
                    z,
                    470.0
                );
            double dx = x - anchorX + warpX;
            double dz = z - anchorZ + warpZ;
            double c = Math.cos(rotation);
            double sin = Math.sin(rotation);
            double rx = dx * c + dz * sin;
            double rz = -dx * sin + dz * c;

            double mainScore =
                1.0 - Math.hypot(rx / radius, rz / (radius * aspect));

            double lobe1X =
                radius *
                (0.24 + 0.10 * hashValue(this.manifest.seed() + 77223L, key, 37L));
            double lobe1Z =
                radius * aspect * 0.26 *
                hashValue(this.manifest.seed() + 77229L, key, 41L);
            double lobe1Score =
                0.78 - Math.hypot(
                    (rx - lobe1X) / (radius * 0.62),
                    (rz - lobe1Z) / (radius * aspect * 0.58)
                );

            double lobe2X =
                -radius *
                (
                    0.20 +
                    0.12 *
                        (
                            0.5 +
                            0.5 *
                                hashValue(
                                    this.manifest.seed() + 77231L,
                                    key,
                                    43L
                                )
                        )
                );
            double lobe2Z =
                radius * aspect * 0.30 *
                hashValue(this.manifest.seed() + 77237L, key, 47L);
            double lobe2Score =
                0.73 - Math.hypot(
                    (rx - lobe2X) / (radius * 0.55),
                    (rz - lobe2Z) / (radius * aspect * 0.52)
                );

            double score = Math.max(
                mainScore,
                Math.max(lobe1Score, lobe2Score)
            );
            score +=
                0.12 * valueNoise(
                    this.manifest.seed() + 77241L + key * 109L,
                    x,
                    z,
                    260.0
                ) +
                0.05 * valueNoise(
                    this.manifest.seed() + 77243L + key * 113L,
                    x,
                    z,
                    125.0
                );
            double shape = smoothstep(-0.24, 0.24, score);
            best = Math.max(best, shape);
        }
        return clamp(best, 0, 1);
    }

    private double guaranteedOuterIslandShapeV111(
        double x,
        double z,
        double transition
    ) {
        double centerZ = minZ() + this.geometry.spanZBlocks() * 0.5;
        double best = 0;
        for (int side = 0; side < 2; side++) {
            long key = side + 1L;
            double distanceJitter =
                0.5 + 0.5 * hashValue(this.manifest.seed() + 77201L, key, 11L);
            double zJitter =
                hashValue(this.manifest.seed() + 77203L, key, 17L);
            double radiusJitter =
                0.5 + 0.5 * hashValue(this.manifest.seed() + 77207L, key, 23L);
            double aspectJitter =
                0.5 + 0.5 * hashValue(this.manifest.seed() + 77211L, key, 29L);
            double phase =
                Math.PI * hashValue(this.manifest.seed() + 77213L, key, 31L);

            double offshore =
                transition * (0.26 + 0.12 * distanceJitter);
            double anchorX = side == 0
                ? minX() - offshore
                : maxX() + offshore;
            double anchorZ =
                centerZ + zJitter * 0.22 * this.geometry.spanZBlocks();
            double radiusScale = clamp(
                0.75 +
                    0.25 *
                        this.worldProfile.parameters().outerIslandAbundance(),
                0.60,
                1.40
            );
            double radius = (360.0 + 260.0 * radiusJitter) * radiusScale;
            double aspect = 0.68 + 0.42 * aspectJitter;
            double dx = x - anchorX;
            double dz = z - anchorZ;
            double angle = Math.atan2(dz, dx);
            double coastWarp = 1.0 +
                0.14 * valueNoise(
                    this.manifest.seed() + 77217L + key * 97L,
                    x,
                    z,
                    420.0
                ) +
                0.08 * Math.sin(3.0 * angle + phase) +
                0.05 * Math.sin(5.0 * angle - 0.7 * phase);
            coastWarp = clamp(coastWarp, 0.72, 1.28);
            double normalizedDistance = Math.hypot(
                dx / radius,
                dz / (radius * aspect)
            ) / coastWarp;
            double shape =
                1.0 - smoothstep(0.38, 1.0, normalizedDistance);
            best = Math.max(best, shape);
        }
        return clamp(best, 0, 1);
    }
    private double guaranteedOuterIslandShapeV11(
        double x,
        double z,
        double transition
    ) {
        double centerZ = minZ() + this.geometry.spanZBlocks() * 0.5;
        double best = 0;
        for (int side = 0; side < 2; side++) {
            long key = side + 1L;
            double distanceJitter =
                0.5 + 0.5 * hashValue(this.manifest.seed() + 77201L, key, 11L);
            double zJitter =
                hashValue(this.manifest.seed() + 77203L, key, 17L);
            double radiusJitter =
                0.5 + 0.5 * hashValue(this.manifest.seed() + 77207L, key, 23L);

            double offshore =
                transition * (0.26 + 0.12 * distanceJitter);
            double anchorX = side == 0
                ? minX() - offshore
                : maxX() + offshore;
            double anchorZ =
                centerZ + zJitter * 0.22 * this.geometry.spanZBlocks();
            double radiusScale = clamp(
                0.75 +
                    0.25 *
                        this.worldProfile.parameters().outerIslandAbundance(),
                0.60,
                1.40
            );
            double radius = (360.0 + 260.0 * radiusJitter) * radiusScale;
            double distance = Math.hypot(x - anchorX, z - anchorZ);
            double shape =
                1.0 - smoothstep(radius * 0.35, radius, distance);
            best = Math.max(best, shape);
        }
        return clamp(best, 0, 1);
    }

    // Keep this formula in exact parity with internal/runtimeworld.
    // v1.1.7 enriches the infinite exterior with basin-scale relief plus
    // sparse ridges, seamounts and trenches. Older locked rules preserve the
    // historical single-noise seafloor exactly.
    private double outerOceanSeafloor(
        double x,
        double z,
        double sea,
        double deep
    ) {
        double legacy =
            deep +
            0.018 *
                valueNoise(
                    this.manifest.seed() + 77003L,
                    x,
                    z,
                    5200.0
                );
        String generationRulesVersion =
            this.worldProfile.generationRulesVersion();
        if (
            !"geoworld-rules-v1.1.7".equals(generationRulesVersion) &&
            !"geoworld-rules-v1.1.8".equals(generationRulesVersion) &&
            !"geoworld-rules-v1.1.9".equals(generationRulesVersion) &&
            !"geoworld-rules-v1.1.10".equals(generationRulesVersion)
        ) {
            return legacy;
        }

        double basin =
            0.020 *
                valueNoise(
                    this.manifest.seed() + 77011L,
                    x,
                    z,
                    16000.0
                ) +
            0.010 *
                valueNoise(
                    this.manifest.seed() + 77017L,
                    x,
                    z,
                    7200.0
                );

        double ridgeLine =
            1.0 -
            Math.abs(
                valueNoise(
                    this.manifest.seed() + 77021L,
                    x,
                    z,
                    11500.0
                )
            );
        double ridgeGate = smoothstep(
            0.08,
            0.58,
            valueNoise(
                this.manifest.seed() + 77023L,
                x,
                z,
                24000.0
            )
        );
        double ridge =
            0.030 *
            smoothstep(0.82, 0.98, ridgeLine) *
            ridgeGate;

        double seamountSignal =
            (0.5 +
                0.5 *
                    valueNoise(
                        this.manifest.seed() + 77031L,
                        x,
                        z,
                        4800.0
                    )) *
            (0.5 +
                0.5 *
                    valueNoise(
                        this.manifest.seed() + 77033L,
                        x,
                        z,
                        2600.0
                    ));
        double seamountGate = smoothstep(
            0.20,
            0.70,
            valueNoise(
                this.manifest.seed() + 77037L,
                x,
                z,
                18000.0
            )
        );
        double seamount =
            0.045 *
            smoothstep(0.78, 0.95, seamountSignal) *
            seamountGate;

        double trenchLine =
            1.0 -
            Math.abs(
                valueNoise(
                    this.manifest.seed() + 77041L,
                    x,
                    z,
                    13000.0
                )
            );
        double trenchGate = smoothstep(
            0.18,
            0.70,
            valueNoise(
                this.manifest.seed() + 77043L,
                x,
                z,
                22000.0
            )
        );
        double trench =
            0.035 *
            smoothstep(0.86, 0.985, trenchLine) *
            trenchGate;

        double detail =
            0.006 *
            valueNoise(
                this.manifest.seed() + 77051L,
                x,
                z,
                2800.0
            );
        double floor =
            deep + basin + ridge + seamount - trench + detail;
        return clamp(
            floor,
            Math.max(0.06, deep - 0.09),
            sea - 0.035
        );
    }

    private RuntimeSample sampleOuterOcean(double x, double z) {
        double edgeX = clampModelX(x);
        double edgeZ = clampModelZ(z);
        WorldSample edge = sampleWorld(edgeX, edgeZ);
        double distance = outsideDistance(x, z);
        double transition = Math.max(
            2048.0,
            this.geometry.cellSize() * 24.0 * this.worldProfile.parameters().transitionWidth()
        );
        double t = smoothstepUnit(clamp(distance / transition, 0, 1));
        double sea = this.manifest.seaLevelNormalized();
        double deep = Math.max(0.08, sea - 0.18);
        double seafloor = outerOceanSeafloor(x, z, sea, deep);
        double edgeFloor = Math.min(edge.groundHeight(), sea - 0.004);
        double ground = lerp(edgeFloor, seafloor, t);

        double polar = polarFactor(z);
        ClimateSample climate = sampleOuterClimate(x, z);
        String islandRules = this.worldProfile.generationRulesVersion();
        boolean legacyOuterIslands =
            islandRules.isEmpty() ||
            "geoworld-rules-v1.1".equals(islandRules);
        boolean radialOuterIslands =
            "geoworld-rules-v1.1.1".equals(islandRules);
        boolean naturalOuterIslandsV112 =
            "geoworld-rules-v1.1.2".equals(islandRules);
        boolean naturalOuterIslands =
            !legacyOuterIslands &&
            !radialOuterIslands &&
            !naturalOuterIslandsV112;
        boolean naturalOuterIslandFamily =
            naturalOuterIslandsV112 || naturalOuterIslands;

        // Sparse offshore islands live only in the near/far-ocean transition.
        // Their density fades with distance so the infinite exterior can never
        // rebuild a second continent. Polar suppression leaves the north/south
        // extreme as uninterrupted frozen ocean.
        double abundance = this.worldProfile.parameters().outerIslandAbundance();
        double islandNoise = 0.68 * valueNoise(
            this.manifest.seed() + 77101L, x, z, 1800.0
        ) + 0.32 * valueNoise(
            this.manifest.seed() + 77123L, x, z, 720.0
        );
        double islandSignal = 0.5 + 0.5 * islandNoise;
        double islandThreshold = clamp(
            0.86 - 0.08 * (abundance - 1.0),
            0.70,
            0.94
        );
        double islandShape = smoothstep(
            islandThreshold,
            Math.min(0.995, islandThreshold + 0.10),
            islandSignal
        );
        // Two deterministic shelf-linked anchor islands guarantee that the
        // normal island system remains observable for every seed without
        // filling the exterior ocean. Broad noise still creates the rest.
        islandShape = Math.max(
            islandShape,
            guaranteedOuterIslandShape(x, z, transition)
        );
        double distanceFade = 1.0 - smoothstepUnit(clamp(
            (distance - transition * 0.20) / (transition * 2.20),
            0,
            1
        ));
        double polarSuppression = 1.0 - smoothstepUnit(clamp(
            (polar - 0.35) / 0.45,
            0,
            1
        ));
        double islandWeight = clamp(
            islandShape * distanceFade * polarSuppression,
            0,
            1
        );
        double islandRelief = 0.0;
        if (islandWeight > 0) {
            if (naturalOuterIslands) {
                double terrainWarpX =
                    110.0 * valueNoise(
                        this.manifest.seed() + 77301L,
                        x,
                        z,
                        780.0
                    );
                double terrainWarpZ =
                    95.0 * valueNoise(
                        this.manifest.seed() + 77303L,
                        x,
                        z,
                        690.0
                    );
                double tx = x + terrainWarpX;
                double tz = z + terrainWarpZ;
                double broad =
                    0.56 * valueNoise(
                        this.manifest.seed() + 77311L,
                        tx,
                        tz,
                        980.0
                    ) +
                    0.29 * valueNoise(
                        this.manifest.seed() + 77317L,
                        tx,
                        tz,
                        470.0
                    ) +
                    0.15 * valueNoise(
                        this.manifest.seed() + 77319L,
                        tx,
                        tz,
                        235.0
                    );
                double ridge =
                    1.0 -
                    Math.abs(
                        valueNoise(
                            this.manifest.seed() + 77323L,
                            tx,
                            tz,
                            720.0
                        )
                    );
                ridge *= ridge;
                islandRelief = clamp(
                    0.44 + 0.30 * broad + 0.20 * ridge,
                    0,
                    1
                );
                double detail = valueNoise(
                    this.manifest.seed() + 77329L,
                    tx,
                    tz,
                    170.0
                );
                double coastField =
                    0.72 * valueNoise(
                        this.manifest.seed() + 77331L,
                        x,
                        z,
                        920.0
                    ) +
                    0.28 * valueNoise(
                        this.manifest.seed() + 77337L,
                        x,
                        z,
                        430.0
                    );
                double cliffSector =
                    smoothstep(0.58, 0.84, coastField);
                double gentleRise = clamp(
                    0.004 + 0.030 * islandRelief + 0.004 * detail,
                    0.003,
                    0.040
                );
                double cliffRise = clamp(
                    0.020 + 0.070 * islandRelief + 0.008 * detail,
                    0.016,
                    0.095
                );
                double gentleRamp =
                    smoothstep(0.12, 0.80, islandWeight);
                double cliffRamp =
                    smoothstep(0.05, 0.34, islandWeight);
                double coastRamp =
                    lerp(gentleRamp, cliffRamp, cliffSector);
                double rise =
                    lerp(gentleRise, cliffRise, cliffSector);
                double target = sea + rise * coastRamp;
                double blend =
                    smoothstep(0.06, 0.48, islandWeight);
                ground = lerp(ground, target, blend);
            } else if (naturalOuterIslandsV112) {
                double terrainWarpX =
                    110.0 * valueNoise(
                        this.manifest.seed() + 77301L,
                        x,
                        z,
                        780.0
                    );
                double terrainWarpZ =
                    95.0 * valueNoise(
                        this.manifest.seed() + 77303L,
                        x,
                        z,
                        690.0
                    );
                double tx = x + terrainWarpX;
                double tz = z + terrainWarpZ;
                double broad =
                    0.56 * valueNoise(
                        this.manifest.seed() + 77311L,
                        tx,
                        tz,
                        980.0
                    ) +
                    0.29 * valueNoise(
                        this.manifest.seed() + 77317L,
                        tx,
                        tz,
                        470.0
                    ) +
                    0.15 * valueNoise(
                        this.manifest.seed() + 77319L,
                        tx,
                        tz,
                        235.0
                    );
                double ridge =
                    1.0 -
                    Math.abs(
                        valueNoise(
                            this.manifest.seed() + 77323L,
                            tx,
                            tz,
                            720.0
                        )
                    );
                ridge *= ridge;
                islandRelief = clamp(
                    0.48 + 0.34 * broad + 0.24 * ridge,
                    0,
                    1
                );
                double detail = valueNoise(
                    this.manifest.seed() + 77329L,
                    tx,
                    tz,
                    170.0
                );
                double rise = clamp(
                    0.008 + 0.038 * islandRelief + 0.006 * detail,
                    0.006,
                    0.060
                );
                double coastCharacter =
                    0.5 +
                    0.5 *
                        valueNoise(
                            this.manifest.seed() + 77331L,
                            tx,
                            tz,
                            1100.0
                        );
                double cliffSector =
                    smoothstep(0.72, 0.90, coastCharacter);
                double coastStart = lerp(0.20, 0.08, cliffSector);
                double coastFull = lerp(0.72, 0.48, cliffSector);
                double coastRamp =
                    smoothstep(coastStart, coastFull, islandWeight);
                double target = sea + rise * coastRamp;
                double blend =
                    smoothstep(0.08, 0.52, islandWeight);
                ground = lerp(ground, target, blend);
            } else if (radialOuterIslands) {
                double relief = 0.5 + 0.5 * valueNoise(
                    this.manifest.seed() + 77147L, x, z, 260.0
                );
                double islandTop = sea + 0.012 +
                    0.060 * Math.pow(islandWeight, 1.15) +
                    0.018 * relief * islandWeight;
                ground = lerp(ground, islandTop, islandWeight);
            } else {
                double islandTop = sea + 0.018 + 0.082 * islandWeight;
                ground = lerp(ground, islandTop, islandWeight);
            }
        }
        boolean islandLand =
            islandWeight >= 0.58 && ground > sea + 0.002;
        if (radialOuterIslands) {
            islandLand = islandWeight > 0.18 && ground > sea + 0.001;
        } else if (naturalOuterIslandsV112) {
            islandLand = islandWeight > 0.14 && ground > sea + 0.001;
        } else if (naturalOuterIslands) {
            islandLand = islandWeight > 0.20 && ground > sea + 0.0015;
        }

        if (islandLand) {
            int biomeCode;
            int surfaceCode;
            int vegetationCode;
            if (polar > 0.28 || climate.annualMeanTempC() < 1.0) {
                biomeCode = 6; // Tundra
                surfaceCode = 5;
                vegetationCode = 2;
            } else if (climate.annualMeanTempC() < 9.0) {
                biomeCode = 8; // Boreal Forest
                surfaceCode = 9;
                vegetationCode = 4;
            } else if (
                climate.annualMeanTempC() > 22.0 &&
                climate.annualPrecipMM() > 1400.0
            ) {
                biomeCode = 17; // Tropical Rainforest
                surfaceCode = 8;
                vegetationCode = 7;
            } else if (climate.annualMeanTempC() > 18.0) {
                biomeCode = 15; // Savanna
                surfaceCode = 11;
                vegetationCode = 12;
            } else {
                biomeCode = 10; // Temperate Grassland
                surfaceCode = 10;
                vegetationCode = 9;
            }
            int terrainCode = 2;
            if (naturalOuterIslandFamily) {
                if (islandRelief > 0.60) {
                    terrainCode = 4;
                } else if (islandRelief > 0.38) {
                    terrainCode = 3;
                }
            } else if (islandWeight > 0.82) {
                terrainCode = 4;
            }
            double maxIslandHeight =
                naturalOuterIslandFamily ? 0.90 : 0.72;
            return new RuntimeSample(
                clamp(ground, sea + 0.002, maxIslandHeight),
                WaterKind.NONE.display,
                false,
                null,
                0.0,
                0.0,
                0.0,
                -1,
                islandWeight,
                true,
                enumName(ECO_REGION_NAMES, biomeCode),
                islandWeight,
                enumName(BIOME_NAMES, biomeCode),
                enumName(TERRAIN_NAMES, terrainCode),
                enumName(SURFACE_NAMES, surfaceCode),
                enumName(VEGETATION_NAMES, vegetationCode),
                islandWeight,
                false,
                false
            );
        }

        if (naturalOuterIslandFamily) {
            ground = Math.min(ground, sea - 0.004);
        } else if (radialOuterIslands) {
            ground = Math.min(ground, sea - 0.002);
        } else {
            ground = Math.min(ground, sea - 0.012);
        }
        SourceSelection source = selectSourceNode(edgeX, edgeZ, false);
        int biomeCode;
        int surfaceCode;
        if (polar >= 0.72) {
            biomeCode = 4; // Frozen Ocean
            surfaceCode = 3;
        } else if (polar >= 0.34 || climate.annualMeanTempC() < 8.0) {
            biomeCode = 3; // Cold Ocean
            surfaceCode = 2;
        } else if (climate.annualMeanTempC() > 20.0) {
            biomeCode = 1; // Warm Ocean
            surfaceCode = 1;
        } else {
            biomeCode = 2; // Temperate Ocean
            surfaceCode = 2;
        }

        return new RuntimeSample(
            ground,
            WaterKind.OCEAN.display,
            true,
            sea,
            sea - ground,
            1.0,
            0.0,
            source.node(),
            source.weight(),
            false,
            enumName(ECO_REGION_NAMES, biomeCode),
            1.0,
            enumName(BIOME_NAMES, biomeCode),
            enumName(TERRAIN_NAMES, 1),
            enumName(SURFACE_NAMES, surfaceCode),
            enumName(VEGETATION_NAMES, 1),
            1.0,
            false,
            false
        );
    }

    public static GeoWorldV1 load(Path path) throws IOException {
        byte[] compressed = Files.readAllBytes(path);
        String json;
        try (
            GZIPInputStream gzip = new GZIPInputStream(new ByteArrayInputStream(compressed));
            ByteArrayOutputStream out = new ByteArrayOutputStream()
        ) {
            gzip.transferTo(out);
            json = out.toString(StandardCharsets.UTF_8);
        }

        Json.Root parsed = Json.parseEnvelope(json);
        Map<String, Object> envelope = parsed.object();
        Map<String, Object> manifestObject = Json.object(Json.required(envelope, "manifest"), "manifest");
        Manifest manifest = parseManifest(manifestObject);
        validateManifest(manifest);

        byte[] rawPayload = parsed.rawPayload().getBytes(StandardCharsets.UTF_8);
        if (rawPayload.length != manifest.payloadBytes()) {
            throw new IllegalArgumentException(
                "payloadBytes mismatch: file=" + rawPayload.length + " manifest=" + manifest.payloadBytes()
            );
        }
        String checksum = sha256Hex(rawPayload);
        if (!checksum.equals(manifest.payloadSha256())) {
            throw new IllegalArgumentException(
                "payload SHA-256 mismatch: got=" + checksum + " want=" + manifest.payloadSha256()
            );
        }

        Map<String, Object> payload = Json.object(Json.required(envelope, "payload"), "payload");
        Map<String, Object> config = Json.object(Json.required(payload, "config"), "payload.config");
        validateResolvedConfig(config, manifest);
        WorldProfile worldProfile = parseWorldProfile(config, manifest);

        Map<String, Object> surface = Json.object(Json.required(payload, "surface"), "payload.surface");
        boolean[] landMask = booleanArray(Json.required(surface, "landMask"), "surface.landMask");
        boolean[] enclosedBelowSea = booleanArray(
            Json.required(surface, "enclosedBelowSea"),
            "surface.enclosedBelowSea"
        );
        validateSurface(manifest.geometry(), landMask, enclosedBelowSea);

        Terrain terrain = parseTerrain(
            manifest.geometry(),
            Json.object(Json.required(payload, "terrain"), "payload.terrain")
        );

        Map<String, Object> waterbedObject = Json.object(
            Json.required(payload, "waterbed"),
            "payload.waterbed"
        );
        Map<String, Object> waterbedConfigObject = Json.object(
            Json.required(waterbedObject, "config"),
            "waterbed.config"
        );
        WaterbedConfig waterbedConfig = new WaterbedConfig(
            Json.doubleValue(
                waterbedConfigObject,
                "MinRiverHalfWidthBlocks"
            ),
            Json.doubleValue(
                waterbedConfigObject,
                "MaxRiverHalfWidthBlocks"
            ),
            waterbedConfigObject.containsKey("MinLakeWaterDepthNormalized")
                ? Json.doubleValue(
                    waterbedConfigObject,
                    "MinLakeWaterDepthNormalized"
                )
                : 0.003
        );
        List<RiverSegment> rivers = parseRivers(
            Json.arrayOrEmpty(
                Json.required(waterbedObject, "rivers"),
                "waterbed.rivers"
            )
        );
        List<LakeNode> lakes = parseLakes(
            Json.arrayOrEmpty(
                Json.required(waterbedObject, "lakes"),
                "waterbed.lakes"
            )
        );
        validateWaterbed(manifest.geometry(), rivers, lakes);

        Map<String, Object> worldSampleObject = Json.object(
            Json.required(config, "WorldSample"),
            "config.WorldSample"
        );
        Object minRiverWaterDepthRaw =
            worldSampleObject.get("MinRiverWaterDepthNormalized");
        WorldSampleConfig worldSampleConfig = new WorldSampleConfig(
            Json.doubleValue(worldSampleObject, "RiverCenterDepthNormalized"),
            minRiverWaterDepthRaw == null
                ? 0.0015
                : Json.doubleValue(
                    worldSampleObject,
                    "MinRiverWaterDepthNormalized"
                ),
            Json.doubleValue(worldSampleObject, "OceanEligibilityThreshold")
        );

        ScalarGrid climateTemp = null;
        ScalarGrid climatePrecip = null;
        ScalarGrid climateMonsoon = null;
        ScalarGrid climateWindX = null;
        ScalarGrid climateWindZ = null;
        Object climatePreviewRaw = payload.get("climatePreview");
        if (climatePreviewRaw != null) {
            Map<String, Object> climatePreview = Json.object(
                climatePreviewRaw,
                "payload.climatePreview"
            );
            if (climatePreview.get("annualMeanTempC") != null) {
                double[] temp = doubleArray(
                    Json.required(climatePreview, "annualMeanTempC"),
                    "climatePreview.annualMeanTempC"
                );
                double[] precip = doubleArray(
                    Json.required(climatePreview, "annualPrecipMM"),
                    "climatePreview.annualPrecipMM"
                );
                double[] monsoon = doubleArray(
                    Json.required(climatePreview, "monsoonIndex"),
                    "climatePreview.monsoonIndex"
                );
                double[] windX = doubleArray(
                    Json.required(climatePreview, "windX"),
                    "climatePreview.windX"
                );
                double[] windZ = doubleArray(
                    Json.required(climatePreview, "windZ"),
                    "climatePreview.windZ"
                );
                int climateN = manifest.geometry().nodeCount();
                requireLength(
                    "climatePreview.annualMeanTempC",
                    temp.length,
                    climateN
                );
                requireLength(
                    "climatePreview.annualPrecipMM",
                    precip.length,
                    climateN
                );
                requireLength(
                    "climatePreview.monsoonIndex",
                    monsoon.length,
                    climateN
                );
                requireLength(
                    "climatePreview.windX",
                    windX.length,
                    climateN
                );
                requireLength(
                    "climatePreview.windZ",
                    windZ.length,
                    climateN
                );
                climateTemp = new ScalarGrid(manifest.geometry(), temp);
                climatePrecip = new ScalarGrid(manifest.geometry(), precip);
                climateMonsoon = new ScalarGrid(manifest.geometry(), monsoon);
                climateWindX = new ScalarGrid(manifest.geometry(), windX);
                climateWindZ = new ScalarGrid(manifest.geometry(), windZ);
            }
        }

        int[] basinId = null;
        int[] flowTo = null;
        double[] flowAccumulation = null;
        byte[] riverOrder = null;
        Object hydrologyQueryRaw = payload.get("hydrologyQuery");
        if (hydrologyQueryRaw != null) {
            Map<String, Object> hydrologyQuery = Json.object(
                hydrologyQueryRaw,
                "payload.hydrologyQuery"
            );
            basinId = intArray(
                Json.required(hydrologyQuery, "basinId"),
                "hydrologyQuery.basinId"
            );
            flowTo = intArray(
                Json.required(hydrologyQuery, "flowTo"),
                "hydrologyQuery.flowTo"
            );
            flowAccumulation = doubleArray(
                Json.required(hydrologyQuery, "flowAccumulation"),
                "hydrologyQuery.flowAccumulation"
            );
            riverOrder = byteSlice(
                Json.required(hydrologyQuery, "riverOrder"),
                "hydrologyQuery.riverOrder"
            );
            int hydrologyN = manifest.geometry().nodeCount();
            requireLength("hydrologyQuery.basinId", basinId.length, hydrologyN);
            requireLength("hydrologyQuery.flowTo", flowTo.length, hydrologyN);
            requireLength(
                "hydrologyQuery.flowAccumulation",
                flowAccumulation.length,
                hydrologyN
            );
            requireLength("hydrologyQuery.riverOrder", riverOrder.length, hydrologyN);
            for (int i = 0; i < hydrologyN; i++) {
                if (
                    basinId[i] < 0 ||
                    flowTo[i] < -1 ||
                    flowTo[i] >= hydrologyN ||
                    flowAccumulation[i] < 0
                ) {
                    throw new IllegalArgumentException(
                        "invalid hydrology query value at node " + i
                    );
                }
            }
        }

        Map<String, Object> ecology = Json.object(Json.required(payload, "ecology"), "payload.ecology");
        boolean[] ecologyLandMask = booleanArray(Json.required(ecology, "landMask"), "ecology.landMask");
        byte[] ecoRegion = byteSlice(Json.required(ecology, "ecoRegion"), "ecology.ecoRegion");
        double[] regionConfidence = doubleArray(
            Json.required(ecology, "regionConfidence"),
            "ecology.regionConfidence"
        );

        Map<String, Object> profile = Json.object(Json.required(payload, "profile"), "payload.profile");
        boolean[] profileLandMask = booleanArray(Json.required(profile, "landMask"), "profile.landMask");
        byte[] biomeFamily = byteSlice(Json.required(profile, "biomeFamily"), "profile.biomeFamily");
        byte[] terrainVariant = byteSlice(
            Json.required(profile, "terrainVariant"),
            "profile.terrainVariant"
        );
        byte[] surfaceProfile = byteSlice(
            Json.required(profile, "surfaceProfile"),
            "profile.surfaceProfile"
        );
        byte[] vegetationProfile = byteSlice(
            Json.required(profile, "vegetationProfile"),
            "profile.vegetationProfile"
        );
        double[] profileConfidence = doubleArray(
            Json.required(profile, "profileConfidence"),
            "profile.profileConfidence"
        );

        int n = manifest.geometry().nodeCount();
        requireLength("ecology.landMask", ecologyLandMask.length, n);
        requireLength("ecology.ecoRegion", ecoRegion.length, n);
        requireLength("ecology.regionConfidence", regionConfidence.length, n);
        requireLength("profile.landMask", profileLandMask.length, n);
        requireLength("profile.biomeFamily", biomeFamily.length, n);
        requireLength("profile.terrainVariant", terrainVariant.length, n);
        requireLength("profile.surfaceProfile", surfaceProfile.length, n);
        requireLength("profile.vegetationProfile", vegetationProfile.length, n);
        requireLength("profile.profileConfidence", profileConfidence.length, n);

        for (int i = 0; i < n; i++) {
            if (ecologyLandMask[i] != landMask[i] || profileLandMask[i] != landMask[i]) {
                throw new IllegalArgumentException("persisted land masks disagree at node " + i);
            }
            requireEnum("ecoRegion", unsigned(ecoRegion[i]), ECO_REGION_NAMES.length, i);
            requireEnum("biomeFamily", unsigned(biomeFamily[i]), BIOME_NAMES.length, i);
            requireEnum("terrainVariant", unsigned(terrainVariant[i]), TERRAIN_NAMES.length, i);
            requireEnum("surfaceProfile", unsigned(surfaceProfile[i]), SURFACE_NAMES.length, i);
            requireEnum("vegetationProfile", unsigned(vegetationProfile[i]), VEGETATION_NAMES.length, i);
            require01("regionConfidence", regionConfidence[i], i);
            require01("profileConfidence", profileConfidence[i], i);
        }

        return new GeoWorldV1(
            manifest,
            worldProfile,
            landMask,
            terrain,
            List.copyOf(rivers),
            List.copyOf(lakes),
            waterbedConfig,
            worldSampleConfig,
            climateTemp,
            climatePrecip,
            climateMonsoon,
            climateWindX,
            climateWindZ,
            basinId,
            flowTo,
            flowAccumulation,
            riverOrder,
            ecoRegion,
            regionConfidence,
            biomeFamily,
            terrainVariant,
            surfaceProfile,
            vegetationProfile,
            profileConfidence
        );
    }

    private WorldSample sampleWorld(double x, double z) {
        GroundSample ground = sampleGround(x, z);
        double ocean = oceanEligibility.sampleSmoothstepBilinear(x, z);

        if (
            ground.waterKind() == WaterKind.LAKE &&
            ground.waterSurfaceDefined() &&
            ground.finalHeight() < ground.waterSurfaceElevation()
        ) {
            return new WorldSample(
                ground.finalHeight(),
                WaterKind.LAKE,
                true,
                ground.waterSurfaceElevation(),
                ground.waterSurfaceElevation() - ground.finalHeight(),
                ocean,
                ground.riverWeight()
            );
        }

        double riverSurface = 0;
        double riverDepth = 0;
        boolean riverValid = false;
        if (
            ground.waterKind() == WaterKind.RIVER &&
            ground.riverWeight() > 0 &&
            ground.riverBedDefined()
        ) {
            double widthSpan = Math.max(
                1.0e-9,
                waterbedConfig.maxRiverHalfWidthBlocks() -
                    waterbedConfig.minRiverHalfWidthBlocks()
            );
            double widthT = clamp(
                (ground.riverHalfWidth() -
                    waterbedConfig.minRiverHalfWidthBlocks()) /
                    widthSpan,
                0,
                1
            );
            double centerDepth =
                worldSampleConfig.riverCenterDepthNormalized() *
                (1 + 2 * Math.pow(widthT, 1.15));
            riverSurface =
                ground.riverBedElevation() + centerDepth;
            if (
                ground.riverReceiverSurfaceDefined() &&
                ground.riverReceiverSurface() > riverSurface &&
                ground.riverReceiverWeight() > 0
            ) {
                riverSurface =
                    ground.riverReceiverSurface();
            }
            riverSurface = Math.min(1.0, riverSurface);
            riverDepth = riverSurface - ground.finalHeight();
            riverValid =
                riverDepth >=
                worldSampleConfig.minRiverWaterDepthNormalized();
        }

        if (
            ocean >= worldSampleConfig.oceanEligibilityThreshold() &&
            ground.finalHeight() < manifest.seaLevelNormalized()
        ) {
            double oneBlock = normalizedBlockStepAbove(
                manifest.seaLevelNormalized()
            );
            if (
                !riverValid ||
                riverSurface <= manifest.seaLevelNormalized() + oneBlock
            ) {
                return new WorldSample(
                    ground.finalHeight(),
                    WaterKind.OCEAN,
                    true,
                    manifest.seaLevelNormalized(),
                    manifest.seaLevelNormalized() - ground.finalHeight(),
                    ocean,
                    ground.riverWeight()
                );
            }
        }

        if (riverValid) {
            return new WorldSample(
                ground.finalHeight(),
                WaterKind.RIVER,
                true,
                riverSurface,
                riverDepth,
                ocean,
                ground.riverWeight()
            );
        }

        return new WorldSample(
            ground.finalHeight(),
            WaterKind.NONE,
            false,
            0,
            0,
            ocean,
            ground.riverWeight()
        );
    }

    private static double normalizedBlockStepAbove(
        double height
    ) {
        return height < 0.5
            ? 1.0 / 254.0
            : 1.0 / 514.0;
    }

    private GroundSample sampleGround(double x, double z) {
        double baseHeight = terrain.sampleNormalized(x, z);
        double finalHeight = baseHeight;
        double riverWeight = 0;
        double riverHalfWidth = 0;
        boolean riverBedDefined = false;
        double riverBedElevation = 0;
        double riverDatumWeight = -1;
        double riverDatumT = 2;
        boolean geometryWetOwned = false;
        double bestOxbowSurface = Double.NEGATIVE_INFINITY;
        boolean receiverDefined = false;
        double receiverSurface = 0;
        double receiverWeight = 0;
        WaterKind kind = WaterKind.NONE;

        List<RiverSegment> riverCandidates = this.riverCandidates(x, z);
        boolean activeRiverBankProtected = riverBankProtectedAt(
            riverCandidates,
            x,
            z
        );

        for (RiverSegment segment : riverCandidates) {
            PointSegment point = pointRiverSegment(x, z, segment);
            double mouthHalfWidth = riverMouthHalfWidth(segment, point.t());
            double bankRadius =
                mouthHalfWidth +
                riverBankBlendBlocks(segment.halfWidth());
            double fluvialReach =
                Math.max(bankRadius, riverFluvialReach(segment));
            DeltaPoint delta = riverDeltaPoint(
                segment,
                point.t(),
                x,
                z
            );
            boolean deltaInBank =
                delta != null && delta.distance() < delta.bankRadius();
            boolean deltaInWater =
                delta != null && delta.distance() < delta.halfWidth();
            if (point.distance() >= fluvialReach && !deltaInBank) {
                continue;
            }
            double bed = sampleRiverProfile(segment, point.t());

            double widthSpan = Math.max(
                1.0e-9,
                waterbedConfig.maxRiverHalfWidthBlocks() -
                    waterbedConfig.minRiverHalfWidthBlocks()
            );
            double widthT = clamp(
                (segment.halfWidth() -
                    waterbedConfig.minRiverHalfWidthBlocks()) /
                    widthSpan,
                0,
                1
            );
            double centerDepth =
                worldSampleConfig.riverCenterDepthNormalized() *
                (1 + 2 * Math.pow(widthT, 1.15));
            double naturalSurface = bed + centerDepth;
            double surface = riverConnectionSurface(
                segment,
                point.t(),
                naturalSurface
            );

            if (deltaInBank && !deltaInWater) {
                double innerBank =
                    Math.min(baseHeight, surface + 0.0018);
                double bankWeight =
                    1 -
                    smoothstep(
                        delta.halfWidth(),
                        delta.bankRadius(),
                        delta.distance()
                    );
                double candidate =
                    lerp(baseHeight, innerBank, bankWeight);
                if (candidate < finalHeight) {
                    finalHeight = candidate;
                }
            }

            if (point.distance() >= mouthHalfWidth && !deltaInWater) {
                if (point.distance() < bankRadius) {
                    double innerBank =
                        Math.min(baseHeight, surface + 0.0022);
                    double bankWeight =
                        1 -
                        smoothstep(
                            mouthHalfWidth,
                            bankRadius,
                            point.distance()
                        );
                    double candidate =
                        lerp(baseHeight, innerBank, bankWeight);
                    if (candidate < finalHeight) {
                        finalHeight = candidate;
                    }
                }

                // Fluvial dry detail belongs outside every active river-bank
                // protection corridor. A meander from one segment must not
                // excavate another live segment at a confluence or overlap.
                if (!activeRiverBankProtected) {
                    finalHeight = applyRiverFluvialDryDetail(
                        segment,
                        point.t(),
                        x,
                        z,
                        baseHeight,
                        finalHeight,
                        surface,
                        point.distance()
                    );

                    double[] oxbow = riverOxbowWater(
                        segment,
                        point.t(),
                        x,
                        z,
                        point.distance()
                    );
                    if (
                        oxbow != null &&
                        finalHeight <= oxbow[0] + 0.010
                    ) {
                        double candidate = lerp(
                            finalHeight,
                            Math.min(finalHeight, oxbow[1]),
                            oxbow[2]
                        );
                        if (candidate < finalHeight) {
                            finalHeight = candidate;
                        }
                        if (
                            finalHeight <
                                oxbow[0] -
                                    waterbedConfig.minLakeWaterDepthNormalized() &&
                            oxbow[0] > bestOxbowSurface
                        ) {
                            bestOxbowSurface = oxbow[0];
                        }
                    }
                }
                continue;
            }

            double channelHalfWidth = mouthHalfWidth;
            double channelDistance = point.distance();
            double weight = -1.0;
            if (point.distance() < mouthHalfWidth) {
                weight =
                    1 -
                    smoothstep(
                        0,
                        mouthHalfWidth,
                        point.distance()
                    );
            }
            if (deltaInWater) {
                double deltaWeight =
                    1 -
                    smoothstep(
                        0,
                        delta.halfWidth(),
                        delta.distance()
                    );
                if (deltaWeight > weight) {
                    weight = deltaWeight;
                    channelHalfWidth = delta.halfWidth();
                    channelDistance = delta.distance();
                }
            }
            if (weight <= 0.0) {
                continue;
            }
            if (weight > riverWeight) {
                riverWeight = weight;
            }
            if (
                !riverBedDefined ||
                weight > riverDatumWeight + 1.0e-12 ||
                (
                    Math.abs(weight - riverDatumWeight) <= 1.0e-12 &&
                    (
                        point.t() < riverDatumT - 1.0e-12 ||
                        (
                            Math.abs(point.t() - riverDatumT) <= 1.0e-12 &&
                            bed < riverBedElevation
                        )
                    )
                )
            ) {
                riverBedDefined = true;
                riverBedElevation = bed;
                // Flaring the river footprint must not inflate centerDepth.
                riverHalfWidth = segment.halfWidth();
                receiverDefined =
                    surface > naturalSurface + 1.0e-12;
                receiverSurface = surface;
                receiverWeight = receiverDefined ? 1.0 : 0.0;
                riverDatumWeight = weight;
                riverDatumT = point.t();
            }
            double channelShoulder =
                Math.min(baseHeight, surface + 0.0022);
            double outletWeight =
                lakeOutletSourceWeight(segment, point.t());
            if (outletWeight > 0) {
                double wetShoulder = Math.min(
                    baseHeight,
                    surface -
                        waterbedConfig.minLakeWaterDepthNormalized()
                );
                channelShoulder = lerp(
                    channelShoulder,
                    wetShoulder,
                    outletWeight
                );
            }
            double crossT =
                smoothstep(
                    0,
                    channelHalfWidth,
                    channelDistance
                );
            if (segment.mouthExpansionVersion() >= 3) {
                crossT = Math.pow(
                    crossT,
                    1.65 + 0.95 * widthT
                );
            } else if (segment.mouthExpansionVersion() >= 2) {
                crossT = Math.pow(
                    crossT,
                    1.35 + 0.65 * widthT
                );
            }
            double candidate =
                Math.min(
                    baseHeight,
                    lerp(bed, channelShoulder, crossT)
                );
            boolean v4WetCore =
                segment.mouthExpansionVersion() >= 4 &&
                !segment.oceanMouth() &&
                !segment.lakeMouth() &&
                !segment.lakeOutlet();
            boolean insideWetCore =
                !v4WetCore ||
                channelDistance <=
                    channelHalfWidth * RIVER_WET_CORE_FRACTION;
            if (v4WetCore && insideWetCore) {
                geometryWetOwned = true;
                double minimumWetDepth = 0.5 * centerDepth;
                candidate = Math.min(
                    candidate,
                    Math.max(0.0, surface - minimumWetDepth)
                );
            }
            if (candidate < finalHeight) {
                finalHeight = candidate;
            }
            if (kind == WaterKind.NONE) {
                // The whole hydraulic corridor remains a river candidate.
                // The v4 core only guarantees a minimum depth in its interior.
                kind = WaterKind.RIVER;
            }
        }
        if (
            geometryWetOwned &&
            kind == WaterKind.RIVER &&
            riverBedDefined
        ) {
            double widthSpan = Math.max(
                1.0e-9,
                waterbedConfig.maxRiverHalfWidthBlocks() -
                    waterbedConfig.minRiverHalfWidthBlocks()
            );
            double widthT = clamp(
                (riverHalfWidth -
                    waterbedConfig.minRiverHalfWidthBlocks()) /
                    widthSpan,
                0,
                1
            );
            double ownerDepth =
                worldSampleConfig.riverCenterDepthNormalized() *
                (1 + 2 * Math.pow(widthT, 1.15));
            double ownerSurface = riverBedElevation + ownerDepth;
            if (receiverDefined && receiverSurface > ownerSurface) {
                ownerSurface = receiverSurface;
            }
            finalHeight = Math.min(
                finalHeight,
                Math.max(0.0, ownerSurface - 0.5 * ownerDepth)
            );
        }
        boolean waterSurfaceDefined = false;
        double waterSurface = 0;
        if (
            kind == WaterKind.NONE &&
            bestOxbowSurface > Double.NEGATIVE_INFINITY &&
            finalHeight <
                bestOxbowSurface -
                    this.waterbedConfig.minLakeWaterDepthNormalized()
        ) {
            kind = WaterKind.LAKE;
            waterSurfaceDefined = true;
            waterSurface = bestOxbowSurface;
        }
        double bestLakeSurface = Double.NEGATIVE_INFINITY;
        for (LakeNode lake : this.lakeCandidates(x, z)) {
            // The macro lake mask identifies a depression, but the detailed
            // terrain owns the final shoreline. Do not dig high terrain down
            // merely because it lies inside a macro lake cell.
            final double lakeShoreAllowanceNormalized = 0.0015;
            if (
                baseHeight >
                lake.waterSurface() + lakeShoreAllowanceNormalized &&
                !lake.coreCarve()
            ) {
                continue;
            }
            double dx = x - lake.x();
            double dz = z - lake.z();
            double distance = lakeDistance(dx, dz);
            if (distance >= lake.outerRadius()) {
                continue;
            }
            double weight = 1.0;
            if (distance > lake.innerRadius()) {
                weight = 1 - smoothstep(lake.innerRadius(), lake.outerRadius(), distance);
            }
            double candidate =
                baseHeight - Math.max(0, baseHeight - lake.bedCeiling()) * weight;
            if (candidate < finalHeight) {
                finalHeight = candidate;
            }
            if (
                finalHeight < lake.waterSurface() -
                    this.waterbedConfig.minLakeWaterDepthNormalized() &&
                lake.waterSurface() > bestLakeSurface
            ) {
                bestLakeSurface = lake.waterSurface();
                kind = WaterKind.LAKE;
                waterSurfaceDefined = true;
                waterSurface = lake.waterSurface();
            }
        }

        return new GroundSample(
            finalHeight,
            riverWeight,
            riverHalfWidth,
            riverBedDefined,
            riverBedElevation,
            receiverDefined,
            receiverSurface,
            receiverWeight,
            kind,
            waterSurfaceDefined,
            waterSurface
        );
    }

    private record ProfileCodes(
        int biomeCode,
        int surfaceCode,
        int vegetationCode
    ) {}

    private ProfileCodes applyElevationZoning(
        double worldX,
        double worldZ,
        WorldSample world,
        boolean sourceLand,
        int terrainCode,
        int biomeCode,
        int surfaceCode,
        int vegetationCode
    ) {
        if (!sourceLand || world.waterKind() == WaterKind.OCEAN) {
            return new ProfileCodes(biomeCode, surfaceCode, vegetationCode);
        }

        double context = terrainElevationContext(
            enumName(TERRAIN_NAMES, terrainCode)
        );
        if (context < 0.45) {
            return new ProfileCodes(biomeCode, surfaceCode, vegetationCode);
        }

        double sea = manifest.seaLevelNormalized();
        if (sea <= 0 || sea >= 1) {
            sea = 0.5;
        }
        double relativeHeight = clamp(
            (world.groundHeight() - sea) / (1 - sea),
            0,
            1
        );

        double cellSize = 0.5 * (geometry.spacingX() + geometry.spacingZ());
        long seed = terrain.config.seed();
        double bandNoise =
            0.028 * valueNoise(seed + 33071L, worldX, worldZ, cellSize * 5.5) +
            0.012 * valueNoise(seed + 33091L, worldX, worldZ, cellSize * 2.2);
        double effectiveHeight = clamp(
            relativeHeight + bandNoise + 0.025 * (context - 0.5),
            0,
            1
        );

        String biome = enumName(BIOME_NAMES, biomeCode);
        double snowLine = biomeSnowLine(biome);
        if (effectiveHeight >= snowLine) {
            int alpine = enumCode(BIOME_NAMES, "Alpine");
            int alpineVegetation = enumCode(
                VEGETATION_NAMES,
                "Alpine Sparse"
            );
            // Mirror Go's slope-driven bare rock rule. A snow-line
            // elevation band must not turn into a horizontal stone stripe.
            double baseHeight = terrain.sampleNormalized(worldX, worldZ);
            double dx = (
                terrain.sampleNormalized(worldX + 16, worldZ) - baseHeight
            ) * 514 / 16;
            double dz = (
                terrain.sampleNormalized(worldX, worldZ + 16) - baseHeight
            ) * 514 / 16;
            double slope = Math.hypot(dx, dz);
            double exposure = smoothstep(0.32, 0.85, slope);
            double rockPatch = 0.5 + 0.5 * valueNoise(
                seed + 33131L,
                worldX,
                worldZ,
                cellSize * 1.1
            );
            int alpineSurface = enumCode(
                SURFACE_NAMES,
                exposure > 0.52 + 0.18 * (rockPatch - 0.5)
                    ? "Alpine Rock"
                    : "Alpine Snow"
            );
            return new ProfileCodes(
                alpine,
                alpineSurface,
                alpineVegetation
            );
        }

        if (context < 0.65 || effectiveHeight < snowLine - 0.12) {
            return new ProfileCodes(biomeCode, surfaceCode, vegetationCode);
        }

        return switch (biome) {
            case "Tropical Rainforest", "Tropical Monsoon Forest" ->
                new ProfileCodes(
                    enumCode(BIOME_NAMES, "Subtropical Forest"),
                    enumCode(SURFACE_NAMES, "Forest Loam"),
                    enumCode(VEGETATION_NAMES, "Broadleaf Forest")
                );
            case "Subtropical Forest" ->
                new ProfileCodes(
                    enumCode(BIOME_NAMES, "Temperate Forest"),
                    enumCode(SURFACE_NAMES, "Forest Loam"),
                    enumCode(VEGETATION_NAMES, "Mixed Forest")
                );
            case "Temperate Forest" ->
                new ProfileCodes(
                    enumCode(BIOME_NAMES, "Boreal Forest"),
                    enumCode(SURFACE_NAMES, "Podzol"),
                    enumCode(VEGETATION_NAMES, "Conifer Forest")
                );
            case "Boreal Forest" ->
                new ProfileCodes(
                    enumCode(BIOME_NAMES, "Tundra"),
                    enumCode(SURFACE_NAMES, "Tundra Soil"),
                    enumCode(VEGETATION_NAMES, "Tundra Sparse")
                );
            case "Savanna", "Mediterranean Shrubland" ->
                new ProfileCodes(
                    enumCode(BIOME_NAMES, "Temperate Grassland"),
                    enumCode(SURFACE_NAMES, "Dry Grass Soil"),
                    enumCode(VEGETATION_NAMES, "Dry Grassland")
                );
            case "Hot Desert" ->
                new ProfileCodes(
                    enumCode(BIOME_NAMES, "Temperate Desert"),
                    enumCode(SURFACE_NAMES, "Shrub Soil"),
                    enumCode(VEGETATION_NAMES, "Desert Sparse")
                );
            case "Wetland" ->
                new ProfileCodes(
                    enumCode(BIOME_NAMES, "Temperate Grassland"),
                    enumCode(SURFACE_NAMES, "Grass Loam"),
                    enumCode(VEGETATION_NAMES, "Grassland")
                );
            default -> new ProfileCodes(
                biomeCode,
                surfaceCode,
                vegetationCode
            );
        };
    }

    private static double terrainElevationContext(String terrainName) {
        return switch (terrainName) {
            case "Mountain" -> 1.0;
            case "Foothills" -> 0.82;
            case "Plateau" -> 0.68;
            case "Hills" -> 0.56;
            default -> 0.0;
        };
    }

    private static double biomeSnowLine(String biomeName) {
        return switch (biomeName) {
            case "Ice Sheet", "Tundra" -> 0.20;
            case "Alpine" -> 0.24;
            case "Boreal Forest" -> 0.30;
            case "Temperate Forest",
                 "Temperate Grassland",
                 "Temperate Desert",
                 "Wetland" -> 0.44;
            case "Mediterranean Shrubland" -> 0.49;
            case "Subtropical Forest", "Savanna" -> 0.55;
            case "Hot Desert" -> 0.60;
            case "Tropical Monsoon Forest", "Tropical Rainforest" -> 0.62;
            default -> 0.46;
        };
    }

    private static int enumCode(String[] names, String value) {
        for (int i = 0; i < names.length; i++) {
            if (names[i].equals(value)) {
                return i;
            }
        }
        throw new IllegalArgumentException("unknown enum label: " + value);
    }

    private record SourceSelection(int node, double weight, boolean fallback) {}

    private SourceSelection selectSourceNode(double x, double z, boolean wantLand) {
        double spacingX = geometry.spacingX();
        double spacingZ = geometry.spacingZ();

        // Keep categorical profile regions organic instead of exposing the
        // underlying macro-grid as rectangular biome patches. This must mirror
        // internal/runtimeprofile exactly so Go generation, Java Runtime,
        // preview and Minecraft chunks all choose the same profile node.
        double cellSize = 0.5 * (spacingX + spacingZ);
        double continuity = this.worldProfile.parameters().biomeContinuity();
        if (continuity <= 0) {
            continuity = 1.0;
        }
        double wavelength = cellSize * 5.0 * continuity;
        double amplitude = cellSize * 0.42 / Math.sqrt(continuity);
        long seed = terrain.config.seed();
        double warpX =
            0.72 * valueNoise(seed + 12011L, x, z, wavelength) +
            0.28 * valueNoise(seed + 12029L, x, z, wavelength * 0.5);
        double warpZ =
            0.72 * valueNoise(seed + 22003L, x, z, wavelength) +
            0.28 * valueNoise(seed + 22027L, x, z, wavelength * 0.5);
        double profileX = x + amplitude * warpX;
        double profileZ = z + amplitude * warpZ;

        double gx = clamp(
            (profileX - geometry.originX()) / spacingX,
            0,
            geometry.width() - 1.0
        );
        double gz = clamp(
            (profileZ - geometry.originZ()) / spacingZ,
            0,
            geometry.height() - 1.0
        );

        int x0 = (int) Math.floor(gx);
        int z0 = (int) Math.floor(gz);
        int x1 = Math.min(x0 + 1, geometry.width() - 1);
        int z1 = Math.min(z0 + 1, geometry.height() - 1);

        double tx = smoothstepUnit(gx - x0);
        double tz = smoothstepUnit(gz - z0);

        int[] nodes = {
            z0 * geometry.width() + x0,
            z0 * geometry.width() + x1,
            z1 * geometry.width() + x0,
            z1 * geometry.width() + x1
        };
        double[] weights = {
            (1 - tx) * (1 - tz),
            tx * (1 - tz),
            (1 - tx) * tz,
            tx * tz
        };

        SourceSelection selected = selectErodedCandidate(
            nodes,
            weights,
            wantLand,
            seed,
            x,
            z,
            cellSize
        );
        if (selected != null) {
            return selected;
        }

        // A warped categorical coordinate may cross a coastline even though
        // Ground/Water intentionally sampled the original coordinate. Retry
        // that unwarped cell before exposing a semantic fallback.
        double originalGX = clamp(
            (x - geometry.originX()) / spacingX,
            0,
            geometry.width() - 1.0
        );
        double originalGZ = clamp(
            (z - geometry.originZ()) / spacingZ,
            0,
            geometry.height() - 1.0
        );
        int originalX0 = (int) Math.floor(originalGX);
        int originalZ0 = (int) Math.floor(originalGZ);
        int originalX1 = Math.min(originalX0 + 1, geometry.width() - 1);
        int originalZ1 = Math.min(originalZ0 + 1, geometry.height() - 1);
        double originalTX = smoothstepUnit(originalGX - originalX0);
        double originalTZ = smoothstepUnit(originalGZ - originalZ0);
        int[] originalNodes = {
            originalZ0 * geometry.width() + originalX0,
            originalZ0 * geometry.width() + originalX1,
            originalZ1 * geometry.width() + originalX0,
            originalZ1 * geometry.width() + originalX1
        };
        double[] originalWeights = {
            (1 - originalTX) * (1 - originalTZ),
            originalTX * (1 - originalTZ),
            (1 - originalTX) * originalTZ,
            originalTX * originalTZ
        };
        selected = selectErodedCandidate(
            originalNodes,
            originalWeights,
            wantLand,
            seed,
            x,
            z,
            cellSize
        );
        if (selected != null) {
            return selected;
        }

        int best = nodes[0];
        double bestWeight = weights[0];
        for (int i = 1; i < nodes.length; i++) {
            if (weights[i] > bestWeight) {
                best = nodes[i];
                bestWeight = weights[i];
            }
        }
        return new SourceSelection(best, bestWeight, true);
    }

    private SourceSelection selectErodedCandidate(
        int[] nodes,
        double[] weights,
        boolean wantLand,
        long seed,
        double x,
        double z,
        double cellSize
    ) {
        int[] uniqueNodes = new int[nodes.length];
        double[] uniqueWeights = new double[nodes.length];
        int count = 0;

        for (int i = 0; i < nodes.length; i++) {
            int node = nodes[i];
            if (landMask[node] != wantLand) {
                continue;
            }

            int duplicate = -1;
            for (int j = 0; j < count; j++) {
                if (uniqueNodes[j] == node) {
                    duplicate = j;
                    break;
                }
            }
            if (duplicate >= 0) {
                if (weights[i] > uniqueWeights[duplicate]) {
                    uniqueWeights[duplicate] = weights[i];
                }
                continue;
            }

            uniqueNodes[count] = node;
            uniqueWeights[count] = weights[i];
            count++;
        }

        if (count == 0) {
            return null;
        }
        if (count == 1) {
            return new SourceSelection(
                uniqueNodes[0],
                uniqueWeights[0],
                false
            );
        }

        double bestBase = -1;
        double secondBase = -1;
        for (int i = 0; i < count; i++) {
            double weight = uniqueWeights[i];
            if (weight > bestBase) {
                secondBase = bestBase;
                bestBase = weight;
            } else if (weight > secondBase) {
                secondBase = weight;
            }
        }

        double gap = Math.max(0, bestBase - secondBase);
        double ambiguity =
            1 - smoothstepUnit(clamp(gap / 0.38, 0, 1));
        double erosionStrength = 0.16 * ambiguity;

        int best = uniqueNodes[0];
        double bestWeight = uniqueWeights[0];
        double bestScore = -Double.MAX_VALUE;
        for (int i = 0; i < count; i++) {
            int node = uniqueNodes[i];
            long nodeSeed = node;
            double competitionNoise =
                0.66 * valueNoise(
                    seed + 31013L + nodeSeed * 7919L,
                    x,
                    z,
                    cellSize * 0.34
                ) +
                0.34 * valueNoise(
                    seed + 41017L + nodeSeed * 1543L,
                    x,
                    z,
                    cellSize * 0.14
                );
            double score =
                uniqueWeights[i] + erosionStrength * competitionNoise;
            if (score > bestScore) {
                best = node;
                bestWeight = uniqueWeights[i];
                bestScore = score;
            }
        }

        return new SourceSelection(best, bestWeight, false);
    }

    private record PointSegment(double t, double distance) {}

    private static double lakeDistance(double dx, double dz) {
        double ax = Math.abs(dx);
        double az = Math.abs(dz);
        return Math.pow(
            ax * ax * ax * ax + az * az * az * az,
            0.25
        );
    }


    private static double connectionBasinSignal(RiverSegment segment) {
        if (segment.mouthExpansionVersion() < 3) {
            return 0.0;
        }
        double drop = Math.max(0.0, segment.bed0() - segment.bed1());
        return 1.0 - smoothstep(0.004, 0.016, drop);
    }

    private static double lakeOutletExtraHalfWidth(RiverSegment segment) {
        if (segment.mouthExpansionVersion() < 3) {
            return Math.min(
                24.0,
                Math.max(14.0, 1.25 * segment.halfWidth())
            );
        }
        double basin = connectionBasinSignal(segment);
        double factor = 1.25 + 0.85 * basin;
        return Math.min(
            48.0,
            Math.max(18.0, factor * segment.halfWidth())
        );
    }

    private static double riverDiagonalRasterWidthAmplitude(
        RiverSegment segment
    ) {
        if (
            segment.mouthExpansionVersion() < 4 ||
            segment.oceanMouth() ||
            segment.lakeMouth() ||
            segment.lakeOutlet() ||
            segment.halfWidth() < 24 ||
            !riverChordIsGridDiagonal(segment)
        ) {
            return 0.0;
        }
        return clamp(0.19 * segment.halfWidth(), 4.0, 9.0);
    }

    private static double riverDiagonalRasterWidthDelta(
        RiverSegment segment,
        double t
    ) {
        double amplitude = riverDiagonalRasterWidthAmplitude(segment);
        if (amplitude <= 0) {
            return 0.0;
        }
        t = clamp(t, 0, 1);
        double s = Math.sin(Math.PI * t);
        return -amplitude * s * s * Math.sin(4 * Math.PI * t);
    }

    private static double riverFluvialDetailStrength(
        RiverSegment segment
    ) {
        if (
            segment.mouthExpansionVersion() < 5 ||
            segment.oceanMouth() ||
            segment.lakeMouth() ||
            segment.lakeOutlet() ||
            segment.halfWidth() < 14
        ) {
            return 0.0;
        }
        double length = Math.hypot(
            segment.x1() - segment.x0(),
            segment.z1() - segment.z0()
        );
        if (length <= 1.0e-9) {
            return 0.0;
        }
        double start = sampleRiverProfile(segment, 0.0);
        double end = sampleRiverProfile(segment, 1.0);
        double drop =
            Math.max(0.0, start - end) * 128.0 / length;
        double widthWeight =
            smoothstep(14.0, 34.0, segment.halfWidth());
        double lowGradient =
            1.0 - smoothstep(0.008, 0.030, drop);
        return clamp(widthWeight * lowGradient, 0.0, 1.0);
    }

    private static double riverFluvialReach(
        RiverSegment segment
    ) {
        double bankRadius =
            riverMouthHalfWidth(segment, 0.5) +
            riverBankBlendBlocks(segment.halfWidth());
        double strength = riverFluvialDetailStrength(segment);
        if (strength <= 0.0) {
            return bankRadius;
        }
        return bankRadius +
            strength * (28.0 + 2.15 * segment.halfWidth());
    }

    private static long riverFluvialKey(RiverSegment segment) {
        long h = 1469598103934665603L;
        long[] values = new long[]{
            Math.round(segment.x0()),
            Math.round(segment.z0()),
            Math.round(segment.x1()),
            Math.round(segment.z1())
        };
        for (long value : values) {
            h ^= value;
            h *= 1099511628211L;
        }
        return h;
    }

    private static double riverFluvialPhase(
        RiverSegment segment
    ) {
        double inv53 = 1.0 / 9007199254740992.0;
        long h = riverFluvialKey(segment);
        return ((double)(h >>> 11)) *
            inv53 * 2.0 * Math.PI;
    }

    private static double riverAbandonedMeanderOffset(
        RiverSegment segment,
        double t,
        long key
    ) {
        t = clamp(t, 0.0, 1.0);
        double s = Math.sin(Math.PI * t);
        double side = (key & 1L) != 0L ? -1.0 : 1.0;
        double bankRadius =
            riverMouthHalfWidth(segment, t) +
            riverBankBlendBlocks(segment.halfWidth());
        double bulge =
            (0.55 * segment.halfWidth() + 6.0) * s * s;
        return side * (bankRadius + 4.0 + bulge);
    }

    private static boolean riverNaturalLeveeEnabled(
        RiverSegment segment
    ) {
        double strength = riverFluvialDetailStrength(segment);
        if (strength < 0.56) {
            return false;
        }
        return Long.remainderUnsigned(
            riverFluvialKey(segment),
            5L
        ) < 2L;
    }

    private static double riverNaturalLeveeLift(
        RiverSegment segment,
        double t,
        double worldX,
        double worldZ,
        double n
    ) {
        if (!riverNaturalLeveeEnabled(segment)) {
            return 0.0;
        }
        t = clamp(t, 0.0, 1.0);
        n = clamp(n, 0.0, 1.0);
        double strength = riverFluvialDetailStrength(segment);
        if (strength <= 0.0) {
            return 0.0;
        }

        double bankBand =
            smoothstep(0.015, 0.065, n) *
            (1.0 - smoothstep(0.16, 0.28, n));
        if (bankBand <= 0.0) {
            return 0.0;
        }

        double[] frame =
            riverCenterAndNormalAt(segment, t);
        double lateral =
            (worldX - frame[0]) * frame[2] +
            (worldZ - frame[1]) * frame[3];
        double side = lateral < 0.0 ? -1.0 : 1.0;
        double phase = riverFluvialPhase(segment);
        double sectionSignal =
            0.5 +
            0.5 *
                Math.sin(
                    2.0 * Math.PI * t +
                    phase +
                    0.82 * side
                );
        double continuity =
            smoothstep(0.34, 0.72, sectionSignal);
        double endFade =
            smoothstep(0.04, 0.16, t) *
            (1.0 - smoothstep(0.84, 0.96, t));
        return
            0.0042 *
            strength *
            bankBand *
            continuity *
            endFade;
    }

    private static double applyRiverFluvialDryDetail(
        RiverSegment segment,
        double t,
        double worldX,
        double worldZ,
        double baseHeight,
        double currentHeight,
        double surface,
        double distance
    ) {
        double strength = riverFluvialDetailStrength(segment);
        if (strength <= 0.0) {
            return currentHeight;
        }
        double halfWidth = riverMouthHalfWidth(segment, t);
        double bankRadius =
            halfWidth + riverBankBlendBlocks(segment.halfWidth());
        double reach = riverFluvialReach(segment);
        if (distance < bankRadius || distance >= reach) {
            return currentHeight;
        }

        double phase = riverFluvialPhase(segment);
        if (distance >= bankRadius) {
            double n = clamp(
                (distance - bankRadius) /
                    Math.max(1.0e-9, reach - bankRadius),
                0.0,
                1.0
            );
            double envelope =
                (1.0 - smoothstep(0.0, 1.0, n)) * strength;
            double terrace =
                smoothstep(0.38, 0.56, n) *
                (1.0 - smoothstep(0.74, 0.92, n)) *
                strength;
            double floodTarget =
                surface +
                0.0055 +
                0.0011 *
                    Math.sin(2.0 * Math.PI * t + phase);
            double terraceTarget =
                surface +
                0.015 +
                0.0018 *
                    Math.sin(Math.PI * t + phase * 0.5);
            double target =
                lerp(floodTarget, terraceTarget, terrace);
            target += riverNaturalLeveeLift(
                segment,
                t,
                worldX,
                worldZ,
                n
            );
            double dryWeight =
                envelope * (0.74 - 0.24 * terrace);
            double candidate = lerp(
                baseHeight,
                Math.min(baseHeight, target),
                clamp(dryWeight, 0.0, 1.0)
            );
            currentHeight = Math.min(
                currentHeight,
                candidate
            );
        }

        long key = riverFluvialKey(segment);
        if (
            distance >= bankRadius &&
            Long.remainderUnsigned(key, 13L) == 0L &&
            strength >= 0.52
        ) {
            double[] frame =
                riverCenterAndNormalAt(segment, t);
            double lateral =
                (worldX - frame[0]) * frame[2] +
                (worldZ - frame[1]) * frame[3];
            double s =
                Math.sin(Math.PI * clamp(t, 0.0, 1.0));
            double offset =
                riverAbandonedMeanderOffset(
                    segment,
                    t,
                    key
                );
            double oldDistance =
                Math.abs(lateral - offset);
            double radius =
                clamp(0.18 * segment.halfWidth(), 4.0, 11.0);
            double grooveWeight =
                (1.0 -
                    smoothstep(
                        0.35 * radius,
                        radius,
                        oldDistance
                    )) *
                s * s *
                strength;
            if (grooveWeight > 0.0) {
                double oldTarget =
                    surface +
                    0.0045 +
                    0.0008 *
                        Math.sin(
                            2.0 * Math.PI * t + phase
                        );
                double candidate = lerp(
                    baseHeight,
                    Math.min(baseHeight, oldTarget),
                    clamp(0.72 * grooveWeight, 0.0, 1.0)
                );
                currentHeight = Math.min(
                    currentHeight,
                    candidate
                );
            }
        }
        return currentHeight;
    }

    private double[] riverOxbowWater(
        RiverSegment segment,
        double t,
        double worldX,
        double worldZ,
        double activeDistance
    ) {
        double strength = riverFluvialDetailStrength(segment);
        double bankRadius =
            riverMouthHalfWidth(segment, t) +
            riverBankBlendBlocks(segment.halfWidth());
        if (
            segment.mouthExpansionVersion() < 5 ||
            segment.oceanMouth() ||
            segment.lakeMouth() ||
            segment.lakeOutlet() ||
            segment.halfWidth() < 18 ||
            strength < 0.62 ||
            activeDistance < bankRadius
        ) {
            return null;
        }

        long key = riverFluvialKey(segment);
        if (
            Long.remainderUnsigned(key, 13L) != 0L
        ) {
            return null;
        }
        long quotient = Long.divideUnsigned(key, 13L);
        if (
            Long.remainderUnsigned(quotient, 3L) != 0L
        ) {
            return null;
        }

        t = clamp(t, 0.0, 1.0);
        double s = Math.sin(Math.PI * t);
        if (s <= 0.18) {
            return null;
        }
        double[] frame = riverCenterAndNormalAt(segment, t);
        double lateral =
            (worldX - frame[0]) * frame[2] +
            (worldZ - frame[1]) * frame[3];
        double offset =
            riverAbandonedMeanderOffset(
                segment,
                t,
                key
            );
        double oldDistance =
            Math.abs(lateral - offset);
        double radius =
            clamp(0.20 * segment.halfWidth(), 5.0, 13.0);
        double core =
            1.0 -
            smoothstep(
                0.35 * radius,
                radius,
                oldDistance
            );
        double weight = clamp(
            core * s * s * (0.82 + 0.18 * strength),
            0.0,
            1.0
        );
        if (weight <= 0.0) {
            return null;
        }

        double midBed = sampleRiverProfile(segment, 0.5);
        double widthSpan = Math.max(
            1.0e-9,
            this.waterbedConfig.maxRiverHalfWidthBlocks() -
                this.waterbedConfig.minRiverHalfWidthBlocks()
        );
        double widthT = clamp(
            (
                segment.halfWidth() -
                this.waterbedConfig.minRiverHalfWidthBlocks()
            ) / widthSpan,
            0.0,
            1.0
        );
        double centerDepth =
            this.worldSampleConfig.riverCenterDepthNormalized() *
            (1.0 + 2.0 * Math.pow(widthT, 1.15));
        double midNaturalSurface = midBed + centerDepth;
        double surface = riverConnectionSurface(
            segment,
            0.5,
            midNaturalSurface
        );
        double depth =
            Math.max(
                this.waterbedConfig.minLakeWaterDepthNormalized() *
                    1.35,
                0.0042
            ) *
            (0.90 + 0.25 * strength);
        double floor = Math.max(0.0, surface - depth);
        return new double[]{surface, floor, weight};
    }

    private static long riverDeltaKey(RiverSegment segment) {
        return riverFluvialKey(segment) ^ 0xD371A5EED17A5EEDL;
    }

    private static double riverDeltaStrength(RiverSegment segment) {
        if (
            segment.mouthExpansionVersion() < 5 ||
            !segment.oceanMouth() ||
            segment.lakeMouth() ||
            segment.lakeOutlet() ||
            segment.halfWidth() < 16.0
        ) {
            return 0.0;
        }
        double basin = connectionBasinSignal(segment);
        if (basin < 0.52) {
            return 0.0;
        }
        long key = riverDeltaKey(segment);
        if (Long.remainderUnsigned(key, 3L) != 0L) {
            return 0.0;
        }
        double width = smoothstep(16.0, 34.0, segment.halfWidth());
        double lowGradient = smoothstep(0.52, 0.90, basin);
        return clamp(width * lowGradient, 0.0, 1.0);
    }

    private static int riverDeltaBranchCount(RiverSegment segment) {
        double strength = riverDeltaStrength(segment);
        if (strength <= 0.0) {
            return 0;
        }
        long key = riverDeltaKey(segment);
        long quotient = Long.divideUnsigned(key, 3L);
        if (
            strength >= 0.72 &&
            Long.remainderUnsigned(quotient, 2L) == 0L
        ) {
            return 2;
        }
        return 1;
    }

    private static double riverDeltaBranchSide(
        RiverSegment segment,
        int branch
    ) {
        int count = riverDeltaBranchCount(segment);
        if (count <= 0) {
            return 0.0;
        }
        if (count == 2) {
            return branch == 0 ? -1.0 : 1.0;
        }
        return (riverDeltaKey(segment) & 1L) != 0L ? -1.0 : 1.0;
    }

    private static double riverDeltaCentralFlareScale(
        RiverSegment segment
    ) {
        double strength = riverDeltaStrength(segment);
        if (strength <= 0.0) {
            return 1.0;
        }
        return 0.40 - 0.08 * strength;
    }

    private record DeltaBranchGeometry(
        double offset,
        double halfWidth,
        double bankRadius
    ) {}

    private static DeltaBranchGeometry riverDeltaBranchGeometry(
        RiverSegment segment,
        double t,
        int branch
    ) {
        double strength = riverDeltaStrength(segment);
        int count = riverDeltaBranchCount(segment);
        if (strength <= 0.0 || branch < 0 || branch >= count) {
            return null;
        }
        double start = 0.46;
        double u = clamp((t - start) / (1.0 - start), 0.0, 1.0);
        if (u <= 0.0) {
            return null;
        }
        double growth = smoothstep(0.0, 0.24, u);
        double targetWidth = clamp(
            segment.halfWidth() * (0.22 + 0.07 * strength),
            4.5,
            13.5
        );
        double halfWidth = targetWidth * growth;
        if (halfWidth < 1.25) {
            return null;
        }

        double normalExtra =
            mouthMaxHalfWidth(segment) - segment.halfWidth();
        double centralEnd =
            segment.halfWidth() +
            normalExtra * riverDeltaCentralFlareScale(segment);
        double landGap = 3.5 + 4.5 * strength;
        double terminalSeparation = Math.min(
            82.0,
            centralEnd + targetWidth + landGap
        );
        double separation =
            terminalSeparation * smoothstep(0.0, 1.0, u);
        double offset =
            riverDeltaBranchSide(segment, branch) * separation;
        double bankRadius =
            halfWidth + riverBankBlendBlocks(halfWidth);
        return new DeltaBranchGeometry(offset, halfWidth, bankRadius);
    }

    private static double riverDeltaSupportRadius(RiverSegment segment) {
        int count = riverDeltaBranchCount(segment);
        if (count <= 0) {
            return 0.0;
        }
        double maxSupport = 0.0;
        for (int branch = 0; branch < count; branch++) {
            DeltaBranchGeometry geometry =
                riverDeltaBranchGeometry(segment, 1.0, branch);
            if (geometry == null) {
                continue;
            }
            maxSupport = Math.max(
                maxSupport,
                Math.abs(geometry.offset()) + geometry.bankRadius()
            );
        }
        return maxSupport;
    }

    private record DeltaPoint(
        int branch,
        double distance,
        double halfWidth,
        double bankRadius
    ) {}

    private static DeltaPoint riverDeltaPoint(
        RiverSegment segment,
        double t,
        double worldX,
        double worldZ
    ) {
        int count = riverDeltaBranchCount(segment);
        if (count <= 0) {
            return null;
        }
        double[] frame = riverCenterAndNormalAt(segment, t);
        double lateral =
            (worldX - frame[0]) * frame[2] +
            (worldZ - frame[1]) * frame[3];
        double bestDistance = Double.POSITIVE_INFINITY;
        int bestBranch = -1;
        double bestWidth = 0.0;
        double bestBank = 0.0;
        for (int branch = 0; branch < count; branch++) {
            DeltaBranchGeometry geometry =
                riverDeltaBranchGeometry(segment, t, branch);
            if (geometry == null) {
                continue;
            }
            double distance = Math.abs(lateral - geometry.offset());
            if (distance < bestDistance) {
                bestDistance = distance;
                bestBranch = branch;
                bestWidth = geometry.halfWidth();
                bestBank = geometry.bankRadius();
            }
        }
        if (bestBranch < 0) {
            return null;
        }
        return new DeltaPoint(
            bestBranch,
            bestDistance,
            bestWidth,
            bestBank
        );
    }

    private static double riverMaxHalfWidth(RiverSegment segment) {
        double maxWidth =
            mouthMaxHalfWidth(segment) +
            riverDiagonalRasterWidthAmplitude(segment);
        if (segment.mouthExpansionVersion() >= 2 && segment.lakeOutlet()) {
            maxWidth = Math.max(
                maxWidth,
                segment.halfWidth() + lakeOutletExtraHalfWidth(segment)
            );
        }
        return maxWidth;
    }

    private static double mouthMaxHalfWidth(RiverSegment segment) {
        if (segment.mouthExpansionVersion() < 2) {
            if (!segment.oceanMouth()) {
                return segment.halfWidth();
            }
            return segment.halfWidth() +
                Math.min(12.0, Math.max(9.0, segment.halfWidth()));
        }
        if (segment.mouthExpansionVersion() == 2) {
            if (segment.oceanMouth()) {
                double extra = Math.min(
                    18.0,
                    Math.max(12.0, 0.85 * segment.halfWidth())
                );
                return segment.halfWidth() + extra;
            }
            if (segment.lakeMouth()) {
                double extra = Math.min(
                    12.0,
                    Math.max(6.0, 0.55 * segment.halfWidth())
                );
                return segment.halfWidth() + extra;
            }
            return segment.halfWidth();
        }

        double basin = connectionBasinSignal(segment);
        if (segment.oceanMouth()) {
            double extra = Math.min(
                32.0,
                Math.max(
                    16.0,
                    (0.90 + 0.35 * basin) * segment.halfWidth()
                )
            );
            return segment.halfWidth() + extra;
        }
        if (segment.lakeMouth()) {
            if (segment.mouthExpansionVersion() >= 6) {
                double extra = Math.max(
                    18.0,
                    (1.35 + 0.60 * basin) * segment.halfWidth()
                );
                return segment.halfWidth() + extra;
            }
            double extra = Math.min(
                42.0,
                Math.max(
                    16.0,
                    (0.80 + 1.05 * basin) * segment.halfWidth()
                )
            );
            return segment.halfWidth() + extra;
        }
        return segment.halfWidth();
    }

    private static double riverMouthHalfWidth(RiverSegment segment, double t) {
        if (
            segment.mouthExpansionVersion() >= 3 &&
            segment.lakeOutlet() &&
            segment.oceanMouth()
        ) {
            double sourceExtra = lakeOutletExtraHalfWidth(segment);
            double receiverExtra =
                mouthMaxHalfWidth(segment) - segment.halfWidth();
            double targetExtra = Math.max(sourceExtra, receiverExtra);
            double progress = smoothstep(0.04, 0.90, clamp(t, 0, 1));
            return segment.halfWidth() +
                lerp(sourceExtra, targetExtra, progress);
        }
        if (segment.mouthExpansionVersion() < 2) {
            if (!segment.oceanMouth()) {
                return segment.halfWidth();
            }
            double progress = clamp((t - 0.12) / 0.62, 0, 1);
            return segment.halfWidth() +
                (mouthMaxHalfWidth(segment) - segment.halfWidth()) *
                smoothstep(0, 1, progress);
        }

        double width =
            segment.halfWidth() +
            riverDiagonalRasterWidthDelta(segment, t);
        if (segment.oceanMouth() || segment.lakeMouth()) {
            double start = 0.18;
            double span = 0.70;
            if (segment.lakeMouth()) {
                start = 0.34;
                span = 0.56;
            }
            if (segment.mouthExpansionVersion() >= 3) {
                if (segment.oceanMouth()) {
                    start = 0.04;
                    span = 0.86;
                } else {
                    start = 0.18;
                    span = 0.76;
                }
            }
            double progress = clamp((t - start) / span, 0, 1);
            double receiverExtra =
                (mouthMaxHalfWidth(segment) - segment.halfWidth()) *
                smoothstep(0, 1, progress);
            if (segment.oceanMouth()) {
                receiverExtra *= riverDeltaCentralFlareScale(segment);
            }
            double receiverWidth = segment.halfWidth() + receiverExtra;
            width = Math.max(width, receiverWidth);
        }

        if (segment.lakeOutlet()) {
            double extra = lakeOutletExtraHalfWidth(segment);
            double length = Math.hypot(
                segment.x1() - segment.x0(),
                segment.z1() - segment.z0()
            );
            double along = clamp(t, 0, 1) * length;
            double startFade = 96.0;
            double endFade = 176.0;
            if (segment.mouthExpansionVersion() >= 3) {
                double basin = connectionBasinSignal(segment);
                startFade = 88.0 - 24.0 * basin;
                endFade = 190.0 + 90.0 * basin;
            }
            double fade = 1 - smoothstep(startFade, endFade, along);
            width = Math.max(
                width,
                segment.halfWidth() + extra * fade
            );
        }
        return width;
    }

    private static double lakeOutletSourceWeight(
        RiverSegment segment,
        double t
    ) {
        if (
            segment.mouthExpansionVersion() < 2 ||
            !segment.lakeOutletSurfaceDefined()
        ) {
            return 0.0;
        }
        double length = Math.hypot(
            segment.x1() - segment.x0(),
            segment.z1() - segment.z0()
        );
        double along = clamp(t, 0, 1) * length;
        if (segment.mouthExpansionVersion() >= 3) {
            double basin = connectionBasinSignal(segment);
            return 1 - smoothstep(
                72.0 - 16.0 * basin,
                196.0 + 96.0 * basin,
                along
            );
        }
        return 1 - smoothstep(96.0, 176.0, along);
    }
    private static double riverConnectionSurface(
        RiverSegment segment,
        double t,
        double naturalSurface
    ) {
        double surface = naturalSurface;
        if (
            segment.receiverSurfaceDefined() &&
            segment.receiverSurface() > surface
        ) {
            surface = segment.receiverSurface();
        }
        double outletWeight = lakeOutletSourceWeight(segment, t);
        if (
            outletWeight > 0 &&
            segment.lakeOutletSurface() > naturalSurface
        ) {
            double outletSurface = lerp(
                naturalSurface,
                segment.lakeOutletSurface(),
                outletWeight
            );
            surface = Math.max(surface, outletSurface);
        }
        return Math.min(1.0, surface);
    }

    private static double riverBankBlendBlocks(double halfWidth) {
        return clamp(10 + 0.65 * halfWidth, 12, 24);
    }

    private static boolean riverChordIsGridDiagonal(
        RiverSegment segment
    ) {
        double dx = Math.abs(segment.x1() - segment.x0());
        double dz = Math.abs(segment.z1() - segment.z0());
        double scale = Math.max(1.0, Math.max(dx, dz));
        return dx > 1.0e-9 &&
            dz > 1.0e-9 &&
            Math.abs(dx - dz) <= 1.0e-9 * scale;
    }

    private static double[] riverMeanderShape(
        RiverSegment segment,
        double t
    ) {
        t = clamp(t, 0, 1);
        double phase = Math.PI * t;
        double s = Math.sin(phase);
        double c = Math.cos(phase);

        if (segment.mouthExpansionVersion() >= 3) {
            return new double[]{
                s * s,
                2 * Math.PI * s * c,
                2 * Math.PI * Math.PI * (c * c - s * s)
            };
        }
        return new double[]{
            s,
            Math.PI * c,
            -Math.PI * Math.PI * s
        };
    }

    private static double[] riverCenterAndNormalAt(
        RiverSegment segment,
        double t
    ) {
        t = clamp(t, 0, 1);
        double dx = segment.x1() - segment.x0();
        double dz = segment.z1() - segment.z0();
        double length = Math.hypot(dx, dz);
        if (length <= 1.0e-12) {
            return new double[]{segment.x0(), segment.z0(), 1, 0};
        }
        double chordNX = -dz / length;
        double chordNZ = dx / length;
        double[] meanderShape = riverMeanderShape(segment, t);
        double shape = meanderShape[0];
        double shapeDerivative = meanderShape[1];
        double x =
            segment.x0() +
            dx * t +
            chordNX * segment.meanderOffset() * shape;
        double z =
            segment.z0() +
            dz * t +
            chordNZ * segment.meanderOffset() * shape;
        double tx =
            dx + chordNX * segment.meanderOffset() * shapeDerivative;
        double tz =
            dz + chordNZ * segment.meanderOffset() * shapeDerivative;
        double tangentLength = Math.hypot(tx, tz);
        if (tangentLength <= 1.0e-12) {
            return new double[]{x, z, chordNX, chordNZ};
        }
        return new double[]{
            x,
            z,
            -tz / tangentLength,
            tx / tangentLength
        };
    }

    private static PointSegment pointRiverSegment(
        double px,
        double pz,
        RiverSegment segment
    ) {
        double dx = segment.x1() - segment.x0();
        double dz = segment.z1() - segment.z0();
        double len2 = dx * dx + dz * dz;
        if (len2 <= 1.0e-18) {
            return new PointSegment(
                0,
                Math.hypot(px - segment.x0(), pz - segment.z0())
            );
        }
        double length = Math.sqrt(len2);
        double nx = -dz / length;
        double nz = dx / length;
        double t = clamp(
            (
                (px - segment.x0()) * dx +
                (pz - segment.z0()) * dz
            ) / len2,
            0,
            1
        );

        for (
            int iter = 0;
            iter < 4 && Math.abs(segment.meanderOffset()) > 1.0e-9;
            iter++
        ) {
            double[] meanderShape = riverMeanderShape(segment, t);
            double shape = meanderShape[0];
            double shapeDerivative = meanderShape[1];
            double shapeSecondDerivative = meanderShape[2];
            double x =
                segment.x0() +
                dx * t +
                nx * segment.meanderOffset() * shape;
            double z =
                segment.z0() +
                dz * t +
                nz * segment.meanderOffset() * shape;
            double tx =
                dx + nx * segment.meanderOffset() * shapeDerivative;
            double tz =
                dz + nz * segment.meanderOffset() * shapeDerivative;
            double ax =
                nx * segment.meanderOffset() * shapeSecondDerivative;
            double az =
                nz * segment.meanderOffset() * shapeSecondDerivative;
            double rx = x - px;
            double rz = z - pz;
            double f = rx * tx + rz * tz;
            double fp =
                tx * tx +
                tz * tz +
                rx * ax +
                rz * az;
            if (Math.abs(fp) <= 1.0e-12) {
                break;
            }
            double next = clamp(t - f / fp, 0, 1);
            if (Math.abs(next - t) <= 1.0e-8) {
                t = next;
                break;
            }
            t = next;
        }

        double[] center = riverCenterAndNormalAt(segment, t);
        return new PointSegment(
            t,
            Math.hypot(px - center[0], pz - center[1])
        );
    }

    private static PointSegment pointSegment(
        double px, double pz,
        double x0, double z0,
        double x1, double z1
    ) {
        double dx = x1 - x0;
        double dz = z1 - z0;
        double len2 = dx * dx + dz * dz;
        if (len2 <= 1e-18) {
            return new PointSegment(0, Math.hypot(px - x0, pz - z0));
        }
        double t = ((px - x0) * dx + (pz - z0) * dz) / len2;
        t = clamp(t, 0, 1);
        double qx = x0 + t * dx;
        double qz = z0 + t * dz;
        return new PointSegment(t, Math.hypot(px - qx, pz - qz));
    }

    private static double sampleRiverProfile(RiverSegment segment, double t) {
        double[] profile = segment.profile();
        if (profile.length == 0) {
            return lerp(segment.bed0(), segment.bed1(), t);
        }
        double pos = clamp(t, 0, 1) * (profile.length - 1.0);
        int i0 = (int) Math.floor(pos);
        int i1 = Math.min(i0 + 1, profile.length - 1);
        double f = pos - i0;
        return lerp(profile[i0], profile[i1], f);
    }

    private static Manifest parseManifest(Map<String, Object> object) {
        return new Manifest(
            Json.string(object, "magic"),
            Json.intValue(object, "schemaVersion"),
            Json.string(object, "generatorVersion"),
            optionalString(object, "worldProfileVersion"),
            optionalString(object, "generationRulesVersion"),
            optionalString(object, "worldModelVersion"),
            optionalString(object, "preset"),
            optionalString(object, "parameterSnapshotSha256"),
            Json.string(object, "fileEncoding"),
            Json.string(object, "checksumAlgorithm"),
            Json.string(object, "payloadSha256"),
            Json.longValue(object, "payloadBytes"),
            Json.longValue(object, "seed"),
            parseGeometry(Json.object(Json.required(object, "geometry"), "manifest.geometry")),
            Json.doubleValue(object, "seaLevelNormalized")
        );
    }

    private static String optionalString(Map<String, Object> object, String key) {
        return object.containsKey(key) ? Json.string(object, key) : "";
    }

    private static Geometry parseGeometry(Map<String, Object> object) {
        Geometry geometry = new Geometry(
            Json.intValue(object, "Width"),
            Json.intValue(object, "Height"),
            Json.doubleValue(object, "OriginX"),
            Json.doubleValue(object, "OriginZ"),
            Json.doubleValue(object, "SpanXBlocks"),
            Json.doubleValue(object, "SpanZBlocks")
        );
        if (geometry.width() < 2 || geometry.height() < 2) {
            throw new IllegalArgumentException("geometry width/height must be >= 2");
        }
        geometry.cellSize();
        geometry.nodeCount();
        return geometry;
    }

    private static void validateManifest(Manifest manifest) {
        if (!FORMAT_MAGIC.equals(manifest.magic())) {
            throw new IllegalArgumentException("unsupported GeoWorld magic: " + manifest.magic());
        }
        if (manifest.schemaVersion() != SCHEMA_VERSION) {
            throw new IllegalArgumentException(
                "unsupported schemaVersion: " + manifest.schemaVersion()
            );
        }
        if (!GENERATOR_VERSION.equals(manifest.generatorVersion())) {
            throw new IllegalArgumentException(
                "unsupported generatorVersion: " + manifest.generatorVersion()
            );
        }
        boolean identityPresent = !manifest.worldProfileVersion().isEmpty() ||
            !manifest.generationRulesVersion().isEmpty() ||
            !manifest.worldModelVersion().isEmpty() || !manifest.preset().isEmpty() ||
            !manifest.parameterSnapshotSha256().isEmpty();
        if (identityPresent && (
            manifest.worldProfileVersion().isEmpty() ||
            manifest.generationRulesVersion().isEmpty() ||
            manifest.worldModelVersion().isEmpty() || manifest.preset().isEmpty() ||
            manifest.parameterSnapshotSha256().length() != 64
        )) {
            throw new IllegalArgumentException("incomplete world profile identity");
        }
        if (!FILE_ENCODING.equals(manifest.fileEncoding())) {
            throw new IllegalArgumentException(
                "unsupported fileEncoding: " + manifest.fileEncoding()
            );
        }
        if (!CHECKSUM_ALGORITHM.equals(manifest.checksumAlgorithm())) {
            throw new IllegalArgumentException(
                "unsupported checksumAlgorithm: " + manifest.checksumAlgorithm()
            );
        }
        if (manifest.payloadBytes() <= 0 || manifest.payloadSha256().length() != 64) {
            throw new IllegalArgumentException("invalid payload metadata");
        }
        if (
            !finite(manifest.seaLevelNormalized()) ||
            manifest.seaLevelNormalized() <= 0 ||
            manifest.seaLevelNormalized() >= 1
        ) {
            throw new IllegalArgumentException("invalid sea level");
        }
    }

    private static WorldProfile parseWorldProfile(
        Map<String, Object> config,
        Manifest manifest
    ) {
        Object raw = config.get("WorldProfile");
        if (raw == null) {
            return new WorldProfile(
                "legacy-rc1",
                "legacy-rc1",
                "legacy-model-v1",
                "legacy",
                ParameterSnapshot.standard(),
                true
            );
        }
        Map<String, Object> profile = Json.object(raw, "config.WorldProfile");
        Map<String, Object> parameters = Json.object(
            Json.required(profile, "Parameters"),
            "config.WorldProfile.Parameters"
        );
        WorldProfile parsed = new WorldProfile(
            Json.string(profile, "WorldProfileVersion"),
            Json.string(profile, "GenerationRulesVersion"),
            Json.string(profile, "WorldModelVersion"),
            Json.string(profile, "Preset"),
            new ParameterSnapshot(
                parameter(parameters, "MainlandScale"),
                parameter(parameters, "RegionScale"),
                parameter(parameters, "TransitionWidth"),
                parameter(parameters, "MountainStrength"),
                parameter(parameters, "PlainContinuity"),
                parameter(parameters, "PlateauScale"),
                parameter(parameters, "CliffRarity"),
                parameter(parameters, "RareMacroTerrain"),
                parameter(parameters, "RiverDensity"),
                parameter(parameters, "MainRiverScale"),
                parameter(parameters, "LakeAbundance"),
                parameter(parameters, "WetlandAbundance"),
                parameter(parameters, "ClimateRegionScale"),
                parameter(parameters, "ClimateTransitionWidth"),
                parameter(parameters, "RainShadowStrength"),
                parameter(parameters, "BiomeContinuity"),
                parameter(parameters, "CaveRichness"),
                parameter(parameters, "LargeCaveRarity"),
                parameter(parameters, "MineralRegionalization"),
                parameter(parameters, "SpawnPreference"),
                parameter(parameters, "StructureInfluence"),
                parameter(parameters, "OuterIslandAbundance"),
                parameter(parameters, "PolarTransitionWidth"),
                parameter(parameters, "OceanClimateStrength"),
                parameter(parameters, "PrevailingWindStrength")
            ),
            false
        );
        if (!manifest.worldProfileVersion().isEmpty()) {
            if (!parsed.worldProfileVersion().equals(manifest.worldProfileVersion()) ||
                !parsed.generationRulesVersion().equals(manifest.generationRulesVersion()) ||
                !parsed.worldModelVersion().equals(manifest.worldModelVersion()) ||
                !parsed.preset().equals(manifest.preset())) {
                throw new IllegalArgumentException("manifest/config world profile mismatch");
            }
        }
        return parsed;
    }

    private static double parameter(Map<String, Object> object, String key) {
        double value = object.containsKey(key) ? Json.doubleValue(object, key) : 1.0;
        double min = key.equals("SpawnPreference") || key.equals("StructureInfluence")
            ? 0.0
            : 0.25;
        if (!finite(value) || value < min || value > 2.5) {
            throw new IllegalArgumentException("invalid world profile parameter: " + key);
        }
        return value;
    }

    private static void validateResolvedConfig(
        Map<String, Object> config,
        Manifest manifest
    ) {
        if (Json.longValue(config, "Seed") != manifest.seed()) {
            throw new IllegalArgumentException("manifest/config seed mismatch");
        }
        if (Json.doubleValue(config, "SeaLevelNormalized") != manifest.seaLevelNormalized()) {
            throw new IllegalArgumentException("manifest/config sea level mismatch");
        }
        Geometry configGeometry = parseGeometry(
            Json.object(Json.required(config, "Geometry"), "config.Geometry")
        );
        if (!configGeometry.equals(manifest.geometry())) {
            throw new IllegalArgumentException("manifest/config geometry mismatch");
        }
    }

    private static void validateSurface(
        Geometry geometry,
        boolean[] land,
        boolean[] enclosed
    ) {
        int n = geometry.nodeCount();
        requireLength("surface.landMask", land.length, n);
        requireLength("surface.enclosedBelowSea", enclosed.length, n);
        for (int i = 0; i < n; i++) {
            if (enclosed[i] && !land[i]) {
                throw new IllegalArgumentException(
                    "enclosed-below-sea node cannot be ocean: " + i
                );
            }
        }
    }

    private static Terrain parseTerrain(
        Geometry geometry,
        Map<String, Object> object
    ) {
        Map<String, Object> configObject = Json.object(
            Json.required(object, "config"),
            "terrain.config"
        );
        TerrainConfig config = new TerrainConfig(
            Json.longValue(configObject, "Seed"),
            Json.doubleValue(configObject, "PlainAmplitude"),
            Json.doubleValue(configObject, "HillAmplitude"),
            Json.doubleValue(configObject, "FoothillAmplitude"),
            Json.doubleValue(configObject, "MountainAmplitude"),
            configObject.containsKey("MountainStrengthScale")
                ? Json.doubleValue(configObject, "MountainStrengthScale")
                : 0.0,
            Json.doubleValue(configObject, "PlateauAmplitude"),
            Json.doubleValue(configObject, "ValleyDepth"),
            Json.doubleValue(configObject, "DepositionLift"),
            Json.doubleValue(configObject, "AlluvialFanLift"),
            Json.doubleValue(configObject, "CanyonExtraDepth"),
            configObject.containsKey("CliffFrequency")
                ? Json.doubleValue(configObject, "CliffFrequency")
                : 1.0
        );

        double[] correction = null;
        Object rawCorrection = object.get("heightCorrection");
        if (rawCorrection != null) {
            correction = doubleArray(rawCorrection, "terrain.heightCorrection");
        }

        return new Terrain(
            geometry,
            config,
            doubleArray(Json.required(object, "baseElevation"), "terrain.baseElevation"),
            doubleArray(
                Json.required(object, "rollingPlainWeight"),
                "terrain.rollingPlainWeight"
            ),
            doubleArray(Json.required(object, "hillWeight"), "terrain.hillWeight"),
            doubleArray(Json.required(object, "foothillWeight"), "terrain.foothillWeight"),
            doubleArray(Json.required(object, "mountainWeight"), "terrain.mountainWeight"),
            doubleArray(Json.required(object, "plateauWeight"), "terrain.plateauWeight"),
            doubleArray(Json.required(object, "valleyWeight"), "terrain.valleyWeight"),
            doubleArray(
                Json.required(object, "floodplainWeight"),
                "terrain.floodplainWeight"
            ),
            doubleArray(
                Json.required(object, "alluvialPlainWeight"),
                "terrain.alluvialPlainWeight"
            ),
            doubleArray(
                Json.required(object, "alluvialFanWeight"),
                "terrain.alluvialFanWeight"
            ),
            doubleArray(Json.required(object, "canyonWeight"), "terrain.canyonWeight"),
            doubleArray(
                Json.required(object, "valleyFloorWidth"),
                "terrain.valleyFloorWidth"
            ),
            doubleArray(
                Json.required(object, "depositionPotential"),
                "terrain.depositionPotential"
            ),
            correction
        );
    }

    private static List<RiverSegment> parseRivers(List<Object> values) {
        List<RiverSegment> out = new ArrayList<>(values.size());
        for (int i = 0; i < values.size(); i++) {
            Map<String, Object> r = Json.object(values.get(i), "river[" + i + "]");
            out.add(new RiverSegment(
                Json.intValue(r, "From"),
                Json.intValue(r, "To"),
                Json.doubleValue(r, "X0"),
                Json.doubleValue(r, "Z0"),
                Json.doubleValue(r, "X1"),
                Json.doubleValue(r, "Z1"),
                Json.doubleValue(r, "Bed0"),
                Json.doubleValue(r, "Bed1"),
                Json.doubleValue(r, "HalfWidth"),
                r.get("MeanderOffset") == null
                    ? 0.0
                    : Json.doubleValue(r, "MeanderOffset"),
                doubleArray(Json.required(r, "Profile"), "river.Profile"),
                Json.bool(r, "ReceiverSurfaceDefined"),
                Json.doubleValue(r, "ReceiverSurface"),
                r.containsKey("OceanMouth") && Json.bool(r, "OceanMouth"),
                r.containsKey("LakeMouth") && Json.bool(r, "LakeMouth"),
                r.containsKey("LakeOutlet") && Json.bool(r, "LakeOutlet"),
                r.containsKey("LakeOutletSurfaceDefined") &&
                    Json.bool(r, "LakeOutletSurfaceDefined"),
                r.containsKey("LakeOutletSurface")
                    ? Json.doubleValue(r, "LakeOutletSurface")
                    : 0.0,
                r.containsKey("MouthExpansionVersion")
                    ? Json.intValue(r, "MouthExpansionVersion")
                    : 0
            ));
        }
        return out;
    }

    private static List<LakeNode> parseLakes(List<Object> values) {
        List<LakeNode> out = new ArrayList<>(values.size());
        for (int i = 0; i < values.size(); i++) {
            Map<String, Object> l = Json.object(values.get(i), "lake[" + i + "]");
            out.add(new LakeNode(
                Json.intValue(l, "Index"),
                Json.intValue(l, "LakeID"),
                Json.doubleValue(l, "X"),
                Json.doubleValue(l, "Z"),
                Json.doubleValue(l, "BedCeiling"),
                Json.doubleValue(l, "InnerRadius"),
                Json.doubleValue(l, "OuterRadius"),
                Json.doubleValue(l, "WaterSurface"),
                l.containsKey("CoreCarve") && Json.bool(l, "CoreCarve")
            ));
        }
        return out;
    }

    private static void validateWaterbed(
        Geometry geometry,
        List<RiverSegment> rivers,
        List<LakeNode> lakes
    ) {
        int n = geometry.nodeCount();
        for (int i = 0; i < rivers.size(); i++) {
            RiverSegment river = rivers.get(i);
            if (
                river.from() < 0 || river.from() >= n ||
                river.to() < 0 || river.to() >= n ||
                river.from() == river.to() ||
                river.halfWidth() <= 0 ||
                !finite(river.meanderOffset()) ||
                river.profile().length == 0
            ) {
                throw new IllegalArgumentException("invalid persisted river: " + i);
            }
            double previous = Double.POSITIVE_INFINITY;
            for (double value : river.profile()) {
                if (!finite(value) || value < 0 || value > 1 || value > previous + 1e-12) {
                    throw new IllegalArgumentException("invalid river profile: " + i);
                }
                previous = value;
            }
            if (
                river.receiverSurfaceDefined() &&
                (
                    !finite(river.receiverSurface()) ||
                    river.receiverSurface() < 0 ||
                    river.receiverSurface() > 1
                )
            ) {
                throw new IllegalArgumentException("invalid river receiver surface: " + i);
            }
        }

        for (int i = 0; i < lakes.size(); i++) {
            LakeNode lake = lakes.get(i);
            if (
                lake.index() < 0 || lake.index() >= n ||
                lake.lakeId() <= 0 ||
                lake.innerRadius() < 0 ||
                lake.outerRadius() <= lake.innerRadius() ||
                lake.bedCeiling() < 0 || lake.bedCeiling() > 1 ||
                lake.waterSurface() <= 0 || lake.waterSurface() > 1 ||
                lake.bedCeiling() >= lake.waterSurface()
            ) {
                throw new IllegalArgumentException("invalid persisted lake: " + i);
            }
        }
    }

    private static final class Terrain {
        private final TerrainConfig config;
        private final ScalarGrid baseElevation;
        private final ScalarGrid rollingPlainWeight;
        private final ScalarGrid hillWeight;
        private final ScalarGrid foothillWeight;
        private final ScalarGrid mountainWeight;
        private final ScalarGrid plateauWeight;
        private final ScalarGrid valleyWeight;
        private final ScalarGrid floodplainWeight;
        private final ScalarGrid alluvialPlainWeight;
        private final ScalarGrid alluvialFanWeight;
        private final ScalarGrid canyonWeight;
        private final ScalarGrid valleyFloorWidth;
        private final ScalarGrid depositionPotential;
        private final ScalarGrid heightCorrection;

        Terrain(
            Geometry geometry,
            TerrainConfig config,
            double[] baseElevation,
            double[] rollingPlainWeight,
            double[] hillWeight,
            double[] foothillWeight,
            double[] mountainWeight,
            double[] plateauWeight,
            double[] valleyWeight,
            double[] floodplainWeight,
            double[] alluvialPlainWeight,
            double[] alluvialFanWeight,
            double[] canyonWeight,
            double[] valleyFloorWidth,
            double[] depositionPotential,
            double[] heightCorrection
        ) {
            this.config = config;
            this.baseElevation = new ScalarGrid(geometry, baseElevation);
            this.rollingPlainWeight = new ScalarGrid(geometry, rollingPlainWeight);
            this.hillWeight = new ScalarGrid(geometry, hillWeight);
            this.foothillWeight = new ScalarGrid(geometry, foothillWeight);
            this.mountainWeight = new ScalarGrid(geometry, mountainWeight);
            this.plateauWeight = new ScalarGrid(geometry, plateauWeight);
            this.valleyWeight = new ScalarGrid(geometry, valleyWeight);
            this.floodplainWeight = new ScalarGrid(geometry, floodplainWeight);
            this.alluvialPlainWeight = new ScalarGrid(geometry, alluvialPlainWeight);
            this.alluvialFanWeight = new ScalarGrid(geometry, alluvialFanWeight);
            this.canyonWeight = new ScalarGrid(geometry, canyonWeight);
            this.valleyFloorWidth = new ScalarGrid(geometry, valleyFloorWidth);
            this.depositionPotential = new ScalarGrid(geometry, depositionPotential);
            this.heightCorrection =
                heightCorrection == null ? null : new ScalarGrid(geometry, heightCorrection);
        }

        double sampleNormalized(double x, double z) {
            double base = baseElevation.sampleCubicBSpline(x, z);
            double plain = clamp01(rollingPlainWeight.sampleCubicBSpline(x, z));
            double hill = clamp01(hillWeight.sampleCubicBSpline(x, z));
            double foothill = clamp01(foothillWeight.sampleCubicBSpline(x, z));
            double mountain = clamp01(mountainWeight.sampleCubicBSpline(x, z));
            double plateau = clamp01(plateauWeight.sampleCubicBSpline(x, z));
            double valley = clamp01(valleyWeight.sampleCubicBSpline(x, z));
            double floodplain = clamp01(floodplainWeight.sampleCubicBSpline(x, z));
            double alluvialPlain = clamp01(alluvialPlainWeight.sampleCubicBSpline(x, z));
            double alluvialFan = clamp01(alluvialFanWeight.sampleCubicBSpline(x, z));
            double canyon = clamp01(canyonWeight.sampleCubicBSpline(x, z));
            double valleyFloor = clamp01(valleyFloorWidth.sampleCubicBSpline(x, z));
            double deposition = clamp01(depositionPotential.sampleCubicBSpline(x, z));

            double plainDetail =
                fbm(config.seed() + 101, x, z, 2300, 2) * config.plainAmplitude();
            double hillDetail =
                fbm(config.seed() + 211, x, z, 1250, 3) * config.hillAmplitude();
            double foothillDetail =
                fbm(config.seed() + 307, x, z, 1500, 3) * config.foothillAmplitude();
            double mountainDetail =
                fbm(config.seed() + 401, x, z, 1050, 4) *
                config.mountainAmplitude() *
                0.62;

            double ridgeBroad =
                ridgedFbm(config.seed() + 433, x, z, 760, 4);
            double ridgeFine =
                ridgedFbm(config.seed() + 461, x, z, 360, 3);
            double crest = clamp(
                0.70 * ridgeBroad + 0.30 * ridgeFine,
                0,
                1
            );
            double mountainCrest =
                (Math.pow(crest, 2.20) - 0.34) * 0.115;

            double ridgeIntersection = smoothstep(
                0.30,
                0.78,
                ridgeBroad * ridgeFine
            );
            double summitMod =
                0.72 +
                0.28 *
                (
                    0.5 +
                    0.5 *
                    valueNoise(
                        config.seed() + 487,
                        x,
                        z,
                        560
                    )
                );
            double summit =
                Math.pow(ridgeIntersection, 2.35) *
                0.075 *
                summitMod;
            double peakBlob = cellularRidgePeak(
                config.seed() + 547,
                x,
                z,
                720
            );
            double localizedSummit =
                Math.pow(peakBlob, 4.2) *
                smoothstep(0.44, 0.82, crest) *
                0.125;

            double peakCore = cellularPeak(
                config.seed() + 571,
                x,
                z,
                420
            );
            double summitSpire =
                Math.pow(peakCore, 5.8) *
                smoothstep(0.56, 0.86, crest) *
                0.110;

            double faultSignal =
                0.74 * valueNoise(config.seed() + 631, x, z, 980) +
                0.26 * valueNoise(config.seed() + 653, x, z, 310);
            double faultRegion =
                0.5 +
                0.5 * valueNoise(
                    config.seed() + 677,
                    x,
                    z,
                    1700
                );
            double cliffShift = clamp(
                (config.cliffFrequency() - 1.0) * 0.10,
                -0.14,
                0.14
            );
            double cliffGate =
                smoothstep(
                    0.56 - cliffShift,
                    0.80 - cliffShift * 0.8,
                    faultRegion
                ) *
                smoothstep(0.46, 0.78, crest);
            double faultStep =
                smoothstep(-0.010, 0.010, faultSignal);
            double cliffEscarpment =
                (faultStep - 0.5) *
                0.090 *
                cliffGate;

            double structuralSaddle =
                Math.pow(clamp(1 - crest, 0, 1), 2.4) *
                0.028;
            // Cut head gullies between subsidiary ridges, without broad uplift.
            double ridgeIncision =
                smoothstep(0.37, 0.68, 1 - ridgeFine) *
                smoothstep(0.50, 0.74, ridgeBroad) *
                0.050;
            double slopeWave =
                fbm(config.seed() + 509, x, z, 270, 3) *
                0.014;

            double plateauDetail =
                fbm(config.seed() + 503, x, z, 2600, 2) * config.plateauAmplitude();

            double regionalDetail =
                plain * plainDetail +
                hill * hillDetail +
                foothill * foothillDetail +
                mountain * (
                    mountainDetail +
                    mountainCrest +
                    summit +
                    localizedSummit +
                    summitSpire +
                    cliffEscarpment +
                    slopeWave -
                    structuralSaddle -
                    ridgeIncision
                ) +
                plateau * plateauDetail;

            double flatness = clamp(
                0.48 * floodplain +
                0.36 * alluvialPlain +
                0.34 * valleyFloor,
                0,
                0.88
            );
            regionalDetail *= 1 - 0.62 * flatness;
            regionalDetail *= 1 - 0.28 * alluvialFan;

            double valleyMod =
                0.82 +
                0.18 * (0.5 + 0.5 * valueNoise(config.seed() + 607, x, z, 1800));
            double valleyReliefGain =
                1 + 1.6 * foothill + 4.2 * mountain;
            double valleyCarve =
                valley *
                config.valleyDepth() *
                valleyMod *
                valleyReliefGain;
            double canyonReliefGain =
                1 + 0.9 * foothill + 2.4 * mountain;
            double canyonCarve =
                canyon *
                config.canyonExtraDepth() *
                canyonReliefGain;

            // Mirror the Go terrain morphology lift exactly. Ecological
            // families are laid over this terrain; they do not define its shape.
            // A zero MountainStrengthScale is the legacy marker for locked
            // pre-v1.1.7 models; current models scale only the broad mountain
            // lift while MountainAmplitude continues to control local relief.
            double mountainLiftScale = 1.0;
            if (config.mountainStrengthScale() > 0) {
                double strength = clamp(
                    config.mountainStrengthScale(),
                    0.25,
                    2.5
                );
                mountainLiftScale = 1 + 0.30 * (strength - 1);
            }
            double orographicLift =
                0.012 * hill +
                0.030 * foothill +
                0.090 * mountain * mountainLiftScale +
                0.020 * plateau;

            double depositionLift =
                deposition * config.depositionLift() +
                alluvialFan * config.alluvialFanLift();

            double workingBase = base;
            if (base > 0.82 && mountain > 0) {
                double upper = (base - 0.82) / 0.18;
                double compression =
                    mountain *
                    smoothstep(0, 1, upper) *
                    (base - 0.82) *
                    0.42;
                workingBase -= compression;
            }

            double result = boundedAdd(
                workingBase,
                regionalDetail + orographicLift + depositionLift - valleyCarve - canyonCarve
            );
            if (heightCorrection != null) {
                result = clamp(
                    result + heightCorrection.sampleSmoothstepBilinear(x, z),
                    0,
                    1
                );
            }
            return result;
        }
    }

    private static final class ScalarGrid {
        private final int width;
        private final int height;
        private final double cellSize;
        private final double originX;
        private final double originZ;
        private final double[] values;

        ScalarGrid(Geometry geometry, double[] values) {
            requireLength("scalar grid", values.length, geometry.nodeCount());
            this.width = geometry.width();
            this.height = geometry.height();
            this.cellSize = geometry.cellSize();
            this.originX = geometry.originX();
            this.originZ = geometry.originZ();
            this.values = values;
        }

        double sampleSmoothstepBilinear(double x, double z) {
            double gx = clamp((x - originX) / cellSize, 0, width - 1.0);
            double gz = clamp((z - originZ) / cellSize, 0, height - 1.0);

            int x0 = (int) Math.floor(gx);
            int z0 = (int) Math.floor(gz);
            int x1 = Math.min(x0 + 1, width - 1);
            int z1 = Math.min(z0 + 1, height - 1);

            double tx = smoothstepUnit(gx - x0);
            double tz = smoothstepUnit(gz - z0);

            double top = lerp(at(x0, z0), at(x1, z0), tx);
            double bottom = lerp(at(x0, z1), at(x1, z1), tx);
            return lerp(top, bottom, tz);
        }

        double sampleCubicBSpline(double x, double z) {
            double gx = clamp((x - originX) / cellSize, 0, width - 1.0);
            double gz = clamp((z - originZ) / cellSize, 0, height - 1.0);

            int xBase = (int) Math.floor(gx);
            int zBase = (int) Math.floor(gz);
            double tx = gx - xBase;
            double tz = gz - zBase;

            double[] wx = cubicBSplineWeights(tx);
            double[] wz = cubicBSplineWeights(tz);

            double sum = 0;
            for (int dz = 0; dz < 4; dz++) {
                int zi = clampInt(zBase + dz - 1, 0, height - 1);
                for (int dx = 0; dx < 4; dx++) {
                    int xi = clampInt(xBase + dx - 1, 0, width - 1);
                    sum += at(xi, zi) * wx[dx] * wz[dz];
                }
            }
            return sum;
        }

        private double at(int x, int z) {
            return values[z * width + x];
        }
    }

    private static double[] cubicBSplineWeights(double t) {
        t = clamp(t, 0, 1);
        double t2 = t * t;
        double t3 = t2 * t;
        double oneMinus = 1 - t;
        return new double[] {
            oneMinus * oneMinus * oneMinus / 6.0,
            (3 * t3 - 6 * t2 + 4) / 6.0,
            (-3 * t3 + 3 * t2 + 3 * t + 1) / 6.0,
            t3 / 6.0
        };
    }

    private static double boundedAdd(double base, double delta) {
        base = clamp(base, 0, 1);
        if (delta == 0) {
            return base;
        }
        if (delta > 0) {
            double headroom = 1 - base;
            if (headroom <= 1e-12) {
                return base;
            }
            return base + headroom * (1 - Math.exp(-delta / headroom));
        }
        double footroom = base;
        if (footroom <= 1e-12) {
            return base;
        }
        return base - footroom * (1 - Math.exp(delta / footroom));
    }

    private static double fbm(
        long seed,
        double x,
        double z,
        double wavelength,
        int octaves
    ) {
        if (octaves <= 0) {
            return 0;
        }
        double amplitude = 1;
        double sumAmplitude = 0;
        double sum = 0;
        double currentWavelength = wavelength;
        for (int octave = 0; octave < octaves; octave++) {
            sum +=
                valueNoise(seed + (long) octave * 7919L, x, z, currentWavelength) *
                amplitude;
            sumAmplitude += amplitude;
            amplitude *= 0.5;
            currentWavelength *= 0.5;
        }
        return sumAmplitude <= 0 ? 0 : sum / sumAmplitude;
    }

    // The structural-axis summit field is continuous across noise cells and
    // gives each mountain ridge a long crest and a steep transverse flank.
    private static double cellularRidgePeak(
        long seed,
        double x,
        double z,
        double cellSize
    ) {
        if (cellSize <= 0) {
            return 0;
        }
        long gx = (long)Math.floor(x / cellSize);
        long gz = (long)Math.floor(z / cellSize);
        double best = 0;
        for (long dz = -1; dz <= 1; dz++) {
            for (long dx = -1; dx <= 1; dx++) {
                long cx = gx + dx;
                long cz = gz + dz;
                double ux = 0.5 + 0.5 * hashValue(seed + 17, cx, cz);
                double uz = 0.5 + 0.5 * hashValue(seed + 53, cx, cz);
                double px = (cx + 0.16 + 0.68 * ux) * cellSize;
                double pz = (cz + 0.16 + 0.68 * uz) * cellSize;
                double heading = Math.PI * hashValue(seed + 97, cx, cz);
                double c = Math.cos(heading);
                double s = Math.sin(heading);
                double ox = x - px;
                double oz = z - pz;
                double along = ox * c + oz * s;
                double across = -ox * s + oz * c;
                double d = Math.hypot(
                    along / (cellSize * 0.68),
                    across / (cellSize * 0.29)
                );
                double ridge = 1 - smoothstep(0.08, 1, d);
                if (ridge > best) {
                    best = ridge;
                }
            }
        }
        return clamp(best, 0, 1);
    }

    private static double cellularPeak(
        long seed,
        double x,
        double z,
        double cellSize
    ) {
        if (cellSize <= 0) {
            return 0;
        }
        long gx = (long)Math.floor(x / cellSize);
        long gz = (long)Math.floor(z / cellSize);
        double best = 0;
        for (long dz = -1; dz <= 1; dz++) {
            for (long dx = -1; dx <= 1; dx++) {
                long cx = gx + dx;
                long cz = gz + dz;
                double ux =
                    0.5 + 0.5 * hashValue(seed + 17, cx, cz);
                double uz =
                    0.5 + 0.5 * hashValue(seed + 53, cx, cz);
                double px =
                    (cx + 0.16 + 0.68 * ux) * cellSize;
                double pz =
                    (cz + 0.16 + 0.68 * uz) * cellSize;
                double d =
                    Math.hypot(x - px, z - pz) /
                    (cellSize * 0.46);
                double blob =
                    1 - smoothstep(0.08, 1, d);
                if (blob > best) {
                    best = blob;
                }
            }
        }
        return clamp(best, 0, 1);
    }

    private static double ridgedFbm(
        long seed,
        double x,
        double z,
        double wavelength,
        int octaves
    ) {
        if (octaves <= 0) {
            return 0;
        }
        double amplitude = 1;
        double sumAmplitude = 0;
        double sum = 0;
        double currentWavelength = wavelength;
        for (int octave = 0; octave < octaves; octave++) {
            double n = valueNoise(
                seed + (long) octave * 104729L,
                x,
                z,
                currentWavelength
            );
            double ridge = 1 - Math.abs(n);
            sum += ridge * amplitude;
            sumAmplitude += amplitude;
            amplitude *= 0.55;
            currentWavelength *= 0.5;
        }
        return sumAmplitude <= 0
            ? 0
            : clamp(sum / sumAmplitude, 0, 1);
    }

    private static double valueNoise(
        long seed,
        double x,
        double z,
        double wavelength
    ) {
        if (wavelength <= 0) {
            return 0;
        }
        double gx = x / wavelength;
        double gz = z / wavelength;
        long x0 = (long) Math.floor(gx);
        long z0 = (long) Math.floor(gz);
        double tx = gx - x0;
        double tz = gz - z0;

        double fx = quinticFade(tx);
        double fz = quinticFade(tz);

        double v00 = hashValue(seed, x0, z0);
        double v10 = hashValue(seed, x0 + 1, z0);
        double v01 = hashValue(seed, x0, z0 + 1);
        double v11 = hashValue(seed, x0 + 1, z0 + 1);

        return lerp(lerp(v00, v10, fx), lerp(v01, v11, fx), fz);
    }

    private static double hashValue(long seed, long x, long z) {
        long h =
            seed ^
            (x * 0x9E3779B185EBCA87L) ^
            (z * 0xC2B2AE3D27D4EB4FL);
        h ^= h >>> 30;
        h *= 0xBF58476D1CE4E5B9L;
        h ^= h >>> 27;
        h *= 0x94D049BB133111EBL;
        h ^= h >>> 31;

        double unit = (double) (h >>> 11) / (double) (1L << 53);
        return unit * 2 - 1;
    }

    private static double quinticFade(double t) {
        t = clamp(t, 0, 1);
        return t * t * t * (t * (t * 6 - 15) + 10);
    }

    private static double smoothstep(double edge0, double edge1, double x) {
        if (edge1 <= edge0) {
            return x >= edge1 ? 1 : 0;
        }
        double t = clamp((x - edge0) / (edge1 - edge0), 0, 1);
        return t * t * (3 - 2 * t);
    }

    private static double smoothstepUnit(double t) {
        t = clamp(t, 0, 1);
        return t * t * (3 - 2 * t);
    }

    private static double clamp01(double value) {
        return clamp(value, 0, 1);
    }

    private static double clamp(double value, double lo, double hi) {
        if (value < lo) {
            return lo;
        }
        if (value > hi) {
            return hi;
        }
        return value;
    }

    private static int clampInt(int value, int lo, int hi) {
        return Math.max(lo, Math.min(hi, value));
    }

    private static double lerp(double a, double b, double t) {
        return a + (b - a) * t;
    }

    private static boolean isOceanFamily(int code) {
        return code >= 1 && code <= 4;
    }

    private static int unsigned(byte value) {
        return Byte.toUnsignedInt(value);
    }

    private static String enumName(String[] names, int code) {
        if (code < 0 || code >= names.length) {
            return "Unknown";
        }
        return names[code];
    }

    private static void requireEnum(
        String name,
        int code,
        int count,
        int node
    ) {
        if (code <= 0 || code >= count) {
            throw new IllegalArgumentException(
                name + " contains invalid code " + code + " at node " + node
            );
        }
    }

    private static void require01(String name, double value, int node) {
        if (!finite(value) || value < 0 || value > 1) {
            throw new IllegalArgumentException(
                name + " contains invalid value at node " + node
            );
        }
    }

    private static void requireLength(String name, int got, int want) {
        if (got != want) {
            throw new IllegalArgumentException(
                name + " length=" + got + " want=" + want
            );
        }
    }

    private static boolean[] booleanArray(Object value, String name) {
        List<Object> source = Json.array(value, name);
        boolean[] out = new boolean[source.size()];
        for (int i = 0; i < source.size(); i++) {
            Object item = source.get(i);
            if (!(item instanceof Boolean b)) {
                throw new IllegalArgumentException(name + "[" + i + "] must be boolean");
            }
            out[i] = b;
        }
        return out;
    }

    private static int[] intArray(Object value, String name) {
        List<Object> source = Json.array(value, name);
        int[] out = new int[source.size()];
        for (int i = 0; i < source.size(); i++) {
            Object item = source.get(i);
            if (!(item instanceof Number n)) {
                throw new IllegalArgumentException(name + "[" + i + "] must be number");
            }
            long v = n.longValue();
            if (v < Integer.MIN_VALUE || v > Integer.MAX_VALUE) {
                throw new IllegalArgumentException(name + "[" + i + "] is outside int range");
            }
            out[i] = (int)v;
        }
        return out;
    }

    private static double[] doubleArray(Object value, String name) {
        List<Object> source = Json.array(value, name);
        double[] out = new double[source.size()];
        for (int i = 0; i < source.size(); i++) {
            Object item = source.get(i);
            if (!(item instanceof Number n)) {
                throw new IllegalArgumentException(name + "[" + i + "] must be number");
            }
            double v = n.doubleValue();
            if (!finite(v)) {
                throw new IllegalArgumentException(name + "[" + i + "] must be finite");
            }
            out[i] = v;
        }
        return out;
    }

    private static byte[] byteSlice(Object value, String name) {
        if (!(value instanceof String encoded)) {
            throw new IllegalArgumentException(
                name + " must be the Go encoding/json Base64 representation of []uint8"
            );
        }
        try {
            return Base64.getDecoder().decode(encoded);
        } catch (IllegalArgumentException e) {
            throw new IllegalArgumentException(name + " is invalid Base64", e);
        }
    }

    private static String sha256Hex(byte[] bytes) {
        try {
            byte[] digest = MessageDigest.getInstance("SHA-256").digest(bytes);
            StringBuilder out = new StringBuilder(digest.length * 2);
            for (byte b : digest) {
                out.append(Character.forDigit((b >>> 4) & 0xf, 16));
                out.append(Character.forDigit(b & 0xf, 16));
            }
            return out.toString();
        } catch (NoSuchAlgorithmException e) {
            throw new IllegalStateException("SHA-256 is unavailable", e);
        }
    }

    private static boolean finite(double value) {
        return !Double.isNaN(value) && !Double.isInfinite(value);
    }
}
