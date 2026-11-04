package geoworld.neoforge.client;

import geoworld.minecraft.client.GeoWorldGameWebUi;
import geoworld.minecraft.v1_21_1common.Minecraft1_21_1Profile;
import geoworld.minecraft.v1_21_1common.runtime.GeoWorldWorldService;
import geoworld.minecraft.v26common.runtime.GeoWorldV26WorldService;
import net.minecraft.Util;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.server.MinecraftServer;
import net.minecraft.server.level.ServerPlayer;

import java.net.URI;
import java.util.Optional;

final class GeoWorldWebUiPlatform implements GeoWorldGameWebUi.Platform {
    static final GeoWorldWebUiPlatform INSTANCE = new GeoWorldWebUiPlatform();

    private GeoWorldWebUiPlatform() {}

    @Override
    public String gameClient() {
        return "Minecraft 1.21.1 NeoForge";
    }

    @Override
    public String minecraftVersion() {
        return Minecraft1_21_1Profile.INSTANCE.minecraftVersion();
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
        return server.isSingleplayerOwner(player.getGameProfile());
    }

    @Override
    public Object chunkyInstance() {
        return null;
    }

    @Override
    public void openUri(Screen source, URI uri) {
        Util.getPlatform().openUri(uri);
    }

    @Override
    public void setScreen(Screen screen) {
        Minecraft.getInstance().setScreen(screen);
    }
}