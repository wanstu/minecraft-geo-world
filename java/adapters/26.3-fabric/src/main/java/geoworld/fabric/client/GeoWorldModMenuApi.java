package geoworld.fabric.client;

import com.terraformersmc.modmenu.api.ConfigScreenFactory;
import com.terraformersmc.modmenu.api.ModMenuApi;
import geoworld.minecraft.client.GeoWorldWebUiSettingsScreen;

public final class GeoWorldModMenuApi implements ModMenuApi {
    @Override
    public ConfigScreenFactory<?> getModConfigScreenFactory() {
        return GeoWorldWebUiSettingsScreen::new;
    }
}