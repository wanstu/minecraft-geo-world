#!/usr/bin/env python3
from __future__ import annotations

import argparse, gzip, hashlib, json, math, sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "scripts"))
from r81c_actual_mouth_voxel_audit import Anvil, block_at, read_chunk

SUBSTRATE = {
    "stone","deepslate","granite","diorite","andesite","tuff","calcite",
    "sand","red_sand","sandstone","red_sandstone","gravel","dirt","grass_block",
    "coarse_dirt","rooted_dirt","podzol","mycelium","mud","packed_mud","clay",
    "terracotta","snow_block","moss_block","pale_moss_block","basalt","blackstone",
    "ice","packed_ice","blue_ice",
}
SUBSTRATE |= {f"{c}_terracotta" for c in (
    "white","orange","magenta","light_blue","yellow","lime","pink","gray",
    "light_gray","cyan","purple","blue","brown","green","red","black"
)}

def sha256(path: Path) -> str:
    h=hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda:f.read(1024*1024), b""):
            h.update(chunk)
    return h.hexdigest()

def terrain_surface(anvil: Anvil, x:int, z:int) -> dict:
    col=anvil.column(x,z)
    out={"status":col["status"],"topY":col.get("topY"),"topBlock":col.get("topBlock")}
    if col["status"]!="full":
        return out
    chunk=anvil.chunk(x,z)
    sections={s.get("Y"):s for s in chunk.get("sections",[])}
    start=min(319, max(-64, int(col["topY"])))
    for y in range(start, -65, -1):
        name=block_at(sections,y,x,z)
        if name and name.removeprefix("minecraft:") in SUBSTRATE:
            out["terrainY"]=y
            out["terrainBlock"]=name
            return out
    out["terrainY"]=None
    out["terrainBlock"]=None
    return out

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--world-root",required=True)
    ap.add_argument("--fixture",required=True)
    ap.add_argument("--targets",required=True)
    ap.add_argument("--output",required=True)
    args=ap.parse_args()
    world=Path(args.world_root)
    fixture=json.loads(Path(args.fixture).read_text())
    with gzip.open(world/"data/geoworld/world.geoworld","rt",encoding="utf-8") as f:
        model=json.load(f)
    manifest=model["manifest"]
    assert manifest["seed"]==fixture["seed"]
    assert manifest["generationRulesVersion"]==fixture["rulesVersion"]=="geoworld-rules-v1.1.25"

    targets=set()
    for line in Path(args.targets).read_text().splitlines():
        if not line.strip(): continue
        p=line.split()
        x,z=int(p[-2]),int(p[-1])
        targets.add((x//16,z//16))
    full_targets=0
    missing_targets=[]
    for cx,cz in sorted(targets):
        if read_chunk(world/"region",cx,cz) is not None:
            full_targets+=1
        else:
            missing_targets.append([cx,cz])

    anvil=Anvil(world/"region")
    rows=[]
    errors=[]
    for row in fixture["rows"]:
        actual=terrain_surface(anvil,row["x"],row["z"])
        expected_surface=round(row["expectedGroundY"])-1
        # Volcanic crater lava is an intentional visible surface above the
        # natural substrate. Compare that visible lava top to the model shape;
        # ordinary water still compares against its terrain bed.
        geometry_y = actual.get("topY") if actual.get("topBlock") == "minecraft:lava" else actual.get("terrainY")
        err=None if geometry_y is None else geometry_y-expected_surface
        if err is not None: errors.append(err)
        rows.append({**row,"expectedSurfaceY":expected_surface,"actual":actual,
                     "actualGeometryY":geometry_y,"surfaceErrorBlocks":err})

    groups={}
    for row in rows:
        key=f'{row["featureId"]}:{row["kind"]}:{row["axis"]}'
        groups.setdefault(key,[]).append(row)
    group_summary={}
    for key,grp in groups.items():
        valid=[r for r in grp if r["surfaceErrorBlocks"] is not None]
        special=[r for r in valid if r["specialFeature"]]
        group_summary[key]={
            "points":len(grp),
            "fullPoints":len(valid),
            "specialPoints":len(special),
            "maxAbsSurfaceError":max((abs(r["surfaceErrorBlocks"]) for r in valid),default=None),
            "meanAbsSurfaceError":(sum(abs(r["surfaceErrorBlocks"]) for r in valid)/len(valid)) if valid else None,
            "expectedMinSurfaceY":min((r["expectedSurfaceY"] for r in valid),default=None),
            "expectedMaxSurfaceY":max((r["expectedSurfaceY"] for r in valid),default=None),
            "actualMinGeometryY":min((r["actualGeometryY"] for r in valid),default=None),
            "actualMaxGeometryY":max((r["actualGeometryY"] for r in valid),default=None),
        }

    feature_summary={}
    for fid in sorted({r["featureId"] for r in rows}):
        fr=[r for r in rows if r["featureId"]==fid and r["surfaceErrorBlocks"] is not None]
        center_candidates=[r for r in fr if (r["x"],r["z"]) in ((10944,-6464),(-5184,-3648))]
        center=center_candidates[0] if center_candidates else None
        feature_summary[str(fid)]={
            "kind":fr[0]["kind"] if fr else None,
            "points":len(fr),
            "maxAbsSurfaceError":max((abs(r["surfaceErrorBlocks"]) for r in fr),default=None),
            "centerExpectedSurfaceY":center["expectedSurfaceY"] if center else None,
            "centerActualGeometryY":center["actualGeometryY"] if center else None,
            "profileMinActualGeometryY":min((r["actualGeometryY"] for r in fr),default=None),
            "profileMaxActualGeometryY":max((r["actualGeometryY"] for r in fr),default=None),
        }

    out={
        "schemaVersion":1,
        "evidenceClass":"fresh_saved_minecraft_1_21_1_neoforge_anvil_vs_runtime_model",
        "world":world.name,
        "seed":manifest["seed"],
        "rulesVersion":manifest["generationRulesVersion"],
        "modelSha256":sha256(world/"data/geoworld/world.geoworld"),
        "targetChunks":{"expected":len(targets),"full":full_targets,"missing":missing_targets},
        "surfaceAgreement":{
            "points":len(rows),
            "fullPoints":sum(r["surfaceErrorBlocks"] is not None for r in rows),
            "maxAbsErrorBlocks":max((abs(e) for e in errors),default=None),
            "meanAbsErrorBlocks":sum(abs(e) for e in errors)/len(errors) if errors else None,
            "within1Block":sum(abs(e)<=1 for e in errors),
            "within2Blocks":sum(abs(e)<=2 for e in errors),
        },
        "groups":group_summary,
        "features":feature_summary,
        "rows":rows,
    }
    if full_targets != len(targets):
        raise AssertionError(f"incomplete Anvil evidence: {full_targets}/{len(targets)} target chunks FULL")
    if out["surfaceAgreement"]["fullPoints"] != len(rows):
        raise AssertionError("not all model profile samples resolved to FULL Anvil columns")
    if out["surfaceAgreement"]["maxAbsErrorBlocks"] != 0:
        raise AssertionError(
            f"model/voxel profile mismatch: max error={out['surfaceAgreement']['maxAbsErrorBlocks']} blocks"
        )
    Path(args.output).write_text(json.dumps(out,ensure_ascii=False,indent=2)+"\n")
    print(json.dumps({k:out[k] for k in ("targetChunks","surfaceAgreement","features")},ensure_ascii=False,indent=2))

if __name__=="__main__":
    main()
