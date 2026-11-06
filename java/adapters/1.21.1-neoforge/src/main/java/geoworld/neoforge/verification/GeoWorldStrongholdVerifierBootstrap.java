package geoworld.neoforge.verification;

import geoworld.minecraft.v1_21_1common.worldgen.GeoWorldStrongholdPreviewVerifier;
import net.neoforged.fml.loading.LoadingModList;

import java.util.List;
import java.util.Map;

/** Provides the empty mod list expected by NeoForge's vanilla registry bootstrap. */
public final class GeoWorldStrongholdVerifierBootstrap {
    private GeoWorldStrongholdVerifierBootstrap() {}

    public static void main(String[] args) throws Exception {
        if (LoadingModList.get() == null) {
            LoadingModList.of(List.of(), List.of(), List.of(), List.of(), Map.of());
        }
        GeoWorldStrongholdPreviewVerifier.main(args);
    }
}
