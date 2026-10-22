package geoworld.minecraft.v26_2common.platform;

import com.mojang.serialization.MapCodec;
import geoworld.minecraft.common.platform.MinecraftPlatformAdapter;
import net.minecraft.server.MinecraftServer;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkGenerator;


public interface Minecraft26_2PlatformAdapter extends MinecraftPlatformAdapter<
    MinecraftServer,
    MapCodec<? extends ChunkGenerator>,
    MapCodec<? extends BiomeSource>
> {
    void registerWorldgen(
        String id,
        MapCodec<? extends ChunkGenerator> chunkGeneratorCodec,
        MapCodec<? extends BiomeSource> biomeSourceCodec
    );
}
