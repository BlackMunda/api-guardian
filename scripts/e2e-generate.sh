#!/usr/bin/env bash
# Stage 3 rehearsal: specs GENERATED FROM CODE, no GitHub involved.
#
# For each scenario in demo/scenarios/*.patch it builds a scratch git repo:
#   base commit = the demo API,   PR commit = the demo API + the scenario patch
# generates the OpenAPI spec from the Go code on BOTH commits (exactly what the
# GitHub Action does), runs Guardian, and checks the verdict.
#
# Needs: go, git, python3 (for the fake GitHub server in the last step).
# Usage: scripts/e2e-generate.sh     (or GUARDIAN_BIN=/path/to/guardian ...)
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK=$(mktemp -d); SERVER=""
trap '[ -n "$SERVER" ] && kill "$SERVER" 2>/dev/null; rm -rf "$WORK"' EXIT

if [ ! -f "$ROOT/demo/go.sum" ] && [ -z "${GOFLAGS:-}" ]; then
  echo "demo/go.sum is missing. Run:  (cd demo && go mod tidy)" >&2; exit 2
fi
BIN=${GUARDIAN_BIN:-$WORK/guardian}
if [ -z "${GUARDIAN_BIN:-}" ]; then (cd "$ROOT" && go build -o "$BIN" ./cmd/guardian) || exit 1; fi

fail=0
ok()   { echo "ok   $*"; }
bad()  { echo "FAIL $*"; fail=1; }

# new_repo <dir> <patch-or-empty>: base commit = demo API; PR commit = base + patch
new_repo() {
  local dir=$1 patch=$2
  mkdir -p "$dir" && cd "$dir" && git init -q -b main && git config user.email t@t && git config user.name t
  mkdir demo && cp -r "$ROOT/demo/." demo/ && rm -rf demo/scenarios
  git add -A && git commit -qm base && BASE_SHA=$(git rev-parse HEAD)
  if [ -n "$patch" ]; then
    git apply --directory=demo "$patch" || return 1
    git commit -qam "PR: $(basename "$patch" .patch)"
  else
    git commit -q --allow-empty -m "PR: no change"
  fi
}

# generate both specs the way the Action does
specs() {
  rm -rf "$WORK/out"
  BASE_REF="$BASE_SHA" OUT_DIR="$WORK/out" WORKSPACE="$PWD" \
    SPEC_COMMAND="go run ./cmd/api openapi" WORKING_DIRECTORY=demo \
    "$ROOT/scripts/prepare-specs.sh"
}

scenario() { # scenario <name> <patch-or-empty> <expected-exit> <expected-text>
  local name=$1 patch=$2 want_exit=$3 want_text=$4
  new_repo "$WORK/$name" "$patch" || { bad "$name: patch does not apply to the current demo code"; return; }
  specs >"$WORK/$name.log" 2>&1 || { bad "$name: could not generate specs"; sed 's/^/     /' "$WORK/$name.log" | head -15; return; }
  out=$("$BIN" diff "$WORK/out/base.yaml" "$WORK/out/pr.yaml"); code=$?
  if [ "$code" -eq "$want_exit" ] && echo "$out" | grep -qF -- "$want_text"; then
    ok "$name -> exit $code, found \"$want_text\""
  else
    bad "$name -> exit $code (want $want_exit), expected text \"$want_text\""; echo "$out" | sed 's/^/     /'
  fi
}

SC="$ROOT/demo/scenarios"
echo "--- code changes -> generated spec -> Guardian verdict"
scenario "no-change"       ""                          0 "No API changes detected."
scenario "remove-field"    "$SC/remove-field.patch"    1 "Response field 'email' was removed"
scenario "required-field"  "$SC/required-field.patch"  1 "Request field 'currency' is now required"
scenario "type-change"     "$SC/type-change.patch"     1 "type changed from integer to string"
scenario "remove-endpoint" "$SC/remove-endpoint.patch" 1 "Endpoint was removed"
scenario "safe-add"        "$SC/safe-add.patch"        0 "New response field 'nickname'"

echo "--- the demo API is new in the PR (absent on the base branch) -> skipped, not failed"
mkdir -p "$WORK/newapi" && cd "$WORK/newapi" && git init -q -b main && git config user.email t@t && git config user.name t
echo hi > README.md && git add -A && git commit -qm base && BASE_SHA=$(git rev-parse HEAD)
mkdir demo && cp -r "$ROOT/demo/." demo/ && git add -A && git commit -qm "add the API"
specs >/dev/null 2>&1; code=$?
[ "$code" -eq 3 ] && ok "new API -> prepare-specs exits 3 (skip)" || bad "new API: exit $code, want 3"

echo "--- full PR flow: generated specs -> named comment on a (fake) PR"
PORT=${PORT:-8766}
python3 "$ROOT/scripts/e2e/fake_github.py" "$PORT" > "$WORK/api.log" 2>&1 &
SERVER=$!
for _ in $(seq 50); do (echo > /dev/tcp/127.0.0.1/"$PORT") 2>/dev/null && break; sleep 0.1; done
new_repo "$WORK/flow" "$SC/remove-field.patch" && specs >/dev/null 2>&1
echo '{"pull_request": {"number": 21}}' > "$WORK/event.json"
export GITHUB_REPOSITORY=acme/shop GITHUB_EVENT_PATH="$WORK/event.json" GITHUB_TOKEN=t \
       GITHUB_API_URL="http://127.0.0.1:$PORT" GITHUB_STEP_SUMMARY="$WORK/summary.md"
out=$("$BIN" comment --name demo-api "$WORK/out/base.yaml" "$WORK/out/pr.yaml"); code=$?
[ "$code" -eq 1 ] && ok "comment flow exits 1 on a breaking change" || bad "comment flow exit $code"
echo "$out" | grep -q "comment created" && ok "PR comment created" || bad "no comment created"
grep -q "guardian-report:demo-api" "$WORK/summary.md" && ok "comment is named 'demo-api' (own marker)" || bad "marker missing"
grep -q "Guardian API check (demo-api): ❌ 3 breaking changes" "$WORK/summary.md" && ok "report says 3 breaking changes" || bad "unexpected report"

[ $fail -eq 0 ] && echo "STAGE 3 E2E PASS" || echo "STAGE 3 E2E FAILED"
exit $fail
