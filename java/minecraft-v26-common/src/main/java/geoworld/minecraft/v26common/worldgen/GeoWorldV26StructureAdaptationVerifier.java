package geoworld.minecraft.v26common.worldgen;

/**
 * Regression coverage for the player-facing structure influence control.
 *
 * This verifier intentionally exercises the pure adaptation policy used by the
 * real StructureStart path so a future refactor cannot leave the creation
 * slider persisted but causally disconnected from terrain generation.
 */
public final class GeoWorldV26StructureAdaptationVerifier {
    private GeoWorldV26StructureAdaptationVerifier() {}

    public static void main(String[] args) {
        GeoWorldV26StructureAdaptation.AdaptationPolicy disabled =
            GeoWorldV26StructureAdaptation.policy(0.0);
        GeoWorldV26StructureAdaptation.AdaptationPolicy standard =
            GeoWorldV26StructureAdaptation.policy(1.0);
        GeoWorldV26StructureAdaptation.AdaptationPolicy strong =
            GeoWorldV26StructureAdaptation.policy(2.5);

        require(!disabled.enabled(), "structure influence 0 must disable active terrain concessions");
        require(standard.enabled(), "structure influence 1 must enable default terrain concessions");
        require(strong.enabled(), "structure influence 2.5 must enable terrain concessions");
        require(strong.margin() > standard.margin(), "strong influence must expand the local concession margin");
        require(strong.strength() > standard.strength(), "strong influence must increase local adaptation strength");
        require(strong.maxShift() > standard.maxShift(), "strong influence must permit a larger bounded height shift");

        int natural = 80;
        int structure = 96;
        int disabledTarget = GeoWorldV26StructureAdaptation.adaptedBoundary(
            natural, structure, 1.0, disabled
        );
        int standardTarget = GeoWorldV26StructureAdaptation.adaptedBoundary(
            natural, structure, 1.0, standard
        );
        int strongTarget = GeoWorldV26StructureAdaptation.adaptedBoundary(
            natural, structure, 1.0, strong
        );
        int edgeTarget = GeoWorldV26StructureAdaptation.adaptedBoundary(
            natural, structure, 0.0, strong
        );
        int halfTarget = GeoWorldV26StructureAdaptation.adaptedBoundary(
            natural, structure, 0.5, standard
        );

        require(disabledTarget == natural, "disabled influence changed terrain");
        require(standardTarget > natural, "default influence did not move terrain toward the structure");
        require(strongTarget > standardTarget, "strong influence must produce a stronger terrain concession");
        require(edgeTarget == natural, "adaptation must fade to zero at the concession edge");
        require(halfTarget > natural && halfTarget < standardTarget, "adaptation must fade smoothly with local weight");
        require(
            Math.abs(strongTarget - natural) <= strong.maxShift(),
            "strong influence exceeded its bounded height shift"
        );

        int thinMargin =
            GeoWorldV26StructureAdaptation.beardThinMargin(standard);
        require(
            thinMargin >= 12,
            "BEARD_THIN must keep at least Vanilla's 12-block local kernel"
        );
        int thinLocal = GeoWorldV26StructureAdaptation.beardThinBoundary(
            natural,
            88,
            0.0,
            standard
        );
        int thinOtherPiece = GeoWorldV26StructureAdaptation.beardThinBoundary(
            natural,
            96,
            0.0,
            standard
        );
        int thinFar = GeoWorldV26StructureAdaptation.beardThinBoundary(
            natural,
            96,
            thinMargin,
            standard
        );
        require(
            thinLocal > natural && thinLocal < thinOtherPiece,
            "BEARD_THIN must respond to each piece's local ground height"
        );
        require(
            thinFar == natural,
            "BEARD_THIN must not flatten terrain beyond the local piece kernel"
        );
        int lowerStructure = 64;
        int lowered = GeoWorldV26StructureAdaptation.adaptedBoundary(
            natural, lowerStructure, 1.0, strong
        );
        require(lowered < natural, "adaptation must also lower terrain toward a lower structure");
        require(
            Math.abs(lowered - natural) <= strong.maxShift(),
            "lowering exceeded the bounded height shift"
        );

        System.out.printf(
            "GeoWorld structure adaptation PASS disabled=%d standard=%d strong=%d half=%d lower=%d thin=%d/%d margin=%d margins=%d/%d shifts=%d/%d%n",
            disabledTarget,
            standardTarget,
            strongTarget,
            halfTarget,
            lowered,
            thinLocal,
            thinOtherPiece,
            thinMargin,
            standard.margin(),
            strong.margin(),
            standard.maxShift(),
            strong.maxShift()
        );
    }

    private static void require(boolean condition, String message) {
        if (!condition) {
            throw new IllegalStateException(message);
        }
    }
}
