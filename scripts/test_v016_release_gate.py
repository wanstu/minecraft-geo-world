#!/usr/bin/env python3
"""GeoWorld v0.1.6 automated release-readiness gate.

This gate intentionally does not replace the manual in-game visual acceptance.
"""
import json
import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
WORKFLOW = (ROOT / ".github/workflows/release.yml").read_text(encoding="utf-8").replace("\r\n", "\n")
PROFILE = (ROOT / "internal/generator/profile.go").read_text(encoding="utf-8")
ENGINE = (ROOT / "java/minecraft-common/src/main/java/geoworld/minecraft/common/engine/GeoWorldEngineBridge.java").read_text(encoding="utf-8")
R81C = json.loads((ROOT / "docs/acceptance/r81c_receiver_model_candidate.json").read_text(encoding="utf-8"))
ANVIL = json.loads((ROOT / "docs/acceptance/r81c_actual_anvil_water_scan.json").read_text(encoding="utf-8"))
READINESS = json.loads((ROOT / "docs/acceptance/v016_release_readiness.json").read_text(encoding="utf-8"))

class V016ReleaseGate(unittest.TestCase):
    def test_rules_identity_is_v123(self):
        self.assertRegex(PROFILE, r'CurrentGenerationRulesVersion\s*=\s*"geoworld-rules-v1\.1\.23"')
        self.assertRegex(ENGINE, r'EXPECTED_GENERATION_RULES_VERSION\s*=\s*\r?\n?\s*"geoworld-rules-v1\.1\.23"')

    def test_r81c_actual_anvil_acceptance(self):
        self.assertEqual(R81C["candidateRules"], "geoworld-rules-v1.1.23")
        self.assertEqual(R81C["minecraftVoxelAcceptance"], "passed_neoforge_1.21.1_full_anvil")
        self.assertFalse(R81C["published"])
        self.assertTrue(R81C["actualWorldBeforeAfterModelBytesIdentical"])
        self.assertEqual(R81C["actualAnvilAcceptance"]["oldWorldNegativeControlDetectedWalls"], 50)
        for key in ("r509", "r915"):
            self.assertEqual(ANVIL[key]["missingColumns"], 0)
            self.assertEqual(ANVIL[key]["waterWallsGt1"], 0)
            self.assertEqual(ANVIL[key]["maxAdjacentWetTopDrop"], 0)

    def test_ci_release_matrix_is_nine_targets(self):
        def matrix(job, next_job):
            section = WORKFLOW.split(f"  {job}:\n", 1)[1].split(f"  {next_job}:\n", 1)[0]
            return re.findall(
                r"\{ minecraft: '([^']+)', loader: ([a-z]+), java: '([0-9]+)', module: '([^']+)' \}",
                section,
            )
        verify = matrix("implemented-adapters", "release-build")
        release = matrix("release-build", "publish")
        self.assertEqual(len(verify), 9)
        self.assertEqual(len(set(verify)), 9)
        self.assertEqual(verify, release)

    def test_release_workflow_injects_tag_version_and_requires_master(self):
        self.assertIn("-Pmod_version=$env:MOD_VERSION", WORKFLOW)
        self.assertIn("Release tag must point to current master.", WORKFLOW)
        self.assertIn("Exactly nine verified Minecraft/loader artifacts are required.", WORKFLOW)

    def test_candidate_artifact_evidence_is_complete(self):
        artifacts = READINESS["candidateArtifacts"]
        self.assertEqual(len(artifacts), 9)
        self.assertEqual(len({(a["minecraft"], a["loader"]) for a in artifacts}), 9)
        for artifact in artifacts:
            self.assertEqual(artifact["version"], "0.1.6")
            self.assertRegex(artifact["sha256"], r"^[0-9a-f]{64}$")
            self.assertGreater(artifact["bytes"], 0)
            self.assertTrue(artifact["sixNativeEnginesVerified"])

    def test_manual_visual_gate_remains_explicitly_pending(self):
        self.assertEqual(READINESS["automatedReleaseReadiness"], "passed")
        self.assertEqual(READINESS["manualVisualAcceptance"], "pending")
        self.assertFalse(READINESS["releaseAllowed"])
        self.assertFalse(READINESS["published"])

if __name__ == "__main__":
    unittest.main()
