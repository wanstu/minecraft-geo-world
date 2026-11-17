#!/usr/bin/env python3
"""GeoWorld v0.1.7 R8.1d read-only sinkhole/river actual-voxel audit."""
from __future__ import annotations

import argparse
import gzip
import hashlib
import json
import re
from pathlib import Path

from r81c_actual_mouth_voxel_audit import Anvil, WATER_TOP, block_at

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

CASE_RE = re.compile(r"R81D-T(\d+)-OFF([+-])(\d+)$")


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as fp:
        for chunk in iter(lambda: fp.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def parse_site(case: dict) -> dict:
    site = {"id": case["id"], "xSource": case["x"], "zSource": case["z"]}
    match = CASE_RE.fullmatch(case["id"])
    if match:
        site["t"] = int(match.group(1)) / 128.0
        site["offset"] = (1 if match.group(2) == "+" else -1) * int(match.group(3))
    else:
        site["control"] = True
    return site


def natural_substrate(anvil: Anvil, row: dict, x: int, z: int) -> dict | None:
    if row["status"] != "full":
        return None
    chunk = anvil.chunk(x, z)
    if not chunk:
        return None
    sections = {section.get("Y"): section for section in chunk.get("sections", [])}
    max_y = min(180, max(90, row["topY"]))
    for y in range(max_y, 20, -1):
        block = block_at(sections, y, x, z)
        if block and block.removeprefix("minecraft:") in SUBSTRATE:
            return {"y": y, "material": block.removeprefix("minecraft:")}
    return None


def actual_column(anvil: Anvil, x_value: float, z_value: float) -> dict:
    x, z = round(x_value), round(z_value)
    row = anvil.column(x, z)
    out = {"x": x, "z": z, "status": row["status"]}
    if row["status"] != "full":
        return out

    chunk = anvil.chunk(x, z)
    sections = {section.get("Y"): section for section in chunk.get("sections", [])}
    wet_y = [
        y for y in range(20, 181)
        if block_at(sections, y, x, z) in WATER_TOP
    ]
    substrate = natural_substrate(anvil, row, x, z)
    out.update(
        {
            "topY": row["topY"],
            "floorY": row["floorY"],
            "topBlock": row["topBlock"],
            "naturalSubstrateY": None if substrate is None else substrate["y"],
            "naturalSubstrateMaterial": None if substrate is None else substrate["material"],
            "waterTopY": max(wet_y) if wet_y else None,
            "y69": block_at(sections, 69, x, z),
            "y79": block_at(sections, 79, x, z),
            "y96": block_at(sections, 96, x, z),
            "y98": block_at(sections, 98, x, z),
        }
    )
    return out


def deficit_summary(banks: list[dict]) -> dict:
    values = [
        bank["riverWaterAboveSubstrate"]
        for bank in banks
        if bank["riverWaterAboveSubstrate"] is not None
    ]
    return {
        "count": len(values),
        "positiveCount": sum(value > 0 for value in values),
        "max": max(values) if values else None,
        "min": min(values) if values else None,
        "values": values,
    }


def summarize_rows(rows: list[dict]) -> tuple[list[dict], dict]:
    transects = []
    for t in (0.5, 0.625, 0.75, 0.875):
        group = [row for row in rows if row["site"].get("t") == t]
        center = next(row for row in group if row["site"]["offset"] == 0)
        water_y = center["voxel"].get("waterTopY")
        banks = []
        for row in group:
            offset = row["site"]["offset"]
            if offset == 0:
                continue
            substrate_y = row["voxel"].get("naturalSubstrateY")
            banks.append(
                {
                    "offset": offset,
                    "x": row["voxel"]["x"],
                    "z": row["voxel"]["z"],
                    "substrateY": substrate_y,
                    "waterTopY": row["voxel"].get("waterTopY"),
                    "riverWaterAboveSubstrate": (
                        None
                        if water_y is None or substrate_y is None
                        else water_y - substrate_y
                    ),
                }
            )
        transects.append(
            {
                "t": t,
                "center": {
                    "x": center["voxel"]["x"],
                    "z": center["voxel"]["z"],
                    "waterTopY": water_y,
                    "substrateY": center["voxel"].get("naturalSubstrateY"),
                    "status": center["voxel"]["status"],
                },
                "banks": banks,
            }
        )

    controls = {
        row["site"]["id"]: row["voxel"]
        for row in rows
        if row["site"].get("control")
    }
    right = [
        bank
        for transect in transects
        for bank in transect["banks"]
        if bank["offset"] > 0
    ]
    left = [
        bank
        for transect in transects
        for bank in transect["banks"]
        if bank["offset"] < 0
    ]
    summary = {
        "sampleCount": len(rows),
        "fullCount": sum(row["voxel"]["status"] == "full" for row in rows),
        "missingCount": sum(row["voxel"]["status"] != "full" for row in rows),
        "rightBankRiverAboveSubstrate": deficit_summary(right),
        "leftBankRiverAboveSubstrate": deficit_summary(left),
        "sinkholeCenter": controls.get("R81D-PIT-CENTER"),
        "namedBank": controls.get("R81D-BANK"),
        "namedRiver": controls.get("R81D-RIVER"),
        "namedRiverEnd": controls.get("R81D-RIVER-END"),
    }
    return transects, summary


def audit(world: Path, fixture: dict) -> dict:
    model = world / "data/geoworld/world.geoworld"
    region = world / "region/r.-4.10.mca"
    with gzip.open(model, "rt", encoding="utf-8") as fp:
        envelope = json.load(fp)
    manifest = envelope["manifest"]
    config = envelope["payload"]["waterbed"]["config"]

    if fixture["seed"] != manifest["seed"]:
        raise ValueError(f"seed mismatch: fixture={fixture['seed']} model={manifest['seed']}")
    if fixture["rulesVersion"] != manifest["generationRulesVersion"]:
        raise ValueError(
            "rules mismatch: "
            f"fixture={fixture['rulesVersion']} model={manifest['generationRulesVersion']}"
        )

    anvil = Anvil(world / "region")
    rows = []
    for case in fixture["cases"]:
        rows.append(
            {
                "site": parse_site(case),
                "voxel": actual_column(anvil, case["x"], case["z"]),
            }
        )
    transects, summary = summarize_rows(rows)
    if summary["missingCount"] != 0:
        raise ValueError(f"incomplete voxel evidence: {summary['missingCount']} samples missing")

    rules = manifest["generationRulesVersion"]
    if rules == "geoworld-rules-v1.1.24":
        scope = "v0.1.7 Slice B R8.1d river-safe sinkhole siting candidate"
        conclusion_key = "sliceBConclusion"
        conclusion = (
            "passed_actual_voxel_sinkhole_river_conflict_removed"
            if summary["rightBankRiverAboveSubstrate"]["positiveCount"] == 0
            else "failed_actual_voxel_sinkhole_river_conflict_remains"
        )
        notes = [
            "Actual saved Minecraft 1.21.1 NeoForge Anvil evidence from rules v1.1.24 / RiverGeometryVersion 8.",
            "The authoritative river remains fixed; only new sinkhole site selection may change.",
            "Positive riverWaterAboveSubstrate means dry substrate is below the same-transect river water surface.",
        ]
    else:
        scope = "v0.1.6 geomV8 read-only R8.1d sinkhole #11 / river #735 baseline"
        conclusion_key = "sliceAConclusion"
        conclusion = "confirmed_specialfeatures_siting_conflict"
        notes = [
            "Read-only Slice A evidence; no generator change.",
            "not_generated never counts as dry land.",
            "Positive riverWaterAboveSubstrate means the sampled dry substrate is below the same-transect river water surface.",
        ]

    result = {
        "schemaVersion": 1,
        "evidenceClass": "real_saved_full_minecraft_anvil_section_block_states",
        "scope": scope,
        "seed": manifest["seed"],
        "generationRules": rules,
        "riverGeometryVersion": config["RiverGeometryVersion"],
        "worldModelSHA256": sha256(model),
        "regionSHA256": sha256(region),
        "summary": summary,
        "transects": transects,
        "samples": rows,
        "notes": notes,
    }
    result[conclusion_key] = conclusion
    return result


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--world-root", required=True)
    parser.add_argument("--cases", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    fixture = json.loads(Path(args.cases).read_text(encoding="utf-8"))
    result = audit(Path(args.world_root), fixture)
    Path(args.output).write_text(
        json.dumps(result, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )
    print(f"Saved R8.1d actual-voxel audit: {args.output}")


if __name__ == "__main__":
    main()
