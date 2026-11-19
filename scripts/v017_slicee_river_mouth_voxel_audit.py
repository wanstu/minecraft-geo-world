#!/usr/bin/env python3
from pathlib import Path
import argparse, gzip, json, sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
from r81c_actual_mouth_voxel_audit import Anvil, HYDRO_SURFACE, sha256

def is_coral(name):
    return bool(name) and "coral" in name

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--world-root", required=True)
    ap.add_argument("--fixture", required=True)
    ap.add_argument("--output", required=True)
    args=ap.parse_args()

    world=Path(args.world_root)
    fixture=json.loads(Path(args.fixture).read_text())
    model=world/"data/geoworld/world.geoworld"
    with gzip.open(model,"rt") as f:
        manifest=json.load(f)["manifest"]
    assert manifest["generationRulesVersion"]=="geoworld-rules-v1.1.26"
    assert manifest["seed"]==fixture["seed"]

    A=Anvil(world/"region")
    target_chunks={(cx,cz) for cz in range(-159,-152) for cx in range(-160,-153)}
    full_chunks=0
    for cx,cz in sorted(target_chunks):
        if A.chunk(cx*16,cz*16):
            full_chunks+=1
    if full_chunks != len(target_chunks):
        raise AssertionError(f"incomplete target chunks {full_chunks}/{len(target_chunks)}")

    core=0
    core_shallow=0
    core_min=10**9
    coastal_wet=0
    coastal_coral=[]
    outside_coral=[]
    coastal_coral_count=0
    outside_coral_count=0
    warm_coastal_ocean=0
    checked_columns=0

    for row in fixture["rows"]:
        x,z=row["x"],row["z"]
        c=A.column(x,z)
        if c["status"]!="full":
            raise AssertionError(f"missing full column {x},{z}")
        checked_columns+=1
        wet=c.get("hydroTop",False)
        if row["coastal"] and wet:
            coastal_wet+=1
        if row["coastal"] and row["kind"]=="Ocean" and row["family"]=="Warm Ocean":
            warm_coastal_ocean+=1

        if row["kind"]=="River" and row["weight"]>=0.5:
            core+=1
            depth=c["topY"]-c["floorY"] if wet else 0
            core_min=min(core_min,depth)
            if depth<3:
                core_shallow+=1

        # Scan the actual saved water/floor column. This catches coral blocks
        # crossing in from an adjacent warm-ocean decoration feature, not only
        # biome-name mistakes.
        lo=max(-64,c["floorY"]-2)
        hi=min(319,c["topY"]+2)
        for y in range(lo,hi+1):
            name=A.block(x,z,y)
            if not is_coral(name):
                continue
            item={"x":x,"y":y,"z":z,"block":name,"coastal":row["coastal"],"kind":row["kind"],"family":row["family"]}
            if row["coastal"]:
                coastal_coral_count+=1
                if len(coastal_coral)<32: coastal_coral.append(item)
            else:
                outside_coral_count+=1
                if len(outside_coral)<32: outside_coral.append(item)

    out={
        "evidenceClass":"fresh_saved_minecraft_full_anvil_blockstate",
        "world":world.name,
        "seed":manifest["seed"],
        "rulesVersion":manifest["generationRulesVersion"],
        "modelSha256":sha256(model),
        "targetChunks":{"full":full_chunks,"total":len(target_chunks)},
        "columns":checked_columns,
        "navigationCore":{"columns":core,"minActualWaterBlocks":core_min,"shallowLt3":core_shallow},
        "mouthInfluence":{"wetColumns":coastal_wet,"warmOceanFamilyColumns":warm_coastal_ocean,
                          "coralBlocks":coastal_coral_count,"coralExamples":coastal_coral},
        "outsideInfluence":{"coralBlocks":outside_coral_count,"coralExamples":outside_coral},
        "pass":core_shallow==0 and coastal_coral_count==0 and full_chunks==len(target_chunks),
    }
    Path(args.output).write_text(json.dumps(out,ensure_ascii=False,indent=2)+"\n")
    print("Slice E actual mouth voxel:",
          f"chunks={full_chunks}/{len(target_chunks)}",
          f"core={core} minDepth={core_min} shallow={core_shallow}",
          f"coastalWet={coastal_wet} warmFamily={warm_coastal_ocean}",
          f"coastalCoral={coastal_coral_count} outsideCoral={outside_coral_count}")
    if not out["pass"]:
        raise AssertionError("Slice E actual mouth voxel acceptance failed")

if __name__=="__main__":
    main()
