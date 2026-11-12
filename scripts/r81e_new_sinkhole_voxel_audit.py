#!/usr/bin/env python3
"""GeoWorld R8.1e: real Anvil block audit for relocated sinkhole #11.

This inspects the *saved v1.1.24 Minecraft world*, not offline sampled
geometry, and never treats missing chunks as valid terrain. Output images
are diagnostic block-state projections, NOT gameplay screenshots.
"""
from __future__ import annotations
import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

from r81_inspect_voxels import Anvil
from r81d_sinkhole_voxel_audit import actual_voxel
from verify_geoworld_voxel_world import inspect

SEED = -1818694639628404530
RULES = "geoworld-rules-v1.1.25"
CX, CZ = 320,6592
EAST_WEST = [160,192,200,208,216,224,232,240,256,280,304,320,336,360,384,400,408,424,432,448,464,480]
NORTH_SOUTH = [6464,6480,6496,6504,6512,6528,6544,6560,6576,6592,6608,6624,6640,6656,6672,6688,6704,6720]

def profile(anvil:Anvil,axis:str)->list[dict]:
    points=[(x,CZ) for x in EAST_WEST] if axis=="x" else [(CX,z) for z in NORTH_SOUTH]
    result=[]
    for x,z in points:
        row=actual_voxel(anvil,x,z)
        row["axis"]=axis
        row["axisCoordinate"]=x if axis=="x" else z
        result.append(row)
    return result

def y_of(rows: list[dict], coord:int)->int|None:
    return next((v.get("naturalSubstrateY") for v in rows if v["axisCoordinate"]==coord),None)

def render(anvil:Anvil, horizontal:list[dict],vertical:list[dict],output:Path)->dict:
    # Actual Minecraft blocks; every skipped/unloaded pixel stays grey.
    x0,x1=144,496
    z0,z1=6432,6752
    step=2
    w=(x1-x0)//step+1
    h=(z1-z0)//step+1
    img=Image.new("RGB",(w,h),(64,66,72))
    px=img.load()
    hist=Counter()
    for zi,z in enumerate(range(z0,z1+1,step)):
        for xi,x in enumerate(range(x0,x1+1,step)):
            v=anvil.column(x,z)
            if v["status"]!="full":
                hist["unknown"]+=1
                continue
            hist["full"]+=1
            y=v["floorY"]
            color=(47,108,171) if v["wet"] else (
                (102,126,75) if y>=110 else
                (120,135,79) if y>=100 else
                (158,146,91) if y>=90 else
                (188,161,109) if y>=80 else
                (209,181,131)
            )
            px[xi,zi]=color
    scale=3
    W,H=w*scale,h*scale
    panel=Image.new("RGB",(W+500,H+130),(23,30,40))
    d=ImageDraw.Draw(panel)
    title="GeoWorld R8.1e / sinkhole #11 / actual Minecraft Anvil blocks"
    d.text((20,16),title,fill=(232,236,242))
    d.text((20,36),"Seed -1818694639628404530 | v1.1.24 | plan view: OCEAN_FLOOR heightmap",fill=(179,192,205))
    d.text((20,56),"Gray=not generated; separate side profiles use natural substrate Y",fill=(255,220,167))
    panel.paste(img.resize((W,H),Image.Resampling.NEAREST),(20,96))
    d=ImageDraw.Draw(panel)
    xcenter=20+(CX-x0)*scale//step
    zcenter=96+(CZ-z0)*scale//step
    d.ellipse((xcenter-6,zcenter-6,xcenter+6,zcenter+6),outline=(235,87,78),width=2)
    left=W+48
    d.text((left,104),"Center X=320, Z=6592",fill=(235,232,215))
    def chart(rows,axis,top,label):
        d.text((left,top),label,fill=(231,231,235))
        baseline=top+175
        plot_h=132
        wplot=395
        lo,hi=(160,480) if axis=="x" else (6464,6720)
        d.line((left,baseline,left+wplot,baseline),fill=(109,125,148),width=1)
        d.line((left,baseline-plot_h,left,baseline),fill=(109,125,148),width=1)
        previous=None
        for row in rows:
            c=row["axisCoordinate"]
            y=row.get("naturalSubstrateY")
            if y is None:
                previous=None
                continue
            xp=left+(c-lo)/(hi-lo)*wplot
            yp=baseline-(max(60,min(130,y))-60)/70*plot_h
            if previous is not None:
                d.line((previous[0],previous[1],xp,yp),fill=(227,176,94),width=2)
            d.ellipse((xp-2,yp-2,xp+2,yp+2),fill=(235,204,122))
            previous=(xp,yp)
        d.text((left,baseline+7),f"{lo}  ...  {hi}  | Y=60 to 130",fill=(164,179,194))
    chart(horizontal,"x",160,"East-West natural substrate profile")
    chart(vertical,"z",435,"North-South natural substrate profile")
    output.parent.mkdir(parents=True,exist_ok=True)
    panel.save(output,optimize=True)
    return {"sampleGridStep":step,"gridFullCells":hist["full"],"gridMissingCells":hist["unknown"],
            "heightMap":"OCEAN_FLOOR (not natural substrate)",
            "profileHeights":"actual blockstate geological/soil top",
            "gameplayScreenshot":False}

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("--world",type=Path,required=True)
    p.add_argument("--output",type=Path,required=True)
    p.add_argument("--image",type=Path)
    args=p.parse_args()
    checked=inspect(args.world,RULES,SEED)
    if not checked["pass"]:
        raise SystemExit("Invalid saved world rules/seed binding")
    a=Anvil(args.world/"region")
    horizontal=profile(a,"x")
    vertical=profile(a,"z")
    core=[*horizontal,*vertical]
    core_full=all(v["status"]=="full" for v in core)
    center=next(v for v in horizontal if v["axisCoordinate"]==CX)
    flat_x=[v for v in horizontal if 240<=v["axisCoordinate"]<=400]
    flat_z=[v for v in vertical if 6528<=v["axisCoordinate"]<=6656]
    slope_pairs=[
        {"fromX":aa,"toX":bb,"fromY":y_of(horizontal,aa),"toY":y_of(horizontal,bb),
         "risePerHorizontalBlock":None if y_of(horizontal,aa) is None or y_of(horizontal,bb) is None else
            abs(y_of(horizontal,bb)-y_of(horizontal,aa))/(bb-aa)}
        for aa,bb in [(208,216),(216,224),(224,232),(400,408),(408,424),(424,432)]
    ]
    evidence={
        "schemaVersion":1,
        "evidenceClass":"minecraft_anvil_saved_full_blockstates",
        "seed":SEED,"rulesVersion":RULES,
        "center":[CX,CZ],
        "preflight":checked,
        "modelSha256":hashlib.sha256((args.world/"data/geoworld/world.geoworld").read_bytes()).hexdigest(),
        "profileSampling":{"eastWest":horizontal,"northSouth":vertical},
        "summary":{
            "profileSamples":len(core),"fullSamples":sum(v["status"]=="full" for v in core),
            "allProfileSamplesFull":core_full,
            "actualCenterNaturalY":center.get("naturalSubstrateY"),
            "actualCenterWaterTopY":center.get("waterTopY"),
            "westOuterNaturalY":y_of(horizontal,160),
            "eastOuterNaturalY":y_of(horizontal,480),
            "northOuterNaturalY":y_of(vertical,6464),
            "southOuterNaturalY":y_of(vertical,6720),
            "flatCoreEastWestAllY77":len(flat_x)>=9 and all(v.get("naturalSubstrateY")==77 for v in flat_x),
            "flatCoreNorthSouthAllY77":len(flat_z)>=9 and all(v.get("naturalSubstrateY")==77 for v in flat_z),
            "steepSlopeSamples":slope_pairs,
        },
        "releaseEligible":False,
        "gameplayVisualAcceptance":"NOT_DONE",
        "conclusion":"new placement exists and original river overlap removed; flat 77-Y pit floor and steep walls require visual geomorphology review"
    }
    if args.image:
        evidence["visualization"]={
            "path":str(args.image),
            **render(a,horizontal,vertical,args.image)
        }
    args.output.parent.mkdir(parents=True,exist_ok=True)
    args.output.write_text(json.dumps(evidence,ensure_ascii=False,indent=2)+"\n")
    print(json.dumps(evidence["summary"],ensure_ascii=False))
    if not core_full:
        raise SystemExit("Incomplete full-block profile evidence: unknown columns present")
if __name__=="__main__":
    main()
