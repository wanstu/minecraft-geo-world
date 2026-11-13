#!/usr/bin/env python3
"""Verify one locally built v0.1.5 candidate JAR has release metadata and bundled native engines."""
import json
import re
import sys
import zipfile
from pathlib import Path

def verify(module: str, loader: str, version: str):
    lib=Path(module)/"build/libs"
    jars=[p for p in lib.glob("*.jar") if not re.search(r"(sources|javadoc|dev|shadow)",p.name)]
    assert len(jars)==1,f"{module}: expected 1 production jar, got {[p.name for p in jars]}"
    jar=jars[0]
    expected=["native/windows-x64/geoworld-engine.exe",
              "native/windows-arm64/geoworld-engine.exe",
              "native/linux-x64/geoworld-engine",
              "native/linux-arm64/geoworld-engine",
              "native/macos-x64/geoworld-engine",
              "native/macos-arm64/geoworld-engine"]
    with zipfile.ZipFile(jar) as z:
        names=set(z.namelist())
        metadata="fabric.mod.json" if loader=="fabric" else "META-INF/neoforge.mods.toml"
        assert metadata in names,(jar,metadata)
        for native in expected:
            assert native in names,(jar,native)
            assert z.getinfo(native).file_size>0,(jar,native,"empty")
        meta=z.read(metadata).decode("utf-8")
        if loader=="fabric":
            entry=json.loads(meta)
            assert entry.get("version")==version,(jar,entry.get("version"),version)
        else:
            # Official NeoForge metadata contains a literal version property.
            assert re.search(r"(?m)^version\s*=\s*['\"]"+re.escape(version)+r"['\"]",meta),jar
    print(f"PASS {module} {loader}: {jar.name} bytes={jar.stat().st_size} version={version} six native engines")

if __name__=="__main__":
    assert len(sys.argv)==4,"usage: verify_v015_candidate_jar.py MODULE LOADER VERSION"
    verify(*sys.argv[1:])
