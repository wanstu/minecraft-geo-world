package geoworld.fabric.client;

import net.fabricmc.api.ClientModInitializer;
import net.fabricmc.fabric.api.event.lifecycle.v1.ServerLifecycleEvents;

public final class GeoWorldFabricClientMod implements ClientModInitializer {
    @Override
    public void onInitializeClient() {
        GeoWorldGameWebUi.start();
        ServerLifecycleEvents.SERVER_STARTED.register(GeoWorldGameWebUi::bind);
        ServerLifecycleEvents.SERVER_STOPPED.register(
            server -> GeoWorldGameWebUi.unbind("已退出当前世界，WebUI 已解除绑定")
        );
    }
}
