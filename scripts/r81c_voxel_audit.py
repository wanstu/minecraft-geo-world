#!/usr/bin/env python3
"""R8.1c actual Minecraft water-block verification against saved Anvil chunks.

Only FULL chunks count. Do not infer the water surface from GeoWorld's model.
"""
import argparse
import hashlib
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from r81_inspect_voxels import Anvil
from verify_geoworld_voxel_world import inspect

WATER = {"minecraft:water","minecraft:seagrass","minecraft:tall_seagrass",
         "minecraft:kelp","minecraft:kelp_plant","minecraft:bubble_column"}
ICE = {"minecraft:ice", "minecraft:blue_ice", "minecraft:packed_ice"}


def water_top_y(column):
    if column.get("status") != "full":
        return None
    wet = [(height, name) for height, name in column.get("verticalTop", [])
           if name in WATER]
    return max((height for height, _ in wet), default=None)


def surface_top_y(column):
    """Actual block-water surface or one frozen top block, never model data.

    Ice at Y=62 is a physical cap over water at Y=61. A taller iceberg
    does not raise the surrounding ocean datum when Y=62 still contains water.
    """
    if column.get("status") != "full":
        return None
    top = column.get("topBlock")
    if top in WATER or top == "minecraft:ice":
        return column["topY"]
    at_sea = column.get("seaY62Block")
    below = column.get("seaY61Block")
    if at_sea in (WATER | ICE) and (below in (WATER | ICE) or column.get("seaY60Block") in WATER):
        return 62
    return water_top_y(column)


def ordered_rises(rows):
    results = []
    for before, after in zip(rows, rows[1:]):
        a, b = surface_top_y(before), surface_top_y(after)
        if a is None or b is None or b <= a:
            continue
        results.append({"upstream": [before["x"],before["z"]],
                        "downstream": [after["x"],after["z"]],
                        "riseBlocks": b - a})
    return results


def sample(inspection, x, z):
    row = inspection.column(x,z)
    row["waterTopY"] = water_top_y(row)
    row["surfaceTopY"] = surface_top_y(row)
    row["frozenSurface"] = row.get("seaY62Block") in ICE and row.get("seaY61Block") in (WATER | ICE)
    row["icebergAboveSurface"] = (row.get("topBlock") in ICE and row.get("topY",-999) > 62 and row.get("surfaceTopY") == 62)
    return row


def inspect_sites(root):
    a = Anvil(root / "region")
    pair509 = {str(x): [sample(a,x,z) for z in range(-1024,-1032,-1)]
               for x in (-1071,-1070,-1069,-1068)}
    transverse509 = {str(z): [sample(a,x,z) for x in range(-1110,-1023,4)]
                     for z in (-1016,-1026,-1027,-1038,-1050)}
    center915 = [sample(a,x,10432) for x in range(1616,1701)]
    transverse915 = {str(x): [sample(a,x,z) for z in range(10384,10481,4)]
                     for x in (1656,1660,1664,1668,1672,1680,1696)}
    result = {
        "509": {"direction":"decreasing_z","boundaryColumns":pair509,
                "crossSections":transverse509,
                "rises":{x:ordered_rises(rows) for x,rows in pair509.items()}},
        "915": {"direction":"increasing_x","centerLine":center915,
                "crossSections":transverse915,
                "rises":ordered_rises(center915)},
        "sampledChunks":len(a.cache),
        "loadedChunks":sum(chunk is not None for chunk in a.cache.values()),
    }
    return result


def summarize(report):
    records509 = report["509"]["boundaryColumns"]
    pairs = {}
    for x, rows in records509.items():
        a = next(r for r in rows if r["z"] == -1026)
        b = next(r for r in rows if r["z"] == -1027)
        pairs[x] = {"upstreamTop": a["waterTopY"],"downstreamTop":b["waterTopY"],
                    "upstreamStatus": a["status"],"downstreamStatus":b["status"]}
    rows915 = report["915"]["centerLine"]
    return {"509_boundaryPairs":pairs,
            "509_rises":report["509"]["rises"],
            "915_centerTrace":[{"x":r["x"],"z":r["z"],"waterTopY":r["waterTopY"],
                                "surfaceTopY":r["surfaceTopY"],"topBlock":r.get("topBlock"),
                                "status":r["status"]} for r in rows915
                                if r["x"]%4 == 0 or r["x"] in (1663,1665,1678,1682,1683)],
            "915_fullColumns":sum(r["status"]=="full" for r in rows915),
            "915_surfaceY62Count":sum(r["surfaceTopY"]==62 for r in rows915),
            "915_frozenColumns":sum(r["frozenSurface"] for r in rows915),
            "915_icebergsAboveY62":sum(r["icebergAboveSurface"] for r in rows915),
            "915_unresolvedColumns":[r["x"] for r in rows915 if r["surfaceTopY"] is None],
            "915_rises":report["915"]["rises"],
            "sampledChunks":report["sampledChunks"],
            "loadedChunks":report["loadedChunks"]}


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--world",type=Path,required=True)
    parser.add_argument("--expect-rules",default="geoworld-rules-v1.1.23")
    parser.add_argument("--expect-seed",type=int,default=-1818694639628404530)
    parser.add_argument("--output",type=Path)
    args=parser.parse_args()
    preflight=inspect(args.world,args.expect_rules,args.expect_seed)
    if not preflight["pass"]:
        print(json.dumps({"preflight":preflight},ensure_ascii=False,indent=2))
        return 2
    report=inspect_sites(args.world)
    summary=summarize(report)
    output={"schemaVersion":1,"evidenceClass":"minecraft_anvil_full_blockstates",
            "world":str(args.world),"preflight":preflight,"summary":summary,
            "sample":report}
    if args.output:
        args.output.parent.mkdir(parents=True,exist_ok=True)
        args.output.write_text(json.dumps(output,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
    print(json.dumps(summary,ensure_ascii=False,indent=2))
    return 0


if __name__=="__main__":
    sys.exit(main())
