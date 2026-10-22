package geoworld.minecraft.v1_20_1common.platform;

import geoworld.minecraft.v1_20_1common.GeoWorldConstants;
import geoworld.minecraft.v1_20_1common.runtime.GeoWorldWorldService;
import geoworld.minecraft.v1_20_1common.worldgen.GeoWorldBiomeSource;
import geoworld.minecraft.v1_20_1common.worldgen.GeoWorldChunkGenerator;

public final class GeoWorldBootstrap {
    private GeoWorldBootstrap() {}

    public static void initialize(Minecraft1_20_1PlatformAdapter platform) {
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
