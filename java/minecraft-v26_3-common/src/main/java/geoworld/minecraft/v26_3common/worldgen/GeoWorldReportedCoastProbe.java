package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26ReportedCoastProbe;

/** Minecraft 26.3 wrapper for shared 26.x verification/benchmark logic. */
public final class GeoWorldReportedCoastProbe {
    private GeoWorldReportedCoastProbe() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26ReportedCoastProbe.run(
            Minecraft26_3Profile.INSTANCE,
            args
        );
    }
}
