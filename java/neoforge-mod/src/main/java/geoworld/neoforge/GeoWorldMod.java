package geoworld.neoforge;

import com.mojang.logging.LogUtils;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.bus.api.SubscribeEvent;
import net.neoforged.fml.common.Mod;
import net.neoforged.neoforge.common.NeoForge;
import net.neoforged.neoforge.event.server.ServerStartingEvent;
import net.neoforged.neoforge.event.server.ServerStoppedEvent;
import org.slf4j.Logger;

@Mod(GeoWorldMod.MOD_ID)
public final class GeoWorldMod {
    public static final String MOD_ID = "geoworld";
    public static final Logger LOGGER = LogUtils.getLogger();

    public GeoWorldMod(IEventBus modEventBus) {
        NeoForge.EVENT_BUS.register(this);
        LOGGER.info("GeoWorld NeoForge bridge loaded");
    }

    @SubscribeEvent
    public void onServerStarting(ServerStartingEvent event) {
        GeoWorldWorldService.activate(event.getServer());
    }

    @SubscribeEvent
    public void onServerStopped(ServerStoppedEvent event) {
        GeoWorldWorldService.deactivate(event.getServer());
    }
}
