#!/usr/bin/env bash
# GeoWorld developer prerequisites; intentionally read-only.
set -uo pipefail
fail=0
check() {
  local name="$1"; shift
  if "$@" >/dev/null 2>&1; then printf 'OK   %s\n' "$name"; else printf 'MISS %s\n' "$name"; fail=1; fi
}
check 'Git' command -v git
check 'Go (>=1.26)' bash -c 'v=$(go version 2>/dev/null | sed -nE "s/.*go([0-9]+)\.([0-9]+).*/\1 \2/p"); read -r major minor <<< "$v"; [[ "${major:-0}" -gt 1 ]] || { [[ "${major:-0}" -eq 1 && "${minor:-0}" -ge 26 ]]; }'
check 'JDK 21 (Gradle 1.21.1)' bash -c 'command -v java >/dev/null && java -version 2>&1 | grep -Eq "version \"21[.\"]|openjdk 21"'
if command -v java >/dev/null 2>&1; then
  printf 'INFO java: %s\n' "$(java -version 2>&1 | head -1)"
fi
check 'Java compiler' command -v javac
check 'curl (Gradle/Go downloads)' command -v curl
if [[ -f java/neoforge-mod/gradlew ]]; then
  printf 'OK   Gradle wrapper present: java/neoforge-mod/gradlew\n'
else
  printf 'MISS Gradle wrapper java/neoforge-mod/gradlew\n'; fail=1
fi
if (( fail != 0 )); then
  printf '\nDevelopment prerequisites are missing; do not report tests as passed.\n'
  exit 1
fi
printf '\nPrerequisite check passed; run Go/Gradle tests separately.\n'