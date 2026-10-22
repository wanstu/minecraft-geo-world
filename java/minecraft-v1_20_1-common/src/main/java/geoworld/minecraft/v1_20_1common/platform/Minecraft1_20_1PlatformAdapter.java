package geoworld.minecraft.v1_20_1common.platform;

import com.mojang.serialization.Codec;
import geoworld.minecraft.common.platform.MinecraftPlatformAdapter;
import net.minecraft.server.MinecraftServer;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkGenerator;


public interface Minecraft1_20_1PlatformAdapter extends MinecraftPlatformAdapter<
    MinecraftServer,
    Codec<? extends ChunkGenerator>,
    Codec<? extends BiomeSource>
> {
    void registerWorldgen(
        String id,
        Codec<? extends ChunkGenerator> chunkGeneratorCodec,
        Codec<? extends BiomeSource> biomeSourceCodec
    );
}
