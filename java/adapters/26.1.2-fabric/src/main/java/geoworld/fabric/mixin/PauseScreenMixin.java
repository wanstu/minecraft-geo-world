package geoworld.fabric.mixin;

import geoworld.minecraft.v26_1_2common.client.GeoWorldPreviewScreen;
import geoworld.minecraft.v26_1_2common.runtime.GeoWorldWorldService;
import net.minecraft.client.gui.components.Button;
import net.minecraft.client.gui.screens.PauseScreen;
import net.minecraft.client.gui.screens.Screen;
import net.minecraft.network.chat.Component;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

@Mixin(PauseScreen.class)
public abstract class PauseScreenMixin extends Screen {
    protected PauseScreenMixin(Component title) {
        super(title);
    }

    @Inject(method = "init", at = @At("TAIL"))
    private void geoworld$addPreviewButton(CallbackInfo callback) {
        if (this.minecraft == null || this.minecraft.player == null) {
            return;
        }

        var server = this.minecraft.getSingleplayerServer();
        if (server == null) {
            return;
        }

        GeoWorldWorldService.active(server).ifPresent(active -> {
            PauseScreen parent = (PauseScreen)(Object)this;
            Button preview = Button.builder(
                Component.translatable("geoworld.pause.preview"),
                button -> this.minecraft.setScreenAndShow(
                    new GeoWorldPreviewScreen(
                        parent,
                        active.model(),
                        this.minecraft.player.getX(),
                        this.minecraft.player.getZ()
                    )
                )
            ).width(204).build();

            int menuBottom = 0;
            for (var child : this.children()) {
                if (
                    child instanceof Button existing &&
                    existing.getWidth() >= 98 &&
                    Math.abs(
                        existing.getX() +
                        existing.getWidth() / 2 -
                        this.width / 2
                    ) <= 110
                ) {
                    menuBottom = Math.max(
                        menuBottom,
                        existing.getY() + existing.getHeight()
                    );
                }
            }

            int y = Math.min(
                this.height - preview.getHeight() - 8,
                menuBottom + 4
            );
            preview.setPosition(
                (this.width - preview.getWidth()) / 2,
                y
            );
            this.addRenderableWidget(preview);
        });
    }
}
