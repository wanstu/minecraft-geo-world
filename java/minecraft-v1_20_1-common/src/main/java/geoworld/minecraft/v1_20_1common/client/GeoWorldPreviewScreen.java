package geoworld.minecraft.v1_20_1common.client;

import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.v1_20_1common.Minecraft1_20_1Profile;
import geoworld.minecraft.v1_20_1common.worldgen.GeoWorldBiomeSource;
import geoworld.runtime.GeoWorldV1;
import net.minecraft.client.Minecraft;
import net.minecraft.client.gui.screens.Screen;

/** Minecraft 1.20.1 facade over the shared 26.x preview UI. */
public final class GeoWorldPreviewScreen extends GeoWorld1_20_1PreviewScreenBase {
    public GeoWorldPreviewScreen(
        Screen parent,
        GeoWorldEngineBridge.GeneratedModel generated
    ) {
        super(
            parent,
            generated,
            Minecraft1_20_1Profile.INSTANCE.heightMapping(),
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
            Minecraft1_20_1Profile.INSTANCE.heightMapping(),
            GeoWorldBiomeSource::minecraftBiomeId,
            screen -> Minecraft.getInstance().setScreen(screen)
        );
    }
}
