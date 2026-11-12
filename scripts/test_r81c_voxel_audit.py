#!/usr/bin/env python3
"""Regression tests for real Anvil river-water / frozen-ocean levels."""
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from r81c_voxel_audit import water_top_y, surface_top_y, ordered_rises


class ActualWaterLevelTests(unittest.TestCase):
    def row(self, x, top_y, top_block, y62=None, y61=None, y60=None, **extra):
        return {
            "status": "full", "x": x, "z": 10432,
            "topY": top_y, "topBlock": top_block,
            "seaY62Block": y62, "seaY61Block": y61, "seaY60Block": y60,
            "verticalTop": [
                [top_y, top_block],
                [top_y - 1, y61 if top_y == 62 else "minecraft:water"],
            ],
            **extra,
        }

    def test_real_open_water_increases_downstream(self):
        old = self.row(0, 61, "minecraft:water")
        sea = self.row(1, 62, "minecraft:water", "minecraft:water", "minecraft:water")
        self.assertEqual(surface_top_y(old), 61)
        self.assertEqual(surface_top_y(sea), 62)
        self.assertEqual(ordered_rises([old, sea])[0]["riseBlocks"], 1)

    def test_ice_cap_and_open_sea_water_have_same_surface(self):
        ice = self.row(0, 62, "minecraft:ice", "minecraft:ice", "minecraft:water")
        open_water = self.row(1, 62, "minecraft:water", "minecraft:water", "minecraft:water")
        self.assertEqual(water_top_y(ice), 61)
        self.assertEqual(water_top_y(open_water), 62)
        self.assertEqual(surface_top_y(ice), 62)
        self.assertEqual(surface_top_y(open_water), 62)
        self.assertFalse(ordered_rises([ice, open_water]))

    def test_iceberg_above_sea_does_not_raise_water_surface(self):
        iceberg = self.row(
            0, 67, "minecraft:blue_ice", "minecraft:water",
            "minecraft:water", "minecraft:water",
        )
        self.assertEqual(surface_top_y(iceberg), 62)

    def test_blue_ice_in_water_column_can_still_be_sea_level(self):
        row = self.row(0, 62, "minecraft:water", "minecraft:water",
                       "minecraft:blue_ice", "minecraft:water")
        self.assertEqual(surface_top_y(row), 62)

    def test_unloaded_chunk_has_no_water_evidence(self):
        row = {"x": 0, "z": 0, "status": "not_generated"}
        self.assertIsNone(surface_top_y(row))
        self.assertIsNone(water_top_y(row))


if __name__ == "__main__":
    unittest.main()
