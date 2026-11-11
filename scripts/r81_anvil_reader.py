from pathlib import Path
from collections import Counter
import struct,zlib,gzip,math,json,sys

class NBT:
 def __init__(self,buf): self.b=memoryview(buf); self.pos=0
 def n(self,sz=4,signed=True):
  v=int.from_bytes(self.b[self.pos:self.pos+sz],'big',signed=signed);self.pos+=sz;return v
 def txt(self):
  length=self.n(2,False);val=bytes(self.b[self.pos:self.pos+length]).decode('utf-8','replace');self.pos+=length;return val
 def tag(self,t,keep=False):
  if t==0:return None
  if t in [1,2,3,4]:
   return self.n({1:1,2:2,3:4,4:8}[t]) if keep else self.skip({1:1,2:2,3:4,4:8}[t])
  if t in [5,6]:
   sz=4 if t==5 else 8
   v=struct.unpack_from('>f' if sz==4 else '>d',self.b,self.pos)[0] if keep else None
   self.pos+=sz;return v
  if t==7:
   length=self.n();v=bytes(self.b[self.pos:self.pos+length]) if keep else None;self.pos+=length;return v
  if t==8:
   if keep:return self.txt()
   l=self.n(2,False);self.pos+=l;return None
  if t==9:
   type_=self.n(1,False);length=self.n()
   if length<0 or length>3_000_000:raise ValueError("bad list size")
   if keep:return [self.tag(type_,True) for i in range(length)]
   for i in range(length):self.tag(type_,False)
   return None
  if t==10:
   result={} if keep else None
   while True:
    child=self.n(1,False)
    if child==0:break
    name=self.txt();val=self.tag(child,keep)
    if keep:result[name]=val
   return result
  if t in (11,12):
   length=self.n(); sz=4 if t==11 else 8
   if length<0 or length>3_000_000:raise ValueError("bad array size")
   if keep:
    vals=[int.from_bytes(self.b[self.pos+i*sz:self.pos+(i+1)*sz],'big',signed=False) for i in range(length)]
   else:vals=None
   self.pos+=length*sz;return vals
  raise ValueError(f"NBT tag type {t}")
 def skip(self,n):self.pos+=n;return None
 def root(self):
  if self.n(1,False)!=10:raise ValueError("root not compound")
  self.txt()
  result={}
  while True:
   t=self.n(1,False)
   if not t:break
   name=self.txt()
   result[name]=self.tag(t,name in ('Heightmaps','Status','xPos','zPos'))
  return result


def unpack_height(words):
 if not words or len(words)!=math.ceil(256/7):return None
 return [((words[j//7]>>((j%7)*9)) & 511)-65 for j in range(256)]

