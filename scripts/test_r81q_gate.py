#!/usr/bin/env python3
"""R8.1q negative gate: numeric build success cannot override real-voxel and visual failures."""
import hashlib
import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
ACCEPT = ROOT / "docs/acceptance"

def load(filename):
    return json.loads((ACCEPT / filename).read_text(encoding="utf-8"))

class R81qReleaseGate(unittest.TestCase):
    def test_go_java_parity_and_frozen_planners(self):
        for seed in (123456, 20261112):
            with self.subTest(seed=seed):
                j = load(f"r81q_seed{seed}_candidate.json")
                self.assertEqual(j["seed"],seed)
                self.assertEqual(j["legacyRules"],"geoworld-rules-v1.1.29")
                self.assertEqual(j["candidateRules"],"geoworld-rules-v1.1.30")
                self.assertTrue(j["riversIdentical"])
                self.assertTrue(j["specialFeaturesIdentical"])
                self.assertTrue(j["macroSurfaceIdentical"])
                self.assertEqual((j["oldCoastalVersion"],j["newCoastalVersion"]),(2,2))
                self.assertEqual(len(j["sites"]),4)
                self.assertEqual(sum(s["comparedPoints"] for s in j["sites"]),1536)
                self.assertGreater(sum(s["changedPoints"] for s in j["sites"]),400)
                self.assertTrue(all(s["waterChanged"]==0 for s in j["sites"]))
                self.assertTrue(all(s["groundRaised"]==0 for s in j["sites"]))
                self.assertFalse(j["releaseEligible"])

    def test_real_anvil_is_full_but_slope_regresses(self):
        a=load("r81q_real_client_visual_evidence.json")
        old=a["minecraft"]["previousV129ActualVoxelSummary"]
        new=a["minecraft"]["candidateV130ActualVoxelSummary"]
        raw=load("r81q_seed123456_actual_voxels.json")
        self.assertEqual(raw["evidenceClass"],"saved_minecraft_full_anvil_blockstate")
        self.assertEqual(raw["rulesVersion"],"geoworld-rules-v1.1.30")
        self.assertTrue(raw["preflight"]["pass"])
        self.assertEqual(new["columnsFull"],54)
        self.assertEqual(new["columnsExpected"],54)
        self.assertTrue(new["crossSectionsFull"])
        self.assertEqual(new["groundAtCenter"],74)
        self.assertIsNone(new["waterAtCenter"])
        self.assertEqual(new["fullCounts"],{"eastWest":27,"northSouth":27})
        self.assertEqual(new["flatFloorCounts"],{"eastWest":1,"northSouth":1})
        for direction in ("eastWest","northSouth"):
            self.assertGreater(new["maxNaturalSlopePer8Blocks"][direction],
                               old["maxNaturalSlopePer8Blocks"][direction])
        self.assertFalse(a["releaseGate"]["slopeNotWorse"])
        self.assertFalse(a["releaseGate"]["releaseEligible"])

    def test_real_client_pictures_and_negative_gate(self):
        a=load("r81q_real_client_visual_evidence.json")
        self.assertEqual(a["candidateRules"],"geoworld-rules-v1.1.30")
        self.assertEqual(a["client"]["visualVerdict"],"FAIL")
        self.assertTrue(a["minecraft"]["realClient"])
        self.assertTrue(a["minecraft"]["realServer"])
        self.assertEqual(a["minecraft"]["forceLoadedChunks"],225)
        self.assertEqual(len(a["client"]["screenshots"]),2)
        for shot in a["client"]["screenshots"]:
            p=ROOT/shot["path"]
            self.assertTrue(p.is_file())
            self.assertGreater(p.stat().st_size,20000)
            self.assertEqual(hashlib.sha256(p.read_bytes()).hexdigest(),shot["sha256"])
        for key in ("visualPass","slopeNotWorse","releaseEligible","merged","published"):
            self.assertFalse(a["releaseGate"][key])
        self.assertEqual(a["releaseGate"]["rc"],"HOLD")

    def test_handoff_contains_actual_failed_result(self):
        doc=(ROOT/"docs/REALISTIC_GEOMORPHOLOGY_R81Q_NONCONCENTRIC_SINKHOLE_20261113.md").read_text()
        for expected in ("v1.1.30","0.875","0.625","视觉 FAIL","HOLD",
                         "r81q_v130_client_seed123456_topdown_FAIL.jpg",
                         "不允许次级几何以圆心到点的距离"):
            self.assertIn(expected,doc)

if __name__=="__main__":
    unittest.main()
