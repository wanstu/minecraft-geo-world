#!/usr/bin/env python3
"""Freeze the unresolved NeoForge RC blockers; never label them PASS."""
import json
from pathlib import Path
import unittest

ROOT=Path(__file__).resolve().parent.parent
MATRIX=ROOT/"docs/acceptance/r815_rc_local_build_matrix_20261112.json"
WATER=ROOT/"docs/acceptance/r815_rc_neoforge1211_water_ownership_failure.json"

class R815NeoForgeBlockersTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.matrix=json.loads(MATRIX.read_text())
        cls.water=json.loads(WATER.read_text())

    def test_water_failure_is_real_planned_voxel_evidence_not_anvil(self):
        w=self.water
        self.assertEqual(w["status"],"FAIL")
        self.assertEqual(w["failedTask"],"verifyGeoWorldWaterOwnership")
        self.assertEqual(w["evidenceClass"],
                         "minecraft_java_final_voxel_column_planner_not_saved_anvil")
        self.assertFalse(w["previousSavedMinecraftAnvilCheckedAtThisSite"])
        self.assertEqual(w["seed"],123456)
        self.assertEqual(w["rules"],"geoworld-rules-v1.1.25")

    def test_77_unsafe_water_faces_not_hid_by_other_passes(self):
        v=self.water["verifier"]
        self.assertEqual(v["chunks"],1183)
        self.assertEqual(v["columns"],302848)
        self.assertEqual(v["freeFallWaterFaces"],77)
        self.assertEqual(v["unownedSurfaceWaterSources"],0)
        self.assertEqual(v["unsafeExposedWaterSources"],0)
        self.assertEqual(v["runtimeWaterLost"],0)
        self.assertEqual(v["riverWaterBoundaryModelMismatches"],0)

    def test_identified_abrupt_two_column_river_to_sea_drop(self):
        f=self.water["face"]
        self.assertTrue(f["highSide"]["coastalSeaReceiver"])
        self.assertTrue(f["lowSide"]["coastalSeaReceiver"])
        self.assertEqual(f["highSide"]["waterKind"],"River")
        self.assertEqual(f["lowSide"]["waterKind"],"River")
        self.assertGreater(f["highSide"]["groundHeightNormalized"],.5)
        self.assertLess(f["lowSide"]["groundHeightNormalized"],.5)
        self.assertEqual(f["highSide"]["waterSurfaceY"],66)
        self.assertEqual(f["lowSide"]["waterSurfaceY"],63)
        self.assertEqual(f["faceDropBlocks"],3)

    def test_neighborhood_is_not_misreported_as_same_scan(self):
        n=self.water["local144x144Neighborhood"]
        self.assertEqual(n["highDifferenceBothRiverPairs"],78)
        self.assertEqual(n["waterInvolvedAdjacencies"],12602)
        self.assertNotEqual(n["highDifferenceBothRiverPairs"],
                            self.water["verifier"]["freeFallWaterFaces"])

    def test_neoforge_matrix_is_not_published(self):
        m=self.matrix
        cases={r["module"]:r for r in m["targets"]}
        self.assertEqual(cases["java/adapters/1.21.1-neoforge"]["status"],
                         "LOCAL_CLEAN_CHECK_BUILD_FAILED")
        self.assertEqual(cases["java/neoforge-mod"]["status"],
                         "LOCAL_RESOURCE_BLOCKED")
        self.assertEqual(m["locallyFullCleanCheckBuildPassCount"],2)
        self.assertFalse(m["officialWindowsCi"]["started"])
        self.assertFalse(m["releaseEligible"])

    def test_historical_water_feature_not_automatically_rewritten(self):
        h=self.water["historicalConfig"]
        self.assertEqual(h["coastalBackwaterVersion"],1)
        self.assertEqual(h["currentVersion"],1)
        self.assertFalse(self.water["geometryAlgorithmModifiedThisTurn"])
        self.assertFalse(self.water["releaseEligible"])

if __name__=="__main__":unittest.main()
