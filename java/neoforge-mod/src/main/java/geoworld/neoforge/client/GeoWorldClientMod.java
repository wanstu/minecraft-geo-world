package geoworld.neoforge.client;

import geoworld.minecraft.client.GeoWorldGameWebUi;
import geoworld.minecraft.client.GeoWorldWebUiSettingsScreen;
import geoworld.minecraft.v26_3common.client.GeoWorldPresetScreen;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26WorldPresets;
import geoworld.neoforge.GeoWorldMod;
import net.neoforged.api.distmarker.Dist;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.bus.api.SubscribeEvent;
import net.neoforged.fml.ModContainer;
import net.neoforged.fml.common.Mod;
import net.neoforged.neoforge.client.event.RegisterPresetEditorsEvent;
import net.neoforged.neoforge.client.gui.IConfigScreenFactory;
import net.neoforged.neoforge.common.NeoForge;
import net.neoforged.neoforge.event.server.ServerStartedEvent;
import net.neoforged.neoforge.event.server.ServerStoppedEvent;

@Mod(value = GeoWorldMod.MOD_ID, dist = Dist.CLIENT)
public final class GeoWorldClientMod {
    public GeoWorldClientMod(
        IEventBus modEventBus,
        ModContainer modContainer
    ) {
        modEventBus.register(GeoWorldClientMod.class);
        GeoWorldGameWebUi.configure(GeoWorldWebUiPlatform.INSTANCE);
        GeoWorldGameWebUi.start();
        IConfigScreenFactory configScreenFactory =
            (container, parent) ->
                new GeoWorldWebUiSettingsScreen(parent);
        modContainer.registerExtensionPoint(
            IConfigScreenFactory.class,
            configScreenFactory
        );
        NeoForge.EVENT_BUS.addListener(GeoWorldClientMod::onServerStarted);
        NeoForge.EVENT_BUS.addListener(GeoWorldClientMod::onServerStopped);
        GeoWorldMod.LOGGER.info("GeoWorld current client hooks loaded");
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

    private static void onServerStarted(ServerStartedEvent event) {
        GeoWorldGameWebUi.bindAfterServerStarted(event.getServer());
    }

    private static void onServerStopped(ServerStoppedEvent event) {
        GeoWorldGameWebUi.unbind(
            "已退出当前世界，WebUI 已解除绑定"
        );
    }
}