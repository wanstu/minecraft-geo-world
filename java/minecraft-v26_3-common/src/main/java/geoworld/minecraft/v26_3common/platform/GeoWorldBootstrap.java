package geoworld.minecraft.v26_3common.platform;

import geoworld.minecraft.v26_3common.GeoWorldConstants;
import geoworld.minecraft.v26_3common.runtime.GeoWorldWorldService;
import geoworld.minecraft.v26_3common.worldgen.GeoWorldBiomeSource;
import geoworld.minecraft.v26_3common.worldgen.GeoWorldChunkGenerator;

public final class GeoWorldBootstrap {
    private GeoWorldBootstrap() {}

    public static void initialize(Minecraft26_3PlatformAdapter platform) {
        platform.registerWorldgen(
            GeoWorldConstants.MOD_ID,
            GeoWorldChunkGenerator.CODEC,
            GeoWorldBiomeSource.CODEC
        );
        platform.registerServerLifecycle(
            GeoWorldWorldService::activate,
            GeoWorldWorldService::deactivate
        );
    }
}
