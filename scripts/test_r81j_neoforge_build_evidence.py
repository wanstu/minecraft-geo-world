#!/usr/bin/env python3
"""R8.1j frozen NeoForge v1.1.26 full-build acceptance evidence.

Do not treat local Gradle build completion or JAR integrity as official
GitHub Windows CI, gameplay camera acceptance, or a public release.
"""
import json
import unittest
from pathlib import Path

ROOT=Path(__file__).resolve().parent.parent
R=ROOT/"docs/acceptance"

class R81jNeoForgeEvidence(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.matrix=json.loads((R/"r81j_current_rules_local_matrix.json").read_text())
  cls.summary=json.loads((R/"r81j_two_neoforge_full_builds.json").read_text())
  cls.details=[
    json.loads((R/"r81j_neoforge262_v126_full_build.json").read_text()),
    json.loads((R/"r81j_neoforge2612_v126_full_build.json").read_text())
  ]

 def test_two_distinct_full_neoforge_builds_pass(self):
  self.assertEqual({d["minecraft"] for d in self.details},{"26.2","26.1.2"})
  self.assertEqual({d["duration"] for d in self.details},{"11m 4s","10m 44s"})
  self.assertTrue(all(d["status"]=="LOCAL_CLEAN_CHECK_BUILD_PASS" for d in self.details))
  self.assertTrue(all(d["rules"]=="geoworld-rules-v1.1.26" for d in self.details))
  for d in self.details:
   with self.subTest(mc=d["minecraft"]):
    self.assertEqual(d["tasks"],30)
    self.assertEqual(d["executedTasks"],29)
    self.assertEqual(d["upToDateTasks"],1)
    self.assertEqual(d["jdk"],25)
    self.assertEqual(d["gradleVersion"],"9.2.1")
    self.assertEqual(len(d["logSha256"]),64)
    self.assertEqual(d["sourceCommit"],
      "1d8dbb41f750f161efdade94c330623303cda921")

 def test_water_verifier_is_full_and_has_zero_defects(self):
  for d in self.details:
   with self.subTest(mc=d["minecraft"]):
    w=d["waterOwnership"]
    self.assertEqual(w["chunks"],1183)
    self.assertEqual(w["columns"],302848)
    self.assertEqual(w["waterColumns"],158884)
    for k in (
      "unownedSurfaceWaterSources","unsafeExposedWaterSources",
      "freeFallWaterFaces","runtimeWaterLost",
      "riverWaterBoundaryModelMismatches","riverLocalWaterPeaks",
      "wetlandPoolLeaks",
    ):
     self.assertEqual(w[k],0,k)

 def test_module_specific_biome_coverage_and_query(self):
  by={d["minecraft"]:d for d in self.details}
  self.assertEqual(by["26.2"]["realizedBiomeCoverage"],"55/55")
  self.assertEqual(by["26.1.2"]["realizedBiomeCoverage"],"54/54")
  for d in self.details:
   self.assertEqual(d["biomeFingerprintPoints"],24576)
   self.assertEqual(d["queryContractPasses"],"34/34 across 7 persisted models")
   self.assertGreater(float(d["decompileDuration"].rstrip("s")),200.)

 def test_jar_inspection_and_compiled_engine_identity(self):
  for d in self.details:
   with self.subTest(mc=d["minecraft"]):
    j=json.loads((ROOT/d["jarInspectionFile"]).read_text())
    self.assertTrue(j["passed"])
    self.assertTrue(j["allSixNativeEngines"])
    self.assertEqual(j["missingEntries"],[])
    self.assertEqual(j["sha256"],d["jarSha256"])
    self.assertEqual(j["bytes"],d["jarBytes"])
    self.assertEqual(j["metadata"]["modVersionLine"],['version="0.1.4"'])
    self.assertEqual(j["expectedVersion"],"0.1.4")
    self.assertTrue(d["compiledEngineRulesV126"])
    self.assertFalse(d["compiledEngineRulesV125"])

 def test_isolated_memory_bounds_no_host_oom(self):
  for d in self.details:
   with self.subTest(mc=d["minecraft"]):
    m=d["memoryIsolation"]
    self.assertIn("systemd",m["mode"])
    self.assertEqual(m["maxBytes"],5100*1024**2)
    self.assertEqual(m["swapMaxBytes"],256*1024**2)
    self.assertFalse(m["oomKilled"])

 def test_log_evidence_text_is_present_and_unchanged(self):
  for d in self.details:
   with self.subTest(mc=d["minecraft"]):
    p=ROOT/d["buildEvidenceTextFile"]
    self.assertTrue(p.exists())
    txt=p.read_text()
    self.assertIn("BUILD SUCCESSFUL",txt)
    self.assertIn("> Task :compileJava",txt)
    self.assertIn("> Task :check",txt)
    self.assertIn("> Task :build",txt)
    self.assertIn("GeoWorld final water ownership",txt)
    self.assertIn(d["logSha256"],txt)
    self.assertIn(d["jarSha256"],txt)

 def test_current_nine_target_matrix_has_exactly_seven_passes(self):
  m=self.matrix
  self.assertEqual(m["candidateRules"],"geoworld-rules-v1.1.26")
  self.assertEqual(m["targetCount"],9)
  self.assertEqual(len(m["targets"]),9)
  self.assertEqual(len({t["module"] for t in m["targets"]}),9)
  self.assertEqual(m["currentRulesLocalCleanCheckBuildPassed"],7)
  passed=[t for t in m["targets"] if t["status"]=="CURRENT_V126_LOCAL_FULL_PASS"]
  blocked=[t for t in m["targets"] if t["status"]=="RESOURCE_BLOCKED"]
  missing=[t for t in m["targets"] if t["status"]=="NOT_LOCAL_FULL_CHECKED"]
  self.assertEqual(len(passed),7)
  self.assertEqual(len(blocked),1)
  self.assertEqual(blocked[0]["module"],"java/neoforge-mod")
  self.assertEqual(len(missing),1)
  self.assertEqual(missing[0]["module"],"java/adapters/1.20.1-fabric")
  self.assertEqual({d["module"] for d in self.details}.issubset(
   {t["module"] for t in passed}),True)

 def test_no_official_ci_or_published_build_is_claimed(self):
  self.assertFalse(self.matrix["releaseEligible"])
  self.assertFalse(self.summary["releaseEligible"])
  self.assertEqual(self.matrix["officialGithubWindowsNineTargetCI"],"NOT_RUN")
  self.assertEqual(self.summary["officialGithubWindowsCi"],"NOT_RUN")
  self.assertEqual(self.summary["gameCameraAcceptance"],"NOT_DONE")
  for d in self.details:
   self.assertEqual(d["officialGithubCI"],"NOT_RUN")
   self.assertEqual(d["gameCameraAcceptance"],"NOT_DONE")
   self.assertFalse(d["releaseEligible"])
  for k in ("pushed","merged","publishedJar"):
   self.assertFalse(self.matrix[k])
   self.assertFalse(self.summary[k])

if __name__=="__main__":
 unittest.main()
