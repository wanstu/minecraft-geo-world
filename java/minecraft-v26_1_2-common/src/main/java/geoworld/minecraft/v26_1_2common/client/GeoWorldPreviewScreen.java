package geoworld.minecraft.v26_1_2common.client;

import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.v26common.client.GeoWorldV26PreviewScreen;
import geoworld.minecraft.v26_1_2common.Minecraft26_1_2Profile;
import geoworld.minecraft.v26_1_2common.worldgen.GeoWorldBiomeSource;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.Screen;

/** Minecraft 26.1.2 facade over the shared 26.x preview UI. */
public final class GeoWorldPreviewScreen extends GeoWorldV26PreviewScreen {
    public GeoWorldPreviewScreen(
        Screen parent,
        GeoWorldEngineBridge.GeneratedModel generated
    ) {
        super(
            parent,
            generated,
            Minecraft26_1_2Profile.INSTANCE.heightMapping(),
            GeoWorldBiomeSource::minecraftBiomeId,
            screen -> Minecraft.getInstance().setScreen(screen)
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
            Minecraft26_1_2Profile.INSTANCE.heightMapping(),
            GeoWorldBiomeSource::minecraftBiomeId,
            screen -> Minecraft.getInstance().setScreen(screen)
        );
    }
}
