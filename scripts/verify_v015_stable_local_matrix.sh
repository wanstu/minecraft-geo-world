#!/usr/bin/env bash
# Real clean/check/build for all nine supported MC adapters, with candidate version metadata.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOMAXPROCS=2
OUT="output/v015-stable-local-matrix"
mkdir -p "$OUT"
printf 'minecraft\tloader\tjava\tmodule\tresult\n' > "$OUT/results.tsv"
while read -r mc loader jdk module; do
  [[ -n "${mc:-}" ]] || continue
  echo "START $mc $loader JDK$jdk ($module)"
  log="$OUT/${mc}-${loader}.log"
  if (
    export JAVA_HOME="/home/wanstu/sdk/jdk/$jdk"
    export PATH="$JAVA_HOME/bin:$PATH"
    export JAVA_TOOL_OPTIONS='-XX:ActiveProcessorCount=2'
    cd "$module"
    timeout 1200s bash ./gradlew --offline --no-daemon --max-workers=1 \
      clean check build -Pmod_version=0.1.5 \
      '-Dorg.gradle.jvmargs=-Xmx1400m -XX:MaxMetaspaceSize=650m'
  ) > "$log" 2>&1; then
    if python3 scripts/verify_v015_candidate_jar.py "$module" "$loader" 0.1.5 >> "$log" 2>&1; then
      printf '%s\t%s\t%s\t%s\tPASS\n' "$mc" "$loader" "$jdk" "$module" >> "$OUT/results.tsv"
      tail -n 6 "$log"
      echo "PASS $mc $loader"
    else
      tail -n 80 "$log"
      echo "FAIL JAR verification $mc $loader"
      exit 32
    fi
  else
    tail -n 100 "$log"
    echo "FAIL build $mc $loader"
    exit 31
  fi
done <<'TARGETS'
26.3 neoforge 25 java/neoforge-mod
26.3 fabric 25 java/adapters/26.3-fabric
26.2 neoforge 25 java/adapters/26.2-neoforge
26.2 fabric 25 java/adapters/26.2-fabric
26.1.2 neoforge 25 java/adapters/26.1.2-neoforge
26.1.2 fabric 25 java/adapters/26.1.2-fabric
1.21.1 neoforge 21 java/adapters/1.21.1-neoforge
1.21.1 fabric 21 java/adapters/1.21.1-fabric
1.20.1 fabric 17 java/adapters/1.20.1-fabric
TARGETS
echo 'NINE_TARGET_LOCAL_MATRIX_PASS'
cat "$OUT/results.tsv"
