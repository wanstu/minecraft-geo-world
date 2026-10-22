package geoworld.neoforge;

import com.mojang.logging.LogUtils;
import geoworld.minecraft.v26_3common.GeoWorldConstants;
import geoworld.minecraft.v26_3common.platform.GeoWorldBootstrap;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.fml.common.Mod;
import org.slf4j.Logger;

@Mod(GeoWorldConstants.MOD_ID)
public final class GeoWorldMod {
    public static final String MOD_ID = GeoWorldConstants.MOD_ID;
    public static final Logger LOGGER = LogUtils.getLogger();

    public GeoWorldMod(IEventBus modEventBus) {
        GeoWorldBootstrap.initialize(
            new GeoWorldNeoForgePlatform(modEventBus)
        );
        LOGGER.info("GeoWorld NeoForge platform adapter loaded");
    }
}
