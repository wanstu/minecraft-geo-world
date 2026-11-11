#!/usr/bin/env python3
"""Read-only R8.1c final mouth seam and coastal bed comparison.

Use exact persisted river-centerline positions exported from the saved GeoWorld
model by cmd/geoworld-r81-mouth-points, not guessed diagonal lines.
"""
from __future__ import annotations
import argparse
import csv
from collections import Counter
import hashlib
import json
import math
from pathlib import Path

from r81_inspect_voxels import Anvil, block_at
from r81c_voxel_audit import ordered_rises, sample
from verify_geoworld_voxel_world import inspect

SEED = -1818694639628404530
RULES_OLD = "geoworld-rules-v1.1.22"
RULES_NEW = "geoworld-rules-v1.1.23"

# Minecraft OCEAN_FLOOR includes trees, coral and other block states that are
# not the natural sediment or rock bed. Keep that heightmap separately and
# identify the first geological/soil surface below all decorative objects.
SUBSTRATE = {
    "stone","deepslate","granite","diorite","andesite","tuff","calcite",
    "sand","red_sand","sandstone","red_sandstone","gravel","dirt",
    "grass_block","coarse_dirt","rooted_dirt","podzol","mycelium",
    "mud","packed_mud","clay","terracotta","snow_block","moss_block",
    "pale_moss_block","basalt","blackstone"
}
SUBSTRATE.update(f"{shade}_terracotta" for shade in (
    "white","orange","magenta","light_blue","yellow","lime","pink",
    "gray","light_gray","cyan","purple","blue","brown","green","red","black"
))


def natural_substrate(anvil: Anvil, column: dict, x: int, z: int) -> dict | None:
    if column["status"] != "full":
        return None
    chunk = anvil.cache[(x//16,z//16)]
    if chunk is None:
        return None
    sections = {s["Y"]:s for s in chunk["sections"]}
    max_y = min(150,max(75,column["topY"]))
    for y in range(max_y,30,-1):
        block = block_at(sections,y,x,z)
        if block is None:
            continue
        material = block.removeprefix("minecraft:")
        if material in SUBSTRATE:
            return {"y":y,"material":material}
    return None


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def surface_rows(inspection: Anvil, points: list[dict], min_t: float) -> list[dict]:
    rows = []
    coords = set()
    for p in points:
        if p["t"] + 1e-9 < min_t:
            continue
        x, z = round(p["x"]), round(p["z"])
        if (x, z) in coords:
            continue
        coords.add((x,z))
        row = sample(inspection,x,z)
        row["t"] = p["t"]
        rows.append(row)
    return rows


def transverse_samples(inspection: Anvil, points: list[dict]) -> list[dict]:
    cuts = []
    for t in (0.625, 0.6484375, 0.671875, 0.75, 0.875):
        p = min(points, key=lambda item:abs(item["t"]-t))
        idx = points.index(p)
        before, after = points[max(0,idx-1)], points[min(len(points)-1,idx+1)]
        dx, dz = after["x"]-before["x"], after["z"]-before["z"]
        length = math.hypot(dx,dz)
        if length == 0:
            raise ValueError("invalid stationary river centerline")
        nx, nz = -dz/length, dx/length
        rows=[]
        for offset in range(-32,33,8):
            x,z=round(p["x"]+nx*offset),round(p["z"]+nz*offset)
            row=sample(inspection,x,z)
            row["perpendicularOffset"]=offset
            rows.append(row)
        problems=[]
        for left,right in zip(rows,rows[1:]):
            if left["status"]!="full" or right["status"]!="full":
                continue
            for wet,dry in ((left,right),(right,left)):
                if (wet["surfaceTopY"] is not None and
                    dry["surfaceTopY"] is None and
                    dry["topY"] < wet["surfaceTopY"] - 1):
                    problems.append({"wet":[wet["x"],wet["z"]],
                                     "dry":[dry["x"],dry["z"]],
                                     "wetY":wet["surfaceTopY"],
                                     "dryY":dry["topY"]})
        cuts.append({
            "t":p["t"],"center":[round(p["x"]),round(p["z"])],
            "fullColumns":sum(r["status"]=="full" for r in rows),
            "surfaceY62":sum(r["surfaceTopY"]==62 for r in rows),
            "unknown":sum(r["status"]!="full" for r in rows),
            "lowDryBankGaps":problems,
            "columns":rows
        })
    return cuts


def verify_mouth(inspection: Anvil, mouth: dict) -> dict:
    assert mouth["oceanMouth"], f"river {mouth['id']} not an ocean mouth"
    # Include the exact saved model seam t=.6484375 with upstream samples.
    min_t = 0.625 if mouth["id"]==535 else 0.609375
    rows = surface_rows(inspection,mouth["points"],min_t)
    seam = [r for r in rows if 0.625-1e-9 <= r["t"] <= 0.6875+1e-9]
    cuts = transverse_samples(inspection,mouth["points"])
    return {
        "riverId":mouth["id"],
        "oceanMouth":mouth["oceanMouth"],
        "riverHalfWidthBlocks":mouth["halfWidth"],
        "exactModelCenterline":True,
        "seamT":[0.625,0.6875],
        "seamRows":seam,
        "seamExpectedCount":9,
        "seamFullCount":sum(r["status"]=="full" for r in seam),
        "seamSurfaceY62Count":sum(r["surfaceTopY"]==62 for r in seam),
        "seamInverseRise":ordered_rises(seam),
        "fullExtendedRows":sum(r["status"]=="full" for r in rows),
        "extendedRowsCount":len(rows),
        "extendedSurfaceY62Count":sum(r["surfaceTopY"]==62 for r in rows),
        "extendedInverseRise":ordered_rises(rows),
        "transverseCuts":cuts,
        "transverseFullCount":sum(c["fullColumns"] for c in cuts),
        "transverseMissingCount":sum(c["unknown"] for c in cuts),
        "transverseLowDryBankGaps":sum(len(c["lowDryBankGaps"]) for c in cuts),
    }


def compare_shore(old: Anvil, new: Anvil) -> dict:
    coords = [(x,z) for z in range(-1052,-997) for x in range(-1120,-1009)]
    differences = Counter()
    substrate_differences = Counter()
    substrate_material_changes = Counter()
    substrate_unknown = []
    comparable = 0
    old_cells = {}
    new_cells = {}
    changed_points = []
    for x,z in coords:
        a,b=old.column(x,z),new.column(x,z)
        if a["status"]!="full" or b["status"]!="full":
            continue
        comparable+=1
        old_cells[(x,z)]=a
        new_cells[(x,z)]=b
        delta=b["floorY"]-a["floorY"]
        differences[delta]+=1
        old_substrate=natural_substrate(old,a,x,z)
        new_substrate=natural_substrate(new,b,x,z)
        if old_substrate is None or new_substrate is None:
            substrate_unknown.append((x,z))
        else:
            substrate_differences[new_substrate["y"]-old_substrate["y"]]+=1
            substrate_material_changes[(old_substrate["material"],new_substrate["material"])]+=1
        if delta and len(changed_points)<20:
            changed_points.append({"x":x,"z":z,"deltaY":delta})
    def slope(cells):
        jumps=Counter()
        water_wall=[]
        for (x,z),a in cells.items():
            for dx,dz in ((1,0),(0,1)):
                b=cells.get((x+dx,z+dz))
                if b is None:
                    continue
                d=abs(a["floorY"]-b["floorY"])
                jumps[d]+=1
                for wet,dry in ((a,b),(b,a)):
                    if (wet["wet"] and not dry["wet"] and
                        dry["topY"] < wet["topY"]-1):
                        water_wall.append({
                            "wet":[wet["x"],wet["z"]],
                            "dry":[dry["x"],dry["z"]],
                            "wetTopY":wet["topY"],"dryTopY":dry["topY"]
                        })
        return {
            "absFloorStepCounts":{str(k):v for k,v in sorted(jumps.items())},
            "adjacentFloorStepsGe4":sum(v for k,v in jumps.items() if k>=4),
            "adjacentFloorStepsGe6":sum(v for k,v in jumps.items() if k>=6),
            "lowDryBankGaps":water_wall[:30],
            "lowDryBankGapCount":len(water_wall)
        }
    return {
        "region":{"xMin":-1120,"xMax":-1010,"zMin":-1052,"zMax":-998},
        "testedCoordinates":len(coords),
        "comparableFullColumns":comparable,
        "unknownColumns":len(coords)-comparable,
        "heightmapFloorDeltaHistogram":{str(k):v for k,v in sorted(differences.items())},
        "heightmapUnchangedColumns":differences[0],
        "heightmapChangedColumns":comparable-differences[0],
        "naturalSubstrateDeltaHistogram":{str(k):v for k,v in sorted(substrate_differences.items())},
        "naturalSubstrateUnchangedColumns":substrate_differences[0],
        "naturalSubstrateChangedColumns":sum(v for k,v in substrate_differences.items() if k!=0),
        "naturalSubstrateUnknownColumns":substrate_unknown,
        "naturalSubstrateMaterialChanges":{
            f"{a}->{b}":v for (a,b),v in substrate_material_changes.most_common()
        },
        "method":"natural substrate ignores leaves, coral, fluid, aquatic plants and other decoration; scans top-down for soil, sediment, stone and recognized rock/terracotta from Y<=150 to Y=31",
        "heightmapExamples":changed_points,
        "oldLocalTerrain":slope(old_cells),
        "newLocalTerrain":slope(new_cells),
    }


def java_shoreline_samples(old: Anvil, new: Anvil, rows_path: Path | None) -> dict | None:
    if rows_path is None:
        return None
    rows=[]
    with rows_path.open(newline="",encoding="utf-8") as stream:
        for raw in csv.reader(stream):
            if not raw:
                continue
            if len(raw)!=14:
                raise ValueError(f"invalid Java shoreline planner row: {raw!r}")
            x,z=int(raw[0]),int(raw[1])
            old_s=natural_substrate(old,old.column(x,z),x,z)
            new_s=natural_substrate(new,new.column(x,z),x,z)
            if old_s is None or new_s is None:
                raise ValueError(f"missing real Anvil column at {(x,z)}")
            old_planned=int(raw[4])-1
            new_planned=int(raw[5])-1
            old_actual=old_s["y"]
            new_actual=new_s["y"]
            rows.append({
                "x":x,"z":z,
                "plannedOldY":old_planned,"plannedNewY":new_planned,
                "actualOldNaturalSubstrateY":old_actual,
                "actualNewNaturalSubstrateY":new_actual,
                "plannedDeltaY":new_planned-old_planned,
                "actualDeltaY":new_actual-old_actual,
                "oldWaterBoundaryY":int(raw[6]),"newWaterBoundaryY":int(raw[7]),
                "oldShoreWaterKind":raw[10],"newShoreWaterKind":raw[11],
            })
    return {
        "source":"Java GeoWorldShorelinePlanner.planModel for saved models versus read-only full Anvil natural substrate",
        "sampleCount":len(rows),
        "matchingDeltaCount":sum(r["plannedDeltaY"]==r["actualDeltaY"] for r in rows),
        "mismatches":[r for r in rows if r["plannedDeltaY"]!=r["actualDeltaY"]],
        "rows":rows,
        "caveat":"One sand cover may be placed at the shoreline without changing its planModel ground boundary"
    }


def ground_samples(old_audit: Path | None,new_audit: Path | None) -> dict | None:
    if old_audit is None or new_audit is None:
        return None
    a=json.loads(old_audit.read_text())
    b=json.loads(new_audit.read_text())
    first={s["case"]["id"]:s for s in a["sites"]}
    second={s["case"]["id"]:s for s in b["sites"]}
    if first.keys()!=second.keys():
        raise ValueError("model ground audit case mismatch")
    results=[]
    for key in first:
        old,new=first[key],second[key]
        if (old["case"]["x"],old["case"]["z"])!=(new["case"]["x"],new["case"]["z"]):
            raise ValueError("model ground position mismatch")
        results.append({
            "id":key,
            "x":old["case"]["x"],"z":old["case"]["z"],
            "oldGroundY":old["point"]["groundY"],"newGroundY":new["point"]["groundY"],
            "oldWaterKind":old["point"]["kind"],"newWaterKind":new["point"]["kind"],
        })
    return {
        "source":"read-only Go WorldSample of the two persisted models",
        "sampleCount":len(results),
        "identicalGroundY":sum(r["oldGroundY"]==r["newGroundY"] for r in results),
        "rows":results,
        "scope":"stratified 22 model ground points, not all 6105 voxel floor differences"
    }


def main() -> int:
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("--legacy-world",type=Path,required=True)
    p.add_argument("--candidate-world",type=Path,required=True)
    p.add_argument("--centerlines",type=Path,required=True)
    p.add_argument("--old-model-audit",type=Path)
    p.add_argument("--new-model-audit",type=Path)
    p.add_argument("--java-shoreline-csv",type=Path)
    p.add_argument("--output",type=Path,required=True)
    args=p.parse_args()
    old_pass=inspect(args.legacy_world,RULES_OLD,SEED)
    new_pass=inspect(args.candidate_world,RULES_NEW,SEED)
    if not old_pass["pass"] or not new_pass["pass"]:
        raise SystemExit("Preflight failed: legacy or candidate save is not a locked GeoWorld world")
    rivers={r["id"]:r for r in json.loads(args.centerlines.read_text())}
    if not all(i in rivers for i in (535,851)):
        raise SystemExit("Official centerlines for river #535 and #851 missing")
    old=Anvil(args.legacy_world/"region")
    new=Anvil(args.candidate_world/"region")
    shoreline=compare_shore(old,new)
    mouths={str(i):verify_mouth(new,rivers[i]) for i in (535,851)}
    grounds=ground_samples(args.old_model_audit,args.new_model_audit)
    java_shores=java_shoreline_samples(old,new,args.java_shoreline_csv)
    output={
        "schemaVersion":1,
        "evidenceClass":"actual_full_minecraft_anvil_plus_persisted_model",
        "seed":SEED,
        "legacyRules":RULES_OLD,
        "candidateRules":RULES_NEW,
        "legacyPreflight":old_pass,
        "candidatePreflight":new_pass,
        "modelSha256":{
            "legacy":sha256(args.legacy_world/"data/geoworld/world.geoworld"),
            "candidate":sha256(args.candidate_world/"data/geoworld/world.geoworld")
        },
        "regionSha256":{
            "535":sha256(args.candidate_world/"region/r.-20.4.mca"),
            "851":sha256(args.candidate_world/"region/r.-20.2.mca")
        },
        "mouths":mouths,
        "mouth509Floor":shoreline,
        "groundSampleComparison":grounds,
        "javaShorelineComparison":java_shores,
        "allSeamTestsPass":all(
            m["seamFullCount"]==9 and
            m["seamSurfaceY62Count"]==9 and
            m["seamInverseRise"]==[] for m in mouths.values()
        ),
        "releaseEligible":False,
        "caveats":[
            "No old v1.1.22 Minecraft full chunks at #535/#851; old +1 is model evidence",
            "Supplementary exact seam tests do not prove every coastline or multi-loader compatibility",
            "R8.1c coastal water ownership can reshape voxel shoreline while model ground stays stable",
            "R8.1d sinkhole #11/drainage conflict remains outside this stage"
        ]
    }
    args.output.parent.mkdir(parents=True,exist_ok=True)
    args.output.write_text(json.dumps(output,ensure_ascii=False,indent=2)+"\n",encoding="utf-8")
    print(json.dumps({
        "allSeamTestsPass":output["allSeamTestsPass"],
        "mouths":{k:{
            "full":v["seamFullCount"],"sameY62":v["seamSurfaceY62Count"],
            "extendedFull":v["fullExtendedRows"],"extendedTotal":v["extendedRowsCount"],
            "extendedSameY62":v["extendedSurfaceY62Count"],
            "transverseFull":v["transverseFullCount"],"bankGaps":v["transverseLowDryBankGaps"]
        } for k,v in mouths.items()},
        "heightmapChanged":shoreline["heightmapChangedColumns"],
        "naturalSubstrateChanged":shoreline["naturalSubstrateChangedColumns"],
        "floorTotal":shoreline["comparableFullColumns"],
        "modelGround":None if grounds is None else [grounds["identicalGroundY"],grounds["sampleCount"]],
        "javaShoreline":None if java_shores is None else [java_shores["matchingDeltaCount"],java_shores["sampleCount"]],
        "output":str(args.output)
    },ensure_ascii=False,indent=2))
    return 0 if output["allSeamTestsPass"] else 3


if __name__=="__main__":
    raise SystemExit(main())
