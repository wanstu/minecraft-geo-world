package geoworld.fabric.client;

import geoworld.minecraft.v26_3common.client.GeoWorldPreviewScreen;
import geoworld.minecraft.v26_3common.runtime.GeoWorldWorldService;
import net.fabricmc.api.ClientModInitializer;
import net.fabricmc.fabric.api.client.screen.v1.ScreenEvents;
import net.fabricmc.fabric.api.client.screen.v1.Screens;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.screens.PauseScreen;
import net.minecraft.network.chat.Component;

public final class GeoWorldFabricClientMod implements ClientModInitializer {
    @Override
    public void onInitializeClient() {
        ScreenEvents.AFTER_INIT.register((client, screen, scaledWidth, scaledHeight) -> {
            if (!(screen instanceof PauseScreen pauseScreen)) {
                return;
            }
            addPausePreviewButton(client, pauseScreen);
        });
    }

    private static void addPausePreviewButton(
        Minecraft minecraft,
        PauseScreen pauseScreen
    ) {
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
            ).width(204).build();

            int menuBottom = 0;
            for (var child : Screens.getWidgets(pauseScreen)) {
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
                y
            );
            Screens.getWidgets(pauseScreen).add(preview);
        });
    }
}
