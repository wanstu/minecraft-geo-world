#!/usr/bin/env python3
"""Validate a locally assembled GeoWorld adapter jar; does not publish it."""
import argparse
import hashlib
import json
from pathlib import Path
from zipfile import BadZipFile, ZipFile

NATIVE = (
    "native/windows-x64/geoworld-engine.exe",
    "native/windows-arm64/geoworld-engine.exe",
    "native/linux-x64/geoworld-engine",
    "native/linux-arm64/geoworld-engine",
    "native/macos-x64/geoworld-engine",
    "native/macos-arm64/geoworld-engine",
)
METADATA = {
    "fabric":"fabric.mod.json",
    "neoforge":"META-INF/neoforge.mods.toml",
}

def verify_jar(path:Path,loader:str,version:str)->dict:
    if not path.is_file():
        return {"passed":False,"error":"jar not found","jar":str(path)}
    try:
        with ZipFile(path) as jar:
            entries=set(jar.namelist())
            need=[METADATA[loader],*NATIVE,
                  "geoworld/runtime/GeoWorldV1.class"]
            missing=[n for n in need if n not in entries]
            actual_metadata=None
            version_ok=False
            if METADATA[loader] in entries:
                source=jar.read(METADATA[loader]).decode("utf-8","replace")
                if loader=="fabric":
                    parsed=json.loads(source)
                    actual_metadata={
                        "id":parsed.get("id"),
                        "version":parsed.get("version"),
                    }
                    version_ok=parsed.get("version")==version
                else:
                    actual_metadata={
                        "modVersionLine":[x.strip() for x in source.splitlines()
                                          if x.lstrip().startswith("version=")]
                    }
                    version_ok=any(x.strip().strip("'\"")==version
                                   for x in (line.split("=",1)[1]
                                     for line in source.splitlines()
                                     if line.lstrip().startswith("version=")))
            return {
                "jar":str(path),"loader":loader,"expectedVersion":version,
                "sha256":hashlib.sha256(path.read_bytes()).hexdigest(),
                "bytes":path.stat().st_size,
                "allSixNativeEngines":all(x in entries for x in NATIVE),
                "missingEntries":missing,
                "metadata":actual_metadata,
                "versionMatches":version_ok,
                "passed":not missing and version_ok,
                "note":"Local build verification, not a published release",
            }
    except (OSError,BadZipFile,ValueError) as exc:
        return {"passed":False,"error":str(exc),"jar":str(path)}

def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument("--jar",type=Path,required=True)
    p.add_argument("--loader",choices=tuple(METADATA),required=True)
    p.add_argument("--expected-version",required=True)
    p.add_argument("--output",type=Path)
    args=p.parse_args()
    result=verify_jar(args.jar,args.loader,args.expected_version)
    if args.output:
        args.output.parent.mkdir(exist_ok=True,parents=True)
        args.output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+"\n")
    print(json.dumps(result,ensure_ascii=False,indent=2))
    raise SystemExit(0 if result["passed"] else 2)

if __name__=="__main__":
    main()
