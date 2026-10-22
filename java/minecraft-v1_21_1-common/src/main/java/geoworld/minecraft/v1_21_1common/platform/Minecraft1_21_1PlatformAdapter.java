package geoworld.minecraft.v1_21_1common.platform;

import com.mojang.serialization.MapCodec;
import geoworld.minecraft.common.platform.MinecraftPlatformAdapter;
import net.minecraft.server.MinecraftServer;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkGenerator;


public interface Minecraft1_21_1PlatformAdapter extends MinecraftPlatformAdapter<
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
