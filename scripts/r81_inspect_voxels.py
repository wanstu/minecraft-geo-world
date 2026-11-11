#!/usr/bin/env python3
"""Read-only Minecraft Anvil block-state, water surface and section helpers."""
from pathlib import Path
import sys,math,struct,zlib,gzip,json,argparse
from r81_anvil_reader import NBT,unpack_height

wetnames={'minecraft:water','minecraft:seagrass','minecraft:tall_seagrass','minecraft:kelp','minecraft:kelp_plant','minecraft:bubble_column'}

def block_at(sec,y,x,z):
 v=sec.get(y//16)
 if v is None:return None
 bs=v.get('block_states') or {};pal=bs.get('palette') or []
 if not pal:return None
 if len(pal)==1:return pal[0].get('Name')
 bits=max(4,(len(pal)-1).bit_length())
 words=bs.get('data') or []
 if not words:return None
 per=64//bits;j=((y%16)*16+(z%16))*16+(x%16)
 p=(words[j//per]>>((j%per)*bits))&((1<<bits)-1)
 if p>=len(pal):return None
 return pal[p].get('Name')

def read_chunk(fp,slot):
 fp.seek(slot*4)
 loc=fp.read(4)
 if len(loc)!=4:return None
 sector=int.from_bytes(loc[:3],'big')
 if not sector:return None
 fp.seek(sector*4096);length=int.from_bytes(fp.read(4),'big')
 if not (2<=length<=10_000_000):return None
 method=fp.read(1)[0];body=fp.read(length-1)
 raw=zlib.decompress(body) if method==2 else gzip.decompress(body) if method==1 else body if method==3 else None
 if raw is None:return None
 n=NBT(raw);assert n.n(1,False)==10;n.txt()
 tags={}
 while True:
  t=n.n(1,False)
  if not t:break
  k=n.txt();v=n.tag(t,k in ('sections','Heightmaps','Status','xPos','zPos'))
  if k in ('sections','Heightmaps','Status','xPos','zPos'):tags[k]=v
 return tags

class Anvil:
 def __init__(self,root):
  self.root=Path(root)
  self.regions={}
  self.cache={}
 def column(self,x,z):
  x,z=int(x),int(z)
  cx,cz=x//16,z//16
  key=(cx,cz)
  if key not in self.cache:
   rx,rz=cx//32,cz//32
   r=self.root/f'r.{rx}.{rz}.mca'
   if not r.exists() or r.stat().st_size<8192:self.cache[key]=None
   else:
    with r.open('rb') as f:
     try:n=read_chunk(f,(cz%32)*32+(cx%32))
     except Exception as e:print(f'WARN {r} {key} {e}',file=sys.stderr);n=None
    self.cache[key]=n if n is not None and n.get('Status')=='minecraft:full' and n.get('xPos')==cx and n.get('zPos')==cz else None
  n=self.cache[key]
  if n is None:return {'x':x,'z':z,'status':'not_generated'}
  hm=n.get('Heightmaps') or {}
  top=unpack_height(hm.get('WORLD_SURFACE'))
  floor=unpack_height(hm.get('OCEAN_FLOOR'))
  if top is None or floor is None:return {'x':x,'z':z,'status':'missing_heightmap'}
  j=(z%16)*16+(x%16);y=top[j]
  sec={a.get('Y'):a for a in n.get('sections',[])}
  name=block_at(sec,y,x,z)
  lower=[(yy,block_at(sec,yy,x,z)) for yy in range(y,y-4,-1)]
  return {'x':x,'z':z,'status':'full','topY':y,'floorY':floor[j],'topBlock':name,'wet':name in wetnames,
    'seaY62Block':block_at(sec,62,x,z), 'seaY61Block':block_at(sec,61,x,z),
    'seaY60Block':block_at(sec,60,x,z), 'verticalTop':lower}
