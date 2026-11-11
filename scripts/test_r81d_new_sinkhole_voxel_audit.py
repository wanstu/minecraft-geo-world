#!/usr/bin/env python3
"""Fixed R8.1d new #11 sinkhole actual-block evidence guardrail."""
import json
import unittest
from pathlib import Path

BASE=Path(__file__).resolve().parent.parent/"docs/acceptance"

class SinkholeRelocationVoxelEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.a=json.loads((BASE/"r81d_new_sinkhole_actual_voxels.json").read_text())
        cls.m=json.loads((BASE/"r81d_new_sinkhole_v124_model_audit.json").read_text())

    def test_locked_world_binding(self):
        self.assertTrue(self.a["preflight"]["pass"])
        self.assertEqual(self.a["seed"],-1818694639628404530)
        self.assertEqual(self.a["rulesVersion"],"geoworld-rules-v1.1.24")
        self.assertEqual(self.a["center"],[320,6592])

    def test_no_unknown_on_required_cross_sections(self):
        s=self.a["summary"]
        self.assertEqual(s["profileSamples"],40)
        self.assertEqual(s["fullSamples"],40)
        self.assertTrue(s["allProfileSamplesFull"])

    def test_real_pit_depth_and_dry_floor(self):
        s=self.a["summary"]
        self.assertEqual(s["actualCenterNaturalY"],77)
        self.assertIsNone(s["actualCenterWaterTopY"])
        self.assertEqual((s["westOuterNaturalY"],s["eastOuterNaturalY"]),(115,116))
        self.assertEqual((s["northOuterNaturalY"],s["southOuterNaturalY"]),(115,116))

    def test_flat_floor_shape_is_visible_and_not_approved(self):
        s=self.a["summary"]
        self.assertTrue(s["flatCoreEastWestAllY77"])
        self.assertTrue(s["flatCoreNorthSouthAllY77"])
        self.assertGreaterEqual(
            max(r["risePerHorizontalBlock"] for r in s["steepSlopeSamples"]),1.8
        )
        self.assertEqual(self.a["gameplayVisualAcceptance"],"NOT_DONE")
        self.assertFalse(self.a["releaseEligible"])

    def test_generated_visualization_never_claims_to_be_screenshot(self):
        self.assertFalse(self.a["visualization"]["gameplayScreenshot"])
        self.assertGreater(self.a["visualization"]["gridFullCells"],10000)
        self.assertGreater(self.a["visualization"]["gridMissingCells"],0)

if __name__=="__main__":
    unittest.main()
