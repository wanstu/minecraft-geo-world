package geoworld.neoforge.client;

import geoworld.neoforge.GeoWorldMod;
import geoworld.neoforge.GeoWorldWorldService;
import geoworld.neoforge.worldgen.GeoWorldWorldPresets;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.screens.PauseScreen;
import net.minecraft.network.chat.Component;
import net.neoforged.api.distmarker.Dist;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.bus.api.SubscribeEvent;
import net.neoforged.fml.common.Mod;
import net.neoforged.neoforge.client.event.RegisterPresetEditorsEvent;
import net.neoforged.neoforge.client.event.ScreenEvent;
import net.neoforged.neoforge.common.NeoForge;

@Mod(value = GeoWorldMod.MOD_ID, dist = Dist.CLIENT)
public final class GeoWorldClientMod {
    public GeoWorldClientMod(IEventBus modEventBus) {
        modEventBus.register(GeoWorldClientMod.class);
        NeoForge.EVENT_BUS.addListener(GeoWorldClientMod::addPausePreviewButton);
        GeoWorldMod.LOGGER.info("GeoWorld client creation UI hooks loaded");
    }

    @SubscribeEvent
    static void registerPresetEditors(RegisterPresetEditorsEvent event) {
        event.register(
            GeoWorldWorldPresets.GEOWORLD,
            (parent, settings) -> new GeoWorldPresetScreen(parent, settings)
        );
        GeoWorldMod.LOGGER.info(
            "Registered GeoWorld preset editor for {}",
            GeoWorldWorldPresets.GEOWORLD.identifier()
        );
    }

    private static void addPausePreviewButton(ScreenEvent.Init.Post event) {
        if (!(event.getScreen() instanceof PauseScreen pauseScreen)) {
            return;
        }

        Minecraft minecraft = Minecraft.getInstance();
        if (minecraft.player == null) {
            return;
        }

        var server = minecraft.getSingleplayerServer();
        if (server == null) {
            return;
        }

        GeoWorldWorldService.active(server).ifPresent(active -> {
            Button preview = Button.builder(
                Component.translatable("geoworld.pause.preview"),
                button -> minecraft.gui.setScreen(
                    new GeoWorldPreviewScreen(
                        pauseScreen,
                        active.model(),
                        minecraft.player.getX(),
                        minecraft.player.getZ()
                    )
                )
            )
                .width(112)
                .build();
            preview.setPosition(
                Math.max(6, pauseScreen.width - preview.getWidth() - 6),
                6
            );
            event.addListener(preview);
        });
    }
}
