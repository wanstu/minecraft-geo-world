package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.runtime.GeoWorldV1;

import java.nio.file.Path;

/** Lightweight shoreline probe for a saved World Model and one reported coordinate. */
public final class GeoWorldShorelinePointProbe {
    private GeoWorldShorelinePointProbe() {}

    public static void main(String[] args) throws Exception {
        if (args.length < 3) {
            throw new IllegalArgumentException(
                "usage: <world.geoworld> <x> <z> [radius]"
            );
        }
        Path modelPath = Path.of(args[0]);
        int x0 = Integer.parseInt(args[1]);
        int z0 = Integer.parseInt(args[2]);
        int radius = args.length >= 4 ? Integer.parseInt(args[3]) : 48;

        GeoWorldV1 runtime = GeoWorldV1.load(modelPath);
        GeoWorldShorelinePlanner shoreline = new GeoWorldShorelinePlanner(
            Minecraft26_3Profile.INSTANCE.heightMapping()
        );

        var center = shoreline.planModel(runtime, x0, z0);
        System.out.printf(
            "POINT seed=%d rules=%s at=(%d,%d) raw=%d ground=%d water=%d shore=%d kind=%s terrain=%s%n",
            runtime.manifest().seed(),
            runtime.manifest().generationRulesVersion(),
            x0,
            z0,
            center.originalGroundBoundary(),
            center.groundBoundary(),
            center.shoreWaterBoundary(),
            center.shoreDistance(),
            center.shoreWaterKind(),
            center.sample().terrainVariant()
        );

        int maxStep = 0;
        int maxX = x0;
        int maxZ = z0;
        String maxAxis = "?";
        int altered = 0;
        int natural = 0;
        int shore32 = 0;
        int infinite = 0;
        for (int z = z0 - radius; z <= z0 + radius; z++) {
            for (int x = x0 - radius; x <= x0 + radius; x++) {
                var c = shoreline.planModel(runtime, x, z);
                if (c.groundBoundary() != c.originalGroundBoundary()) {
                    altered++;
                } else {
                    natural++;
                }
                if (c.shoreDistance() == GeoWorldShorelinePlanner.MAX_ADAPTIVE_BANK_DISTANCE) {
                    shore32++;
                }
                if (c.shoreDistance() == Integer.MAX_VALUE) {
                    infinite++;
                }
                var east = shoreline.planModel(runtime, x + 1, z);
                var south = shoreline.planModel(runtime, x, z + 1);
                int stepE = Math.abs(c.groundBoundary() - east.groundBoundary());
                int stepS = Math.abs(c.groundBoundary() - south.groundBoundary());
                if (stepE > maxStep) {
                    maxStep = stepE;
                    maxX = x;
                    maxZ = z;
                    maxAxis = "E";
                }
                if (stepS > maxStep) {
                    maxStep = stepS;
                    maxX = x;
                    maxZ = z;
                    maxAxis = "S";
                }
            }
        }
        System.out.printf(
            "WINDOW radius=%d altered=%d natural=%d shoreD32=%d shoreInfinite=%d maxStep=%d at=(%d,%d) axis=%s%n",
            radius,
            altered,
            natural,
            shore32,
            infinite,
            maxStep,
            maxX,
            maxZ,
            maxAxis
        );

        printTransect("X", shoreline, runtime, x0, z0, radius, true);
        printTransect("Z", shoreline, runtime, x0, z0, radius, false);
    }

    private static void printTransect(
        String label,
        GeoWorldShorelinePlanner shoreline,
        GeoWorldV1 runtime,
        int x0,
        int z0,
        int radius,
        boolean alongX
    ) {
        System.out.println("TRANSECT " + label + " pos raw->ground/water d kind");
        for (int offset = -radius; offset <= radius; offset++) {
            int x = alongX ? x0 + offset : x0;
            int z = alongX ? z0 : z0 + offset;
            var c = shoreline.planModel(runtime, x, z);
            if (
                offset % 2 == 0 ||
                c.groundBoundary() != c.originalGroundBoundary() ||
                c.shoreDistance() <= 3
            ) {
                System.out.printf(
                    " %d %d,%d %d->%d/%d d=%s %s%n",
                    offset,
                    x,
                    z,
                    c.originalGroundBoundary(),
                    c.groundBoundary(),
                    c.shoreWaterBoundary(),
                    c.shoreDistance() == Integer.MAX_VALUE
                        ? "inf"
                        : Integer.toString(c.shoreDistance()),
                    c.shoreWaterKind()
                );
            }
        }
    }
}
