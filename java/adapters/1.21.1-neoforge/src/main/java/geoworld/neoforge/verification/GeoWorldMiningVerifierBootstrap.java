package geoworld.neoforge.verification;

import geoworld.minecraft.v1_21_1common.worldgen.GeoWorldMiningMaterialVerifier;
import net.neoforged.fml.loading.LoadingModList;
import java.util.List;
import java.util.Map;

/** Isolated native registry bootstrap, matching the existing stronghold verifier. */
public final class GeoWorldMiningVerifierBootstrap {
    private GeoWorldMiningVerifierBootstrap() {}
    public static void main(String[] args) throws Exception {
        if (LoadingModList.get() == null) {
            LoadingModList.of(List.of(), List.of(), List.of(), List.of(), Map.of());
        }
        GeoWorldMiningMaterialVerifier.main(args);
    }
}
