package geoworld.minecraft.common.biome;

import java.util.Set;

public record GeoWorldBiomeRegion(
    Set<GeoWorldBiomeTag> tags,
    double annualMeanTempC,
    double annualPrecipMM,
    double monsoonIndex
) {
    public GeoWorldBiomeRegion {
        tags = Set.copyOf(tags);
    }
}
