package geoworld.fabric;

import com.mojang.serialization.Codec;
import geoworld.minecraft.v1_20_1common.GeoWorldConstants;
import geoworld.minecraft.v1_20_1common.platform.Minecraft1_20_1PlatformAdapter;
import net.minecraft.core.Registry;
import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.resources.ResourceLocation;
import net.minecraft.server.MinecraftServer;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkGenerator;

import java.util.function.Consumer;

public final class GeoWorldFabricPlatform implements Minecraft1_20_1PlatformAdapter {
    @Override
    public void registerWorldgen(
        String id,
        Codec<? extends ChunkGenerator> chunkGeneratorCodec,
        Codec<? extends BiomeSource> biomeSourceCodec
    ) {
        ResourceLocation key = new ResourceLocation(
            GeoWorldConstants.MOD_ID,
            id
        );
        Registry.register(
            BuiltInRegistries.CHUNK_GENERATOR,
            key,
            chunkGeneratorCodec
        );
        Registry.register(
            BuiltInRegistries.BIOME_SOURCE,
            key,
            biomeSourceCodec
        );
    }

    @Override
    public void registerServerLifecycle(
        Consumer<MinecraftServer> onReady,
        Consumer<MinecraftServer> onStopped
    ) {
        GeoWorldFabricLifecycle.register(onReady, onStopped);
    }
}
