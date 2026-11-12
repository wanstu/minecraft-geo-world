#!/usr/bin/env python3
"""Regression evidence for two genuinely new R8.1f Minecraft worlds.

These assertions protect measured, saved first-boot evidence. They never
substitute Go model projections for actual full Anvil chunks.
"""
import datetime
import json
from pathlib import Path
import unittest

FILE=Path(__file__).resolve().parent.parent/"docs"/"acceptance"/"r81f_fresh_world_first_boot.json"

class R81fFreshWorldFirstBootEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.data=json.loads(FILE.read_text(encoding="utf-8"))

    def test_exact_two_independent_fresh_world_seeds(self):
        cases=self.data["cases"]
        self.assertEqual([v["seed"] for v in cases],[123456,20261112])
        self.assertEqual([v["world"] for v in cases],["R81fFreshSeed123456","R81fFreshCold20261112"])
        self.assertTrue(all(v["newWorldDirectoryAbsentBeforeStart"] for v in cases))

    def test_real_geoworld_binding_not_vanilla_or_other_version(self):
        for case in self.data["cases"]:
            with self.subTest(seed=case["seed"]):
                checks=case["preflight"]
                self.assertTrue(checks["pass"])
                self.assertEqual(checks["checks"]["generator"],"geoworld:geoworld")
                self.assertEqual(checks["checks"]["worldSeed"],case["seed"])
                self.assertEqual(checks["checks"]["bindingSeed"],str(case["seed"]))
                self.assertEqual(checks["checks"]["bindingRules"],"geoworld-rules-v1.1.25")
                self.assertEqual(case["rulesVersion"],"geoworld-rules-v1.1.25")
                self.assertGreater(checks["checks"]["regionFiles"],0)

    def test_spawn_y_above_actual_full_minecraft_ground(self):
        expected={(123456):(1088,107,448,106),(20261112):(192,104,192,103)}
        for case in self.data["cases"]:
            with self.subTest(seed=case["seed"]):
                x,y,z,ground=expected[case["seed"]]
                self.assertEqual(case["savedSpawn"],{"x":x,"y":y,"z":z})
                self.assertEqual(case["anvilGround"]["chunkStatus"],"full")
                self.assertEqual(case["anvilGround"]["naturalSubstrateY"],ground)
                self.assertEqual(case["anvilGround"]["surfaceWaterTopY"],None)
                self.assertTrue(case["spawnOneAboveGround"])
                self.assertEqual(y,ground+1)
                self.assertGreater(y,-64)

    def test_warm_and_cold_cache_are_distinct_verified_scenarios(self):
        warm,cold=self.data["cases"]
        self.assertTrue(warm["modelCachePresentAtStart"])
        self.assertFalse(cold["modelCachePresentAtStart"])
        self.assertTrue(self.data["coldSeedNotInModelCacheBeforeStart"])
        self.assertEqual(self.data["firstBootWarmCacheSeconds"],16.678)
        self.assertEqual(self.data["firstBootColdCacheSeconds"],45.427)

    def test_console_stage_times_are_monotonic_and_match_done(self):
        for case in self.data["cases"]:
            with self.subTest(seed=case["seed"]):
                t=case["logMarkers"]
                values=[datetime.datetime.strptime(t[k],"%H:%M:%S.%f") for k in (
                    "preparingLevel","preparingStartRegion","done","modelPersisted"
                )]
                self.assertEqual(values,sorted(values))
                reported=case["minecraftDoneSeconds"]
                measured=(values[2]-values[0]).total_seconds()
                self.assertLess(abs(reported-measured),0.2)
                self.assertGreater(measured,10)
                self.assertLess(measured,80)

    def test_evidence_filenames_models_and_no_ci_claim(self):
        for case in self.data["cases"]:
            self.assertEqual(len(case["modelSha256"]),64)
            self.assertEqual(case["gameCameraVisualAcceptance"],"NOT_DONE")
        self.assertTrue(self.data["bothWorldsSuccessfullySavedAndStopped"])
        self.assertFalse(self.data["previous535SecondBootReproduced"])
        self.assertFalse(self.data["performanceRootCauseConfirmed"])
        self.assertFalse(self.data["releaseEligible"])

if __name__=="__main__":
    unittest.main()
