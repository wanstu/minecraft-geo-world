package geoworld.fabric.mixin;

import geoworld.fabric.GeoWorldFabricLifecycle;
import net.minecraft.server.MinecraftServer;
import org.spongepowered.asm.mixin.Mixin;
import org.spongepowered.asm.mixin.injection.At;
import org.spongepowered.asm.mixin.injection.Inject;
import org.spongepowered.asm.mixin.injection.callback.CallbackInfo;

@Mixin(MinecraftServer.class)
public abstract class MinecraftServerLifecycleMixin {
    @Inject(method = "loadLevel", at = @At("TAIL"))
    private void geoworld$serverReady(CallbackInfo callback) {
        GeoWorldFabricLifecycle.serverReady(
            (MinecraftServer)(Object)this
        );
    }

    @Inject(method = "stopServer", at = @At("HEAD"))
    private void geoworld$serverStopped(CallbackInfo callback) {
        GeoWorldFabricLifecycle.serverStopped(
            (MinecraftServer)(Object)this
        );
    }
}
