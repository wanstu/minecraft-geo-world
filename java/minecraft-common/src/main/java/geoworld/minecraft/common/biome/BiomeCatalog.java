package geoworld.minecraft.common.biome;

import java.util.Set;

public interface BiomeCatalog {
    Set<String> candidateBiomeIds();

    String choose(
        GeoWorldBiomeRegion region,
        long worldSeed,
        double x,
        double z,
        String fallback
    );
}
