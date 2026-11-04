package geoworld.runtime;

import java.util.Arrays;

final class GeoWorldRegionalAdvantage {
    record Score(
        double buildingAdvantage,
        double agricultureAdvantage,
        double portAdvantage,
        double miningAdvantage,
        double transportAdvantage,
        double specializationStrength,
        double advantageOverlapRisk,
        int highAdvantageCount,
        String dominantAdvantage,
        String regionalClass
    ) {}

    private static final double HIGH_ADVANTAGE_THRESHOLD = 0.68;

    private GeoWorldRegionalAdvantage() {}

    static Score evaluate(
        GeoWorldV1.RuntimeSample sample,
        GeoWorldSettlementSuitability.Score settlement,
        GeoWorldNavigationSuitability.Score navigation,
        GeoWorldMiningDistrict.Score mining
    ) {
        boolean land =
            sample.sourceLand() && "None".equals(sample.waterKind());
        if (!land) {
            return new Score(
                0, 0, 0, 0, 0,
                0, 0, 0,
                "none", "none"
            );
        }

        double settlementSuitability =
            clamp01(settlement.settlementSuitability());
        double build = clamp01(settlement.buildability());
        double fresh = clamp01(settlement.freshWaterAccess());
        double floodSafety = 1.0 - clamp01(settlement.floodRisk());
        double agriculture =
            clamp01(settlement.agricultureSuitability());
        double transport = clamp01(settlement.transportAccess());
        double resource = clamp01(settlement.resourceAccess());
        double waterway =
            clamp01(navigation.waterwayNavigability());
        double port = clamp01(navigation.portSuitability());
        double navigationSafety =
            1.0 - clamp01(navigation.navigationBottleneckRisk());
        double miningSuitability =
            clamp01(mining.miningSuitability());
        double industrial = clamp01(mining.industrialValue());
        double diversity = clamp01(mining.resourceDiversity());

        double buildingAdvantage = clamp01(
            0.45 * build +
            0.30 * settlementSuitability +
            0.15 * floodSafety +
            0.10 * transport
        );
        double agricultureAdvantage = clamp01(
            0.50 * agriculture +
            0.18 * fresh +
            0.12 * floodSafety +
            0.10 * settlementSuitability +
            0.10 * transport
        );
        double portAdvantage = clamp01(
            0.65 * port +
            0.12 * waterway +
            0.08 * transport +
            0.07 * settlementSuitability +
            0.08 * navigationSafety
        );
        double miningAdvantage = clamp01(
            0.48 * miningSuitability +
            0.22 * industrial +
            0.15 * resource +
            0.08 * transport +
            0.07 * diversity
        );
        double transportAdvantage = clamp01(
            0.55 * transport +
            0.22 * build +
            0.08 * waterway +
            0.08 * settlementSuitability +
            0.07 * navigationSafety
        );

        double[] values = {
            buildingAdvantage,
            agricultureAdvantage,
            portAdvantage,
            miningAdvantage,
            transportAdvantage
        };
        String[] labels = {
            "building",
            "agriculture",
            "port",
            "mining",
            "transport"
        };
        Integer[] order = {0, 1, 2, 3, 4};
        Arrays.sort(
            order,
            (a, b) -> Double.compare(values[b], values[a])
        );

        double best = values[order[0]];
        double second = values[order[1]];
        double third = values[order[2]];
        int highCount = 0;
        for (double value : values) {
            if (value >= HIGH_ADVANTAGE_THRESHOLD) {
                highCount++;
            }
        }

        double specialization =
            clamp01((best - second) / 0.25);
        double overlap =
            clamp01((third - 0.45) / 0.35);
        String dominant = labels[order[0]];
        String regionalClass = dominant + "_region";

        if (best < 0.45) {
            regionalClass = "low_potential";
        } else if (
            highCount >= 4 &&
            best - second <= 0.05
        ) {
            regionalClass = "multi_advantage_region";
        }

        return new Score(
            buildingAdvantage,
            agricultureAdvantage,
            portAdvantage,
            miningAdvantage,
            transportAdvantage,
            specialization,
            overlap,
            highCount,
            dominant,
            regionalClass
        );
    }

    private static double clamp01(double value) {
        if (!Double.isFinite(value) || value <= 0) {
            return 0;
        }
        return Math.min(1, value);
    }
}
