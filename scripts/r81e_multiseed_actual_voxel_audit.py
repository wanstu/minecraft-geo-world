#!/usr/bin/env python3
"""Read-only independent-seed Minecraft Anvil validation for GeoWorld R8.1e.

No generated model height may substitute for an absent chunk. This audit
proves only local voxel cross-sections, NOT full-world visual realism.
"""
import argparse
import hashlib
import json
from pathlib import Path
from PIL import Image, ImageDraw

from r81_inspect_voxels import Anvil
from r81d_sinkhole_voxel_audit import actual_voxel
from verify_geoworld_voxel_world import inspect

RULES="geoworld-rules-v1.1.25"
OFFSETS=list(range(-160,161,8))

def section(anvil,axis,cx,cz):
    out=[]
    for offset in OFFSETS:
        x,z=(cx+offset,cz) if axis=="eastWest" else (cx,cz+offset)
        row=actual_voxel(anvil,x,z)
        row["offset"]=offset
        out.append(row)
    return out

def max_slope(rows):
    if any(r["status"]!="full" or r["naturalSubstrateY"] is None for r in rows):
        return None
    values=[r["naturalSubstrateY"] for r in rows]
    return max(abs(b-a)/8 for a,b in zip(values,values[1:]))

def render(anvil,cx,cz,seed,path):
    radius=176
    step=4
    size=(radius*2//step+1,radius*2//step+1)
    base=Image.new("RGB",size,(63,66,72));pix=base.load()
    full=0
    for zi,z in enumerate(range(cz-radius,cz+radius+1,step)):
        for xi,x in enumerate(range(cx-radius,cx+radius+1,step)):
            row=anvil.column(x,z)
            if row["status"]!="full":continue
            full+=1;y=row["floorY"]
            pix[xi,zi]=(48,106,162) if row["wet"] else (
                (109,126,84) if y>=110 else
                (134,149,91) if y>=100 else
                (153,140,95) if y>=90 else
                (194,169,112) if y>=80 else
                (210,181,132)
            )
    scale=4
    output=Image.new("RGB",(size[0]*scale,size[1]*scale+80),(25,31,40))
    output.paste(base.resize((size[0]*scale,size[1]*scale),Image.Resampling.NEAREST),(0,80))
    d=ImageDraw.Draw(output)
    d.text((14,13),f"GeoWorld R8.1e | actual Minecraft Anvil blocks | seed {seed}",fill=(233,235,239))
    d.text((14,33),f"Sinkhole center ({cx},{cz}); 4-block sampling; OCEAN_FLOOR top",fill=(201,215,226))
    d.text((14,53),"GRAY=ungenerated; NOT an in-game camera screenshot",fill=(255,224,169))
    middle=size[0]//2*scale
    d.ellipse((middle-6,80+middle-6,middle+6,80+middle+6),outline=(255,114,110),width=2)
    path.parent.mkdir(parents=True,exist_ok=True)
    output.save(path,optimize=True)
    return {"fullPixels":full,"totalPixels":size[0]*size[1],"sampleStep":step,
            "notInGameScreenshot":True}

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("--world",type=Path,required=True)
    p.add_argument("--seed",type=int,required=True)
    p.add_argument("--center-x",type=int,required=True)
    p.add_argument("--center-z",type=int,required=True)
    p.add_argument("--site-id",type=int,required=True)
    p.add_argument("--output",type=Path,required=True)
    p.add_argument("--image",type=Path)
    args=p.parse_args()
    preflight=inspect(args.world,RULES,args.seed)
    if not preflight["pass"]:
        raise SystemExit("Invalid GeoWorld model binding; cannot claim Minecraft voxel verification")
    anvil=Anvil(args.world/"region")
    sections={axis:section(anvil,axis,args.center_x,args.center_z)
              for axis in ("eastWest","northSouth")}
    complete={axis:sum(x["status"]=="full" for x in rows) for axis,rows in sections.items()}
    worst={axis:max_slope(rows) for axis,rows in sections.items()}
    center=next(row for row in sections["eastWest"] if row["offset"]==0)
    report={
        "schemaVersion":1,"evidenceClass":"saved_minecraft_full_anvil_blockstate",
        "seed":args.seed,"rulesVersion":RULES,
        "site":{"id":args.site_id,"centerX":args.center_x,"centerZ":args.center_z},
        "preflight":preflight,
        "modelSha256":hashlib.sha256((args.world/"data/geoworld/world.geoworld").read_bytes()).hexdigest(),
        "summary":{"columnsExpected":len(OFFSETS)*2,
                   "columnsFull":sum(complete.values()),
                   "crossSectionsFull":all(x==len(OFFSETS) for x in complete.values()),
                   "groundAtCenter":center.get("naturalSubstrateY"),
                   "waterAtCenter":center.get("waterTopY"),
                   "maxNaturalSlopePer8Blocks":worst,
                   "fullCounts":complete,
                   "flatFloorCounts":{k:sum(r.get("naturalSubstrateY")==center.get("naturalSubstrateY") for r in rows) for k,rows in sections.items()}},
        "profiles":sections,
        "releaseEligible":False,
        "gameCameraVisualAcceptance":"NOT_DONE"
    }
    if args.image:
        report["visualization"]={"file":str(args.image),**render(anvil,args.center_x,args.center_z,args.seed,args.image)}
    args.output.parent.mkdir(parents=True,exist_ok=True)
    args.output.write_text(json.dumps(report,ensure_ascii=False,indent=2)+"\n")
    print(json.dumps(report["summary"],ensure_ascii=False,indent=2))
    if not report["summary"]["crossSectionsFull"]:
        raise SystemExit("Missing actual Minecraft full chunks: do not accept")
if __name__=="__main__":main()
