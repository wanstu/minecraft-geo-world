#!/usr/bin/env python3
"""R8.1c: frozen-water mouth geometry and actual substrate regression tests."""
from pathlib import Path
import sys
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).parent))
from r81c_coastal_followup import natural_substrate, surface_rows, verify_mouth


class FakeAnvil:
    def __init__(self, missing=frozenset()):
        self.cache = {}
        self.missing = missing

    def column(self, x, z):
        if (x, z) in self.missing:
            return {"x": x, "z": z, "status": "not_generated"}
        return {
            "x": x, "z": z, "status": "full",
            "topY": 62, "floorY": 57,
            "topBlock": "minecraft:water",
            "seaY62Block": "minecraft:water",
            "seaY61Block": "minecraft:water",
            "verticalTop": [[62, "minecraft:water"], [61, "minecraft:water"]],
        }


class CoastalFollowupTests(unittest.TestCase):
    def test_substrate_does_not_count_tree_leaves_or_corals(self):
        fake = FakeAnvil()
        fake.cache[(0,0)]={"sections":[{"Y":y} for y in range(3,5)]}
        blocks={67:"minecraft:oak_leaves",66:"minecraft:tube_coral",
                65:"minecraft:water",64:"minecraft:sand",63:"minecraft:dirt"}
        with patch("r81c_coastal_followup.block_at", side_effect=lambda sections,y,x,z: blocks.get(y,"minecraft:air")):
            result = natural_substrate(fake,{"status":"full","topY":67},0,0)
        self.assertEqual(result,{"y":64,"material":"sand"})

    def test_substrate_scans_beyond_ocean_floor_decorations(self):
        fake = FakeAnvil()
        fake.cache[(0,0)]={"sections":[{"Y":4}]}
        blocks={70:"minecraft:blue_ice",67:"minecraft:bubble_coral_block",63:"minecraft:grass_block"}
        with patch("r81c_coastal_followup.block_at", side_effect=lambda sec,y,x,z:blocks.get(y,"minecraft:air")):
            bed=natural_substrate(fake,{"status":"full","topY":70},0,0)
        self.assertEqual(bed["y"],63)

    def test_unknown_chunk_not_claimed_as_valid_ground(self):
        self.assertIsNone(natural_substrate(FakeAnvil(),{"status":"not_generated"},0,0))

    def test_exact_mouth_seam_requires_all_nine_real_columns(self):
        points=[{"t":i/128,"x":128+i,"z":128-i} for i in range(129)]
        mouth={"id":535,"oceanMouth":True,"halfWidth":12.3,"points":points}
        result=verify_mouth(FakeAnvil(),mouth)
        self.assertEqual(result["seamFullCount"],9)
        self.assertEqual(result["seamSurfaceY62Count"],9)
        self.assertEqual(result["seamInverseRise"],[])

    def test_missing_seam_chunk_does_not_count_as_water(self):
        points=[{"t":i/128,"x":128+i,"z":128-i} for i in range(129)]
        row=points[83]
        missing={(round(row["x"]),round(row["z"]))}
        mouth={"id":535,"oceanMouth":True,"halfWidth":12.3,"points":points}
        result=verify_mouth(FakeAnvil(missing),mouth)
        self.assertEqual(result["seamFullCount"],8)
        self.assertEqual(result["seamSurfaceY62Count"],8)


if __name__=="__main__":
    unittest.main()
