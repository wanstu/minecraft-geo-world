#!/usr/bin/env python3
"""Frozen saved-Minecraft evidence for the R8.1f spawn-height correction.

The source world and the isolated replay world are not needed for these
metadata-level guardrails. This file does not infer untested fresh worlds.
"""
import json
from pathlib import Path
import unittest

EVIDENCE = Path(__file__).resolve().parent.parent / "docs" / "acceptance" / "r81f_spawn_height_replay.json"

class R81fSpawnHeightReplayEvidenceTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.r = json.loads(EVIDENCE.read_text(encoding="utf-8"))

    def test_generator_rules_and_seed_are_locked(self):
        r = self.r
        self.assertEqual(r["seed"],123456)
        self.assertEqual(r["rulesVersion"],"geoworld-rules-v1.1.25")
        self.assertTrue(r["previousWorldPreflightPass"])
        self.assertTrue(r["replayedWorldPreflightPass"])
        for key in ("previousWorldSpawn","replayedWorldSpawn"):
            self.assertEqual(r[key]["generator"],"geoworld:geoworld")
            self.assertEqual(r[key]["worldSeed"],123456)

    def test_actual_invalid_spawn_y_is_preserved_as_failing_baseline(self):
        self.assertEqual(self.r["previousWorldSpawn"]["spawnY"],-64)
        self.assertEqual(
            (self.r["previousWorldSpawn"]["spawnX"],self.r["previousWorldSpawn"]["spawnZ"]),
            (1088,448)
        )

    def test_server_replay_repairs_y_without_changing_x_z(self):
        r=self.r
        old,new=r["previousWorldSpawn"],r["replayedWorldSpawn"]
        self.assertEqual((new["spawnX"],new["spawnZ"]),(old["spawnX"],old["spawnZ"]))
        self.assertTrue(r["spawnXZUnchanged"])
        self.assertEqual(new["spawnY"],107)
        self.assertGreater(new["spawnY"],63)

    def test_actual_saved_anvil_column_supports_surface_height(self):
        r=self.r
        v=r["actualFullAnvilBlock"]
        self.assertEqual(v["chunkStatus"],"full")
        self.assertEqual((v["x"],v["z"]),(1088,448))
        self.assertEqual(v["naturalSubstrateY"],106)
        self.assertEqual(r["replayedWorldSpawn"]["spawnY"],v["naturalSubstrateY"]+1)
        self.assertTrue(r["spawnYCorrectlyAboveNaturalGround"])

    def test_world_model_was_not_recomputed_or_migrated(self):
        r=self.r
        self.assertEqual(r["previousWorldModelSha256"],r["replayedWorldModelSha256"])
        self.assertFalse(r["originalSourceSaveModified"])
        self.assertTrue(r["localServerStartedAndSaved"])

    def test_no_fresh_world_visual_or_ci_claim(self):
        r=self.r
        self.assertFalse(r["firstBootFreshWorldVerified"])
        self.assertFalse(r["fullNineTargetCI"])
        self.assertFalse(r["releaseEligible"])

if __name__=="__main__":
    unittest.main()
