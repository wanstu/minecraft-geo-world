#!/usr/bin/env python3
import importlib.util
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("v017_r81d_sinkhole_voxel_audit.py")
spec = importlib.util.spec_from_file_location("v017_r81d", MODULE_PATH)
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)


class SinkholeVoxelAuditTest(unittest.TestCase):
    def test_parse_transect_id(self):
        site = mod.parse_site({"id": "R81D-T096-OFF+032", "x": 1, "z": 2})
        self.assertEqual(site["t"], 0.75)
        self.assertEqual(site["offset"], 32)

    def test_deficit_summary(self):
        got = mod.deficit_summary(
            [
                {"riverWaterAboveSubstrate": 17},
                {"riverWaterAboveSubstrate": 27},
                {"riverWaterAboveSubstrate": -4},
            ]
        )
        self.assertEqual(got["count"], 3)
        self.assertEqual(got["positiveCount"], 2)
        self.assertEqual(got["max"], 27)
        self.assertEqual(got["min"], -4)

    def test_candidate_pass_requires_no_positive_bank_deficit(self):
        got = mod.deficit_summary(
            [
                {"riverWaterAboveSubstrate": -4},
                {"riverWaterAboveSubstrate": -7},
                {"riverWaterAboveSubstrate": -5},
            ]
        )
        self.assertEqual(got["positiveCount"], 0)
        self.assertLessEqual(got["max"], 0)


if __name__ == "__main__":
    unittest.main()
