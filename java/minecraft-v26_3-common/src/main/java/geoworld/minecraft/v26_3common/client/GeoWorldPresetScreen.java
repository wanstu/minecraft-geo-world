package geoworld.minecraft.v26_3common.client;

import geoworld.minecraft.client.GeoWorldPresetScreenBase;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.v26_3common.worldgen.GeoWorldBiomeSource;
import geoworld.minecraft.v26_3common.worldgen.GeoWorldChunkGenerator;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.components.CycleButton;
import net.minecraft.client.gui.screens.worldselection.CreateWorldScreen;
import net.minecraft.client.gui.screens.worldselection.WorldCreationContext;
import net.minecraft.core.HolderGetter;
import net.minecraft.network.chat.Component;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.chunk.ChunkGenerator;

import java.util.List;
import java.util.function.BiConsumer;
import java.util.function.Function;

/** Minecraft 26.3 thin bridge over the shared GeoWorld creation UI. */
public final class GeoWorldPresetScreen extends GeoWorldPresetScreenBase {
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

    private static final UiAdapter UI = new UiAdapter() {
        @Override
        public <T> CycleButton<T> createCycleButton(
            Function<T, Component> valueStringifier,
            T initialValue,
            List<T> values,
            int width,
            Component message,
            BiConsumer<CycleButton<T>, T> onValueChange
        ) {
            return CycleButton
                .builder(valueStringifier, initialValue)
                .withValues(values)
                .create(
                    0,
                    0,
                    width,
                    20,
                    message,
                    (button, value) -> onValueChange.accept(button, value)
                );
        }

        @Override
        public int parameterPageSize() {
            return 0;
        }

        @Override
        public net.minecraft.client.gui.layouts.LayoutElement createContent(
            Minecraft minecraft,
            net.minecraft.client.gui.layouts.LinearLayout body,
            int contentHeight,
            int minWidth
        ) {
            net.minecraft.client.gui.components.ScrollableLayout content =
                new net.minecraft.client.gui.components.ScrollableLayout(
                    minecraft,
                    body,
                    contentHeight
                );
            content.setMinWidth(minWidth);
            return content;
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
            UI,
            screen -> Minecraft.getInstance().gui.setScreen(screen)
        );
    }
}