package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.worldgen.GeoWorldMineralPlanner;
import geoworld.minecraft.common.worldgen.GeoWorldMineralRules;
import geoworld.minecraft.v26common.MinecraftV26HeightMapping;

import java.util.ArrayDeque;
import java.util.HashSet;
import java.util.Set;

/** Ensures every overworld mineral, even the rarest, forms a real 3D vein. */
public final class GeoWorldV26MineralBodyVerifier {
    private GeoWorldV26MineralBodyVerifier() {}

    public static void main(String[] args) {
        GeoWorldMineralPlanner planner = new GeoWorldMineralPlanner(
            new MinecraftV26HeightMapping()
        );
        for (var mineral : GeoWorldMineralRules.Kind.values()) {
            int normal = 0;
            for (boolean exposed : new boolean[]{false, true}) {
                for (long seed : new long[]{71L, 123456L, 10400L}) {
                    Set<GeoWorldMineralPlanner.Placement> body = new HashSet<>();
                    planner.planReferenceBody(
                        mineral, exposed, seed, body::add
                    );
                    if (body.isEmpty()) {
                        throw new AssertionError("empty " + mineral);
                    }
                    int minX = Integer.MAX_VALUE, minY = Integer.MAX_VALUE;
                    int minZ = Integer.MAX_VALUE;
                    int maxX = Integer.MIN_VALUE, maxY = Integer.MIN_VALUE;
                    int maxZ = Integer.MIN_VALUE;
                    for (var p : body) {
                        minX = Math.min(minX, p.x());
                        minY = Math.min(minY, p.y());
                        minZ = Math.min(minZ, p.z());
                        maxX = Math.max(maxX, p.x());
                        maxY = Math.max(maxY, p.y());
                        maxZ = Math.max(maxZ, p.z());
                    }
                    int length = Math.max(maxX - minX, maxZ - minZ) + 1;
                    int width = Math.min(maxX - minX, maxZ - minZ) + 1;
                    int height = maxY - minY + 1;
                    // Demand a real 3D granite-style body, but do not
                    // give diamond the same ore volume as a coal deposit.
                    int minVolume = switch (mineral) {
                        case COAL, IRON, COPPER -> 200;
                        case GOLD, LAPIS, REDSTONE -> 95;
                        case DIAMOND, EMERALD -> 48;
                    };
                    if (body.size() < minVolume ||
                        length < 9 ||
                        width < 4 || length > width * 3.8 ||
                        height < (exposed ? 6 : 4)) {
                        throw new AssertionError(
                            "undersized " + mineral + " exposed=" + exposed +
                            " size=" + body.size() + " length=" + length +
                            " height=" + height
                        );
                    }
                    int connected = largest6ConnectedComponent(body);
                    if (connected < body.size() * 0.96) {
                        throw new AssertionError(
                            "fragmented " + mineral + " exposed=" + exposed +
                            " connected=" + connected + "/" + body.size()
                        );
                    }
                    if (!exposed && seed == 71) normal = body.size();
                    System.out.printf(
                        "ORE_BODY %s exposed=%s seed=%d voxels=%d " +
                        "length=%d height=%d connected=%d%n",
                        mineral, exposed, seed, body.size(),
                        length, height, connected
                    );
                }
            }
            System.out.printf("ORE_FAMILY %s baseVolume=%d PASS%n", mineral, normal);
        }
        System.out.println("GeoWorld mineral body size and continuity PASS 8/8");
    }

    private static int largest6ConnectedComponent(
        Set<GeoWorldMineralPlanner.Placement> points
    ) {
        Set<GeoWorldMineralPlanner.Placement> remaining =
            new HashSet<>(points);
        ArrayDeque<GeoWorldMineralPlanner.Placement> queue =
            new ArrayDeque<>();
        int largest = 0;
        final int[][] dirs = {
            {1,0,0},{-1,0,0},{0,1,0},{0,-1,0},{0,0,1},{0,0,-1}
        };
        while (!remaining.isEmpty()) {
            var initial = remaining.iterator().next();
            remaining.remove(initial);
            queue.add(initial);
            int count = 0;
            while (!queue.isEmpty()) {
                var p = queue.removeFirst();
                count++;
                for (var d : dirs) {
                    var neighbor = new GeoWorldMineralPlanner.Placement(
                        p.x() + d[0], p.y() + d[1], p.z() + d[2], p.mineral()
                    );
                    if (remaining.remove(neighbor)) queue.addLast(neighbor);
                }
            }
            largest = Math.max(largest, count);
        }
        return largest;
    }
}
