#!/usr/bin/env python3
"""Render a diagnostic top-down comparison from SAVED Minecraft Anvil blocks.

This is an Anvil block-state projection, not an in-game screenshot.
Unknown/unloaded areas stay gray, never invented terrain. Shared color scale.
"""
import argparse
import json
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
from r81_inspect_voxels import Anvil

X0,X1 = -1760,-1544
Z0,Z1 = 5250,5466
SCALE=3
SIZE=(X1-X0+1,Z1-Z0+1)
MARGIN=25
BACKGROUND=(26,32,43)
UNKNOWN=(66,69,77)

def clamp(x,a,b):return max(a,min(b,x))
def rgb_ground(height, block):
    name=(block or "").removeprefix("minecraft:")
    if name in ("water","seagrass","tall_seagrass","kelp","kelp_plant","bubble_column"):
        t=clamp((height-60)/60,0,1)
        return (int(34+20*t),int(91+34*t),int(154+45*t))
    if "ice" in name or name=="snow":
        return (173,202,215)
    if "sand" in name or "sandstone" in name:
        t=clamp((height-65)/55,0,1)
        return (int(169+46*t),int(137+39*t),int(97+29*t))
    if "leaves" in name or "grass" in name or name in ("moss_block","podzol"):
        t=clamp((height-65)/55,0,1)
        return (int(64+24*t),int(98+52*t),int(58+23*t))
    if height<=80:
        t=clamp((height-60)/20,0,1)
        return (int(108+16*t),int(73+18*t),int(58+20*t))
    if height>=105:
        t=clamp((height-100)/50,0,1)
        return (int(110+28*t),int(136+18*t),int(95+34*t))
    t=clamp((height-80)/25,0,1)
    return (int(124+2*t),int(109+29*t),int(83+17*t))

def text(draw, xy, value, fill=(227,232,237),font=None):
    draw.text(xy,str(value),fill=fill,font=font)

def main():
    p=argparse.ArgumentParser()
    p.add_argument("--legacy-world",type=Path,required=True)
    p.add_argument("--candidate-world",type=Path,required=True)
    p.add_argument("--output",type=Path,required=True)
    p.add_argument("--coverage",type=Path)
    args=p.parse_args()
    old=Anvil(args.legacy_world/"region")
    new=Anvil(args.candidate_world/"region")
    maps=[Image.new("RGB",SIZE,UNKNOWN),Image.new("RGB",SIZE,UNKNOWN)]
    delta=Image.new("RGB",SIZE,UNKNOWN)
    summary={"schemaVersion":1,"evidenceClass":"real_minecraft_full_anvil_world_surface_projection",
             "worlds":["v1.1.23","v1.1.24"],"xRange":[X0,X1],"zRange":[Z0,Z1],
             "measurement":"WORLD_SURFACE top Y and top block; can include vegetation, not natural substrate",
             "renderedScreenshot":False,
             "fullOld":0,"fullNew":0,"fullPaired":0,"unknownPaired":0,
             "changedTopHeight":0,"unchangedTopHeight":0,"oldLowLand":0,"newLowLand":0}
    pix=[m.load() for m in maps]; dp=delta.load()
    for zz,z in enumerate(range(Z0,Z1+1)):
        for xx,x in enumerate(range(X0,X1+1)):
            a,b=old.column(x,z),new.column(x,z)
            for i,row in enumerate((a,b)):
                if row["status"]=="full":
                    pix[i][xx,zz]=rgb_ground(row["topY"],row["topBlock"])
            summary["fullOld"]+=int(a["status"]=="full")
            summary["fullNew"]+=int(b["status"]=="full")
            if a["status"]!="full" or b["status"]!="full":
                summary["unknownPaired"]+=1
                continue
            summary["fullPaired"]+=1
            ad,bd=a["topY"],b["topY"]
            if ad==bd:
                summary["unchangedTopHeight"]+=1
                dp[xx,zz]=(65,88,78)
            else:
                summary["changedTopHeight"]+=1
                dy=bd-ad
                dp[xx,zz]=(199,112,69) if dy>0 else (75,130,188)
            if ad<=80:summary["oldLowLand"]+=1
            if bd<=80:summary["newLowLand"]+=1
    map_w=SIZE[0]*SCALE
    map_h=SIZE[1]*SCALE
    header=115
    footer=82
    gap=24
    width=3*map_w + 4*MARGIN + 2*gap
    height=header+map_h+footer
    target=Image.new("RGB",(width,height),BACKGROUND)
    draw=ImageDraw.Draw(target)
    f=ImageFont.load_default()
    text(draw,(MARGIN,16),"GeoWorld R8.1d  /  actual Minecraft Anvil block states",font=f)
    text(draw,(MARGIN,37),"Seed -1818694639628404530   |   1.21.1 NeoForge   |   fixed world coordinate crop",font=f)
    text(draw,(MARGIN,58),"Diagnostic plan-view, NOT a Minecraft gameplay screenshot. Gray = ungenerated / unverified.",fill=(202,196,162),font=f)
    titles=("OLD  v1.1.23","NEW  v1.1.24","WORLD_SURFACE height change")
    for i,m in enumerate([*maps,delta]):
        left=MARGIN+i*(map_w+gap+MARGIN)
        top=header
        text(draw,(left,header-24),titles[i],font=f)
        target.paste(m.resize((map_w,map_h),Image.Resampling.NEAREST),(left,top))
        draw.rectangle((left-1,top-1,left+map_w,top+map_h),outline=(156,169,179),width=1)
        # Key coordinates deliberately marked on both worlds, same transform.
        for (cx,cz,label,color) in [
            (-1600,5312,"#11 old pit",(255,204,111)),
            (-1699,5347,"#735 river",(255,118,118)),
        ]:
            px=left+(cx-X0)*SCALE
            py=top+(cz-Z0)*SCALE
            draw.ellipse((px-5,py-5,px+5,py+5),outline=color,width=2)
            if i<2:
                draw.text((px+7,py-10),label,fill=color,font=f)
        draw.text((left,top+map_h+9),f"X {X0}..{X1}  /  Z {Z0}..{Z1}",font=f,fill=(202,213,224))
    text(draw,(MARGIN,height-25),
         f"paired full columns {summary['fullPaired']}   |   paired unknown {summary['unknownPaired']}   |   changed WORLD_SURFACE Y {summary['changedTopHeight']}",font=f)
    args.output.parent.mkdir(parents=True,exist_ok=True)
    target.save(args.output,optimize=True)
    if args.coverage:
        args.coverage.parent.mkdir(parents=True,exist_ok=True)
        args.coverage.write_text(json.dumps(summary,ensure_ascii=False,indent=2)+"\n")
    print(json.dumps({**summary,"output":str(args.output)},ensure_ascii=False))

if __name__=="__main__":
    main()
