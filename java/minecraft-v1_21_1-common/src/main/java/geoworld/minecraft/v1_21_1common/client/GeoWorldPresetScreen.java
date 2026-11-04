package geoworld.minecraft.v1_21_1common.client;

import com.mojang.logging.LogUtils;
import org.slf4j.Logger;
import geoworld.minecraft.client.GeoWorldPresetScreenBase;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.common.engine.GeoWorldMineralContext;
import geoworld.minecraft.v26common.worldgen.GeoWorldOreCatalog;
import geoworld.minecraft.v1_21_1common.worldgen.GeoWorldBiomeSource;
import geoworld.minecraft.v1_21_1common.worldgen.GeoWorldChunkGenerator;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.components.CycleButton;
import net.minecraft.client.gui.screens.worldselection.CreateWorldScreen;
import net.minecraft.client.gui.screens.worldselection.WorldCreationContext;
import net.minecraft.core.HolderGetter;
import net.minecraft.core.registries.Registries;
import net.minecraft.network.chat.Component;
import net.minecraft.world.level.biome.Biome;
import net.minecraft.world.level.chunk.ChunkGenerator;

import java.util.List;
import java.util.function.BiConsumer;
import java.util.function.Function;

/** Minecraft 1.21.1 thin bridge over the shared GeoWorld creation UI. */
public final class GeoWorldPresetScreen extends GeoWorldPresetScreenBase {
    private static final Logger LOGGER = LogUtils.getLogger();
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
                .<T>builder(valueStringifier)
                .withInitialValue(initialValue)
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
            // The 1.21.1 form is now genuinely scrollable, so parameter
            // pagination only adds friction. Render all parameters in one
            // continuous scrollable form.
            return 0;
        }

        @Override
        public net.minecraft.client.gui.layouts.LayoutElement createContent(
            Minecraft minecraft,
            net.minecraft.client.gui.layouts.LinearLayout body,
            int contentHeight,
            int minWidth
        ) {
            return new GeoWorldScrollableContent(
                body,
                contentHeight,
                minWidth
            );
        }
    };

    @Override
    protected void prepareGenerationContext(WorldCreationContext settings) {
        GeoWorldOreCatalog.invalidate();
        var blockRegistry = settings
            .worldgenLoadContext()
            .registryOrThrow(Registries.BLOCK);
        var catalog = GeoWorldOreCatalog.scanWorldgenRegistry(blockRegistry);
        GeoWorldMineralContext.publish(
            catalog.dictionaryFamilies(),
            catalog.families()
        );
        LOGGER.info(
            "GeoWorld 1.21.1 creation mineral context: available={}",
            catalog.families()
        );
    }

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
            screen -> Minecraft.getInstance().setScreen(screen)
        );
    }
}