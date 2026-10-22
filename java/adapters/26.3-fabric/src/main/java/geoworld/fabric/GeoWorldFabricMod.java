package geoworld.fabric;

import geoworld.minecraft.v26_3common.platform.GeoWorldBootstrap;
import net.fabricmc.api.ModInitializer;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public final class GeoWorldFabricMod implements ModInitializer {
    private static final Logger LOGGER = LoggerFactory.getLogger("GeoWorld");

    @Override
    public void onInitialize() {
        GeoWorldBootstrap.initialize(new GeoWorldFabricPlatform());
        LOGGER.info("GeoWorld Fabric platform adapter loaded");
    }
}
