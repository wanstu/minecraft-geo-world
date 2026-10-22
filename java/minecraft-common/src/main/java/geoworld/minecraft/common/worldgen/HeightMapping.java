package geoworld.minecraft.common.worldgen;

/**
 * Converts GeoWorld's normalized vertical coordinates to one Minecraft
 * version's integer block grid.
 */
public interface HeightMapping {
    int minY();
    int height();
    int maxYExclusive();
    int seaLevel();

    int boundaryY(double normalizedHeight);
    int boundaryYCeil(double normalizedHeight);

    default int relativeHeightmapValue(int firstAvailableY) {
        if (firstAvailableY < minY() || firstAvailableY > maxYExclusive()) {
            throw new IllegalArgumentException(
                "heightmap Y outside build range: " + firstAvailableY
            );
        }
        return firstAvailableY - minY();
    }
}
