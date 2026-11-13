package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.v26common.MinecraftV26HeightMapping;
import geoworld.runtime.GeoWorldV1;

/** Checks the reported basin against actual integer block columns. */
public final class GeoWorldV26BasinRiverVerifier {
    private GeoWorldV26BasinRiverVerifier() {}

    public static void main(String[] args) throws Exception {
        var config = new GeoWorldCreationConfig(
            320, 200, 128, 18, 0.68, 0.35000000000000003, 0.65,
            GeoWorldCreationConfig.ContinentPreset.DEFAULT
        );
        var model = GeoWorldEngineBridge.generate(123456L, config).model();
        if (!("geoworld-rules-v1.1.22".equals(model.manifest().generationRulesVersion()) || ("geoworld-rules-v1.1.23".equals(model.manifest().generationRulesVersion()) || ("geoworld-rules-v1.1.24".equals(model.manifest().generationRulesVersion()) || ("geoworld-rules-v1.1.25".equals(model.manifest().generationRulesVersion()) || ("geoworld-rules-v1.1.26".equals(model.manifest().generationRulesVersion()) || ("geoworld-rules-v1.1.27".equals(model.manifest().generationRulesVersion()) || ("geoworld-rules-v1.1.28".equals(model.manifest().generationRulesVersion()) || ("geoworld-rules-v1.1.29".equals(model.manifest().generationRulesVersion()) || ("geoworld-rules-v1.1.30".equals(model.manifest().generationRulesVersion()) || "geoworld-rules-v1.1.31".equals(model.manifest().generationRulesVersion()))))))))))) {
            throw new AssertionError("basin verifier requires v1.1.20");
        }
        var heights = new MinecraftV26HeightMapping();
        var shoreline = new GeoWorldShorelinePlanner(heights);
        int checked = 0;
        int rawRaised = 0;
        int voxelRaised = 0;
        int dryChannel = 0;
        for (int z = 2500; z <= 3000; z += 4) {
            for (int x = 5200; x <= 5700; x += 4) {
                var river = model.sample(x, z);
                if (!"River".equals(river.waterKind()) || !isBasin(river)) {
                    continue;
                }
                int nearest = Integer.MAX_VALUE;
                GeoWorldV1.RuntimeSample bank = null;
                for (int a = 0; a < 16; a++) {
                    double angle = 2.0 * Math.PI * a / 16.0;
                    for (int d = 4; d <= 64; d += 4) {
                        int bx = x + (int)Math.round(Math.cos(angle) * d);
                        int bz = z + (int)Math.round(Math.sin(angle) * d);
                        var candidate = model.sample(bx, bz);
                        if (!"None".equals(candidate.waterKind()) || !isBasin(candidate)) {
                            continue;
                        }
                        if (d < nearest) {
                            nearest = d;
                            bank = candidate;
                        }
                        break;
                    }
                }
                if (bank == null) {
                    continue;
                }
                checked++;
                if (river.waterSurfaceElevation() > bank.groundHeight() + 0.002) {
                    rawRaised++;
                }
                var channelColumn = shoreline.planModel(model, x, z);
                // Test fluid-facing cardinal faces. A bank several blocks
                // downstream may legitimately follow a lower river terrace.
                for (int[] delta : new int[][]{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}) {
                    int nx = x + delta[0];
                    int nz = z + delta[1];
                    var neighbor = shoreline.planModel(model, nx, nz);
                    if ("None".equals(neighbor.sample().waterKind()) &&
                        neighbor.waterBoundary() <= neighbor.groundBoundary() &&
                        channelColumn.waterBoundary() > neighbor.groundBoundary()) {
                        voxelRaised++;
                        if (voxelRaised <= 5) {
                            System.out.printf(
                                "LEAK x=%d z=%d waterY=%d bedY=%d bank=(%d,%d) rawBankY=%d bankY=%d%n",
                                x, z, channelColumn.waterBoundary(), channelColumn.groundBoundary(),
                                nx, nz, neighbor.originalGroundBoundary(), neighbor.groundBoundary()
                            );
                        }
                    }
                }
                if (channelColumn.waterBoundary() <= channelColumn.groundBoundary()) {
                    dryChannel++;
                }
            }
        }
        System.out.printf(
            "BASIN_RIVER seed=123456 checks=%d rawRaised=%d voxelRaised=%d dryChannel=%d%n",
            checked, rawRaised, voxelRaised, dryChannel
        );
        if (checked < 100 || rawRaised != 0 || voxelRaised != 0 || dryChannel != 0) {
            throw new AssertionError("basin river containment failed");
        }
    }

    private static boolean isBasin(GeoWorldV1.RuntimeSample sample) {
        return "basin_cherry".equals(sample.specialFeature()) ||
            "basin_snow".equals(sample.specialFeature());
    }
}
