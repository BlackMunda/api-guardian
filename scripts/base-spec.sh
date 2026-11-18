#!/usr/bin/env bash
# Extract the OpenAPI spec as it existed at a git ref (normally the PR's base).
#
# usage: base-spec.sh <git-ref> <spec-path> <output-file>
#
# Exit codes:
#   0  wrote the base spec to <output-file>
#   2  the ref does not exist (checkout needs fetch-depth: 0)
#   3  the spec file does not exist at that ref (e.g. the PR adds it for the
#      first time), so there is nothing to compare against
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "usage: $0 <git-ref> <spec-path> <output-file>" >&2
  exit 2
fi
ref=$1
spec=$2
out=$3

if ! git cat-file -e "${ref}^{commit}" 2>/dev/null; then
  echo "error: git ref '${ref}' not found. Use actions/checkout with 'fetch-depth: 0'." >&2
  exit 2
fi

if ! git cat-file -e "${ref}:${spec}" 2>/dev/null; then
  echo "notice: '${spec}' does not exist at ${ref}; nothing to compare against." >&2
  exit 3
fi

git show "${ref}:${spec}" > "${out}"
