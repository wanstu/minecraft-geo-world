package geoworld.minecraft.v26common.worldgen;

import geoworld.minecraft.common.creation.GeoWorldCreationConfig;
import geoworld.minecraft.common.engine.GeoWorldEngineBridge;
import geoworld.minecraft.common.worldgen.GeoWorldMiningDistrictPlanner;
import geoworld.minecraft.v26common.MinecraftV26HeightMapping;
import geoworld.runtime.GeoWorldV1;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * v1.1.15 mining-district acceptance: districts must be rare on the world map,
 * internally rich, and deterministic across chunk request boundaries.
 */
public final class GeoWorldV26MiningDistrictVerifier {
    private GeoWorldV26MiningDistrictVerifier() {}

    public static void main(String[] args) throws Exception {
        long seed = args.length > 0
            ? Long.parseLong(args[0])
            : 123456L;
        if (!GeoWorldCreationConfig.defaults().surfaceOrePile()) {
            throw new AssertionError(
                "surface ore pile must be enabled by default"
            );
        }
        GeoWorldV1 model = GeoWorldEngineBridge.generate(
            seed,
            GeoWorldCreationConfig.defaults()
        ).model();
        if (!model.surfaceOrePileEnabled()) {
            throw new AssertionError(
                "surface ore pile option was not persisted into World Model"
            );
        }
        if (!("geoworld-rules-v1.1.22".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.23".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.24".equals(
            model.manifest().generationRulesVersion()
        ) || ("geoworld-rules-v1.1.25".equals(
            model.manifest().generationRulesVersion()
        ) || "geoworld-rules-v1.1.26".equals(
            model.manifest().generationRulesVersion()
        )))))) {
            throw new AssertionError(
                "mining district verifier requires v1.1.16, got " +
                model.manifest().generationRulesVersion()
            );
        }

        var g = model.manifest().geometry();
        int minX = (int)Math.floor(g.originX());
        int maxX = (int)Math.ceil(g.originX() + g.spanXBlocks());
        int minZ = (int)Math.floor(g.originZ());
        int maxZ = (int)Math.ceil(g.originZ() + g.spanZBlocks());

        MinecraftV26HeightMapping heightMapping =
            new MinecraftV26HeightMapping();
        GeoWorldMiningDistrictPlanner planner =
            new GeoWorldMiningDistrictPlanner(heightMapping);
        List<GeoWorldMiningDistrictPlanner.District> districts =
            new ArrayList<>();
        planner.planDistricts(
            model, seed,
            minX, maxX, minZ, maxZ,
            districts::add
        );
        if (districts.isEmpty()) {
            throw new AssertionError("no mining districts generated");
        }

        double worldArea =
            Math.max(1.0, g.spanXBlocks() * g.spanZBlocks());
        double summedDistrictArea = 0.0;
        for (var district : districts) {
            summedDistrictArea +=
                Math.PI * district.radius() * district.radius();
            if (district.primaryLodes() < 8) {
                throw new AssertionError(
                    "district has too few primary lodes: " + district
                );
            }
            if (district.companionLodes() < 3) {
                throw new AssertionError(
                    "district has too few companion lodes: " + district
                );
            }
        }
        double nominalCoverage = summedDistrictArea / worldArea;
        if (nominalCoverage > 0.16) {
            throw new AssertionError(
                "mining districts are too common: nominalCoverage=" +
                nominalCoverage + " districts=" + districts.size()
            );
        }

        var probe = districts.get(0);
        int probeX = (int)Math.round(probe.centerX());
        int probeZ = (int)Math.round(probe.centerZ());
        int radius = 64;

        Set<GeoWorldV26MineralVeins.SurfaceOrePileColumn> allPiles =
            new HashSet<>();
        GeoWorldV26MineralVeins.planSurfaceOrePiles(
            model,
            seed,
            minX,
            maxX,
            minZ,
            maxZ,
            allPiles::add
        );
        if (allPiles.isEmpty()) {
            throw new AssertionError(
                "surface ore pile option produced no marker piles"
            );
        }

        Map<Long, Set<GeoWorldV26MineralVeins.SurfaceOrePileColumn>>
            pilesByDistrict = new HashMap<>();
        for (var column : allPiles) {
            pilesByDistrict
                .computeIfAbsent(
                    column.districtId(),
                    ignored -> new HashSet<>()
                )
                .add(column);
        }

        districts.stream()
            .filter(district -> !"None".equals(
                model.sample(
                    (int)Math.round(district.centerX()),
                    (int)Math.round(district.centerZ())
                ).waterKind()
            ))
            .findFirst()
            .ifPresent(district -> {
                if (!pilesByDistrict.containsKey(district.id())) {
                    throw new AssertionError(
                        "underwater mining district lost its surface ore pile: " +
                        district.id()
                    );
                }
            });

        GeoWorldMiningDistrictPlanner.District pileProbe = districts.stream()
            .filter(district -> pilesByDistrict.containsKey(district.id()))
            .findFirst()
            .orElseThrow(() -> new AssertionError(
                "surface marker pile was not tied to a planned district"
            ));
        Set<GeoWorldV26MineralVeins.SurfaceOrePileColumn> pileWhole =
            pilesByDistrict.get(pileProbe.id());
        if (pileWhole.size() < 12 || pileWhole.size() > 120) {
            throw new AssertionError(
                "surface ore pile must stay a small marker: columns=" +
                pileWhole.size()
            );
        }
        long companionPileColumns = pileWhole.stream()
            .filter(GeoWorldV26MineralVeins.SurfaceOrePileColumn::companion)
            .count();
        double companionPileShare =
            (double)companionPileColumns / pileWhole.size();
        if (
            companionPileColumns == 0 ||
            companionPileColumns == pileWhole.size() ||
            companionPileShare < 0.10 ||
            companionPileShare > 0.30
        ) {
            throw new AssertionError(
                "surface ore pile must mix primary and companion ores: " +
                "companionShare=" + companionPileShare +
                " columns=" + pileWhole.size()
            );
        }

        int centerPileX = (int)Math.round(pileProbe.centerX());
        int centerPileZ = (int)Math.round(pileProbe.centerZ());
        var centerPileColumn = pileWhole.stream()
            .filter(column ->
                column.x() == centerPileX && column.z() == centerPileZ
            )
            .findFirst()
            .orElseThrow(() -> new AssertionError(
                "surface ore pile has no center peak column"
            ));
        if (
            centerPileColumn.levels() < 5 ||
            centerPileColumn.levels() > 8
        ) {
            throw new AssertionError(
                "surface ore pile peak height must be 5-8: " +
                centerPileColumn
            );
        }

        double furthestPileColumn = 0.0;
        boolean hasLowerSlope = false;
        for (var column : pileWhole) {
            if (column.levels() < 1 || column.levels() > 8) {
                throw new AssertionError(
                    "surface ore pile column height out of range: " + column
                );
            }
            if (column.levels() <= centerPileColumn.levels() - 2) {
                hasLowerSlope = true;
            }
            furthestPileColumn = Math.max(
                furthestPileColumn,
                Math.hypot(
                    column.x() - pileProbe.centerX(),
                    column.z() - pileProbe.centerZ()
                )
            );
        }
        if (!hasLowerSlope) {
            throw new AssertionError(
                "surface ore pile must form a mound with lower outer columns"
            );
        }
        if (furthestPileColumn > 72.0) {
            throw new AssertionError(
                "surface ore pile drifted too far from district center: " +
                furthestPileColumn
            );
        }

        Set<GeoWorldV26MineralVeins.SurfaceOrePileColumn> pileSplit =
            new HashSet<>();
        int splitX = (minX + maxX) / 2;
        GeoWorldV26MineralVeins.planSurfaceOrePiles(
            model,
            seed,
            minX,
            splitX,
            minZ,
            maxZ,
            pileSplit::add
        );
        GeoWorldV26MineralVeins.planSurfaceOrePiles(
            model,
            seed,
            splitX + 1,
            maxX,
            minZ,
            maxZ,
            pileSplit::add
        );
        if (!allPiles.equals(pileSplit)) {
            throw new AssertionError(
                "surface ore pile changes across region split"
            );
        }

        Set<GeoWorldMiningDistrictPlanner.Placement> whole =
            new HashSet<>();
        planner.plan(
            model, seed,
            probeX - radius, probeX + radius - 1,
            probeZ - 16, probeZ + 15,
            whole::add
        );
        Set<GeoWorldMiningDistrictPlanner.Placement> split =
            new HashSet<>();
        planner.plan(
            model, seed,
            probeX - radius, probeX - 1,
            probeZ - 16, probeZ + 15,
            split::add
        );
        planner.plan(
            model, seed,
            probeX, probeX + radius - 1,
            probeZ - 16, probeZ + 15,
            split::add
        );
        if (!whole.equals(split)) {
            throw new AssertionError(
                "mining district geometry changes across chunk/region split"
            );
        }

        List<GeoWorldMiningDistrictPlanner.Lode> lodes =
            new ArrayList<>();
        planner.planLodes(
            model, seed,
            (int)Math.floor(probe.centerX() - probe.radius()),
            (int)Math.ceil(probe.centerX() + probe.radius()),
            (int)Math.floor(probe.centerZ() - probe.radius()),
            (int)Math.ceil(probe.centerZ() + probe.radius()),
            lodes::add
        );
        long probeLodes = lodes.stream()
            .filter(lode -> lode.districtId() == probe.id())
            .count();
        double longestPrimary = lodes.stream()
            .filter(lode ->
                lode.districtId() == probe.id() &&
                !lode.companion()
            )
            .mapToDouble(GeoWorldMiningDistrictPlanner.Lode::length)
            .max()
            .orElse(0.0);
        boolean hasCompanion = lodes.stream().anyMatch(
            lode ->
                lode.districtId() == probe.id() &&
                lode.companion()
        );
        if (
            probeLodes < 46 ||
            longestPrimary < 24.0 ||
            longestPrimary > 80.0 ||
            !hasCompanion
        ) {
            throw new AssertionError(
                "district must be rich without giant single lodes: lodes=" +
                probeLodes +
                " longestPrimary=" + longestPrimary +
                " companion=" + hasCompanion
            );
        }

        double oreEdge =
            GeoWorldMiningDistrictPlanner.districtOreFractionTarget(1.0);
        double oreCore =
            GeoWorldMiningDistrictPlanner.districtOreFractionTarget(0.0);
        double ratioEpsilon = 1.0e-9;
        if (
            oreEdge < 0.40 - ratioEpsilon ||
            oreCore > 0.70 + ratioEpsilon ||
            1.0 - oreCore < 0.30 - ratioEpsilon ||
            1.0 - oreEdge > 0.60 + ratioEpsilon
        ) {
            throw new AssertionError(
                "district ore/rock ratio regression: ore=" +
                oreEdge + ".." + oreCore +
                " rock=" + (1.0 - oreCore) + ".." +
                (1.0 - oreEdge)
            );
        }
        if (whole.size() < 12000) {
            throw new AssertionError(
                "district still behaves like sparse veins instead of a " +
                "continuous mineralized mass: placements=" + whole.size()
            );
        }
        Set<GeoWorldMiningDistrictPlanner.Placement> probeMass =
            whole.stream()
                .filter(p -> p.districtId() == probe.id())
                .collect(java.util.stream.Collectors.toSet());
        int minPlacementY = probeMass.stream()
            .mapToInt(GeoWorldMiningDistrictPlanner.Placement::y)
            .min()
            .orElseThrow();
        int maxPlacementY = probeMass.stream()
            .mapToInt(GeoWorldMiningDistrictPlanner.Placement::y)
            .max()
            .orElseThrow();
        int verticalSpan = maxPlacementY - minPlacementY + 1;
        if (verticalSpan < heightMapping.height() - 24) {
            throw new AssertionError(
                "mining district must span the full underground chunk " +
                "column vertically: span=" + verticalSpan +
                " expectedNear=" + heightMapping.height()
            );
        }

        Set<Long> occupiedChunks = new HashSet<>();
        Set<Long> occupiedColumns = new HashSet<>();
        for (var placement : probeMass) {
            long chunkKey =
                ((long)Math.floorDiv(placement.x(), 16) << 32) ^
                (Math.floorDiv(placement.z(), 16) & 0xffffffffL);
            occupiedChunks.add(chunkKey);
            long columnKey =
                ((long)placement.x() << 32) ^
                (placement.z() & 0xffffffffL);
            occupiedColumns.add(columnKey);
        }
        if (occupiedChunks.size() < 8) {
            throw new AssertionError(
                "mining district must cover multiple horizontal chunks: " +
                occupiedChunks.size()
            );
        }

        double realizedOreShare =
            occupiedColumns.isEmpty()
                ? 0.0
                : (double)probeMass.size() /
                  ((double)occupiedColumns.size() * verticalSpan);
        if (realizedOreShare < 0.40 || realizedOreShare > 0.70) {
            throw new AssertionError(
                "realized mining-district ore share out of range: " +
                realizedOreShare +
                " placements=" + probeMass.size() +
                " columns=" + occupiedColumns.size() +
                " height=" + verticalSpan
            );
        }

        int sliceY = (minPlacementY + maxPlacementY) / 2;
        Set<Long> slice = new HashSet<>();
        for (var placement : probeMass) {
            if (placement.y() != sliceY) continue;
            long key =
                ((long)placement.x() << 32) ^
                (placement.z() & 0xffffffffL);
            slice.add(key);
        }
        int transitions = 0;
        int alignedToThree = 0;
        int alignedToNine = 0;
        for (int z = probeZ - 16; z <= probeZ + 15; z++) {
            boolean previous = false;
            boolean initialized = false;
            for (int x = probeX - radius; x < probeX + radius; x++) {
                long key =
                    ((long)x << 32) ^ (z & 0xffffffffL);
                boolean current = slice.contains(key);
                if (initialized && current != previous) {
                    transitions++;
                    if (Math.floorMod(x, 3) == 0) alignedToThree++;
                    if (Math.floorMod(x, 9) == 0) alignedToNine++;
                }
                previous = current;
                initialized = true;
            }
        }
        if (
            transitions >= 40 &&
            (
                (double)alignedToThree / transitions > 0.55 ||
                (double)alignedToNine / transitions > 0.30
            )
        ) {
            throw new AssertionError(
                "mining district texture still shows grid periodicity: " +
                "transitions=" + transitions +
                " mod3=" + alignedToThree +
                " mod9=" + alignedToNine
            );
        }

        long companionPlacements = probeMass.stream()
            .filter(GeoWorldMiningDistrictPlanner.Placement::companion)
            .count();
        double companionShare =
            probeMass.isEmpty()
                ? 0.0
                : (double)companionPlacements / probeMass.size();
        if (companionShare < 0.07 || companionShare > 0.30) {
            throw new AssertionError(
                "companion ore must remain a minority inside the " +
                "mineralized mass: share=" + companionShare
            );
        }

        System.out.printf(
            "GeoWorld mining districts PASS seed=%d districts=%d " +
            "nominalCoverage=%.3f probeRadius=%.1f lodes=%d " +
            "longestPrimary=%.1f massPlacements=%d companionShare=%.3f " +
            "verticalSpan=%d chunks=%d oreShare=%.3f transitions=%d " +
            "grid3=%d grid9=%d surfacePileColumns=%d chunkSplit=PASS%n",
            seed,
            districts.size(),
            nominalCoverage,
            probe.radius(),
            probeLodes,
            longestPrimary,
            whole.size(),
            companionShare,
            verticalSpan,
            occupiedChunks.size(),
            realizedOreShare,
            transitions,
            alignedToThree,
            alignedToNine,
            pileWhole.size()
        );
    }
}
