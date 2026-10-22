package geoworld.minecraft.common.platform;

/** Loader-neutral world-generation registration capability. */
public interface WorldgenRegistrationAdapter<ChunkCodec, BiomeCodec> {
    void registerWorldgen(
        String id,
        ChunkCodec chunkGeneratorCodec,
        BiomeCodec biomeSourceCodec
    );
}
