#!/usr/bin/env python3
import importlib.util
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("v017_r82_island_voxel_audit.py")
spec = importlib.util.spec_from_file_location("v017_r82_island_voxel_audit", MODULE_PATH)
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)


class IslandAuditTest(unittest.TestCase):
    def test_substrate_palette_excludes_mushroom_structure_blocks(self):
        self.assertIn("mycelium", mod.SUBSTRATE)
        self.assertIn("sand", mod.SUBSTRATE)
        self.assertNotIn("brown_mushroom_block", mod.SUBSTRATE)
        self.assertNotIn("red_mushroom_block", mod.SUBSTRATE)

    def test_fixed_transects_use_four_block_spacing(self):
        self.assertEqual(mod.STEP, 4)
        self.assertEqual(mod.Z, -2880)
        self.assertEqual(mod.TRANSECTS[0], ("west", 16232, 16376))
        self.assertEqual(mod.TRANSECTS[1], ("east", 16520, 16664))


if __name__ == "__main__":
    unittest.main()
