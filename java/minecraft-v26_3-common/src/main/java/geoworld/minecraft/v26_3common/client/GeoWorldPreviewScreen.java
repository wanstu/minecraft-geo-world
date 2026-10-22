package geoworld.minecraft.v26_3common.client;

import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.v26common.client.GeoWorldV26PreviewScreen;
import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26_3common.worldgen.GeoWorldBiomeSource;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.Screen;

/** Minecraft 26.3 facade over the shared 26.x preview UI. */
public final class GeoWorldPreviewScreen extends GeoWorldV26PreviewScreen {
    public GeoWorldPreviewScreen(
        Screen parent,
        GeoWorldEngineBridge.GeneratedModel generated
    ) {
        super(
            parent,
            generated,
            Minecraft26_3Profile.INSTANCE.heightMapping(),
            GeoWorldBiomeSource::minecraftBiomeId,
            screen -> Minecraft.getInstance().gui.setScreen(screen)
        );
    }

    public GeoWorldPreviewScreen(
        Screen parent,
        GeoWorldV1 model,
        Double focusX,
        Double focusZ
    ) {
        super(
            parent,
            model,
            focusX,
            focusZ,
            Minecraft26_3Profile.INSTANCE.heightMapping(),
            GeoWorldBiomeSource::minecraftBiomeId,
            screen -> Minecraft.getInstance().gui.setScreen(screen)
        );
    }
}
