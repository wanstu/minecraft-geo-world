#!/usr/bin/env python3
"""Static evidence/gate regression for R8.1n, not a substitute for Minecraft gameplay."""
import hashlib
import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parent.parent
ACCEPT = ROOT / "docs/acceptance"


def read(name):
    return json.loads((ACCEPT / name).read_text(encoding="utf-8"))


class R81nVisualGate(unittest.TestCase):
    def test_three_seed_pairwise_models_preserve_terrain_planners(self):
        for filename, seed in (
            ("r81n_seed123456_candidate.json", 123456),
            ("r81n_seed20261112_candidate.json", 20261112),
            ("r81n_seedNeg90210_candidate.json", -90210),
        ):
            with self.subTest(seed=seed):
                j = read(filename)
                self.assertEqual(j["seed"], seed)
                self.assertEqual(j["legacyRules"], "geoworld-rules-v1.1.26")
                self.assertEqual(j["candidateRules"], "geoworld-rules-v1.1.27")
                self.assertTrue(j["riversIdentical"])
                self.assertTrue(j["specialFeaturesIdentical"])
                self.assertTrue(j["macroSurfaceIdentical"])
                self.assertEqual((j["oldCoastalVersion"], j["newCoastalVersion"]), (2, 2))
                self.assertEqual(len(j["sites"]), 4)
                self.assertEqual(sum(s["comparedPoints"] for s in j["sites"]), 1536)
                self.assertGreater(sum(s["changedPoints"] for s in j["sites"]), 500)
                for site in j["sites"]:
                    self.assertEqual(site["waterChanged"], 0)
                    self.assertEqual(site["groundRaised"], 0)
                    self.assertLess(site["maxNormalizedStepYPerBlock"], 0.95)
                self.assertTrue(j["notRealGameVisual"])
                self.assertFalse(j["releaseEligible"])

    def test_true_anvil_saved_world_requires_full_chunk_samples(self):
        j = read("r81n_seed123456_actual_voxels.json")
        self.assertEqual(j["evidenceClass"], "saved_minecraft_full_anvil_blockstate")
        self.assertEqual(j["seed"], 123456)
        self.assertEqual(j["rulesVersion"], "geoworld-rules-v1.1.27")
        self.assertTrue(j["preflight"]["pass"])
        self.assertEqual(j["summary"]["columnsFull"], 54)
        self.assertEqual(j["summary"]["columnsExpected"], 54)
        self.assertTrue(j["summary"]["crossSectionsFull"])
        self.assertEqual(j["summary"]["groundAtCenter"], 74)
        self.assertIsNone(j["summary"]["waterAtCenter"])
        self.assertEqual(j["summary"]["fullCounts"], {"eastWest": 27, "northSouth": 27})
        for axis in ("eastWest", "northSouth"):
            self.assertLessEqual(j["summary"]["maxNaturalSlopePer8Blocks"][axis], 0.75)
        self.assertFalse(j["releaseEligible"])

    def test_visual_evidence_is_true_client_and_not_released(self):
        j = read("r81n_real_client_visual_evidence.json")
        self.assertEqual(j["candidateRules"], "geoworld-rules-v1.1.27")
        self.assertEqual(j["testWorld"]["worldModelBinding"], j["candidateRules"])
        self.assertEqual(j["visual"]["evidenceClass"], "actual_Minecraft_NeoForge_client_camera")
        self.assertEqual(j["visual"]["verdict"], "IMPROVED_BUT_FAIL")
        self.assertEqual(j["verification"]["localNeoForge1211FullCheckBuild"], "PASS")
        self.assertEqual(j["verification"]["allNineTargetsWindowsCIForV127"], "NOT_RUN")
        self.assertFalse(j["releaseEligible"])
        self.assertFalse(j["merged"])
        self.assertFalse(j["published"])
        self.assertEqual(len(j["visual"]["screenshots"]), 2)
        for photo in j["visual"]["screenshots"]:
            path = ROOT / photo["path"]
            self.assertTrue(path.is_file(), str(path))
            self.assertGreater(path.stat().st_size, 20000)
            self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), photo["sha256"])

    def test_document_does_not_claim_visual_pass(self):
        text = (ROOT / "docs/REALISTIC_GEOMORPHOLOGY_R81N_NATURAL_SINKHOLE_20261112.md").read_text()
        self.assertIn("IMPROVED_BUT_FAIL", text)
        self.assertIn("HOLD", text)
        self.assertIn("尚未运行 GitHub 官方 Windows 九目标 CI", text)
        for phrase in ("r81n_v127_client_sinkhole_seed123456_topdown.jpg",
                       "r81n_v127_client_sinkhole_seed123456_oblique.jpg"):
            self.assertIn(phrase, text)


if __name__ == "__main__":
    unittest.main()
