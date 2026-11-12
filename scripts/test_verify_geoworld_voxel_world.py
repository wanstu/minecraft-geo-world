#!/usr/bin/env python3
"""Deterministic fixtures for the GeoWorld voxel-save preflight."""
import gzip
import struct
import tempfile
import unittest
from pathlib import Path

from verify_geoworld_voxel_world import inspect


def short_string(value: str) -> bytes:
    encoded = value.encode("utf-8")
    return struct.pack(">H", len(encoded)) + encoded


def item(kind: int, name: str, value: bytes) -> bytes:
    return bytes([kind]) + short_string(name) + value


def compound(*items: bytes) -> bytes:
    return b"".join(items) + b"\0"


def level_dat(generator: str, seed: int) -> bytes:
    generator_tag = item(8, "type", short_string(generator))
    dims = compound(item(10, "minecraft:overworld", compound(item(10, "generator", compound(generator_tag)))))
    worldgen = compound(item(4, "seed", struct.pack(">q", seed)), item(10, "dimensions", dims))
    data = compound(item(10, "WorldGenSettings", worldgen))
    return gzip.compress(b"\x0a" + short_string("") + compound(item(10, "Data", data)))


class VoxelWorldPreflightTests(unittest.TestCase):
    SEED = -1818694639628404530
    RULES = "geoworld-rules-v1.1.23"

    def setUp(self):
        self.root = tempfile.TemporaryDirectory()
        self.addCleanup(self.root.cleanup)
        self.world = Path(self.root.name)
        (self.world / "region").mkdir()
        (self.world / "region" / "r.0.0.mca").write_bytes(b"\0" * 8192)
        (self.world / "data" / "geoworld").mkdir(parents=True)
        self.model = self.world / "data" / "geoworld" / "world.geoworld"
        self.model.write_bytes(b"fixture")
        self.write_level("geoworld:geoworld", self.SEED)
        self.write_binding(self.RULES, self.SEED)

    def write_level(self, generator, seed):
        (self.world / "level.dat").write_bytes(level_dat(generator, seed))

    def write_binding(self, rules, seed):
        (self.world / "data" / "geoworld" / "geoworld.properties").write_text(
            f"format=geoworld-binding\nversion=2\nmodel=world.geoworld\n"
            f"generationRulesVersion={rules}\nseed={seed}\n",
            encoding="utf-8",
        )

    def check(self):
        return inspect(self.world, self.RULES, self.SEED)

    def test_valid_fresh_geoworld_world(self):
        result = self.check()
        self.assertTrue(result["pass"], result["failures"])
        self.assertEqual(result["checks"]["generator"], "geoworld:geoworld")

    def test_vanilla_generator_is_rejected_even_if_binding_exists(self):
        self.write_level("minecraft:noise", self.SEED)
        self.assertIn("wrong generator", str(self.check()["failures"]))

    def test_mismatched_generation_rules_are_rejected(self):
        self.write_binding("geoworld-rules-v1.1.22", self.SEED)
        self.assertIn("rules mismatch", str(self.check()["failures"]))

    def test_mismatched_seed_is_rejected(self):
        self.write_level("geoworld:geoworld", self.SEED + 1)
        self.assertIn("seed mismatch", str(self.check()["failures"]))

    def test_unbound_world_is_rejected(self):
        (self.world / "data" / "geoworld" / "geoworld.properties").unlink()
        self.assertIn("GeoWorld binding missing", str(self.check()["failures"]))

    def test_missing_model_is_rejected(self):
        self.model.unlink()
        self.assertIn("bound model file missing", str(self.check()["failures"]))


if __name__ == "__main__":
    unittest.main()
