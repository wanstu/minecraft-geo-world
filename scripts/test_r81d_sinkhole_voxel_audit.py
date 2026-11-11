#!/usr/bin/env python3
"""R8.1d frozen regression evidence integrity tests (no Minecraft server needed)."""
import json
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).parent))
from r81d_sinkhole_voxel_audit import actual_voxel

EVIDENCE = Path(__file__).resolve().parent.parent / "docs" / "acceptance"


class FakeAnvil:
    def column(self, x, z):
        return {"x":x,"z":z,"status":"not_generated"}

class R81dSinkholeTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.voxels=json.loads((EVIDENCE/"r81d_actual_voxel_v124_comparison.json").read_text())
        cls.model=json.loads((EVIDENCE/"r81d_v124_candidate_summary.json").read_text())

    def test_ungenerated_chunk_never_counts_as_dry_land(self):
        self.assertEqual(actual_voxel(FakeAnvil(),-1704,5416),{
            "x":-1704,"z":5416,"status":"not_generated"
        })

    def test_saved_world_seed_version_and_preflight(self):
        self.assertEqual(self.voxels["seed"],-1818694639628404530)
        self.assertTrue(self.voxels["preflight"]["legacy"]["pass"])
        self.assertTrue(self.voxels["preflight"]["candidate"]["pass"])
        self.assertEqual(self.model["oldRules"],"geoworld-rules-v1.1.23")
        self.assertEqual(self.model["newRules"],"geoworld-rules-v1.1.24")

    def test_only_sinkhole11_and_rules_config_changed(self):
        self.assertEqual(self.model["modifiedFeatureId"],11)
        self.assertEqual(self.model["oldSpecialFeatureCount"],20)
        self.assertEqual(self.model["candidateSpecialFeatureCount"],20)
        self.assertEqual(len(self.model["preservedFeatureIds"]),19)
        self.assertTrue(self.model["identicalWaterbed"])
        self.assertTrue(self.model["identicalMinerals"])
        self.assertEqual(set(self.model["changedPayloadSections"]),{"config","specialFeatures"})
        self.assertEqual([self.model["oldSinkhole11"]["centerX"],self.model["oldSinkhole11"]["centerZ"]],[-1600,5312])
        self.assertEqual([self.model["candidateSinkhole11"]["centerX"],self.model["candidateSinkhole11"]["centerZ"]],[320,6592])

    def test_four_real_river_water_samples_unchanged(self):
        centers=[r for r in self.voxels["samples"] if r["site"]["offset"]==0]
        self.assertEqual(len(centers),4)
        for r in centers:
            self.assertEqual(r["legacy"]["status"],"full")
            self.assertEqual(r["new"]["status"],"full")
            self.assertEqual(r["legacy"]["waterTopY"],96)
            self.assertEqual(r["new"]["waterTopY"],96)
            self.assertEqual(r["waterTopDeltaY"],0)

    def test_twelve_real_right_bank_columns_stay_above_river(self):
        banks=[r for r in self.voxels["samples"] if r["site"]["offset"] in (16,32,48)]
        self.assertEqual(len(banks),12)
        for r in banks:
            self.assertEqual(r["legacy"]["status"],"full")
            self.assertEqual(r["new"]["status"],"full")
            self.assertLess(r["legacy"]["naturalSubstrateY"],96)
            self.assertGreaterEqual(r["new"]["naturalSubstrateY"],96)

    def test_former_pit_floor_restored(self):
        center=next(r for r in self.voxels["samples"]
                    if r["site"]["x"]==-1600 and r["site"]["z"]==5312 and r["site"]["control"])
        self.assertEqual(center["legacy"]["naturalSubstrateY"],69)
        self.assertEqual(center["new"]["naturalSubstrateY"],103)

    def test_unknown_outer_samples_explicit_not_passed(self):
        acceptance=self.voxels["acceptance"]
        self.assertTrue(acceptance["fixedRiverAndBankCasePass"])
        self.assertFalse(acceptance["full32PointCoverage"])
        self.assertEqual(acceptance["untestedPeripheralCoordinates"],[[-1704,5416],[-1750,5359]])
        self.assertEqual(self.voxels["summary"]["actualFullPairs"],30)
        self.assertEqual(self.voxels["summary"]["unknownPairs"],2)
        self.assertFalse(self.voxels["releaseEligible"])


if __name__=="__main__":
    unittest.main()
