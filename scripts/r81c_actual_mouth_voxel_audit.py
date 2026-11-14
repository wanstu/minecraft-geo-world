#!/usr/bin/env python3
"""R8.1c read-only audit of actual Minecraft Anvil mouth voxels."""
from pathlib import Path
import argparse, gzip, hashlib, json, math, struct, zlib

WATER_TOP = {
    "minecraft:water", "minecraft:seagrass", "minecraft:tall_seagrass",
    "minecraft:kelp", "minecraft:kelp_plant", "minecraft:bubble_column",
}
HYDRO_SURFACE = WATER_TOP | {
    "minecraft:ice", "minecraft:blue_ice", "minecraft:packed_ice",
    "minecraft:frosted_ice",
}

class NBT:
    def __init__(self, buf):
        self.b = memoryview(buf); self.pos = 0
    def n(self, sz=4, signed=True):
        v = int.from_bytes(self.b[self.pos:self.pos+sz], "big", signed=signed)
        self.pos += sz; return v
    def txt(self):
        length = self.n(2, False)
        v = bytes(self.b[self.pos:self.pos+length]).decode("utf-8", "replace")
        self.pos += length; return v
    def skip(self, n):
        self.pos += n; return None
    def tag(self, t, keep=False):
        if t == 0: return None
        if t in (1,2,3,4):
            return self.n({1:1,2:2,3:4,4:8}[t]) if keep else self.skip({1:1,2:2,3:4,4:8}[t])
        if t in (5,6):
            sz = 4 if t == 5 else 8
            v = struct.unpack_from(">f" if sz == 4 else ">d", self.b, self.pos)[0] if keep else None
            self.pos += sz; return v
        if t == 7:
            length = self.n(); v = bytes(self.b[self.pos:self.pos+length]) if keep else None
            self.pos += length; return v
        if t == 8:
            if keep: return self.txt()
            length = self.n(2, False); self.pos += length; return None
        if t == 9:
            typ = self.n(1, False); length = self.n()
            if length < 0 or length > 3_000_000: raise ValueError("bad list size")
            if keep: return [self.tag(typ, True) for _ in range(length)]
            for _ in range(length): self.tag(typ, False)
            return None
        if t == 10:
            out = {} if keep else None
            while True:
                child = self.n(1, False)
                if child == 0: break
                name = self.txt(); value = self.tag(child, keep)
                if keep: out[name] = value
            return out
        if t in (11,12):
            length = self.n(); sz = 4 if t == 11 else 8
            if length < 0 or length > 3_000_000: raise ValueError("bad array size")
            vals = [int.from_bytes(self.b[self.pos+i*sz:self.pos+(i+1)*sz], "big", signed=False)
                    for i in range(length)] if keep else None
            self.pos += length * sz; return vals
        raise ValueError(f"NBT tag type {t}")

def read_chunk(path, cx, cz):
    rx, rz = cx // 32, cz // 32
    region = path / f"r.{rx}.{rz}.mca"
    if not region.exists() or region.stat().st_size < 8192: return None
    slot = (cz % 32) * 32 + (cx % 32)
    with region.open("rb") as fp:
        fp.seek(slot * 4); loc = fp.read(4)
        if len(loc) != 4: return None
        sector = int.from_bytes(loc[:3], "big")
        if not sector: return None
        fp.seek(sector * 4096)
        length = int.from_bytes(fp.read(4), "big")
        if not (2 <= length <= 10_000_000): return None
        method = fp.read(1)[0]; body = fp.read(length - 1)
    raw = zlib.decompress(body) if method == 2 else gzip.decompress(body) if method == 1 else body if method == 3 else None
    if raw is None: return None
    n = NBT(raw)
    if n.n(1, False) != 10: raise ValueError("root not compound")
    n.txt(); tags = {}
    keep = {"sections","Heightmaps","Status","xPos","zPos"}
    while True:
        t = n.n(1, False)
        if not t: break
        name = n.txt(); value = n.tag(t, name in keep)
        if name in keep: tags[name] = value
    if tags.get("Status") != "minecraft:full" or tags.get("xPos") != cx or tags.get("zPos") != cz:
        return None
    return tags

def unpack_height(words):
    if not words or len(words) != math.ceil(256 / 7): return None
    return [((words[j//7] >> ((j%7)*9)) & 511) - 65 for j in range(256)]

def block_at(sections, y, x, z):
    sec = sections.get(y // 16)
    if sec is None: return None
    states = sec.get("block_states") or {}
    palette = states.get("palette") or []
    if not palette: return None
    if len(palette) == 1: return palette[0].get("Name")
    bits = max(4, (len(palette)-1).bit_length())
    words = states.get("data") or []
    if not words: return None
    per = 64 // bits
    j = ((y % 16) * 16 + (z % 16)) * 16 + (x % 16)
    idx = (words[j//per] >> ((j%per)*bits)) & ((1 << bits) - 1)
    return palette[idx].get("Name") if idx < len(palette) else None

class Anvil:
    def __init__(self, region_root):
        self.root = Path(region_root); self.cache = {}
    def chunk(self, x, z):
        key = (x // 16, z // 16)
        if key not in self.cache: self.cache[key] = read_chunk(self.root, *key)
        return self.cache[key]
    def block(self, x, z, y):
        n = self.chunk(x, z)
        if not n: return None
        sections = {s.get("Y"): s for s in n.get("sections", [])}
        return block_at(sections, y, x, z)
    def column(self, x, z):
        n = self.chunk(x, z)
        if not n: return {"x":x,"z":z,"status":"not_generated"}
        hm = n.get("Heightmaps") or {}
        top = unpack_height(hm.get("WORLD_SURFACE")); floor = unpack_height(hm.get("OCEAN_FLOOR"))
        if top is None or floor is None: return {"x":x,"z":z,"status":"missing_heightmap"}
        j = (z % 16) * 16 + (x % 16); y = top[j]
        name = self.block(x, z, y)
        return {"x":x,"z":z,"status":"full","topY":y,"floorY":floor[j],
                "topBlock":name,"wet":name in WATER_TOP,"hydroTop":name in HYDRO_SURFACE}

def sha256(path):
    h = hashlib.sha256()
    with Path(path).open("rb") as f:
        for b in iter(lambda:f.read(1024*1024), b""): h.update(b)
    return h.hexdigest()

def scan_wet_adjacency(anvil, bounds):
    x0,z0,x1,z1=bounds
    columns={}; full=missing=wet=0
    for z in range(z0,z1):
        for x in range(x0,x1):
            c=anvil.column(x,z); columns[(x,z)]=c
            if c["status"]=="full":
                full+=1
                if c.get("wet"): wet+=1
            else:
                missing+=1
    max_drop=walls=0; examples=[]
    for z in range(z0,z1):
        for x in range(x0,x1):
            a=columns[(x,z)]
            if a["status"]!="full" or not a.get("wet"): continue
            for dx,dz in ((1,0),(0,1)):
                nx,nz=x+dx,z+dz
                if nx>=x1 or nz>=z1: continue
                b=columns[(nx,nz)]
                if b["status"]!="full" or not b.get("wet"): continue
                drop=abs(a["topY"]-b["topY"])
                max_drop=max(max_drop,drop)
                if drop>1:
                    walls+=1
                    if len(examples)<12:
                        examples.append({
                            "a":[x,z,a["topY"],a["floorY"],a["topBlock"]],
                            "b":[nx,nz,b["topY"],b["floorY"],b["topBlock"]],
                            "drop":drop,
                        })
    return {
        "bounds":bounds,"fullColumns":full,"missingColumns":missing,
        "wetColumns":wet,"maxAdjacentWetTopDrop":max_drop,
        "waterWallsGt1":walls,"examples":examples,
    }

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument("--world-root", required=True)
    ap.add_argument("--output", required=True)
    args=ap.parse_args()
    world=Path(args.world_root); region=world/"region"
    model=world/"data/geoworld/world.geoworld"
    with gzip.open(model,"rt") as f: manifest=json.load(f)["manifest"]
    assert manifest["generationRulesVersion"] == "geoworld-rules-v1.1.23"
    assert manifest["seed"] == -1818694639628404530
    A=Anvil(region)

    anchors509=[(-1088,-960),(-1078,-992),(-1069,-1024),(-1070,-1027),
                (-1073,-1038),(-1078,-1056),(-1084,-1069),(-1088,-1088)]
    a509=[A.column(x,z) for x,z in anchors509]
    exact509=[A.column(-1069,-1026),A.column(-1069,-1027)]
    assert all(c["status"]=="full" and c["wet"] and c["topY"]==62 for c in a509)
    assert all(c["status"]=="full" and c["wet"] and c["topY"]==62 for c in exact509)

    cross509=[A.column(x,-1027) for x in range(-1168,-991)]
    full509=[c for c in cross509 if c["status"]=="full"]
    exposed=[]
    for a,b in zip(full509,full509[1:]):
        if b["x"] != a["x"]+1: continue
        if a["wet"] != b["wet"]:
            water=a if a["wet"] else b; dry=b if a["wet"] else a
            exposed.append({"waterX":water["x"],"waterTopY":water["topY"],
                            "dryX":dry["x"],"dryTopY":dry["topY"],
                            "drop":water["topY"]-dry["topY"]})
    assert exposed and max(x["drop"] for x in exposed) <= 1

    long915=[]
    for x in range(1640,1701):
        row=A.column(x,10432)
        row["y62"]=A.block(x,10432,62); row["y61"]=A.block(x,10432,61)
        long915.append(row)
    assert all(r["status"]=="full" for r in long915)
    assert all(r["y62"] in HYDRO_SURFACE and r["y61"] in HYDRO_SURFACE for r in long915)

    cross915=[]
    for z in range(10400,10465):
        row=A.column(1664,z)
        row["y62"]=A.block(1664,z,62); row["y61"]=A.block(1664,z,61)
        cross915.append(row)
    assert all(r["status"]=="full" for r in cross915)
    assert all(r["y62"] in HYDRO_SURFACE and r["y61"] in HYDRO_SURFACE for r in cross915)

    area509=scan_wet_adjacency(A,[-1136,-1104,-1008,-976])
    area915=scan_wet_adjacency(A,[1632,10400,1712,10480])
    assert area509["missingColumns"]==0 and area915["missingColumns"]==0
    assert area509["waterWallsGt1"]==0, (
        f'#509 still has {area509["waterWallsGt1"]} adjacent wet pairs '
        f'with >1 block water-top drop'
    )
    assert area915["waterWallsGt1"]==0, (
        f'#915 still has {area915["waterWallsGt1"]} adjacent wet pairs '
        f'with >1 block water-top drop'
    )

    r509=region/"r.-3.-3.mca"; r915=region/"r.3.20.mca"
    out={
      "evidenceClass":"saved_minecraft_full_anvil_blockstate",
      "world":world.name,"seed":manifest["seed"],"rulesVersion":manifest["generationRulesVersion"],
      "modelSha256":sha256(model),
      "regionSha256":{"r.-3.-3.mca":sha256(r509),"r.3.20.mca":sha256(r915)},
      "mouth509":{"oldDefect":[[-1069,-1026],[-1069,-1027]],"oldRiseBlocks":1,
                  "actualExact":exact509,"centerlineAnchors":a509,
                  "crossSectionTransitions":exposed,
                  "maxDrySideExposedDrop":max(x["drop"] for x in exposed)},
      "mouth915":{"oldModelRiseBlocks":4,"oldModelT":0.5,"center":[1664,10432],
                  "longitudinal":{"xMin":1640,"xMax":1700,"z":10432,"columns":long915},
                  "crossSection":{"x":1664,"zMin":10400,"zMax":10464,"columns":cross915},
                  "continuousSeaLevelHydrologyY61Y62":True},
      "fullAreaWetAdjacency":{"mouth509":area509,"mouth915":area915},
      "summary":{"mouth509Fixed":True,"mouth509NoVerticalBankWall":True,
                 "mouth509WaterWallsGt1":area509["waterWallsGt1"],
                 "mouth509MaxAdjacentWetTopDrop":area509["maxAdjacentWetTopDrop"],
                 "mouth915FourBlockRiseAbsent":True,"mouth915FrozenWaterContinuous":True,
                 "mouth915WaterWallsGt1":area915["waterWallsGt1"],
                 "mouth915MaxAdjacentWetTopDrop":area915["maxAdjacentWetTopDrop"],
                 "pass":True}
    }
    Path(args.output).write_text(json.dumps(out,ensure_ascii=False,indent=2)+"\n")
    print("R8.1c actual voxel PASS: #509 full-area waterWallsGt1=0; #915 full-area waterWallsGt1=0; bank wall max drop <=1")

if __name__=="__main__": main()
