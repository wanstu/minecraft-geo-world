#!/usr/bin/env python3
"""Read-only GeoWorld RC preflight for every CI Minecraft adapter target.

This validates matrix metadata and installed JDKs, NOT compilation or release.
"""
import json
import re
from pathlib import Path
from datetime import date

ROOT=Path(__file__).resolve().parent.parent
WORKFLOW=ROOT/".github/workflows/release.yml"
pattern=re.compile(
    r"- \{ minecraft: '([^']+)', loader: (fabric|neoforge), "
    r"java: '([^']+)', module: '([^']+)' \}"
)
raw=pattern.findall(WORKFLOW.read_text(encoding="utf-8"))
targets=list(dict.fromkeys(raw))
if len(targets)!=9:
    raise SystemExit(f"Expected 9 release targets, found {len(targets)}: {targets}")
results=[]
for mc,loader,java,module in targets:
    d=ROOT/module
    properties=d/"gradle.properties"
    wrapper=d/"gradlew"
    if not properties.is_file() or not wrapper.is_file():
        raise SystemExit(f"Missing module Gradle config: {module}")
    props=dict(line.split("=",1) for line in properties.read_text().splitlines()
               if line.strip() and not line.lstrip().startswith("#") and "=" in line)
    got_mc=props.get("minecraft_version")
    if got_mc!=mc:
        raise SystemExit(f"Wrong Minecraft target: {module} {got_mc!r}, CI expects {mc!r}")
    # NeoForge 26.3 uses the separate root project; other targets use adapters.
    metadata=(d/"src/main/resources/fabric.mod.json" if loader=="fabric"
              else d/"src/main/templates/META-INF/neoforge.mods.toml")
    if not metadata.is_file():
        raise SystemExit(f"Missing source loader metadata: {metadata}")
    jdk=Path("/home/wanstu/sdk/jdk")/java
    results.append({
        "minecraft":mc,"loader":loader,"java":java,"module":module,
        "configuredMinecraftVersion":got_mc,"configValid":True,
        "gradleWrapperExists":True,"sourceLoaderMetadataExists":True,
        "localJdkAvailable":(jdk/"bin/java").is_file(),
        "note":"Static target validation only, no compile/build implied",
    })
report={"schemaVersion":1,"releaseTargetCount":len(results),
        "workflow":".github/workflows/release.yml",
        "targetStaticCheck":"PASS","targets":results}
dest=ROOT/"docs/acceptance/r815_rc_build_matrix_preflight.json"
dest.parent.mkdir(exist_ok=True,parents=True)
dest.write_text(json.dumps(report,indent=2,ensure_ascii=False)+"\n",encoding="utf-8")
print(json.dumps({
    "targetStaticCheck":"PASS","releaseTargetCount":len(results),
    "withLocalJdk":sum(t["localJdkAvailable"] for t in results),
    "missingLocalJdk":[t["module"] for t in results if not t["localJdkAvailable"]],
},ensure_ascii=False))
