#!/usr/bin/env bash
# Tests gen-spec.sh and prepare-specs.sh in throw-away git repos. No Go needed:
# the "spec generator" is just `cat spec.yaml`.   Usage: scripts/test-prepare-specs.sh
set -uo pipefail
S="$(cd "$(dirname "$0")" && pwd)"
WORK=$(mktemp -d); trap 'rm -rf "$WORK"' EXIT
fail=0
check() { # check <name> <want-exit> <got-exit>
  if [ "$2" -eq "$3" ]; then echo "ok   $1"; else echo "FAIL $1 (exit $3, want $2)"; fail=1; fi
}
same() { # same <name> <expected-content> <file>
  if [ "$(cat "$3" 2>/dev/null)" = "$2" ]; then echo "ok   $1"; else echo "FAIL $1 (got '$(cat "$3" 2>/dev/null)')"; fail=1; fi
}

# A repo: base commit has svc/spec.yaml and api/openapi.yaml; the PR changes both.
R="$WORK/repo"; mkdir -p "$R/svc" "$R/api" && cd "$R"
git init -q -b main && git config user.email t@t && git config user.name t
echo "spec: old" > svc/spec.yaml; echo "file: old" > api/openapi.yaml
git add -A && git commit -qm base && BASE=$(git rev-parse HEAD)
echo "spec: new" > svc/spec.yaml; echo "file: new" > api/openapi.yaml
mkdir other && echo "x: 1" > other/spec.yaml
git add -A && git commit -qm pr

run() { # run <env assignments...>  (OUT_DIR is fresh each time)
  rm -rf "$WORK/out"; env BASE_REF="$BASE" OUT_DIR="$WORK/out" WORKSPACE="$R" "$@" "$S/prepare-specs.sh" >/dev/null 2>&1
}

echo "--- generate mode"
run SPEC_COMMAND="cat spec.yaml" WORKING_DIRECTORY=svc;   check "generates base and PR specs" 0 $?
same "base.yaml is from the BASE commit" "spec: old" "$WORK/out/base.yaml"
same "pr.yaml is from the PR"            "spec: new" "$WORK/out/pr.yaml"
[ "$(git -C "$R" worktree list | wc -l)" -eq 1 ] && echo "ok   temporary worktree cleaned up" || { echo "FAIL worktree left behind"; fail=1; }
run SPEC_COMMAND="cat spec.yaml" WORKING_DIRECTORY=other; check "dir absent at base -> skip (3)" 3 $?
run SPEC_COMMAND="exit 1" WORKING_DIRECTORY=svc;          check "failing command -> error (2)" 2 $?
run SPEC_COMMAND="true" WORKING_DIRECTORY=svc;            check "command printing nothing -> error (2)" 2 $?
run SPEC_COMMAND="cat spec.yaml" WORKING_DIRECTORY=svc BASE_REF=deadbeef; check "unknown base ref -> error (2)" 2 $?

echo "--- file mode"
run SPEC_PATH=api/openapi.yaml;                           check "file mode works" 0 $?
same "base.yaml is the OLD file" "file: old" "$WORK/out/base.yaml"
same "pr.yaml is the NEW file"   "file: new" "$WORK/out/pr.yaml"
run SPEC_PATH=other/spec.yaml;                            check "file absent at base -> skip (3)" 3 $?

echo "--- configuration errors"
run;                                                      check "neither mode set -> error (2)" 2 $?
run SPEC_PATH=a SPEC_COMMAND=b;                           check "both modes set -> error (2)" 2 $?

echo "--- gen-spec.sh alone"
"$S/gen-spec.sh" "$R" svc "cat spec.yaml" "$WORK/g.yaml" 2>/dev/null;  check "gen-spec works" 0 $?
"$S/gen-spec.sh" "$R" nope "cat spec.yaml" "$WORK/g2.yaml" 2>/dev/null; check "gen-spec: missing dir -> 3" 3 $?
"$S/gen-spec.sh" a b 2>/dev/null;                                       check "gen-spec: bad usage -> 2" 2 $?
exit $fail
