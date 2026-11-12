#!/usr/bin/env python3
"""R8.1i lock 26.2/26.1.2 v1.1.26 Fabric real full-build results.

These tests verify frozen evidence, not a substitute for live GitHub CI.
"""
import json
import unittest
from pathlib import Path

ROOT=Path(__file__).resolve().parent.parent
E=ROOT/"docs/acceptance/r81i_two_fabric_full_builds.json"
M=ROOT/"docs/acceptance/r81i_current_rules_local_matrix.json"

class R81iFabricResults(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.e=json.loads(E.read_text())
        cls.m=json.loads(M.read_text())

    def test_two_full_builds_on_current_rules(self):
        e=self.e
        self.assertEqual(e["newFullPasses"],2)
        self.assertEqual(e["currentFullBuildPassCountIncludingPrior"],5)
        self.assertEqual(e["rules"],"geoworld-rules-v1.1.26")
        self.assertEqual({x["minecraft"] for x in e["artifacts"]},{"26.2","26.1.2"})
        self.assertEqual({x["status"] for x in e["artifacts"]},{"CURRENT_V126_LOCAL_FULL_PASS"})
        self.assertEqual({x["duration"] for x in e["artifacts"]},{"2m 18s","2m 1s"})
        for x in e["artifacts"]:
            self.assertEqual((x["actionableTasks"],x["executedTasks"],x["upToDateTasks"]),(15,14,1))
            self.assertEqual(x["jdk"],25)
            self.assertEqual(x["loomResolvedVersion"],"1.17.21")
            self.assertEqual(len(x["logSha256"]),64)

    def test_both_jars_have_loader_native_engines_and_compiled_rule(self):
        for x in self.e["artifacts"]:
            with self.subTest(mc=x["minecraft"]):
                j=json.loads((ROOT/x["jarInspectionFile"]).read_text())
                self.assertTrue(j["passed"])
                self.assertTrue(j["allSixNativeEngines"])
                self.assertEqual(j["missingEntries"],[])
                self.assertEqual(j["metadata"],{"id":"geoworld","version":"0.1.4"})
                self.assertEqual(j["sha256"],x["jarSha256"])
                self.assertEqual(j["bytes"],x["jarBytes"])
                self.assertTrue(x["compiledRulesV126"])
                self.assertFalse(x["compiledRulesV125HandshakePresent"])
                self.assertTrue(x["notOfficialRelease"])

    def test_build_logs_have_all_required_real_tasks(self):
        for x in self.e["artifacts"]:
            with self.subTest(mc=x["minecraft"]):
                content=(ROOT/x["buildEvidenceFile"]).read_text()
                self.assertIn("BUILD SUCCESSFUL",content)
                self.assertIn("> Task :compileJava",content)
                self.assertIn("> Task :check",content)
                self.assertIn("> Task :build",content)
                self.assertIn("points=24576",content)
                self.assertIn("passed=34/34",content)
                self.assertIn("Query facade lifecycle",content)
                self.assertIn(x["logSha256"],content)
                self.assertIn(x["jarSha256"],content)

    def test_failed_offline_lack_of_loom_was_not_source_failure(self):
        x=self.e["offlineFirstAttempt"]
        self.assertEqual(x["target"],"26.2 Fabric")
        self.assertEqual(x["result"],"PLUGIN_NOT_CACHED")
        self.assertTrue(x["notACompileFailure"])
        self.assertTrue(self.e["bothOnlineRetryPassed"])

    def test_latest_nine_target_matrix_five_pass_one_block_three_unchecked(self):
        m=self.m
        self.assertEqual(m["candidateRules"],"geoworld-rules-v1.1.26")
        self.assertEqual(m["targetCount"],9)
        self.assertEqual(len(m["targets"]),9)
        self.assertEqual(len({t["module"] for t in m["targets"]}),9)
        statuses=[t["status"] for t in m["targets"]]
        self.assertEqual(statuses.count("CURRENT_V126_LOCAL_FULL_PASS"),5)
        self.assertEqual(statuses.count("RESOURCE_BLOCKED"),1)
        self.assertEqual(statuses.count("NOT_LOCAL_FULL_CHECKED"),3)
        self.assertEqual(m["currentRulesLocalCleanCheckBuildPassed"],5)
        self.assertEqual(m["resourceBlocked"],1)
        self.assertEqual(m["notYetCheckedCurrentRules"],4)
        passed={t["module"] for t in m["targets"] if t["status"]=="CURRENT_V126_LOCAL_FULL_PASS"}
        self.assertEqual(passed,{
            "java/adapters/26.3-fabric",
            "java/adapters/26.2-fabric",
            "java/adapters/26.1.2-fabric",
            "java/adapters/1.21.1-fabric",
            "java/adapters/1.21.1-neoforge",
        })

    def test_26_3_neoforge_is_resource_blocked_not_source_failure(self):
        s={t["module"]:t for t in self.m["targets"]}
        self.assertEqual(s["java/neoforge-mod"]["status"],"RESOURCE_BLOCKED")
        self.assertEqual(s["java/adapters/26.2-neoforge"]["status"],"NOT_LOCAL_FULL_CHECKED")
        self.assertEqual(s["java/adapters/26.1.2-neoforge"]["status"],"NOT_LOCAL_FULL_CHECKED")
        self.assertEqual(s["java/adapters/1.20.1-fabric"]["status"],"NOT_LOCAL_FULL_CHECKED")

    def test_never_claim_release_or_ci(self):
        self.assertEqual(self.e["officialWindowsCi"],"NOT_RUN")
        self.assertEqual(self.m["officialGithubWindowsNineTargetCI"],"NOT_RUN")
        self.assertFalse(self.e["releaseEligible"])
        self.assertFalse(self.m["releaseEligible"])
        self.assertFalse(self.e["pushed"])
        self.assertFalse(self.m["pushed"])
        self.assertFalse(self.e["merged"])
        self.assertFalse(self.m["merged"])
        self.assertFalse(self.e["publishedJar"])
        self.assertFalse(self.m["publishedJar"])
        self.assertEqual(self.m["gameplayCameraAcceptance"],"NOT_DONE")

if __name__=="__main__":
    unittest.main()
