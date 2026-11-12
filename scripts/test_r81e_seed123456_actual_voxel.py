#!/usr/bin/env python3
"""Frozen second-seed R8.1e real-Minecraft Anvil acceptance guardrail."""
import json
from pathlib import Path
import unittest

EVIDENCE = Path(__file__).resolve().parent.parent / "docs" / "acceptance"

class R81eSeed123456ActualVoxelTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.v = json.loads((EVIDENCE / "r81e_seed123456_actual_voxels.json").read_text())
        cls.multi = json.loads((EVIDENCE / "r81e_multiseed_model_validation.json").read_text())

    def test_not_a_simulated_or_vanilla_world(self):
        self.assertEqual(self.v["evidenceClass"],"saved_minecraft_full_anvil_blockstate")
        self.assertTrue(self.v["preflight"]["pass"])
        self.assertEqual(self.v["preflight"]["checks"]["generator"],"geoworld:geoworld")
        self.assertEqual(self.v["seed"],123456)
        self.assertEqual(self.v["rulesVersion"],"geoworld-rules-v1.1.25")
        self.assertEqual(self.v["preflight"]["checks"]["bindingRules"],self.v["rulesVersion"])
        self.assertEqual(self.v["preflight"]["checks"]["worldSeed"],self.v["seed"])

    def test_matches_predeclared_second_seed_sinkhole(self):
        self.assertEqual(self.v["site"],{"id":9,"centerX":6080,"centerZ":7744})
        case=next(c for c in self.multi["cases"] if c["seed"]==123456)
        site=next(s for s in case["sinkholes"] if s["id"]==9)
        self.assertEqual((site["X"],site["Z"]),(6080,7744))
        self.assertTrue(case["riversSame"])
        self.assertTrue(case["featuresSame"])

    def test_all_82_columns_are_genuine_full_chunks(self):
        self.assertEqual(self.v["summary"]["columnsExpected"],82)
        self.assertEqual(self.v["summary"]["columnsFull"],82)
        self.assertTrue(self.v["summary"]["crossSectionsFull"])
        for axis in ("eastWest","northSouth"):
            sections=self.v["profiles"][axis]
            self.assertEqual(len(sections),41)
            self.assertEqual(self.v["summary"]["fullCounts"][axis],41)
            self.assertTrue(all(s["status"]=="full" for s in sections))

    def test_center_is_dry_and_slope_is_gradual(self):
        s=self.v["summary"]
        self.assertEqual(s["groundAtCenter"],74)
        self.assertIsNone(s["waterAtCenter"])
        self.assertEqual(s["flatFloorCounts"],{"eastWest":3,"northSouth":3})
        for axis in ("eastWest","northSouth"):
            self.assertLessEqual(s["maxNaturalSlopePer8Blocks"][axis],0.5)

    def test_server_saved_model_agrees_with_seed_plan_and_boot_record(self):
        meta=json.loads((EVIDENCE / "r81e_seed123456_world_boot_evidence.json").read_text())
        self.assertEqual(meta["seed"],123456)
        self.assertTrue(meta["allSinkholeSitesEqualGoMultiSeedModel"])
        self.assertTrue(meta["sameWorldReload"]["savedModelLoaded"])
        self.assertTrue(meta["serverStoppedWithSave"])
        self.assertEqual(meta["persistedRiverCount"],955)
        self.assertEqual(meta["persistedSpecialFeatureCount"],18)
        self.assertEqual(meta["worldModelSha256"],self.v["modelSha256"])
        self.assertEqual(meta["fullRequiredVoxels"],82)
        self.assertEqual(meta["missingRequiredVoxels"],0)
        self.assertGreater(meta["firstBoot"]["reportedSeconds"],500)
        self.assertLess(meta["sameWorldReload"]["reportedDoneSeconds"],30)
        self.assertFalse(meta["releaseEligible"])

    def test_no_unsupported_release_or_visual_claims(self):
        self.assertFalse(self.v["releaseEligible"])
        self.assertEqual(self.v["gameCameraVisualAcceptance"],"NOT_DONE")
        self.assertTrue(self.v["visualization"]["notInGameScreenshot"])
        self.assertLess(self.v["visualization"]["fullPixels"],self.v["visualization"]["totalPixels"])

if __name__=="__main__":
    unittest.main()
