package geoworld.minecraft.v26_3common.worldgen;

import geoworld.minecraft.v26common.worldgen.GeoWorldStrongholdPreviewCli;
import geoworld.runtime.GeoWorldV1;
import java.util.List;
import java.util.Set;

/** Compatibility entry point; ring math lives in the shared version-aware planner. */
public final class GeoWorldV26_3StrongholdPreviewCli {
    public static final String SET_ID = GeoWorldStrongholdPreviewCli.SET_ID;
    public static final int DISTANCE = GeoWorldStrongholdPreviewCli.DISTANCE;
    public static final int INITIAL_SPREAD = GeoWorldStrongholdPreviewCli.INITIAL_SPREAD;
    public static final int COUNT = GeoWorldStrongholdPreviewCli.COUNT;
    public static final int BIOME_SEARCH_RADIUS_BLOCKS =
        GeoWorldStrongholdPreviewCli.BIOME_SEARCH_RADIUS_BLOCKS;

    private GeoWorldV26_3StrongholdPreviewCli() {}

    public record Position(int index, int chunkX, int chunkZ, int x, int z) {}

    public static Set<String> preferredBiomes() {
        return GeoWorldStrongholdPreviewCli.preferredBiomes("26.3");
    }

    public static List<Position> positions(GeoWorldV1 model) {
        return GeoWorldStrongholdPreviewCli.positions(model, "26.3").stream()
            .map(p -> new Position(p.index(), p.chunkX(), p.chunkZ(), p.x(), p.z()))
            .toList();
    }

    public static void main(String[] args) throws Exception {
        if (args.length != 1) {
            throw new IllegalArgumentException("expected <model.geoworld>");
        }
        GeoWorldStrongholdPreviewCli.main(new String[]{args[0], "26.3"});
    }
}
