package geoworld.fabric;

import com.mojang.serialization.MapCodec;
import geoworld.minecraft.v1_21_1common.GeoWorldConstants;
import geoworld.minecraft.v1_21_1common.platform.Minecraft1_21_1PlatformAdapter;
import net.minecraft.core.Registry;
import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.resources.ResourceLocation;
import net.minecraft.server.MinecraftServer;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkGenerator;

import java.util.function.Consumer;

public final class GeoWorldFabricPlatform implements Minecraft1_21_1PlatformAdapter {
    @Override
    public void registerWorldgen(
        String id,
        MapCodec<? extends ChunkGenerator> chunkGeneratorCodec,
        MapCodec<? extends BiomeSource> biomeSourceCodec
    ) {
        ResourceLocation key = ResourceLocation.fromNamespaceAndPath(
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
