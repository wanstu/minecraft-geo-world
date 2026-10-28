package geoworld.fabric.client;

import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.network.chat.Component;

import java.net.URI;

final class GeoWorldWebUiSettingsScreen extends Screen {
    private final Screen parent;

    GeoWorldWebUiSettingsScreen(Screen parent) {
        super(Component.literal("GeoWorld WebUI"));
        this.parent = parent;
    }

    @Override
    protected void init() {
        int buttonWidth = Math.min(360, Math.max(200, this.width - 48));
        int x = (this.width - buttonWidth) / 2;
        int y = Math.max(40, this.height / 2 - 18);

        URI uri = GeoWorldGameWebUi.currentWebUiUri();
        String openLabel;
        if (GeoWorldGameWebUi.isRunning() && uri != null) {
            openLabel =
                "打开 GeoWorld WebUI · 127.0.0.1:" +
                GeoWorldGameWebUi.currentWebUiPort();
        } else {
            openLabel = "启动并打开 GeoWorld WebUI";
        }

        this.addRenderableWidget(
            Button.builder(
                Component.literal(openLabel),
                button -> openWebUi()
            ).bounds(x, y, buttonWidth, 20).build()
        );

        this.addRenderableWidget(
            Button.builder(
                Component.literal("完成"),
                button -> onClose()
            ).bounds(x, y + 28, buttonWidth, 20).build()
        );
    }

    private void openWebUi() {
        if (!GeoWorldGameWebUi.isRunning()) {
            GeoWorldGameWebUi.start();
        }
        URI uri = GeoWorldGameWebUi.currentWebUiUri();
        if (uri != null) {
            clickUrlAction(this.minecraft, this, uri);
        }
    }

    @Override
    public void onClose() {
        if (this.minecraft != null) {
            this.minecraft.setScreenAndShow(this.parent);
        }
    }
}
