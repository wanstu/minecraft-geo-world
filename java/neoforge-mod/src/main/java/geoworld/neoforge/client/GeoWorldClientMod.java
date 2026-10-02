package geoworld.neoforge.client;

import geoworld.neoforge.GeoWorldMod;
import geoworld.neoforge.worldgen.GeoWorldWorldPresets;
import net.neoforged.api.distmarker.Dist;
import net.neoforged.bus.api.IEventBus;
import net.neoforged.bus.api.SubscribeEvent;
import net.neoforged.fml.common.Mod;
import net.neoforged.neoforge.client.event.RegisterPresetEditorsEvent;

@Mod(value = GeoWorldMod.MOD_ID, dist = Dist.CLIENT)
public final class GeoWorldClientMod {
    public GeoWorldClientMod(IEventBus modEventBus) {
        modEventBus.register(GeoWorldClientMod.class);
        GeoWorldMod.LOGGER.info("GeoWorld client creation UI hooks loaded");
    }

    @SubscribeEvent
    static void registerPresetEditors(RegisterPresetEditorsEvent event) {
        event.register(
            GeoWorldWorldPresets.GEOWORLD,
            (parent, settings) -> new GeoWorldPresetScreen(parent, settings)
        );
        GeoWorldMod.LOGGER.info(
            "Registered GeoWorld preset editor for {}",
            GeoWorldWorldPresets.GEOWORLD.identifier()
        );
    }
}
