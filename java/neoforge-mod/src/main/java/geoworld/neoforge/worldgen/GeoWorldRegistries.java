package geoworld.neoforge.worldgen;

import com.mojang.serialization.MapCodec;
import geoworld.neoforge.GeoWorldMod;
import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkGenerator;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.neoforge.registries.DeferredRegister;

public final class GeoWorldRegistries {
    private static final DeferredRegister<MapCodec<? extends ChunkGenerator>>
        CHUNK_GENERATORS = DeferredRegister.create(
            BuiltInRegistries.CHUNK_GENERATOR,
            GeoWorldMod.MOD_ID
        );

    private static final DeferredRegister<MapCodec<? extends BiomeSource>>
        BIOME_SOURCES = DeferredRegister.create(
            BuiltInRegistries.BIOME_SOURCE,
            GeoWorldMod.MOD_ID
        );

    static {
        CHUNK_GENERATORS.register(
            "geoworld",
            () -> GeoWorldChunkGenerator.CODEC
        );
        BIOME_SOURCES.register(
            "geoworld",
            () -> GeoWorldBiomeSource.CODEC
        );
    }

    private GeoWorldRegistries() {}

    public static void register(IEventBus modEventBus) {
        CHUNK_GENERATORS.register(modEventBus);
        BIOME_SOURCES.register(modEventBus);
    }
}
