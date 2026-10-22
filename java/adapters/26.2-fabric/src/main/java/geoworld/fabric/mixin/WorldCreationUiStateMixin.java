package geoworld.fabric.mixin;

import geoworld.minecraft.v26_2common.client.GeoWorldPresetScreen;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26WorldPresets;
import net.minecraft.client.gui.screens.worldselection.PresetEditor;
import net.minecraft.client.gui.screens.worldselection.WorldCreationUiState;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfoReturnable;

@Mixin(WorldCreationUiState.class)
public abstract class WorldCreationUiStateMixin {
    @Inject(
        method = "getPresetEditor",
        at = @At("HEAD"),
        cancellable = true
    )
    private void geoworld$getPresetEditor(
        CallbackInfoReturnable<PresetEditor> callback
    ) {
        WorldCreationUiState state =
            (WorldCreationUiState)(Object)this;
        var preset = state.getWorldType().preset();
        if (
            preset != null &&
            preset.unwrapKey()
                .filter(GeoWorldV26WorldPresets.GEOWORLD::equals)
                .isPresent()
        ) {
            callback.setReturnValue(
                (parent, settings) ->
                    new GeoWorldPresetScreen(parent, settings)
            );
        }
    }
}
