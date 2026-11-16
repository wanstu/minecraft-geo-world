#!/usr/bin/env python3
"""GeoWorld v0.1.6 release-readiness gate.

This gate validates the accepted geomV8 candidate state: automated checks,
manual visual acceptance evidence, and the post-v8 nine-target release matrix.
It does not perform merge, tag creation, push, or publication.
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
FRAGMENTATION = json.loads((ROOT / "docs/acceptance/r81c_fragmentation_fix.json").read_text(encoding="utf-8"))
READINESS = json.loads((ROOT / "docs/acceptance/v016_release_readiness.json").read_text(encoding="utf-8"))


class V016ReleaseGate(unittest.TestCase):
    def test_rules_identity_and_geometry_versioning(self):
        self.assertRegex(PROFILE, r'CurrentGenerationRulesVersion\s*=\s*"geoworld-rules-v1\.1\.23"')
        self.assertRegex(ENGINE, r'EXPECTED_GENERATION_RULES_VERSION\s*=\s*\r?\n?\s*"geoworld-rules-v1\.1\.23"')
        self.assertRegex(PROFILE, r'case "geoworld-rules-v1\.1\.23":\s*\n\s*cfg\.Waterbed\.RiverGeometryVersion = 8')
        self.assertRegex(PROFILE, r'case "geoworld-rules-v1\.1\.22":\s*\n\s*cfg\.Waterbed\.RiverGeometryVersion = 7')

    def test_original_r81c_anvil_water_wall_acceptance_remains_recorded(self):
        self.assertEqual(R81C["candidateRules"], "geoworld-rules-v1.1.23")
        self.assertEqual(R81C["minecraftVoxelAcceptance"], "passed_neoforge_1.21.1_full_anvil")
        self.assertFalse(R81C["published"])
        self.assertEqual(R81C["actualAnvilAcceptance"]["oldWorldNegativeControlDetectedWalls"], 50)
        for key in ("r509", "r915"):
            self.assertEqual(ANVIL[key]["missingColumns"], 0)
            self.assertEqual(ANVIL[key]["waterWallsGt1"], 0)
            self.assertEqual(ANVIL[key]["maxAdjacentWetTopDrop"], 0)

    def test_fragmentation_regression_is_zero_in_all_reported_regions(self):
        comparison = FRAGMENTATION["fixedSeedModelComparison"]
        self.assertEqual(FRAGMENTATION["geometryVersion"]["old"], 7)
        self.assertEqual(FRAGMENTATION["geometryVersion"]["new"], 8)
        self.assertTrue(comparison["compatibilityBaselineRestored"])
        self.assertEqual(comparison["oldTotalArtificialEdges"], 2299)
        self.assertEqual(comparison["newTotalArtificialEdges"], 0)
        self.assertEqual(len(comparison["regions"]), 4)
        for region in comparison["regions"]:
            self.assertGreater(region["oldArtificialEdges"], 0)
            self.assertEqual(region["newArtificialEdges"], 0)
            self.assertGreater(region["oldMaxArtificialJump"], 0)
            self.assertEqual(region["newMaxArtificialJump"], 0)

    def test_ci_release_matrix_definition_is_nine_targets(self):
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

    def test_current_1211_neoforge_candidate_is_revalidated(self):
        candidate = READINESS["currentCandidate"]
        evidence = FRAGMENTATION["candidateJar"]
        self.assertEqual(candidate["minecraft"], "1.21.1")
        self.assertEqual(candidate["loader"], "neoforge")
        self.assertEqual(candidate["version"], "0.1.6")
        self.assertEqual(candidate["sha256"], evidence["sha256"])
        self.assertEqual(candidate["bytes"], evidence["bytes"])
        self.assertRegex(candidate["sha256"], r"^[0-9a-f]{64}$")
        self.assertTrue(candidate["sixNativeEnginesVerified"])
        self.assertEqual(candidate["cleanCheckBuild"], "passed")
        self.assertEqual(candidate["fullGoTest"], "passed")
        self.assertEqual(candidate["engineBridgeCacheVerifier"], "passed")
        self.assertEqual(candidate["expectedRiverGeometryVersion"], 8)
        self.assertEqual(candidate["cacheFormatVersion"], "v27")

    def test_stale_model_cache_is_versioned_by_engine_and_geometry(self):
        incident = FRAGMENTATION["staleCacheIncident"]
        self.assertEqual(incident["status"], "confirmed")
        self.assertEqual(incident["saveModel"]["riverGeometryVersion"], 7)
        self.assertEqual(
            incident["saveModel"]["sha256"],
            incident["matchingCache"]["sha256"],
        )
        self.assertEqual(incident["fix"]["cacheFormatVersion"], "v27")
        self.assertEqual(incident["fix"]["expectedRiverGeometryVersion"], 8)
        self.assertIn("embeddedNativeEngineSha256", incident["fix"]["cacheKeyIncludes"])
        self.assertRegex(ENGINE, r'CACHE_FORMAT_VERSION\s*=\s*"v27"')
        self.assertRegex(ENGINE, r'EXPECTED_RIVER_GEOMETRY_VERSION\s*=\s*8')
        self.assertIn("engineFingerprint", ENGINE)
        self.assertIn("model.riverGeometryVersion()", ENGINE)

    def test_release_is_ready_but_not_published(self):
        self.assertEqual(READINESS["automatedReleaseReadiness"], "passed")
        self.assertEqual(READINESS["manualVisualAcceptance"], "passed")
        self.assertEqual(READINESS["releaseMatrix"]["postV8CurrentMatrix"], "passed_9_of_9")
        self.assertEqual(READINESS["releaseMatrix"]["currentValidatedTargets"], 9)
        self.assertTrue(READINESS["releaseMatrix"]["allCandidateJarsSixNativeEnginesVerified"])
        self.assertTrue(READINESS["releaseAllowed"])
        self.assertFalse(READINESS["published"])
        self.assertEqual(READINESS["releaseState"], "ready_for_merge_tag_release_not_executed")
        self.assertEqual(FRAGMENTATION["manualVisualRetest"], "passed_actual_geom_v8")
        self.assertEqual(FRAGMENTATION["fullNineTargetPostV8Matrix"], "passed_9_of_9")
        self.assertTrue(FRAGMENTATION["releaseAllowed"])
        self.assertFalse(FRAGMENTATION["published"])
        visual = READINESS["visualAcceptance"]
        self.assertEqual(visual["result"], "passed")
        self.assertEqual(visual["actualSaveModel"]["riverGeometryVersion"], 8)
        self.assertEqual(len(visual["locations"]), 5)
        self.assertTrue(all(item["result"] == "passed" for item in visual["locations"]))
        artifacts = READINESS["postV8Artifacts"]
        self.assertEqual(len(artifacts), 9)
        self.assertEqual(len({(a["minecraft"], a["loader"]) for a in artifacts}), 9)
        for artifact in artifacts:
            self.assertEqual(artifact["version"], "0.1.6")
            self.assertTrue(artifact["sixNativeEnginesVerified"])
            self.assertGreater(artifact["bytes"], 0)
            self.assertRegex(artifact["sha256"], r"^[0-9a-f]{64}$")


if __name__ == "__main__":
    unittest.main()
