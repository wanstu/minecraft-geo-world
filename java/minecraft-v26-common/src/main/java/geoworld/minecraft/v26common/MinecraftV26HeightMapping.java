package geoworld.minecraft.v26common;

import geoworld.minecraft.common.worldgen.HeightMapping;

/** Shared vertical mapping for supported Minecraft 26.x releases. */
public class MinecraftV26HeightMapping implements HeightMapping {
    public static final int MIN_Y = -64;
    public static final int HEIGHT = 384;
    public static final int MAX_Y_EXCLUSIVE = MIN_Y + HEIGHT;
    public static final int SEA_LEVEL = 63;
    public static final double GEOWORLD_SEA = 0.5;

    @Override
    public int minY() { return MIN_Y; }

    @Override
    public int height() { return HEIGHT; }

    @Override
    public int maxYExclusive() { return MAX_Y_EXCLUSIVE; }

    @Override
    public int seaLevel() { return SEA_LEVEL; }

    @Override
    public int boundaryY(double normalizedHeight) {
        int y = (int)Math.round(mappedHeight(normalizedHeight));
        return clampBoundary(y);
    }

    @Override
    public int boundaryYCeil(double normalizedHeight) {
        int y = (int)Math.ceil(mappedHeight(normalizedHeight) - 1.0e-9);
        return clampBoundary(y);
    }

    @Override
    public int relativeHeightmapValue(int firstAvailableY) {
        if (firstAvailableY < MIN_Y || firstAvailableY > MAX_Y_EXCLUSIVE) {
            throw new IllegalArgumentException(
                "heightmap Y outside build range: " + firstAvailableY
            );
        }
        return firstAvailableY - MIN_Y;
    }

    private static double mappedHeight(double normalizedHeight) {
        double h = clamp(normalizedHeight, 0.0, 1.0);
        if (h <= GEOWORLD_SEA) {
            double t = h / GEOWORLD_SEA;
            return lerp(MIN_Y, SEA_LEVEL, t);
        }
        double t = (h - GEOWORLD_SEA) / (1.0 - GEOWORLD_SEA);
        return lerp(SEA_LEVEL, MAX_Y_EXCLUSIVE, t);
    }

    private static int clampBoundary(int y) {
        return Math.max(MIN_Y, Math.min(MAX_Y_EXCLUSIVE, y));
    }

    private static double lerp(double a, double b, double t) {
        return a + (b - a) * t;
    }

    private static double clamp(double value, double lo, double hi) {
        return Math.max(lo, Math.min(hi, value));
    }
}
