package geoworld.neoforge;

import com.mojang.serialization.MapCodec;
import geoworld.minecraft.v26_1_2common.GeoWorldConstants;
import geoworld.minecraft.v26_1_2common.platform.Minecraft26_1_2PlatformAdapter;
import net.minecraft.core.registries.BuiltInRegistries;
import net.minecraft.server.MinecraftServer;
import net.minecraft.world.level.biome.BiomeSource;
import net.minecraft.world.level.chunk.ChunkGenerator;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.neoforge.common.NeoForge;
import net.neoforged.neoforge.event.server.ServerStartingEvent;
import net.neoforged.neoforge.event.server.ServerStoppedEvent;
import net.neoforged.neoforge.registries.DeferredRegister;

import java.util.function.Consumer;

public final class GeoWorldNeoForgePlatform implements Minecraft26_1_2PlatformAdapter {
    private final IEventBus modEventBus;

    private final DeferredRegister<MapCodec<? extends ChunkGenerator>>
        chunkGenerators = DeferredRegister.create(
            BuiltInRegistries.CHUNK_GENERATOR,
            GeoWorldConstants.MOD_ID
        );

    private final DeferredRegister<MapCodec<? extends BiomeSource>>
        biomeSources = DeferredRegister.create(
            BuiltInRegistries.BIOME_SOURCE,
            GeoWorldConstants.MOD_ID
        );

    public GeoWorldNeoForgePlatform(IEventBus modEventBus) {
        this.modEventBus = modEventBus;
    }

    @Override
    public void registerWorldgen(
        String id,
        MapCodec<? extends ChunkGenerator> chunkGeneratorCodec,
        MapCodec<? extends BiomeSource> biomeSourceCodec
    ) {
        chunkGenerators.register(id, () -> chunkGeneratorCodec);
        biomeSources.register(id, () -> biomeSourceCodec);
        chunkGenerators.register(modEventBus);
        biomeSources.register(modEventBus);
    }

    @Override
    public void registerServerLifecycle(
        Consumer<MinecraftServer> onReady,
        Consumer<MinecraftServer> onStopped
    ) {
        NeoForge.EVENT_BUS.addListener(
            (ServerStartingEvent event) -> onReady.accept(event.getServer())
        );
        NeoForge.EVENT_BUS.addListener(
            (ServerStoppedEvent event) -> onStopped.accept(event.getServer())
        );
    }
}
