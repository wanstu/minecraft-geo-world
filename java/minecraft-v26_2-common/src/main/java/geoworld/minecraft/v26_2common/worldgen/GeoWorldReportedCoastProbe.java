package geoworld.minecraft.v26_2common.worldgen;

import geoworld.minecraft.v26_2common.Minecraft26_2Profile;
import geoworld.minecraft.v26common.worldgen.GeoWorldV26ReportedCoastProbe;

/** Minecraft 26.2 wrapper for shared 26.x verification/benchmark logic. */
public final class GeoWorldReportedCoastProbe {
    private GeoWorldReportedCoastProbe() {}

    public static void main(String[] args) throws Exception {
        GeoWorldV26ReportedCoastProbe.run(
            Minecraft26_2Profile.INSTANCE,
            args
        );
    }
}
