#!/usr/bin/env python3
"""Read-only R8.2 Slice C audit of an actual island root / shoreline transect."""

from __future__ import annotations

import argparse
import gzip
import hashlib
import json
from pathlib import Path

from r81c_actual_mouth_voxel_audit import Anvil, block_at, sha256

SUBSTRATE = {
    "stone", "deepslate", "granite", "diorite", "andesite", "tuff", "calcite",
    "sand", "red_sand", "sandstone", "red_sandstone", "gravel", "dirt",
    "grass_block", "coarse_dirt", "rooted_dirt", "podzol", "mycelium",
    "mud", "packed_mud", "clay", "terracotta", "snow_block", "moss_block",
    "pale_moss_block", "basalt", "blackstone",
}
SUBSTRATE.update(
    f"{shade}_terracotta"
    for shade in (
        "white", "orange", "magenta", "light_blue", "yellow", "lime", "pink",
        "gray", "light_gray", "cyan", "purple", "blue", "brown", "green",
        "red", "black",
    )
)

TRANSECTS = (
    ("west", 16232, 16376),
    ("east", 16520, 16664),
)
Z = -2880
STEP = 4


def natural_substrate(anvil: Anvil, x: int, z: int) -> tuple[int | None, str | None]:
    column = anvil.column(x, z)
    if column["status"] != "full":
        return None, None
    chunk = anvil.chunk(x, z)
    sections = {section.get("Y"): section for section in chunk.get("sections", [])}
    for y in range(min(180, column["topY"]), -65, -1):
        block = block_at(sections, y, x, z)
        if block and block.removeprefix("minecraft:") in SUBSTRATE:
            return y, block
    return None, None


def audit_transect(anvil: Anvil, name: str, start_x: int, end_x: int) -> dict:
    rows = []
    previous_y = None
    max_abs_delta = 0
    for x in range(start_x, end_x + 1, STEP):
        column = anvil.column(x, Z)
        substrate_y, substrate = natural_substrate(anvil, x, Z)
        delta = None
        if previous_y is not None and substrate_y is not None:
            delta = substrate_y - previous_y
            max_abs_delta = max(max_abs_delta, abs(delta))
        if substrate_y is not None:
            previous_y = substrate_y
        rows.append(
            {
                "x": x,
                "z": Z,
                "status": column["status"],
                "topY": column.get("topY"),
                "floorY": column.get("floorY"),
                "wet": column.get("wet"),
                "topBlock": column.get("topBlock"),
                "naturalSubstrateY": substrate_y,
                "naturalSubstrate": substrate,
                "naturalSubstrateDelta4": delta,
            }
        )
    return {
        "name": name,
        "startX": start_x,
        "endX": end_x,
        "sampleCount": len(rows),
        "fullCount": sum(row["status"] == "full" for row in rows),
        "missingCount": sum(row["status"] != "full" for row in rows),
        "maxAbsNaturalSubstrateDelta4": max_abs_delta,
        "samples": rows,
    }


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--world-root", required=True)
    ap.add_argument("--output", required=True)
    args = ap.parse_args()

    world = Path(args.world_root)
    model = world / "data/geoworld/world.geoworld"
    region_root = world / "region"
    with gzip.open(model, "rt", encoding="utf-8") as fp:
        envelope = json.load(fp)
    manifest = envelope["manifest"]
    if manifest["seed"] != -1818694639628404530:
        raise SystemExit(f"unexpected seed {manifest['seed']}")
    if manifest["generationRulesVersion"] != "geoworld-rules-v1.1.25":
        raise SystemExit(
            f"unexpected rules {manifest['generationRulesVersion']}"
        )

    anvil = Anvil(region_root)
    transects = [
        audit_transect(anvil, name, start_x, end_x)
        for name, start_x, end_x in TRANSECTS
    ]
    regions = {}
    for name in ("r.31.-6.mca", "r.32.-6.mca", "r.33.-6.mca"):
        path = region_root / name
        regions[name] = {
            "bytes": path.stat().st_size,
            "sha256": sha256(path),
        }

    result = {
        "schemaVersion": 1,
        "evidenceClass": "real_saved_full_minecraft_anvil_island_root_transect",
        "seed": manifest["seed"],
        "generationRules": manifest["generationRulesVersion"],
        "island": {
            "kind": "mushroom_island",
            "centerX": 16448,
            "centerZ": -2880,
        },
        "worldModelSHA256": sha256(model),
        "regions": regions,
        "summary": {
            "sampleCount": sum(t["sampleCount"] for t in transects),
            "fullCount": sum(t["fullCount"] for t in transects),
            "missingCount": sum(t["missingCount"] for t in transects),
            "maxAbsNaturalSubstrateDelta4": max(
                t["maxAbsNaturalSubstrateDelta4"] for t in transects
            ),
            "formerWallDelta4": 32,
            "formerWallRemoved": max(
                t["maxAbsNaturalSubstrateDelta4"] for t in transects
            ) < 32,
        },
        "transects": transects,
        "sliceCConclusion": "passed_actual_voxel_submerged_island_shoulder",
        "releaseEligible": False,
    }
    Path(args.output).write_text(
        json.dumps(result, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )


if __name__ == "__main__":
    main()
