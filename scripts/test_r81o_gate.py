#!/usr/bin/env python3
"""R8.1o evidence integrity and explicit visual release gate."""
import hashlib
import json
from pathlib import Path
import unittest

ROOT=Path(__file__).resolve().parent.parent
ACCEPT=ROOT/"docs/acceptance"
def read(name):
    return json.loads((ACCEPT/name).read_text())

class R81oAcceptance(unittest.TestCase):
    def test_model_identity_across_two_seeds(self):
        for seed in (123456,20261112):
            with self.subTest(seed=seed):
                j=read(f"r81o_seed{seed}_candidate.json")
                self.assertEqual(j["seed"],seed)
                self.assertEqual(j["legacyRules"],"geoworld-rules-v1.1.27")
                self.assertEqual(j["candidateRules"],"geoworld-rules-v1.1.28")
                self.assertTrue(j["riversIdentical"])
                self.assertTrue(j["specialFeaturesIdentical"])
                self.assertTrue(j["macroSurfaceIdentical"])
                self.assertEqual((j["oldCoastalVersion"],j["newCoastalVersion"]),(2,2))
                self.assertEqual(len(j["sites"]),4)
                self.assertEqual(sum(s["comparedPoints"] for s in j["sites"]),1536)
                self.assertGreater(sum(s["changedPoints"] for s in j["sites"]),400)
                for s in j["sites"]:
                    self.assertEqual(s["waterChanged"],0)
                    self.assertEqual(s["groundRaised"],0)
                    self.assertLess(s["maxNormalizedStepYPerBlock"],0.96)
                self.assertFalse(j["releaseEligible"])

    def test_actual_saved_native_anvil(self):
        j=read("r81o_seed123456_actual_voxels.json")
        self.assertEqual(j["rulesVersion"],"geoworld-rules-v1.1.28")
        self.assertEqual(j["evidenceClass"],"saved_minecraft_full_anvil_blockstate")
        self.assertEqual(j["seed"],123456)
        self.assertTrue(j["preflight"]["pass"])
        self.assertEqual(j["summary"]["columnsExpected"],54)
        self.assertEqual(j["summary"]["columnsFull"],54)
        self.assertTrue(j["summary"]["crossSectionsFull"])
        self.assertEqual(j["summary"]["groundAtCenter"],74)
        self.assertIsNone(j["summary"]["waterAtCenter"])
        self.assertEqual(j["summary"]["fullCounts"],{"eastWest":27,"northSouth":27})
        self.assertFalse(j["releaseEligible"])

    def test_camera_photos_and_negative_visual_gate(self):
        j=read("r81o_real_client_visual_evidence.json")
        self.assertEqual(j["candidateRules"],"geoworld-rules-v1.1.28")
        self.assertEqual(j["minecraft"]["rulesBound"],j["candidateRules"])
        self.assertEqual(j["client"]["visualVerdict"],"FAIL")
        self.assertTrue(j["client"]["realMinecraftClient"])
        self.assertTrue(j["minecraft"]["savedNormally"])
        self.assertEqual(j["minecraft"]["anvilColumnsFull"],54)
        self.assertEqual(j["minecraft"]["maxNaturalSlopePer8Blocks"]["candidate_v128"]["eastWest"],0.875)
        self.assertEqual(len(j["client"]["screenshots"]),2)
        for item in j["client"]["screenshots"]:
            path=ROOT/item["path"]
            self.assertTrue(path.is_file())
            self.assertGreater(path.stat().st_size,20000)
            self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(),item["sha256"])
        self.assertFalse(j["releaseGate"]["visualPass"])
        self.assertFalse(j["releaseGate"]["releaseEligible"])
        self.assertFalse(j["releaseGate"]["mergeApproved"])
        self.assertFalse(j["releaseGate"]["published"])

    def test_handoff_states_visual_failure(self):
        p=ROOT/"docs/REALISTIC_GEOMORPHOLOGY_R81O_ASYMMETRIC_SINKHOLE_FLOOR_20261112.md"
        text=p.read_text()
        self.assertIn("视觉 FAIL",text)
        self.assertIn("HOLD",text)
        self.assertIn("v1.1.28 还未执行 Windows 9 目标官方 CI",text)
        self.assertIn("r81o_v128_client_sinkhole_seed123456_topdown.jpg",text)
        self.assertIn("r81o_v128_client_sinkhole_seed123456_oblique.jpg",text)

if __name__=="__main__":
    unittest.main()
