package geoworld.runtime;

final class GeoWorldCivilizationSuitability {
    record Score(
        double agriculturalSettlementSuitability,
        double portCitySuitability,
        double miningTownSuitability,
        double transportNodeSuitability,
        double nodeSuitability,
        double specializationStrength,
        String nodeClass
    ) {}

    private GeoWorldCivilizationSuitability() {}

    static Score evaluate(
        GeoWorldV1.RuntimeSample sample,
        GeoWorldSettlementSuitability.Score settlement,
        GeoWorldNavigationSuitability.Score navigation,
        GeoWorldMiningDistrict.Score mining
    ) {
        boolean land =
            sample.sourceLand() && "None".equals(sample.waterKind());
        if (!land) {
            return new Score(0, 0, 0, 0, 0, 0, "none");
        }

        double settlementSuitability =
            clamp01(settlement.settlementSuitability());
        double build = clamp01(settlement.buildability());
        double fresh = clamp01(settlement.freshWaterAccess());
        double floodSafety = 1.0 - clamp01(settlement.floodRisk());
        double agriculture = clamp01(settlement.agricultureSuitability());
        double transport = clamp01(settlement.transportAccess());
        double resource = clamp01(settlement.resourceAccess());
        double waterway = clamp01(navigation.waterwayNavigability());
        double port = clamp01(navigation.portSuitability());
        double navigationSafety =
            1.0 - clamp01(navigation.navigationBottleneckRisk());
        double miningSuitability = clamp01(mining.miningSuitability());
        double industrial = clamp01(mining.industrialValue());
        double diversity = clamp01(mining.resourceDiversity());

        double agriculturalSettlement = clamp01(
            0.28 * settlementSuitability +
            0.24 * agriculture +
            0.15 * fresh +
            0.13 * build +
            0.10 * transport +
            0.10 * floodSafety
        );

        double portCity = clamp01(
            0.32 * port +
            0.18 * settlementSuitability +
            0.14 * transport +
            0.10 * build +
            0.09 * fresh +
            0.09 * industrial +
            0.08 * navigationSafety
        );
        if (port < 0.20) {
            portCity *= 0.55 + 1.75 * port;
        }

        double miningTown = clamp01(
            0.31 * miningSuitability +
            0.17 * industrial +
            0.13 * resource +
            0.13 * settlementSuitability +
            0.11 * transport +
            0.08 * build +
            0.07 * diversity
        );

        double transportNode = clamp01(
            0.28 * transport +
            0.18 * settlementSuitability +
            0.15 * build +
            0.13 * port +
            0.11 * waterway +
            0.08 * miningSuitability +
            0.07 * navigationSafety
        );

        double[] values = {
            agriculturalSettlement,
            portCity,
            miningTown,
            transportNode
        };
        String[] labels = {
            "agricultural_settlement",
            "port_city",
            "mining_town",
            "transport_node"
        };
        int bestIndex = 0;
        int secondIndex = 1;
        if (values[secondIndex] > values[bestIndex]) {
            int tmp = bestIndex;
            bestIndex = secondIndex;
            secondIndex = tmp;
        }
        for (int i = 2; i < values.length; i++) {
            if (values[i] > values[bestIndex]) {
                secondIndex = bestIndex;
                bestIndex = i;
            } else if (values[i] > values[secondIndex]) {
                secondIndex = i;
            }
        }

        double best = values[bestIndex];
        double second = values[secondIndex];
        double overall = clamp01(
            0.68 * best +
            0.20 * second +
            0.12 * settlementSuitability
        );
        double specialization =
            clamp01((best - second) / 0.24);

        String nodeClass = labels[bestIndex];
        if (overall < 0.45) {
            nodeClass = "low_potential";
        } else if (
            best >= 0.68 &&
            second >= 0.64 &&
            best - second <= 0.07
        ) {
            nodeClass = "multi_role_hub";
        }

        return new Score(
            agriculturalSettlement,
            portCity,
            miningTown,
            transportNode,
            overall,
            specialization,
            nodeClass
        );
    }

    private static double clamp01(double value) {
        if (!Double.isFinite(value) || value < 0) {
            return 0;
        }
        return Math.min(1, value);
    }
}
