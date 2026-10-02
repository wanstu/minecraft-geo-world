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
import java.util.List;
import java.util.Map;
import java.util.zip.GZIPInputStream;

public final class GeoWorldV1 {
    public static final int SCHEMA_VERSION = 1;
    public static final String GENERATOR_VERSION = "phase6.3-v1";
    public static final String FORMAT_MAGIC = "geoworld-model";
    public static final String FILE_ENCODING = "gzip+json";
    public static final String CHECKSUM_ALGORITHM = "sha256";

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
        boolean riverReceiverSurfaceDefined,
        double riverReceiverSurface,
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
        double[] profile,
        boolean receiverSurfaceDefined,
        double receiverSurface
    ) {}

    private record LakeNode(
        int index,
        int lakeId,
        double x,
        double z,
        double bedCeiling,
        double innerRadius,
        double outerRadius,
        double waterSurface
    ) {}

    private record TerrainConfig(
        long seed,
        double plainAmplitude,
        double hillAmplitude,
        double foothillAmplitude,
        double mountainAmplitude,
        double plateauAmplitude,
        double valleyDepth,
        double depositionLift,
        double alluvialFanLift,
        double canyonExtraDepth
    ) {}

    private record WorldSampleConfig(
        double riverCenterDepthNormalized,
        double oceanEligibilityThreshold
    ) {}

    private final Manifest manifest;
    private final Geometry geometry;
    private final boolean[] landMask;
    private final Terrain terrain;
    private final List<RiverSegment> rivers;
    private final List<LakeNode> lakes;
    private final ScalarGrid oceanEligibility;
    private final WorldSampleConfig worldSampleConfig;
    private final byte[] ecoRegion;
    private final double[] regionConfidence;
    private final byte[] biomeFamily;
    private final byte[] terrainVariant;
    private final byte[] surfaceProfile;
    private final byte[] vegetationProfile;
    private final double[] profileConfidence;

    private GeoWorldV1(
        Manifest manifest,
        boolean[] landMask,
        Terrain terrain,
        List<RiverSegment> rivers,
        List<LakeNode> lakes,
        WorldSampleConfig worldSampleConfig,
        byte[] ecoRegion,
        double[] regionConfidence,
        byte[] biomeFamily,
        byte[] terrainVariant,
        byte[] surfaceProfile,
        byte[] vegetationProfile,
        double[] profileConfidence
    ) {
        this.manifest = manifest;
        this.geometry = manifest.geometry();
        this.landMask = landMask;
        this.terrain = terrain;
        this.rivers = rivers;
        this.lakes = lakes;
        this.worldSampleConfig = worldSampleConfig;
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

    public Manifest manifest() {
        return manifest;
    }

    public RuntimeSample sample(double worldX, double worldZ) {
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
        WorldSampleConfig worldSampleConfig = new WorldSampleConfig(
            Json.doubleValue(worldSampleObject, "RiverCenterDepthNormalized"),
            Json.doubleValue(worldSampleObject, "OceanEligibilityThreshold")
        );

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
            landMask,
            terrain,
            List.copyOf(rivers),
            List.copyOf(lakes),
            worldSampleConfig,
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

        if (
            ocean >= worldSampleConfig.oceanEligibilityThreshold() &&
            ground.finalHeight() < manifest.seaLevelNormalized()
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

        if (ground.waterKind() == WaterKind.RIVER && ground.riverWeight() > 0) {
            double depth =
                worldSampleConfig.riverCenterDepthNormalized() * ground.riverWeight();
            if (
                ground.riverReceiverSurfaceDefined() &&
                ground.riverReceiverSurface() > ground.finalHeight()
            ) {
                double backwaterDepth =
                    (ground.riverReceiverSurface() - ground.finalHeight()) * ground.riverWeight();
                if (backwaterDepth > depth) {
                    depth = backwaterDepth;
                }
            }
            if (depth > 1e-12) {
                double surface = Math.min(1.0, ground.finalHeight() + depth);
                if (surface > ground.finalHeight()) {
                    return new WorldSample(
                        ground.finalHeight(),
                        WaterKind.RIVER,
                        true,
                        surface,
                        surface - ground.finalHeight(),
                        ocean,
                        ground.riverWeight()
                    );
                }
            }
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

    private GroundSample sampleGround(double x, double z) {
        double baseHeight = terrain.sampleNormalized(x, z);
        double finalHeight = baseHeight;
        double riverWeight = 0;
        boolean receiverDefined = false;
        double receiverSurface = 0;
        WaterKind kind = WaterKind.NONE;

        for (RiverSegment segment : rivers) {
            PointSegment point = pointSegment(
                x, z,
                segment.x0(), segment.z0(),
                segment.x1(), segment.z1()
            );
            if (point.distance() >= segment.halfWidth()) {
                continue;
            }
            double bed = sampleRiverProfile(segment, point.t());
            double weight = 1 - smoothstep(0, segment.halfWidth(), point.distance());
            if (weight > riverWeight) {
                riverWeight = weight;
            }
            if (
                segment.receiverSurfaceDefined() &&
                (!receiverDefined || segment.receiverSurface() > receiverSurface)
            ) {
                receiverDefined = true;
                receiverSurface = segment.receiverSurface();
            }
            double candidate =
                baseHeight - Math.max(0, baseHeight - bed) * weight;
            if (candidate < finalHeight) {
                finalHeight = candidate;
            }
            if (kind == WaterKind.NONE) {
                kind = WaterKind.RIVER;
            }
        }

        boolean waterSurfaceDefined = false;
        double waterSurface = 0;
        double bestLakeSurface = Double.NEGATIVE_INFINITY;
        for (LakeNode lake : lakes) {
            double dx = Math.abs(x - lake.x());
            double dz = Math.abs(z - lake.z());
            double distance = Math.max(dx, dz);
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
                finalHeight < lake.waterSurface() - 1e-12 &&
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
            receiverDefined,
            receiverSurface,
            kind,
            waterSurfaceDefined,
            waterSurface
        );
    }

    private record SourceSelection(int node, double weight, boolean fallback) {}

    private SourceSelection selectSourceNode(double x, double z, boolean wantLand) {
        double gx = clamp(
            (x - geometry.originX()) / geometry.spacingX(),
            0,
            geometry.width() - 1.0
        );
        double gz = clamp(
            (z - geometry.originZ()) / geometry.spacingZ(),
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

        int best = -1;
        double bestWeight = -1;
        for (int i = 0; i < nodes.length; i++) {
            if (landMask[nodes[i]] != wantLand) {
                continue;
            }
            if (weights[i] > bestWeight) {
                best = nodes[i];
                bestWeight = weights[i];
            }
        }
        if (best >= 0) {
            return new SourceSelection(best, bestWeight, false);
        }

        for (int i = 0; i < nodes.length; i++) {
            if (weights[i] > bestWeight) {
                best = nodes[i];
                bestWeight = weights[i];
            }
        }
        return new SourceSelection(best, bestWeight, true);
    }

    private record PointSegment(double t, double distance) {}

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
            Json.string(object, "fileEncoding"),
            Json.string(object, "checksumAlgorithm"),
            Json.string(object, "payloadSha256"),
            Json.longValue(object, "payloadBytes"),
            Json.longValue(object, "seed"),
            parseGeometry(Json.object(Json.required(object, "geometry"), "manifest.geometry")),
            Json.doubleValue(object, "seaLevelNormalized")
        );
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
            Json.doubleValue(configObject, "PlateauAmplitude"),
            Json.doubleValue(configObject, "ValleyDepth"),
            Json.doubleValue(configObject, "DepositionLift"),
            Json.doubleValue(configObject, "AlluvialFanLift"),
            Json.doubleValue(configObject, "CanyonExtraDepth")
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
                doubleArray(Json.required(r, "Profile"), "river.Profile"),
                Json.bool(r, "ReceiverSurfaceDefined"),
                Json.doubleValue(r, "ReceiverSurface")
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
                Json.doubleValue(l, "WaterSurface")
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
                fbm(config.seed() + 401, x, z, 1750, 4) * config.mountainAmplitude();
            double plateauDetail =
                fbm(config.seed() + 503, x, z, 2600, 2) * config.plateauAmplitude();

            double regionalDetail =
                plain * plainDetail +
                hill * hillDetail +
                foothill * foothillDetail +
                mountain * mountainDetail +
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
            double valleyCarve = valley * config.valleyDepth() * valleyMod;
            double canyonCarve = canyon * config.canyonExtraDepth();

            double depositionLift =
                deposition * config.depositionLift() +
                alluvialFan * config.alluvialFanLift();

            double result = boundedAdd(
                base,
                regionalDetail + depositionLift - valleyCarve - canyonCarve
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
