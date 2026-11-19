#!/usr/bin/env bash
# Generate an OpenAPI spec by running a command inside a source tree.
#
# usage: gen-spec.sh <root-dir> <working-dir> <command> <output-file>
#
# Runs <command> in <root-dir>/<working-dir> and saves its stdout as the spec.
# The command's own stderr (e.g. compile errors) is passed through.
#
# Exit codes:
#   0  spec written to <output-file>
#   2  the command failed or printed nothing
#   3  <working-dir> does not exist in <root-dir> (nothing to generate from)
set -uo pipefail

if [ "$#" -ne 4 ]; then
  echo "usage: $0 <root-dir> <working-dir> <command> <output-file>" >&2
  exit 2
fi
root=$1; workdir=$2; command=$3; out=$4

case "$out" in /*) ;; *) out="$PWD/$out" ;; esac   # we cd below, so make it absolute

dir="$root/$workdir"
if [ ! -d "$dir" ]; then
  echo "notice: directory '$workdir' does not exist in $root; nothing to generate from." >&2
  exit 3
fi

cd "$dir" || exit 2
if ! bash -c "$command" > "$out"; then
  echo "error: spec command failed in '$workdir': $command" >&2
  exit 2
fi
if [ ! -s "$out" ]; then
  echo "error: spec command printed nothing: $command" >&2
  exit 2
fi
