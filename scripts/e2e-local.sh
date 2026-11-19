#!/usr/bin/env bash
# Rehearses the whole GitHub Action locally, with no GitHub involved:
#   scratch git repo -> base-spec.sh -> guardian comment -> fake GitHub API
# Usage: scripts/e2e-local.sh            (builds guardian itself)
#        GUARDIAN_BIN=/path/to/guardian scripts/e2e-local.sh
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK=$(mktemp -d); SERVER=""
trap '[ -n "$SERVER" ] && kill "$SERVER" 2>/dev/null; rm -rf "$WORK"' EXIT
PORT=${PORT:-8765}

BIN=${GUARDIAN_BIN:-$WORK/guardian}
if [ -z "${GUARDIAN_BIN:-}" ]; then (cd "$ROOT" && go build -o "$BIN" ./cmd/guardian) || exit 1; fi

python3 "$ROOT/scripts/e2e/fake_github.py" "$PORT" > "$WORK/api.log" 2>&1 &
SERVER=$!
for _ in $(seq 50); do (echo > /dev/tcp/127.0.0.1/"$PORT") 2>/dev/null && break; sleep 0.1; done

# A repo whose base commit has the demo spec, and a PR commit that removes `email`.
mkdir "$WORK/repo" && cd "$WORK/repo"
git init -q -b main && git config user.email t@t && git config user.name t
mkdir api && cp "$ROOT/api/openapi.yaml" api/ && git add -A && git commit -qm base
BASE_SHA=$(git rev-parse HEAD)
git checkout -qb feature
sed -i '/email: {type: string}/d' api/openapi.yaml && git commit -qam "remove email"
echo '{"pull_request": {"number": 12}}' > "$WORK/event.json"

# The variables GitHub Actions provides.
export GITHUB_REPOSITORY=acme/shop GITHUB_EVENT_PATH="$WORK/event.json" GITHUB_TOKEN=ghs_fake \
       GITHUB_API_URL="http://127.0.0.1:$PORT" GITHUB_STEP_SUMMARY="$WORK/summary.md"

fail=0
expect() { if [ "$2" = "$3" ]; then echo "ok   $1"; else echo "FAIL $1 (got '$3', want '$2')"; fail=1; fi; }

"$ROOT/scripts/base-spec.sh" "$BASE_SHA" api/openapi.yaml "$WORK/base.yaml"

out=$("$BIN" comment "$WORK/base.yaml" api/openapi.yaml); code=$?
expect "push 1: breaking change -> exit 1" 1 "$code"
echo "$out" | grep -q "comment created" && echo "ok   push 1: comment created" || { echo "FAIL push 1: not created"; fail=1; }

git checkout -q main -- api/openapi.yaml && git commit -qam "restore email"
out=$("$BIN" comment "$WORK/base.yaml" api/openapi.yaml); code=$?
expect "push 2: fixed -> exit 0" 0 "$code"
echo "$out" | grep -q "comment updated" && echo "ok   push 2: comment updated in place" || { echo "FAIL push 2: not updated"; fail=1; }

expect "exactly one comment created" 1 "$(grep -c '^POST' "$WORK/api.log")"
expect "exactly one in-place update" 1 "$(grep -c '^PATCH' "$WORK/api.log")"
grep -q "Bearer ghs_fake" "$WORK/api.log" && echo "ok   token sent as bearer auth" || { echo "FAIL no auth header"; fail=1; }
grep -q "Guardian API check" "$WORK/summary.md" && echo "ok   job summary written" || { echo "FAIL no job summary"; fail=1; }

[ $fail -eq 0 ] && echo "E2E PASS" || echo "E2E FAILED"
exit $fail
