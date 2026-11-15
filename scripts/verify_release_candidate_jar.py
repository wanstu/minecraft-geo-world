#!/usr/bin/env python3
"""Verify one locally built GeoWorld release-candidate JAR."""
import json
import re
import sys
import zipfile
from pathlib import Path

NATIVE_ENGINES = [
    "native/windows-x64/geoworld-engine.exe",
    "native/windows-arm64/geoworld-engine.exe",
    "native/linux-x64/geoworld-engine",
    "native/linux-arm64/geoworld-engine",
    "native/macos-x64/geoworld-engine",
    "native/macos-arm64/geoworld-engine",
]

def verify(module: str, loader: str, version: str) -> None:
    lib = Path(module) / "build/libs"
    jars = [
        p for p in lib.glob("*.jar")
        if not re.search(r"(sources|javadoc|dev|shadow)", p.name)
    ]
    assert len(jars) == 1, (
        f"{module}: expected 1 production jar, got {[p.name for p in jars]}"
    )
    jar = jars[0]
    metadata = (
        "fabric.mod.json" if loader == "fabric"
        else "META-INF/neoforge.mods.toml"
    )
    with zipfile.ZipFile(jar) as zf:
        names = set(zf.namelist())
        assert metadata in names, (jar, metadata)
        for native in NATIVE_ENGINES:
            assert native in names, (jar, native)
            assert zf.getinfo(native).file_size > 0, (jar, native, "empty")
        meta = zf.read(metadata).decode("utf-8")
        if loader == "fabric":
            assert json.loads(meta).get("version") == version
        elif loader == "neoforge":
            assert re.search(
                r"(?m)^version\s*=\s*['\"]" + re.escape(version) + r"['\"]",
                meta,
            ), jar
        else:
            raise AssertionError(f"unknown loader: {loader}")
    print(
        f"PASS {module} {loader}: {jar.name} bytes={jar.stat().st_size} "
        f"version={version} six native engines"
    )

if __name__ == "__main__":
    assert len(sys.argv) == 4, (
        "usage: verify_release_candidate_jar.py MODULE LOADER VERSION"
    )
    verify(*sys.argv[1:])
