#!/usr/bin/env python3
"""Regression gate for R8.1e new-world, model-only three-seed audit.

No claim of Minecraft voxel generation arises from this file.
"""
import json
import unittest
from pathlib import Path

REPORT=Path(__file__).resolve().parent.parent/"docs/acceptance/r81e_multiseed_model_validation.json"

class R81eMultiSeedModelEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.data=json.loads(REPORT.read_text())

    def test_exact_independent_seed_matrix_and_rule_lock(self):
        d=self.data
        self.assertEqual(d["schemaVersion"],1)
        self.assertEqual(d["evidenceClass"],"GO_MODEL_ONLY_not_real_Minecraft_chunks")
        self.assertEqual(d["rules"],["geoworld-rules-v1.1.24","geoworld-rules-v1.1.25"])
        self.assertEqual([c["seed"] for c in d["cases"]],[123456,20261112,-90210])
        self.assertFalse(d["releaseEligible"])

    def test_river_planning_is_unmodified_by_new_slope(self):
        for c in self.data["cases"]:
            with self.subTest(seed=c["seed"]):
                self.assertTrue(c["riversSame"])
                self.assertGreater(c["riverCount"],800)
                self.assertTrue(c["macroSurfaceSame"])

    def test_every_original_special_feature_is_identical(self):
        for c in self.data["cases"]:
            with self.subTest(seed=c["seed"]):
                self.assertTrue(c["featuresSame"])
                self.assertGreaterEqual(c["featureCount"],17)
                self.assertEqual(len(c["sinkholes"]),4)
                self.assertEqual(c.get("warnings",[]),[])

    def test_all_24_profiles_reduce_steep_grade(self):
        for c in self.data["cases"]:
            for f in c["sinkholes"]:
                self.assertEqual([p["axis"] for p in f["profiles"]],["x","z"])
                for p in f["profiles"]:
                    with self.subTest(seed=c["seed"],site=f["id"],axis=p["axis"]):
                        self.assertGreater(p["oldMaxSlopeProxy"],1.5)
                        self.assertLess(p["newMaxSlopeProxy"],0.6)
                        self.assertLess(p["newMaxSlopeProxy"],p["oldMaxSlopeProxy"])
                        self.assertEqual(p["waterChanged"],0)
                        self.assertEqual(p["groundRaised"],0)

    def test_old_flat_floor_area_is_reduced_at_all_sites(self):
        for c in self.data["cases"]:
            for f in c["sinkholes"]:
                for p in f["profiles"]:
                    with self.subTest(seed=c["seed"],site=f["id"],axis=p["axis"]):
                        self.assertGreaterEqual(p["oldNearFloorCount"],34)
                        self.assertLessEqual(p["newNearFloorCount"],11)
                        self.assertLess(p["newNearFloorCount"],p["oldNearFloorCount"])

if __name__=="__main__":
    unittest.main()
