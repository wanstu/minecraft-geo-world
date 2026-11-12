#!/usr/bin/env python3
"""Compare saved v1.1.23 and v1.1.24 Minecraft sinkhole/river voxels.

Read-only. Unloaded/unfinalized chunks count as missing, NEVER as dry land.
"""
from __future__ import annotations
import argparse
import hashlib
import json
from pathlib import Path

from r81_inspect_voxels import Anvil, block_at, wetnames
from r81c_coastal_followup import natural_substrate
from verify_geoworld_voxel_world import inspect

SEED = -1818694639628404530
OLD = "geoworld-rules-v1.1.23"
NEW = "geoworld-rules-v1.1.24"


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def actual_voxel(anvil: Anvil, x: int, z: int) -> dict:
    row = anvil.column(x,z)
    out = {"x":x,"z":z,"status":row["status"]}
    if row["status"] != "full":
        return out
    chunk=anvil.cache[(x//16,z//16)]
    sections={s["Y"]:s for s in chunk["sections"]}
    wet_y=[y for y in range(58,131) if block_at(sections,y,x,z) in wetnames]
    substrate=natural_substrate(anvil,row,x,z)
    return {
        **out,
        "topY":row["topY"],
        "topBlock":row["topBlock"],
        "oceanFloorHeightmapY":row["floorY"],
        "naturalSubstrateY":None if substrate is None else substrate["y"],
        "naturalSubstrateMaterial":None if substrate is None else substrate["material"],
        "waterTopY":max(wet_y) if wet_y else None,
        "y69":block_at(sections,69,x,z),
        "y79":block_at(sections,79,x,z),
        "y96":block_at(sections,96,x,z),
    }


def main() -> int:
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("--legacy-world",type=Path,required=True)
    p.add_argument("--new-world",type=Path,required=True)
    p.add_argument("--sites",type=Path,required=True)
    p.add_argument("--output",type=Path,required=True)
    args=p.parse_args()
    checks={
        "legacy":inspect(args.legacy_world,OLD,SEED),
        "candidate":inspect(args.new_world,NEW,SEED),
    }
    if not all(c["pass"] for c in checks.values()):
        print(json.dumps(checks,ensure_ascii=False,indent=2))
        raise SystemExit("world preflight rejected; not valid actual voxel evidence")

    old,new=Anvil(args.legacy_world/"region"),Anvil(args.new_world/"region")
    sites=json.loads(args.sites.read_text())
    rows=[]
    for site in sites["cases"]:
        x,z=site["x"],site["z"]
        a,b=actual_voxel(old,x,z),actual_voxel(new,x,z)
        rows.append({
            "site":{"x":x,"z":z,"t":site.get("t"),"offset":site.get("offset"),
                    "control":site.get("control",False)},
            "legacy":a,"new":b,
            "substrateDeltaY":None if a.get("naturalSubstrateY") is None or b.get("naturalSubstrateY") is None
                  else b["naturalSubstrateY"]-a["naturalSubstrateY"],
            "waterTopDeltaY":None if a.get("waterTopY") is None or b.get("waterTopY") is None
                  else b["waterTopY"]-a["waterTopY"],
        })

    centers=[r for r in rows if r["site"].get("offset")==0]
    near_banks=[r for r in rows if r["site"].get("offset") in (16,32,48)]
    sinkhole_center=next((r for r in rows if r["site"]["x"]==-1600 and
                            r["site"]["z"]==5312 and r["site"]["control"]),None)
    summary={
        "actualFullPairs":sum(r["legacy"]["status"]=="full" and r["new"]["status"]=="full" for r in rows),
        "unknownPairs":sum(r["legacy"]["status"]!="full" or r["new"]["status"]!="full" for r in rows),
        "centerCount":len(centers),
        "unchangedCenterWaterY":sum(r["legacy"].get("waterTopY")==r["new"].get("waterTopY") and
                                  r["legacy"].get("waterTopY") is not None for r in centers),
        "bankPairs":len(near_banks),
        "bankHeights": [
            {
                "t":r["site"]["t"],"offset":r["site"]["offset"],
                "oldNaturalY":r["legacy"].get("naturalSubstrateY"),
                "newNaturalY":r["new"].get("naturalSubstrateY"),
            } for r in near_banks
        ],
        "oldSinkholeCenterNaturalY":None if sinkhole_center is None else sinkhole_center["legacy"].get("naturalSubstrateY"),
        "newFormerSinkholeCenterNaturalY":None if sinkhole_center is None else sinkhole_center["new"].get("naturalSubstrateY"),
    }
    # The two ungenerated peripheral samples (offset -48) are explicitly
    # outside the decisive centerline and low-bank cross-sections. Do not
    # claim coverage for them; all required river/bank samples must be real.
    core_centers = [r for r in rows if r["site"].get("offset") == 0]
    core_banks = [r for r in rows if r["site"].get("offset") in (16,32,48)]
    core_full = all(
        r["legacy"]["status"] == "full" and r["new"]["status"] == "full"
        for r in core_centers + core_banks
    )
    centers_same_wet_surface = len(core_centers) == 4 and all(
        r["legacy"].get("waterTopY") is not None and
        r["legacy"].get("waterTopY") == r["new"].get("waterTopY")
        for r in core_centers
    )
    center_levels = {
        r["site"]["t"]:r["new"].get("waterTopY") for r in core_centers
    }
    banks_above_river = len(core_banks) == 12 and all(
        r["new"].get("naturalSubstrateY") is not None and
        center_levels.get(r["site"]["t"]) is not None and
        r["new"]["naturalSubstrateY"] >= center_levels[r["site"]["t"]]
        for r in core_banks
    )
    pit_restored = (
        sinkhole_center is not None and
        sinkhole_center["legacy"].get("naturalSubstrateY") is not None and
        sinkhole_center["new"].get("naturalSubstrateY") is not None and
        sinkhole_center["new"]["naturalSubstrateY"] -
        sinkhole_center["legacy"]["naturalSubstrateY"] >= 20
    )
    missing = [r for r in rows if r["legacy"]["status"] != "full" or r["new"]["status"] != "full"]
    peripheral_only_missing = all(
        r["site"].get("offset") == -48 and not r["site"].get("control")
        for r in missing
    )
    acceptance = {
        "fixedRiverAndBankCasePass": (
            core_full and centers_same_wet_surface and
            banks_above_river and pit_restored and peripheral_only_missing
        ),
        "fourCenterlineWaterLevelsUnchanged":centers_same_wet_surface,
        "twelveRightBankSubstratesAboveWater":banks_above_river,
        "formerPitSubstrateRaisedAtLeast20Blocks":pit_restored,
        "allMandatoryVoxelPairsFull":core_full,
        "peripheralUnknownOnly":peripheral_only_missing,
        "full32PointCoverage":len(missing)==0,
        "untestedPeripheralCoordinates":[
            [r["site"]["x"],r["site"]["z"]] for r in missing
        ],
        "scope":"R8.1d river #735 / sinkhole #11 fixed regression only; not RC release"
    }
    output={
        "schemaVersion":1,
        "evidenceClass":"real_saved_full_minecraft_anvil_section_block_states",
        "seed":SEED,
        "preflight":checks,
        "worldModelSHA256":{
            "legacy":sha(args.legacy_world/"data/geoworld/world.geoworld"),
            "new":sha(args.new_world/"data/geoworld/world.geoworld"),
        },
        "newRegionSHA256":{r.name:sha(r) for r in (args.new_world/"region").glob("r.-4.*.mca")},
        "summary":summary,
        "acceptance":acceptance,
        "samples":rows,
        "releaseEligible":False,
        "caveats":["Sampled local cross-sections are not whole-world visual acceptance",
                   "Control old and new worlds were created separately under their own locked rules",
                   "Never interpret not_generated as dry terrain"]
    }
    args.output.parent.mkdir(parents=True,exist_ok=True)
    args.output.write_text(json.dumps(output,ensure_ascii=False,indent=2)+"\n")
    print(json.dumps({"summary":summary,"acceptance":acceptance},ensure_ascii=False,indent=2))
    return 0 if acceptance["fixedRiverAndBankCasePass"] else 3


if __name__=="__main__":
    raise SystemExit(main())
