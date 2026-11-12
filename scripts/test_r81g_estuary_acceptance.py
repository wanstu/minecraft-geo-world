#!/usr/bin/env python3
"""R8.1g frozen v1.1.25/v1.1.26 river-to-sea acceptance evidence.

Never replaces saved Minecraft Anvil voxels with an offline prediction.
"""
import json
import unittest
from pathlib import Path

DIR=Path(__file__).resolve().parent.parent/"docs"/"acceptance"
MODEL=DIR/"r81g_model_and_runtime_acceptance.json"
ANVIL=DIR/"r81g_seed123456_estuary_actual_anvil.json"

class R81gVersionedEstuaryTests(unittest.TestCase):
 @classmethod
 def setUpClass(cls):
  cls.m=json.loads(MODEL.read_text())
  cls.a=json.loads(ANVIL.read_text())
 def test_seed_and_versions_lock_old_world(self):
  a,b=self.m["historical"],self.m["candidate"]
  self.assertEqual(self.m["seed"],123456)
  self.assertEqual(a["rules"],"geoworld-rules-v1.1.25")
  self.assertEqual(b["rules"],"geoworld-rules-v1.1.26")
  self.assertEqual(a["coastalBackwaterVersion"],1)
  self.assertEqual(b["coastalBackwaterVersion"],2)
  self.assertNotEqual(a["savedModelSha256"],b["generatorModelSha256"])
 def test_exact_regressed_pair_was_a_real_discontinuity(self):
  a=self.m["historical"]
  b=self.m["candidate"]
  self.assertEqual((a["a"]["x"],a["a"]["z"]),(10814,2467))
  self.assertEqual((a["b"]["x"],a["b"]["z"]),(10815,2467))
  self.assertEqual(a["a"]["waterSurfaceNormalized"],.5)
  self.assertGreater(a["b"]["waterSurfaceNormalized"]-.5,.006)
  self.assertLess(
    abs(b["a"]["waterSurfaceNormalized"]-b["b"]["waterSurfaceNormalized"]),
    .0004)
  self.assertEqual(a["a"]["ground"],b["a"]["ground"])
  self.assertEqual(a["b"]["ground"],b["b"]["ground"])
 def test_no_unplanned_macro_model_or_river_migration(self):
  d=self.m
  self.assertTrue(all(d["persistedMacroModelsIdentical"].values()))
  self.assertEqual(d["identicalRiverCount"],955)
  self.assertEqual(d["identicalSpecialFeaturesCount"],18)
  self.assertEqual(
    d["nonHydrologicFixtureDifferences"]["configFieldsDiffer"],
    ["WorldProfile","SurfaceOrePile","WorldSample"])
  self.assertFalse(d["nonHydrologicFixtureDifferences"]["surfaceOrePileFromHistoricalUserSave"])
  self.assertTrue(d["nonHydrologicFixtureDifferences"]["surfaceOrePileInNewCandidateWorld"])
 def test_go_java_parity_old_and_new_141_points_each(self):
  for name,value in self.m["GoJavaParity"].items():
   with self.subTest(rules=name):
    self.assertEqual(value["points"],141)
    self.assertEqual(value["mismatches"],0)
    self.assertEqual(value["maxDeltaNormalized"],0)
 def test_complete_water_ownership_not_just_local_patch(self):
  d=self.m["fullWaterOwnershipVerification"]
  self.assertTrue(self.m["waterVerifierPass"])
  self.assertEqual(d["chunks"],1183)
  self.assertEqual(d["columns"],302848)
  self.assertEqual(d["waterColumns"],158884)
  for key in ("freeFallWaterFaces","unownedSurfaceWaterSources",
     "unsafeExposedWaterSources","riverWaterBoundaryModelMismatches",
     "riverLocalWaterPeaks","runtimeWaterLost","wetlandPoolLeaks"):
   self.assertEqual(d[key],0,key)
 def test_real_saved_minecraft_has_all_columns_and_no_cliff(self):
  a=self.a
  self.assertTrue(a["preflight"]["pass"])
  self.assertEqual(a["evidenceClass"],"saved_minecraft_neoforge_anvil_full_block_states")
  self.assertEqual(a["seed"],123456)
  self.assertEqual(a["rulesVersion"],"geoworld-rules-v1.1.26")
  s=a["summary"]
  self.assertEqual(s["expectedFullColumns"],1517)
  self.assertEqual(s["actualFullColumns"],1517)
  self.assertEqual(s["missingOrIncompleteColumns"],0)
  self.assertEqual(s["adjacentWetPairs"],2762)
  self.assertEqual(s["wetWaterSurfaceStepAboveOneBlock"],0)
  self.assertEqual(s["waterToLowerDryExposureSignals"],0)
  self.assertEqual(s["originalFailurePairWaterY"],[65,65])
  self.assertTrue(s["originalFailurePairContinuous"])
  self.assertEqual(a["worldModelSha256"],self.m["candidate"]["serverSavedModelSha256"])
 def test_anvil_evidence_not_derived_from_height_only(self):
  rows=self.a["allSampledColumns"]
  self.assertEqual(len(rows),1517)
  self.assertTrue(all(x["status"]=="full" for x in rows))
  wet={(x["x"],x["z"]):x["waterTopY"] for x in rows if x["waterTopY"] is not None}
  pairs=0
  for (x,z),y in wet.items():
   for n in ((x+1,z),(x,z+1)):
    if n in wet:
     pairs+=1
     self.assertLessEqual(abs(y-wet[n]),1,(x,z,y,n,wet[n]))
  self.assertEqual(pairs,self.a["summary"]["adjacentWetPairs"])
 def test_current_rules_matrix_does_not_inherit_old_fabric_passes(self):
  matrix=json.loads((DIR/"r81g_current_rules_local_matrix.json").read_text())
  self.assertEqual(matrix["candidateRules"],"geoworld-rules-v1.1.26")
  self.assertEqual(matrix["targetCount"],9)
  self.assertEqual(matrix["currentRulesLocalCleanCheckBuildPassed"],1)
  self.assertEqual(matrix["priorRulesLocalCleanCheckBuildPassedOnly"],2)
  self.assertFalse(matrix["releaseEligible"])
  self.assertEqual(matrix["officialGithubWindowsNineTargetCI"],"NOT_RUN")
  self.assertEqual(matrix["fabric263V126CompileJava"]["status"],
                   "LOCAL_COMPILE_JAVA_PASS_NOT_FULL_CHECK_BUILD")
  statuses={r["module"]:r["status"] for r in matrix["targets"]}
  self.assertEqual(statuses["java/adapters/1.21.1-neoforge"],"CURRENT_V126_LOCAL_FULL_PASS")
  self.assertEqual(statuses["java/adapters/26.3-fabric"],"PRIOR_V125_FULL_PASS_RECHECK_REQUIRED")
  self.assertEqual(statuses["java/adapters/1.21.1-fabric"],"PRIOR_V125_FULL_PASS_RECHECK_REQUIRED")

 def test_no_release_while_game_client_or_ci_unverified(self):
  self.assertFalse(self.m["releaseEligible"])
  self.assertFalse(self.a["releaseEligible"])
  self.assertEqual(self.m["officialWindowsNineTargetCI"],"NOT_RUN")
  self.assertEqual(self.m["clientCameraAcceptance"],"NOT_DONE")
  self.assertFalse(self.a["gameCameraScreenshot"])

if __name__=="__main__":
 unittest.main()
