package geoworld.minecraft.common.platform;

/**
 * Loader-neutral and Minecraft-version-neutral registration surface.
 *
 * Each version module binds the generic server and codec types to the
 * concrete Minecraft API exposed by that version.
 */
public interface MinecraftPlatformAdapter<S, ChunkCodec, BiomeCodec>
    extends ServerLifecycleAdapter<S>,
            WorldgenRegistrationAdapter<ChunkCodec, BiomeCodec> {
}
