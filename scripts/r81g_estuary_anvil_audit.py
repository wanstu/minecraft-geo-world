#!/usr/bin/env python3
"""GeoWorld R8.1g v1.1.26 river-to-sea real Anvil water step audit.

This is NOT an offline model projection: requires saved Minecraft
minecraft:full chunks and reads actual block states at each water column.
"""
import argparse
import hashlib
import json
from pathlib import Path

from r81_inspect_voxels import Anvil
from r81d_sinkhole_voxel_audit import actual_voxel
from verify_geoworld_voxel_world import inspect

RULES="geoworld-rules-v1.1.26"
SEED=123456
X0,X1,Z0,Z1=10804,10840,2465,2505

def inspect_world(path:Path):
    binding=inspect(path,RULES,SEED)
    if not binding["pass"]:
        raise ValueError("Minecraft/GeoWorld world binding mismatch: "+str(binding))
    anvil=Anvil(path/"region")
    rows={}
    for z in range(Z0,Z1+1):
        for x in range(X0,X1+1):
            a=actual_voxel(anvil,x,z)
            rows[(x,z)]={
                "x":x,"z":z,"status":a["status"],
                "waterTopY":a.get("waterTopY"),
                "topY":a.get("topY"),
                "topBlock":a.get("topBlock"),
                "naturalSubstrateY":a.get("naturalSubstrateY")
            }
    missing=[v for v in rows.values() if v["status"]!="full"]
    wet_pairs=[]
    violations=[]
    dry_adjacent_exposure=[]
    for (x,z),a in rows.items():
        for dx,dz in ((1,0),(0,1)):
            b=rows.get((x+dx,z+dz))
            if b is None or a["status"]!="full" or b["status"]!="full":continue
            aw,bw=a["waterTopY"],b["waterTopY"]
            if aw is not None and bw is not None:
                diff=abs(aw-bw)
                item={"a":[x,z,aw],"b":[x+dx,z+dz,bw],"deltaY":diff}
                wet_pairs.append(item)
                if diff>1:violations.append(item)
            elif aw is not None and bw is None and b["naturalSubstrateY"] is not None:
                if aw>b["naturalSubstrateY"]+1:
                    dry_adjacent_exposure.append({"a":[x,z,aw],"b":[x+dx,z+dz,b["naturalSubstrateY"]]})
            elif bw is not None and aw is None and a["naturalSubstrateY"] is not None:
                if bw>a["naturalSubstrateY"]+1:
                    dry_adjacent_exposure.append({"a":[x+dx,z+dz,bw],"b":[x,z,a["naturalSubstrateY"]]})
    targetA=rows[(10814,2467)]
    targetB=rows[(10815,2467)]
    targets={k:rows[k] for k in ((10814,2467),(10815,2467),(10808,2474),
                                   (10809,2474),(10838,2504),(10839,2504))}
    result={
       "schemaVersion":1,
       "evidenceClass":"saved_minecraft_neoforge_anvil_full_block_states",
       "seed":SEED,"rulesVersion":RULES,
       "region":{"x0":X0,"x1":X1,"z0":Z0,"z1":Z1},
       "preflight":binding,
       "worldModelSha256":hashlib.sha256(
           (path/"data/geoworld/world.geoworld").read_bytes()).hexdigest(),
       "summary":{
          "expectedFullColumns":(X1-X0+1)*(Z1-Z0+1),
          "actualFullColumns":len(rows)-len(missing),
          "missingOrIncompleteColumns":len(missing),
          "adjacentWetPairs":len(wet_pairs),
          "wetWaterSurfaceStepAboveOneBlock":len(violations),
          "waterToLowerDryExposureSignals":len(dry_adjacent_exposure),
          "originalFailurePairWaterY":[targetA["waterTopY"],targetB["waterTopY"]],
          "originalFailurePairContinuous":(
             targetA["waterTopY"] is not None and targetB["waterTopY"] is not None and
             abs(targetA["waterTopY"]-targetB["waterTopY"])<=1)
       },
       "selectedColumns":list(targets.values()),
       "incompleteColumns":missing,
       "waterSurfaceViolations":violations,
       "dryExposureExamples":dry_adjacent_exposure[:30],
       "allSampledColumns":list(rows.values()),
       "gameCameraScreenshot":False,
       "releaseEligible":False,
    }
    return result

def main():
 p=argparse.ArgumentParser(description=__doc__)
 p.add_argument("--world",type=Path,required=True)
 p.add_argument("--output",type=Path,required=True)
 args=p.parse_args()
 d=inspect_world(args.world)
 args.output.parent.mkdir(exist_ok=True,parents=True)
 args.output.write_text(json.dumps(d,ensure_ascii=False,indent=2)+"\n")
 print(json.dumps(d["summary"],ensure_ascii=False,indent=2))
 if d["summary"]["missingOrIncompleteColumns"] or d["summary"]["wetWaterSurfaceStepAboveOneBlock"] or not d["summary"]["originalFailurePairContinuous"]:
  raise SystemExit("R8.1g actual Minecraft water columns failed acceptance")

if __name__=="__main__":
 main()
