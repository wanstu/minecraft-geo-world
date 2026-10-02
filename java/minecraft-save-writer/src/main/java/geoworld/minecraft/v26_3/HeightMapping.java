package geoworld.minecraft.v26_3;

import geoworld.minecraft.v26_3common.Minecraft26_3HeightMapping;

public final class HeightMapping {
    public static final int MIN_Y = Minecraft26_3HeightMapping.MIN_Y;
    public static final int HEIGHT = Minecraft26_3HeightMapping.HEIGHT;
    public static final int MAX_Y_EXCLUSIVE =
        Minecraft26_3HeightMapping.MAX_Y_EXCLUSIVE;
    public static final int SEA_LEVEL = Minecraft26_3HeightMapping.SEA_LEVEL;
    public static final double GEOWORLD_SEA =
        Minecraft26_3HeightMapping.GEOWORLD_SEA;

    private final Minecraft26_3HeightMapping delegate =
        new Minecraft26_3HeightMapping();

    public int boundaryY(double normalizedHeight) {
        return this.delegate.boundaryY(normalizedHeight);
    }

    public int relativeHeightmapValue(int firstAvailableY) {
        return this.delegate.relativeHeightmapValue(firstAvailableY);
    }
}
