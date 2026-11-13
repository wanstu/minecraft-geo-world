#!/usr/bin/env python3
"""R8.1p negative release gate: confirmed numerical improvements cannot override a visual FAIL."""
import hashlib
import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
ACCEPT = ROOT / "docs/acceptance"

def read(filename):
    return json.loads((ACCEPT / filename).read_text(encoding="utf-8"))

class R81pGate(unittest.TestCase):
    def test_all_model_pairs_keep_landscape_planners(self):
        for stage in ("a", "b"):
            for seed in (123456, 20261112):
                with self.subTest(stage=stage,seed=seed):
                    j=read(f"r81p_seed{seed}_candidate_{stage}.json")
                    self.assertEqual(j["legacyRules"],"geoworld-rules-v1.1.28")
                    self.assertEqual(j["candidateRules"],"geoworld-rules-v1.1.29")
                    self.assertEqual(j["seed"],seed)
                    self.assertTrue(j["riversIdentical"])
                    self.assertTrue(j["specialFeaturesIdentical"])
                    self.assertTrue(j["macroSurfaceIdentical"])
                    self.assertEqual((j["oldCoastalVersion"],j["newCoastalVersion"]),(2,2))
                    self.assertEqual(len(j["sites"]),4)
                    self.assertEqual(sum(s["comparedPoints"] for s in j["sites"]),1536)
                    for s in j["sites"]:
                        self.assertEqual(s["waterChanged"],0)
                        self.assertEqual(s["groundRaised"],0)
                        self.assertLess(s["maxNormalizedStepYPerBlock"],0.96)
                    self.assertFalse(j["releaseEligible"])

    def test_actual_minecraft_anvil_is_full_and_improved(self):
        old=read("r81o_seed123456_actual_voxels.json")["summary"]
        a=read("r81p_seed123456_actual_voxels_a.json")
        b=read("r81p_seed123456_actual_voxels_b.json")
        for stage,j in (("a",a),("b",b)):
            with self.subTest(stage=stage):
                self.assertEqual(j["evidenceClass"],"saved_minecraft_full_anvil_blockstate")
                self.assertEqual(j["rulesVersion"],"geoworld-rules-v1.1.29")
                self.assertTrue(j["preflight"]["pass"])
                self.assertEqual(j["summary"]["columnsExpected"],54)
                self.assertEqual(j["summary"]["columnsFull"],54)
                self.assertTrue(j["summary"]["crossSectionsFull"])
                self.assertEqual(j["summary"]["groundAtCenter"],74)
                self.assertIsNone(j["summary"]["waterAtCenter"])
                self.assertEqual(j["summary"]["fullCounts"],{"eastWest":27,"northSouth":27})
                self.assertEqual(j["summary"]["flatFloorCounts"],{"eastWest":1,"northSouth":1})
                self.assertFalse(j["releaseEligible"])
        self.assertGreater(old["flatFloorCounts"]["eastWest"],b["summary"]["flatFloorCounts"]["eastWest"])
        for axis in ("eastWest","northSouth"):
            self.assertLess(b["summary"]["maxNaturalSlopePer8Blocks"][axis],
                old["maxNaturalSlopePer8Blocks"][axis])

    def test_actual_client_evidence_and_visual_hold(self):
        j=read("r81p_real_client_visual_evidence.json")
        self.assertEqual(j["candidateRules"],"geoworld-rules-v1.1.29")
        self.assertEqual(j["engine"]["worldModelBinding"],j["candidateRules"])
        self.assertEqual(j["visual"]["candidateAResult"],"FAIL_CIRCULAR_FUNNEL")
        self.assertEqual(j["visual"]["candidateBResult"],"FAIL_RINGS_REMAIN")
        self.assertEqual(j["engine"]["windowsNineTargetCI"],"NOT_RUN v129b")
        self.assertIn("NOT_RUN",j["engine"]["neoForgeCheckBuild"])
        self.assertFalse(j["releaseGate"]["releaseEligible"])
        self.assertFalse(j["releaseGate"]["visualPass"])
        self.assertFalse(j["releaseGate"]["merged"])
        self.assertFalse(j["releaseGate"]["published"])
        self.assertEqual(len(j["visual"]["screenshots"]),4)
        for item in j["visual"]["screenshots"]:
            p=ROOT/item["path"]
            self.assertTrue(p.is_file(),p)
            self.assertGreater(p.stat().st_size,20000)
            self.assertEqual(hashlib.sha256(p.read_bytes()).hexdigest(),item["sha256"])

    def test_handoff_does_not_claim_release(self):
        doc=(ROOT/"docs/REALISTIC_GEOMORPHOLOGY_R81P_SEDIMENT_FANS_20261113.md").read_text()
        self.assertIn("视觉仍不能验收",doc)
        self.assertIn("HOLD",doc)
        self.assertIn("B 未再次执行完整",doc)
        self.assertIn("源/扇",doc.replace("源与沉积","源/扇"))
        self.assertIn("r81p_v129b_client_seed123456_topdown.jpg",doc)

if __name__ == "__main__":
    unittest.main()
