package geoworld.minecraft.v26common.client;

import net.minecraft.client.gui.screens.Screen;

/**
 * Adapts the client screen transition API, which moved between
 * Minecraft 26.1.2 and 26.2.
 */
@FunctionalInterface
public interface GeoWorldV26ScreenNavigator {
    void setScreen(Screen screen);
}
