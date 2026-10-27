package geoworld.minecraft.v26_3common.client;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.v26common.client.GeoWorldV26PresetScreen;
import net.minecraft.client.Minecraft;
import geoworld.minecraft.v26_3common.worldgen.GeoWorldBiomeSource;
import geoworld.minecraft.v26_3common.worldgen.GeoWorldChunkGenerator;
import net.minecraft.client.gui.screens.worldselection.CreateWorldScreen;
import net.minecraft.client.gui.screens.worldselection.WorldCreationContext;
import net.minecraft.core.HolderGetter;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.chunk.ChunkGenerator;

/** Minecraft 26.3 facade over the shared 26.x preset UI. */
public final class GeoWorldPresetScreen extends GeoWorldV26PresetScreen {
    private static final WorldgenAdapter WORLDGEN =
        new WorldgenAdapter() {
            @Override
            public GeoWorldCreationConfig configOf(
                ChunkGenerator generator
            ) {
                return generator instanceof GeoWorldChunkGenerator geoworld
                    ? geoworld.config()
                    : null;
            }

            @Override
            public ChunkGenerator create(
                HolderGetter<Biome> biomes,
                GeoWorldCreationConfig config,
                long seed,
                GeoWorldEngineBridge.GeneratedModel generated
            ) {
                GeoWorldBiomeSource biomeSource =
                    GeoWorldBiomeSource.createDefault(biomes);
                GeoWorldChunkGenerator generator =
                    new GeoWorldChunkGenerator(biomeSource, config);
                generator.attachModel(
                    seed,
                    generated.model(),
                    generated.path()
                );
                return generator;
            }
        };

    public GeoWorldPresetScreen(
        CreateWorldScreen parent,
        WorldCreationContext settings
    ) {
        super(
            parent,
            settings,
            WORLDGEN,
            null,
            screen -> Minecraft.getInstance().gui.setScreen(screen)
        );
    }
}
