package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.MinecraftVersionProfile;
import geoworld.minecraft.common.worldgen.GeoWorldShorelinePlanner;
import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner;
import geoworld.minecraft.v26common.GeoWorldV26BiomeRealizer;
import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;

import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.ArrayList;
import java.util.Set;
import java.util.HashSet;
import java.util.ArrayDeque;

/** Confirms rare high-rock lodes actually intersect exposed vanilla stone. */
public final class GeoWorldV26SurfaceOutcropVerifier {
    private GeoWorldV26SurfaceOutcropVerifier() {}

    private static GeoWorldV26BiomeRealizer BIOMES;
    private static GeoWorldShorelinePlanner SHORELINE;

    public static void run(
        MinecraftVersionProfile profile,
        String[] args
    ) throws Exception {
        BIOMES = new GeoWorldV26BiomeRealizer(profile);
        SHORELINE = new GeoWorldShorelinePlanner(profile.heightMapping());
        GeoWorldMineralPlanner mineralPlanner =
            new GeoWorldMineralPlanner(profile.heightMapping());
        long seed = args.length > 0
            ? Long.parseLong(args[0]) : -563659577384754347L;
        var model = GeoWorldEngineBridge.generate(
            seed, GeoWorldCreationConfig.defaults()
        ).model();

        // Representative high mountain near the screenshot, rather than only
        // artificial ore-class eligibility values.
        List<GeoWorldMineralPlanner.Placement> planned = new ArrayList<>();
        mineralPlanner.planSurfaceOutcrops(
            model, seed, 1680, 2700, -2950, -1650, planned::add
        );

        int upperDeposits = 0;
        int visibleTopRock = 0;
        int upperCandidateRock = 0;
        GeoWorldMineralPlanner.Placement first = null;
        Set<Long> visibleCoords = new HashSet<>();
        Map<Long, GeoWorldShorelinePlanner.Column> columns = new HashMap<>();
        Map<Long, Set<Integer>> oreColumnHeights = new HashMap<>();
        for (var placement : planned) {
            if (placement.y() < 241) continue;
            upperDeposits++;
            var column = columns.computeIfAbsent(
                coordKey(placement.x(), placement.z()),
                unused -> SHORELINE.planModel(
                    model, placement.x(), placement.z()
                )
            );
            if (column.surfaceOverride() != null ||
                !"None".equals(column.sample().waterKind())) continue;
            int surface = column.groundBoundary() - 1;
            if (placement.y() > surface ||
                placement.y() < surface - 23) continue;

            String biome = BIOMES.biomeIdFromSample(
                model, placement.x(), surface,
                placement.z(), column.sample()
            );
            String top = GeoWorldV26SurfaceMaterials.top(
                biome, column.sample().surfaceProfile(),
                column.surfaceOverride()
            );
            if (!"minecraft:stone".equals(top)) continue;
            upperCandidateRock++;
            oreColumnHeights.computeIfAbsent(
                coordKey(placement.x(), placement.z()),
                ignored -> new HashSet<>()
            ).add(placement.y());
            if (placement.y() == surface) {
                visibleTopRock++;
                visibleCoords.add(coordKey(placement.x(), placement.z()));
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

        int largestExposedPatch = largestPatch(visibleCoords);
        int deepestExposedColumn = 0;
        int thickLodeColumns = 0;
        for (long key : visibleCoords) {
            Set<Integer> ys = oreColumnHeights.get(key);
            if (ys == null) continue;
            int depth = ys.size();
            deepestExposedColumn = Math.max(deepestExposedColumn, depth);
            if (depth >= 4) thickLodeColumns++;
        }
        if (deepestExposedColumn < 5 || thickLodeColumns < 30) {
            throw new AssertionError(
                "mineral bodies do not extend beneath exposed rock: " +
                "deepestColumn=" + deepestExposedColumn +
                ", thickColumns=" + thickLodeColumns
            );
        }
        // The former long ribbons covered more cliff columns but only a
        // thin layer. New granite-like blobs expose smaller, thicker
        // connected patches. Bound both visible coverage and connectivity
        // so the rock face is mineralized without being entirely ore.
        if (seed == -563659577384754347L &&
            (visibleCoords.size() < 280 || largestExposedPatch < 70 ||
                visibleCoords.size() > 1100)) {
            throw new AssertionError(
                "rocky ore outcrop size/density regression: unique=" +
                visibleCoords.size() + " largest=" + largestExposedPatch
            );
        }
        int baseX = Math.floorDiv(first.x(), 16) * 16;
        int baseZ = Math.floorDiv(first.z(), 16) * 16;
        Map<GeoWorldMineralPlanner.Placement, Integer> whole = new HashMap<>();
        Map<GeoWorldMineralPlanner.Placement, Integer> split = new HashMap<>();
        mineralPlanner.planSurfaceOutcrops(
            model, seed, baseX - 16, baseX + 15, baseZ, baseZ + 15,
            p -> whole.merge(p, 1, Integer::sum)
        );
        mineralPlanner.planSurfaceOutcrops(
            model, seed, baseX - 16, baseX - 1, baseZ, baseZ + 15,
            p -> split.merge(p, 1, Integer::sum)
        );
        mineralPlanner.planSurfaceOutcrops(
            model, seed, baseX, baseX + 15, baseZ, baseZ + 15,
            p -> split.merge(p, 1, Integer::sum)
        );
        if (!whole.equals(split) || whole.isEmpty()) {
            throw new AssertionError("surface veins depend on chunk order");
        }
        System.out.printf(
            "GeoWorld high mountain outcrops PASS seed=%d planned=%d " +
            "above240=%d candidateRock=%d visibleStoneTop=%d " +
            "distinctVisible=%d largestConnected=%d " +
            "deepestColumn=%d thickLodeColumns=%d " +
            "example=(%d,%d,%d) chunkBoundary=PASS%n",
            seed, planned.size(), upperDeposits, upperCandidateRock,
            visibleTopRock, visibleCoords.size(), largestExposedPatch,
            deepestExposedColumn, thickLodeColumns,
            first.x(), first.y(), first.z()
        );
    }

    private static long coordKey(int x, int z) {
        return ((long)x << 32) | (z & 0xffffffffL);
    }

    private static int largestPatch(Set<Long> visible) {
        Set<Long> remaining = new HashSet<>(visible);
        int largest = 0;
        ArrayDeque<Long> queue = new ArrayDeque<>();
        while (!remaining.isEmpty()) {
            Long start = remaining.iterator().next();
            remaining.remove(start);
            queue.add(start);
            int size = 0;
            while (!queue.isEmpty()) {
                long point = queue.removeFirst();
                size++;
                int x = (int)(point >> 32);
                int z = (int)point;
                for (int[] direction : new int[][] {
                    {1,0}, {-1,0}, {0,1}, {0,-1}
                }) {
                    long neighbor = coordKey(
                        x + direction[0], z + direction[1]
                    );
                    if (remaining.remove(neighbor)) queue.addLast(neighbor);
                }
            }
            largest = Math.max(largest, size);
        }
        return largest;
    }
}
