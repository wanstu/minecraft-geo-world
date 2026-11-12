#!/usr/bin/env python3
"""Freeze v1.1.26 full-build evidence without confusing local builds with CI.

These records preserve verified local facts but never grant release eligibility.
The two binary artifacts are not committed as releases.
"""
import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
EVIDENCE = ROOT / "docs/acceptance/r81h_fabric_full_builds_20261112.json"
MATRIX = ROOT / "docs/acceptance/r81h_current_rules_local_matrix.json"
NEO = ROOT / "docs/acceptance/r81g_current_rules_local_matrix.json"

class R81hFabricCurrentRulesTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.evidence = json.loads(EVIDENCE.read_text())
        cls.matrix = json.loads(MATRIX.read_text())
        cls.neo = json.loads(NEO.read_text())

    def test_new_fabric_builds_are_real_full_builds_and_two_distinct_targets(self):
        e = self.evidence
        self.assertEqual(e["rules"], "geoworld-rules-v1.1.26")
        self.assertEqual(e["localFabricCompleteBuildPasses"], 2)
        items = e["artifacts"]
        self.assertEqual({x["module"] for x in items}, {
            "java/adapters/1.21.1-fabric",
            "java/adapters/26.3-fabric"
        })
        self.assertTrue(all(x["status"] == "LOCAL_CLEAN_CHECK_BUILD_PASS" for x in items))
        self.assertEqual({x["jdk"] for x in items}, {21, 25})
        self.assertEqual({x["actionableTasks"] for x in items}, {17, 41})
        self.assertEqual([x["executedTasks"] for x in items],
                         [x["actionableTasks"] for x in items])
        self.assertEqual({x["duration"] for x in items}, {"1m 38s", "3m 54s"})

    def test_both_v126_artifacts_have_six_native_engines_and_fabric_metadata(self):
        for x in self.evidence["artifacts"]:
            with self.subTest(x=x["module"]):
                record = json.loads((ROOT / x["jarInspectionFile"]).read_text())
                self.assertTrue(record["passed"])
                self.assertTrue(record["allSixNativeEngines"])
                self.assertEqual(record["missingEntries"], [])
                self.assertEqual(record["metadata"], {"id": "geoworld", "version": "0.1.4"})
                self.assertEqual(record["sha256"], x["jarSha256"])
                self.assertEqual(record["bytes"], x["jarBytes"])
                self.assertTrue(x["compiledRulesV126Confirmed"])
                self.assertTrue(x["compiledRulesV125Rejected"])
                self.assertEqual(len(x["compiledGeneratorIdentityClassSha256"]), 64)
                self.assertTrue(x["notOfficialReleasedBinary"])

    def test_full_build_logs_are_frozen_by_sha_and_task_details(self):
        for x in self.evidence["artifacts"]:
            with self.subTest(x=x["module"]):
                self.assertEqual(len(x["rawBuildLogSha256"]), 64)
                self.assertTrue((ROOT / x["buildEvidenceFile"]).is_file())
                data = (ROOT / x["buildEvidenceFile"]).read_text()
                self.assertIn("NOT official CI or release", data)
                self.assertIn("SHA256", data)
                self.assertIn("> Task :check", data)
                self.assertIn("> Task :build", data)
                self.assertIn("BUILD SUCCESSFUL", data)
                self.assertIn(x["rawBuildLogSha256"], data)
                self.assertIn(x["jarSha256"], data)

    def test_latest_matrix_passes_exactly_three_of_nine(self):
        m = self.matrix
        self.assertEqual(m["candidateRules"], "geoworld-rules-v1.1.26")
        self.assertEqual(m["targetCount"], 9)
        targets = m["targets"]
        self.assertEqual(len(targets), 9)
        self.assertEqual(len({t["module"] for t in targets}), 9)
        passed = {t["module"] for t in targets if t["status"] == "CURRENT_V126_LOCAL_FULL_PASS"}
        self.assertEqual(passed, {
            "java/adapters/1.21.1-neoforge",
            "java/adapters/1.21.1-fabric",
            "java/adapters/26.3-fabric"
        })
        self.assertEqual(m["currentRulesLocalCleanCheckBuildPassed"], 3)
        self.assertEqual(m["priorRulesLocalCleanCheckBuildPassedOnly"], 0)
        self.assertEqual(m["notYetCheckedCurrentRules"], 6)

    def test_blocked_and_unchecked_targets_not_marked_pass(self):
        targets = self.matrix["targets"]
        blocked = [x for x in targets if x["status"] == "RESOURCE_BLOCKED"]
        missing = [x for x in targets if x["status"] == "NOT_LOCAL_FULL_CHECKED"]
        self.assertEqual(len(blocked), 1)
        self.assertEqual(blocked[0]["module"], "java/neoforge-mod")
        self.assertEqual(len(missing), 5)
        self.assertEqual({x["minecraft"] for x in missing}, {"26.2", "26.1.2", "1.20.1"})

    def test_historical_neo_pass_and_water_zero_are_preserved(self):
        r = self.neo["currentNeoForge1211"]
        self.assertTrue(r["priorWaterFailure77Resolved"])
        self.assertEqual(r["finalWaterFreefallFaces"], 0)
        self.assertEqual(r["totalActionableTasks"], 45)
        self.assertTrue(r["sixNativeEngines"])

    def test_never_claim_release_or_official_ci(self):
        for d in (self.evidence, self.matrix):
            self.assertFalse(d["releaseEligible"])
        self.assertFalse(self.evidence["mergeDone"])
        self.assertFalse(self.evidence["pushDone"])
        self.assertFalse(self.evidence["publishedJar"])
        self.assertFalse(self.matrix["merged"])
        self.assertFalse(self.matrix["pushed"])
        self.assertFalse(self.matrix["publishedJar"])
        self.assertEqual(self.evidence["officialGithubNineTargetCI"], "NOT_RUN")
        self.assertEqual(self.matrix["officialGithubWindowsNineTargetCI"], "NOT_RUN")
        self.assertEqual(self.matrix["gameplayCameraAcceptance"], "NOT_DONE")

if __name__ == "__main__":
    unittest.main()
