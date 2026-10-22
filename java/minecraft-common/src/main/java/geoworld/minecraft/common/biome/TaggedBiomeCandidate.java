package geoworld.minecraft.common.biome;

import java.util.Set;

public record TaggedBiomeCandidate(
    String id,
    Set<GeoWorldBiomeTag> requiredAll,
    Set<GeoWorldBiomeTag> requiredAny,
    Set<GeoWorldBiomeTag> preferred,
    Set<GeoWorldBiomeTag> avoided,
    double tempMin,
    double tempMax,
    double precipMin,
    double precipMax,
    double baseWeight
) {
    public TaggedBiomeCandidate {
        requiredAll = Set.copyOf(requiredAll);
        requiredAny = Set.copyOf(requiredAny);
        preferred = Set.copyOf(preferred);
        avoided = Set.copyOf(avoided);
    }
}
