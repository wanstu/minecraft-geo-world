#!/usr/bin/env python3
"""Freeze R8.1k nine-target *local* full build evidence.

These assertions deliberately do not mistake local clean check build success
for official GitHub Windows CI, in-game visual approval, or public release.
"""
import json
import unittest
from pathlib import Path

ROOT=Path(__file__).resolve().parent.parent
R=ROOT/"docs/acceptance"

def data(name):
    return json.loads((R/name).read_text())

class R81kLocalBuildAcceptance(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.matrix=data("r81k_current_rules_local_matrix.json")
        cls.summary=data("r81k_all_nine_local_full_builds.json")
        cls.neo=data("r81k_neoforge263_v126_full_build.json")
        cls.fabric=data("r81k_fabric1201_v126_full_build.json")
        cls.webui=data("r81k_neoforge263_webui_timewait_repro.json")

    def test_exactly_nine_unique_current_rules_full_builds(self):
        m=self.matrix
        self.assertEqual(m["candidateRules"],"geoworld-rules-v1.1.26")
        self.assertEqual(m["targetCount"],9)
        self.assertEqual(m["currentRulesLocalCleanCheckBuildPassed"],9)
        self.assertEqual(m["resourceBlocked"],0)
        self.assertEqual(m["notYetCheckedCurrentRules"],0)
        self.assertEqual(m["priorRulesLocalCleanCheckBuildPassedOnly"],0)
        self.assertEqual(len(m["targets"]),9)
        self.assertEqual(len({t["module"] for t in m["targets"]}),9)
        self.assertTrue(all(t["status"]=="CURRENT_V126_LOCAL_FULL_PASS" for t in m["targets"]))
        self.assertEqual({(t["minecraft"],t["loader"]) for t in m["targets"]},{
            ("26.3","fabric"),("26.3","neoforge"),
            ("26.2","fabric"),("26.2","neoforge"),
            ("26.1.2","fabric"),("26.1.2","neoforge"),
            ("1.21.1","fabric"),("1.21.1","neoforge"),("1.20.1","fabric")
        })

    def test_historical_seven_pass_snapshot_not_mutated(self):
        old=data("r81j_current_rules_local_matrix.json")
        self.assertEqual(old["currentRulesLocalCleanCheckBuildPassed"],7)
        self.assertEqual(old["resourceBlocked"],1)
        self.assertEqual(old["notYetCheckedCurrentRules"],2)
        self.assertEqual(self.matrix["currentRulesLocalCleanCheckBuildPassed"],9)
        self.assertEqual(self.summary["localFullCleanCheckBuildPasses"],9)
        self.assertEqual(self.summary["totalTargets"],9)

    def test_new_neoforge_real_full_build(self):
        n=self.neo
        self.assertEqual(n["module"],"java/neoforge-mod")
        self.assertEqual(n["rules"],"geoworld-rules-v1.1.26")
        self.assertEqual(n["status"],"LOCAL_CLEAN_CHECK_BUILD_PASS")
        self.assertEqual(n["cleanCheckBuildDuration"],"3m 25s")
        self.assertEqual((n["tasks"],n["executedTasks"],n["fromCacheTasks"]),(41,40,1))
        self.assertEqual(n["realizedBiomeCoverage"],"56/56")
        self.assertEqual(n["biomeFingerprintPoints"],24576)
        self.assertEqual(n["queryContracts"],"34/34 across 7 persisted models")
        self.assertTrue(n["webUiDynamicPortAndWatchdogPass"])
        self.assertTrue(n["webUiActualHealthHttp200AfterCrashRecovery"])
        self.assertTrue(n["originalMcArtifactsPrepared"])
        self.assertEqual(n["originalMcRecompiledSourceFiles"],7301)
        self.assertFalse(n["memoryIsolation"]["oomKilled"])

    def test_neoforge_hydrology_no_water_exposure_or_freefall(self):
        w=self.neo["waterOwnership"]
        self.assertEqual(w["chunks"],1183)
        self.assertEqual(w["columns"],302848)
        self.assertEqual(w["waterColumns"],158884)
        for k in ("unownedSurfaceWaterSources","unsafeExposedWaterSources","freeFallWaterFaces",
                  "runtimeWaterLost","riverWaterBoundaryModelMismatches",
                  "riverLocalWaterPeaks","wetlandPoolLeaks"):
            self.assertEqual(w[k],0,k)

    def test_neoforge_webui_failure_reproduced_then_actual_health_verified(self):
        x=self.webui
        self.assertEqual(x["status"],"FIXED_TEST_INCORRECT_ALWAYS_REUSE_PORT_ASSERTION")
        self.assertEqual(x["reproductionFromInitialCleanFullBuild"]["failedTask"],"verifyGeoWorldWebUiLaunch")
        self.assertNotEqual(x["reproductionFromInitialCleanFullBuild"]["firstPort"],
                            x["reproductionFromInitialCleanFullBuild"]["recoveredPort"])
        self.assertNotEqual(x["reproductionIsolated"]["firstPort"],
                            x["reproductionIsolated"]["recoveredPort"])
        self.assertEqual(x["tcpTimeWaitObserved"]["socketState"],"TIME_WAIT")
        self.assertEqual(x["tcpTimeWaitObserved"]["javaEquivalentBindReuseAddressFalse"],"EADDRINUSE")
        self.assertEqual(x["tcpTimeWaitObserved"]["socketBindReuseAddressTrue"],"AVAILABLE")
        self.assertFalse(x["productionWebUiRuntimeChanged"])
        self.assertTrue(x["onlyNeoForge263VerifierChanged"])
        self.assertTrue(x["positiveIsolatedTests"]["passed"])
        self.assertTrue(x["positiveFullBuild"]["passed"])
        self.assertEqual(x["positiveIsolatedTests"]["duration"],"23s")
        for v in (x["reproductionFromInitialCleanFullBuild"],
                  x["reproductionIsolated"],
                  x["positiveIsolatedTests"],
                  x["positiveFullBuild"]):
            self.assertEqual(len(v["rawLogSha256"]),64)

    def test_fabric_1201_real_jdk17_full_build(self):
        f=self.fabric
        self.assertEqual(f["module"],"java/adapters/1.20.1-fabric")
        self.assertEqual(f["rules"],"geoworld-rules-v1.1.26")
        self.assertEqual(f["status"],"LOCAL_CLEAN_CHECK_BUILD_PASS")
        self.assertEqual(f["duration"],"5m 29s")
        self.assertEqual(f["jdk"]["major"],17)
        self.assertEqual(f["jdk"]["version"],"17.0.21+9")
        self.assertTrue(f["jdk"]["downloadSha256Validated"])
        self.assertEqual(len(f["jdk"]["tarVendorSha256"]),64)
        self.assertEqual(f["gradleVersion"],"8.10.2")
        self.assertEqual(f["loomVersion"],"1.7.4")
        self.assertEqual(f["actionableTasks"],16)
        self.assertEqual(f["executedTasks"],16)
        self.assertEqual(f["queryContractPasses"],"34/34 across 7 persisted models")
        self.assertTrue(f["queryLifecyclePass"])

    def test_both_jars_verified_with_six_engines_and_current_rules(self):
        for d in (self.neo,self.fabric):
            with self.subTest(target=d["module"]):
                j=json.loads((ROOT/d["jarInspectionFile"]).read_text())
                self.assertTrue(j["passed"])
                self.assertTrue(j["allSixNativeEngines"])
                self.assertEqual(j["missingEntries"],[])
                self.assertEqual(j["sha256"],d["jarSha256"])
                self.assertEqual(j["bytes"],d["jarBytes"])
                self.assertEqual(j["expectedVersion"],"0.1.4")
                if d is self.fabric:
                    self.assertTrue(d["compiledRulesV126"])
                    self.assertFalse(d["compiledRulesV125HandshakePresent"])
                    self.assertEqual(j["metadata"],{"id":"geoworld","version":"0.1.4"})
                else:
                    self.assertTrue(d["compiledRulesV126"])
                    self.assertFalse(d["compiledRulesV125HandshakePresent"])
                    self.assertEqual(j["metadata"]["modVersionLine"],['version="0.1.4"'])

    def test_build_log_evidence_present_and_hashed(self):
        for d in (self.neo,self.fabric):
            with self.subTest(target=d["module"]):
                name=d["evidenceTextFile"]
                text=(ROOT/name).read_text()
                self.assertIn("> Task :compileJava",text)
                self.assertIn("> Task :check",text)
                self.assertIn("> Task :build",text)
                self.assertIn("BUILD SUCCESSFUL",text)
                key="rawLogSha256"
                self.assertEqual(len(d[key]),64)
                self.assertIn(d[key],text)
                self.assertIn(d["jarSha256"],text)

    def test_no_windows_ci_visual_approval_or_release_claim(self):
        self.assertEqual(self.matrix["officialGithubWindowsNineTargetCI"],"NOT_RUN")
        self.assertEqual(self.matrix["gameplayCameraAcceptance"],"NOT_DONE")
        self.assertEqual(self.summary["officialGithubWindowsCI"],"NOT_RUN")
        self.assertEqual(self.summary["gameCameraAcceptance"],"NOT_DONE")
        self.assertFalse(self.matrix["releaseEligible"])
        self.assertFalse(self.summary["releaseEligible"])
        for d in (self.neo,self.fabric):
            self.assertEqual(d["officialGithubCI"],"NOT_RUN")
            self.assertEqual(d["gameCameraAcceptance"],"NOT_DONE")
            self.assertFalse(d["releaseEligible"])
            self.assertFalse(d["publishedJar"])
        for k in ("pushed","merged","publishedJar"):
            self.assertFalse(self.matrix[k])
            self.assertFalse(self.summary[k])

if __name__=="__main__":
    unittest.main()
