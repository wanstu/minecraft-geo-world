#!/usr/bin/env python3
"""Negative tests: missing Minecraft voxels can never become acceptance evidence."""
import unittest
from pathlib import Path
import sys
sys.path.insert(0,str(Path(__file__).resolve().parent))
from r81e_multiseed_actual_voxel_audit import OFFSETS, section, max_slope

class MissingAnvil:
    def column(self,x,z):
        return {"status":"not_generated","x":x,"z":z}

class R81eMultiSeedVoxelPreflightTests(unittest.TestCase):
    def test_full_cross_section_size_and_center(self):
        self.assertEqual(len(OFFSETS),41)
        self.assertEqual(OFFSETS[0],-160)
        self.assertEqual(OFFSETS[-1],160)
        self.assertIn(0,OFFSETS)

    def test_missing_chunks_are_never_assumed_generated(self):
        west=section(MissingAnvil(),"eastWest",6080,7744)
        north=section(MissingAnvil(),"northSouth",6080,7744)
        self.assertEqual(len(west),41)
        self.assertEqual(len(north),41)
        self.assertTrue(all(x["status"]=="not_generated" for x in west+north))
        self.assertIsNone(max_slope(west))
        self.assertIsNone(max_slope(north))
        self.assertEqual(west[0]["x"],5920)
        self.assertEqual(north[-1]["z"],7904)

    def test_no_proxy_slope_for_unverified_profile(self):
        rows=[{"status":"full","naturalSubstrateY":110},
              {"status":"not_generated"},
              {"status":"full","naturalSubstrateY":95}]
        self.assertIsNone(max_slope(rows))

    def test_real_known_heights_compute_only_adjacent_slopes(self):
        rows=[{"status":"full","naturalSubstrateY":110},
              {"status":"full","naturalSubstrateY":106},
              {"status":"full","naturalSubstrateY":101}]
        self.assertEqual(max_slope(rows),5/8)

if __name__=="__main__":
    unittest.main()
