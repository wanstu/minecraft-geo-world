package geoworld.fabric.client;

import geoworld.minecraft.client.GeoWorldGameWebUi;
import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26_3common.runtime.GeoWorldWorldService;
import geoworld.minecraft.v26common.runtime.GeoWorldV26WorldService;
import net.fabricmc.api.ModInitializer;
import net.fabricmc.loader.api.FabricLoader;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.network.chat.Component;
import net.minecraft.server.MinecraftServer;
import net.minecraft.server.level.ServerPlayer;

import java.net.URI;
import java.util.Optional;

final class GeoWorldWebUiPlatform implements GeoWorldGameWebUi.Platform {
    static final GeoWorldWebUiPlatform INSTANCE = new GeoWorldWebUiPlatform();

    private GeoWorldWebUiPlatform() {}

    @Override
    public String gameClient() {
        return "Minecraft 26.3 Fabric";
    }

    @Override
    public String minecraftVersion() {
        return Minecraft26_3Profile.INSTANCE.minecraftVersion();
    }

    @Override
    public Optional<GeoWorldV26WorldService.ActiveWorld> activeWorld(
        MinecraftServer server
    ) {
        return GeoWorldWorldService.active(server);
    }

    @Override
    public boolean isSingleplayerOwner(
        MinecraftServer server,
        ServerPlayer player
    ) {
        return server.isSingleplayerOwner(player.nameAndId());
    }

    @Override
    public Object chunkyInstance() {
        for (
            var initializer :
                FabricLoader.getInstance().getEntrypoints(
                    "main",
                    ModInitializer.class
                )
        ) {
            if (
                initializer
                    .getClass()
                    .getName()
                    .equals("org.popcraft.chunky.ChunkyFabric")
            ) {
                try {
                    return initializer
                        .getClass()
                        .getMethod("getChunky")
                        .invoke(initializer);
                } catch (Exception ignored) {
                    return null;
                }
            }
        }
        return null;
    }

    @Override
    public void openUri(Screen source, URI uri) {
        ScreenAccess.open(source, uri);
    }

    @Override
    public void setScreen(Screen screen) {
        Minecraft.getInstance().gui.setScreen(screen);
    }

    private static final class ScreenAccess extends Screen {
        private ScreenAccess() {
            super(Component.empty());
        }

        private static void open(Screen source, URI uri) {
            clickUrlAction(Minecraft.getInstance(), source, uri);
        }
    }
}