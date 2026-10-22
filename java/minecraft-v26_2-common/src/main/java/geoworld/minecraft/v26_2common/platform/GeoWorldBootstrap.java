package geoworld.minecraft.v26_2common.platform;

import geoworld.minecraft.v26_2common.GeoWorldConstants;
import geoworld.minecraft.v26_2common.runtime.GeoWorldWorldService;
import geoworld.minecraft.v26_2common.worldgen.GeoWorldBiomeSource;
import geoworld.minecraft.v26_2common.worldgen.GeoWorldChunkGenerator;

public final class GeoWorldBootstrap {
    private GeoWorldBootstrap() {}

    public static void initialize(Minecraft26_2PlatformAdapter platform) {
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
