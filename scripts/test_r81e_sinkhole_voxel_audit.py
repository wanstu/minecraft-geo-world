#!/usr/bin/env python3
"""Frozen real-Minecraft R8.1e sinkhole case, versions are never conflated."""
import json
import unittest
from pathlib import Path

ROOT=Path(__file__).resolve().parent.parent/"docs"/"acceptance"

class R81eSinkholeVoxelEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.old=json.loads((ROOT/"r81d_new_sinkhole_actual_voxels.json").read_text())
        cls.new=json.loads((ROOT/"r81e_new_sinkhole_actual_voxels.json").read_text())
        cls.diff=json.loads((ROOT/"r81e_v125_sinkhole_acceptance.json").read_text())

    def test_different_locked_world_versions_and_binding(self):
        self.assertEqual(self.old["rulesVersion"],"geoworld-rules-v1.1.24")
        self.assertEqual(self.new["rulesVersion"],"geoworld-rules-v1.1.25")
        self.assertTrue(self.old["preflight"]["pass"])
        self.assertTrue(self.new["preflight"]["pass"])
        self.assertEqual(self.old["seed"],self.new["seed"])
        self.assertEqual(self.new["center"],[320,6592])

    def test_all_40_paired_real_minecraft_columns(self):
        self.assertEqual(self.old["summary"]["fullSamples"],40)
        self.assertEqual(self.new["summary"]["fullSamples"],40)
        for axis in ("eastWest","northSouth"):
            aa=self.old["profileSampling"][axis]
            bb=self.new["profileSampling"][axis]
            self.assertEqual([(a["x"],a["z"]) for a in aa],
                             [(b["x"],b["z"]) for b in bb])
            self.assertTrue(all(a["status"]==b["status"]=="full" for a,b in zip(aa,bb)))

    def test_old_flat_floor_removed_with_same_depth(self):
        before=self.old["summary"];after=self.new["summary"]
        self.assertEqual(before["actualCenterNaturalY"],after["actualCenterNaturalY"])
        self.assertEqual(after["actualCenterNaturalY"],77)
        for k in ("westOuterNaturalY","eastOuterNaturalY","northOuterNaturalY","southOuterNaturalY"):
            self.assertEqual(before[k],after[k],k)
        self.assertTrue(before["flatCoreEastWestAllY77"])
        self.assertTrue(before["flatCoreNorthSouthAllY77"])
        self.assertFalse(after["flatCoreEastWestAllY77"])
        self.assertFalse(after["flatCoreNorthSouthAllY77"])
        self.assertIsNone(after["actualCenterWaterTopY"])

    def test_both_real_profiles_are_less_steep(self):
        for axis in ("eastWest","northSouth"):
            p=self.diff["actualAnvilProfiles"][axis]
            self.assertGreaterEqual(p["oldMaxSampledSlope"][0],1.8)
            self.assertLessEqual(p["newMaxSampledSlope"][0],0.5)
            self.assertGreater(p["changedNaturalGroundCount"],10)

    def test_only_rules_profile_changes_in_saved_world_model(self):
        self.assertEqual(self.diff["payloadChangedSections"],["config"])
        self.assertEqual(self.diff["configChangedFields"],["WorldProfile"])
        for attr in ("persistedRiverNetUnchanged",
                     "persistedSpecialFeaturesUnchanged",
                     "persistedMineralsUnchanged",
                     "persistedTerrainUnchanged"):
            self.assertTrue(self.diff[attr],attr)
        self.assertEqual(self.diff["riverCount"],953)
        self.assertEqual(self.diff["oceanMouthRiseSignals"],0)
        self.assertEqual(self.diff["sinkholeRiverOverlapSignals"],0)

    def test_java_go_parity_in_old_and_new_worlds(self):
        for rules in ("old","new"):
            result=self.diff["goJavaParity"][rules]
            self.assertEqual(result["samplePairs"],41)
            self.assertEqual(result["maxGroundDelta"],0)
            self.assertFalse(result["hasAnyMismatch"])

    def test_no_in_game_visual_claim_or_release_eligibility(self):
        self.assertFalse(self.diff["gameplayScreenshotsTaken"])
        self.assertFalse(self.diff["releaseEligible"])
        self.assertEqual(self.new["gameplayVisualAcceptance"],"NOT_DONE")
        self.assertFalse(self.new["visualization"]["gameplayScreenshot"])


if __name__=="__main__":
    unittest.main()
