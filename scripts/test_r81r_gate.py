#!/usr/bin/env python3
"""Real-client evidence and explicit FAIL release gate for R8.1r."""
from pathlib import Path
import hashlib
import json
import unittest

ROOT=Path(__file__).resolve().parent.parent
A=ROOT/"docs/acceptance"
def get(name):return json.loads((A/name).read_text())

class R81rGate(unittest.TestCase):
    def test_model_parity_and_planning_boundary(self):
        for seed in (123456,20261112):
            with self.subTest(seed=seed):
                j=get(f"r81r_seed{seed}_candidate.json")
                self.assertEqual(j["seed"],seed)
                self.assertEqual(j["legacyRules"],"geoworld-rules-v1.1.29")
                self.assertEqual(j["candidateRules"],"geoworld-rules-v1.1.31")
                self.assertEqual(len(j["sites"]),4)
                self.assertEqual(sum(site["comparedPoints"] for site in j["sites"]),1536)
                for key in ("riversIdentical","specialFeaturesIdentical","macroSurfaceIdentical"):
                    self.assertTrue(j[key],key)
                self.assertTrue(all(site["waterChanged"]==0 and site["groundRaised"]==0 for site in j["sites"]))
                self.assertFalse(j["releaseEligible"])

    def test_real_anvil_evidence_and_not_worse_slope(self):
        ev=get("r81r_real_client_visual_evidence.json")
        a=ev["nativeVoxel"]["baselineV129"]
        b=ev["nativeVoxel"]["candidateV131"]
        raw=get("r81r_seed123456_actual_voxels.json")
        self.assertEqual(raw["evidenceClass"],"saved_minecraft_full_anvil_blockstate")
        self.assertTrue(raw["preflight"]["pass"])
        self.assertEqual(raw["rulesVersion"],"geoworld-rules-v1.1.31")
        self.assertEqual(b["columnsFull"],54)
        self.assertEqual(b["columnsExpected"],54)
        self.assertEqual(b["groundAtCenter"],74)
        self.assertIsNone(b["waterAtCenter"])
        self.assertEqual(b["fullCounts"],{"eastWest":27,"northSouth":27})
        self.assertEqual(b["flatFloorCounts"],{"eastWest":1,"northSouth":1})
        for key in ("eastWest","northSouth"):
            self.assertEqual(b["maxNaturalSlopePer8Blocks"][key],a["maxNaturalSlopePer8Blocks"][key])
            self.assertEqual(b["maxNaturalSlopePer8Blocks"][key],0.5)

    def test_real_client_failed_and_release_blocked(self):
        e=get("r81r_real_client_visual_evidence.json")
        self.assertEqual(e["candidateRules"],"geoworld-rules-v1.1.31")
        self.assertEqual(e["visual"]["verdict"],"FAIL")
        self.assertEqual(e["nativeVoxel"]["forceLoadedChunks"],225)
        self.assertEqual(len(e["visual"]["screenshots"]),2)
        for s in e["visual"]["screenshots"]:
            path=ROOT/s["path"]
            self.assertTrue(path.is_file())
            self.assertGreater(path.stat().st_size,20000)
            self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(),s["sha256"])
        self.assertFalse(e["release"]["visualPass"])
        self.assertFalse(e["release"]["releaseEligible"])
        self.assertFalse(e["release"]["merged"])
        self.assertFalse(e["release"]["published"])
        self.assertFalse(e["release"]["crossPlatformReleaseChecksPass"])
        self.assertEqual(e["release"]["officialRC"],"HOLD")

    def test_report_contains_stop_decision(self):
        text=(ROOT/"docs/REALISTIC_GEOMORPHOLOGY_R81R_FLOW_ALIGNED_SINKHOLE_20261113.md").read_text()
        for term in ("视觉 FAIL","44/44","225","v1.1.31","发布不批准","HOLD"):
            self.assertIn(term,text)

if __name__=="__main__":
    unittest.main()
