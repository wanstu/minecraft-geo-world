package geoworld.runtime;

import java.util.ArrayDeque;
import java.util.ArrayList;
import java.util.BitSet;
import java.util.List;
import java.util.Locale;

/**
 * Lazy connected-component index for regional advantages.
 *
 * The index is derived only from the locked World Model. It never observes or
 * loads Minecraft chunks. A component is resolved on first query and cached
 * for subsequent queries, avoiding a whole-world scan for ordinary point
 * lookups.
 */
final class GeoWorldRegionalAdvantageRegions {
    interface ScoreProvider {
        GeoWorldRegionalAdvantage.Score scoreAt(int gridX, int gridZ);
    }

    record Membership(
        String key,
        int cellCount,
        double areaBlocks2,
        double centroidX,
        double centroidZ
    ) {
        private static final Membership NONE =
            new Membership("none", 0, 0, 0, 0);
    }

    private static final double MIN_POTENTIAL = 0.55;

    private final GeoWorldV1.Geometry geometry;
    private final ScoreProvider scores;
    private final GeoWorldRegionalAdvantage.Score[] scoreCache;
    private final boolean[] scoreResolved;
    private final Membership[] membershipCache;

    GeoWorldRegionalAdvantageRegions(
        GeoWorldV1.Geometry geometry,
        ScoreProvider scores
    ) {
        this.geometry = geometry;
        this.scores = scores;
        int count = geometry.nodeCount();
        this.scoreCache = new GeoWorldRegionalAdvantage.Score[count];
        this.scoreResolved = new boolean[count];
        this.membershipCache = new Membership[count];
    }

    synchronized Membership membershipAt(double worldX, double worldZ) {
        int gx = clampGrid(
            (int)Math.round(
                (worldX - geometry.originX()) / geometry.spacingX()
            ),
            geometry.width()
        );
        int gz = clampGrid(
            (int)Math.round(
                (worldZ - geometry.originZ()) / geometry.spacingZ()
            ),
            geometry.height()
        );
        int index = gz * geometry.width() + gx;
        Membership cached = membershipCache[index];
        if (cached != null) {
            return cached;
        }
        resolveComponent(index);
        return membershipCache[index];
    }

    private void resolveComponent(int start) {
        GeoWorldRegionalAdvantage.Score first = scoreAt(start);
        if (!eligible(first)) {
            membershipCache[start] = Membership.NONE;
            return;
        }

        String dominant = first.dominantAdvantage();
        ArrayDeque<Integer> queue = new ArrayDeque<>();
        List<Integer> cells = new ArrayList<>();
        BitSet visited = new BitSet(geometry.nodeCount());
        queue.add(start);
        visited.set(start);

        int anchor = start;
        double sumX = 0;
        double sumZ = 0;
        while (!queue.isEmpty()) {
            int index = queue.removeFirst();
            if (membershipCache[index] != null) {
                continue;
            }
            GeoWorldRegionalAdvantage.Score score = scoreAt(index);
            if (
                !eligible(score) ||
                !dominant.equals(score.dominantAdvantage())
            ) {
                continue;
            }

            cells.add(index);
            anchor = Math.min(anchor, index);
            int gx = index % geometry.width();
            int gz = index / geometry.width();
            sumX += geometry.originX() + gx * geometry.spacingX();
            sumZ += geometry.originZ() + gz * geometry.spacingZ();

            enqueue(index - 1, gx > 0, dominant, visited, queue);
            enqueue(
                index + 1,
                gx + 1 < geometry.width(),
                dominant,
                visited,
                queue
            );
            enqueue(
                index - geometry.width(),
                gz > 0,
                dominant,
                visited,
                queue
            );
            enqueue(
                index + geometry.width(),
                gz + 1 < geometry.height(),
                dominant,
                visited,
                queue
            );
        }

        if (cells.isEmpty()) {
            membershipCache[start] = Membership.NONE;
            return;
        }

        int anchorGX = anchor % geometry.width();
        int anchorGZ = anchor / geometry.width();
        String key = String.format(
            Locale.ROOT,
            "%s-g%04d-%04d",
            dominant,
            anchorGX,
            anchorGZ
        );
        int count = cells.size();
        Membership membership = new Membership(
            key,
            count,
            count * geometry.spacingX() * geometry.spacingZ(),
            sumX / count,
            sumZ / count
        );
        for (int index : cells) {
            membershipCache[index] = membership;
        }
    }

    private void enqueue(
        int index,
        boolean inBounds,
        String dominant,
        BitSet visited,
        ArrayDeque<Integer> queue
    ) {
        if (
            !inBounds ||
            visited.get(index) ||
            membershipCache[index] != null
        ) {
            return;
        }
        visited.set(index);
        GeoWorldRegionalAdvantage.Score score = scoreAt(index);
        if (!eligible(score)) {
            membershipCache[index] = Membership.NONE;
            return;
        }
        if (dominant.equals(score.dominantAdvantage())) {
            queue.addLast(index);
        }
    }

    private GeoWorldRegionalAdvantage.Score scoreAt(int index) {
        if (!scoreResolved[index]) {
            int gx = index % geometry.width();
            int gz = index / geometry.width();
            scoreCache[index] = scores.scoreAt(gx, gz);
            scoreResolved[index] = true;
        }
        return scoreCache[index];
    }

    private static boolean eligible(GeoWorldRegionalAdvantage.Score score) {
        if (score == null) {
            return false;
        }
        String regionalClass = score.regionalClass();
        if (
            regionalClass == null ||
            regionalClass.isEmpty() ||
            "none".equals(regionalClass) ||
            "low_potential".equals(regionalClass)
        ) {
            return false;
        }
        String dominant = score.dominantAdvantage();
        if (
            !"building".equals(dominant) &&
            !"agriculture".equals(dominant) &&
            !"port".equals(dominant) &&
            !"mining".equals(dominant) &&
            !"transport".equals(dominant)
        ) {
            return false;
        }
        double best = Math.max(
            Math.max(score.buildingAdvantage(), score.agricultureAdvantage()),
            Math.max(
                score.portAdvantage(),
                Math.max(
                    score.miningAdvantage(),
                    score.transportAdvantage()
                )
            )
        );
        return best >= MIN_POTENTIAL;
    }

    private static int clampGrid(int value, int size) {
        return Math.max(0, Math.min(size - 1, value));
    }
}
