#!/usr/bin/env bash
# Tests scripts/base-spec.sh in a throw-away git repo.  Usage: scripts/test-base-spec.sh
set -uo pipefail
SCRIPT="$(cd "$(dirname "$0")" && pwd)/base-spec.sh"
WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT
cd "$WORK"
git init -q -b main && git config user.email t@t && git config user.name t
mkdir api && echo "version: one" > api/openapi.yaml && git add -A && git commit -qm base
BASE=$(git rev-parse HEAD)
echo "version: two" > api/openapi.yaml && git commit -qam change

fail=0
check() { # check <name> <expected-exit> <actual-exit>
  if [ "$2" -eq "$3" ]; then echo "ok   $1"; else echo "FAIL $1 (exit $3, want $2)"; fail=1; fi
}

"$SCRIPT" "$BASE" api/openapi.yaml out.yaml 2>/dev/null; check "extracts spec from base ref" 0 $?
[ "$(cat out.yaml)" = "version: one" ] && echo "ok   extracted the OLD content" || { echo "FAIL wrong content"; fail=1; }
"$SCRIPT" "$BASE" api/missing.yaml out2.yaml 2>/dev/null; check "spec absent at base -> exit 3 (skip)" 3 $?
"$SCRIPT" deadbeef api/openapi.yaml out3.yaml 2>/dev/null; check "unknown ref -> exit 2" 2 $?
"$SCRIPT" onlyone 2>/dev/null;                             check "bad usage -> exit 2" 2 $?
exit $fail
