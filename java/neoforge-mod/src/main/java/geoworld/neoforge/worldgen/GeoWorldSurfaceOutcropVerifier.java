package geoworld.neoforge.worldgen;

import geoworld.minecraft.v26_3common.Minecraft26_3BiomeRealizer;
import geoworld.neoforge.creation.GeoWorldCreationConfig;
import geoworld.neoforge.engine.GeoWorldEngineBridge;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.ArrayList;

/** Confirms rare high-rock lodes actually intersect exposed vanilla stone. */
public final class GeoWorldSurfaceOutcropVerifier {
    private GeoWorldSurfaceOutcropVerifier() {}

    public static void main(String[] args) throws Exception {
        long seed = args.length > 0
            ? Long.parseLong(args[0]) : -563659577384754347L;
        var model = GeoWorldEngineBridge.generate(
            seed, GeoWorldCreationConfig.defaults()
        ).model();

        // Representative high mountain near the screenshot, rather than only
        // artificial ore-class eligibility values.
        List<GeoWorldMineralVeins.Placement> planned = new ArrayList<>();
        GeoWorldMineralVeins.planSurfaceOutcrops(
            model, seed, 1680, 2700, -2950, -1650, planned::add
        );

        int upperDeposits = 0;
        int visibleTopRock = 0;
        int upperCandidateRock = 0;
        GeoWorldMineralVeins.Placement first = null;
        for (var placement : planned) {
            if (placement.y() < 241) continue;
            upperDeposits++;
            var column = GeoWorldShoreline.plan(
                model, placement.x(), placement.z()
            );
            if (column.surfaceOverride() != null ||
                !"None".equals(column.sample().waterKind())) continue;
            int surface = column.groundBoundary() - 1;
            if (placement.y() > surface ||
                placement.y() < surface - 6) continue;

            String biome = Minecraft26_3BiomeRealizer.biomeIdFromSample(
                model, placement.x(), surface,
                placement.z(), column.sample()
            );
            String top = GeoWorldSurfaceMaterials.top(
                biome, column.sample().surfaceProfile(),
                column.surfaceOverride()
            );
            if (!"minecraft:stone".equals(top)) continue;
            upperCandidateRock++;
            if (placement.y() == surface) {
                visibleTopRock++;
                if (first == null) first = placement;
            }
        }
        if (upperDeposits == 0 || visibleTopRock == 0) {
            throw new AssertionError(
                "no high-elevation exposed vein around test mountain: planned=" +
                planned.size() + " upper=" + upperDeposits +
                " rock=" + upperCandidateRock +
                " visible=" + visibleTopRock
            );
        }

        int baseX = Math.floorDiv(first.x(), 16) * 16;
        int baseZ = Math.floorDiv(first.z(), 16) * 16;
        Map<GeoWorldMineralVeins.Placement, Integer> whole = new HashMap<>();
        Map<GeoWorldMineralVeins.Placement, Integer> split = new HashMap<>();
        GeoWorldMineralVeins.planSurfaceOutcrops(
            model, seed, baseX - 16, baseX + 15, baseZ, baseZ + 15,
            p -> whole.merge(p, 1, Integer::sum)
        );
        GeoWorldMineralVeins.planSurfaceOutcrops(
            model, seed, baseX - 16, baseX - 1, baseZ, baseZ + 15,
            p -> split.merge(p, 1, Integer::sum)
        );
        GeoWorldMineralVeins.planSurfaceOutcrops(
            model, seed, baseX, baseX + 15, baseZ, baseZ + 15,
            p -> split.merge(p, 1, Integer::sum)
        );
        if (!whole.equals(split) || whole.isEmpty()) {
            throw new AssertionError("surface veins depend on chunk order");
        }
        System.out.printf(
            "GeoWorld high mountain outcrops PASS seed=%d planned=%d " +
            "above240=%d candidateRock=%d visibleStoneTop=%d " +
            "example=(%d,%d,%d) chunkBoundary=PASS%n",
            seed, planned.size(), upperDeposits, upperCandidateRock,
            visibleTopRock, first.x(), first.y(), first.z()
        );
    }
}
