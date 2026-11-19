#!/usr/bin/env bash
# Produce <OUT_DIR>/base.yaml and <OUT_DIR>/pr.yaml, the two specs Guardian compares.
#
# Two modes. Set exactly one of:
#   SPEC_PATH     the spec is a file in the repo (hand-written, or committed)
#   SPEC_COMMAND  the spec is GENERATED FROM CODE: a command that prints it to
#                 stdout, run in WORKING_DIRECTORY on both the base and the PR
#
# Environment:
#   BASE_REF           git ref to compare against (required)
#   OUT_DIR            where to write base.yaml and pr.yaml (required)
#   WORKSPACE          the checked-out PR (default: current directory)
#   WORKING_DIRECTORY  generate mode: directory to run the command in (default ".")
#
# Exit codes: 0 ready, 2 error, 3 nothing to compare (the PR introduces the spec)
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
: "${BASE_REF:?BASE_REF is required}"
: "${OUT_DIR:?OUT_DIR is required}"
WORKSPACE="${WORKSPACE:-$PWD}"
WORKING_DIRECTORY="${WORKING_DIRECTORY:-.}"
SPEC_PATH="${SPEC_PATH:-}"
SPEC_COMMAND="${SPEC_COMMAND:-}"

if [ -n "$SPEC_PATH" ] && [ -n "$SPEC_COMMAND" ]; then
  echo "error: set either spec-path or spec-command, not both." >&2; exit 2
fi
if [ -z "$SPEC_PATH" ] && [ -z "$SPEC_COMMAND" ]; then
  echo "error: set spec-path (a spec file) or spec-command (a command that prints the spec)." >&2; exit 2
fi
mkdir -p "$OUT_DIR"
cd "$WORKSPACE" || exit 2

# ---- file mode -----------------------------------------------------------
if [ -n "$SPEC_PATH" ]; then
  "$SCRIPT_DIR/base-spec.sh" "$BASE_REF" "$SPEC_PATH" "$OUT_DIR/base.yaml"
  code=$?
  [ "$code" -ne 0 ] && exit "$code"
  if [ ! -f "$SPEC_PATH" ]; then
    echo "error: '$SPEC_PATH' does not exist in the pull request." >&2; exit 2
  fi
  cp "$SPEC_PATH" "$OUT_DIR/pr.yaml"
  exit 0
fi

# ---- generate mode -------------------------------------------------------
if ! git cat-file -e "${BASE_REF}^{commit}" 2>/dev/null; then
  echo "error: git ref '${BASE_REF}' not found. Use actions/checkout with 'fetch-depth: 0'." >&2; exit 2
fi

# Check the base commit out into a separate directory, so the PR's files stay put.
tree="$(mktemp -d)"
cleanup() { git worktree remove --force "$tree" >/dev/null 2>&1; rm -rf "$tree"; }
trap cleanup EXIT
if ! git worktree add --quiet --detach "$tree" "$BASE_REF"; then
  echo "error: could not check out ${BASE_REF}." >&2; exit 2
fi

"$SCRIPT_DIR/gen-spec.sh" "$tree" "$WORKING_DIRECTORY" "$SPEC_COMMAND" "$OUT_DIR/base.yaml"
code=$?
if [ "$code" -eq 3 ]; then exit 3; fi   # the API did not exist on the base branch
if [ "$code" -ne 0 ]; then echo "error: could not generate the BASE spec." >&2; exit 2; fi

"$SCRIPT_DIR/gen-spec.sh" "$WORKSPACE" "$WORKING_DIRECTORY" "$SPEC_COMMAND" "$OUT_DIR/pr.yaml"
code=$?
if [ "$code" -ne 0 ]; then echo "error: could not generate the PR spec (does the PR code build?)." >&2; exit 2; fi
exit 0
