package geoworld.neoforge.client;

import geoworld.neoforge.GeoWorldMod;
import geoworld.minecraft.v26_3common.client.GeoWorldPresetScreen;
import geoworld.minecraft.v26_3common.client.GeoWorldPreviewScreen;
import geoworld.minecraft.v26_3common.runtime.GeoWorldWorldService;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26WorldPresets;
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
            GeoWorldV26WorldPresets.GEOWORLD,
            (parent, settings) -> new GeoWorldPresetScreen(parent, settings)
        );
        GeoWorldMod.LOGGER.info(
            "Registered GeoWorld preset editor for {}",
            GeoWorldV26WorldPresets.GEOWORLD.identifier()
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
                .width(204)
                .build();

            int menuBottom = 0;
            for (var child : pauseScreen.children()) {
                if (
                    child instanceof Button existing &&
                    existing.getWidth() >= 98 &&
                    Math.abs(
                        existing.getX() +
                        existing.getWidth() / 2 -
                        pauseScreen.width / 2
                    ) <= 110
                ) {
                    menuBottom = Math.max(
                        menuBottom,
                        existing.getY() + existing.getHeight()
                    );
                }
            }
            int y = Math.min(
                pauseScreen.height - preview.getHeight() - 8,
                menuBottom + 4
            );
            preview.setPosition(
                (pauseScreen.width - preview.getWidth()) / 2,
                Math.max(6, y)
            );
            event.addListener(preview);
        });
    }
}
