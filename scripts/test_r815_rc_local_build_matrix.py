#!/usr/bin/env python3
"""Freeze R8.1 RC local build status without confusing it with a release."""
import json
import unittest
from pathlib import Path

ROOT=Path(__file__).resolve().parent.parent
FILE=ROOT/"docs/acceptance/r815_rc_local_build_matrix_20261112.json"

class R815LocalBuildMatrixEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.matrix=json.loads(FILE.read_text())

    def test_nine_configuration_targets_are_distinct(self):
        data=self.matrix
        self.assertEqual(data["fullReleaseTargetCount"],9)
        self.assertEqual(data["staticPreflightPassCount"],9)
        target=data["targets"]
        self.assertEqual(len(target),9)
        self.assertEqual(len({v["module"] for v in target}),9)

    def test_only_two_fabric_builds_are_marked_full_pass(self):
        full={v["module"] for v in self.matrix["targets"]
              if v["status"]=="LOCAL_CLEAN_CHECK_BUILD_PASS"}
        self.assertEqual(full,{
            "java/adapters/1.21.1-fabric",
            "java/adapters/26.3-fabric"
        })
        self.assertEqual(self.matrix["locallyFullCleanCheckBuildPassCount"],2)

    def test_both_jars_have_metadata_and_native_engines(self):
        passed=[v for v in self.matrix["targets"]
                if v["status"]=="LOCAL_CLEAN_CHECK_BUILD_PASS"]
        for t in passed:
            with self.subTest(module=t["module"]):
                p=ROOT/t["jarEvidence"]
                e=json.loads(p.read_text())
                self.assertTrue(e["passed"])
                self.assertTrue(e["allSixNativeEngines"])
                self.assertEqual(e["metadata"],{"id":"geoworld","version":"0.1.4"})
                self.assertEqual(e["sha256"],t["jarSha256"])
                self.assertEqual(len(t["buildLogSha256"]),64)
                self.assertTrue((ROOT/t["buildEvidenceText"]).is_file())

    def test_recorded_failure_is_not_hidden(self):
        fix=self.matrix["priorFailedCheck"]
        self.assertEqual(fix["task"],"verifyGeoWorldSpecialLandscapes")
        self.assertTrue(fix["testFixed"])
        self.assertTrue(fix["isolatedRetestPassed"])
        self.assertTrue(fix["fullCleanCheckBuildRetestPassed"])
        self.assertFalse(fix["generationAlgorithmModified"])

    def test_legacy_rules_are_not_redefined_by_new_check(self):
        r=self.matrix["legacyModelCheck"]
        self.assertEqual(r["v124SinkholesChecked"],4)
        self.assertTrue(r["oldRulesRemainFlat"])
        self.assertEqual(len(r["oldCenterToInnerDeltasNormalized"]),4)
        self.assertTrue(all(abs(d)<0.001 for d in r["oldCenterToInnerDeltasNormalized"]))

    def test_unsupported_ci_or_public_release_is_never_claimed(self):
        j=self.matrix
        self.assertFalse(j["officialWindowsCi"]["started"])
        self.assertFalse(j["releaseEligible"])
        self.assertFalse(j["merged"])
        self.assertFalse(j["pushed"])
        self.assertFalse(j["publishedJar"])
        notfull=[v for v in j["targets"] if v["status"]!="LOCAL_CLEAN_CHECK_BUILD_PASS"]
        self.assertEqual(len(notfull),7)

if __name__=="__main__":
    unittest.main()
